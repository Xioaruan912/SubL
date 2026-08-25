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
	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("创建 socks dialer 失败: %v", err)
	}

	// 过滤目标
	targets := FilterChinaTargets(provinces, isps)

	result := &ChinaPingResult{NodeName: nodeName, Targets: make([]ChinaPingTarget, 0, len(targets))}

	// 并发 TCP ping 中国目标（限并发 12）
	sem := make(chan struct{}, 12)
	var wg sync.WaitGroup
	results := make([]ChinaPingTarget, len(targets))
	for i, t := range targets {
		wg.Add(1)
		go func(i int, t ChinaTarget) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			rtt := tcpPingVia(dialer, net.JoinHostPort(t.IP, strconv.Itoa(t.Port)), timeout)
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
		rtt := tcpPingVia(dialer, net.JoinHostPort(zstaticcdnHost, strconv.Itoa(p)), timeout)
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
	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("创建 socks dialer 失败: %v", err)
	}

	targets := FilterChinaTargets(provinces, isps)
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

	// 逐省处理（每省内并发，限并发 12）
	sem := make(chan struct{}, 12)
	for _, prov := range provOrder {
		// 客户端断开则提前返回，释放锁
		select {
		case <-ctx.Done():
			return result, nil
		default:
		}
		provTargets := provMap[prov]
		results := make([]ChinaPingTarget, len(provTargets))
		var wg sync.WaitGroup
		for i, t := range provTargets {
			wg.Add(1)
			go func(i int, t ChinaTarget) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				rtt := tcpPingVia(dialer, net.JoinHostPort(t.IP, strconv.Itoa(t.Port)), timeout)
				results[i] = ChinaPingTarget{
					Province: t.Province, City: t.City, ISP: t.ISP,
					Lat: t.Lat, Lng: t.Lng, IP: t.IP, Port: t.Port, Rtt: rtt,
				}
			}(i, t)
		}
		// 等待本省全部目标测完；ctx 取消（停止/右上角停止）时立即返回释放锁
		waitCh := make(chan struct{})
		go func() { wg.Wait(); close(waitCh) }()
		select {
		case <-waitCh:
		case <-ctx.Done():
			return result, nil
		}
		// 立即回调推送该省结果
		if onProvince != nil {
			onProvince(prov, results)
		}
		result.Targets = append(result.Targets, results...)
	}

	// zstaticcdn 延迟
	if len(zstaticPorts) == 0 {
		zstaticPorts = []int{443}
	}
	for _, p := range zstaticPorts {
		rtt := tcpPingVia(dialer, net.JoinHostPort(zstaticcdnHost, strconv.Itoa(p)), timeout)
		result.ZStatic = append(result.ZStatic, ZStaticResult{Port: p, Rtt: rtt})
	}

	return result, nil
}

// tcpPingVia 通过 socks dialer 做 TCP connect 计时，返回毫秒；失败/超时返回 -1。
func tcpPingVia(dialer proxy.Dialer, addr string, timeout time.Duration) int {
	done := make(chan int, 1)
	go func() {
		start := time.Now()
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			done <- -1
			return
		}
		conn.Close()
		ms := time.Since(start).Milliseconds()
		if ms < 1 {
			ms = 1
		}
		done <- int(ms)
	}()
	select {
	case rtt := <-done:
		return rtt
	case <-time.After(timeout):
		return -1 // 超时视为不可达
	}
}