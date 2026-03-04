package service

import (
	"testing"
	"time"
)

func TestIssueAndParseToken(t *testing.T) {
	svc := NewAuthService("secret")
	token, err := svc.IssueToken("t1", "Admin", "u1", time.Hour)
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}
	claims, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("parse token failed: %v", err)
	}
	if claims.TenantID != "t1" || claims.Role != "Admin" || claims.UserID != "u1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}
