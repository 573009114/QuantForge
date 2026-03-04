package model

import "time"

type DeadLetterJob struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId"`
	JobType   string    `json:"jobType"`
	JobID     string    `json:"jobId"`
	Reason    string    `json:"reason"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"createdAt"`
}
