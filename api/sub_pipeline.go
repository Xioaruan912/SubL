package api

import (
	"encoding/json"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"ppeelink/models"
	"ppeelink/node"

	"github.com/gin-gonic/gin"
)

type SubscriptionPipeline struct {
	Include           string   `json:"include"`
	Exclude           string   `json:"exclude"`
	RenamePattern     string   `json:"renamePattern"`
	RenameReplacement string   `json:"renameReplacement"`
	Protocols         []string `json:"protocols"`
	Sort              string   `json:"sort"`
	Dedupe            bool     `json:"dedupe"`
	MaxNodes          int      `json:"maxNodes"`
}

type PipelinePreview struct {
	Before   int            `json:"before"`
	After    int            `json:"after"`
	Rejected map[string]int `json:"rejected"`
	Nodes    []models.Node  `json:"nodes"`
}

func protocolOf(link string) string {
	if i := strings.Index(link, "://"); i > 0 {
		return strings.ToLower(link[:i])
	}
	return "unknown"
}

func renameNode(n models.Node, pattern *regexp.Regexp, replacement string) models.Node {
	newName := pattern.ReplaceAllString(n.Name, replacement)
	if newName == n.Name || newName == "" {
		return n
	}
	n.Name = newName
	if parsed, err := url.Parse(n.Link); err == nil && parsed.Scheme != "" {
		parsed.Fragment = newName
		n.Link = parsed.String()
	}
	return n
}

func ApplySubscriptionPipeline(nodes []models.Node, raw string) (PipelinePreview, error) {
	preview := PipelinePreview{Before: len(nodes), Rejected: map[string]int{}, Nodes: make([]models.Node, 0, len(nodes))}
	if strings.TrimSpace(raw) == "" {
		preview.Nodes = append(preview.Nodes, nodes...)
		preview.After = len(preview.Nodes)
		return preview, nil
	}
	var cfg SubscriptionPipeline
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return preview, err
	}
	var include, exclude, rename *regexp.Regexp
	var err error
	if cfg.Include != "" {
		if include, err = regexp.Compile(cfg.Include); err != nil {
			return preview, err
		}
	}
	if cfg.Exclude != "" {
		if exclude, err = regexp.Compile(cfg.Exclude); err != nil {
			return preview, err
		}
	}
	if cfg.RenamePattern != "" {
		if rename, err = regexp.Compile(cfg.RenamePattern); err != nil {
			return preview, err
		}
	}
	allowed := map[string]bool{}
	for _, p := range cfg.Protocols {
		allowed[strings.ToLower(p)] = true
	}
	seen := map[string]bool{}
	for _, item := range nodes {
		if include != nil && !include.MatchString(item.Name) {
			preview.Rejected["不匹配包含规则"]++
			continue
		}
		if exclude != nil && exclude.MatchString(item.Name) {
			preview.Rejected["命中排除规则"]++
			continue
		}
		if len(allowed) > 0 && !allowed[protocolOf(item.Link)] {
			preview.Rejected["协议过滤"]++
			continue
		}
		key := strings.ToLower(strings.TrimSpace(item.Link))
		if cfg.Dedupe && seen[key] {
			preview.Rejected["重复节点"]++
			continue
		}
		seen[key] = true
		if rename != nil {
			item = renameNode(item, rename, cfg.RenameReplacement)
		}
		preview.Nodes = append(preview.Nodes, item)
	}
	if cfg.Sort == "name" {
		sort.SliceStable(preview.Nodes, func(i, j int) bool { return preview.Nodes[i].Name < preview.Nodes[j].Name })
	} else if cfg.Sort == "country" {
		sort.SliceStable(preview.Nodes, func(i, j int) bool {
			hi, _ := node.ExtractServerHost(preview.Nodes[i].Link)
			hj, _ := node.ExtractServerHost(preview.Nodes[j].Link)
			return node.LookupCountry(hi) < node.LookupCountry(hj)
		})
	} else if cfg.Sort == "latency" || cfg.Sort == "quality" {
		stats, _ := models.GetNodeQualityStats(time.Now().Add(-24 * time.Hour))
		sort.SliceStable(preview.Nodes, func(i, j int) bool {
			a, aok := stats[preview.Nodes[i].ID]
			b, bok := stats[preview.Nodes[j].ID]
			if !aok {
				return false
			}
			if !bok {
				return true
			}
			if cfg.Sort == "quality" {
				return a.Score > b.Score
			}
			if a.AverageRtt < 0 {
				return false
			}
			if b.AverageRtt < 0 {
				return true
			}
			return a.AverageRtt < b.AverageRtt
		})
	}
	if cfg.MaxNodes > 0 && len(preview.Nodes) > cfg.MaxNodes {
		preview.Rejected["超过数量上限"] += len(preview.Nodes) - cfg.MaxNodes
		preview.Nodes = preview.Nodes[:cfg.MaxNodes]
	}
	preview.After = len(preview.Nodes)
	return preview, nil
}

func SubPipelinePreview(c *gin.Context) {
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"msg": "订阅 id 无效"})
		return
	}
	var sub models.Subcription
	if err := models.DB.First(&sub, id).Error; err != nil {
		c.JSON(404, gin.H{"msg": "订阅不存在"})
		return
	}
	persistedPipeline := sub.Pipeline
	sub.Pipeline = "" // preview must start from the unprocessed node set
	if err := mergeGroupNodes(&sub); err != nil {
		c.JSON(500, gin.H{"msg": "合并节点失败"})
		return
	}
	raw := c.PostForm("pipeline")
	if raw == "" {
		raw = persistedPipeline
	}
	preview, err := ApplySubscriptionPipeline(sub.Nodes, raw)
	if err != nil {
		c.JSON(400, gin.H{"msg": "处理规则无效: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": preview, "msg": "处理链预览"})
}
