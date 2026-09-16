package node

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// parseIntList 解析逗号分隔的整数列表（如 wireguard reserved）。
func parseIntList(s string) []int {
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			out = append(out, n)
		}
	}
	return out
}

/* ---------------- AnyTLS ---------------- */

type AnyTLS struct {
	Password string
	Host     string
	Port     int
	Sni      string
	Insecure int
	Fp       string
	Name     string
}

func EncodeAnyTLSURL(a AnyTLS) string {
	if a.Name == "" {
		a.Name = fmt.Sprintf("%s:%d", a.Host, a.Port)
	}
	u := url.URL{
		Scheme:   "anytls",
		User:     url.User(a.Password),
		Host:     fmt.Sprintf("%s:%d", a.Host, a.Port),
		Fragment: a.Name,
	}
	q := u.Query()
	q.Set("sni", a.Sni)
	q.Set("fp", a.Fp)
	if a.Insecure != 0 {
		q.Set("insecure", strconv.Itoa(a.Insecure))
	}
	for k, v := range q {
		if v[0] == "" {
			delete(q, k)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func DecodeAnyTLSURL(s string) (AnyTLS, error) {
	u, err := url.Parse(s)
	if err != nil {
		return AnyTLS{}, err
	}
	if u.Scheme != "anytls" {
		return AnyTLS{}, fmt.Errorf("非 anytls 协议: %s", s)
	}
	port, _ := strconv.Atoi(u.Port())
	insecure, _ := strconv.Atoi(u.Query().Get("insecure"))
	name := u.Fragment
	if name == "" {
		name = u.Hostname() + ":" + u.Port()
	}
	return AnyTLS{
		Password: u.User.Username(), Host: u.Hostname(), Port: port,
		Sni: u.Query().Get("sni"), Insecure: insecure, Fp: u.Query().Get("fp"), Name: name,
	}, nil
}

/* ---------------- Snell ---------------- */

type Snell struct {
	Psk      string
	Host     string
	Port     int
	Version  string
	Obfs     string
	ObfsHost string
	Name     string
}

func EncodeSnellURL(s Snell) string {
	if s.Name == "" {
		s.Name = fmt.Sprintf("%s:%d", s.Host, s.Port)
	}
	u := url.URL{
		Scheme:   "snell",
		User:     url.User(s.Psk),
		Host:     fmt.Sprintf("%s:%d", s.Host, s.Port),
		Fragment: s.Name,
	}
	q := u.Query()
	q.Set("version", s.Version)
	q.Set("obfs", s.Obfs)
	q.Set("obfs-host", s.ObfsHost)
	for k, v := range q {
		if v[0] == "" {
			delete(q, k)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func DecodeSnellURL(s string) (Snell, error) {
	u, err := url.Parse(s)
	if err != nil {
		return Snell{}, err
	}
	if u.Scheme != "snell" {
		return Snell{}, fmt.Errorf("非 snell 协议: %s", s)
	}
	port, _ := strconv.Atoi(u.Port())
	name := u.Fragment
	if name == "" {
		name = u.Hostname() + ":" + u.Port()
	}
	return Snell{
		Psk: u.User.Username(), Host: u.Hostname(), Port: port,
		Version: u.Query().Get("version"), Obfs: u.Query().Get("obfs"),
		ObfsHost: u.Query().Get("obfs-host"), Name: name,
	}, nil
}

/* ---------------- SOCKS5 / HTTP ---------------- */

type Socks struct {
	Type     string // socks5 | http
	Username string
	Password string
	Host     string
	Port     int
	Tls      bool
	Name     string
}

func EncodeSocksURL(s Socks) string {
	if s.Type == "" {
		s.Type = "socks5"
	}
	if s.Name == "" {
		s.Name = fmt.Sprintf("%s:%d", s.Host, s.Port)
	}
	var user *url.Userinfo
	if s.Username != "" {
		if s.Password != "" {
			user = url.UserPassword(s.Username, s.Password)
		} else {
			user = url.User(s.Username)
		}
	}
	u := url.URL{
		Scheme:   s.Type,
		User:     user,
		Host:     fmt.Sprintf("%s:%d", s.Host, s.Port),
		Fragment: s.Name,
	}
	if s.Tls && s.Type == "http" {
		u.Scheme = "https"
	}
	return u.String()
}

func DecodeSocksURL(s string) (Socks, error) {
	u, err := url.Parse(s)
	if err != nil {
		return Socks{}, err
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "socks5" && scheme != "socks5h" && scheme != "http" && scheme != "https" {
		return Socks{}, fmt.Errorf("非 socks/http 协议: %s", s)
	}
	port, _ := strconv.Atoi(u.Port())
	name := u.Fragment
	if name == "" {
		name = u.Hostname() + ":" + u.Port()
	}
	password, _ := u.User.Password()
	typ := "socks5"
	if scheme == "http" || scheme == "https" {
		typ = "http"
	}
	return Socks{
		Type: typ, Username: u.User.Username(), Password: password,
		Host: u.Hostname(), Port: port, Tls: scheme == "https", Name: name,
	}, nil
}

/* ---------------- SSH ---------------- */

type SSH struct {
	User     string
	Password string
	Host     string
	Port     int
	Name     string
}

func EncodeSSHURL(s SSH) string {
	if s.Name == "" {
		s.Name = fmt.Sprintf("%s:%d", s.Host, s.Port)
	}
	var user *url.Userinfo
	if s.User != "" {
		if s.Password != "" {
			user = url.UserPassword(s.User, s.Password)
		} else {
			user = url.User(s.User)
		}
	}
	u := url.URL{
		Scheme:   "ssh",
		User:     user,
		Host:     fmt.Sprintf("%s:%d", s.Host, s.Port),
		Fragment: s.Name,
	}
	return u.String()
}

func DecodeSSHURL(s string) (SSH, error) {
	u, err := url.Parse(s)
	if err != nil {
		return SSH{}, err
	}
	if u.Scheme != "ssh" {
		return SSH{}, fmt.Errorf("非 ssh 协议: %s", s)
	}
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 22
	}
	name := u.Fragment
	if name == "" {
		name = u.Hostname() + ":" + strconv.Itoa(port)
	}
	password, _ := u.User.Password()
	return SSH{User: u.User.Username(), Password: password, Host: u.Hostname(), Port: port, Name: name}, nil
}

/* ---------------- WireGuard ---------------- */

type WireGuard struct {
	PrivateKey string
	PublicKey  string
	Host       string
	Port       int
	Address    string
	Mtu        int
	Reserved   string
	Name       string
}

func EncodeWireGuardURL(w WireGuard) string {
	if w.Name == "" {
		w.Name = fmt.Sprintf("%s:%d", w.Host, w.Port)
	}
	u := url.URL{
		Scheme:   "wireguard",
		User:     url.User(w.PrivateKey),
		Host:     fmt.Sprintf("%s:%d", w.Host, w.Port),
		Fragment: w.Name,
	}
	q := u.Query()
	q.Set("publickey", w.PublicKey)
	q.Set("address", w.Address)
	if w.Mtu != 0 {
		q.Set("mtu", strconv.Itoa(w.Mtu))
	}
	q.Set("reserved", w.Reserved)
	for k, v := range q {
		if v[0] == "" {
			delete(q, k)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func DecodeWireGuardURL(s string) (WireGuard, error) {
	u, err := url.Parse(s)
	if err != nil {
		return WireGuard{}, err
	}
	if u.Scheme != "wireguard" && u.Scheme != "wg" {
		return WireGuard{}, fmt.Errorf("非 wireguard 协议: %s", s)
	}
	port, _ := strconv.Atoi(u.Port())
	mtu, _ := strconv.Atoi(u.Query().Get("mtu"))
	name := u.Fragment
	if name == "" {
		name = u.Hostname() + ":" + u.Port()
	}
	publicKey := u.Query().Get("publickey")
	if publicKey == "" {
		publicKey = u.Query().Get("public-key")
	}
	return WireGuard{
		PrivateKey: u.User.Username(), PublicKey: publicKey, Host: u.Hostname(), Port: port,
		Address: u.Query().Get("address"), Mtu: mtu, Reserved: u.Query().Get("reserved"), Name: name,
	}, nil
}
