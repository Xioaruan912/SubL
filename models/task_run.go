package models

import "time"

// TaskRun is a persistent execution record for long-running administrative
// work. RequestJSON stores only non-secret replay parameters.
type TaskRun struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Type        string     `gorm:"size:64;index;not null" json:"type"`
	Name        string     `gorm:"size:160;not null" json:"name"`
	Status      string     `gorm:"size:24;index;not null" json:"status"`
	Progress    int        `gorm:"not null;default:0" json:"progress"`
	Message     string     `gorm:"size:500" json:"message"`
	Error       string     `gorm:"size:1000" json:"error"`
	RequestJSON string     `gorm:"type:text" json:"-"`
	ResultJSON  string     `gorm:"type:text" json:"resultJson,omitempty"`
	RetryOf     *uint      `gorm:"index" json:"retryOf,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
}

func RecoverInterruptedTasks() error {
	now := time.Now()
	return DB.Model(&TaskRun{}).Where("status IN ?", []string{"running","queued"}).Updates(map[string]any{
		"status":"failed", "message":"服务重启导致任务中断", "error":"interrupted by service restart", "finished_at":&now,
	}).Error
}
