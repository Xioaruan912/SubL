package client

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ppeelink/models"
)

// ClientPlatform 单平台定义
type ClientPlatform struct {
	Key   string // win-x64 / win-arm64 / mac-x64 / mac-arm64
	Label string // Windows x64 …
	Match string // 文件名包含匹配
}

// ClientSource 客户端源定义
type ClientSource struct {
	Name      string
	Owner     string
	Repo      string
	Icon      string // emoji 图标
	Platforms []ClientPlatform
}

// Sources 收录的客户端（可扩展）
var Sources = []ClientSource{
	{
		Name: "clash-verge-rev", Owner: "clash-verge-rev", Repo: "clash-verge-rev", Icon: "🛡️",
		Platforms: []ClientPlatform{
			{Key: "win-x64", Label: "Windows x64", Match: "x64-setup.exe"},
			{Key: "win-arm64", Label: "Windows arm64", Match: "arm64-setup.exe"},
			{Key: "mac-arm64", Label: "macOS Apple Silicon", Match: "aarch64.dmg"},
			{Key: "mac-x64", Label: "macOS Intel", Match: "x64.dmg"},
		},
	},
	{
		Name: "v2rayN", Owner: "2dust", Repo: "v2rayN", Icon: "🚀",
		Platforms: []ClientPlatform{
			{Key: "win-x64", Label: "Windows x64", Match: "windows-64.zip"},
			{Key: "win-arm64", Label: "Windows arm64", Match: "windows-arm64.zip"},
			{Key: "mac-x64", Label: "macOS Intel", Match: "macos-64.zip"},
			{Key: "mac-arm64", Label: "macOS Apple Silicon", Match: "macos-arm64.zip"},
		},
	},
	{
		Name: "FlClash", Owner: "chen08209", Repo: "FlClash", Icon: "💥",
		Platforms: []ClientPlatform{
			{Key: "win-x64", Label: "Windows x64", Match: "windows-amd64-setup.exe"},
			{Key: "win-arm64", Label: "Windows arm64", Match: "windows-arm64-setup.exe"},
			{Key: "mac-x64", Label: "macOS Intel", Match: "macos-amd64.dmg"},
			{Key: "mac-arm64", Label: "macOS Apple Silicon", Match: "macos-arm64.dmg"},
		},
	},
}

// 下载目录（运行时相对 WorkingDirectory）
var downloadDir = "downloads"

var (
	mu          sync.Mutex
	running     bool
	lastChecked time.Time
)

// GitHub release 响应结构
type ghRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		Size               int64  `json:"size"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// apiURL 组装 GitHub API 地址
func apiURL(src ClientSource) string {
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", src.Owner, src.Repo)
}

// fetchLatest 请求 GitHub API，返回最新 release
func fetchLatest(src ClientSource) (*ghRelease, error) {
	req, err := http.NewRequest("GET", apiURL(src), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ppeelink-downloader")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API %d", resp.StatusCode)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// findAsset 按平台匹配规则找资产
func findAsset(rel *ghRelease, plat ClientPlatform) (string, int64, string, bool) {
	for i := range rel.Assets {
		a := &rel.Assets[i]
		if strings.Contains(a.Name, plat.Match) && !strings.HasSuffix(a.Name, ".sig") {
			return a.Name, a.Size, a.BrowserDownloadURL, true
		}
	}
	return "", 0, "", false
}

// checkPlatform 检查并下载单个平台
func checkPlatform(src ClientSource, plat ClientPlatform, rel *ghRelease) {
	rec := &models.ClientVersion{Client: src.Name, Platform: plat.Key}
	// 本地已有且版本相同 → 跳过
	existing, err := rec.ByClientPlatform(src.Name, plat.Key)
	if err == nil && existing.Status == "ready" && existing.Version == rel.TagName {
		if _, statErr := os.Stat(filepath.Join(downloadDir, existing.FileName)); statErr == nil {
			return
		}
	}

	assetName, assetSize, assetURL, ok := findAsset(rel, plat)
	if !ok {
		rec.Save()
		rec.SetStatus("failed", "未找到匹配资产: "+plat.Match)
		return
	}

	_ = rec.Save()
	_ = rec.SetStatus("downloading", "")

	if err := download(src.Name, plat.Key, assetURL, assetSize); err != nil {
		_ = rec.SetStatus("failed", err.Error())
		return
	}

	// 落库
	fileName := fmt.Sprintf("%s_%s.%s", src.Name, plat.Key, extOf(assetName))
	_ = os.Rename(filepath.Join(downloadDir, tmpName(src.Name, plat.Key)), filepath.Join(downloadDir, fileName))
	rec2 := &models.ClientVersion{
		Client: src.Name, Platform: plat.Key, Version: rel.TagName,
		FileName: fileName, Size: assetSize, Status: "ready", UpdatedAt: time.Now().Unix(),
	}
	_ = rec2.Save()
	// 清理同客户端同平台的其它旧文件
	cleanOld(src.Name, plat.Key, fileName)
	log.Printf("[client] %s %s 更新到 %s (%dMB)", src.Name, plat.Key, rel.TagName, assetSize/1024/1024)
}

// download 下载到临时文件（支持 302 重定向，流式写入）
func download(name, platform, url string, size int64) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("下载 HTTP %d", resp.StatusCode)
	}
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(downloadDir, tmpName(name, platform))
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("写入失败: %v", err)
	}
	return nil
}

func tmpName(name, platform string) string {
	return fmt.Sprintf("%s_%s.tmp", name, platform)
}

func extOf(filename string) string {
	if i := strings.LastIndex(filename, "."); i >= 0 {
		return filename[i+1:]
	}
	return "bin"
}

// cleanOld 删除同客户端同平台前缀的其它文件
func cleanOld(name, platform, keep string) {
	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		return
	}
	prefix := fmt.Sprintf("%s_%s.", name, platform)
	for _, e := range entries {
		if e.Name() != keep && strings.HasPrefix(e.Name(), prefix) {
			_ = os.Remove(filepath.Join(downloadDir, e.Name()))
		}
	}
}

// CheckAll 检查全部客户端更新（手动触发或定时调用）
func CheckAll() error {
	mu.Lock()
	if running {
		mu.Unlock()
		return fmt.Errorf("检查进行中")
	}
	running = true
	mu.Unlock()
	defer func() { mu.Lock(); running = false; lastChecked = time.Now(); mu.Unlock() }()

	for _, src := range Sources {
		rel, err := fetchLatest(src)
		if err != nil {
			log.Printf("[client] %s 获取最新版本失败: %v", src.Name, err)
			continue
		}
		for _, plat := range src.Platforms {
			checkPlatform(src, plat, rel)
		}
	}
	return nil
}

// Start 启动定时检查：启动立即执行 + 每 24h
func Start() {
	go func() {
		// 延迟 5s 等数据库就绪
		time.Sleep(5 * time.Second)
		_ = CheckAll()
		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			_ = CheckAll()
		}
	}()
}

// LastChecked 上次检查时间
func LastChecked() time.Time { return lastChecked }

// StatusList 返回前端列表数据
func StatusList() []map[string]any {
	var items []map[string]any
	for _, src := range Sources {
		row := map[string]any{
			"name": src.Name, "icon": src.Icon,
			"owner": src.Owner, "repo": src.Repo,
			"platforms": []map[string]any{},
		}
		plats := []map[string]any{}
		for _, plat := range src.Platforms {
			rec, _ := (&models.ClientVersion{}).ByClientPlatform(src.Name, plat.Key)
			p := map[string]any{
				"key": plat.Key, "label": plat.Label,
				"version": "", "size": 0, "status": "idle", "errMsg": "", "updatedAt": 0,
			}
			if rec != nil {
				p["version"] = rec.Version
				p["size"] = rec.Size
				p["status"] = rec.Status
				p["errMsg"] = rec.ErrMsg
				p["updatedAt"] = rec.UpdatedAt
			}
			plats = append(plats, p)
		}
		row["platforms"] = plats
		items = append(items, row)
	}
	return items
}