package cmd

import "testing"

func TestTokenPresenceLabelDoesNotExposeToken(t *testing.T) {
	if got := tokenPresenceLabel("secret-token-value"); got != "present" {
		t.Fatalf("expected present label, got %q", got)
	}

	if got := tokenPresenceLabel("   "); got != "missing" {
		t.Fatalf("expected missing label, got %q", got)
	}
}
