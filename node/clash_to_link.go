package node

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ClashNode is a parsed node with an explicit display name and a share link.
type ClashNode struct {
	Name string
	Link string
}

type clashConfig struct {
	Proxies []map[string]interface{} `yaml:"proxies"`
}

// IsClashConfig reports whether the payload is a Clash/Mihomo YAML config that
// carries at least one proxy. Plain or base64 node-link lists return false.
func IsClashConfig(data []byte) bool {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return false
	}
	var cfg clashConfig
	if err := yaml.Unmarshal([]byte(trimmed), &cfg); err != nil {
		return false
	}
	return len(cfg.Proxies) > 0
}

// ParseClashToNodes converts a Clash/Mihomo YAML config into share links.
func ParseClashToNodes(data []byte) ([]ClashNode, error) {
	var cfg clashConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析 Clash YAML 失败: %w", err)
	}
	if len(cfg.Proxies) == 0 {
		return nil, fmt.Errorf("Clash YAML 中没有 proxies 节点")
	}
	nodes := make([]ClashNode, 0, len(cfg.Proxies))
	var firstErr error
	for idx, p := range cfg.Proxies {
		n, err := ClashProxyToLink(p)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("第 %d 个节点(%s)转换失败: %w", idx+1, stringVal(p, "name"), err)
			}
			continue
		}
		nodes = append(nodes, n)
	}
	if len(nodes) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, fmt.Errorf("Clash YAML 中的节点均无法识别")
	}
	return nodes, nil
}

// ClashProxyToLink converts a single Clash proxy map into a share link.
func ClashProxyToLink(p map[string]interface{}) (ClashNode, error) {
	typ := strings.ToLower(stringVal(p, "type"))
	server := stringVal(p, "server")
	port := intVal(p, "port")
	name := stringVal(p, "name")
	if server == "" || port <= 0 {
		return ClashNode{}, fmt.Errorf("缺少 server 或 port")
	}
	if name == "" {
		name = fmt.Sprintf("%s:%d", server, port)
	}
	link, err := clashProxyLink(typ, name, server, port, p)
	if err != nil {
		return ClashNode{}, err
	}
	return ClashNode{Name: name, Link: link}, nil
}

func clashProxyLink(typ, name, server string, port int, p map[string]interface{}) (string, error) {
	switch typ {
	case "ss":
		cipher := stringVal(p, "cipher")
		password := stringVal(p, "password")
		if cipher == "" || password == "" {
			return "", fmt.Errorf("ss 缺少 cipher/password")
		}
		return EncodeSSURL(Ss{Param: Param{Cipher: cipher, Password: password}, Server: server, Port: port, Name: name}), nil

	case "ssr":
		ssr := Ssr{
			Server:   server,
			Port:     port,
			Protocol: stringVal(p, "protocol"),
			Method:   stringVal(p, "cipher"),
			Obfs:     stringVal(p, "obfs"),
			Password: stringVal(p, "password"),
			Qurey: Ssrquery{
				Obfsparam: stringVal(p, "obfs-param"),
				Remarks:   name,
			},
		}
		return EncodeSSRURL(ssr), nil

	case "vmess":
		v := Vmess{
			Add:  server,
			Port: strconv.Itoa(port),
			Aid:  intVal(p, "alterId"),
			Id:   stringVal(p, "uuid"),
			Net:  defaultString(stringVal(p, "network"), "tcp"),
			Type: "none",
			Ps:   name,
			Scy:  defaultString(stringVal(p, "cipher"), "auto"),
			Tls:  boolTLS(p),
			Sni:  defaultString(stringVal(p, "servername"), stringVal(p, "sni")),
			Fp:   stringVal(p, "client-fingerprint"),
		}
		applyVmessTransport(&v, p)
		return EncodeVmessURL(v), nil

	case "vless":
		q := VLESSQuery{
			Security:   clashSecurity(p),
			Sni:        defaultString(stringVal(p, "servername"), stringVal(p, "sni")),
			Fp:         stringVal(p, "client-fingerprint"),
			Flow:       stringVal(p, "flow"),
			Encryption: "none",
			Type:       defaultString(stringVal(p, "network"), "tcp"),
			HeaderType: "none",
		}
		applyRealityOpts(&q, p)
		applyVLESSHostPath(&q, p)
		return EncodeVLESSURL(VLESS{Name: name, Uuid: stringVal(p, "uuid"), Server: server, Port: port, Query: q}), nil

	case "trojan":
		tq := TrojanQuery{
			Type:     defaultString(stringVal(p, "network"), "tcp"),
			Security: "tls",
			Fp:       stringVal(p, "client-fingerprint"),
			Sni:      defaultString(stringVal(p, "sni"), stringVal(p, "servername")),
			Peer:     defaultString(stringVal(p, "sni"), stringVal(p, "servername")),
			Flow:     stringVal(p, "flow"),
		}
		if ws := nested(p, "ws-opts"); ws != nil {
			tq.Path = stringVal(ws, "path")
			if h := nested(ws, "headers"); h != nil {
				tq.Host = stringVal(h, "Host")
			}
		}
		return EncodeTrojanURL(Trojan{Password: stringVal(p, "password"), Hostname: server, Port: port, Query: tq, Name: name, Type: "trojan"}), nil

	case "hysteria":
		hy := HY{
			Host:     server,
			Port:     port,
			Insecure: boolInt(boolVal(p, "skip-cert-verify")),
			Peer:     defaultString(stringVal(p, "sni"), stringVal(p, "peer")),
			Auth:     defaultString(stringVal(p, "auth_str"), stringVal(p, "auth-str")),
			UpMbps:   intVal(p, "up"),
			DownMbps: intVal(p, "down"),
			ALPN:     stringSlice(p, "alpn"),
			Name:     name,
		}
		return EncodeHYURL(hy), nil

	case "hysteria2", "hy2":
		hy2 := HY2{
			Password:     stringVal(p, "password"),
			Host:         server,
			Port:         port,
			Insecure:     boolInt(boolVal(p, "skip-cert-verify")),
			Peer:         defaultString(stringVal(p, "sni"), stringVal(p, "peer")),
			UpMbps:       intVal(p, "up"),
			DownMbps:     intVal(p, "down"),
			ALPN:         stringSlice(p, "alpn"),
			Name:         name,
			Sni:          stringVal(p, "sni"),
			Obfs:         stringVal(p, "obfs"),
			ObfsPassword: stringVal(p, "obfs-password"),
		}
		return EncodeHY2URL(hy2), nil

	case "tuic":
		t := Tuic{
			Name:               name,
			Password:           stringVal(p, "password"),
			Host:               server,
			Port:               port,
			Uuid:               stringVal(p, "uuid"),
			Congestion_control: defaultString(stringVal(p, "congestion-controller"), stringVal(p, "congestion_control")),
			Alpn:               stringSlice(p, "alpn"),
			Sni:                stringVal(p, "sni"),
			Udp_relay_mode:     defaultString(stringVal(p, "udp-relay-mode"), stringVal(p, "udp_relay_mode")),
			Disable_sni:        boolInt(boolVal(p, "disable-sni")),
		}
		return EncodeTuicURL(t), nil

	case "anytls":
		return EncodeAnyTLSURL(AnyTLS{
			Password: stringVal(p, "password"),
			Host:     server,
			Port:     port,
			Sni:      defaultString(stringVal(p, "sni"), stringVal(p, "servername")),
			Insecure: boolInt(boolVal(p, "skip-cert-verify")),
			Fp:       stringVal(p, "client-fingerprint"),
			Name:     name,
		}), nil

	case "socks5":
		return EncodeSocksURL(Socks{
			Type:     "socks5",
			Username: stringVal(p, "username"),
			Password: stringVal(p, "password"),
			Host:     server,
			Port:     port,
			Name:     name,
		}), nil

	case "http":
		return EncodeSocksURL(Socks{
			Type:     "http",
			Username: stringVal(p, "username"),
			Password: stringVal(p, "password"),
			Host:     server,
			Port:     port,
			Tls:      boolVal(p, "tls"),
			Name:     name,
		}), nil

	default:
		return "", fmt.Errorf("不支持的节点类型: %s", typ)
	}
}

func applyVmessTransport(v *Vmess, p map[string]interface{}) {
	switch strings.ToLower(v.Net) {
	case "ws":
		v.Type = "none"
		if ws := nested(p, "ws-opts"); ws != nil {
			v.Path = stringVal(ws, "path")
			if h := nested(ws, "headers"); h != nil {
				v.Host = stringVal(h, "Host")
			}
		}
	case "grpc":
		v.Type = "gun"
		if g := nested(p, "grpc-opts"); g != nil {
			v.Path = stringVal(g, "grpc-service-name")
		}
	default:
		v.Type = "none"
	}
}

func applyRealityOpts(q *VLESSQuery, p map[string]interface{}) {
	if r := nested(p, "reality-opts"); r != nil {
		q.Pbk = stringVal(r, "public-key")
		q.Sid = stringVal(r, "short-id")
	}
}

func applyVLESSHostPath(q *VLESSQuery, p map[string]interface{}) {
	switch strings.ToLower(q.Type) {
	case "ws":
		if ws := nested(p, "ws-opts"); ws != nil {
			q.Path = stringVal(ws, "path")
			if h := nested(ws, "headers"); h != nil {
				q.Host = stringVal(h, "Host")
			}
		}
	case "grpc":
		if g := nested(p, "grpc-opts"); g != nil {
			q.ServiceName = stringVal(g, "grpc-service-name")
		}
	}
}

func clashSecurity(p map[string]interface{}) string {
	if nested(p, "reality-opts") != nil {
		return "reality"
	}
	if boolVal(p, "tls") {
		return "tls"
	}
	return ""
}

func boolTLS(p map[string]interface{}) string {
	if boolVal(p, "tls") {
		return "tls"
	}
	return ""
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func stringVal(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func intVal(m map[string]interface{}, key string) int {
	if m == nil {
		return 0
	}
	switch t := m[key].(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	default:
		return 0
	}
}

func boolVal(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	switch t := m[key].(type) {
	case bool:
		return t
	case string:
		return t == "true"
	default:
		return false
	}
}

func nested(m map[string]interface{}, key string) map[string]interface{} {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

func stringSlice(m map[string]interface{}, key string) []string {
	if m == nil {
		return nil
	}
	raw, ok := m[key]
	if !ok || raw == nil {
		return nil
	}
	switch t := raw.(type) {
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, v := range t {
			out = append(out, strings.TrimSpace(fmt.Sprintf("%v", v)))
		}
		return out
	case []string:
		return t
	case string:
		if t == "" {
			return nil
		}
		parts := strings.Split(t, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	default:
		return nil
	}
}
