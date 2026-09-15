package utils

import (
	"net"
	"testing"
)

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.1", "169.254.169.254", "100.64.0.1", "::1", "fe80::1"}
	for _, raw := range blocked {
		if !IsBlockedIP(net.ParseIP(raw)) {
			t.Fatalf("expected %s to be blocked", raw)
		}
	}
	allowed := []string{"1.1.1.1", "8.8.8.8", "2606:4700:4700::1111"}
	for _, raw := range allowed {
		if IsBlockedIP(net.ParseIP(raw)) {
			t.Fatalf("expected %s to be allowed", raw)
		}
	}
}

func TestValidateOutboundURL(t *testing.T) {
	rejected := []string{"file:///etc/passwd", "http://127.0.0.1/x", "https://169.254.169.254/latest/meta-data", "http://10.0.0.1/"}
	for _, raw := range rejected {
		if err := ValidateOutboundURL(raw); err == nil {
			t.Fatalf("expected %s to be rejected", raw)
		}
	}
	if err := ValidateOutboundURL("https://example.com/path"); err != nil {
		t.Fatalf("expected public https URL to pass, got %v", err)
	}
}
