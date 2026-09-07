package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveProjectConfigPreservesOtherProjectSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".intercube.yaml")
	existing := "# existing settings\nsync:\n  files:\n    items: []\n"
	if err := os.WriteFile(path, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	config := &ProjectConfig{
		Version:    1,
		Repository: ProjectRepository{URL: "https://github.com/intercube/shop.git"},
		Environments: map[string]Environment{
			"production": {
				Branch: "main", Domain: "shop.example.com", ManagedDomain: "quiet-river.mycube.dev",
				IntentID: "intent-1", IntentRevision: "rev-1", IdempotencyKey: "setup-1", Status: "validated",
			},
		},
	}
	if err := SaveProjectConfig(path, config); err != nil {
		t.Fatal(err)
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	if !strings.Contains(text, "sync:") || !strings.Contains(text, "# existing settings") {
		t.Fatalf("unrelated project settings were not preserved:\n%s", text)
	}

	loaded, err := LoadProjectConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Environments["production"].ManagedDomain != "quiet-river.mycube.dev" {
		t.Fatalf("setup environment did not round-trip: %#v", loaded.Environments["production"])
	}
}
