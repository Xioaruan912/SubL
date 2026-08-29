package api

import (
	"testing"

	"ppeelink/models"
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
