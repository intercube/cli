package cmd

import (
	"testing"

	setupapi "github.com/intercube/cli/util/setup"
	"github.com/spf13/cobra"
)

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

func TestPreparedSetupOptionsChanged(t *testing.T) {
	config := &setupapi.ProjectConfig{Repository: setupapi.ProjectRepository{Directory: "wordpress"}}
	environment := setupapi.Environment{
		Branch: "main", Domain: "shop.example.com", NoCharge: false,
		Prepared: &setupapi.PreparedSetup{PlanKey: "medium", ServerID: 0},
	}

	matching := setupOptionsTestCommand(t, map[string]string{
		"branch": "main", "domain": "shop.example.com", "directory": "wordpress", "plan": "medium",
	})
	if preparedSetupOptionsChanged(matching, config, environment) {
		t.Fatal("matching setup options should resume the prepared intent")
	}

	changed := setupOptionsTestCommand(t, map[string]string{"domain": "new.example.com"})
	if !preparedSetupOptionsChanged(changed, config, environment) {
		t.Fatal("a changed setup option should prepare a new quote")
	}
}

func setupOptionsTestCommand(t *testing.T, values map[string]string) *cobra.Command {
	t.Helper()
	command := &cobra.Command{}
	command.Flags().String("branch", "", "")
	command.Flags().String("domain", "", "")
	command.Flags().String("directory", "", "")
	command.Flags().String("plan", "", "")
	command.Flags().Int("server", 0, "")
	command.Flags().Bool("no-charge", false, "")

	setupBranch = values["branch"]
	setupDomain = values["domain"]
	setupDirectory = values["directory"]
	setupPlan = values["plan"]
	for name, value := range values {
		if err := command.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	return command
}
