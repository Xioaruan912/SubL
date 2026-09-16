package api

import (
	"strconv"
	"strings"
	"testing"
)

func TestNormalizeSpeedTarget(t *testing.T) {
	out := normalizeSpeedTarget("https://speed.cloudflare.com/__down?bytes=50000000", 10*1024*1024)
	if !strings.Contains(out, "bytes="+strconv.Itoa(10*1024*1024)) {
		t.Fatalf("bytes param not aligned: %s", out)
	}
	if strings.Contains(out, "50000000") {
		t.Fatalf("old bytes value should be replaced: %s", out)
	}
	if got := normalizeSpeedTarget("https://example.com/file.bin", 1024); got != "https://example.com/file.bin" {
		t.Fatalf("url without bytes should be unchanged, got %s", got)
	}
}
