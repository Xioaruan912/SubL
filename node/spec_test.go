package node

import "testing"

func TestOutboundRoundTrip(t *testing.T) {
	links := []string{
		EncodeSSURL(Ss{Param: Param{Cipher: "aes-256-gcm", Password: "pw"}, Server: "1.2.3.4", Port: 8388, Name: "ss-node"}),
		EncodeVmessURL(Vmess{Add: "5.6.7.8", Port: "443", Id: "11111111-2222-3333-4444-555555555555", Net: "tcp", Ps: "vmess-node", Scy: "auto"}),
		EncodeVLESSURL(VLESS{Name: "vless-node", Uuid: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", Server: "9.9.9.9", Port: 8443, Query: VLESSQuery{Security: "tls", Sni: "x.com", Type: "tcp", Encryption: "none"}}),
		EncodeTrojanURL(Trojan{Password: "tpw", Hostname: "2.2.2.2", Port: 443, Name: "trojan-node", Type: "trojan"}),
		EncodeHY2URL(HY2{Password: "hpw", Host: "3.3.3.3", Port: 443, Name: "hy2-node", Sni: "b.com"}),
		EncodeTuicURL(Tuic{Name: "tuic-node", Uuid: "99999999-8888-7777-6666-555555555555", Password: "tupw", Host: "4.4.4.4", Port: 443}),
	}
	wantServer := map[string]string{
		"ss-node": "1.2.3.4", "vmess-node": "5.6.7.8", "vless-node": "9.9.9.9",
		"trojan-node": "2.2.2.2", "hy2-node": "3.3.3.3", "tuic-node": "4.4.4.4",
	}
	for _, link := range links {
		p, err := ParseOutbound(link)
		if err != nil {
			t.Fatalf("ParseOutbound(%s): %v", link, err)
		}
		if want, ok := wantServer[p.Name]; !ok || p.Server != want {
			t.Fatalf("unexpected parse: name=%q server=%q want=%q", p.Name, p.Server, want)
		}
		if p.Port == 0 {
			t.Fatalf("port not parsed for %s", p.Name)
		}
		reparsed, err := ParseOutbound(p.ToLink())
		if err != nil {
			t.Fatalf("re-parse %s: %v", p.Name, err)
		}
		if reparsed.Server != p.Server || reparsed.Name != p.Name {
			t.Fatalf("round trip changed node: %#v -> %#v", p, reparsed)
		}
	}
}

func TestOutboundWithName(t *testing.T) {
	p, err := ParseOutbound(EncodeSSURL(Ss{Param: Param{Cipher: "aes-256-gcm", Password: "pw"}, Server: "1.2.3.4", Port: 8388, Name: "old"}))
	if err != nil {
		t.Fatal(err)
	}
	renamed := p.WithName("new-name")
	if renamed.Name != "new-name" {
		t.Fatalf("name not updated: %q", renamed.Name)
	}
	reparsed, err := ParseOutbound(renamed.Link)
	if err != nil {
		t.Fatal(err)
	}
	if reparsed.Name != "new-name" {
		t.Fatalf("link fragment not updated: %q", reparsed.Name)
	}
}

func TestFlagEmojiAndCountryName(t *testing.T) {
	if got := FlagEmoji("HK"); got != "🇭🇰" {
		t.Fatalf("FlagEmoji(HK) = %q", got)
	}
	if got := FlagEmoji("us"); got != "🇺🇸" {
		t.Fatalf("FlagEmoji(us) = %q", got)
	}
	if FlagEmoji("XYZ") != "" {
		t.Fatal("expected empty flag for invalid code")
	}
	cases := map[string]string{"香港 01": "HK", "日本 IEPL": "JP", "新加坡": "SG", "US-01": "US", "未知": ""}
	for name, want := range cases {
		if got := CountryFromName(name); got != want {
			t.Fatalf("CountryFromName(%q)=%q want %q", name, got, want)
		}
	}
}

func TestResolveDomainIPPassthrough(t *testing.T) {
	if ip, ok := ResolveDomain("1.2.3.4"); ok || ip != "1.2.3.4" {
		t.Fatalf("IP should pass through, got %q ok=%v", ip, ok)
	}
}
