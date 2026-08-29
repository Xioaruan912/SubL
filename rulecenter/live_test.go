package rulecenter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ppeelink/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestLiveRuleSources(t *testing.T) {
	if os.Getenv("RULECENTER_LIVE") != "1" { t.Skip("set RULECENTER_LIVE=1 to run external source verification") }
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "rules.db")), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	models.DB = db
	if err := db.AutoMigrate(&models.RuleSource{}, &models.RuleCatalog{}); err != nil { t.Fatal(err) }
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second); defer cancel()
	if err := SyncAll(ctx); err != nil { t.Fatal(err) }
	var count int64
	if err := db.Model(&models.RuleCatalog{}).Count(&count).Error; err != nil { t.Fatal(err) }
	if count < 100 { t.Fatalf("catalog unexpectedly small: %d", count) }
	var openAI models.RuleCatalog
	if err := db.Where("source_key = ? AND platform = ? AND name = ?", "ios_rule_script", "Clash", "OpenAI").First(&openAI).Error; err != nil { t.Fatal(err) }
	item, rules, err := LoadItem(ctx, openAI.ExternalID)
	if err != nil { t.Fatal(err) }
	if item.RuleCount == 0 || !MatchDomain(rules, "chatgpt.com") { t.Fatalf("OpenAI rule payload did not match chatgpt.com: count=%d", item.RuleCount) }
}
