package cmd

import "testing"

func TestFilterAndRankSSHTargetsMatchesSiteAndServer(t *testing.T) {
	options := []sshTargetOption{
		{
			HostID:       "h-1",
			HostName:     "shared.scary-fine-eggplant.intercube.cloud",
			ServerName:   "shared.scary-fine-eggplant.intercube.cloud",
			Title:        "shared.scary-fine-eggplant.intercube.cloud",
			Meta:         "shop.example.com",
			SitePreview:  "shop.example.com, blog.example.com",
			SearchFields: []string{"sharedscaryfineeggplantintercubecloud", "shopexamplecom", "blogexamplecom"},
		},
		{
			HostID:       "h-2",
			HostName:     "other-host.intercube.cloud",
			ServerName:   "other-host",
			Title:        "other-host",
			Meta:         "other.example.com",
			SitePreview:  "other.example.com",
			SearchFields: []string{"otherhostintercubecloud", "otherexamplecom"},
		},
	}

	siteMatches := filterAndRankSSHTargets(options, "shop.example.com")
	if len(siteMatches) != 1 || siteMatches[0].HostID != "h-1" {
		t.Fatalf("expected site query to match h-1, got %+v", siteMatches)
	}

	serverMatches := filterAndRankSSHTargets(options, "shared.scary")
	if len(serverMatches) != 1 || serverMatches[0].HostID != "h-1" {
		t.Fatalf("expected server query to match h-1, got %+v", serverMatches)
	}

	noMatches := filterAndRankSSHTargets(options, "missing.example.com")
	if len(noMatches) != 0 {
		t.Fatalf("expected no matches for missing query, got %+v", noMatches)
	}
}

func TestSSHTokenizeSearch(t *testing.T) {
	tokens := sshTokenizeSearch("Shared.scary-fine_eggplant")
	if len(tokens) != 4 {
		t.Fatalf("expected 4 tokens, got %d (%v)", len(tokens), tokens)
	}

	expected := []string{"shared", "scary", "fine", "eggplant"}
	for i := range expected {
		if tokens[i] != expected[i] {
			t.Fatalf("expected token %q at index %d, got %q", expected[i], i, tokens[i])
		}
	}
}
