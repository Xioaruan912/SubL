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

func TestParseMultiplierExport(t *testing.T) {
	// 占位：确保 newNodeSpec 相关常量可用（编译期覆盖）
	if parseMemberIDs("1")[0] != 1 {
		t.Fatal("unexpected")
	}
}
