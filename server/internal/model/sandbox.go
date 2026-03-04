package model

import "time"

type SandboxRunStatus string

const (
	SandboxRunQueued    SandboxRunStatus = "QUEUED"
	SandboxRunRunning   SandboxRunStatus = "RUNNING"
	SandboxRunSucceeded SandboxRunStatus = "SUCCEEDED"
	SandboxRunFailed    SandboxRunStatus = "FAILED"
	SandboxRunStopped   SandboxRunStatus = "STOPPED"
	SandboxRunTimeout   SandboxRunStatus = "TIMEOUT"
)

type SandboxPolicy struct {
	MemoryMB      int  `json:"memoryMb"`
	CPUCoresMilli int  `json:"cpuMilli"`
	NetworkNone   bool `json:"networkNone"`
	ReadOnlyFS    bool `json:"readOnlyFs"`
	TimeoutSec    int  `json:"timeoutSec"`
}

type SandboxRun struct {
	ID          string           `json:"id"`
	TenantID    string           `json:"tenantId"`
	StrategyID  string           `json:"strategyId"`
	Status      SandboxRunStatus `json:"status"`
	Policy      SandboxPolicy    `json:"policy"`
	ExitMessage string           `json:"exitMessage,omitempty"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	StartedAt   *time.Time       `json:"startedAt,omitempty"`
	FinishedAt  *time.Time       `json:"finishedAt,omitempty"`
}

type CreateSandboxRunRequest struct {
	StrategyID string         `json:"strategyId"`
	Policy     *SandboxPolicy `json:"policy,omitempty"`
	ForceFail  bool           `json:"forceFail,omitempty"`
}
