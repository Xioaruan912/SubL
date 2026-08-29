package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"ppeelink/models"

	"github.com/gin-gonic/gin"
)

func currentAlertSetting() models.AlertSetting {
	var setting models.AlertSetting
	if err := models.DB.First(&setting).Error; err != nil {
		setting = models.AlertSetting{FailureThreshold: 3}
		models.DB.Create(&setting)
	}
	if setting.FailureThreshold < 1 {
		setting.FailureThreshold = 3
	}
	return setting
}

func inMaintenance(setting models.AlertSetting, now time.Time) bool {
	if setting.MaintenanceStart == "" || setting.MaintenanceEnd == "" {
		return false
	}
	current := now.Format("15:04")
	if setting.MaintenanceStart <= setting.MaintenanceEnd {
		return current >= setting.MaintenanceStart && current <= setting.MaintenanceEnd
	}
	return current >= setting.MaintenanceStart || current <= setting.MaintenanceEnd
}

func notifyHealthEvent(setting models.AlertSetting, event models.NodeHealthEvent) {
	if !setting.Enabled || setting.WebhookURL == "" || inMaintenance(setting, time.Now()) {
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{"event": event.Type, "node": event.NodeName, "message": event.Message, "time": event.CreatedAt})
	go func() {
		req, err := http.NewRequest("POST", setting.WebhookURL, bytes.NewReader(payload))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 8 * time.Second}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}()
}

func processNodeHealthEvents(items []NodeOverviewItem) {
	setting := currentAlertSetting()
	for _, item := range items {
		var recent []models.NodeQualitySample
		models.DB.Where("node_id = ?", item.ID).Order("checked_at desc").Limit(setting.FailureThreshold).Find(&recent)
		var last models.NodeHealthEvent
		models.DB.Where("node_id = ?", item.ID).Order("id desc").First(&last)
		allFailed := len(recent) >= setting.FailureThreshold
		for _, sample := range recent {
			if sample.Success {
				allFailed = false
				break
			}
		}
		var event *models.NodeHealthEvent
		if allFailed && last.Type != "down" {
			e := models.NodeHealthEvent{NodeID: item.ID, NodeName: item.Name, Type: "down", Message: fmt.Sprintf("连续 %d 次连接失败", setting.FailureThreshold), CreatedAt: time.Now()}
			event = &e
		} else if item.Rtt >= 0 && last.Type == "down" {
			e := models.NodeHealthEvent{NodeID: item.ID, NodeName: item.Name, Type: "recovery", Message: fmt.Sprintf("节点恢复，当前延迟 %dms", item.Rtt), CreatedAt: time.Now()}
			event = &e
		}
		if event != nil {
			models.DB.Create(event)
			notifyHealthEvent(setting, *event)
		}
	}
}

func NodeHealthEvents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit < 1 || limit > 200 {
		limit = 50
	}
	var events []models.NodeHealthEvent
	q := models.DB.Order("created_at desc").Limit(limit)
	if id := c.Query("id"); id != "" {
		q = q.Where("node_id = ?", id)
	}
	if err := q.Find(&events).Error; err != nil {
		c.JSON(500, gin.H{"msg": "读取健康事件失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": events, "msg": "健康事件"})
}

func GetAlertSetting(c *gin.Context) {
	c.JSON(200, gin.H{"code": "00000", "data": currentAlertSetting(), "msg": "告警设置"})
}

func UpdateAlertSetting(c *gin.Context) {
	setting := currentAlertSetting()
	setting.Enabled = c.PostForm("enabled") == "true"
	setting.WebhookURL = c.PostForm("webhookUrl")
	setting.MaintenanceStart = c.PostForm("maintenanceStart")
	setting.MaintenanceEnd = c.PostForm("maintenanceEnd")
	if n, err := strconv.Atoi(c.PostForm("failureThreshold")); err == nil && n >= 1 && n <= 20 {
		setting.FailureThreshold = n
	}
	if err := models.DB.Save(&setting).Error; err != nil {
		c.JSON(500, gin.H{"msg": "保存告警设置失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "告警设置已保存"})
}

type recommendation struct {
	NodeID     int      `json:"nodeId"`
	Name       string   `json:"name"`
	Score      int      `json:"score"`
	Confidence int      `json:"confidence"`
	Reasons    []string `json:"reasons"`
	Rtt        int      `json:"rtt"`
}

func NodeRecommendations(c *gin.Context) {
	var nodes []models.Node
	models.DB.Find(&nodes)
	stats, err := models.GetNodeQualityStats(time.Now().Add(-24 * time.Hour))
	if err != nil {
		c.JSON(500, gin.H{"msg": "读取质量数据失败"})
		return
	}
	// Load unlock observations once. The old implementation queried per node
	// and per scene, causing hundreds of SQLite queries on every dashboard load.
	services := []string{"openai", "claude", "google-gemini", "netflix", "youtube", "disney"}
	var observations []models.UnlockObservation
	models.DB.Where("service IN ?", services).Order("node_id asc, service asc, checked_at desc").Find(&observations)
	latestUnlock := make(map[int]map[string]bool)
	for _, o := range observations {
		perNode := latestUnlock[o.NodeID]
		if perNode == nil {
			perNode = make(map[string]bool)
			latestUnlock[o.NodeID] = perNode
		}
		if _, exists := perNode[o.Service]; !exists {
			perNode[o.Service] = o.Available
		}
	}
	type scene struct {
		Key   string           `json:"key"`
		Name  string           `json:"name"`
		Nodes []recommendation `json:"nodes"`
	}
	scenes := []scene{{Key: "stability", Name: "稳定优先"}, {Key: "speed", Name: "低延迟"}, {Key: "ai", Name: "AI 服务"}, {Key: "media", Name: "流媒体"}}
	for si := range scenes {
		list := make([]recommendation, 0, len(nodes))
		for _, n := range nodes {
			s, ok := stats[n.ID]
			if !ok {
				continue
			}
			score := s.Score
			reasons := []string{fmt.Sprintf("24h 可用率 %.1f%%", s.Availability), fmt.Sprintf("P95 %dms", s.P95Rtt)}
			if scenes[si].Key == "speed" {
				score = max(0, 100-s.AverageRtt/5)
				reasons = []string{fmt.Sprintf("平均延迟 %dms", s.AverageRtt), fmt.Sprintf("抖动 %dms", s.Jitter)}
			}
			if scenes[si].Key == "ai" || scenes[si].Key == "media" {
				sceneServices := []string{"openai", "claude", "google-gemini"}
				if scenes[si].Key == "media" {
					sceneServices = []string{"netflix", "youtube", "disney"}
				}
				unlocked, total := 0, 0
				perNode := latestUnlock[n.ID]
				for _, service := range sceneServices {
					available, ok := perNode[service]
					if !ok {
						continue
					}
					total++
					if available {
						unlocked++
					}
				}
				if total > 0 {
					score = s.Score*6/10 + unlocked*40/total
					reasons = append(reasons, fmt.Sprintf("已解锁 %d/%d 项", unlocked, total))
				} else {
					score = s.Score * 6 / 10
					reasons = append(reasons, "尚无解锁样本")
				}
			}
			list = append(list, recommendation{NodeID: n.ID, Name: n.Name, Score: score, Confidence: s.Confidence, Reasons: reasons, Rtt: s.AverageRtt})
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Score > list[j].Score })
		if len(list) > 3 {
			list = list[:3]
		}
		scenes[si].Nodes = list
	}
	c.JSON(200, gin.H{"code": "00000", "data": scenes, "msg": "场景推荐"})
}
