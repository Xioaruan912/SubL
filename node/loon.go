package node

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// EncodeLoon 将节点链接列表转为 Loon 配置文本（填充 [Proxy] 段）。
// 策略组靠模板内的 [Remote Filter] NameRegex 自动筛选，不做组填充。
func EncodeLoon(urls []string, sqlconfig SqlConfig) (string, error) {
	// 未配置 Loon 模板时使用默认本地模板
	if sqlconfig.Loon == "" {
		sqlconfig.Loon = "./template/loon.conf"
	}
	var proxys []string
	for _, link := range urls {
		scheme := strings.Split(link, "://")[0]
		switch {
		case scheme == "ss":
			ss, err := DecodeSSURL(link)
			if err != nil {
				log.Println(err)
				continue
			}
			p := fmt.Sprintf("%s = Shadowsocks,%s,%d,%s,\"%s\",fast-open=false,udp=%t",
				ss.Name, ss.Server, ss.Port, ss.Param.Cipher, ss.Param.Password, sqlconfig.Udp)
			proxys = append(proxys, p)
		case scheme == "vmess":
			v, err := DecodeVMESSURL(link)
			if err != nil {
				log.Println(err)
				continue
			}
			port, _ := convertToInt(v.Port)
			tls := v.Tls != "none" && v.Tls != ""
			cipher := v.Scy
			if cipher == "" {
				cipher = "auto"
			}
			p := fmt.Sprintf("%s = vmess,%s,%d,%s,\"%s\",transport=%s,alterId=0,over-tls=%t,udp=%t",
				v.Ps, v.Add, port, cipher, v.Id, v.Net, tls, sqlconfig.Udp)
			if v.Net == "ws" {
				p += fmt.Sprintf(",path=%s", v.Path)
				if v.Host != "" && v.Host != "none" {
					p += fmt.Sprintf(",host=%s", v.Host)
				}
			}
			if tls && v.Sni != "" {
				p += fmt.Sprintf(",sni=%s", v.Sni)
			}
			if sqlconfig.Cert {
				p += ",skip-cert-verify=true"
			}
			proxys = append(proxys, p)
		case scheme == "vless":
			v, err := DecodeVLESSURL(link)
			if err != nil {
				log.Println(err)
				continue
			}
			transport := v.Query.Type
			if transport == "" {
				transport = "tcp"
			}
			tls := v.Query.Security != "" && v.Query.Security != "none"
			p := fmt.Sprintf("%s = VLESS,%s,%d,\"%s\",transport=%s,over-tls=%t,udp=%t",
				v.Name, v.Server, v.Port, v.Uuid, transport, tls, sqlconfig.Udp)
			// Reality / XTLS Vision
			if v.Query.Flow != "" {
				p += fmt.Sprintf(",flow=%s", v.Query.Flow)
			}
			if v.Query.Pbk != "" {
				p += fmt.Sprintf(",public-key=\"%s\"", v.Query.Pbk)
			}
			if v.Query.Sid != "" {
				p += fmt.Sprintf(",short-id=%s", v.Query.Sid)
			}
			if transport == "ws" {
				p += fmt.Sprintf(",path=%s", v.Query.Path)
				if v.Query.Host != "" {
					p += fmt.Sprintf(",host=%s", v.Query.Host)
				}
			}
			if v.Query.Sni != "" {
				p += fmt.Sprintf(",sni=%s", v.Query.Sni)
			}
			if sqlconfig.Cert {
				p += ",skip-cert-verify=true"
			}
			proxys = append(proxys, p)
		case scheme == "trojan":
			t, err := DecodeTrojanURL(link)
			if err != nil {
				log.Println(err)
				continue
			}
			p := fmt.Sprintf("%s = trojan,%s,%d,\"%s\",udp=%t",
				t.Name, t.Hostname, t.Port, t.Password, sqlconfig.Udp)
			if t.Query.Type != "" && t.Query.Type != "tcp" {
				p += fmt.Sprintf(",transport=%s", t.Query.Type)
				if t.Query.Path != "" {
					p += fmt.Sprintf(",path=%s", t.Query.Path)
				}
				if t.Query.Host != "" {
					p += fmt.Sprintf(",host=%s", t.Query.Host)
				}
			}
			if t.Query.Sni != "" {
				p += fmt.Sprintf(",sni=%s", t.Query.Sni)
			}
			if sqlconfig.Cert {
				p += ",skip-cert-verify=true"
			}
			proxys = append(proxys, p)
		case scheme == "hysteria2" || scheme == "hy2":
			h, err := DecodeHY2URL(link)
			if err != nil {
				log.Println(err)
				continue
			}
			p := fmt.Sprintf("%s = Hysteria2,%s,%d,\"%s\",udp=%t",
				h.Name, h.Host, h.Port, h.Password, sqlconfig.Udp)
			if h.Sni != "" {
				p += fmt.Sprintf(",sni=%s", h.Sni)
			}
			if h.Obfs != "" {
				p += fmt.Sprintf(",salamander-password=\"%s\"", h.ObfsPassword)
			}
			if sqlconfig.Cert {
				p += ",skip-cert-verify=true"
			}
			proxys = append(proxys, p)
		// Loon 不支持 tuic，跳过
		case scheme == "tuic":
			continue
		}
	}
	return DecodeLoon(proxys, sqlconfig.Loon)
}

// DecodeLoon 读取 Loon 模板（本地文件或 URL），将 [Proxy] 段替换为生成的节点。
func DecodeLoon(proxys []string, file string) (string, error) {
	var raw []byte
	var err error
	if strings.Contains(file, "://") {
		resp, err := http.Get(file)
		if err != nil {
			log.Println("http.Get error", err)
			return "", err
		}
		defer resp.Body.Close()
		raw, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
	} else {
		raw, err = os.ReadFile(file)
		if err != nil {
			return "", err
		}
	}

	proxyReg := regexp.MustCompile(`(?s)\[Proxy\](.*?)\[`)
	out := proxyReg.ReplaceAllStringFunc(string(raw), func(s string) string {
		text := strings.Join(proxys, "\n")
		return "[Proxy]\n" + text + s[len("[Proxy]"):]
	})
	return out, nil
}