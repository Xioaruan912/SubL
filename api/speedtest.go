package api

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"ppeelink/models"
	"ppeelink/node"

	"github.com/gin-gonic/gin"
)

const defaultSpeedTestTarget = "https://speed.cloudflare.com/__down?bytes=3000000"

// NodeSpeedTestStream 真实出口测速（SSE）：逐节点推送 TCP 延迟与经节点下载速率。
// POST /api/v1/nodes/speedtest/stream
// 参数：ids=1,2,3 或 group=分组名；target=下载地址；size=最大字节；timeout=秒
func NodeSpeedTestStream(c *gin.Context) {
	ids := parseIDList(c.PostForm("ids"))
	group := strings.TrimSpace(c.PostForm("group"))
	target := strings.TrimSpace(c.PostForm("target"))
	if target == "" {
		target = defaultSpeedTestTarget
	}
	maxBytes := int64(3 << 20)
	if v, err := strconv.Atoi(c.PostForm("size")); err == nil && v > 0 && v <= 100<<20 {
		maxBytes = int64(v)
	}
	timeout := 20 * time.Second
	if v, err := strconv.Atoi(c.PostForm("timeout")); err == nil && v >= 3 && v <= 120 {
		timeout = time.Duration(v) * time.Second
	}

	nodesList, err := speedTestNodes(ids, group)
	if err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "读取节点失败"})
		return
	}
	if len(nodesList) == 0 {
		c.JSON(400, gin.H{"code": "40000", "msg": "没有可测试的节点"})
		return
	}

	ctx, cancel, ok := node.BeginTest("批量测速", 0, "speed", c.Request.Context())
	if !ok {
		c.JSON(429, gin.H{"code": "42900", "msg": "已有测试进行中，请稍候"})
		return
	}
	defer node.EndTest()
	defer cancel()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	total := len(nodesList)
	for i, n := range nodesList {
		if ctx.Err() != nil {
			break
		}
		result := speedTestOne(ctx, n, target, timeout, maxBytes)
		result["index"] = i + 1
		result["total"] = total
		c.SSEvent("node", result)
		c.Writer.Flush()
	}
	c.SSEvent("done", gin.H{"total": total, "msg": "完成"})
	c.Writer.Flush()
}

func parseIDList(raw string) []int {
	out := []int{}
	for _, s := range strings.Split(raw, ",") {
		if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && id > 0 {
			out = append(out, id)
		}
	}
	return out
}

func speedTestNodes(ids []int, group string) ([]models.Node, error) {
	if len(ids) > 0 {
		var nodes []models.Node
		if err := models.DB.Where("id IN ?", ids).Find(&nodes).Error; err != nil {
			return nil, err
		}
		return models.FilterVisibleNodes(nodes), nil
	}
	if group != "" {
		var g models.GroupNode
		if err := models.DB.Where("name = ?", group).Preload("Nodes").First(&g).Error; err != nil {
			return nil, err
		}
		return models.FilterVisibleNodes(g.Nodes), nil
	}
	return models.GetNodeList()
}

func speedTestOne(ctx context.Context, n models.Node, target string, timeout time.Duration, maxBytes int64) gin.H {
	res := gin.H{"id": n.ID, "name": n.Name, "rtt": -1, "ok": false, "mbps": 0.0, "bytes": 0}
	if host, port := node.ExtractServerHost(n.Link); host != "" && port > 0 {
		res["rtt"] = node.TCPPing(host+":"+strconv.Itoa(port), 4*time.Second)
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	body, _, err := node.FetchURLThroughNode(runCtx, n.Link, target, "SubLinkX-Speedtest/1.0", timeout, maxBytes)
	elapsed := time.Since(start).Seconds()
	if err != nil {
		res["error"] = err.Error()
		return res
	}
	mbps := 0.0
	if elapsed > 0 {
		mbps = float64(len(body)) * 8 / elapsed / 1e6
	}
	res["ok"] = true
	res["bytes"] = len(body)
	res["mbps"] = math.Round(mbps*100) / 100
	res["seconds"] = math.Round(elapsed*100) / 100
	return res
}
