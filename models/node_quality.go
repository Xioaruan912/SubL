package models

import (
	"math"
	"sort"
	"time"
)

// NodeQualitySample stores a server-side TCP health check. Keeping samples
// separate from Node makes quality trends survive restarts and page refreshes.
type NodeQualitySample struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	NodeID    int       `gorm:"index:idx_node_quality_time,priority:1;not null" json:"nodeId"`
	Rtt       int       `gorm:"not null" json:"rtt"`
	Success   bool      `gorm:"not null" json:"success"`
	CheckedAt time.Time `gorm:"index:idx_node_quality_time,priority:2;not null" json:"checkedAt"`
}

type NodeQualityStats struct {
	NodeID              int       `json:"nodeId"`
	Score               int       `json:"score"`
	Availability        float64   `json:"availability"`
	AverageRtt          int       `json:"averageRtt"`
	Jitter              int       `json:"jitter"`
	P95Rtt              int       `json:"p95Rtt"`
	ConsecutiveFailures int       `json:"consecutiveFailures"`
	Confidence          int       `json:"confidence"`
	SampleCount         int       `json:"sampleCount"`
	LastRtt             int       `json:"lastRtt"`
	LastTestedAt        time.Time `json:"lastTestedAt"`
}

func RecordNodeQuality(samples []NodeQualitySample) error {
	if len(samples) == 0 {
		return nil
	}
	return DB.CreateInBatches(samples, 100).Error
}

// GetNodeQualityStats calculates a rolling window in Go. This keeps the query
// portable across SQLite versions and the sample volume is bounded by cleanup.
func GetNodeQualityStats(since time.Time) (map[int]NodeQualityStats, error) {
	var samples []NodeQualitySample
	if err := DB.Where("checked_at >= ?", since).Order("node_id, checked_at").Find(&samples).Error; err != nil {
		return nil, err
	}
	type accumulator struct {
		count, successes    int
		sum, sumSquares     float64
		lastRtt             int
		lastAt              time.Time
		rtts                []int
		consecutiveFailures int
	}
	acc := make(map[int]*accumulator)
	for _, sample := range samples {
		a := acc[sample.NodeID]
		if a == nil {
			a = &accumulator{lastRtt: -1}
			acc[sample.NodeID] = a
		}
		a.count++
		if sample.Success && sample.Rtt >= 0 {
			a.successes++
			a.sum += float64(sample.Rtt)
			a.sumSquares += float64(sample.Rtt * sample.Rtt)
			a.rtts = append(a.rtts, sample.Rtt)
			a.consecutiveFailures = 0
		} else {
			a.consecutiveFailures++
		}
		if sample.CheckedAt.After(a.lastAt) {
			a.lastAt = sample.CheckedAt
			a.lastRtt = sample.Rtt
		}
	}
	result := make(map[int]NodeQualityStats, len(acc))
	for nodeID, a := range acc {
		availability := 0.0
		average, jitter, p95 := -1, 0, -1
		if a.count > 0 {
			availability = float64(a.successes) * 100 / float64(a.count)
		}
		if a.successes > 0 {
			mean := a.sum / float64(a.successes)
			variance := a.sumSquares/float64(a.successes) - mean*mean
			if variance < 0 {
				variance = 0
			}
			average = int(math.Round(mean))
			jitter = int(math.Round(math.Sqrt(variance)))
			sort.Ints(a.rtts)
			p95 = a.rtts[int(math.Ceil(float64(len(a.rtts))*0.95))-1]
		}
		result[nodeID] = NodeQualityStats{
			NodeID: nodeID, Score: CalculateNodeQualityScore(availability, average, jitter),
			Availability: math.Round(availability*10) / 10, AverageRtt: average,
			Jitter: jitter, SampleCount: a.count, LastRtt: a.lastRtt, LastTestedAt: a.lastAt,
			P95Rtt: p95, ConsecutiveFailures: a.consecutiveFailures,
			Confidence: minInt(100, a.count*10),
		}
	}
	return result, nil
}

// CalculateNodeQualityScore intentionally uses only measured signals:
// availability 50%, latency 30%, stability 20%.
func CalculateNodeQualityScore(availability float64, averageRtt, jitter int) int {
	if averageRtt < 0 {
		return 0
	}
	latencyScore := clamp(100-float64(maxInt(averageRtt-50, 0))*100/550, 0, 100)
	jitterScore := clamp(100-float64(jitter)*100/250, 0, 100)
	score := availability*0.5 + latencyScore*0.3 + jitterScore*0.2
	return int(math.Round(clamp(score, 0, 100)))
}

func CleanupNodeQuality(before time.Time) error {
	return DB.Where("checked_at < ?", before).Delete(&NodeQualitySample{}).Error
}

func clamp(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
