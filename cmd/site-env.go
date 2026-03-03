package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/intercube/cli/util/inventory"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	siteEnvListSiteID   string
	siteEnvSetSiteID    string
	siteEnvSetName      string
	siteEnvSetValue     string
	siteEnvSetSecret    bool
	siteEnvGetSiteID    string
	siteEnvGetName      string
	siteEnvGetID        string
	siteEnvDeleteSiteID string
	siteEnvDeleteName   string
	siteEnvDeleteID     string
	siteEnvDeleteYes    bool
)

var siteEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage site environment variables",
}

var siteEnvListCmd = &cobra.Command{
	Use:   "list",
	Short: "List environment variables for a site",
	RunE: func(cmd *cobra.Command, args []string) error {
		inventoryClient, _, err := newInventoryClient(cmd, "")
		if err != nil {
			return err
		}

		site, err := resolveSiteSelection(cmd, inventoryClient, siteEnvListSiteID)
		if err != nil {
			return err
		}

		variables, err := inventoryClient.ListSiteEnvironmentVariables(cmd.Context(), site.ID)
		if err != nil {
			return err
		}

		if len(variables) == 0 {
			fmt.Printf("No environment variables found for site %s (%s)\n", siteDisplayName(*site), site.ID)
			return nil
		}

		sort.SliceStable(variables, func(i, j int) bool {
			return strings.ToLower(variables[i].Name) < strings.ToLower(variables[j].Name)
		})

		for _, variable := range variables {
			secretLabel := "plain"
			if variable.Secret {
				secretLabel = "secret"
			}

			fmt.Printf("%s=%s [%s] (id: %s)\n", variable.Name, variable.Value, secretLabel, variable.ID)
		}

		return nil
	},
}

var siteEnvSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Create or update a site environment variable",
	RunE: func(cmd *cobra.Command, args []string) error {
		inventoryClient, _, err := newInventoryClient(cmd, "")
		if err != nil {
			return err
		}

		site, err := resolveSiteSelection(cmd, inventoryClient, siteEnvSetSiteID)
		if err != nil {
			return err
		}

		name := strings.TrimSpace(siteEnvSetName)
		if name == "" {
			name, err = promptRequiredText("Environment variable name", "")
			if err != nil {
				return err
			}
		}

		value := siteEnvSetValue
		if value == "" {
			value, err = promptRequiredText("Environment variable value", "")
			if err != nil {
				return err
			}
		}

		variables, err := inventoryClient.ListSiteEnvironmentVariables(cmd.Context(), site.ID)
		if err != nil {
			return err
		}

		existing := findEnvironmentVariableByName(variables, name)
		request := inventory.EnvironmentVariableMutate{Name: name, Value: value, Secret: siteEnvSetSecret}

		if existing == nil {
			created, createErr := inventoryClient.CreateSiteEnvironmentVariable(cmd.Context(), site.ID, request)
			if createErr != nil {
				return createErr
			}

			fmt.Printf("Created %s on site %s (%s)\n", created.Name, siteDisplayName(*site), site.ID)
			return nil
		}

		if err := inventoryClient.UpdateSiteEnvironmentVariable(cmd.Context(), site.ID, existing.ID, request); err != nil {
			return err
		}

		fmt.Printf("Updated %s on site %s (%s)\n", existing.Name, siteDisplayName(*site), site.ID)
		return nil
	},
}

var siteEnvGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a site environment variable",
	RunE: func(cmd *cobra.Command, args []string) error {
		inventoryClient, _, err := newInventoryClient(cmd, "")
		if err != nil {
			return err
		}

		site, err := resolveSiteSelection(cmd, inventoryClient, siteEnvGetSiteID)
		if err != nil {
			return err
		}

		variables, err := inventoryClient.ListSiteEnvironmentVariables(cmd.Context(), site.ID)
		if err != nil {
			return err
		}

		if len(variables) == 0 {
			return fmt.Errorf("no environment variables found for site %s (%s)", siteDisplayName(*site), site.ID)
		}

		selected, err := resolveEnvironmentVariableSelection(variables, siteEnvGetID, siteEnvGetName)
		if err != nil {
			return err
		}

		secretLabel := "plain"
		if selected.Secret {
			secretLabel = "secret"
		}

		fmt.Printf("%s=%s [%s] (id: %s)\n", selected.Name, selected.Value, secretLabel, selected.ID)
		return nil
	},
}

var siteEnvDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a site environment variable",
	RunE: func(cmd *cobra.Command, args []string) error {
		inventoryClient, _, err := newInventoryClient(cmd, "")
		if err != nil {
			return err
		}

		site, err := resolveSiteSelection(cmd, inventoryClient, siteEnvDeleteSiteID)
		if err != nil {
			return err
		}

		variables, err := inventoryClient.ListSiteEnvironmentVariables(cmd.Context(), site.ID)
		if err != nil {
			return err
		}

		if len(variables) == 0 {
			return fmt.Errorf("no environment variables found for site %s (%s)", siteDisplayName(*site), site.ID)
		}

		selected, err := resolveEnvironmentVariableSelection(variables, siteEnvDeleteID, siteEnvDeleteName)
		if err != nil {
			return err
		}

		if !siteEnvDeleteYes {
			confirmed, confirmErr := promptYesNo(fmt.Sprintf("Delete %s from %s?", selected.Name, siteDisplayName(*site)))
			if confirmErr != nil {
				return confirmErr
			}

			if !confirmed {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		if err := inventoryClient.DeleteSiteEnvironmentVariable(cmd.Context(), site.ID, selected.ID); err != nil {
			return err
		}

		fmt.Printf("Deleted %s from site %s (%s)\n", selected.Name, siteDisplayName(*site), site.ID)
		return nil
	},
}

func init() {
	siteCmd.AddCommand(siteEnvCmd)
	siteEnvCmd.AddCommand(siteEnvListCmd)
	siteEnvCmd.AddCommand(siteEnvSetCmd)
	siteEnvCmd.AddCommand(siteEnvGetCmd)
	siteEnvCmd.AddCommand(siteEnvDeleteCmd)

	siteEnvListCmd.Flags().StringVar(&siteEnvListSiteID, "site-id", "", "site id")

	siteEnvSetCmd.Flags().StringVar(&siteEnvSetSiteID, "site-id", "", "site id")
	siteEnvSetCmd.Flags().StringVar(&siteEnvSetName, "name", "", "environment variable name")
	siteEnvSetCmd.Flags().StringVar(&siteEnvSetValue, "value", "", "environment variable value")
	siteEnvSetCmd.Flags().BoolVar(&siteEnvSetSecret, "secret", false, "mark environment variable as secret")

	siteEnvGetCmd.Flags().StringVar(&siteEnvGetSiteID, "site-id", "", "site id")
	siteEnvGetCmd.Flags().StringVar(&siteEnvGetName, "name", "", "environment variable name")
	siteEnvGetCmd.Flags().StringVar(&siteEnvGetID, "id", "", "environment variable id")

	siteEnvDeleteCmd.Flags().StringVar(&siteEnvDeleteSiteID, "site-id", "", "site id")
	siteEnvDeleteCmd.Flags().StringVar(&siteEnvDeleteName, "name", "", "environment variable name")
	siteEnvDeleteCmd.Flags().StringVar(&siteEnvDeleteID, "id", "", "environment variable id")
	siteEnvDeleteCmd.Flags().BoolVar(&siteEnvDeleteYes, "yes", false, "delete without confirmation")
}

func resolveEnvironmentVariableSelection(variables []inventory.EnvironmentVariable, variableID, variableName string) (*inventory.EnvironmentVariable, error) {
	id := strings.TrimSpace(variableID)
	if id != "" {
		for i := range variables {
			if strings.EqualFold(strings.TrimSpace(variables[i].ID), id) {
				return &variables[i], nil
			}
		}

		return nil, fmt.Errorf("environment variable id %q not found", variableID)
	}

	name := strings.TrimSpace(variableName)
	if name != "" {
		candidate := findEnvironmentVariableByName(variables, name)
		if candidate == nil {
			return nil, fmt.Errorf("environment variable %q not found", variableName)
		}

		return candidate, nil
	}

	return selectEnvironmentVariable(variables)
}

func findEnvironmentVariableByName(variables []inventory.EnvironmentVariable, name string) *inventory.EnvironmentVariable {
	needle := strings.TrimSpace(name)
	for i := range variables {
		if strings.EqualFold(strings.TrimSpace(variables[i].Name), needle) {
			return &variables[i]
		}
	}

	return nil
}

func selectEnvironmentVariable(variables []inventory.EnvironmentVariable) (*inventory.EnvironmentVariable, error) {
	if len(variables) == 1 {
		return &variables[0], nil
	}

	sort.SliceStable(variables, func(i, j int) bool {
		return strings.ToLower(variables[i].Name) < strings.ToLower(variables[j].Name)
	})

	type variableChoice struct {
		Variable inventory.EnvironmentVariable
		Title    string
		Meta     string
	}

	items := make([]variableChoice, 0, len(variables))
	for _, variable := range variables {
		items = append(items, variableChoice{
			Variable: variable,
			Title:    strings.TrimSpace(variable.Name),
			Meta:     strings.TrimSpace(variable.Value),
		})
	}

	prompt := promptui.Select{
		Label:     "Select environment variable",
		Items:     items,
		Size:      selectSize(len(items)),
		Stdout:    &bellSkipper{},
		Templates: titleMetaSelectTemplates("variable"),
	}

	index, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	selected := items[index].Variable
	return &selected, nil
}

func promptRequiredText(label, defaultValue string) (string, error) {
	prompt := promptui.Prompt{
		Label:   label,
		Default: defaultValue,
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("value is required")
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

func promptYesNo(label string) (bool, error) {
	prompt := promptui.Select{
		Label:     label,
		Items:     []string{"Yes", "No"},
		Size:      2,
		Stdout:    &bellSkipper{},
		Templates: simpleSelectTemplates("option"),
	}

	index, _, err := prompt.Run()
	if err != nil {
		return false, err
	}

	return index == 0, nil
}
