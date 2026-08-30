package models

import "time"

// AuditLog intentionally stores metadata only. Request bodies, credentials,
// node links and subscription URLs are never persisted in audit records.
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Actor     string    `gorm:"size:160;index" json:"actor"`
	AuthType  string    `gorm:"size:32" json:"authType"`
	IP        string    `gorm:"size:96" json:"ip"`
	Method    string    `gorm:"size:12;index" json:"method"`
	Path      string    `gorm:"size:255;index" json:"path"`
	Status    int       `json:"status"`
	Action    string    `gorm:"size:120;index" json:"action"`
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
}
