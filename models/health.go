package models

import "time"

type NodeHealthEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	NodeID    int       `gorm:"index;not null" json:"nodeId"`
	NodeName  string    `gorm:"size:255" json:"nodeName"`
	Type      string    `gorm:"index;size:24;not null" json:"type"`
	Message   string    `gorm:"size:500" json:"message"`
	CreatedAt time.Time `gorm:"index;not null" json:"createdAt"`
}

type AlertSetting struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	Enabled          bool   `json:"enabled"`
	WebhookURL       string `gorm:"type:text" json:"webhookUrl"`
	FailureThreshold int    `gorm:"not null;default:3" json:"failureThreshold"`
	MaintenanceStart string `gorm:"size:5" json:"maintenanceStart"`
	MaintenanceEnd   string `gorm:"size:5" json:"maintenanceEnd"`
}

type UnlockObservation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	NodeID    int       `gorm:"index:idx_unlock_node_time,priority:1;not null" json:"nodeId"`
	Service   string    `gorm:"index;size:64;not null" json:"service"`
	Available bool      `json:"available"`
	Status    string    `gorm:"size:24" json:"status"`
	Region    string    `gorm:"size:32" json:"region"`
	Rtt       int       `json:"rtt"`
	CheckedAt time.Time `gorm:"index:idx_unlock_node_time,priority:2;not null" json:"checkedAt"`
}
