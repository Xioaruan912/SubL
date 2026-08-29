package node

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	singbox "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

// UnlockService 一次解锁检测的目标服务
type UnlockService struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Group string `json:"group"` // ai / video / forum
	Check func(ctx context.Context, c *http.Client) (bool, string)
}

// UnlockCheckResult 单个服务解锁结果
type UnlockCheckResult struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	Group     string    `json:"group"`
	Ok        bool      `json:"ok"`
	Rtt       int       `json:"rtt"` // 毫秒，-1 表示失败
	Note      string    `json:"note"`
	Status    string    `json:"status"` // available / blocked / unknown
	Region    string    `json:"region,omitempty"`
	CheckedAt time.Time `json:"checkedAt"`
}

// UnlockResult 节点解锁测试总结果
type UnlockResult struct {
	NodeName string              `json:"nodeName"`
	Ok       bool                `json:"ok"`
	Results  []UnlockCheckResult `json:"results"`
	Error    string              `json:"error,omitempty"`
}

// 常见解锁检测目标（走节点访问，每个服务定制判定逻辑）
var defaultUnlockServices = []UnlockService{
	// AI
	{Key: "openai", Name: "OpenAI / ChatGPT", Group: "ai", Check: checkOpenAI},
	{Key: "claude", Name: "Claude", Group: "ai", Check: checkClaude},
	{Key: "google-gemini", Name: "Google Gemini", Group: "ai", Check: checkGemini},
	// 影视
	{Key: "netflix", Name: "Netflix", Group: "video", Check: checkNetflix},
	{Key: "youtube", Name: "YouTube", Group: "video", Check: checkYouTube},
	{Key: "disney", Name: "Disney+", Group: "video", Check: checkDisney},
	// 论坛 / 其它
	{Key: "google", Name: "Google", Group: "forum", Check: checkGoogle},
	{Key: "github", Name: "GitHub", Group: "forum", Check: checkGitHub},
	{Key: "telegram", Name: "Telegram", Group: "forum", Check: checkTelegram},
}

var unlockRegistry = struct {
	sync.RWMutex
	services map[string]UnlockService
}{services: make(map[string]UnlockService)}

func init() {
	for _, service := range defaultUnlockServices {
		RegisterUnlockService(service)
	}
}

// RegisterUnlockService makes detectors independently extensible without
// changing the runner. Re-registering a key safely replaces that detector.
func RegisterUnlockService(service UnlockService) {
	if service.Key == "" || service.Name == "" || service.Check == nil {
		return
	}
	unlockRegistry.Lock()
	defer unlockRegistry.Unlock()
	unlockRegistry.services[service.Key] = service
}

func ListUnlockServices() []UnlockService {
	unlockRegistry.RLock()
	defer unlockRegistry.RUnlock()
	result := make([]UnlockService, 0, len(unlockRegistry.services))
	for _, defaultService := range defaultUnlockServices {
		if service, ok := unlockRegistry.services[defaultService.Key]; ok {
			result = append(result, service)
		}
	}
	return result
}

// UnlockTestConfig 一次解锁测试的配置
type UnlockTestConfig struct {
	Link          string
	Timeout       time.Duration // 每个目标请求超时，默认 8s
	ServiceFilter string        // 非空时只测试指定服务 Key（如 google-gemini）
}

var (
	unlockMu sync.Mutex // 同一时刻只跑一个解锁测试，避免资源冲突
)

// UnlockTestBusy 尝试获取解锁测试互斥锁
func UnlockTestBusy() bool {
	return !unlockMu.TryLock()
}

// UnlockTestRelease 释放解锁测试互斥锁
func UnlockTestRelease() {
	unlockMu.Unlock()
}

// singBoxRegistry 缓存 include 提供的注册器（只初始化一次）
var (
	boxCtxOnce sync.Once
	boxCtx     context.Context
)

func getBoxContext() context.Context {
	boxCtxOnce.Do(func() {
		baseCtx := context.Background()
		boxCtx = singbox.Context(baseCtx, include.InboundRegistry(), include.OutboundRegistry(), include.EndpointRegistry())
	})
	return boxCtx
}

// RunUnlockTest 对节点链接做真实解锁检测（通过 sing-box 走该节点）。
// ctx 取消（客户端断开）时提前返回，及时释放锁。
func RunUnlockTest(ctx context.Context, cfg UnlockTestConfig) (*UnlockResult, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	// 解析节点链接
	outboundCfg, nodeName, err := buildOutboundConfig(cfg.Link)
	if err != nil {
		return nil, err
	}
	// 找一个空闲端口作为 socks 入站
	socksPort, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("获取空闲端口失败: %v", err)
	}

	// 构造 sing-box 配置
	config := buildSingboxConfig(socksPort, outboundCfg)
	options, err := parseOptions(config)
	if err != nil {
		return nil, fmt.Errorf("解析 sing-box 配置失败: %v", err)
	}

	instance, err := singbox.New(singbox.Options{
		Context: getBoxContext(),
		Options: *options,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 sing-box 实例失败: %v", err)
	}
	if err := instance.Start(); err != nil {
		instance.Close()
		return nil, fmt.Errorf("启动 sing-box 失败: %v", err)
	}
	defer instance.Close()
	// 通过 socks 代理发起请求
	proxyURL, _ := url.Parse("socks5://127.0.0.1:" + strconv.Itoa(socksPort))
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		DialContext: (&net.Dialer{
			Timeout: timeout,
		}).DialContext,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   4,
		ForceAttemptHTTP2:     true,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	// 是否过滤到指定服务（如仅测 Gemini）
	targets := ListUnlockServices()
	if cfg.ServiceFilter != "" {
		filtered := make([]UnlockService, 0, 1)
		for _, s := range targets {
			if s.Key == cfg.ServiceFilter {
				filtered = append(filtered, s)
			}
		}
		targets = filtered
	}

	// 服务检测互不依赖。受控并发能把最坏耗时从 N×timeout 降到约 ceil(N/4)×timeout，
	// 同时避免一次性向较弱节点发起过多连接。
	result := &UnlockResult{NodeName: nodeName, Results: make([]UnlockCheckResult, len(targets))}
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i, svc := range targets {
		wg.Add(1)
		go func(i int, svc UnlockService) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			checkCtx, checkCancel := context.WithTimeout(ctx, timeout)
			defer checkCancel()
			start := time.Now()
			ok, note := svc.Check(checkCtx, client)
			rtt := max(1, int(time.Since(start).Milliseconds()))
			status := "blocked"
			if ok {
				status = "available"
			} else if note == "连接失败" || note == "异常响应" {
				status = "unknown"
			}
			result.Results[i] = UnlockCheckResult{Key: svc.Key, Name: svc.Name, Group: svc.Group, Ok: ok, Rtt: rtt, Note: note, Status: status, CheckedAt: time.Now()}
		}(i, svc)
	}
	waitCh := make(chan struct{})
	go func() { wg.Wait(); close(waitCh) }()
	select {
	case <-waitCh:
	case <-ctx.Done():
		<-waitCh
	}
	for _, check := range result.Results {
		if check.Ok {
			result.Ok = true
		}
	}
	return result, nil
}

// parseOptions 将 JSON 配置解析为 option.Options（带注册上下文）
func parseOptions(configJSON string) (*option.Options, error) {
	ctx := getBoxContext()
	opts := &option.Options{}
	if err := opts.UnmarshalJSONContext(ctx, []byte(configJSON)); err != nil {
		return nil, err
	}
	return opts, nil
}

// freePort 返回一个当前空闲的 TCP 端口
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

// buildSingboxConfig 构造 sing-box JSON 配置
func buildSingboxConfig(socksPort int, outboundJSON string) string {
	return fmt.Sprintf(`{
  "log": {"disabled": true},
  "inbounds": [
    {
      "type": "socks",
      "tag": "socks-in",
      "listen": "127.0.0.1",
      "listen_port": %d
    }
  ],
  "outbounds": [
    %s,
    {
      "type": "direct",
      "tag": "direct"
    }
  ],
  "route": {
    "final": "proxy"
  }
}`, socksPort, outboundJSON)
}

// buildOutboundConfig 根据节点链接构造 sing-box outbound 配置（JSON）。
func buildOutboundConfig(link string) (string, string, error) {
	link = strings.TrimSpace(link)
	u, err := url.Parse(link)
	if err != nil {
		return "", "", fmt.Errorf("解析节点链接失败: %v", err)
	}
	scheme := u.Scheme
	name := u.Fragment
	if name == "" {
		name = link
	}

	switch scheme {
	case "ss":
		ss, err := DecodeSSURL(link)
		if err != nil {
			return "", "", err
		}
		return fmt.Sprintf(`{
  "type": "shadowsocks",
  "tag": "proxy",
  "server": %q,
  "server_port": %d,
  "method": %q,
  "password": %q
}`, ss.Server, ss.Port, ss.Param.Cipher, ss.Param.Password), ss.Name, nil
	case "ssr":
		// sing-box 对 shadowsocksr 支持有限，直接报错提示
		return "", "", fmt.Errorf("不支持 ssr 协议解锁测试")
	case "trojan":
		t, err := DecodeTrojanURL(link)
		if err != nil {
			return "", "", err
		}
		tls := ""
		if t.Query.Sni != "" || t.Query.AllowInsecure != 0 {
			insecure := "false"
			if t.Query.AllowInsecure != 0 {
				insecure = "true"
			}
			tls = fmt.Sprintf(`, "tls": {"enabled": true, "server_name": %q, "insecure": %s}`, t.Query.Sni, insecure)
		}
		return fmt.Sprintf(`{
  "type": "trojan",
  "tag": "proxy",
  "server": %q,
  "server_port": %d,
  "password": %q%s
}`, t.Hostname, t.Port, t.Password, tls), t.Name, nil
	case "vmess":
		vm, err := DecodeVMESSURL(link)
		if err != nil {
			return "", "", err
		}
		port := 0
		if p, ok := vm.Port.(float64); ok {
			port = int(p)
		} else if p, ok := vm.Port.(string); ok {
			port, _ = strconv.Atoi(p)
		}
		tls := ""
		if vm.Tls == "tls" {
			tls = fmt.Sprintf(`, "tls": {"enabled": true, "server_name": %q, "insecure": true}`, vm.Host)
		}
		return fmt.Sprintf(`{
  "type": "vmess",
  "tag": "proxy",
  "server": %q,
  "server_port": %d,
  "uuid": %q,
  "security": "auto",
  "alter_id": 0%s
}`, vm.Add, port, vm.Id, tls), vm.Ps, nil
	case "vless":
		vl, err := DecodeVLESSURL(link)
		if err != nil {
			return "", "", err
		}
		cfg := buildVLESSOutbound(vl)
		return cfg, vl.Name, nil
	case "hy", "hysteria":
		return "", "", fmt.Errorf("不支持 hysteria1 协议解锁测试")
	case "hy2", "hysteria2":
		hy2, err := DecodeHY2URL(link)
		if err != nil {
			return "", "", err
		}
		tls := fmt.Sprintf(`, "tls": {"enabled": true, "server_name": %q, "insecure": true}`, hy2.Sni)
		return fmt.Sprintf(`{
  "type": "hysteria2",
  "tag": "proxy",
  "server": %q,
  "server_port": %d,
  "password": %q%s
}`, hy2.Host, hy2.Port, hy2.Password, tls), hy2.Name, nil
	case "tuic":
		tuic, err := DecodeTuicURL(link)
		if err != nil {
			return "", "", err
		}
		alpn, _ := json.Marshal(tuic.Alpn)
		// 生成随机 UUID（tuic 需要 uuid）
		uuid := randUUID()
		return fmt.Sprintf(`{
  "type": "tuic",
  "tag": "proxy",
  "server": %q,
  "server_port": %d,
  "uuid": %q,
  "password": %q,
  "congestion_control": %q,
  "alpn": %s,
  "tls": {"enabled": true, "server_name": %q, "insecure": true}
}`, tuic.Host, tuic.Port, uuid, tuic.Password, tuic.Congestion_control, string(alpn), tuic.Sni), tuic.Name, nil
	default:
		return "", "", fmt.Errorf("不支持的协议: %s", scheme)
	}
}

// buildVLESSOutbound 构造 vless outbound（含 reality / tls / transport 支持）
func buildVLESSOutbound(vl VLESS) string {
	base := fmt.Sprintf(`{
  "type": "vless",
  "tag": "proxy",
  "server": %q,
  "server_port": %d,
  "uuid": %q`,
		vl.Server, vl.Port, vl.Uuid)

	var parts []string

	// flow
	if vl.Query.Flow != "" {
		parts = append(parts, fmt.Sprintf(`"flow": %q`, vl.Query.Flow))
	}
	// TLS / Reality
	switch vl.Query.Security {
	case "reality":
		parts = append(parts, fmt.Sprintf(`"tls": {
  "enabled": true,
  "server_name": %q,
  "utls": {"enabled": true, "fingerprint": %q},
  "reality": {"enabled": true, "public_key": %q, "short_id": %q}
}`, vl.Query.Sni, fpOrDefault(vl.Query.Fp), vl.Query.Pbk, vl.Query.Sid))
	case "tls":
		parts = append(parts, fmt.Sprintf(`"tls": {"enabled": true, "server_name": %q, "insecure": true}`, vl.Query.Sni))
	}

	// transport (ws / grpc)
	if vl.Query.Type == "ws" {
		parts = append(parts, fmt.Sprintf(`"transport": {
  "type": "ws",
  "path": %q,
  "headers": {"Host": %q}
}`, vl.Query.Path, hostOrDefault(vl.Query.Host, vl.Query.Sni)))
	} else if vl.Query.Type == "grpc" {
		parts = append(parts, fmt.Sprintf(`"transport": {
  "type": "grpc",
  "service_name": %q
}`, vl.Query.ServiceName))
	} else if vl.Query.Type == "http" {
		parts = append(parts, fmt.Sprintf(`"transport": {
  "type": "http",
  "host": [%q],
  "path": %q
}`, vl.Query.Host, vl.Query.Path))
	}

	if len(parts) > 0 {
		base += ",\n" + strings.Join(parts, ",\n")
	}
	return base + "\n}"
}

func fpOrDefault(fp string) string {
	if fp == "" {
		return "chrome"
	}
	return fp
}

func hostOrDefault(host, sni string) string {
	if host == "" {
		return sni
	}
	return host
}

// randUUID 生成一个随机 UUID（tuic 使用）
func randUUID() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// logUnlockDebug 便于调试
var _ = log.Println
