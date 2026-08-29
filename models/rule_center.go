package models

import "time"

// RuleSource stores synchronization metadata for an external rule repository.
type RuleSource struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Key            string    `gorm:"size:64;uniqueIndex;not null" json:"key"`
	Name           string    `gorm:"size:128;not null" json:"name"`
	Type           string    `gorm:"size:32;not null" json:"type"`
	Repo           string    `gorm:"size:255" json:"repo"`
	Branch         string    `gorm:"size:64" json:"branch"`
	BaseURL        string    `gorm:"size:512" json:"baseUrl"`
	Enabled        bool      `gorm:"not null;default:true" json:"enabled"`
	LastSyncAt     *time.Time `json:"lastSyncAt"`
	LastSyncStatus string    `gorm:"size:32" json:"lastSyncStatus"`
	LastSyncError  string    `gorm:"type:text" json:"lastSyncError"`
	LastRevision   string    `gorm:"size:128" json:"lastRevision"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// RuleCatalog stores searchable metadata only. Rule payloads live in the file cache.
type RuleCatalog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	SourceKey    string    `gorm:"size:64;index;not null" json:"sourceKey"`
	ExternalID   string    `gorm:"size:512;uniqueIndex;not null" json:"externalId"`
	Name         string    `gorm:"size:255;index;not null" json:"name"`
	Category     string    `gorm:"size:64;index" json:"category"`
	Platform     string    `gorm:"size:32;index;not null" json:"platform"`
	Format       string    `gorm:"size:32" json:"format"`
	URL          string    `gorm:"type:text;not null" json:"url"`
	LocalPath    string    `gorm:"type:text" json:"localPath"`
	RuleCount    int       `json:"ruleCount"`
	RemoteUpdate   string    `gorm:"size:128" json:"remoteUpdate"`
	RemoteRevision string    `gorm:"size:128;index" json:"remoteRevision"`
	CacheRevision  string    `gorm:"size:128" json:"cacheRevision"`
	Checksum       string    `gorm:"size:128" json:"checksum"`
	MetadataJSON string    `gorm:"type:text" json:"metadataJson"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
