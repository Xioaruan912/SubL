package api

import (
	"sync"
	"time"

	"ppeelink/models"
	"ppeelink/node"

	"github.com/gin-gonic/gin"
)

// NodeOverviewItem 节点概览（含国家与延迟）
type NodeOverviewItem struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Link        string   `json:"link"`
	Server      string   `json:"server"`
	Country     string   `json:"country"`
	CountryCode string   `json:"countryCode"`
	Rtt         int      `json:"rtt"` // 毫秒，-1 失败
	Groups      []string `json:"groups"`
}

// overview 缓存（60s）
type overviewCache struct {
	mu      sync.Mutex
	data    []NodeOverviewItem
	expires time.Time
}

var oCache = &overviewCache{}

// NodeOverview 返回所有节点概览：国家（GeoIP）+ 服务器 TCP 延迟 + 分组。
// GET /api/v1/nodes/overview
func NodeOverview(c *gin.Context) {
	oCache.mu.Lock()
	if oCache.data != nil && time.Now().Before(oCache.expires) {
		data := oCache.data
		oCache.mu.Unlock()
		c.JSON(200, gin.H{"code": "00000", "data": data, "msg": "节点概览（缓存）"})
		return
	}
	oCache.mu.Unlock()

	nodes, err := models.GetNodeList()
	if err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "获取节点列表失败"})
		return
	}

	// 逐节点解析服务器地址 + GeoIP 国家 + TCP 延迟
	items := make([]NodeOverviewItem, 0, len(nodes))
	var wg sync.WaitGroup
	results := make([]NodeOverviewItem, len(nodes))
	for i, n := range nodes {
		wg.Add(1)
		go func(i int, n models.Node) {
			defer wg.Done()
			item := NodeOverviewItem{
				ID: n.ID, Name: n.Name, Link: n.Link,
				Rtt:    -1,
				Groups: make([]string, 0),
			}
			for _, g := range n.GroupNodes {
				item.Groups = append(item.Groups, g.Name)
			}
			host, port := node.ExtractServerHost(n.Link)
			if host != "" {
				item.Server = host
				item.CountryCode = node.LookupCountry(host)
				item.Country = countryName(item.CountryCode)
				addr := host
				if port > 0 {
					addr = host + ":" + itoa(port)
				}
				item.Rtt = node.TCPPing(addr, 3*time.Second)
			}
			results[i] = item
		}(i, n)
	}
	wg.Wait()
	items = results

	oCache.mu.Lock()
	oCache.data = items
	oCache.expires = time.Now().Add(60 * time.Second)
	oCache.mu.Unlock()

	c.JSON(200, gin.H{"code": "00000", "data": items, "msg": "节点概览"})
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