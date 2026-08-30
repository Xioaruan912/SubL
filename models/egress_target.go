package models

import (
	"strings"
	"time"
)

// EgressTarget stores an administrator-configurable destination used by
// server-side split-routing checks. Country expectations intentionally do not
// live here; they are derived from the active subscription template.
type EgressTarget struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Key              string    `gorm:"uniqueIndex;size:80;not null" json:"key"`
	Name             string    `gorm:"size:120;not null" json:"name"`
	Domain           string    `gorm:"size:255;not null" json:"domain"`
	Group            string    `gorm:"size:80;not null" json:"group"`
	Icon             string    `gorm:"size:32" json:"icon"`
	Path             string    `gorm:"size:255" json:"path"`
	Method           string    `gorm:"size:12;not null;default:GET" json:"method"`
	ExpectedStatus   string    `gorm:"size:80;not null;default:200-399" json:"expectedStatus"`
	ResponseContains string    `gorm:"size:255" json:"responseContains"`
	RequireEgressIP  bool      `gorm:"not null" json:"requireEgressIp"`
	TimeoutSeconds   int       `gorm:"not null;default:7" json:"timeoutSeconds"`
	Retries          int       `gorm:"not null;default:0" json:"retries"`
	Enabled          bool      `gorm:"not null;index" json:"enabled"`
	SortOrder        int       `gorm:"not null;default:0;index" json:"sortOrder"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

var defaultEgressTargets = []EgressTarget{
	{Key: "cloudflare", Name: "Cloudflare", Domain: "www.cloudflare.com", Group: "network", Icon: "☁️", Enabled: true, SortOrder: 10, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "chatgpt", Name: "ChatGPT", Domain: "chatgpt.com", Group: "ai", Icon: "◉", Enabled: true, SortOrder: 20, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "openai", Name: "OpenAI", Domain: "openai.com", Group: "ai", Icon: "◎", Enabled: true, SortOrder: 30, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "gemini", Name: "Gemini", Domain: "gemini.google.com", Group: "ai", Icon: "✨", Path: "/", Enabled: true, SortOrder: 40, RequireEgressIP: false, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "claude", Name: "Claude", Domain: "claude.ai", Group: "ai", Icon: "◌", Enabled: true, SortOrder: 50, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "anthropic", Name: "Anthropic", Domain: "anthropic.com", Group: "ai", Icon: "△", Enabled: true, SortOrder: 60, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "perplexity", Name: "Perplexity", Domain: "www.perplexity.ai", Group: "ai", Icon: "✦", Enabled: true, SortOrder: 70, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "discord", Name: "Discord", Domain: "gateway.discord.gg", Group: "social", Icon: "♬", Enabled: true, SortOrder: 80, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "x", Name: "X", Domain: "x.com", Group: "social", Icon: "𝕏", Enabled: true, SortOrder: 90, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "medium", Name: "Medium", Domain: "medium.com", Group: "content", Icon: "M", Enabled: true, SortOrder: 100, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "coinbase", Name: "Coinbase", Domain: "coinbase.com", Group: "finance", Icon: "₿", Enabled: true, SortOrder: 110, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "notion", Name: "Notion", Domain: "notion.so", Group: "tools", Icon: "N", Enabled: true, SortOrder: 120, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "cdnjs", Name: "Cloudflare CDN", Domain: "cdnjs.cloudflare.com", Group: "developer", Icon: "☁️", Enabled: true, SortOrder: 130, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "npm", Name: "npm Registry", Domain: "registry.npmjs.org", Group: "developer", Icon: "npm", Enabled: true, SortOrder: 140, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "gitlab", Name: "GitLab", Domain: "gitlab.com", Group: "developer", Icon: "◈", Enabled: true, SortOrder: 150, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
	{Key: "crunchyroll", Name: "Crunchyroll", Domain: "crunchyroll.com", Group: "media", Icon: "▶", Enabled: true, SortOrder: 160, RequireEgressIP: true, TimeoutSeconds: 7, ExpectedStatus: "200-399"},
}

func EnsureDefaultEgressTargets() error {
	for _, item := range defaultEgressTargets {
		var count int64
		if err := DB.Model(&EgressTarget{}).Where("key = ?", item.Key).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := DB.Create(&item).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func EnabledEgressTargets() ([]EgressTarget, error) {
	var items []EgressTarget
	err := DB.Where("enabled = ?", true).Order("sort_order ASC, id ASC").Find(&items).Error
	return items, err
}

func NormalizeEgressTarget(item *EgressTarget) {
	item.Key = strings.ToLower(strings.TrimSpace(item.Key))
	item.Name = strings.TrimSpace(item.Name)
	item.Domain = strings.ToLower(strings.TrimSpace(item.Domain))
	item.Group = strings.ToLower(strings.TrimSpace(item.Group))
	item.Method = strings.ToUpper(strings.TrimSpace(item.Method))
	if item.Method == "" {
		item.Method = "GET"
	}
	if item.ExpectedStatus == "" {
		item.ExpectedStatus = "200-399"
	}
	if item.TimeoutSeconds < 1 {
		item.TimeoutSeconds = 7
	}
	if item.TimeoutSeconds > 60 {
		item.TimeoutSeconds = 60
	}
	if item.Retries < 0 {
		item.Retries = 0
	}
	if item.Retries > 5 {
		item.Retries = 5
	}
	if item.Path != "" && !strings.HasPrefix(item.Path, "/") {
		item.Path = "/" + item.Path
	}
}
