package node

import (
	"net"
	"sync"
	"time"
)

// PingTarget 表示一个待测延迟的目标
type PingTarget struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
	Rtt  int    `json:"rtt"` // 毫秒，失败为 -1
}

// PingResult 单次测速结果
type PingResult struct {
	Target PingTarget `json:"target"`
	Ok     bool       `json:"ok"`
}

// TCPPing 对指定地址做 TCP 连接测速（DialTimeout），返回毫秒级延迟；失败返回 -1。
func TCPPing(addr string, timeout time.Duration) int {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return -1
	}
	conn.Close()
	ms := time.Since(start).Milliseconds()
	if ms < 1 {
		ms = 1
	}
	return int(ms)
}

// MultiTCPPing 并发对多个地址测速，返回与输入顺序一致的结果切片。
func MultiTCPPing(addrs []PingTarget, timeout time.Duration) []PingResult {
	results := make([]PingResult, len(addrs))
	var wg sync.WaitGroup
	for i, t := range addrs {
		wg.Add(1)
		go func(i int, t PingTarget) {
			defer wg.Done()
			rtt := TCPPing(t.Addr, timeout)
			results[i] = PingResult{Target: t, Ok: rtt >= 0}
			if rtt >= 0 {
				results[i].Target.Rtt = rtt
			} else {
				results[i].Target.Rtt = -1
			}
		}(i, t)
	}
	wg.Wait()
	return results
}