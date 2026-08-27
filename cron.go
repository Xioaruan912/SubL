package main

import (
	"log"
	"ppeelink/api"

	"github.com/robfig/cron/v3"
)

func StartCronTasks() {
	c := cron.New(cron.WithSeconds()) // 支持秒级
	
	// 每天凌晨 3:00 跑一次机场节点拉取与测活清理
	_, err := c.AddFunc("0 0 3 * * *", func() {
		api.SyncAllAirports()
	})
	
	if err != nil {
		log.Println("[Cron] 添加定时任务失败:", err)
		return
	}
	
	c.Start()
	log.Println("[Cron] 每日定时测活任务 (3:00 AM) 已启动")
}
