package main

import (
	"context"
	"log"
	"ppeelink/api"
	"ppeelink/models"
	"ppeelink/rulecenter"
	"ppeelink/utils"
	"time"

	"github.com/robfig/cron/v3"
)

func StartCronTasks() {
	c := cron.New(cron.WithSeconds()) // 支持秒级

	// 每天凌晨 3:00 跑一次机场节点拉取与测活清理
	_, err := c.AddFunc("0 0 3 * * *", func() {
		defer utils.RecoverPanic("cron-airport-sync")
		api.SyncAllAirports()
		if err := models.CleanupNodeQuality(time.Now().Add(-30 * 24 * time.Hour)); err != nil {
			log.Println("[Cron] 清理节点质量历史失败:", err)
		}
		if err := models.CleanupNodeTargetQuality(time.Now().Add(-30 * 24 * time.Hour)); err != nil {
			log.Println("[Cron] 清理目标质量历史失败:", err)
		}
		models.CleanupMaintenance(time.Now())
	})

	if err != nil {
		log.Println("[Cron] 添加定时任务失败:", err)
		return
	}
	_, err = c.AddFunc("0 30 3 * * *", func() {
		defer utils.RecoverPanic("cron-rule-sync")
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if err := rulecenter.SyncAll(ctx); err != nil {
			log.Println("[Cron] 规则中心同步失败:", err)
		}
	})
	if err != nil {
		log.Println("[Cron] 添加规则中心同步任务失败:", err)
		return
	}

	_, err = c.AddFunc("0 */10 * * * *", func() {
		defer utils.RecoverPanic("cron-node-quality")
		if _, err := api.CollectNodeQuality(); err != nil {
			log.Println("[Cron] 节点质量检测失败:", err)
		}
	})
	if err != nil {
		log.Println("[Cron] 添加节点质量任务失败:", err)
		return
	}

	_, err = c.AddFunc("0 20 */6 * * *", func() {
		defer utils.RecoverPanic("cron-quality-matrix")
		if err := api.RunScheduledQualityMatrixSample(); err != nil {
			log.Println("[Cron] 质量矩阵场景采样失败:", err)
		}
	})
	if err != nil {
		log.Println("[Cron] 添加质量矩阵采样任务失败:", err)
		return
	}

	c.Start()
	api.EnsureInitialQualityMatrixSample()
	go func() {
		defer utils.RecoverPanic("startup-rule-sync")
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if err := rulecenter.SyncAll(ctx); err != nil {
			log.Println("[RuleCenter] 启动同步失败，保留现有缓存:", err)
		}
	}()
	log.Println("[Cron] 机场/规则同步、节点每10分钟 TCP 质量与每6小时目标场景采样已启动")
}
