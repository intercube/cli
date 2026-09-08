package cmd

import (
	"testing"

	"github.com/intercube/cli/util/inventory"
	"github.com/intercube/cli/util/pipeline"
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

func TestSetupShortFlagsAndSelectorOnlyExistingSite(t *testing.T) {
	tests := map[string]string{
		"environment":   "e",
		"branch":        "b",
		"directory":     "d",
		"plan":          "p",
		"organization":  "o",
		"existing-site": "s",
		"yes":           "y",
	}
	for name, shorthand := range tests {
		flag := setupCmd.Flags().Lookup(name)
		if flag == nil {
			t.Fatalf("--%s is not registered", name)
		}
		if flag.Shorthand != shorthand {
			t.Fatalf("--%s shorthand = %q, want %q", name, flag.Shorthand, shorthand)
		}
	}
	if setupCmd.Flags().Lookup("site") != nil {
		t.Fatal("setup must not expose a direct --site flag; existing sites are selected interactively")
	}
}

func TestExistingSiteOptionMatchesSelectorContext(t *testing.T) {
	option := existingSiteOption{
		Site: inventory.SiteServer{
			ID:           "456",
			Username:     "landing",
			MainDomain:   "landing.example.com",
			IsProduction: true,
			ServerID:     "73",
			ServerName:   "web-production-01",
		},
		Meta: "production · web-production-01 · site #456",
	}
	for _, search := range []string{"landing.example", "LANDING", "production", "web-production", "456", "73"} {
		if !existingSiteOptionMatches(option, search) {
			t.Fatalf("expected selector search %q to match", search)
		}
	}
	if existingSiteOptionMatches(option, "staging") {
		t.Fatal("unexpected selector match")
	}
}

func TestProjectForSiteFindsExistingPipelineAssignment(t *testing.T) {
	assignedSiteID := 456
	projects := []pipeline.Project{{ID: 12, Name: "Landing", SiteID: &assignedSiteID}}
	if project := projectForSite(projects, "456"); project == nil || project.ID != 12 {
		t.Fatalf("expected assigned project, got %+v", project)
	}
	if project := projectForSite(projects, "999"); project != nil {
		t.Fatalf("expected no project, got %+v", project)
	}
}

func TestWorkflowTemplateForAnalysisUsesLatestDetectedWorkflow(t *testing.T) {
	templates := []pipeline.WorkflowTemplate{
		{ID: 1, Key: "wordpress", FrameworkKey: "wordpress", Version: "1.0.0", Latest: false},
		{ID: 2, Key: "wordpress", FrameworkKey: "wordpress", Version: "2.0.0", Latest: true},
		{ID: 3, Key: "wordpress", FrameworkKey: "wordpress", Version: "3.0.0", Latest: true, Deprecated: true},
	}
	template, err := workflowTemplateForAnalysis(templates, "wordpress", "wordpress")
	if err != nil {
		t.Fatal(err)
	}
	if template.ID != 2 {
		t.Fatalf("template id = %d, want 2", template.ID)
	}
}
