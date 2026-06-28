package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intercube/cli/util/inventory"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	orgSSHKeyOrgID             string
	orgSSHKeyCreatePath        string
	orgSSHKeyCreateComment     string
	orgSSHKeyCreateExpiration  string
	orgSSHKeyAssignKeyID       string
	orgSSHKeyAssignSiteID      string
	orgSSHKeyUnassignKeyID     string
	orgSSHKeyUnassignSiteID    string
	orgSSHKeyUnassignDeleteKey bool
)

var orgSSHKeyCmd = &cobra.Command{
	Use:   "ssh-key",
	Short: "Manage organization SSH key vault",
}

var orgSSHKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List organization SSH keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		inventoryClient, _, err := newInventoryClient(cmd, orgSSHKeyOrgID)
		if err != nil {
			return err
		}

		keys, err := inventoryClient.ListOrganizationSSHKeys(cmd.Context())
		if err != nil {
			if shouldPromptForOrganization(err) {
				return fmt.Errorf("organization context is required. Run `intercube auth org` (or pass --org-id)")
			}

			return err
		}

		if len(keys) == 0 {
			fmt.Println("No organization SSH keys found.")
			return nil
		}

		sort.SliceStable(keys, func(i, j int) bool {
			return strings.TrimSpace(keys[i].ID) < strings.TrimSpace(keys[j].ID)
		})

		for _, key := range keys {
			comment := strings.TrimSpace(key.Comment)
			if comment == "" {
				comment = "no comment"
			}

			fmt.Printf("%s | %s | assigned sites: %d\n", key.ID, comment, len(key.SiteIDs))
		}

		return nil
	},
}

var orgSSHKeyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create organization SSH key from local public key",
	RunE: func(cmd *cobra.Command, args []string) error {
		inventoryClient, _, err := newInventoryClient(cmd, orgSSHKeyOrgID)
		if err != nil {
			return err
		}

		selectedKey, err := resolveLocalPublicKeySelection(orgSSHKeyCreatePath)
		if err != nil {
			return err
		}

		comment := strings.TrimSpace(orgSSHKeyCreateComment)
		if comment == "" {
			comment = strings.TrimSpace(selectedKey.Comment)
			if comment == "" {
				comment = selectedKey.Name
			}
		}

		keys, err := inventoryClient.ListOrganizationSSHKeys(cmd.Context())
		if err != nil {
			if shouldPromptForOrganization(err) {
				return fmt.Errorf("organization context is required. Run `intercube auth org` (or pass --org-id)")
			}

			return err
		}

		for _, key := range keys {
			if strings.TrimSpace(key.Content) == strings.TrimSpace(selectedKey.Content) {
				fmt.Printf("SSH key already exists in organization vault: %s\n", key.ID)
				return nil
			}
		}

		created, err := inventoryClient.CreateOrganizationSSHKey(cmd.Context(), inventory.OrganizationSSHKeyRequest{
			Content:        selectedKey.Content,
			Comment:        comment,
			ExpirationDate: strings.TrimSpace(orgSSHKeyCreateExpiration),
		})
		if err != nil {
			return err
		}

		fmt.Printf("Created organization SSH key %s\n", created.ID)
		return nil
	},
}

var orgSSHKeyAssignCmd = &cobra.Command{
	Use:   "assign",
	Short: "Assign organization SSH key to a site",
	RunE: func(cmd *cobra.Command, args []string) error {
		inventoryClient, _, err := newInventoryClient(cmd, orgSSHKeyOrgID)
		if err != nil {
			return err
		}

		keys, err := inventoryClient.ListOrganizationSSHKeys(cmd.Context())
		if err != nil {
			if shouldPromptForOrganization(err) {
				return fmt.Errorf("organization context is required. Run `intercube auth org` (or pass --org-id)")
			}

			return err
		}

		if len(keys) == 0 {
			return fmt.Errorf("no organization SSH keys found")
		}

		selectedKey, err := resolveOrganizationSSHKeySelection(keys, orgSSHKeyAssignKeyID)
		if err != nil {
			return err
		}

		site, err := resolveSiteSelection(cmd, inventoryClient, orgSSHKeyAssignSiteID)
		if err != nil {
			return err
		}

		if err := inventoryClient.AssignOrganizationSSHKeyToSite(cmd.Context(), selectedKey.ID, site.ID); err != nil {
			return err
		}

		fmt.Printf("Assigned key %s to site %s (%s)\n", selectedKey.ID, siteDisplayName(*site), site.ID)
		return nil
	},
}

var orgSSHKeyUnassignCmd = &cobra.Command{
	Use:   "unassign",
	Short: "Unassign organization SSH key from a site",
	RunE: func(cmd *cobra.Command, args []string) error {
		inventoryClient, _, err := newInventoryClient(cmd, orgSSHKeyOrgID)
		if err != nil {
			return err
		}

		keys, err := inventoryClient.ListOrganizationSSHKeys(cmd.Context())
		if err != nil {
			if shouldPromptForOrganization(err) {
				return fmt.Errorf("organization context is required. Run `intercube auth org` (or pass --org-id)")
			}

			return err
		}

		if len(keys) == 0 {
			return fmt.Errorf("no organization SSH keys found")
		}

		selectedKey, err := resolveOrganizationSSHKeySelection(keys, orgSSHKeyUnassignKeyID)
		if err != nil {
			return err
		}

		site, err := resolveSiteSelection(cmd, inventoryClient, orgSSHKeyUnassignSiteID)
		if err != nil {
			return err
		}

		if err := inventoryClient.UnassignOrganizationSSHKeyFromSite(cmd.Context(), selectedKey.ID, site.ID, orgSSHKeyUnassignDeleteKey); err != nil {
			return err
		}

		fmt.Printf("Unassigned key %s from site %s (%s)\n", selectedKey.ID, siteDisplayName(*site), site.ID)
		return nil
	},
}

func init() {
	orgCmd.AddCommand(orgSSHKeyCmd)
	orgSSHKeyCmd.AddCommand(orgSSHKeyListCmd)
	orgSSHKeyCmd.AddCommand(orgSSHKeyCreateCmd)
	orgSSHKeyCmd.AddCommand(orgSSHKeyAssignCmd)
	orgSSHKeyCmd.AddCommand(orgSSHKeyUnassignCmd)

	orgSSHKeyCmd.PersistentFlags().StringVar(&orgSSHKeyOrgID, "org-id", "", "organization id")

	orgSSHKeyCreateCmd.Flags().StringVar(&orgSSHKeyCreatePath, "path", "", "path to SSH public key file")
	orgSSHKeyCreateCmd.Flags().StringVar(&orgSSHKeyCreateComment, "comment", "", "comment for the key")
	orgSSHKeyCreateCmd.Flags().StringVar(&orgSSHKeyCreateExpiration, "expiration-date", "", "expiration date (YYYY-MM-DD)")

	orgSSHKeyAssignCmd.Flags().StringVar(&orgSSHKeyAssignKeyID, "key-id", "", "organization SSH key id")
	orgSSHKeyAssignCmd.Flags().StringVar(&orgSSHKeyAssignSiteID, "site-id", "", "site id")

	orgSSHKeyUnassignCmd.Flags().StringVar(&orgSSHKeyUnassignKeyID, "key-id", "", "organization SSH key id")
	orgSSHKeyUnassignCmd.Flags().StringVar(&orgSSHKeyUnassignSiteID, "site-id", "", "site id")
	orgSSHKeyUnassignCmd.Flags().BoolVar(&orgSSHKeyUnassignDeleteKey, "delete-if-unassigned", false, "delete key if it has no assignments left")
}

func resolveLocalPublicKeySelection(path string) (*localPublicKey, error) {
	keyPath := strings.TrimSpace(path)
	if keyPath == "" {
		keys, err := discoverLocalPublicKeys()
		if err != nil {
			return nil, err
		}

		return selectPublicKey(keys)
	}

	absolutePath, err := filepath.Abs(keyPath)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(absolutePath)
	if err != nil {
		return nil, err
	}

	line := strings.TrimSpace(string(content))
	if !looksLikePublicKey(line) {
		return nil, fmt.Errorf("file %s does not look like an SSH public key", absolutePath)
	}

	comment := extractKeyComment(line)
	if comment == "" {
		comment = "no comment"
	}

	return &localPublicKey{
		Path:    absolutePath,
		Name:    filepath.Base(absolutePath),
		Content: line,
		Comment: comment,
	}, nil
}

func resolveOrganizationSSHKeySelection(keys []inventory.OrganizationSSHKey, keyID string) (*inventory.OrganizationSSHKey, error) {
	id := strings.TrimSpace(keyID)
	if id != "" {
		for i := range keys {
			if strings.EqualFold(strings.TrimSpace(keys[i].ID), id) {
				return &keys[i], nil
			}
		}

		return nil, fmt.Errorf("organization SSH key %q not found", keyID)
	}

	return selectOrganizationSSHKey(keys)
}

func selectOrganizationSSHKey(keys []inventory.OrganizationSSHKey) (*inventory.OrganizationSSHKey, error) {
	if len(keys) == 1 {
		return &keys[0], nil
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return strings.TrimSpace(keys[i].ID) < strings.TrimSpace(keys[j].ID)
	})

	type orgKeyChoice struct {
		Key   inventory.OrganizationSSHKey
		Title string
		Meta  string
	}

	items := make([]orgKeyChoice, 0, len(keys))
	for _, key := range keys {
		title := strings.TrimSpace(key.Comment)
		if title == "" {
			title = "SSH key"
		}
		meta := strings.TrimSpace(key.ID)
		items = append(items, orgKeyChoice{Key: key, Title: title, Meta: meta})
	}

	prompt := promptui.Select{
		Label:     "Select organization SSH key",
		Items:     items,
		Size:      selectSize(len(items)),
		Stdout:    &bellSkipper{},
		Templates: titleMetaSelectTemplates("key"),
	}

	index, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	selected := items[index].Key
	return &selected, nil
}
