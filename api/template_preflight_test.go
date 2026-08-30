package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func hasPreflightIssue(report templatePreflightReport, code string) bool {
	for _, issue := range report.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func TestTemplatePreflightClashExplainsPolicyChain(t *testing.T) {
	content := `proxies:
  - name: US-01
    type: vless
    server: example.com
    port: 443
    network: grpc
    grpc-opts:
      grpc-service-name: edge
proxy-groups:
  - name: AI
    type: select
    proxies: [Best]
  - name: Best
    type: url-test
    proxies: [US-01]
rules:
  - DOMAIN,chatgpt.com,AI
  - MATCH,DIRECT
`
	report := buildTemplatePreflight(context.Background(), "clash.yaml", content, []string{"chatgpt.com", "example.net"})
	if !report.Valid || report.Summary.Errors != 0 {
		t.Fatalf("expected valid report: %#v", report)
	}
	if len(report.Routes) != 2 || report.Routes[0].MatchedRule != "DOMAIN,chatgpt.com,AI" {
		t.Fatalf("unexpected routes: %#v", report.Routes)
	}
	wantChain := []string{"AI", "Best", "US-01"}
	if len(report.Routes[0].Chain) != len(wantChain) {
		t.Fatalf("unexpected chain: %#v", report.Routes[0].Chain)
	}
	for index, value := range wantChain {
		if report.Routes[0].Chain[index] != value {
			t.Fatalf("unexpected chain: %#v", report.Routes[0].Chain)
		}
	}
	if report.Routes[1].Policy != "DIRECT" {
		t.Fatalf("MATCH fallback not explained: %#v", report.Routes[1])
	}
}

func TestTemplatePreflightClashBlocksSemanticErrors(t *testing.T) {
	content := `proxies:
  - name: Broken
    type: vless
    network: kcp
proxy-groups:
  - name: A
    type: select
    proxies: [B]
  - name: B
    type: select
    proxies: [A]
    filter: "["
rules:
  - RULE-SET,Missing,A
  - MATCH,A
`
	report := buildTemplatePreflight(context.Background(), "broken.yaml", content, []string{"example.com"})
	if report.Valid || report.Summary.Errors < 4 {
		t.Fatalf("semantic errors must block publication: %#v", report)
	}
	for _, code := range []string{"network_unknown", "group_filter_invalid", "group_cycle", "rule_provider_missing"} {
		if !hasPreflightIssue(report, code) {
			t.Fatalf("missing issue %s: %#v", code, report.Issues)
		}
	}
}

func TestTemplatePreflightINIExplainsLocalRules(t *testing.T) {
	content := `[General]
loglevel = notify
[Proxy]
US-01 = vmess, example.com, 443, username=id, tls=true
[Proxy Group]
AI = select, US-01
[Rule]
DOMAIN-SUFFIX,openai.com,AI
FINAL,DIRECT
`
	report := buildTemplatePreflight(context.Background(), "surge.conf", content, []string{"api.openai.com"})
	if !report.Valid || report.Format != "surge" {
		t.Fatalf("expected valid Surge report: %#v", report)
	}
	if len(report.Routes) != 1 || report.Routes[0].Policy != "AI" || len(report.Routes[0].Chain) != 2 {
		t.Fatalf("unexpected route explanation: %#v", report.Routes)
	}
}

func TestBuiltInTemplatesPassPreflight(t *testing.T) {
	for _, filename := range []string{"clash.yaml", "surge.conf", "loon.conf"} {
		t.Run(filename, func(t *testing.T) {
			body, err := os.ReadFile("../template/" + filename)
			if err != nil {
				t.Fatal(err)
			}
			report := buildTemplatePreflight(context.Background(), filename, string(body), []string{"gemini.google.com", "chatgpt.com"})
			if !report.Valid {
				t.Fatalf("built-in template must pass preflight: %#v", report.Issues)
			}
		})
	}
}

func TestPreflightTempHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("filename", "clash.yaml")
	_ = writer.WriteField("text", "proxy-groups:\n  - name: Proxy\n    type: select\n    proxies: [DIRECT]\nrules:\n  - DOMAIN,example.com,Proxy\n  - MATCH,DIRECT\n")
	_ = writer.WriteField("domains", "example.com")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.POST("/preflight", PreflightTemp)
	request := httptest.NewRequest(http.MethodPost, "/preflight", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Code string                  `json:"code"`
		Data templatePreflightReport `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != "00000" || !payload.Data.Valid || len(payload.Data.Routes) != 1 {
		t.Fatalf("unexpected handler payload: %#v", payload)
	}
}
