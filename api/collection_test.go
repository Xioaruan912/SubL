package api

import "testing"

func TestParseMemberIDs(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"[1,2,3]", 3},
		{"4,5", 2},
		{"", 0},
		{"[7]", 1},
	}
	for _, tc := range cases {
		if got := parseMemberIDs(tc.raw); len(got) != tc.want {
			t.Fatalf("parseMemberIDs(%q) = %v, want %d items", tc.raw, got, tc.want)
		}
	}
}

func TestIsPlaceholderNodeName(t *testing.T) {
	if !isPlaceholderNodeName("订阅已取消，请打开订阅共享重新选择") {
		t.Fatal("expected cancelled-subscription node to be a placeholder")
	}
	if isPlaceholderNodeName("香港 01") {
		t.Fatal("normal node must not be treated as placeholder")
	}
}
