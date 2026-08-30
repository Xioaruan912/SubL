package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
	"ppeelink/models"
	"ppeelink/node"
)

type safePublishRequest struct {
	SubscriptionID int    `json:"subscriptionId"`
	Template       string `json:"template"`
	Client         string `json:"client"`
}

type safePublishReport struct {
	SubscriptionID int                        `json:"subscriptionId"`
	Template       string                     `json:"template"`
	Client         string                     `json:"client"`
	Preflight      templatePreflightReport    `json:"preflight"`
	Regression     []routingRegressionResult  `json:"regression"`
	NodeCapability []templateClientCapability `json:"nodeCapability"`
	Egress         *egressPlanResponse        `json:"egress,omitempty"`
	PreviousLKG    uint                       `json:"previousLastKnownGoodId,omitempty"`
	PreviousSHA    string                     `json:"previousSha256,omitempty"`
	CandidateSHA   string                     `json:"candidateSha256"`
	CandidateBytes int                        `json:"candidateBytes"`
	Changed        bool                       `json:"changed"`
	PublishedID    uint                       `json:"publishedArtifactId,omitempty"`
	Published      bool                       `json:"published"`
}

func capabilityMatrixFromNodes(nodes []models.Node) []templateClientCapability {
	proxies := make([]map[string]interface{}, 0, len(nodes))
	for _, item := range nodes {
		raw := strings.TrimSpace(item.Link)
		if raw == "" || strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
			continue
		}
		scheme := strings.ToLower(strings.SplitN(raw, "://", 2)[0])
		aliases := map[string]string{"hy2": "hysteria2", "socks": "socks5"}
		if mapped := aliases[scheme]; mapped != "" {
			scheme = mapped
		}
		proxy := map[string]interface{}{"type": scheme}
		if parsed, err := urlpkg.Parse(raw); err == nil {
			q := parsed.Query()
			network := strings.ToLower(q.Get("type"))
			if network == "" {
				network = strings.ToLower(q.Get("network"))
			}
			if network != "" {
				proxy["network"] = network
			}
			security := strings.ToLower(q.Get("security"))
			if security == "reality" {
				proxy["reality-opts"] = map[string]interface{}{"enabled": true}
			}
			for _, field := range []string{"udp", "tfo", "mptcp"} {
				if v := q.Get(field); v != "" {
					proxy[field] = v
				}
			}
		}
		proxies = append(proxies, proxy)
	}
	return buildClientCapabilityMatrix(proxies)
}

func safePublishClientCapability(report templatePreflightReport, client string) (templateClientCapability, bool) {
	want := map[string]string{"clash": "Clash/Mihomo", "surge": "Surge", "loon": "Loon"}[strings.ToLower(client)]
	for _, item := range report.CapabilityMatrix {
		if item.Client == want {
			return item, true
		}
	}
	return templateClientCapability{}, false
}

func subscriptionURLsForPublish(ctx context.Context, sub *models.Subcription) ([]string, error) {
	if err := mergeGroupNodes(sub); err != nil {
		return nil, err
	}
	urls := []string{}
	client := &http.Client{Timeout: 20 * time.Second}
	for _, item := range sub.Nodes {
		link := strings.TrimSpace(item.Link)
		if link == "" {
			continue
		}
		if strings.Contains(link, ",") {
			for _, part := range strings.Split(link, ",") {
				if v := strings.TrimSpace(part); v != "" {
					urls = append(urls, v)
				}
			}
			continue
		}
		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
			if err != nil {
				return nil, err
			}
			resp, err := client.Do(req)
			if err != nil {
				return nil, err
			}
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
			_ = resp.Body.Close()
			if readErr != nil {
				return nil, readErr
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return nil, fmt.Errorf("订阅源 HTTP %d", resp.StatusCode)
			}
			decoded := node.Base64Decode(string(body))
			for _, part := range strings.Split(decoded, "\n") {
				if v := strings.TrimSpace(part); v != "" {
					urls = append(urls, v)
				}
			}
			continue
		}
		urls = append(urls, link)
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("订阅没有可用于发布的节点")
	}
	return urls, nil
}

func buildCandidateSubscription(ctx context.Context, sub *models.Subcription, clientName, template string) ([]byte, string, error) {
	urls, err := subscriptionURLsForPublish(ctx, sub)
	if err != nil {
		return nil, "", err
	}
	var cfg node.SqlConfig
	if err := json.Unmarshal([]byte(sub.Config), &cfg); err != nil {
		return nil, "", err
	}
	templatePath := "./template/" + template
	clientName = strings.ToLower(clientName)
	switch clientName {
	case "clash":
		cfg.Clash = templatePath
	case "surge":
		cfg.Surge = templatePath
	case "loon":
		cfg.Loon = templatePath
	default:
		return nil, "", fmt.Errorf("安全发布仅支持 clash/surge/loon")
	}
	rawCfg, _ := json.Marshal(cfg)
	var body []byte
	switch clientName {
	case "clash":
		body, err = node.EncodeClash(urls, cfg)
	case "surge":
		var text string
		text, err = node.EncodeSurge(urls, cfg)
		body = []byte(text)
	case "loon":
		var text string
		text, err = node.EncodeLoon(urls, cfg)
		body = []byte(text)
	}
	if err != nil {
		return nil, "", err
	}
	if err := validateSubscriptionContent(clientName, body); err != nil {
		return nil, "", err
	}
	return body, string(rawCfg), nil
}

func rulesChecksumForTemplate(content string) string {
	var catalogs []models.RuleCatalog
	_ = models.DB.Where("checksum <> ''").Order("external_id asc").Find(&catalogs).Error
	refs := []string{}
	lower := strings.ToLower(content)
	for _, item := range catalogs {
		if strings.Contains(lower, strings.ToLower(item.Name)) {
			refs = append(refs, item.ExternalID+":"+item.Checksum)
		}
	}
	return shaHex([]byte(strings.Join(refs, "\n")))
}

func runSafePublish(ctx context.Context, req safePublishRequest) (safePublishReport, error) {
	req.Client = strings.ToLower(strings.TrimSpace(req.Client))
	req.Template = strings.TrimSpace(req.Template)
	if req.SubscriptionID <= 0 || req.Template == "" {
		return safePublishReport{}, fmt.Errorf("订阅和模板不能为空")
	}
	if req.Client != "clash" && req.Client != "surge" && req.Client != "loon" {
		return safePublishReport{}, fmt.Errorf("目标客户端仅支持 clash/surge/loon")
	}
	path, err := safeFilePath(req.Template)
	if err != nil {
		return safePublishReport{}, err
	}
	templateBody, err := os.ReadFile(path)
	if err != nil {
		return safePublishReport{}, err
	}
	content := string(templateBody)
	var sub models.Subcription
	if err := models.DB.First(&sub, req.SubscriptionID).Error; err != nil {
		return safePublishReport{}, fmt.Errorf("订阅不存在")
	}
	if err := mergeGroupNodes(&sub); err != nil {
		return safePublishReport{}, err
	}
	cases, _ := activeRegressionCases()
	domains := regressionDomains(cases)
	if len(domains) == 0 {
		targets, _ := enabledNodeEgressTargets()
		for _, target := range targets {
			domains = append(domains, target.Domain)
		}
	}
	preflight := buildTemplatePreflight(ctx, req.Template, content, domains)
	report := safePublishReport{SubscriptionID: req.SubscriptionID, Template: req.Template, Client: req.Client, Preflight: preflight, NodeCapability: capabilityMatrixFromNodes(sub.Nodes)}
	if !preflight.Valid {
		return report, fmt.Errorf("模板预检失败，存在阻止发布的问题")
	}
	if capability, ok := safePublishClientCapability(preflight, req.Client); ok && capability.Status == "error" {
		return report, fmt.Errorf("目标客户端协议兼容性检查失败: %s", capability.Detail)
	}
	want := map[string]string{"clash": "Clash/Mihomo", "surge": "Surge", "loon": "Loon"}[req.Client]
	for _, capability := range report.NodeCapability {
		if capability.Client == want && capability.Status == "error" {
			return report, fmt.Errorf("订阅节点与目标客户端不兼容: %s", strings.Join(capability.Unsupported, ","))
		}
	}
	regression, _ := evaluateRoutingRegression(ctx, req.Template, content, cases)
	report.Regression = regression
	for _, item := range regression {
		if !item.Passed {
			return report, fmt.Errorf("分流回归失败: %s", item.Case.Name)
		}
	}
	candidate, newConfig, err := buildCandidateSubscription(ctx, &sub, req.Client, req.Template)
	if err != nil {
		return report, err
	}
	report.CandidateSHA = shaHex(candidate)
	report.CandidateBytes = len(candidate)
	if old, oldErr := lastKnownGoodArtifact(req.SubscriptionID, req.Client); oldErr == nil {
		report.PreviousLKG = old.ID
		report.PreviousSHA = old.ContentChecksum
		report.Changed = old.ContentChecksum != report.CandidateSHA
	} else {
		report.Changed = true
	}
	if req.Client == "clash" {
		if err := mergeGroupNodes(&sub); err != nil {
			return report, err
		}
		stats, _ := models.GetNodeQualityStats(time.Now().Add(-24 * time.Hour))
		targetStats, _ := models.GetNodeTargetQualityStats(time.Now().Add(-24 * time.Hour))
		sceneStats, _ := models.GetNodeSceneQualityStats(time.Now().Add(-24 * time.Hour))
		targets, _ := enabledNodeEgressTargets()
		plan := buildEgressPlan(ctx, &sub, req.Template, content, stats, planQualityMatrix{Targets: targetStats, Scenes: sceneStats}, targets, node.RunEgressTestTargets)
		report.Egress = &plan
		for _, item := range plan.Items {
			if item.SelectedNode == nil || item.Result == nil || (item.Result.Status != "available" && item.Result.Status != "reachable") {
				return report, fmt.Errorf("真实出口验证失败: %s", item.Name)
			}
		}
	}
	inputDigest := shaHex([]byte(fmt.Sprintf("%d\x00%s\x00%s\x00%s", sub.ID, req.Client, newConfig, report.CandidateSHA)))
	testReport, _ := json.Marshal(report)
	artifact := models.SubscriptionArtifact{SubscriptionID: req.SubscriptionID, Client: req.Client, InputDigest: inputDigest, TemplateName: req.Template, TemplateChecksum: shaHex(templateBody), RulesChecksum: rulesChecksumForTemplate(content), ContentChecksum: report.CandidateSHA, ByteSize: len(candidate), ValidationStatus: "valid", TestStatus: "passed", TestReportJSON: string(testReport), Content: candidate}
	err = models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&artifact).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Subcription{}).Where("id = ?", req.SubscriptionID).Update("config", newConfig).Error; err != nil {
			return err
		}
		var pointer models.SubscriptionArtifactPointer
		findErr := tx.Where("subscription_id = ? AND client = ?", req.SubscriptionID, req.Client).First(&pointer).Error
		if findErr == nil {
			return tx.Model(&pointer).Update("last_known_good_artifact_id", artifact.ID).Error
		}
		if findErr != gorm.ErrRecordNotFound {
			return findErr
		}
		return tx.Create(&models.SubscriptionArtifactPointer{SubscriptionID: req.SubscriptionID, Client: req.Client, LastKnownGoodArtifactID: artifact.ID}).Error
	})
	if err != nil {
		return report, err
	}
	report.Published = true
	report.PublishedID = artifact.ID
	return report, nil
}
