package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ppeelink/models"
)

func TestRoutingRegressionDetectsPolicyChange(t *testing.T) {
	cases := []models.RoutingRegressionCase{{Name: "Gemini", Domain: "gemini.google.com", ExpectedPolicy: "AI", ForbiddenPolicy: "DIRECT", Enabled: true}}
	before := "rules:\n  - DOMAIN,gemini.google.com,AI\n  - MATCH,DIRECT\n"
	after := "rules:\n  - DOMAIN,gemini.google.com,DIRECT\n  - MATCH,DIRECT\n"
	diff, oldResults, newResults := compareRoutingTemplates(context.Background(), "clash.yaml", before, after, cases)
	if len(diff) != 1 || !diff[0].Changed {
		t.Fatalf("expected changed route: %#v", diff)
	}
	if len(oldResults) != 1 || !oldResults[0].Passed {
		t.Fatalf("before should pass: %#v", oldResults)
	}
	if len(newResults) != 1 || newResults[0].Passed {
		t.Fatalf("after should fail: %#v", newResults)
	}
}

func TestClientCapabilityMatrixBlocksUnsupportedTransport(t *testing.T) {
	proxies := []map[string]interface{}{{"type": "vless", "network": "grpc", "reality-opts": map[string]interface{}{"enabled": true}, "udp": true}}
	matrix := buildClientCapabilityMatrix(proxies)
	byClient := map[string]templateClientCapability{}
	for _, item := range matrix {
		byClient[item.Client] = item
	}
	if byClient["Clash/Mihomo"].Status != "pass" {
		t.Fatalf("mihomo should pass: %#v", byClient["Clash/Mihomo"])
	}
	if byClient["sing-box"].Status == "error" {
		t.Fatalf("sing-box should not be blocked: %#v", byClient["sing-box"])
	}
	for _, name := range []string{"Surge", "Loon", "Quantumult X"} {
		if byClient[name].Status != "error" {
			t.Fatalf("%s should be blocked: %#v", name, byClient[name])
		}
	}
}

func TestConfiguredDeployRequiresOperatorConfiguration(t *testing.T) {
	old := os.Getenv("SUBLINKX_DEPLOY_SCRIPT")
	t.Cleanup(func() { _ = os.Setenv("SUBLINKX_DEPLOY_SCRIPT", old) })
	_ = os.Unsetenv("SUBLINKX_DEPLOY_SCRIPT")
	if _, err := runConfiguredDeployTask(context.Background()); err == nil {
		t.Fatal("deploy must refuse when script is not configured")
	}
}

func TestSafePublishSurgeIntegration(t *testing.T) {
	oldDB := models.DB
	oldBase := baseTemplateDir
	t.Cleanup(func() { models.DB = oldDB; baseTemplateDir = oldBase })
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	models.DB = db
	if err := db.AutoMigrate(&models.Subcription{}, &models.Node{}, &models.GroupNode{}, &models.Airport{}, &models.RuleCatalog{}, &models.RoutingRegressionCase{}, &models.SubscriptionArtifact{}, &models.SubscriptionArtifactPointer{}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	t.Chdir(root)
	baseTemplateDir = filepath.Join(root, "template")
	if err := os.MkdirAll(baseTemplateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	template := "[General]\nloglevel = notify\n\n[Proxy]\nDIRECT = direct\n\n[Proxy Group]\nProxy = select,DIRECT\n\n[Rule]\nFINAL,Proxy\n"
	if err := os.WriteFile(filepath.Join(baseTemplateDir, "safe.conf"), []byte(template), 0o600); err != nil {
		t.Fatal(err)
	}
	n := models.Node{Name: "US-test", Link: "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQxMjM0NTY3ODk@1.2.3.4:8388#US-test"}
	if err := db.Create(&n).Error; err != nil {
		t.Fatal(err)
	}
	cfg, _ := json.Marshal(models.SubscriptionConfig{Surge: "./template/old.conf"})
	sub := models.Subcription{Name: "safe-publish-test", Config: string(cfg), Nodes: []models.Node{n}}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&sub).Association("Nodes").Replace([]models.Node{n}); err != nil {
		t.Fatal(err)
	}
	report, err := runSafePublish(context.Background(), safePublishRequest{SubscriptionID: sub.ID, Template: "safe.conf", Client: "surge"})
	if err != nil {
		t.Fatalf("safe publish failed: %v", err)
	}
	if !report.Published || report.PublishedID == 0 {
		t.Fatalf("not published: %#v", report)
	}
	var updated models.Subcription
	if err := db.First(&updated, sub.ID).Error; err != nil {
		t.Fatal(err)
	}
	var updatedCfg models.SubscriptionConfig
	if err := json.Unmarshal([]byte(updated.Config), &updatedCfg); err != nil {
		t.Fatal(err)
	}
	if updatedCfg.Surge != "./template/safe.conf" {
		t.Fatalf("unexpected binding: %s", updatedCfg.Surge)
	}
	var pointer models.SubscriptionArtifactPointer
	if err := db.Where("subscription_id = ? AND client = ?", sub.ID, "surge").First(&pointer).Error; err != nil {
		t.Fatal(err)
	}
	if pointer.LastKnownGoodArtifactID != report.PublishedID {
		t.Fatalf("unexpected LKG pointer: %#v", pointer)
	}
}
