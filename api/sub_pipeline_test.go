package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"ppeelink/models"
	"ppeelink/node"

	"github.com/gin-gonic/gin"
)

func TestApplySubscriptionPipeline(t *testing.T) {
	nodes := []models.Node{
		{Name: "[A] 香港 01", Link: "vmess://one#old"},
		{Name: "[A] 日本 01", Link: "ss://two#old"},
		{Name: "[A] 香港 01 重复", Link: "vmess://one#old"},
		{Name: "[A] 官网", Link: "https://example.com/sub"},
	}
	raw := `{"include":"香港|日本|官网","exclude":"官网","renamePattern":"^\\[A\\] ","renameReplacement":"","protocols":["vmess","ss"],"sort":"name","dedupe":true,"maxNodes":10}`
	result, err := ApplySubscriptionPipeline(nodes, raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.Before != 4 || result.After != 2 {
		t.Fatalf("unexpected counts: %#v", result)
	}
	if result.Nodes[0].Name != "日本 01" || result.Nodes[1].Name != "香港 01" {
		t.Fatalf("unexpected nodes: %#v", result.Nodes)
	}
	if result.Rejected["重复节点"] != 1 || result.Rejected["命中排除规则"] != 1 {
		t.Fatalf("unexpected rejection: %#v", result.Rejected)
	}
}

func TestApplySubscriptionPipelineRejectsBadRegex(t *testing.T) {
	_, err := ApplySubscriptionPipeline(nil, `{"include":"["}`)
	if err == nil {
		t.Fatal("expected invalid regex error")
	}
}

func TestPipelineWithOverrides(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/c/?token=x&filter=香港|日本&type=vless,ss&emoji=true&maxNodes=5&rename=^A&renameTo=B", nil)
	out := pipelineWithOverrides("", c)
	var cfg SubscriptionPipeline
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if cfg.Include != "香港|日本" || len(cfg.Protocols) != 2 || cfg.MaxNodes != 5 || !cfg.Emoji || cfg.RenamePattern != "^A" || cfg.RenameReplacement != "B" {
		t.Fatalf("unexpected override: %#v", cfg)
	}
}

func TestPipelineWithOverridesNoQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/c/?token=x", nil)
	base := `{"include":"keep"}`
	if got := pipelineWithOverrides(base, c); got != base {
		t.Fatalf("base should be unchanged, got %q", got)
	}
}

func TestApplySubscriptionPipelinePlaceholderFilter(t *testing.T) {
	nodes := []models.Node{
		{Name: "香港 01", Link: "ss://a#香港"},
		{Name: "官网入口", Link: "ss://b#官网"},
		{Name: "剩余流量 40G", Link: "ss://c#剩余"},
	}
	result, err := ApplySubscriptionPipeline(nodes, `{"excludePlaceholder":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if result.After != 1 || result.Nodes[0].Name != "香港 01" {
		t.Fatalf("unexpected placeholder filter result: %#v", result)
	}
	if result.Rejected["占位/广告节点"] != 2 {
		t.Fatalf("unexpected rejection counts: %#v", result.Rejected)
	}
}

func TestApplySubscriptionPipelineDeleteAndFlags(t *testing.T) {
	nodes := []models.Node{
		{Name: "香港 01", Link: "ss://a#香港"},
		{Name: "测试 02", Link: "ss://b#测试"},
	}
	result, err := ApplySubscriptionPipeline(nodes, `{"deletePattern":"测试","setUdp":true,"setSkipCertVerify":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if result.After != 1 || result.Nodes[0].Name != "香港 01" {
		t.Fatalf("unexpected delete result: %#v", result)
	}
	var flags node.NodeFlags
	if err := json.Unmarshal([]byte(result.Nodes[0].Flags), &flags); err != nil {
		t.Fatalf("bad flags json %q: %v", result.Nodes[0].Flags, err)
	}
	if !flags.UDP || !flags.SkipCertVerify {
		t.Fatalf("flags not applied: %#v", flags)
	}
}

func TestApplySubscriptionPipelineEmoji(t *testing.T) {
	nodes := []models.Node{
		{Name: "香港 01", Link: "ss://a#香港"},
		{Name: "JP 02", Link: "ss://b#jp"},
	}
	result, err := ApplySubscriptionPipeline(nodes, `{"emoji":true,"emojiRemoveOld":true,"sort":"name"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.Nodes[0].Name, "🇭🇰") {
		t.Fatalf("expected HK flag, got %q", result.Nodes[0].Name)
	}
	if !strings.HasPrefix(result.Nodes[1].Name, "🇯🇵") {
		t.Fatalf("expected JP flag, got %q", result.Nodes[1].Name)
	}
}

func TestApplySubscriptionPipelineRegexSort(t *testing.T) {
	nodes := []models.Node{
		{Name: "美国 01", Link: "ss://a#us"},
		{Name: "香港 01", Link: "ss://b#hk"},
		{Name: "日本 01", Link: "ss://c#jp"},
	}
	result, err := ApplySubscriptionPipeline(nodes, `{"regexSort":"香港|日本|美国"}`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"香港 01", "日本 01", "美国 01"}
	for i, name := range want {
		if result.Nodes[i].Name != name {
			t.Fatalf("regex sort wrong at %d: %#v", i, result.Nodes)
		}
	}
}

func TestApplySubscriptionPipelineScript(t *testing.T) {
	nodes := []models.Node{
		{Name: "香港 01", Link: "ss://a#香港"},
		{Name: "日本 01", Link: "ss://b#日本"},
	}
	script := `{"script":"function operator(proxies){ return proxies.filter(function(p){ return p.name.indexOf('香港') >= 0; }); }"}`
	result, err := ApplySubscriptionPipeline(nodes, script)
	if err != nil {
		t.Fatal(err)
	}
	if result.After != 1 || result.Nodes[0].Name != "香港 01" {
		t.Fatalf("unexpected script result: %#v", result)
	}
}
