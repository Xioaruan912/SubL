package api

import (
	"strconv"
	"strings"
	"sync"
	"time"
	"sublink/models"
	"sublink/node"

	"github.com/gin-gonic/gin"
)

// MapPoint 地图打点数据
type MapPoint struct {
	Name        string  `json:"name"`
	Server      string  `json:"server"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
}

// NodeMap 返回所有节点的国家与坐标，用于首页世界地图打点。
func NodeMap(c *gin.Context) {
	nodes, err := models.GetNodeList()
	if err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "获取节点列表失败"})
		return
	}
	points := make([]MapPoint, 0, len(nodes))
	for _, n := range nodes {
		host, _ := node.ExtractServerHost(n.Link)
		if host == "" {
			continue
		}
		countryCode := node.LookupCountry(host)
		coord := node.CountryCoord(countryCode)
		points = append(points, MapPoint{
			Name:        n.Name,
			Server:      host,
			Country:     countryName(countryCode),
			CountryCode: countryCode,
			Lat:         coord[0],
			Lng:         coord[1],
		})
	}
	c.JSON(200, gin.H{"code": "00000", "data": points, "msg": "节点地图"})
}

var countryNames = map[string]string{
	"US": "美国", "JP": "日本", "HK": "香港", "SG": "新加坡",
	"KR": "韩国", "TW": "台湾", "CN": "中国大陆", "DE": "德国",
	"GB": "英国", "FR": "法国", "NL": "荷兰", "RU": "俄罗斯",
	"CA": "加拿大", "AU": "澳大利亚", "IN": "印度", "BR": "巴西",
	"IT": "意大利", "ES": "西班牙", "SE": "瑞典", "FI": "芬兰",
	"NO": "挪威", "DK": "丹麦", "CH": "瑞士", "AT": "奥地利",
	"BE": "比利时", "IE": "爱尔兰", "PL": "波兰", "CZ": "捷克",
	"TR": "土耳其", "AE": "阿联酋", "ID": "印尼", "MY": "马来西亚",
	"TH": "泰国", "VN": "越南", "PH": "菲律宾", "NZ": "新西兰",
	"MX": "墨西哥", "AR": "阿根廷", "ZA": "南非", "EG": "埃及",
	"UA": "乌克兰", "BG": "保加利亚", "GR": "希腊", "PT": "葡萄牙",
	"IL": "以色列", "SA": "沙特", "KZ": "哈萨克", "UZ": "乌兹别克",
	"CO": "哥伦比亚", "CL": "智利", "PE": "秘鲁", "RO": "罗马尼亚",
	"HU": "匈牙利", "LT": "立陶宛", "EE": "爱沙尼亚", "LV": "拉脱维亚",
	"IS": "冰岛", "LU": "卢森堡", "CY": "塞浦路斯", "MT": "马耳他",
}

func countryName(code string) string {
	if n, ok := countryNames[code]; ok {
		return n
	}
	return code
}

// 常见测速目标
var commonTargets = []node.PingTarget{
	{Name: "GitHub", Addr: "github.com:443"},
	{Name: "Google", Addr: "google.com:443"},
	{Name: "Cloudflare", Addr: "cloudflare.com:443"},
	{Name: "Bing", Addr: "www.bing.com:443"},
	{Name: "百度", Addr: "baidu.com:443"},
}

// ping 缓存（60s），避免频繁请求打爆网络
type pingCache struct {
	mu      sync.Mutex
	data    *PingData
	expires time.Time
}

var pCache = &pingCache{}

// PingData 节点延迟响应
type PingData struct {
	Targets []node.PingTarget `json:"targets"`
	Nodes   []NodePingItem    `json:"nodes"`
}

// NodePingItem 单节点延迟
type NodePingItem struct {
	Name   string `json:"name"`
	Server string `json:"server"`
	Rtt    int    `json:"rtt"` // 毫秒，-1 表示失败
}

// NodePing 返回常见目标延迟与每个节点服务器延迟。
func NodePing(c *gin.Context) {
	if data, ok := pCache.get(); ok {
		c.JSON(200, gin.H{"code": "00000", "data": data, "msg": "节点延迟"})
		return
	}
	// 常见目标测速
	results := node.MultiTCPPing(commonTargets, 3*time.Second)
	targets := make([]node.PingTarget, 0, len(results))
	for _, r := range results {
		targets = append(targets, r.Target)
	}

	// 节点服务器测速
	nodes, err := models.GetNodeList()
	if err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "获取节点列表失败"})
		return
	}
	var nodeAddrs []node.PingTarget
	for _, n := range nodes {
		host, port := node.ExtractServerHost(n.Link)
		if host == "" {
			continue
		}
		addr := host
		if port > 0 {
			addr = host + ":" + strconv.Itoa(port)
		}
		nodeAddrs = append(nodeAddrs, node.PingTarget{Name: n.Name, Addr: addr})
	}
	nResults := node.MultiTCPPing(nodeAddrs, 3*time.Second)
	items := make([]NodePingItem, 0, len(nResults))
	for _, r := range nResults {
		items = append(items, NodePingItem{
			Name:   r.Target.Name,
			Server: r.Target.Addr,
			Rtt:    r.Target.Rtt,
		})
	}

	data := &PingData{Targets: targets, Nodes: items}
	pCache.set(data)
	c.JSON(200, gin.H{"code": "00000", "data": data, "msg": "节点延迟"})
}

func (p *pingCache) get() (*PingData, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.data != nil && time.Now().Before(p.expires) {
		return p.data, true
	}
	return nil, false
}

func (p *pingCache) set(d *PingData) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.data = d
	p.expires = time.Now().Add(60 * time.Second)
}

// unlock 缓存：按节点链接缓存解锁测试结果
type unlockCache struct {
	mu      sync.Mutex
	entries map[string]*unlockCacheEntry
}

type unlockCacheEntry struct {
	result  *node.UnlockResult
	expires time.Time
}

var uCache = &unlockCache{entries: make(map[string]*unlockCacheEntry)}

// NodeUnlock 对指定节点做真实解锁测试。
// POST /api/v1/nodes/unlock  body: id 或 link
func NodeUnlock(c *gin.Context) {
	idStr := c.PostForm("id")
	link := c.PostForm("link")
	if idStr == "" && link == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "需要提供节点 id 或 link"})
		return
	}
	// 通过 id 查找节点
	if idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(400, gin.H{"code": "40000", "msg": "节点 id 格式错误"})
			return
		}
		var n models.Node
		n.ID = id
		if err := n.Find(); err != nil {
			c.JSON(404, gin.H{"code": "40400", "msg": "节点不存在"})
			return
		}
		link = n.Link
	}
	if link == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "节点链接为空"})
		return
	}

	// 缓存（60s）
	uCache.mu.Lock()
	if e, ok := uCache.entries[link]; ok && time.Now().Before(e.expires) {
		uCache.mu.Unlock()
		c.JSON(200, gin.H{"code": "00000", "data": e.result, "msg": "解锁测试（缓存）"})
		return
	}
	uCache.mu.Unlock()

	// 全局互斥：同时只跑一个解锁测试，避免大量 sing-box 实例
	if node.UnlockTestBusy() {
		c.JSON(429, gin.H{"code": "42900", "msg": "已有解锁测试进行中，请稍候"})
		return
	}
	defer node.UnlockTestRelease()

	result, err := node.RunUnlockTest(node.UnlockTestConfig{Link: link, Timeout: 8 * time.Second})
	if err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "解锁测试失败: " + err.Error()})
		return
	}

	uCache.mu.Lock()
	uCache.entries[link] = &unlockCacheEntry{result: result, expires: time.Now().Add(60 * time.Second)}
	uCache.mu.Unlock()

	c.JSON(200, gin.H{"code": "00000", "data": result, "msg": "解锁测试完成"})
}

// chinaCache 中国延迟缓存
type chinaCache struct {
	mu      sync.Mutex
	entries map[string]*chinaCacheEntry
}

type chinaCacheEntry struct {
	result  *node.ChinaPingResult
	expires time.Time
}

var cCache = &chinaCache{entries: make(map[string]*chinaCacheEntry)}

// NodeChinaPing 对指定节点做中国各地 TCP 延迟测试。
// POST /api/v1/nodes/chinaping body: id/link + 可选 provinces,isps,zstatic_port
func NodeChinaPing(c *gin.Context) {
	idStr := c.PostForm("id")
	link := c.PostForm("link")
	if idStr == "" && link == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "需要提供节点 id 或 link"})
		return
	}
	if idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(400, gin.H{"code": "40000", "msg": "节点 id 格式错误"})
			return
		}
		var n models.Node
		n.ID = id
		if err := n.Find(); err != nil {
			c.JSON(404, gin.H{"code": "40400", "msg": "节点不存在"})
			return
		}
		link = n.Link
	}
	if link == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "节点链接为空"})
		return
	}

	// 筛选参数
	var provinces, isps []string
	if v := c.PostForm("provinces"); v != "" {
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				provinces = append(provinces, s)
			}
		}
	}
	if v := c.PostForm("isps"); v != "" {
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				isps = append(isps, s)
			}
		}
	}
	var zstaticPorts []int
	if v := c.PostForm("zstatic_port"); v != "" {
		for _, s := range strings.Split(v, ",") {
			if p, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				zstaticPorts = append(zstaticPorts, p)
			}
		}
	}

	// 缓存 key（含筛选条件）
	cacheKey := link + "|" + strings.Join(provinces, ",") + "|" + strings.Join(isps, ",") + "|" + strings.Join(strings.Split(c.PostForm("zstatic_port"), ","), ",")
	cCache.mu.Lock()
	if e, ok := cCache.entries[cacheKey]; ok && time.Now().Before(e.expires) {
		cCache.mu.Unlock()
		c.JSON(200, gin.H{"code": "00000", "data": e.result, "msg": "中国延迟（缓存）"})
		return
	}
	cCache.mu.Unlock()

	if node.UnlockTestBusy() {
		c.JSON(429, gin.H{"code": "42900", "msg": "已有测试进行中，请稍候"})
		return
	}
	defer node.UnlockTestRelease()

	result, err := node.RunChinaPing(node.UnlockTestConfig{Link: link, Timeout: 5 * time.Second}, provinces, isps, zstaticPorts)
	if err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "中国延迟测试失败: " + err.Error()})
		return
	}

	cCache.mu.Lock()
	cCache.entries[cacheKey] = &chinaCacheEntry{result: result, expires: time.Now().Add(60 * time.Second)}
	cCache.mu.Unlock()

	c.JSON(200, gin.H{"code": "00000", "data": result, "msg": "中国延迟测试完成"})
}