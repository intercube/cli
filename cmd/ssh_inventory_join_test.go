package cmd

import (
	"testing"

	"github.com/hashicorp/boundary/api/hosts"
	"github.com/intercube/cli/util/inventory"
)

func TestBuildSSHTargetOptionsJoinsSharedServerSites(t *testing.T) {
	hostsList := []*hosts.Host{
		{Id: "h-1", Name: "shared.scary-fine-eggplant.intercube.cloud"},
		{Id: "h-2", Name: "dedicated.calm-blue-berry.intercube.cloud"},
	}

	sites := []inventory.SiteServer{
		{ID: "s-1", MainDomain: "shop.example.com", ServerName: "shared.scary-fine-eggplant.intercube.cloud", Username: "shop"},
		{ID: "s-2", MainDomain: "blog.example.com", ServerName: "shared", Username: "blog"},
	}

	options := buildSSHTargetOptions(hostsList, sites)
	if len(options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(options))
	}

	var sharedOption sshTargetOption
	for _, option := range options {
		if option.HostID == "h-1" {
			sharedOption = option
			break
		}
	}

	if sharedOption.HostID == "" {
		t.Fatalf("expected shared host option to exist")
	}

	if sharedOption.JoinStatus != "inventory_enriched" {
		t.Fatalf("expected inventory_enriched, got %q", sharedOption.JoinStatus)
	}

	if sharedOption.SitePreview != "blog.example.com, shop.example.com" {
		t.Fatalf("unexpected site preview: %q", sharedOption.SitePreview)
	}

	if sharedOption.Meta != "blog.example.com (+1 more)" {
		t.Fatalf("unexpected meta label: %q", sharedOption.Meta)
	}
}

func TestSummarizeSiteLabelsTruncates(t *testing.T) {
	labels := []string{"a", "b", "c", "d"}
	value := summarizeSiteLabels(labels, 2)
	if value != "a, b (+2 more)" {
		t.Fatalf("expected truncated summary, got %q", value)
	}
}

func TestBuildMetaLabelAvoidsDuplicateTitle(t *testing.T) {
	label := buildMetaLabel("api.beta.intercube.dev", []string{"api.beta.intercube.dev", "shop.example.com"})
	if label != "2 sites" {
		t.Fatalf("expected aggregated site count label, got %q", label)
	}
}

func TestBuildSSHTargetOptionsKeepsBoundaryOnlyHosts(t *testing.T) {
	hostsList := []*hosts.Host{{Id: "h-1", Name: "unmatched.intercube.cloud"}}

	options := buildSSHTargetOptions(hostsList, nil)
	if len(options) != 1 {
		t.Fatalf("expected 1 option, got %d", len(options))
	}

	option := options[0]
	if option.JoinStatus != "boundary_only" {
		t.Fatalf("expected boundary_only, got %q", option.JoinStatus)
	}

	if option.Meta != "" {
		t.Fatalf("expected empty meta for boundary-only host, got %q", option.Meta)
	}
}

func TestBuildSSHTargetOptionsDedupesSameHostName(t *testing.T) {
	hostsList := []*hosts.Host{
		{Id: "h-1", Name: "api.beta.intercube.dev"},
		{Id: "h-2", Name: "api.beta.intercube.dev"},
		{Id: "h-3", Name: "other.intercube.dev"},
	}

	sites := []inventory.SiteServer{{ID: "s-1", MainDomain: "api.beta.intercube.dev", ServerName: "api.beta.intercube.dev"}}

	options := buildSSHTargetOptions(hostsList, sites)
	if len(options) != 2 {
		t.Fatalf("expected 2 deduped options, got %d", len(options))
	}
}
