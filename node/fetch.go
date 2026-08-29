package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	singbox "github.com/sagernet/sing-box"
)

// FetchURLThroughNode fetches an HTTP(S) resource through exactly one SubLinkX
// proxy node. It is used as a fallback for rule hosts that reject the VPS's
// direct egress IP (for example returning HTTP 403).
func FetchURLThroughNode(ctx context.Context, link, rawURL, userAgent string, timeout time.Duration, maxBytes int64) ([]byte, http.Header, error) {
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	if maxBytes <= 0 {
		maxBytes = 4 << 20
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, nil, errors.New("invalid URL")
	}
	outbound, _, err := buildOutboundConfig(link)
	if err != nil {
		return nil, nil, err
	}
	port, err := freePort()
	if err != nil {
		return nil, nil, err
	}
	options, err := parseOptions(buildSingboxConfig(port, outbound))
	if err != nil {
		return nil, nil, err
	}
	instance, err := singbox.New(singbox.Options{Context: getBoxContext(), Options: *options})
	if err != nil {
		return nil, nil, err
	}
	if err := instance.Start(); err != nil {
		instance.Close()
		return nil, nil, err
	}
	defer instance.Close()

	proxyURL, _ := url.Parse("socks5://127.0.0.1:" + strconv.Itoa(port))
	transport := &http.Transport{
		Proxy:                 http.ProxyURL(proxyURL),
		DialContext:           (&net.Dialer{Timeout: timeout}).DialContext,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		MaxIdleConns:          4,
		MaxIdleConnsPerHost:   2,
		ForceAttemptHTTP2:     true,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, nil, err
	}
	if userAgent == "" { userAgent = "SubLinkX-RuleCenter/1.0" }
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.Header, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, resp.Header, err
	}
	if int64(len(body)) > maxBytes {
		return nil, resp.Header, errors.New("response exceeds size limit")
	}
	return body, resp.Header, nil
}
