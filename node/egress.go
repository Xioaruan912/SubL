package node

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	singbox "github.com/sagernet/sing-box"
)

type EgressTarget struct {
	Key        string `json:"key"`
	Name       string `json:"name"`
	Domain     string `json:"domain"`
	Group      string `json:"group"`
	Path       string `json:"-"`
	IPOptional bool   `json:"-"`
}

type EgressCheckResult struct {
	EgressTarget
	Status      string    `json:"status"`
	IP          string    `json:"ip"`
	CountryCode string    `json:"countryCode"`
	Rtt         int       `json:"rtt"`
	Note        string    `json:"note"`
	CheckedAt   time.Time `json:"checkedAt"`
}

type EgressResult struct {
	NodeName string              `json:"nodeName"`
	Results  []EgressCheckResult `json:"results"`
}

var egressTargets = []EgressTarget{
	{Key: "cloudflare", Name: "Cloudflare", Domain: "www.cloudflare.com", Group: "network"},
	{Key: "chatgpt", Name: "ChatGPT", Domain: "chatgpt.com", Group: "ai"},
	{Key: "openai", Name: "OpenAI", Domain: "openai.com", Group: "ai"},
	{Key: "gemini", Name: "Gemini", Domain: "gemini.google.com", Group: "ai", Path: "/", IPOptional: true},
	{Key: "claude", Name: "Claude", Domain: "claude.ai", Group: "ai"},
	{Key: "anthropic", Name: "Anthropic", Domain: "anthropic.com", Group: "ai"},
	{Key: "perplexity", Name: "Perplexity", Domain: "www.perplexity.ai", Group: "ai"},
	{Key: "discord", Name: "Discord", Domain: "gateway.discord.gg", Group: "social"},
	{Key: "x", Name: "X", Domain: "x.com", Group: "social"},
	{Key: "medium", Name: "Medium", Domain: "medium.com", Group: "content"},
	{Key: "coinbase", Name: "Coinbase", Domain: "coinbase.com", Group: "finance"},
	{Key: "notion", Name: "Notion", Domain: "notion.so", Group: "tools"},
	{Key: "cdnjs", Name: "Cloudflare CDN", Domain: "cdnjs.cloudflare.com", Group: "developer"},
	{Key: "npm", Name: "npm Registry", Domain: "registry.npmjs.org", Group: "developer"},
	{Key: "gitlab", Name: "GitLab", Domain: "gitlab.com", Group: "developer"},
	{Key: "crunchyroll", Name: "Crunchyroll", Domain: "crunchyroll.com", Group: "media"},
}

func traceThroughProxy(ctx context.Context, client *http.Client, target EgressTarget) EgressCheckResult {
	result := EgressCheckResult{EgressTarget: target, Status: "unknown", Rtt: -1, CheckedAt: time.Now()}
	path := target.Path
	if path == "" {
		path = "/cdn-cgi/trace"
	}
	request, _ := http.NewRequestWithContext(ctx, "GET", "https://"+target.Domain+path, nil)
	request.Header.Set("User-Agent", unlockUA)
	started := time.Now()
	response, err := client.Do(request)
	if err != nil {
		result.Note = "连接失败"
		return result
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	result.Rtt = max(1, int(time.Since(started).Milliseconds()))
	if err != nil {
		result.Note = "读取失败"
		return result
	}
	entries := map[string]string{}
	for _, line := range strings.Split(string(body), "\n") {
		if index := strings.Index(line, "="); index > 0 {
			entries[line[:index]] = strings.TrimSpace(line[index+1:])
		}
	}
	if entries["ip"] == "" {
		if target.IPOptional && response.StatusCode >= 200 && response.StatusCode < 400 {
			result.Status = "reachable"
			result.Note = "站点可达，目标未提供出口 IP"
			return result
		}
		result.Note = fmt.Sprintf("目标未返回出口 IP (HTTP %d)", response.StatusCode)
		return result
	}
	result.Status = "available"
	result.IP = entries["ip"]
	result.CountryCode = strings.ToUpper(entries["loc"])
	return result
}

// RunEgressTest starts a temporary local SOCKS inbound and routes every trace
// request through exactly one selected subscription node.
func RunEgressTest(ctx context.Context, link string, timeout time.Duration) (*EgressResult, error) {
	return runEgressTest(ctx, link, timeout, egressTargets)
}

// RunEgressTestKeys runs only the requested targets. It is used by the
// template-aware split verifier to avoid testing unrelated sites.
func RunEgressTestKeys(ctx context.Context, link string, timeout time.Duration, keys []string) (*EgressResult, error) {
	wanted := make(map[string]bool, len(keys))
	for _, key := range keys {
		wanted[key] = true
	}
	targets := make([]EgressTarget, 0, len(keys))
	for _, target := range egressTargets {
		if wanted[target.Key] {
			targets = append(targets, target)
		}
	}
	return runEgressTest(ctx, link, timeout, targets)
}

func runEgressTest(ctx context.Context, link string, timeout time.Duration, targets []EgressTarget) (*EgressResult, error) {
	if timeout <= 0 {
		timeout = 7 * time.Second
	}
	outbound, nodeName, err := buildOutboundConfig(link)
	if err != nil {
		return nil, err
	}
	port, err := freePort()
	if err != nil {
		return nil, err
	}
	options, err := parseOptions(buildSingboxConfig(port, outbound))
	if err != nil {
		return nil, err
	}
	instance, err := singbox.New(singbox.Options{Context: getBoxContext(), Options: *options})
	if err != nil {
		return nil, err
	}
	if err := instance.Start(); err != nil {
		instance.Close()
		return nil, err
	}
	defer instance.Close()
	proxyURL, _ := url.Parse("socks5://127.0.0.1:" + strconv.Itoa(port))
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), DialContext: (&net.Dialer{Timeout: timeout}).DialContext, TLSHandshakeTimeout: timeout, ResponseHeaderTimeout: timeout, MaxIdleConns: 24, MaxIdleConnsPerHost: 4, ForceAttemptHTTP2: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: timeout}
	result := &EgressResult{NodeName: nodeName, Results: make([]EgressCheckResult, len(targets))}
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for index, target := range targets {
		wg.Add(1)
		go func(index int, target EgressTarget) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			checkCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			result.Results[index] = traceThroughProxy(checkCtx, client, target)
		}(index, target)
	}
	wg.Wait()
	return result, nil
}
