package node

import (
	"embed"
	"encoding/json"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

//go:embed data/GeoLite2-Country.mmdb
var geoDB embed.FS

var (
	geoOnce sync.Once
	geoRdr  *geoip2.Reader
	geoErr  error
)

// 国家/地区代码 -> 质心坐标（用于地图打点）
var countryCoord = map[string][2]float64{
	"US": {39.8283, -98.5795}, "JP": {36.2048, 138.2529},
	"HK": {22.3193, 114.1694}, "SG": {1.3521, 103.8198},
	"KR": {35.9078, 127.7669}, "TW": {23.6978, 120.9605},
	"CN": {35.8617, 104.1954}, "DE": {51.1657, 10.4515},
	"GB": {55.3781, -3.4360},  "FR": {46.2276, 2.2137},
	"NL": {52.1326, 5.2913},  "RU": {61.5240, 105.3188},
	"CA": {56.1304, -106.3468}, "AU": {-25.2744, 133.7751},
	"IN": {20.5937, 78.9629}, "BR": {-14.2350, -51.9253},
	"IT": {41.8719, 12.5674}, "ES": {40.4637, -3.7492},
	"SE": {60.1282, 18.6435}, "FI": {61.9241, 25.7482},
	"NO": {60.4720, 8.4689},  "DK": {56.2639, 9.5018},
	"CH": {46.8182, 8.2275},  "AT": {47.5162, 14.5501},
	"BE": {50.5039, 4.4699},  "IE": {53.4129, -8.2439},
	"PL": {51.9194, 19.1451}, "CZ": {49.8175, 15.4730},
	"TR": {38.9637, 35.2433}, "AE": {23.4241, 53.8478},
	"ID": {-0.7893, 113.9213}, "MY": {4.2105, 101.9758},
	"TH": {15.8700, 100.9925}, "VN": {14.0583, 108.2772},
	"PH": {12.8797, 121.7740}, "NZ": {-40.9006, 174.8860},
	"MX": {23.6345, -102.5528}, "AR": {-38.4161, -63.6167},
	"ZA": {-30.5595, 22.9375}, "EG": {26.8206, 30.8025},
	"UA": {48.3794, 31.1656},  "BG": {42.7339, 25.4858},
	"GR": {39.0742, 21.8243},  "PT": {39.3999, -8.2245},
	"IL": {31.0461, 34.8516},  "SA": {23.8859, 45.0792},
	"KZ": {48.0196, 66.9237},  "UZ": {41.3775, 64.5853},
	"CO": {4.5709, -74.2973},  "CL": {-35.6751, -71.5430},
	"PE": {-9.1900, -75.0152}, "RO": {45.9432, 24.9668},
	"HU": {47.1625, 19.5033},  "LT": {55.1694, 23.8813},
	"EE": {58.5953, 25.0136},  "LV": {56.8796, 24.6032},
	"IS": {64.9631, -19.0208}, "LU": {49.8153, 6.1296},
	"CY": {35.1264, 33.4299},  "MT": {35.9375, 14.3754},
	"SC": {-4.6796, 55.4920},  "PA": {8.5380, -80.7821},
	"LI": {47.1660, 9.5554},   "SK": {48.6690, 19.6990},
	"HR": {45.1000, 15.2000},  "SI": {46.1512, 14.9955},
	"RS": {44.0165, 21.0059},  "GE": {42.3154, 43.3569},
	"AM": {40.0691, 45.0382},  "AZ": {40.1431, 47.5769},
	"MD": {47.4116, 28.3699},  "BY": {53.7098, 27.9534},
	"AF": {33.9391, 67.7100},  "PK": {30.3753, 69.3451},
	"BD": {23.6850, 90.3563},  "LK": {7.8731, 80.7718},
	"NP": {28.3949, 84.1240},  "MM": {21.9162, 95.9560},
	"KH": {12.5657, 104.9910}, "LA": {19.8563, 102.4955},
	"MO": {22.1987, 113.5439}, "IR": {32.4279, 53.6880},
	"IQ": {33.2232, 43.6793},  "QA": {25.3548, 51.1839},
	"KW": {29.3117, 47.4818},  "BH": {25.9304, 50.6378},
	"OM": {21.5126, 55.9233},  "YE": {15.5527, 48.5164},
	"JO": {30.5852, 36.2384},  "LB": {33.8547, 35.8623},
	"NG": {9.0820, 8.6753},    "KE": {-0.0236, 37.9062},
	"GH": {7.9465, -1.0232},   "DZ": {28.0339, 1.6596},
	"MA": {31.7917, -7.0926},  "TN": {33.8869, 9.5375},
	"LY": {26.3351, 17.2283},  "SD": {12.8628, 30.2176},
	"ET": {9.1450, 40.4897},   "TZ": {-6.3690, 34.8888},
	"UG": {1.3733, 32.2903},   "CI": {7.5400, -5.5471},   "SN": {14.4974, -14.4524},
	"VE": {6.4238, -66.5897},  "EC": {-1.8312, -78.1834},
	"BO": {-16.2902, -63.5887}, "PY": {-23.4425, -58.4438},
	"UY": {-32.5228, -55.7658}, "GT": {15.7835, -90.2308},
	"CR": {9.7489, -83.7534},  "CU": {21.5218, -77.7812},
	"DO": {18.7357, -70.1627}, "JM": {18.1096, -77.2975},
	"PR": {18.2208, -66.5901},
}

var defaultCoord = [2]float64{20.0, 0.0}

func getGeoReader() (*geoip2.Reader, error) {
	geoOnce.Do(func() {
		buf, err := geoDB.ReadFile("data/GeoLite2-Country.mmdb")
		if err != nil {
			geoErr = err
			return
		}
		geoRdr, geoErr = geoip2.FromBytes(buf)
	})
	return geoRdr, geoErr
}

// ExtractServerHost 从任意协议订阅链接中解析服务器主机与端口。
// 返回 host、port（无端口时为 0）。host 可能是域名或 IP。
func ExtractServerHost(link string) (string, int) {
	link = strings.TrimSpace(link)
	if link == "" {
		return "", 0
	}
	// 订阅源链接（http/https）不是节点，跳过
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return "", 0
	}
	// vmess 是 base64 编码的特殊格式，需要单独解析
	if strings.HasPrefix(link, "vmess://") {
		raw := strings.TrimPrefix(link, "vmess://")
		decoded := Base64Decode2(raw)
		return extractVmessHostPort(decoded)
	}
	u, err := url.Parse(link)
	if err != nil {
		return "", 0
	}
	host := u.Hostname()
	if host == "" {
		// 兜底：去掉 scheme 后取 @ 后的部分
		rest := link
		if idx := strings.Index(rest, "://"); idx >= 0 {
			rest = rest[idx+3:]
		}
		if idx := strings.LastIndex(rest, "@"); idx >= 0 {
			rest = rest[idx+1:]
		}
		if idx := strings.Index(rest, "#"); idx >= 0 {
			rest = rest[:idx]
		}
		if idx := strings.Index(rest, "?"); idx >= 0 {
			rest = rest[:idx]
		}
		rest = strings.Trim(rest, "/")
		if h, _, err := net.SplitHostPort(rest); err == nil {
			host = h
		} else {
			host = rest
		}
	}
	port := 0
	if u.Port() != "" {
		if p, err := strconv.Atoi(u.Port()); err == nil {
			port = p
		}
	}
	return strings.Trim(host, "[]"), port
}

type vmessInfo struct {
	Add string      `json:"add"`
	Port interface{} `json:"port"`
}

// extractVmessHostPort 从 vmess 解码后的 JSON 中解析 add/port
func extractVmessHostPort(decoded string) (string, int) {
	if decoded == "" {
		return "", 0
	}
	var vi vmessInfo
	if err := json.Unmarshal([]byte(decoded), &vi); err != nil {
		return "", 0
	}
	host := strings.Trim(vi.Add, "[]")
	port := 0
	switch p := vi.Port.(type) {
	case float64:
		port = int(p)
	case string:
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}
	return host, port
}

// LookupCountry 通过内置 GeoIP 数据库查询 IP 所属国家码。
// 域名无法解析为 IP 时返回空字符串。
func LookupCountry(host string) string {
	ip := net.ParseIP(host)
	if ip == nil {
		addrs, err := net.LookupHost(host)
		if err != nil || len(addrs) == 0 {
			return ""
		}
		ip = net.ParseIP(addrs[0])
		if ip == nil {
			return ""
		}
	}
	rdr, err := getGeoReader()
	if err != nil {
		log.Println("GeoIP reader error:", err)
		return ""
	}
	record, err := rdr.Country(ip)
	if err != nil {
		return ""
	}
	return record.Country.IsoCode
}

// CountryCoord 返回国家码对应的质心坐标，未知国家返回默认坐标。
func CountryCoord(code string) [2]float64 {
	if c, ok := countryCoord[code]; ok {
		return c
	}
	return defaultCoord
}