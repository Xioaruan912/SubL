package rulecenter

import "time"

type NormalizedRule struct {
	Type    string   `json:"type"`
	Value   string   `json:"value"`
	Options []string `json:"options,omitempty"`
}

type RuleItem struct {
	ExternalID string            `json:"externalId"`
	SourceKey  string            `json:"sourceKey"`
	Name       string            `json:"name"`
	Category   string            `json:"category"`
	Platform   string            `json:"platform"`
	Format     string            `json:"format"`
	URL        string            `json:"url"`
	LocalPath  string            `json:"localPath"`
	RuleCount  int               `json:"ruleCount"`
	UpdatedAt      string            `json:"updatedAt"`
	RemoteRevision string            `json:"remoteRevision,omitempty"`
	Checksum       string            `json:"checksum"`
	Metadata   map[string]int    `json:"metadata,omitempty"`
	Warnings   []string          `json:"warnings,omitempty"`
	Sample     []NormalizedRule  `json:"sample,omitempty"`
	FetchedAt  *time.Time        `json:"fetchedAt,omitempty"`
}

type SourceStatus struct {
	Key        string     `json:"key"`
	Name       string     `json:"name"`
	Kind       string     `json:"kind"`
	Repo       string     `json:"repo"`
	Branch     string     `json:"branch"`
	Enabled    bool       `json:"enabled"`
	Status     string     `json:"status"`
	LastSyncAt *time.Time `json:"lastSyncAt"`
	Error      string     `json:"error,omitempty"`
	Count       int64      `json:"count"`
	CachedCount int64      `json:"cachedCount"`
}
