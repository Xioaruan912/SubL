package node

import (
	"fmt"
	"log"
	"strings"
)

// EncodeQX builds a Quantumult X [server_local] proxy list from share links.
// Unsupported protocols are skipped.
func EncodeQX(urls []string, cfg SqlConfig) (string, error) {
	lines := make([]string, 0, len(urls))
	for _, link := range urls {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}
		line, err := qxLine(link)
		if err != nil {
			log.Printf("[qx] 跳过节点: %v", err)
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("没有可输出的 Quantumult X 节点")
	}
	return strings.Join(lines, "\n"), nil
}

func qxTag(name string) string {
	name = strings.TrimSpace(name)
	name = strings.NewReplacer(",", " ", "\n", " ").Replace(name)
	if name == "" {
		return "node"
	}
	return name
}

func qxLine(link string) (string, error) {
	p, err := ParseOutbound(link)
	if err != nil {
		return "", err
	}
	tag := qxTag(p.Name)
	switch v := p.Spec.(type) {
	case Ss:
		return fmt.Sprintf("shadowsocks=%s:%d, method=%s, password=%s, udp-relay=true, tag=%s",
			v.Server, v.Port, v.Param.Cipher, v.Param.Password, tag), nil
	case Vmess:
		port, _ := convertToInt(v.Port)
		parts := []string{
			fmt.Sprintf("vmess=%s:%d", v.Add, port),
			"method=chacha20-poly1305",
			"password=" + v.Id,
		}
		if strings.EqualFold(v.Net, "ws") {
			parts = append(parts, "obfs=ws", "obfs-uri="+defaultNonEmpty(v.Path, "/"))
			if v.Host != "" && v.Host != "none" {
				parts = append(parts, "obfs-host="+v.Host)
			}
		}
		if v.Tls != "" && v.Tls != "none" {
			parts = append(parts, "over-tls=true", "tls-verification=true")
			if v.Sni != "" {
				parts = append(parts, "tls-host="+v.Sni)
			}
		}
		parts = append(parts, "tag="+tag)
		return strings.Join(parts, ", "), nil
	case Trojan:
		parts := []string{
			fmt.Sprintf("trojan=%s:%d", v.Hostname, v.Port),
			"password=" + v.Password,
			"over-tls=true",
			"tls-verification=true",
		}
		if v.Query.Sni != "" {
			parts = append(parts, "tls-host="+v.Query.Sni)
		}
		if strings.EqualFold(v.Query.Type, "ws") {
			parts = append(parts, "obfs=ws", "obfs-uri="+defaultNonEmpty(v.Query.Path, "/"))
			if v.Query.Host != "" {
				parts = append(parts, "obfs-host="+v.Query.Host)
			}
		}
		parts = append(parts, "tag="+tag)
		return strings.Join(parts, ", "), nil
	case VLESS:
		parts := []string{
			fmt.Sprintf("vless=%s:%d", v.Server, v.Port),
			"method=none",
			"password=" + v.Uuid,
		}
		if v.Query.Flow != "" {
			parts = append(parts, "obfs="+v.Query.Flow)
		}
		if strings.EqualFold(v.Query.Type, "ws") {
			parts = append(parts, "obfs=ws", "obfs-uri="+defaultNonEmpty(v.Query.Path, "/"))
			if v.Query.Host != "" {
				parts = append(parts, "obfs-host="+v.Query.Host)
			}
		}
		if v.Query.Security == "tls" || v.Query.Security == "reality" {
			parts = append(parts, "over-tls=true", "tls-verification=true")
			if v.Query.Sni != "" {
				parts = append(parts, "tls-host="+v.Query.Sni)
			}
		}
		parts = append(parts, "tag="+tag)
		return strings.Join(parts, ", "), nil
	case Socks:
		typ := "socks5"
		if v.Type == "http" {
			typ = "http"
		}
		parts := []string{fmt.Sprintf("%s=%s:%d", typ, v.Host, v.Port)}
		if v.Username != "" {
			parts = append(parts, "username="+v.Username)
		}
		if v.Password != "" {
			parts = append(parts, "password="+v.Password)
		}
		parts = append(parts, "tag="+tag)
		return strings.Join(parts, ", "), nil
	default:
		return "", fmt.Errorf("Quantumult X 暂不支持协议: %s", p.Type)
	}
}
