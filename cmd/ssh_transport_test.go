package cmd

import (
	"github.com/intercube/cli/util/inventory"
	"strings"
	"testing"
)

func TestCustomerTargetSelectionNeverFallsBackToRoot(t *testing.T) {
	id := 42
	targets := []inventory.SSHTarget{
		{Name: "nexus-site-42", SiteID: &id, ServerID: 7, ServerName: "shared-web", LinuxUser: "shop_a", OrganizationID: "org_a", Domains: []string{"a.example.com"}},
		{Name: "nexus-site-43", ServerID: 7, ServerName: "shared-web", LinuxUser: "shop_b", OrganizationID: "org_b"},
		{Name: "nexus-root-7", ServerID: 7, ServerName: "shared-web", LinuxUser: "root", Root: true},
	}
	if got := filterGatewayTargets(targets, "shared-web", false, ""); len(got) != 2 {
		t.Fatalf("expected two sites, got %#v", got)
	}
	if got := filterGatewayTargets(targets, "a.example.com", false, ""); len(got) != 1 || got[0].LinuxUser != "shop_a" {
		t.Fatalf("wrong site: %#v", got)
	}
	if got := filterGatewayTargets(targets, "shared-web", true, ""); len(got) != 1 || !got[0].Root {
		t.Fatalf("wrong root selection: %#v", got)
	}
}

func TestRsyncKeepsGatewayColonOutOfDestination(t *testing.T) {
	connection := gatewayConnection(inventory.SSHAccess{Host: "access.example.com", Port: 2222, Username: "nexus-user_test"}, inventory.SSHTarget{Name: "nexus-site-42"})
	socket, err := gatewayControlPath(connection)
	if err != nil {
		t.Fatal(err)
	}
	args := gatewaySSHArgs(connection, socket)
	found := false
	for i, arg := range args {
		if arg == "-l" && i+1 < len(args) && args[i+1] == "nexus-user_test:nexus-site-42" {
			found = true
		}
	}
	if !found {
		t.Fatalf("composite username must be passed as a separate -l argument: %#v", args)
	}
	if !strings.Contains(quoteCommandArgs(args), "'nexus-user_test:nexus-site-42'") {
		t.Fatal("rsync transport must quote the gateway username")
	}
}
