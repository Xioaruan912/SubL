package node

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	singbox "github.com/sagernet/sing-box"
	"golang.org/x/net/proxy"
)

// ChinaPingTarget 单条中国目标结果
type ChinaPingTarget struct {
	Province string  `json:"province"`
	City     string  `json:"city"`
	ISP      string  `json:"isp"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	IP       string  `json:"ip"`
	Port     int     `json:"port"`
	Rtt      int     `json:"rtt"` // 毫秒，-1 失败
}

// ChinaPingResult 中国延迟测试总结果
type ChinaPingResult struct {
	NodeName string            `json:"nodeName"`
	Targets  []ChinaPingTarget `json:"targets"`
	ZStatic  []ZStaticResult   `json:"zstatic"`
	Error    string            `json:"error,omitempty"`
}

// ZStaticResult zstaticcdn 延迟结果
type ZStaticResult struct {
	Port int `json:"port"`
	Rtt  int `json:"rtt"`
}

// RunChinaPing 走节点对中国各地运营商 + zstaticcdn 做 TCP 延迟测试。
func RunChinaPing(ctx context.Context, cfg UnlockTestConfig, provinces, isps []string, zstaticPorts []int) (*ChinaPingResult, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	// 解析节点并启动 sing-box socks
	outboundCfg, nodeName, err := buildOutboundConfig(cfg.Link)
	if err != nil {
		return nil, err
	}
	socksPort, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("获取空闲端口失败: %v", err)
	}
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

	// socks5 代理 dialer
	socksAddr := "127.0.0.1:" + strconv.Itoa(socksPort)
	baseDialer := &net.Dialer{Timeout: timeout}
	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, baseDialer)
	if err != nil {
		return nil, fmt.Errorf("创建 socks dialer 失败: %v", err)
	}
	contextDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("socks dialer 不支持上下文取消")
	}

	// 过滤目标
	targets := compactChinaTargets(FilterChinaTargets(provinces, isps))

	result := &ChinaPingResult{NodeName: nodeName, Targets: make([]ChinaPingTarget, 0, len(targets))}

	// 每省/运营商取一个稳定代表点，全局并发并做双样本，兼顾速度与抗抖动。
	sem := make(chan struct{}, 24)
	var wg sync.WaitGroup
	results := make([]ChinaPingTarget, len(targets))
	for i, t := range targets {
		wg.Add(1)
		go func(i int, t ChinaTarget) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			rtt := tcpPingStableVia(ctx, contextDialer, net.JoinHostPort(t.IP, strconv.Itoa(t.Port)), timeout)
			results[i] = ChinaPingTarget{
				Province: t.Province, City: t.City, ISP: t.ISP,
				Lat: t.Lat, Lng: t.Lng, IP: t.IP, Port: t.Port, Rtt: rtt,
			}
		}(i, t)
	}
	// ctx 取消（停止/右上角停止）时立即返回释放锁
	waitCh := make(chan struct{})
	go func() { wg.Wait(); close(waitCh) }()
	select {
	case <-waitCh:
	case <-ctx.Done():
		return result, nil
	}
	result.Targets = results

	// zstaticcdn 延迟
	if len(zstaticPorts) == 0 {
		zstaticPorts = []int{443}
	}
	for _, p := range zstaticPorts {
		rtt := tcpPingStableVia(ctx, contextDialer, net.JoinHostPort(zstaticcdnHost, strconv.Itoa(p)), timeout)
		result.ZStatic = append(result.ZStatic, ZStaticResult{Port: p, Rtt: rtt})
	}

	return result, nil
}

// ChinaPingStreamCallback 每测完一个省份回调
type ChinaPingStreamCallback func(province string, targets []ChinaPingTarget)

// RunChinaPingStream 走节点对中国各地 TCP 延迟测试，按省分组、组内并发，
// 每测完一个省立即调用 onProvince 回调（用于 SSE 实时推送）。
// ctx 取消（客户端断开）时提前返回，及时释放锁。
func RunChinaPingStream(ctx context.Context, cfg UnlockTestConfig, provinces, isps []string, zstaticPorts []int, onProvince ChinaPingStreamCallback) (*ChinaPingResult, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	// 解析节点并启动 sing-box socks
	outboundCfg, nodeName, err := buildOutboundConfig(cfg.Link)
	if err != nil {
		return nil, err
	}
	socksPort, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("获取空闲端口失败: %v", err)
	}
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

	socksAddr := "127.0.0.1:" + strconv.Itoa(socksPort)
	baseDialer := &net.Dialer{Timeout: timeout}
	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, baseDialer)
	if err != nil {
		return nil, fmt.Errorf("创建 socks dialer 失败: %v", err)
	}
	contextDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("socks dialer 不支持上下文取消")
	}

	targets := compactChinaTargets(FilterChinaTargets(provinces, isps))
	result := &ChinaPingResult{NodeName: nodeName, Targets: make([]ChinaPingTarget, 0, len(targets))}

	// 按省分组
	provMap := map[string][]ChinaTarget{}
	var provOrder []string
	for _, t := range targets {
		if _, ok := provMap[t.Province]; !ok {
			provOrder = append(provOrder, t.Province)
		}
		provMap[t.Province] = append(provMap[t.Province], t)
	}

	// 所有省份同时进入全局工作池，完成一个省就推送一个省，避免旧实现逐省串行等待。
	type provinceResult struct {
		name    string
		targets []ChinaPingTarget
	}
	done := make(chan provinceResult, len(provOrder))
	sem := make(chan struct{}, 24)
	for _, prov := range provOrder {
		go func(prov string, provTargets []ChinaTarget) {
			results := make([]ChinaPingTarget, len(provTargets))
			var wg sync.WaitGroup
			for i, t := range provTargets {
				wg.Add(1)
				go func(i int, t ChinaTarget) {
					defer wg.Done()
					select {
					case sem <- struct{}{}:
					case <-ctx.Done():
						return
					}
					defer func() { <-sem }()
					rtt := tcpPingStableVia(ctx, contextDialer, net.JoinHostPort(t.IP, strconv.Itoa(t.Port)), timeout)
					results[i] = ChinaPingTarget{Province: t.Province, City: t.City, ISP: t.ISP, Lat: t.Lat, Lng: t.Lng, IP: t.IP, Port: t.Port, Rtt: rtt}
				}(i, t)
			}
			wg.Wait()
			select {
			case done <- provinceResult{prov, results}:
			case <-ctx.Done():
			}
		}(prov, provMap[prov])
	}
	for range provOrder {
		select {
		case completed := <-done:
			if onProvince != nil {
				onProvince(completed.name, completed.targets)
			}
			result.Targets = append(result.Targets, completed.targets...)
		case <-ctx.Done():
			return result, nil
		}
	}

	// zstaticcdn 延迟
	if len(zstaticPorts) == 0 {
		zstaticPorts = []int{443}
	}
	for _, p := range zstaticPorts {
		rtt := tcpPingStableVia(ctx, contextDialer, net.JoinHostPort(zstaticcdnHost, strconv.Itoa(p)), timeout)
		result.ZStatic = append(result.ZStatic, ZStaticResult{Port: p, Rtt: rtt})
	}

	return result, nil
}

// tcpPingVia 使用真正的 DialContext，停止测试时不会遗留后台拨号 goroutine。
func tcpPingVia(ctx context.Context, dialer proxy.ContextDialer, addr string, timeout time.Duration) int {
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	conn, err := dialer.DialContext(dialCtx, "tcp", addr)
	if err != nil {
		return -1
	}
	conn.Close()
	return max(1, int(time.Since(start).Milliseconds()))
}

// tcpPingStableVia 取两次成功握手的平均值；一次偶发失败不会把可用节点误判为不可达。
func tcpPingStableVia(ctx context.Context, dialer proxy.ContextDialer, addr string, timeout time.Duration) int {
	values := make([]int, 0, 2)
	for i := 0; i < 2; i++ {
		if rtt := tcpPingVia(ctx, dialer, addr, timeout); rtt >= 0 {
			values = append(values, rtt)
		}
		if ctx.Err() != nil {
			break
		}
	}
	if len(values) == 0 {
		return -1
	}
	if len(values) == 1 {
		return values[0]
	}
	return (values[0] + values[1]) / 2
}

// compactChinaTargets 保留每省每运营商一个代表端点，避免同一网络重复探测大量城市。
func compactChinaTargets(targets []ChinaTarget) []ChinaTarget {
	seen := make(map[string]struct{}, len(targets))
	result := make([]ChinaTarget, 0, len(targets))
	for _, target := range targets {
		key := target.Province + "|" + target.ISP
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, target)
	}
	return result
}
