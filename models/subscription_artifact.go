package models

import "time"

// SubscriptionArtifact is immutable once created. It captures one generated
// client payload plus only hashed/derived build inputs and validation reports.
type SubscriptionArtifact struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	SubscriptionID   int       `gorm:"index:idx_sub_artifact,priority:1;not null" json:"subscriptionId"`
	Client           string    `gorm:"size:24;index:idx_sub_artifact,priority:2;not null" json:"client"`
	InputDigest      string    `gorm:"size:64;not null" json:"inputDigest"`
	TemplateName     string    `gorm:"size:255" json:"templateName"`
	TemplateChecksum string    `gorm:"size:64" json:"templateChecksum"`
	RulesChecksum    string    `gorm:"size:64" json:"rulesChecksum"`
	ContentChecksum  string    `gorm:"size:64;index;not null" json:"contentChecksum"`
	ByteSize         int       `json:"byteSize"`
	ValidationStatus string    `gorm:"size:24;not null" json:"validationStatus"`
	TestStatus       string    `gorm:"size:24;not null" json:"testStatus"`
	TestReportJSON   string    `gorm:"type:text" json:"testReportJson,omitempty"`
	Content          []byte    `gorm:"type:blob" json:"-"`
	CreatedAt        time.Time `json:"createdAt"`
}

type SubscriptionArtifactPointer struct {
	ID                      uint      `gorm:"primaryKey" json:"id"`
	SubscriptionID          int       `gorm:"uniqueIndex:idx_sub_artifact_pointer,priority:1;not null" json:"subscriptionId"`
	Client                  string    `gorm:"size:24;uniqueIndex:idx_sub_artifact_pointer,priority:2;not null" json:"client"`
	LastKnownGoodArtifactID uint      `gorm:"index;not null" json:"lastKnownGoodArtifactId"`
	UpdatedAt               time.Time `json:"updatedAt"`
}
