package main

import (
	"log"
	"ppeelink/api"
	"ppeelink/models"
	"time"

	"github.com/robfig/cron/v3"
)

func StartCronTasks() {
	c := cron.New(cron.WithSeconds()) // 支持秒级

	// 每天凌晨 3:00 跑一次机场节点拉取与测活清理
	_, err := c.AddFunc("0 0 3 * * *", func() {
		api.SyncAllAirports()
		if err := models.CleanupNodeQuality(time.Now().Add(-30 * 24 * time.Hour)); err != nil {
			log.Println("[Cron] 清理节点质量历史失败:", err)
		}
	})

	if err != nil {
		log.Println("[Cron] 添加定时任务失败:", err)
		return
	}
	_, err = c.AddFunc("0 */10 * * * *", func() {
		if _, err := api.CollectNodeQuality(); err != nil {
			log.Println("[Cron] 节点质量检测失败:", err)
		}
	})
	if err != nil {
		log.Println("[Cron] 添加节点质量任务失败:", err)
		return
	}

	c.Start()
	log.Println("[Cron] 机场每日同步与节点每10分钟质量检测已启动")
}
