package node

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var (
	multiplierPrefixRe = regexp.MustCompile(`(?i)(?:x|倍率|倍)\s*([0-9]+(?:\.[0-9]+)?)`)
	multiplierSuffixRe = regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?)\s*(?:x|倍率|倍)`)
)

// ParseMultiplier 从节点名中解析倍率（如 "x2"、"2倍"、"1.5x"），未识别返回 0。
func ParseMultiplier(name string) float64 {
	if name == "" {
		return 0
	}
	if m := multiplierPrefixRe.FindStringSubmatch(name); m != nil {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil && v > 0 {
			return v
		}
	}
	if m := multiplierSuffixRe.FindStringSubmatch(name); m != nil {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil && v > 0 {
			return v
		}
	}
	return 0
}

// NodeFlags carries per-node build overrides that are not always representable
// inside a share link (udp / tfo / skip-cert-verify). They are persisted on
// models.Node as JSON and honoured by the Clash/Surge/Loon encoders.
type NodeFlags struct {
	UDP            bool `json:"udp,omitempty"`
	TFO            bool `json:"tfo,omitempty"`
	SkipCertVerify bool `json:"skipCertVerify,omitempty"`
}

// Empty reports whether no flag is set.
func (f NodeFlags) Empty() bool {
	return !f.UDP && !f.TFO && !f.SkipCertVerify
}

// NodeInput is a link plus its per-node build flags.
type NodeInput struct {
	Link  string
	Flags NodeFlags
}

// Outbound is a normalized representation of a single outbound node. It keeps
// the decoded protocol struct in Spec so operations can mutate common fields
// and re-encode a canonical share link without lossy string surgery.
type Outbound struct {
	Name           string
	Type           string
	Server         string
	Port           int
	UDP            bool
	TFO            bool
	SkipCertVerify bool
	Spec           interface{}
	Link           string
	ScriptID       int // 内部使用：脚本 operator 的稳定标识（>=1），不参与序列化
}

// Flags returns the normalized build flags of the proxy.
func (p Outbound) Flags() NodeFlags {
	return NodeFlags{UDP: p.UDP, TFO: p.TFO, SkipCertVerify: p.SkipCertVerify}
}

// ParseOutbound decodes a share link into a normalized Outbound.
func ParseOutbound(link string) (Outbound, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return Outbound{}, fmt.Errorf("空链接")
	}
	if !strings.Contains(link, "://") {
		return Outbound{}, fmt.Errorf("链接缺少协议头: %s", link)
	}
	scheme := strings.ToLower(strings.SplitN(link, "://", 2)[0])
	p := Outbound{Link: link, Type: scheme}
	var err error
	switch scheme {
	case "ss":
		var v Ss
		v, err = DecodeSSURL(link)
		p.Spec = v
		p.Name, p.Server, p.Port = v.Name, v.Server, v.Port
	case "ssr":
		var v Ssr
		v, err = DecodeSSRURL(link)
		p.Spec = v
		p.Name, p.Server, p.Port = v.Qurey.Remarks, v.Server, v.Port
	case "vmess":
		var v Vmess
		v, err = DecodeVMESSURL(link)
		p.Spec = v
		p.Port, _ = convertToInt(v.Port)
		p.Name, p.Server = v.Ps, v.Add
	case "vless":
		var v VLESS
		v, err = DecodeVLESSURL(link)
		p.Spec = v
		p.Name, p.Server, p.Port = v.Name, v.Server, v.Port
	case "trojan":
		var v Trojan
		v, err = DecodeTrojanURL(link)
		p.Spec = v
		p.Name, p.Server, p.Port = v.Name, v.Hostname, v.Port
	case "hysteria", "hy":
		var v HY
		v, err = DecodeHYURL(link)
		p.Spec = v
		p.Name, p.Server, p.Port = v.Name, v.Host, v.Port
	case "hysteria2", "hy2":
		var v HY2
		v, err = DecodeHY2URL(link)
		p.Spec = v
		p.Name, p.Server, p.Port = v.Name, v.Host, v.Port
	case "tuic":
		var v Tuic
		v, err = DecodeTuicURL(link)
		p.Spec = v
		p.Name, p.Server, p.Port = v.Name, v.Host, v.Port
	case "anytls":
		var v AnyTLS
		v, err = DecodeAnyTLSURL(link)
		p.Spec = v
		p.Name, p.Server, p.Port = v.Name, v.Host, v.Port
	case "snell":
		var v Snell
		v, err = DecodeSnellURL(link)
		p.Spec = v
		p.Name, p.Server, p.Port = v.Name, v.Host, v.Port
	case "socks5", "socks5h", "http", "https":
		var v Socks
		v, err = DecodeSocksURL(link)
		p.Spec = v
		p.Name, p.Server, p.Port = v.Name, v.Host, v.Port
	case "ssh":
		var v SSH
		v, err = DecodeSSHURL(link)
		p.Spec = v
		p.Name, p.Server, p.Port = v.Name, v.Host, v.Port
	case "wireguard", "wg":
		var v WireGuard
		v, err = DecodeWireGuardURL(link)
		p.Spec = v
		p.Name, p.Server, p.Port = v.Name, v.Host, v.Port
	default:
		p.Server, p.Port = ExtractServerHost(link)
		return p, fmt.Errorf("暂不支持的协议: %s", scheme)
	}
	if err != nil {
		return p, err
	}
	if p.Name == "" {
		p.Name = fmt.Sprintf("%s:%d", p.Server, p.Port)
	}
	return p, nil
}

// ToLink re-encodes the proxy back into a canonical share link.
func (p Outbound) ToLink() string {
	name := p.Name
	if name == "" {
		name = fmt.Sprintf("%s:%d", p.Server, p.Port)
	}
	switch v := p.Spec.(type) {
	case Ss:
		v.Name, v.Server, v.Port = name, p.Server, p.Port
		return EncodeSSURL(v)
	case Ssr:
		v.Server, v.Port = p.Server, p.Port
		v.Qurey.Remarks = name
		return EncodeSSRURL(v)
	case Vmess:
		v.Add, v.Ps = p.Server, name
		v.Port = fmt.Sprintf("%d", p.Port)
		return EncodeVmessURL(v)
	case VLESS:
		v.Name, v.Server, v.Port = name, p.Server, p.Port
		return EncodeVLESSURL(v)
	case Trojan:
		v.Name, v.Hostname, v.Port = name, p.Server, p.Port
		return EncodeTrojanURL(v)
	case HY:
		v.Name, v.Host, v.Port = name, p.Server, p.Port
		return EncodeHYURL(v)
	case HY2:
		v.Name, v.Host, v.Port = name, p.Server, p.Port
		return EncodeHY2URL(v)
	case Tuic:
		v.Name, v.Host, v.Port = name, p.Server, p.Port
		return EncodeTuicURL(v)
	case AnyTLS:
		v.Name, v.Host, v.Port = name, p.Server, p.Port
		return EncodeAnyTLSURL(v)
	case Snell:
		v.Name, v.Host, v.Port = name, p.Server, p.Port
		return EncodeSnellURL(v)
	case Socks:
		v.Name, v.Host, v.Port = name, p.Server, p.Port
		return EncodeSocksURL(v)
	case SSH:
		v.Name, v.Host, v.Port = name, p.Server, p.Port
		return EncodeSSHURL(v)
	case WireGuard:
		v.Name, v.Host, v.Port = name, p.Server, p.Port
		return EncodeWireGuardURL(v)
	default:
		return p.Link
	}
}

// WithName returns a copy of the proxy renamed to name, with the link updated.
func (p Outbound) WithName(name string) Outbound {
	p.Name = name
	p.Link = p.ToLink()
	return p
}

// CountryCode resolves the proxy server to an ISO country code using the
// embedded GeoIP database. It returns "" when the server cannot be resolved.
func (p Outbound) CountryCode() string {
	host := p.Server
	if host == "" {
		host, _ = ExtractServerHost(p.Link)
	}
	return LookupCountry(host)
}

// FlagEmoji returns the regional-indicator flag emoji for an ISO country code.
func FlagEmoji(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 2 || code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z' {
		return ""
	}
	return string([]rune{
		rune(0x1F1E6 + int(code[0]-'A')),
		rune(0x1F1E6 + int(code[1]-'A')),
	})
}

// countryKeywords maps common node-name tokens (in several languages) to ISO
// country codes, used as a fallback when GeoIP cannot resolve the server.
var countryKeywords = []struct {
	Keyword string
	Code    string
}{
	{"香港", "HK"}, {"HongKong", "HK"}, {"Hong Kong", "HK"}, {"HKG", "HK"}, {"hk", "HK"},
	{"台湾", "TW"}, {"臺灣", "TW"}, {"Taiwan", "TW"}, {"TWN", "TW"}, {"tw", "TW"},
	{"日本", "JP"}, {"Japan", "JP"}, {"Tokyo", "JP"}, {"Osaka", "JP"}, {"JPN", "JP"}, {"jp", "JP"},
	{"新加坡", "SG"}, {"狮城", "SG"}, {"Singapore", "SG"}, {"SGP", "SG"}, {"sg", "SG"},
	{"韩国", "KR"}, {"韓國", "KR"}, {"Korea", "KR"}, {"Seoul", "KR"}, {"KOR", "KR"}, {"kr", "KR"},
	{"美国", "US"}, {"美國", "US"}, {"United States", "US"}, {"Los Angeles", "US"}, {"Seattle", "US"}, {"USA", "US"}, {"us", "US"},
	{"英国", "GB"}, {"英國", "GB"}, {"United Kingdom", "GB"}, {"London", "GB"}, {"UK", "GB"}, {"gb", "GB"},
	{"德国", "DE"}, {"德國", "DE"}, {"Germany", "DE"}, {"Frankfurt", "DE"}, {"DE", "DE"},
	{"法国", "FR"}, {"France", "FR"}, {"Paris", "FR"}, {"FR", "FR"},
	{"荷兰", "NL"}, {"Netherlands", "NL"}, {"Amsterdam", "NL"}, {"NL", "NL"},
	{"俄罗斯", "RU"}, {"Russia", "RU"}, {"Moscow", "RU"}, {"RU", "RU"},
	{"加拿大", "CA"}, {"Canada", "CA"}, {"Toronto", "CA"}, {"CA", "CA"},
	{"澳大利亚", "AU"}, {"Australia", "AU"}, {"Sydney", "AU"}, {"AU", "AU"},
	{"印度", "IN"}, {"India", "IN"}, {"Mumbai", "IN"}, {"IN", "IN"},
	{"土耳其", "TR"}, {"Turkey", "TR"}, {"TR", "TR"},
	{"马来西亚", "MY"}, {"Malaysia", "MY"}, {"MY", "MY"},
	{"泰国", "TH"}, {"Thailand", "TH"}, {"TH", "TH"},
	{"越南", "VN"}, {"Vietnam", "VN"}, {"VN", "VN"},
	{"菲律宾", "PH"}, {"Philippines", "PH"}, {"PH", "PH"},
	{"印尼", "ID"}, {"印度尼西亚", "ID"}, {"Indonesia", "ID"}, {"ID", "ID"},
	{"巴西", "BR"}, {"Brazil", "BR"}, {"BR", "BR"},
	{"意大利", "IT"}, {"Italy", "IT"}, {"IT", "IT"},
	{"西班牙", "ES"}, {"Spain", "ES"}, {"ES", "ES"},
	{"瑞士", "CH"}, {"Switzerland", "CH"}, {"CH", "CH"},
	{"瑞典", "SE"}, {"Sweden", "SE"}, {"SE", "SE"},
	{"波兰", "PL"}, {"Poland", "PL"}, {"PL", "PL"},
	{"阿根廷", "AR"}, {"Argentina", "AR"}, {"AR", "AR"},
	{"南非", "ZA"}, {"South Africa", "ZA"}, {"ZA", "ZA"},
	{"阿联酋", "AE"}, {"迪拜", "AE"}, {"Dubai", "AE"}, {"AE", "AE"},
	{"以色列", "IL"}, {"Israel", "IL"}, {"IL", "IL"},
	{"乌克兰", "UA"}, {"Ukraine", "UA"}, {"UA", "UA"},
	{"墨西哥", "MX"}, {"Mexico", "MX"}, {"MX", "MX"},
}

// CountryFromName guesses an ISO country code from the node name.
func CountryFromName(name string) string {
	if name == "" {
		return ""
	}
	lower := strings.ToLower(name)
	for _, item := range countryKeywords {
		if strings.Contains(lower, strings.ToLower(item.Keyword)) {
			return item.Code
		}
	}
	return ""
}

// ResolveDomain resolves a domain server to an IP literal. It returns the
// original value when it is already an IP or cannot be resolved.
func ResolveDomain(server string) (string, bool) {
	server = strings.TrimSpace(server)
	if server == "" || net.ParseIP(server) != nil {
		return server, false
	}
	addrs, err := net.LookupHost(server)
	if err != nil {
		return server, false
	}
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && ip.To4() != nil {
			return addr, true
		}
	}
	if len(addrs) > 0 {
		return addrs[0], true
	}
	return server, false
}
