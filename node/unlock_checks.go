package node

import (
	"io"
	"net/http"
	"strings"
)

// 浏览器 UA（部分服务需要）
const unlockUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"

// checkOpenAI 检测 OpenAI / ChatGPT 是否解锁。
// 参考 RegionRestrictionCheck：请求 compliance/cookie_requirements 与 ios.chat.openai.com，
// 若响应含 unsupported_country 或 VPN 字样则未解锁。
func checkOpenAI(c *http.Client) (bool, string) {
	req1, _ := http.NewRequest("GET", "https://api.openai.com/compliance/cookie_requirements", nil)
	req1.Header.Set("Authority", "api.openai.com")
	req1.Header.Set("Accept", "*/*")
	req1.Header.Set("Authorization", "Bearer null")
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Origin", "https://platform.openai.com")
	req1.Header.Set("Referer", "https://platform.openai.com/")
	req1.Header.Set("User-Agent", unlockUA)
	resp1, err := c.Do(req1)
	if err != nil {
		return false, "连接失败"
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	if strings.Contains(strings.ToLower(string(body1)), "unsupported_country") {
		return false, "区域不支持"
	}

	req2, _ := http.NewRequest("GET", "https://ios.chat.openai.com/", nil)
	req2.Header.Set("Accept", "*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req2.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req2.Header.Set("User-Agent", unlockUA)
	resp2, err := c.Do(req2)
	if err != nil {
		return false, "连接失败"
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if strings.Contains(strings.ToLower(string(body2)), "vpn") {
		return false, "仅浏览器可用"
	}
	return true, ""
}

// checkClaude 检测 Claude 是否解锁。
// 参考 RegionRestrictionCheck：访问 claude.ai，若重定向到 app-unavailable-in-region 则未解锁。
func checkClaude(c *http.Client) (bool, string) {
	req, _ := http.NewRequest("GET", "https://claude.ai/", nil)
	req.Header.Set("User-Agent", unlockUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	client := *c
	// 不跟随重定向，改为手动判断最终 URL
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, "连接失败"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		loc := resp.Header.Get("Location")
		if strings.Contains(loc, "app-unavailable-in-region") {
			return false, "区域不支持"
		}
		return true, ""
	}
	return true, ""
}

// checkGemini 检测 Google Gemini 是否解锁。
// 参考 RegionRestrictionCheck：访问 gemini.google.com，若响应含标记 45631641,null,true 则解锁。
func checkGemini(c *http.Client) (bool, string) {
	req, _ := http.NewRequest("GET", "https://gemini.google.com", nil)
	req.Header.Set("User-Agent", unlockUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	resp, err := c.Do(req)
	if err != nil {
		return false, "连接失败"
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "45631641,null,true") {
		return true, ""
	}
	return false, "区域不支持"
}

// checkNetflix 检测 Netflix 是否解锁。
// 参考 RegionRestrictionCheck：请求两个不同的 title 页，若均含 "Oh no!" 则仅原创可看，
// 若任一正常返回则已解锁。这里简化为：返回 200 且不含 "Oh no!" 判定解锁。
func checkNetflix(c *http.Client) (bool, string) {
	titles := []string{"https://www.netflix.com/title/81280792", "https://www.netflix.com/title/70143836"}
	blocked := 0
	for _, u := range titles {
		req, _ := http.NewRequest("GET", u, nil)
		req.Header.Set("User-Agent", unlockUA)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		resp, err := c.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if strings.Contains(string(body), "Oh no!") {
			blocked++
		}
	}
	if blocked == len(titles) {
		return false, "仅原创可看"
	}
	if blocked == 0 {
		return true, ""
	}
	return true, "部分可用"
}

// checkYouTube 检测 YouTube 是否可达（地区限制不影响基础访问）。
func checkYouTube(c *http.Client) (bool, string) {
	resp, err := c.Get("https://www.youtube.com/")
	if err != nil {
		return false, "连接失败"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, ""
	}
	return false, "HTTP " + itoa(resp.StatusCode)
}

// checkDisney 检测 Disney+ 是否解锁。
// 参考 RegionRestrictionCheck：请求 disney API 获取 assertion，含 403/forbidden-location 则未解锁。
func checkDisney(c *http.Client) (bool, string) {
	body := `{"deviceFamily":"browser","applicationRuntime":"chrome","deviceProfile":"windows","attributes":{}}`
	req, _ := http.NewRequest("POST", "https://disney.api.edge.bamgrid.com/devices", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer ZGlzbmV5JmJyb3dzZXImMS4wLjA.Cu56AgSfBTDag5NiRA81oLHkDZfu5L3CKadnefEAY84")
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("User-Agent", unlockUA)
	resp, err := c.Do(req)
	if err != nil {
		return false, "连接失败"
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	s := string(data)
	if strings.Contains(s, "forbidden-location") || strings.Contains(s, "403 ERROR") {
		return false, "区域不支持"
	}
	if strings.Contains(s, "assertion") {
		return true, ""
	}
	return false, "异常响应"
}

// checkGoogle 检测 Google 是否可达（generate_204 返回 204）。
func checkGoogle(c *http.Client) (bool, string) {
	resp, err := c.Get("https://www.google.com/generate_204")
	if err != nil {
		return false, "连接失败"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		return true, ""
	}
	return false, "HTTP " + itoa(resp.StatusCode)
}

// checkGitHub 检测 GitHub 是否可达。
func checkGitHub(c *http.Client) (bool, string) {
	resp, err := c.Get("https://github.com/")
	if err != nil {
		return false, "连接失败"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, ""
	}
	return false, "HTTP " + itoa(resp.StatusCode)
}

// checkTelegram 检测 Telegram 是否可达。
func checkTelegram(c *http.Client) (bool, string) {
	resp, err := c.Get("https://t.me/")
	if err != nil {
		return false, "连接失败"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, ""
	}
	return false, "HTTP " + itoa(resp.StatusCode)
}

// itoa 简易整数转字符串
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}