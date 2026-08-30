package api

import (
	"testing"

	"ppeelink/models"
)

func TestValidateEgressTargetDoesNotContainCountryPolicy(t *testing.T) {
	item := models.EgressTarget{Key: "youtube", Name: "YouTube", Domain: "www.youtube.com", Group: "media", Method: "GET", Enabled: true}
	if msg := validateEgressTarget(&item); msg != "" {
		t.Fatalf("valid target rejected: %s", msg)
	}
	if item.TimeoutSeconds != 7 || item.ExpectedStatus != "200-399" {
		t.Fatalf("defaults not normalized: %#v", item)
	}
}

func TestValidateEgressTargetRejectsURLAndUnsafeKey(t *testing.T) {
	for _, item := range []models.EgressTarget{
		{Key: "Bad Key", Name: "Bad", Domain: "example.com", Group: "test"},
		{Key: "bad", Name: "Bad", Domain: "https://example.com/path", Group: "test"},
	} {
		if msg := validateEgressTarget(&item); msg == "" {
			t.Fatalf("invalid target accepted: %#v", item)
		}
	}
}
