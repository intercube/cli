package cmd

import (
	"fmt"
	"strings"

	"github.com/intercube/cli/util/appconfig"
	authutil "github.com/intercube/cli/util/auth"
	"github.com/intercube/cli/util/inventory"
	"github.com/spf13/cobra"
)

func newInventoryClient(cmd *cobra.Command, organizationOverride string) (*inventory.Client, string, error) {
	appconfig.LoadFromEnv()

	if err := appconfig.ValidateClerk(); err != nil {
		return nil, "", fmt.Errorf("%w (set via env/.env or build-time)", err)
	}

	if err := appconfig.ValidateInventory(); err != nil {
		return nil, "", fmt.Errorf("%w (set via env/.env or build-time)", err)
	}

	store, err := authutil.NewSessionStore("intercube-cli")
	if err != nil {
		return nil, "", err
	}

	organizationID := strings.TrimSpace(organizationOverride)
	if organizationID == "" {
		session, sessionErr := store.Load(cmd.Context())
		if sessionErr == nil {
			organizationID = strings.TrimSpace(session.OrganizationID)
		}
	}

	if organizationID == "" {
		organizationID = strings.TrimSpace(appconfig.OrganizationID)
	}

	clerkClient := &authutil.ClerkClient{
		Issuer:       appconfig.ClerkIssuer,
		ClientID:     appconfig.ClerkClientID,
		Audience:     appconfig.ClerkAudience,
		Scopes:       appconfig.ClerkScopes,
		CallbackPort: appconfig.ParsedCallbackPort(),
	}

	return inventory.NewClient(appconfig.InventoryAPIBaseURL, organizationID, store, clerkClient), organizationID, nil
}

func findSiteByID(sites []inventory.SiteServer, siteID string) (*inventory.SiteServer, bool) {
	needle := strings.TrimSpace(siteID)
	for i := range sites {
		if strings.EqualFold(strings.TrimSpace(sites[i].ID), needle) {
			return &sites[i], true
		}
	}

	return nil, false
}
