package utils

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var blockedIPNets = mustParseCIDRs([]string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
	"ff00::/8",
})

func mustParseCIDRs(items []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(items))
	for _, item := range items {
		if _, n, err := net.ParseCIDR(item); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// allowPrivateFetch reports whether the operator explicitly opted out of the
// private/loopback egress protection (for self-hosted subscription sources).
func allowPrivateFetch() bool {
	v := strings.TrimSpace(os.Getenv("SUBLINKX_ALLOW_PRIVATE_FETCH"))
	return v == "1" || strings.EqualFold(v, "true")
}

// IsBlockedIP reports whether the address is private, loopback, link-local,
// multicast or otherwise reserved and therefore unsafe for server-side fetches.
func IsBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if allowPrivateFetch() {
		return false
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, n := range blockedIPNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ValidateOutboundURL rejects non-http(s) schemes and literal private targets.
func ValidateOutboundURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("仅允许 http/https 出站请求")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("出站请求目标为空")
	}
	if ip := net.ParseIP(host); ip != nil && IsBlockedIP(ip) {
		return fmt.Errorf("目标地址 %s 属于受保护的私有/保留网段", ip)
	}
	return nil
}

// SafeHTTPClient returns an http.Client that validates outbound targets at dial
// time (blocking private/loopback/metadata addresses) with bounded redirects
// and timeouts. Pass timeout <= 0 for the 15s default.
func SafeHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if allowPrivateFetch() {
				return dialer.DialContext(ctx, network, addr)
			}
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			var lastErr error
			for _, item := range ips {
				if IsBlockedIP(item.IP) {
					lastErr = fmt.Errorf("目标地址 %s 属于受保护的私有/保留网段", item.IP)
					continue
				}
				conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(item.IP.String(), port))
				if dialErr == nil {
					return conn, nil
				}
				lastErr = dialErr
			}
			if lastErr == nil {
				lastErr = errors.New("无法解析出站目标地址")
			}
			return nil, lastErr
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          50,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("重定向次数过多")
			}
			return ValidateOutboundURL(req.URL.String())
		},
	}
}
