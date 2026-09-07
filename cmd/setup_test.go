package cmd

import "testing"

func TestParseGitHubRemote(t *testing.T) {
	tests := []struct {
		remote     string
		owner      string
		repository string
	}{
		{"git@github.com:intercube/shop.git", "intercube", "shop"},
		{"https://github.com/intercube/shop.git", "intercube", "shop"},
		{"ssh://git@github.com/intercube/shop.git", "intercube", "shop"},
	}
	for _, test := range tests {
		owner, repository, err := parseGitHubRemote(test.remote)
		if err != nil {
			t.Fatalf("parseGitHubRemote(%q): %v", test.remote, err)
		}
		if owner != test.owner || repository != test.repository {
			t.Fatalf("parseGitHubRemote(%q) = %s/%s", test.remote, owner, repository)
		}
	}
}

func TestValidateSetupDomainProtectsManagedNamespace(t *testing.T) {
	if err := validateSetupDomain("quiet-river.mycube.dev", "quiet-river.mycube.dev"); err != nil {
		t.Fatalf("generated domain should be accepted: %v", err)
	}
	if err := validateSetupDomain("my-preferred-name.mycube.dev", "quiet-river.mycube.dev"); err == nil {
		t.Fatal("custom managed domain should be rejected")
	}
	if err := validateSetupDomain("shop.example.com", "quiet-river.mycube.dev"); err != nil {
		t.Fatalf("customer domain should be accepted: %v", err)
	}
}

func TestSetupIdempotencyKeyIsStablePerEnvironment(t *testing.T) {
	first := setupIdempotencyKey("Intercube", "Shop", "production", "main")
	second := setupIdempotencyKey("intercube", "shop", "production", "main")
	development := setupIdempotencyKey("intercube", "shop", "development", "develop")
	if first != second {
		t.Fatalf("expected case-insensitive stable key, got %q and %q", first, second)
	}
	if first == development {
		t.Fatal("different environments should have different idempotency keys")
	}
}
