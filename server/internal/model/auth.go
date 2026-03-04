package model

type IssueTokenRequest struct {
	TenantID string `json:"tenantId"`
	Role     string `json:"role"`
	UserID   string `json:"userId"`
}

type IssueTokenResponse struct {
	Token string `json:"token"`
}
