package models

import (
	"log"
	"time"
)

// CleanupMaintenance 按保守保留期清理历史表，避免数据库无限增长。
// 审计日志保留 365 天，任务记录 180 天，健康事件与解锁观察 90 天。
func CleanupMaintenance(now time.Time) {
	type job struct {
		name   string
		window time.Duration
		run    func(before time.Time) (int64, error)
	}
	jobs := []job{
		{"node_health_events", 90 * 24 * time.Hour, func(before time.Time) (int64, error) {
			tx := DB.Where("created_at < ?", before).Delete(&NodeHealthEvent{})
			return tx.RowsAffected, tx.Error
		}},
		{"unlock_observations", 90 * 24 * time.Hour, func(before time.Time) (int64, error) {
			tx := DB.Where("checked_at < ?", before).Delete(&UnlockObservation{})
			return tx.RowsAffected, tx.Error
		}},
		{"audit_logs", 365 * 24 * time.Hour, func(before time.Time) (int64, error) {
			tx := DB.Where("created_at < ?", before).Delete(&AuditLog{})
			return tx.RowsAffected, tx.Error
		}},
		{"task_runs", 180 * 24 * time.Hour, func(before time.Time) (int64, error) {
			tx := DB.Where("created_at < ? AND status IN ?", before, []string{"success", "failed", "cancelled"}).Delete(&TaskRun{})
			return tx.RowsAffected, tx.Error
		}},
	}
	for _, j := range jobs {
		deleted, err := j.run(now.Add(-j.window))
		if err != nil {
			log.Printf("[Cleanup] %s 清理失败: %v", j.name, err)
			continue
		}
		if deleted > 0 {
			log.Printf("[Cleanup] %s 清理 %d 行", j.name, deleted)
		}
	}
}
