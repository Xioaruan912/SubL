package node

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// EncodeSingbox builds a complete sing-box client configuration where every
// node becomes an outbound and a "proxy" selector points at all of them.
func EncodeSingbox(urls []string, cfg SqlConfig) (string, error) {
	outbounds := make([]map[string]interface{}, 0, len(urls)+1)
	names := make([]string, 0, len(urls))
	used := map[string]int{}
	for _, link := range urls {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}
		ob, name, err := singboxOutbound(link)
		if err != nil {
			log.Printf("[singbox] 跳过节点: %v", err)
			continue
		}
		name = uniqueTag(name, used)
		ob["tag"] = name
		outbounds = append(outbounds, ob)
		names = append(names, name)
	}
	if len(names) == 0 {
		return "", fmt.Errorf("没有可输出的 sing-box 节点")
	}
	selector := map[string]interface{}{
		"type":      "selector",
		"tag":       "proxy",
		"outbounds": append([]string{}, names...),
		"default":   names[0],
	}
	head := []map[string]interface{}{
		selector,
		{"type": "direct", "tag": "direct"},
	}
	config := map[string]interface{}{
		"log":       map[string]interface{}{"level": "info"},
		"outbounds": append(head, outbounds...),
		"route":     map[string]interface{}{"final": "proxy"},
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func uniqueTag(name string, used map[string]int) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "node"
	}
	if used[name] == 0 {
		used[name] = 1
		return name
	}
	used[name]++
	return fmt.Sprintf("%s #%d", name, used[name])
}

func sbTLS(sni string, insecure bool, fp string) map[string]interface{} {
	t := map[string]interface{}{"enabled": true, "insecure": insecure}
	if sni != "" {
		t["server_name"] = sni
	}
	if fp != "" {
		t["utls"] = map[string]interface{}{"enabled": true, "fingerprint": fp}
	}
	return t
}

func sbWS(path, host string) map[string]interface{} {
	m := map[string]interface{}{"type": "ws", "path": path}
	if host != "" {
		m["headers"] = map[string]interface{}{"Host": host}
	}
	return m
}

func sbGRPC(service string) map[string]interface{} {
	return map[string]interface{}{"type": "grpc", "service_name": service}
}

func singboxOutbound(link string) (map[string]interface{}, string, error) {
	p, err := ParseOutbound(link)
	if err != nil {
		return nil, "", err
	}
	name := p.Name
	switch v := p.Spec.(type) {
	case Ss:
		return map[string]interface{}{
			"type": "shadowsocks", "server": v.Server, "server_port": v.Port,
			"method": v.Param.Cipher, "password": v.Param.Password,
		}, name, nil

	case Vmess:
		port, _ := convertToInt(v.Port)
		m := map[string]interface{}{
			"type": "vmess", "server": v.Add, "server_port": port,
			"uuid": v.Id, "security": "auto", "alter_id": 0,
		}
		switch strings.ToLower(v.Net) {
		case "ws":
			m["transport"] = sbWS(v.Path, v.Host)
		case "grpc":
			m["transport"] = sbGRPC(v.Path)
		}
		if v.Tls != "" && v.Tls != "none" {
			m["tls"] = sbTLS(defaultNonEmpty(v.Sni, v.Host), false, v.Fp)
		}
		return m, name, nil

	case VLESS:
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(buildVLESSOutbound(v)), &m); err != nil {
			return nil, "", err
		}
		delete(m, "tag")
		return m, name, nil

	case Trojan:
		m := map[string]interface{}{"type": "trojan", "server": v.Hostname, "server_port": v.Port, "password": v.Password}
		if v.Query.Sni != "" || v.Query.AllowInsecure != 0 {
			m["tls"] = sbTLS(v.Query.Sni, true, v.Query.Fp)
		}
		if v.Query.Type == "ws" {
			m["transport"] = sbWS(v.Query.Path, v.Query.Host)
		}
		return m, name, nil

	case HY2:
		m := map[string]interface{}{"type": "hysteria2", "server": v.Host, "server_port": v.Port, "password": v.Password}
		m["tls"] = sbTLS(v.Sni, true, "")
		if v.Obfs != "" {
			m["obfs"] = map[string]interface{}{"type": "salamander", "password": v.ObfsPassword}
		}
		return m, name, nil

	case Tuic:
		congestion := v.Congestion_control
		if congestion == "" {
			congestion = "bbr"
		}
		tls := sbTLS(v.Sni, true, "")
		if len(v.Alpn) > 0 {
			tls["alpn"] = v.Alpn
		}
		return map[string]interface{}{
			"type": "tuic", "server": v.Host, "server_port": v.Port,
			"uuid": v.Uuid, "password": v.Password, "congestion_control": congestion, "tls": tls,
		}, name, nil

	case AnyTLS:
		return map[string]interface{}{
			"type": "anytls", "server": v.Host, "server_port": v.Port,
			"password": v.Password, "tls": sbTLS(v.Sni, v.Insecure != 0, v.Fp),
		}, name, nil

	case Socks:
		typ := "socks"
		if v.Type == "http" {
			typ = "http"
		}
		m := map[string]interface{}{"type": typ, "server": v.Host, "server_port": v.Port}
		if v.Username != "" {
			m["username"] = v.Username
		}
		if v.Password != "" {
			m["password"] = v.Password
		}
		if v.Tls {
			m["tls"] = sbTLS("", false, "")
		}
		return m, name, nil

	case WireGuard:
		m := map[string]interface{}{
			"type": "wireguard", "server": v.Host, "server_port": v.Port,
			"private_key": v.PrivateKey, "peer_public_key": v.PublicKey,
		}
		if v.Address != "" {
			m["local_address"] = strings.Split(v.Address, ",")
		}
		if v.Mtu > 0 {
			m["mtu"] = v.Mtu
		}
		if v.Reserved != "" {
			m["reserved"] = parseIntList(v.Reserved)
		}
		return m, name, nil

	default:
		return nil, "", fmt.Errorf("sing-box 暂不支持协议: %s", p.Type)
	}
}

func defaultNonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
