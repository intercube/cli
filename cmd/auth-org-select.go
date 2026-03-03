package cmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/intercube/cli/util/appconfig"
	authutil "github.com/intercube/cli/util/auth"
	"github.com/intercube/cli/util/inventory"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var orgIDInput string

var authOrgSelectCmd = &cobra.Command{
	Use:   "select [org_id]",
	Short: "Select organization id for API calls",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuthOrgSelect(cmd, args, false)
	},
}

func runAuthOrgSelect(cmd *cobra.Command, args []string, forcePrompt bool) error {
	store, err := authutil.NewSessionStore("intercube-cli")
	if err != nil {
		return err
	}

	session, err := store.Load(cmd.Context())
	if err != nil {
		if errors.Is(err, authutil.ErrNoSession) {
			return errors.New("you are not authenticated, run `intercube auth login` first")
		}

		return err
	}

	orgID := strings.TrimSpace(orgIDInput)
	if len(args) > 0 {
		orgID = strings.TrimSpace(args[0])
	}

	organizations, orgErr := listOrganizationsForSelection(cmd, store)
	if orgErr == nil {
		for _, organization := range organizations {
			session.KnownOrgIDs = addKnownOrganizationID(session.KnownOrgIDs, organization.ID)
		}
	} else if forcePrompt {
		fmt.Printf("Could not load organizations automatically: %v\n", orgErr)
	}

	if forcePrompt || orgID == "" {
		orgID, err = selectOrPromptOrgID(session.OrganizationID, appconfig.OrganizationID, session.KnownOrgIDs, organizations, forcePrompt)
		if err != nil {
			return err
		}
	} else if len(organizations) > 0 && !organizationIDExists(organizations, orgID) {
		return fmt.Errorf("organization %q is not available for your account", orgID)
	}

	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return fmt.Errorf("organization id is required")
	}

	session.OrganizationID = orgID
	session.KnownOrgIDs = addKnownOrganizationID(session.KnownOrgIDs, orgID)
	if appconfig.OrganizationID != "" {
		session.KnownOrgIDs = addKnownOrganizationID(session.KnownOrgIDs, appconfig.OrganizationID)
	}

	if err := store.Save(cmd.Context(), session); err != nil {
		return err
	}

	fmt.Printf("Selected organization: %s\n", orgID)
	return nil
}

func init() {
	authOrgCmd.AddCommand(authOrgSelectCmd)
	authOrgSelectCmd.Flags().StringVar(&orgIDInput, "org-id", "", "organization id to set")
}

func selectOrPromptOrgID(currentValue, configuredValue string, knownOrgIDs []string, organizations []inventory.CurrentUserOrganization, forcePrompt bool) (string, error) {
	if len(organizations) > 0 {
		return selectOrganizationFromMemberships(currentValue, organizations)
	}

	candidates := organizationCandidates(currentValue, configuredValue, knownOrgIDs)
	if len(candidates) == 0 {
		return promptOrgID(currentValue)
	}

	if !forcePrompt && len(candidates) == 1 && strings.TrimSpace(currentValue) == "" {
		return strings.TrimSpace(candidates[0]), nil
	}

	items := append([]string{}, candidates...)
	items = append(items, "Enter organization ID manually")

	prompt := promptui.Select{
		Label:     "Select organization",
		Items:     items,
		Size:      selectSize(len(items)),
		Stdout:    &bellSkipper{},
		Templates: simpleSelectTemplates("organization"),
	}

	index, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	if index == len(items)-1 {
		return promptOrgID(currentValue)
	}

	return strings.TrimSpace(items[index]), nil
}

func organizationCandidates(currentValue, configuredValue string, knownOrgIDs []string) []string {
	result := make([]string, 0, len(knownOrgIDs)+2)
	seen := make(map[string]struct{})

	appendIfUnique := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}

		if _, ok := seen[trimmed]; ok {
			return
		}

		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	appendIfUnique(currentValue)
	for _, candidate := range knownOrgIDs {
		appendIfUnique(candidate)
	}
	appendIfUnique(configuredValue)

	return result
}

func addKnownOrganizationID(values []string, orgID string) []string {
	trimmed := strings.TrimSpace(orgID)
	if trimmed == "" {
		return values
	}

	for _, value := range values {
		if strings.TrimSpace(value) == trimmed {
			return values
		}
	}

	return append(values, trimmed)
}

func listOrganizationsForSelection(cmd *cobra.Command, store *authutil.SessionStore) ([]inventory.CurrentUserOrganization, error) {
	if err := appconfig.ValidateInventory(); err != nil {
		return nil, err
	}

	if err := appconfig.ValidateClerk(); err != nil {
		return nil, err
	}

	clerkClient := &authutil.ClerkClient{
		Issuer:       appconfig.ClerkIssuer,
		ClientID:     appconfig.ClerkClientID,
		Audience:     appconfig.ClerkAudience,
		Scopes:       appconfig.ClerkScopes,
		CallbackPort: appconfig.ParsedCallbackPort(),
	}

	inventoryClient := inventory.NewClient(appconfig.InventoryAPIBaseURL, "", store, clerkClient)
	return inventoryClient.ListCurrentUserOrganizations(cmd.Context())
}

func selectOrganizationFromMemberships(currentValue string, organizations []inventory.CurrentUserOrganization) (string, error) {
	if len(organizations) == 1 {
		return strings.TrimSpace(organizations[0].ID), nil
	}

	sort.SliceStable(organizations, func(i, j int) bool {
		return organizationLabel(organizations[i]) < organizationLabel(organizations[j])
	})

	items := make([]string, 0, len(organizations))
	defaultIndex := 0
	for i, organization := range organizations {
		items = append(items, organizationLabel(organization))
		if strings.EqualFold(strings.TrimSpace(currentValue), strings.TrimSpace(organization.ID)) {
			defaultIndex = i
		}
	}

	prompt := promptui.Select{
		Label:     "Select organization",
		Items:     items,
		Size:      selectSize(len(items)),
		CursorPos: defaultIndex,
		Stdout:    &bellSkipper{},
		Templates: simpleSelectTemplates("organization"),
	}

	index, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(organizations[index].ID), nil
}

func organizationIDExists(organizations []inventory.CurrentUserOrganization, organizationID string) bool {
	needle := strings.TrimSpace(organizationID)
	for _, organization := range organizations {
		if strings.EqualFold(strings.TrimSpace(organization.ID), needle) {
			return true
		}
	}

	return false
}

func organizationLabel(organization inventory.CurrentUserOrganization) string {
	parts := make([]string, 0, 3)
	if strings.TrimSpace(organization.Name) != "" {
		parts = append(parts, organization.Name)
	}
	if strings.TrimSpace(organization.Slug) != "" {
		parts = append(parts, organization.Slug)
	}
	if strings.TrimSpace(organization.Role) != "" {
		parts = append(parts, organization.Role)
	}

	prefix := strings.Join(parts, " | ")
	if prefix == "" {
		prefix = "Organization"
	}

	return fmt.Sprintf("%s (%s)", prefix, organization.ID)
}

func promptOrgID(defaultValue string) (string, error) {
	prompt := promptui.Prompt{
		Label:   "Organization ID",
		Default: strings.TrimSpace(defaultValue),
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("organization id is required")
			}

			return nil
		},
		Stdout: &bellSkipper{},
	}

	value, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(value), nil
}
