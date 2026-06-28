package cmd

import (
	"fmt"

	"github.com/intercube/cli/util/inventory"
	"github.com/spf13/cobra"
)

func resolveSiteSelection(cmd *cobra.Command, inventoryClient *inventory.Client, siteID string) (*inventory.SiteServer, error) {
	sites, err := inventoryClient.ListSites(cmd.Context())
	if err != nil {
		if shouldPromptForOrganization(err) {
			return nil, fmt.Errorf("organization context is required. Run `intercube auth org` (or pass --org-id)")
		}

		return nil, err
	}

	if len(sites) == 0 {
		return nil, fmt.Errorf("no sites available for your account")
	}

	if siteID != "" {
		selected, found := findSiteByID(sites, siteID)
		if !found {
			return nil, fmt.Errorf("site %q not found", siteID)
		}

		return selected, nil
	}

	return selectSite(sites)
}
