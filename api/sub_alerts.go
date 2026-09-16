package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ppeelink/models"
	"ppeelink/utils"

	"github.com/gin-gonic/gin"
)

// subscriptionAccessCount 订阅累计访问次数（来自 SubLogs 的 Count 求和），
// 作为没有真实流量上报时的用量近似。
func subscriptionAccessCount(subID int) int {
	var total int64
	models.DB.Model(&models.SubLogs{}).Where("subcription_id = ?", subID).
		Select("COALESCE(SUM(count), 0)").Scan(&total)
	return int(total)
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func notifySubscription(setting models.AlertSetting, sub models.Subcription, eventType, message string) {
	if !setting.Enabled || setting.WebhookURL == "" {
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"event": eventType, "subscription": sub.Name, "message": message, "time": time.Now(),
	})
	go func() {
		defer utils.RecoverPanic("sub-alert-webhook")
		req, err := http.NewRequest("POST", setting.WebhookURL, bytes.NewReader(payload))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 8 * time.Second}
		if resp, err := client.Do(req); err == nil {
			resp.Body.Close()
		}
	}()
}

// notifySubscriptionOnce 同一订阅每天最多提醒一次（自动停用类除外）。
func notifySubscriptionOnce(sub *models.Subcription, setting models.AlertSetting, eventType, message string) {
	now := time.Now()
	if sub.LastRemindedAt != nil && sameDay(*sub.LastRemindedAt, now) {
		return
	}
	notifySubscription(setting, *sub, eventType, message)
	models.DB.Model(sub).Update("last_reminded_at", &now)
}

// CheckSubscriptionAlerts 检查订阅到期与访问量，按设置提醒并可自动停用（清空令牌）。
func CheckSubscriptionAlerts() {
	setting := currentAlertSetting()
	if !setting.Enabled {
		return
	}
	var subs []models.Subcription
	if err := models.DB.Find(&subs).Error; err != nil {
		return
	}
	now := time.Now()
	for i := range subs {
		sub := subs[i]
		if sub.Token == "" {
			continue // 已停用
		}
		if sub.ExpiresAt != nil {
			if now.After(*sub.ExpiresAt) {
				if setting.AutoDisableExpired {
					models.DB.Model(&sub).Update("token", "")
					notifySubscription(setting, sub, "subscription-expired", "订阅已到期，链接已自动停用")
				} else {
					notifySubscriptionOnce(&sub, setting, "subscription-expired", "订阅已到期")
				}
				continue
			}
			if setting.ExpiryReminderDays > 0 && sub.ExpiresAt.Before(now.AddDate(0, 0, setting.ExpiryReminderDays)) {
				notifySubscriptionOnce(&sub, setting, "subscription-expiring",
					fmt.Sprintf("订阅将在 %s 到期", sub.ExpiresAt.Format("2006-01-02")))
			}
		}
		if setting.AccessLimit > 0 {
			count := subscriptionAccessCount(sub.ID)
			if count >= setting.AccessLimit {
				if setting.AutoDisableOverLimit {
					models.DB.Model(&sub).Update("token", "")
					notifySubscription(setting, sub, "subscription-traffic",
						fmt.Sprintf("访问量 %d 已达到上限 %d，链接已自动停用", count, setting.AccessLimit))
				} else {
					notifySubscriptionOnce(&sub, setting, "subscription-traffic",
						fmt.Sprintf("访问量 %d 已达到上限 %d", count, setting.AccessLimit))
				}
			}
		}
	}
}

// SubscriptionCheckAlerts 手动触发一次订阅告警检查（管理员）。
func SubscriptionCheckAlerts(c *gin.Context) {
	CheckSubscriptionAlerts()
	c.JSON(200, gin.H{"code": "00000", "msg": "订阅告警检查已执行"})
}

// SubscriptionUsage 返回订阅用量概览（到期时间 / 访问量）。
func SubscriptionUsage(c *gin.Context) {
	ids := parseIDList(c.Query("id"))
	if len(ids) == 0 {
		c.JSON(400, gin.H{"code": "40000", "msg": "订阅 id 无效"})
		return
	}
	var sub models.Subcription
	if err := models.DB.First(&sub, ids[0]).Error; err != nil {
		c.JSON(404, gin.H{"code": "40400", "msg": "订阅不存在"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": gin.H{
		"id": sub.ID, "name": sub.Name, "expiresAt": sub.ExpiresAt,
		"accessCount": subscriptionAccessCount(sub.ID), "disabled": sub.Token == "",
		"lastRemindedAt": sub.LastRemindedAt,
	}, "msg": "订阅用量"})
}
