package api

import (
	"strconv"
	"sync"
	"time"

	"ppeelink/models"
	"ppeelink/node"

	"github.com/gin-gonic/gin"
)

// NodeOverviewItem 节点概览（含国家与延迟）
type NodeOverviewItem struct {
	ID                  int       `json:"id"`
	Name                string    `json:"name"`
	Link                string    `json:"link"`
	Server              string    `json:"server"`
	Port                int       `json:"port"`
	Country             string    `json:"country"`
	CountryCode         string    `json:"countryCode"`
	Rtt                 int       `json:"rtt"` // 毫秒，-1 失败
	Groups              []string  `json:"groups"`
	Score               int       `json:"score"`
	Availability        float64   `json:"availability"`
	AverageRtt          int       `json:"averageRtt"`
	Jitter              int       `json:"jitter"`
	P95Rtt              int       `json:"p95Rtt"`
	ConsecutiveFailures int       `json:"consecutiveFailures"`
	Confidence          int       `json:"confidence"`
	SampleCount         int       `json:"sampleCount"`
	LastTestedAt        time.Time `json:"lastTestedAt"`
}

// overview 缓存（60s）
type overviewCache struct {
	mu      sync.Mutex
	data    []NodeOverviewItem
	expires time.Time
}

var oCache = &overviewCache{}

// InvalidateOverview 节点数据变更后调用，使概览缓存立即失效
func InvalidateOverview() {
	oCache.mu.Lock()
	oCache.data = nil
	oCache.expires = time.Time{}
	oCache.mu.Unlock()
}

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

	items, err := CollectNodeQuality()
	if err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "获取节点列表失败"})
		return
	}

	oCache.mu.Lock()
	oCache.data = items
	oCache.expires = time.Now().Add(60 * time.Second)
	oCache.mu.Unlock()

	c.JSON(200, gin.H{"code": "00000", "data": items, "msg": "节点概览"})
}

// CollectNodeQuality checks every node, persists the result and returns the
// enriched overview. It is shared by the HTTP endpoint and the scheduler.
func CollectNodeQuality() ([]NodeOverviewItem, error) {
	nodes, err := models.GetNodeList()
	if err != nil {
		return nil, err
	}

	// 逐节点解析服务器地址 + GeoIP 国家 + TCP 延迟
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
				item.Port = port
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
	checkedAt := time.Now()
	samples := make([]models.NodeQualitySample, 0, len(results))
	for _, item := range results {
		samples = append(samples, models.NodeQualitySample{
			NodeID: item.ID, Rtt: item.Rtt, Success: item.Rtt >= 0, CheckedAt: checkedAt,
		})
	}
	if err := models.RecordNodeQuality(samples); err != nil {
		return nil, err
	}
	processNodeHealthEvents(results)
	stats, err := models.GetNodeQualityStats(time.Now().Add(-24 * time.Hour))
	if err != nil {
		return nil, err
	}
	for i := range results {
		if stat, ok := stats[results[i].ID]; ok {
			results[i].Score = stat.Score
			results[i].Availability = stat.Availability
			results[i].AverageRtt = stat.AverageRtt
			results[i].Jitter = stat.Jitter
			results[i].P95Rtt = stat.P95Rtt
			results[i].ConsecutiveFailures = stat.ConsecutiveFailures
			results[i].Confidence = stat.Confidence
			results[i].SampleCount = stat.SampleCount
			results[i].LastTestedAt = stat.LastTestedAt
		}
	}
	return results, nil
}

// NodeQualityHistory returns recent raw samples for a node, oldest first.
func NodeQualityHistory(c *gin.Context) {
	nodeID, err := strconv.Atoi(c.Query("id"))
	if err != nil || nodeID <= 0 {
		c.JSON(400, gin.H{"code": "40000", "msg": "节点 id 格式错误"})
		return
	}
	hours := 24
	if value, parseErr := strconv.Atoi(c.DefaultQuery("hours", "24")); parseErr == nil && value >= 1 && value <= 720 {
		hours = value
	}
	var samples []models.NodeQualitySample
	err = models.DB.Where("node_id = ? AND checked_at >= ?", nodeID, time.Now().Add(-time.Duration(hours)*time.Hour)).
		Order("checked_at ASC").Limit(5000).Find(&samples).Error
	if err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "读取质量历史失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": samples, "msg": "节点质量历史"})
}

// NodeQualitySummary provides actionable dashboard numbers.
func NodeQualitySummary(c *gin.Context) {
	stats, err := models.GetNodeQualityStats(time.Now().Add(-24 * time.Hour))
	if err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "读取质量汇总失败"})
		return
	}
	total, healthy, warning, offline, scoreSum := len(stats), 0, 0, 0, 0
	for _, stat := range stats {
		scoreSum += stat.Score
		switch {
		case stat.LastRtt < 0:
			offline++
		case stat.Score >= 80:
			healthy++
		default:
			warning++
		}
	}
	averageScore := 0
	if total > 0 {
		averageScore = scoreSum / total
	}
	c.JSON(200, gin.H{"code": "00000", "data": gin.H{
		"total": total, "healthy": healthy, "warning": warning,
		"offline": offline, "averageScore": averageScore,
	}, "msg": "节点质量汇总"})
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
