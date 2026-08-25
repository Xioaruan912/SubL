package node

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestCheckFunctionsDirect(t *testing.T) {
	c := &http.Client{Timeout: 15 * time.Second}
	ctx := context.Background()
	tests := []struct {
		name string
		chk  func(ctx context.Context, c *http.Client) (bool, string)
	}{
		{"OpenAI", checkOpenAI},
		{"Claude", checkClaude},
		{"Gemini", checkGemini},
		{"Netflix", checkNetflix},
		{"YouTube", checkYouTube},
		{"Disney", checkDisney},
		{"Google", checkGoogle},
		{"GitHub", checkGitHub},
		{"Telegram", checkTelegram},
	}
	for _, tt := range tests {
		ok, note := tt.chk(ctx, c)
		t.Logf("%-8s ok=%v note=%s", tt.name, ok, note)
	}
}