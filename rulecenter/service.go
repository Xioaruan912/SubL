package rulecenter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"ppeelink/models"
	"ppeelink/node"
	"gorm.io/gorm"
)

const cacheRoot = "db/rules-cache"

var httpClient = &http.Client{Timeout: 12 * time.Second}
var syncMu sync.Mutex

var sourceDefs = []models.RuleSource{
	{Key:"shunt_rules", Name:"ShuntRules", Type:"github-readme", Repo:"luestr/ShuntRules", Branch:"main", BaseURL:"https://rule.kelee.one", Enabled:true},
	{Key:"ios_rule_script", Name:"ios_rule_script", Type:"github-contents", Repo:"blackmatrix7/ios_rule_script", Branch:"master", BaseURL:"https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master", Enabled:true},
}

func EnsureSources() error {
	for _, s := range sourceDefs {
		var existing models.RuleSource
		err := models.DB.Where("key = ?", s.Key).First(&existing).Error
		if err == nil { continue }
		if err := models.DB.Create(&s).Error; err != nil { return err }
	}
	return nil
}

func SyncAll(ctx context.Context) error {
	syncMu.Lock()
	defer syncMu.Unlock()
	if err := EnsureSources(); err != nil { return err }
	var errs []string
	for _, def := range sourceDefs {
		if err := SyncSource(ctx, def.Key); err != nil { errs = append(errs, def.Key+": "+err.Error()) }
	}
	if len(errs) > 0 { return errors.New(strings.Join(errs, "; ")) }
	return nil
}

func SyncSource(ctx context.Context, key string) error {
	var src models.RuleSource
	if err := models.DB.Where("key = ?", key).First(&src).Error; err != nil { return err }
	var items []RuleItem
	var err error
	switch key {
	case "shunt_rules": items, err = syncShuntRules(ctx)
	case "ios_rule_script": items, err = syncIOSRuleScript(ctx)
	default: err = fmt.Errorf("unknown rule source: %s", key)
	}
	now := time.Now()
	if err != nil {
		models.DB.Model(&src).Updates(map[string]any{"last_sync_at": &now, "last_sync_status":"error", "last_sync_error":err.Error()})
		return err
	}
	// Replace only this source's metadata after a successful scan. Keep the old
	// catalog intact if any write fails midway.
	if err := models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("source_key = ?", key).Delete(&models.RuleCatalog{}).Error; err != nil { return err }
		for _, item := range items {
			meta, _ := json.Marshal(item.Metadata)
			rec := models.RuleCatalog{SourceKey:item.SourceKey, ExternalID:item.ExternalID, Name:item.Name, Category:item.Category, Platform:item.Platform, Format:item.Format, URL:item.URL, LocalPath:item.LocalPath, RuleCount:item.RuleCount, RemoteUpdate:item.UpdatedAt, Checksum:item.Checksum, MetadataJSON:string(meta)}
			if err := tx.Create(&rec).Error; err != nil { return err }
		}
		return nil
	}); err != nil { return err }
	return models.DB.Model(&src).Updates(map[string]any{"last_sync_at": &now, "last_sync_status":"ok", "last_sync_error":""}).Error
}

func get(ctx context.Context, rawURL string, max int64) ([]byte, http.Header, error) {
	body, header, err := getDirect(ctx, rawURL, max)
	if err == nil { return body, header, nil }
	proxyBody, proxyHeader, proxyErr := getThroughBestNodes(ctx, rawURL, max)
	if proxyErr == nil { return proxyBody, proxyHeader, nil }
	return nil, header, fmt.Errorf("直连失败: %v; 节点回退失败: %v", err, proxyErr)
}

func getDirect(ctx context.Context, rawURL string, max int64) ([]byte, http.Header, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") { return nil, nil, errors.New("invalid rule URL") }
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	req.Header.Set("User-Agent", ruleUserAgent(rawURL))
	resp, err := httpClient.Do(req)
	if err != nil { return nil, nil, err }
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 { return nil, resp.Header, fmt.Errorf("HTTP %d", resp.StatusCode) }
	lr := io.LimitReader(resp.Body, max+1)
	body, err := io.ReadAll(lr)
	if err != nil { return nil, resp.Header, err }
	if int64(len(body)) > max { return nil, resp.Header, errors.New("rule file exceeds size limit") }
	return body, resp.Header, nil
}

func getThroughBestNodes(ctx context.Context, rawURL string, max int64) ([]byte, http.Header, error) {
	nodes, err := models.GetNodeList()
	if err != nil || len(nodes) == 0 { return nil, nil, errors.New("没有可用节点") }
	stats, _ := models.GetNodeQualityStats(time.Now().Add(-24 * time.Hour))
	sort.SliceStable(nodes, func(i, j int) bool {
		si, iok := stats[nodes[i].ID]; sj, jok := stats[nodes[j].ID]
		if iok != jok { return iok }
		if iok && si.Score != sj.Score { return si.Score > sj.Score }
		if iok && si.AverageRtt >= 0 && sj.AverageRtt >= 0 && si.AverageRtt != sj.AverageRtt { return si.AverageRtt < sj.AverageRtt }
		return nodes[i].ID < nodes[j].ID
	})

	// Do not let the highest-scoring nodes all come from one blocked region.
	// First take one strong node per country, then fill the remaining slots by
	// global quality order. This makes a 403 fallback actually geo-diverse.
	const limit = 8
	candidates := make([]models.Node, 0, limit)
	selected := map[int]bool{}
	countries := map[string]bool{}
	for _, candidate := range nodes {
		host, _ := node.ExtractServerHost(candidate.Link)
		country := node.LookupCountry(host)
		if country == "" { country = "unknown" }
		if countries[country] { continue }
		countries[country] = true
		selected[candidate.ID] = true
		candidates = append(candidates, candidate)
		if len(candidates) >= limit { break }
	}
	if len(candidates) < limit {
		for _, candidate := range nodes {
			if selected[candidate.ID] { continue }
			candidates = append(candidates, candidate)
			if len(candidates) >= limit { break }
		}
	}

	var errs []string
	for _, candidate := range candidates {
		body, header, fetchErr := node.FetchURLThroughNode(ctx, candidate.Link, rawURL, ruleUserAgent(rawURL), 7*time.Second, max)
		if fetchErr == nil { return body, header, nil }
		errs = append(errs, candidate.Name+": "+fetchErr.Error())
		if ctx.Err() != nil { break }
	}
	return nil, nil, errors.New(strings.Join(errs, "; "))
}

func ruleUserAgent(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err == nil {
		host := strings.ToLower(u.Hostname())
		if host == "kelee.one" || strings.HasSuffix(host, ".kelee.one") {
			return "clash.meta"
		}
	}
	return "SubLinkX-RuleCenter/1.0"
}

var shuntLinkRE = regexp.MustCompile(`https://rule\.kelee\.one/(Clash|Loon)/([^\s)]+)`) 

func syncShuntRules(ctx context.Context) ([]RuleItem, error) {
	body, _, err := get(ctx, "https://raw.githubusercontent.com/luestr/ShuntRules/main/README.md", 2<<20)
	if err != nil { return nil, err }
	matches := shuntLinkRE.FindAllStringSubmatch(string(body), -1)
	seen := map[string]bool{}
	items := make([]RuleItem,0,len(matches))
	for _, m := range matches {
		platform, file := m[1], strings.TrimSpace(m[2])
		name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(file)), ".")
		if ext == "lsr" { ext = "list" }
		id := "shunt_rules:"+platform+":"+name
		if seen[id] { continue }; seen[id]=true
		items = append(items, RuleItem{ExternalID:id, SourceKey:"shunt_rules", Name:name, Category:CategoryFor(name), Platform:platform, Format:ext, URL:"https://rule.kelee.one/"+platform+"/"+file})
	}
	sort.Slice(items, func(i,j int) bool { if items[i].Platform==items[j].Platform { return items[i].Name<items[j].Name }; return items[i].Platform<items[j].Platform })
	return items,nil
}

type ghContent struct { Name string `json:"name"`; Type string `json:"type"` }

func syncIOSRuleScript(ctx context.Context) ([]RuleItem, error) {
	items := []RuleItem{}
	for _, platform := range []string{"Clash","Surge","Loon"} {
		apiURL := fmt.Sprintf("https://api.github.com/repos/blackmatrix7/ios_rule_script/contents/rule/%s?ref=master", platform)
		body, _, err := get(ctx, apiURL, 8<<20)
		if err != nil { return nil, fmt.Errorf("%s catalog: %w", platform, err) }
		var entries []ghContent
		if err := json.Unmarshal(body,&entries); err != nil { return nil, err }
		for _, e := range entries {
			if e.Type != "dir" || e.Name == "" { continue }
			ext := "list"
			if platform == "Clash" { ext = "yaml" }
			filename := e.Name+"."+ext
			raw := fmt.Sprintf("https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/%s/%s/%s", platform, url.PathEscape(e.Name), url.PathEscape(filename))
			id := "ios_rule_script:"+platform+":"+e.Name
			items = append(items, RuleItem{ExternalID:id, SourceKey:"ios_rule_script", Name:e.Name, Category:CategoryFor(e.Name), Platform:platform, Format:ext, URL:raw})
		}
	}
	return items,nil
}

func CategoryFor(name string) string {
	n := strings.ToLower(name)
	groups := []struct{cat string; keys []string}{
		{"AI", []string{"openai","chatgpt","gemini","claude","anthropic","copilot","perplexity"}},
		{"流媒体", []string{"netflix","youtube","disney","spotify","primevideo","hbo","twitch","bilibili"}},
		{"社交", []string{"telegram","twitter","facebook","instagram","whatsapp","reddit","discord"}},
		{"Apple", []string{"apple","icloud","testflight"}}, {"Google", []string{"google","youtube","gemini"}},
		{"Microsoft", []string{"microsoft","onedrive","office365","xbox"}}, {"开发者", []string{"github","gitlab","docker","npm","pypi"}},
		{"广告/隐私", []string{"advert","adguard","privacy","tracking"}},
	}
	for _, g := range groups { for _, k := range g.keys { if strings.Contains(n,k) { return g.cat } } }
	return "其他"
}

func ListCatalog(source, platform, category, keyword string, page, pageSize int) ([]models.RuleCatalog, int64, error) {
	q := models.DB.Model(&models.RuleCatalog{})
	if source != "" { q=q.Where("source_key = ?",source) }
	if platform != "" { q=q.Where("platform = ?",platform) }
	if category != "" { q=q.Where("category = ?",category) }
	if keyword != "" { q=q.Where("name LIKE ?","%"+keyword+"%") }
	var total int64; if err:=q.Count(&total).Error; err!=nil{return nil,0,err}
	if page<1 {page=1}; if pageSize<1||pageSize>200 {pageSize=48}
	var out []models.RuleCatalog
	err:=q.Order("category asc, name asc, source_key asc").Offset((page-1)*pageSize).Limit(pageSize).Find(&out).Error
	return out,total,err
}

func Sources() ([]SourceStatus,error) {
	if err:=EnsureSources(); err!=nil{return nil,err}
	var srcs []models.RuleSource; if err:=models.DB.Order("id asc").Find(&srcs).Error;err!=nil{return nil,err}
	out:=make([]SourceStatus,0,len(srcs)); for _,s:=range srcs{var count int64;models.DB.Model(&models.RuleCatalog{}).Where("source_key = ?",s.Key).Count(&count);out=append(out,SourceStatus{Key:s.Key,Name:s.Name,Kind:s.Type,Repo:s.Repo,Branch:s.Branch,Enabled:s.Enabled,Status:s.LastSyncStatus,LastSyncAt:s.LastSyncAt,Error:s.LastSyncError,Count:count})};return out,nil
}

func LoadItem(ctx context.Context, externalID string) (RuleItem, []NormalizedRule, error) {
	var rec models.RuleCatalog
	if err:=models.DB.Where("external_id = ?",externalID).First(&rec).Error;err!=nil{return RuleItem{},nil,err}
	local:=cachePath(rec.SourceKey,rec.Platform,rec.Name,rec.Format)
	data, readErr:=os.ReadFile(local)
	fresh:=false
	if readErr==nil { if info,statErr:=os.Stat(local);statErr==nil { fresh=time.Since(info.ModTime())<24*time.Hour } }
	if readErr!=nil || !fresh {
		body,_,fetchErr:=get(ctx,rec.URL,4<<20)
		if fetchErr==nil {
			if err:=atomicCacheWrite(local,body);err!=nil{return RuleItem{},nil,err}
			data=body
		} else if readErr!=nil {
			return RuleItem{},nil,fetchErr
		}
	}
	rules,warnings,err:=ParseRules(data,rec.Format);if err!=nil{return RuleItem{},nil,err}
	sum:=sha256.Sum256(data);checksum:=hex.EncodeToString(sum[:])
	meta:=CountTypes(rules); metaJSON,_:=json.Marshal(meta)
	_ = models.DB.Model(&rec).Updates(map[string]any{"local_path":local,"rule_count":len(rules),"checksum":checksum,"metadata_json":string(metaJSON)}).Error
	sample:=rules;if len(sample)>30{sample=sample[:30]}
	return RuleItem{ExternalID:rec.ExternalID,SourceKey:rec.SourceKey,Name:rec.Name,Category:rec.Category,Platform:rec.Platform,Format:rec.Format,URL:rec.URL,LocalPath:local,RuleCount:len(rules),UpdatedAt:rec.RemoteUpdate,Checksum:checksum,Metadata:meta,Warnings:warnings,Sample:sample},rules,nil
}

func cachePath(source, platform, name, format string) string {
	clean:=regexp.MustCompile(`[^a-zA-Z0-9._-]+`).ReplaceAllString(name,"_")
	return filepath.Join(cacheRoot,source,platform,clean+"."+format)
}
func atomicCacheWrite(path string,data []byte) error { if err:=os.MkdirAll(filepath.Dir(path),0755);err!=nil{return err};tmp,err:=os.CreateTemp(filepath.Dir(path),".rule-*");if err!=nil{return err};name:=tmp.Name();defer os.Remove(name);if _,err=tmp.Write(data);err==nil{err=tmp.Close()}else{_ = tmp.Close()};if err!=nil{return err};return os.Rename(name,path) }
