package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type AuthClaims struct {
	TenantID string `json:"tenantId"`
	Role     string `json:"role"`
	UserID   string `json:"userId"`
	Exp      int64  `json:"exp"`
}

type AuthService struct {
	secret []byte
}

func NewAuthService(secret string) *AuthService {
	if strings.TrimSpace(secret) == "" {
		secret = "quantforge-dev-secret"
	}
	return &AuthService{secret: []byte(secret)}
}

func (s *AuthService) IssueToken(tenantID, role, userID string, ttl time.Duration) (string, error) {
	if tenantID == "" || role == "" {
		return "", errors.New("tenantId and role are required")
	}
	if ttl <= 0 {
		ttl = 8 * time.Hour
	}
	claims := AuthClaims{TenantID: tenantID, Role: role, UserID: userID, Exp: time.Now().Add(ttl).Unix()}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	p := base64.RawURLEncoding.EncodeToString(payload)
	sig := s.sign(p)
	return p + "." + sig, nil
}

func (s *AuthService) ParseToken(token string) (AuthClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return AuthClaims{}, errors.New("invalid token")
	}
	if !hmac.Equal([]byte(s.sign(parts[0])), []byte(parts[1])) {
		return AuthClaims{}, errors.New("invalid token signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return AuthClaims{}, errors.New("invalid token payload")
	}
	var claims AuthClaims
	if err = json.Unmarshal(raw, &claims); err != nil {
		return AuthClaims{}, errors.New("invalid token claims")
	}
	if claims.Exp <= time.Now().Unix() {
		return AuthClaims{}, errors.New("token expired")
	}
	return claims, nil
}

func (s *AuthService) sign(payload string) string {
	h := hmac.New(sha256.New, s.secret)
	_, _ = h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
