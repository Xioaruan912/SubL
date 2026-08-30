package models

import (
	"math"
	"sort"
	"time"
)

// NodeTargetQualitySample records one real destination check through one node.
// TargetKey identifies a concrete configured destination while Scene groups
// related destinations (ai/media/network/...).
type NodeTargetQualitySample struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	NodeID    int       `gorm:"index:idx_target_quality_node_time,priority:1;not null" json:"nodeId"`
	TargetKey string    `gorm:"size:80;index:idx_target_quality_target_time,priority:1;not null" json:"targetKey"`
	Scene     string    `gorm:"size:80;index:idx_target_quality_scene_time,priority:1;not null" json:"scene"`
	Rtt       int       `gorm:"not null" json:"rtt"`
	Success   bool      `gorm:"not null" json:"success"`
	Status    string    `gorm:"size:32" json:"status"`
	CheckedAt time.Time `gorm:"index:idx_target_quality_node_time,priority:2;index:idx_target_quality_target_time,priority:2;index:idx_target_quality_scene_time,priority:2;not null" json:"checkedAt"`
}

type TargetQualityStats struct {
	NodeID       int       `json:"nodeId"`
	TargetKey    string    `json:"targetKey,omitempty"`
	Scene        string    `json:"scene,omitempty"`
	Score        int       `json:"score"`
	Availability float64   `json:"availability"`
	AverageRtt   int       `json:"averageRtt"`
	P95Rtt       int       `json:"p95Rtt"`
	Confidence   int       `json:"confidence"`
	SampleCount  int       `json:"sampleCount"`
	LastStatus   string    `json:"lastStatus"`
	LastRtt      int       `json:"lastRtt"`
	LastTestedAt time.Time `json:"lastTestedAt"`
}

func RecordNodeTargetQuality(samples []NodeTargetQualitySample) error {
	if len(samples) == 0 { return nil }
	return DB.CreateInBatches(samples, 100).Error
}

type targetQualityKey struct { nodeID int; label string }

func aggregateTargetQuality(samples []NodeTargetQualitySample, byScene bool) map[targetQualityKey]TargetQualityStats {
	type acc struct {
		count, successes int
		rtts []int
		sum float64
		lastAt time.Time
		lastStatus string
		lastRtt int
	}
	accs := map[targetQualityKey]*acc{}
	for _, s := range samples {
		label := s.TargetKey; if byScene { label = s.Scene }
		k := targetQualityKey{s.NodeID, label}
		a := accs[k]; if a == nil { a = &acc{lastRtt:-1}; accs[k] = a }
		a.count++
		if s.Success && s.Rtt >= 0 { a.successes++; a.sum += float64(s.Rtt); a.rtts = append(a.rtts, s.Rtt) }
		if s.CheckedAt.After(a.lastAt) { a.lastAt = s.CheckedAt; a.lastStatus = s.Status; a.lastRtt = s.Rtt }
	}
	result := make(map[targetQualityKey]TargetQualityStats, len(accs))
	for k, a := range accs {
		availability := 0.0; average, p95 := -1, -1
		if a.count > 0 { availability = float64(a.successes)*100/float64(a.count) }
		if len(a.rtts) > 0 {
			average = int(math.Round(a.sum/float64(len(a.rtts))))
			sort.Ints(a.rtts); p95 = a.rtts[int(math.Ceil(float64(len(a.rtts))*0.95))-1]
		}
		stats := TargetQualityStats{NodeID:k.nodeID, Score:CalculateNodeQualityScore(availability, average, 0), Availability:math.Round(availability*10)/10, AverageRtt:average, P95Rtt:p95, Confidence:minInt(100,a.count*10), SampleCount:a.count, LastStatus:a.lastStatus, LastRtt:a.lastRtt, LastTestedAt:a.lastAt}
		if byScene { stats.Scene = k.label } else { stats.TargetKey = k.label }
		result[k] = stats
	}
	return result
}

func GetNodeTargetQualityStats(since time.Time) (map[int]map[string]TargetQualityStats, error) {
	var samples []NodeTargetQualitySample
	if err := DB.Where("checked_at >= ?", since).Order("node_id, target_key, checked_at").Find(&samples).Error; err != nil { return nil, err }
	flat := aggregateTargetQuality(samples, false); result := map[int]map[string]TargetQualityStats{}
	for k, stat := range flat { if result[k.nodeID] == nil { result[k.nodeID] = map[string]TargetQualityStats{} }; result[k.nodeID][k.label] = stat }
	return result, nil
}

func GetNodeSceneQualityStats(since time.Time) (map[int]map[string]TargetQualityStats, error) {
	var samples []NodeTargetQualitySample
	if err := DB.Where("checked_at >= ?", since).Order("node_id, scene, checked_at").Find(&samples).Error; err != nil { return nil, err }
	flat := aggregateTargetQuality(samples, true); result := map[int]map[string]TargetQualityStats{}
	for k, stat := range flat { if result[k.nodeID] == nil { result[k.nodeID] = map[string]TargetQualityStats{} }; result[k.nodeID][k.label] = stat }
	return result, nil
}

func CleanupNodeTargetQuality(before time.Time) error {
	return DB.Where("checked_at < ?", before).Delete(&NodeTargetQualitySample{}).Error
}
