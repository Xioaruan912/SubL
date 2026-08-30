package api

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"ppeelink/models"
	"ppeelink/node"
)

type qualityMatrixSampleRequest struct {
	Mode string `json:"mode"`
}

type qualityMatrixSampleResult struct {
	Mode         string `json:"mode"`
	OnlineNodes  int    `json:"onlineNodes"`
	SkippedNodes int    `json:"skippedNodes"`
	Targets      int    `json:"targets"`
	Samples      int    `json:"samples"`
	Successes    int    `json:"successes"`
	Failures     int    `json:"failures"`
}

func matrixSamplingTargets(mode string, now time.Time) ([]node.EgressTarget, error) {
	targets, err := enabledNodeEgressTargets()
	if err != nil || mode == "full" {
		return targets, err
	}
	byScene := map[string][]node.EgressTarget{}
	for _, target := range targets {
		byScene[target.Group] = append(byScene[target.Group], target)
	}
	scenes := make([]string, 0, len(byScene))
	for scene := range byScene {
		scenes = append(scenes, scene)
	}
	sort.Strings(scenes)
	cycle := int(now.Unix() / int64((6 * time.Hour).Seconds()))
	selected := make([]node.EgressTarget, 0, len(scenes))
	for _, scene := range scenes {
		items := byScene[scene]
		selected = append(selected, items[cycle%len(items)])
	}
	return selected, nil
}

func collectQualityMatrixSamples(ctx context.Context, mode string, progress func(int, string)) (qualityMatrixSampleResult, error) {
	if mode != "full" {
		mode = "scene"
	}
	targets, err := matrixSamplingTargets(mode, time.Now())
	if err != nil {
		return qualityMatrixSampleResult{}, err
	}
	if len(targets) == 0 {
		return qualityMatrixSampleResult{}, fmt.Errorf("没有启用的检测目标")
	}
	nodes, err := models.GetNodeList()
	if err != nil {
		return qualityMatrixSampleResult{}, err
	}
	stats, err := models.GetNodeQualityStats(time.Now().Add(-30 * time.Minute))
	if err != nil {
		return qualityMatrixSampleResult{}, err
	}
	online := make([]models.Node, 0, len(nodes))
	for _, item := range nodes {
		stat, ok := stats[item.ID]
		if ok && stat.LastRtt >= 0 && time.Since(stat.LastTestedAt) <= 30*time.Minute {
			online = append(online, item)
		}
	}
	if len(online) == 0 {
		fresh, collectErr := CollectNodeQuality()
		if collectErr != nil {
			return qualityMatrixSampleResult{}, fmt.Errorf("首次 TCP 检测失败: %w", collectErr)
		}
		byID := make(map[int]models.Node, len(nodes))
		for _, item := range nodes {
			byID[item.ID] = item
		}
		for _, item := range fresh {
			if item.Rtt >= 0 {
				if model, ok := byID[item.ID]; ok {
					online = append(online, model)
				}
			}
		}
	}
	result := qualityMatrixSampleResult{Mode: mode, OnlineNodes: len(online), SkippedNodes: len(nodes) - len(online), Targets: len(targets)}
	if len(online) == 0 {
		return result, fmt.Errorf("最近 30 分钟没有在线节点样本，请先等待/执行 TCP 节点检测")
	}
	if progress != nil {
		progress(5, fmt.Sprintf("准备采样：%d 个在线节点 × %d 个目标", len(online), len(targets)))
	}
	type outcome struct{ samples, successes, failures int }
	jobs := make(chan models.Node)
	outcomes := make(chan outcome, len(online))
	workerCount := 3
	if len(online) < workerCount {
		workerCount = len(online)
	}
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				if ctx.Err() != nil {
					return
				}
				runCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
				check, runErr := node.RunEgressTestTargets(runCtx, item.Link, 7*time.Second, targets)
				cancel()
				if runErr != nil {
					now := time.Now()
					failed := make([]models.NodeTargetQualitySample, 0, len(targets))
					for _, target := range targets {
						failed = append(failed, models.NodeTargetQualitySample{NodeID: item.ID, TargetKey: target.Key, Scene: target.Group, Rtt: -1, Success: false, Status: "error", CheckedAt: now})
					}
					_ = models.RecordNodeTargetQuality(failed)
					outcomes <- outcome{samples: len(targets), failures: len(targets)}
					continue
				}
				_ = recordEgressQuality(item.ID, check)
				o := outcome{samples: len(check.Results)}
				for _, r := range check.Results {
					if r.Status == "available" || r.Status == "reachable" {
						o.successes++
					} else {
						o.failures++
					}
				}
				outcomes <- o
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, item := range online {
			select {
			case jobs <- item:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() { wg.Wait(); close(outcomes) }()
	done := 0
	for o := range outcomes {
		done++
		result.Samples += o.samples
		result.Successes += o.successes
		result.Failures += o.failures
		if progress != nil {
			progress(5+done*90/len(online), fmt.Sprintf("已采样 %d/%d 个在线节点", done, len(online)))
		}
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, nil
}

func RunScheduledQualityMatrixSample() error {
	var active int64
	if err := models.DB.Model(&models.TaskRun{}).Where("type = ? AND status IN ?", "quality-matrix-sample", []string{"queued", "running"}).Count(&active).Error; err == nil && active > 0 {
		return nil
	}
	task := &models.TaskRun{Type: "quality-matrix-sample", Name: "定时质量矩阵场景采样", Status: "queued", Progress: 0, Message: "等待执行", RequestJSON: "{\"mode\":\"scene\"}", CreatedAt: time.Now()}
	if err := models.DB.Create(task).Error; err != nil {
		return err
	}
	go executeStoredTask(task)
	return nil
}

func EnsureInitialQualityMatrixSample() {
	var count int64
	if err := models.DB.Model(&models.NodeTargetQualitySample{}).Where("checked_at >= ?", time.Now().Add(-24*time.Hour)).Count(&count).Error; err != nil || count > 0 {
		return
	}
	go func() {
		time.Sleep(15 * time.Second)
		_ = RunScheduledQualityMatrixSample()
	}()
}
