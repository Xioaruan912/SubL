package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"ppeelink/models"
	"ppeelink/node"

	"github.com/gin-gonic/gin"
)

// SubscriptionPipeline 是订阅节点加工链的配置。字段保持向后兼容：
// 旧配置缺失的新字段默认关闭。
type SubscriptionPipeline struct {
	Include           string   `json:"include"`
	Exclude           string   `json:"exclude"`
	RenamePattern     string   `json:"renamePattern"`
	RenameReplacement string   `json:"renameReplacement"`
	Protocols         []string `json:"protocols"`
	Sort              string   `json:"sort"`
	RegexSort         string   `json:"regexSort"`
	Dedupe            bool     `json:"dedupe"`
	MaxNodes          int      `json:"maxNodes"`

	Emoji              bool        `json:"emoji"`
	EmojiRemoveOld     bool        `json:"emojiRemoveOld"`
	EmojiRules         []EmojiRule `json:"emojiRules"`
	ExcludePlaceholder bool        `json:"excludePlaceholder"`
	PlaceholderTerms   []string    `json:"placeholderTerms"`
	DeletePattern      string      `json:"deletePattern"`
	MaxMultiplier      *float64    `json:"maxMultiplier"`
	SetUDP             *bool       `json:"setUdp"`
	SetTFO             *bool       `json:"setTfo"`
	SetSkipCertVerify  *bool       `json:"setSkipCertVerify"`
	ResolveDomain      bool        `json:"resolveDomain"`
	Script             string      `json:"script"`
}

type EmojiRule struct {
	Match string `json:"match"`
	Emoji string `json:"emoji"`
}

type PipelinePreview struct {
	Before   int            `json:"before"`
	After    int            `json:"after"`
	Rejected map[string]int `json:"rejected"`
	Nodes    []models.Node  `json:"nodes"`
}

// defaultPlaceholderTerms 机场订阅里常见的占位/广告节点关键字。
var defaultPlaceholderTerms = []string{
	"官网", "到期", "剩余", "流量", "套餐", "续费", "余额", "过期", "expire",
	"traffic", "重置", "客服", "电报", "telegram", "邀请", "注册", "试用",
	"购买", "优惠", "活动", "公告", "通知", "维护", "更新", "防失联", "加群",
	"白嫖", "失联", "官网地址", "节点异常", "订阅",
}

type pipelineItem struct {
	ID  int
	P   node.Outbound
	Raw models.Node
}

func protocolOf(link string) string {
	if i := strings.Index(link, "://"); i > 0 {
		return strings.ToLower(link[:i])
	}
	return "unknown"
}

func decodeNodeFlags(raw string) node.NodeFlags {
	var f node.NodeFlags
	if strings.TrimSpace(raw) == "" {
		return f
	}
	_ = json.Unmarshal([]byte(raw), &f)
	return f
}

func encodeNodeFlags(f node.NodeFlags) string {
	if f.Empty() {
		return ""
	}
	b, err := json.Marshal(f)
	if err != nil {
		return ""
	}
	return string(b)
}

func stripFlagEmojis(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r >= 0x1F1E6 && r <= 0x1F1FF {
			if i+1 < len(runes) && runes[i+1] >= 0x1F1E6 && runes[i+1] <= 0x1F1FF {
				i++
			}
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// nodesToItems 将模型节点解析为规范化代理项；解析失败时保留原样，保证
// 仅基于名称/协议的操作（过滤、重命名、排序）仍然可用。
func nodesToItems(nodes []models.Node) []pipelineItem {
	items := make([]pipelineItem, 0, len(nodes))
	for _, n := range nodes {
		p, err := node.ParseOutbound(n.Link)
		if err != nil {
			p = node.Outbound{Link: n.Link, Name: n.Name, Type: protocolOf(n.Link)}
			p.Server, p.Port = node.ExtractServerHost(n.Link)
		}
		if n.Name != "" {
			p.Name = n.Name // 以存储的节点名为准，避免解析结果覆盖用户命名
		}
		if f := decodeNodeFlags(n.Flags); !f.Empty() {
			p.UDP, p.TFO, p.SkipCertVerify = f.UDP, f.TFO, f.SkipCertVerify
		}
		items = append(items, pipelineItem{ID: n.ID, P: p, Raw: n})
	}
	return items
}

func itemsToNodes(items []pipelineItem) []models.Node {
	out := make([]models.Node, 0, len(items))
	for _, it := range items {
		n := it.Raw
		n.ID = it.ID
		n.Name = it.P.Name
		n.Link = it.P.Link
		n.Flags = encodeNodeFlags(it.P.Flags())
		out = append(out, n)
	}
	return out
}

// ApplySubscriptionPipeline applies the node processing chain and returns a
// preview with per-rule rejection counts.
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

	var include, exclude, rename, del *regexp.Regexp
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
	if cfg.DeletePattern != "" {
		if del, err = regexp.Compile(cfg.DeletePattern); err != nil {
			return preview, err
		}
	}

	allowed := map[string]bool{}
	for _, p := range cfg.Protocols {
		allowed[strings.ToLower(strings.TrimSpace(p))] = true
	}
	placeholder := regexp.MustCompile(placeholderPattern(cfg))

	items := nodesToItems(nodes)
	filtered := make([]pipelineItem, 0, len(items))
	seen := map[string]bool{}
	for _, it := range items {
		name := it.P.Name
		if include != nil && !include.MatchString(name) {
			preview.Rejected["不匹配包含规则"]++
			continue
		}
		if exclude != nil && exclude.MatchString(name) {
			preview.Rejected["命中排除规则"]++
			continue
		}
		if len(allowed) > 0 && !allowed[strings.ToLower(it.P.Type)] {
			preview.Rejected["协议过滤"]++
			continue
		}
		if cfg.ExcludePlaceholder && placeholder != nil && placeholder.MatchString(name) {
			preview.Rejected["占位/广告节点"]++
			continue
		}
		if del != nil && del.MatchString(name) {
			preview.Rejected["命中删除规则"]++
			continue
		}
		if cfg.MaxMultiplier != nil && *cfg.MaxMultiplier > 0 {
			if m := node.ParseMultiplier(name); m > *cfg.MaxMultiplier {
				preview.Rejected["倍率超限"]++
				continue
			}
		}
		key := dedupeKey(it.P)
		if cfg.Dedupe && seen[key] {
			preview.Rejected["重复节点"]++
			continue
		}
		seen[key] = true
		filtered = append(filtered, it)
	}

	if rename != nil {
		for i := range filtered {
			newName := rename.ReplaceAllString(filtered[i].P.Name, cfg.RenameReplacement)
			if newName != "" && newName != filtered[i].P.Name {
				filtered[i].P = filtered[i].P.WithName(newName)
			}
		}
	}

	if cfg.Emoji {
		applyEmoji(filtered, cfg)
	}

	if cfg.SetUDP != nil || cfg.SetTFO != nil || cfg.SetSkipCertVerify != nil {
		for i := range filtered {
			if cfg.SetUDP != nil {
				filtered[i].P.UDP = *cfg.SetUDP
			}
			if cfg.SetTFO != nil {
				filtered[i].P.TFO = *cfg.SetTFO
			}
			if cfg.SetSkipCertVerify != nil {
				filtered[i].P.SkipCertVerify = *cfg.SetSkipCertVerify
			}
		}
	}

	if cfg.ResolveDomain {
		for i := range filtered {
			if ip, ok := node.ResolveDomain(filtered[i].P.Server); ok {
				filtered[i].P.Server = ip
				filtered[i].P.Link = filtered[i].P.ToLink()
			}
		}
	}

	if strings.TrimSpace(cfg.Script) != "" {
		sc, err := runScriptOperator(filtered, cfg.Script)
		if err != nil {
			return preview, err
		}
		filtered = sc
	}

	sortItems(filtered, cfg)

	if cfg.MaxNodes > 0 && len(filtered) > cfg.MaxNodes {
		preview.Rejected["超过数量上限"] += len(filtered) - cfg.MaxNodes
		filtered = filtered[:cfg.MaxNodes]
	}

	preview.Nodes = itemsToNodes(filtered)
	preview.After = len(preview.Nodes)
	return preview, nil
}

func placeholderPattern(cfg SubscriptionPipeline) string {
	terms := cfg.PlaceholderTerms
	if len(terms) == 0 {
		terms = defaultPlaceholderTerms
	}
	escaped := make([]string, 0, len(terms))
	for _, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		escaped = append(escaped, regexp.QuoteMeta(t))
	}
	if len(escaped) == 0 {
		return ""
	}
	return "(?i)(" + strings.Join(escaped, "|") + ")"
}

func dedupeKey(p node.Outbound) string {
	if p.Server != "" && p.Port > 0 {
		return strings.ToLower(fmt.Sprintf("%s|%d|%s", p.Type, p.Port, p.Server))
	}
	return strings.ToLower(strings.TrimSpace(p.Link))
}

func applyEmoji(items []pipelineItem, cfg SubscriptionPipeline) {
	custom := make([]EmojiRule, 0, len(cfg.EmojiRules))
	for _, r := range cfg.EmojiRules {
		if strings.TrimSpace(r.Match) != "" && strings.TrimSpace(r.Emoji) != "" {
			custom = append(custom, r)
		}
	}
	for i := range items {
		p := items[i].P
		name := p.Name
		if cfg.EmojiRemoveOld {
			name = stripFlagEmojis(name)
		}
		lower := strings.ToLower(name)
		emoji := ""
		for _, r := range custom {
			if strings.Contains(lower, strings.ToLower(r.Match)) {
				emoji = r.Emoji
				break
			}
		}
		if emoji == "" {
			code := p.CountryCode()
			if code == "" {
				code = node.CountryFromName(name)
			}
			emoji = node.FlagEmoji(code)
		}
		if emoji == "" {
			if name != p.Name {
				items[i].P = p.WithName(name)
			}
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(name), emoji) {
			name = emoji + " " + strings.TrimSpace(name)
		}
		items[i].P = p.WithName(name)
	}
}

func sortItems(items []pipelineItem, cfg SubscriptionPipeline) {
	switch cfg.Sort {
	case "name":
		sort.SliceStable(items, func(i, j int) bool { return items[i].P.Name < items[j].P.Name })
	case "country":
		sort.SliceStable(items, func(i, j int) bool {
			return countryOf(items[i].P) < countryOf(items[j].P)
		})
	case "latency", "quality":
		stats, _ := models.GetNodeQualityStats(time.Now().Add(-24 * time.Hour))
		sort.SliceStable(items, func(i, j int) bool {
			a, aok := stats[items[i].ID]
			b, bok := stats[items[j].ID]
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
	if strings.TrimSpace(cfg.RegexSort) != "" {
		applyRegexSort(items, cfg.RegexSort)
	}
}

func applyRegexSort(items []pipelineItem, pattern string) {
	keywords := make([]string, 0)
	for _, k := range strings.Split(pattern, "|") {
		if k = strings.TrimSpace(k); k != "" {
			keywords = append(keywords, strings.ToLower(k))
		}
	}
	if len(keywords) == 0 {
		return
	}
	rank := func(p node.Outbound) int {
		name := strings.ToLower(p.Name)
		for i, k := range keywords {
			if strings.Contains(name, k) {
				return i
			}
		}
		return len(keywords)
	}
	sort.SliceStable(items, func(i, j int) bool {
		ri, rj := rank(items[i].P), rank(items[j].P)
		if ri != rj {
			return ri < rj
		}
		return items[i].P.Name < items[j].P.Name
	})
}

func countryOf(p node.Outbound) string {
	if code := p.CountryCode(); code != "" {
		return code
	}
	return node.CountryFromName(p.Name)
}

func runScriptOperator(items []pipelineItem, code string) ([]pipelineItem, error) {
	proxies := make([]node.Outbound, len(items))
	for i := range items {
		proxies[i] = items[i].P
		proxies[i].ScriptID = i + 1
	}
	result, err := node.RunProxyScript(code, proxies, 0)
	if err != nil {
		return nil, err
	}
	out := make([]pipelineItem, 0, len(result))
	for _, p := range result {
		if p.ScriptID >= 1 && p.ScriptID <= len(items) {
			it := items[p.ScriptID-1]
			it.P = p
			out = append(out, it)
			continue
		}
		out = append(out, pipelineItem{ID: 0, P: p, Raw: models.Node{Name: p.Name, Link: p.Link}})
	}
	return out, nil
}

// pipelineWithOverrides 用订阅 URL 上的查询参数覆盖已保存的处理链，实现
// 免维护的临时分发（例如 &filter=香港|日本、&type=vless,ss、&emoji=true）。
func pipelineWithOverrides(base string, c *gin.Context) string {
	q := c.Request.URL.Query()
	keys := []string{"include", "exclude", "filter", "rename", "renameTo", "type", "sort", "maxNodes", "emoji"}
	present := false
	for _, k := range keys {
		if q.Get(k) != "" {
			present = true
			break
		}
	}
	if !present {
		return base
	}
	var cfg SubscriptionPipeline
	if strings.TrimSpace(base) != "" {
		_ = json.Unmarshal([]byte(base), &cfg)
	}
	if v := q.Get("include"); v != "" {
		cfg.Include = v
	}
	if v := q.Get("exclude"); v != "" {
		cfg.Exclude = v
	}
	if v := q.Get("filter"); v != "" {
		cfg.Include = v
	}
	if v := q.Get("rename"); v != "" {
		cfg.RenamePattern = v
		cfg.RenameReplacement = q.Get("renameTo")
	}
	if v := q.Get("type"); v != "" {
		parts := []string{}
		for _, p := range strings.Split(v, ",") {
			if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
				parts = append(parts, p)
			}
		}
		cfg.Protocols = parts
	}
	if v := q.Get("sort"); v != "" {
		cfg.Sort = v
	}
	if v := q.Get("maxNodes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxNodes = n
		}
	}
	if q.Get("emoji") == "true" {
		cfg.Emoji = true
		cfg.EmojiRemoveOld = true
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return base
	}
	return string(b)
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
	if err := ensureScriptAllowed(c, raw); err != nil {
		c.JSON(403, gin.H{"code": 403, "msg": err.Error()})
		return
	}
	preview, err := ApplySubscriptionPipeline(sub.Nodes, raw)
	if err != nil {
		c.JSON(400, gin.H{"msg": "处理规则无效: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": preview, "msg": "处理链预览"})
}

// ensureScriptAllowed 仅允许管理员使用脚本操作符。
func ensureScriptAllowed(c *gin.Context, raw string) error {
	if !strings.Contains(raw, "\"script\"") {
		return nil
	}
	var cfg SubscriptionPipeline
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil
	}
	if strings.TrimSpace(cfg.Script) == "" {
		return nil
	}
	authType, _ := c.Get("authType")
	if t, _ := authType.(string); t == "api-token" {
		scopes, _ := c.Get("apiScopes")
		if s, _ := scopes.(string); !strings.Contains(strings.ToLower(s), "admin") {
			return fmt.Errorf("脚本操作符需要 admin 权限的 API Token")
		}
	}
	return nil
}
