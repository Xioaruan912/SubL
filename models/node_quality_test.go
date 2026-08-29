package models

import "testing"

func TestCalculateNodeQualityScore(t *testing.T) {
	tests := []struct {
		name               string
		availability       float64
		averageRtt, jitter int
		wantMin, wantMax   int
	}{
		{"excellent", 100, 50, 0, 99, 100},
		{"unstable", 80, 300, 180, 40, 70},
		{"offline", 0, -1, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateNodeQualityScore(tt.availability, tt.averageRtt, tt.jitter)
			if got < tt.wantMin || got > tt.wantMax {
				t.Fatalf("score=%d, want range %d..%d", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}
