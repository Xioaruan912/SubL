package rulecenter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ppeelink/models"
)

func setupResolverTestDir(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(filepath.Join("template", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join("db", "rules-cache", "providers"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func setupEmptyRuleDB(t *testing.T) {
	t.Helper()
	old := models.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Node{}, &models.NodeQualitySample{}); err != nil {
		t.Fatal(err)
	}
	models.DB = db
	t.Cleanup(func() { models.DB = old })
}

func TestResolveProviderRulesLocalFileAndPathSafety(t *testing.T) {
	setupResolverTestDir(t)
	provider := filepath.Join("template", "rules", "local.yaml")
	if err := os.WriteFile(provider, []byte("payload:\n  - DOMAIN-SUFFIX,example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	template := "rule-providers:\n  Local:\n    type: file\n    behavior: classical\n    path: ./rules/local.yaml\n"
	rules, _, err := ResolveProviderRules(context.Background(), template, "Local")
	if err != nil {
		t.Fatal(err)
	}
	if !MatchDomain(rules, "api.example.com") {
		t.Fatal("local rule-provider did not match payload")
	}

	bad := "rule-providers:\n  Bad:\n    type: file\n    path: ../../etc/passwd\n"
	if _, _, err := ResolveProviderRules(context.Background(), bad, "Bad"); err == nil || !strings.Contains(err.Error(), "越权") {
		t.Fatalf("expected traversal rejection, got %v", err)
	}
}

func TestResolveProviderRulesRemoteCacheFallback(t *testing.T) {
	setupResolverTestDir(t)
	setupEmptyRuleDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("payload:\n  - DOMAIN,chatgpt.com\n"))
	}))
	template := "rule-providers:\n  Remote:\n    type: http\n    behavior: classical\n    url: " + server.URL + "/provider.yaml\n"
	rules, _, err := ResolveProviderRules(context.Background(), template, "Remote")
	if err != nil || !MatchDomain(rules, "chatgpt.com") {
		t.Fatalf("initial remote resolve failed: %v", err)
	}
	server.Close()
	rules, _, err = ResolveProviderRules(context.Background(), template, "Remote")
	if err != nil || !MatchDomain(rules, "chatgpt.com") {
		t.Fatalf("cached resolve failed after remote outage: %v", err)
	}
}

func TestResolveProviderRulesHonorsContextTimeout(t *testing.T) {
	setupResolverTestDir(t)
	setupEmptyRuleDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(150 * time.Millisecond)
		_, _ = w.Write([]byte("payload:\n  - DOMAIN,chatgpt.com\n"))
	}))
	defer server.Close()
	template := "rule-providers:\n  Slow:\n    type: http\n    url: " + server.URL + "/slow.yaml\n"
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, _, err := ResolveProviderRules(ctx, template, "Slow")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if time.Since(started) > 500*time.Millisecond {
		t.Fatalf("context timeout was not honored quickly: %v", time.Since(started))
	}
}
