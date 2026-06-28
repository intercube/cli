package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intercube/cli/util/appconfig"
	authutil "github.com/intercube/cli/util/auth"
	"github.com/intercube/cli/util/inventory"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

type localPublicKey struct {
	Path    string
	Name    string
	Content string
	Comment string
}

var siteOrgID string

var siteAddSSHKeyCmd = &cobra.Command{
	Use:   "add-ssh-key",
	Short: "Add one of your local SSH public keys to a site",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := appconfig.ValidateClerk(); err != nil {
			return fmt.Errorf("%w (set via env/.env or build-time)", err)
		}

		if err := appconfig.ValidateInventory(); err != nil {
			return fmt.Errorf("%w (set via env/.env or build-time)", err)
		}

		store, err := authutil.NewSessionStore("intercube-cli")
		if err != nil {
			return err
		}

		clerkClient := &authutil.ClerkClient{
			Issuer:       appconfig.ClerkIssuer,
			ClientID:     appconfig.ClerkClientID,
			Audience:     appconfig.ClerkAudience,
			Scopes:       appconfig.ClerkScopes,
			CallbackPort: appconfig.ParsedCallbackPort(),
		}

		organizationID := strings.TrimSpace(siteOrgID)
		if organizationID == "" {
			session, sessionErr := store.Load(cmd.Context())
			if sessionErr == nil {
				organizationID = strings.TrimSpace(session.OrganizationID)
			}
		}

		if organizationID == "" {
			organizationID = strings.TrimSpace(appconfig.OrganizationID)
		}

		inventoryClient := inventory.NewClient(appconfig.InventoryAPIBaseURL, organizationID, store, clerkClient)

		sites, err := inventoryClient.ListSites(cmd.Context())
		if err != nil {
			if shouldPromptForOrganization(err) {
				return errors.New("organization context is required. Run `intercube auth org` (or pass --org-id)")
			}

			return err
		}

		if len(sites) == 0 {
			return fmt.Errorf("no sites available for your account")
		}

		selectedSite, err := selectSite(sites)
		if err != nil {
			return err
		}

		keys, err := discoverLocalPublicKeys()
		if err != nil {
			return err
		}

		selectedKey, err := selectPublicKey(keys)
		if err != nil {
			return err
		}

		existingKeys, err := inventoryClient.ListOrganizationSSHKeys(cmd.Context())
		if err != nil {
			return err
		}

		var matchedKey *inventory.OrganizationSSHKey
		for _, existing := range existingKeys {
			if strings.TrimSpace(existing.Content) == strings.TrimSpace(selectedKey.Content) {
				existingCopy := existing
				matchedKey = &existingCopy
				break
			}
		}

		comment := strings.TrimSpace(selectedKey.Comment)
		if comment == "" {
			comment = selectedKey.Name
		}

		if matchedKey == nil {
			createdKey, createErr := inventoryClient.CreateOrganizationSSHKey(cmd.Context(), inventory.OrganizationSSHKeyRequest{
				Content: selectedKey.Content,
				Comment: comment,
			})
			if createErr != nil {
				return createErr
			}

			matchedKey = createdKey
			fmt.Printf("Created organization SSH key %s from %s\n", strings.TrimSpace(createdKey.ID), selectedKey.Name)
		} else {
			fmt.Printf("Using existing organization SSH key %s\n", strings.TrimSpace(matchedKey.ID))
		}

		if strings.TrimSpace(matchedKey.ID) == "" {
			return fmt.Errorf("organization SSH key id is empty")
		}

		if assignErr := inventoryClient.AssignOrganizationSSHKeyToSite(cmd.Context(), matchedKey.ID, selectedSite.ID); assignErr != nil {
			return assignErr
		}

		fmt.Printf("Added SSH key to site %s (%s)\n", siteDisplayName(*selectedSite), selectedSite.ID)
		if strings.TrimSpace(matchedKey.ID) != "" {
			fmt.Printf("Key ID: %s\n", matchedKey.ID)
		}

		return nil
	},
}

func init() {
	siteCmd.AddCommand(siteAddSSHKeyCmd)
	siteAddSSHKeyCmd.Flags().StringVar(&siteOrgID, "org-id", "", "organization id (sets X-Organization-Id for inventory requests)")
}

func selectSite(sites []inventory.SiteServer) (*inventory.SiteServer, error) {
	if len(sites) == 1 {
		return &sites[0], nil
	}

	sort.Slice(sites, func(i, j int) bool {
		return siteDisplayName(sites[i]) < siteDisplayName(sites[j])
	})

	type siteChoice struct {
		Site  inventory.SiteServer
		Title string
		Meta  string
	}

	items := make([]siteChoice, 0, len(sites))
	for _, site := range sites {
		title, meta := siteSelectionLabel(site)
		items = append(items, siteChoice{Site: site, Title: title, Meta: meta})
	}

	prompt := promptui.Select{
		Label:     "Select a site",
		Items:     items,
		Templates: titleMetaSelectTemplates("site"),
		Size:      selectSize(len(items)),
		Stdout:    &bellSkipper{},
		Searcher: func(input string, index int) bool {
			site := items[index].Site
			needle := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(input)), " ", "")
			haystack := strings.ToLower(site.ID + " " + site.Username + " " + site.MainDomain + " " + site.ServerName)
			haystack = strings.ReplaceAll(haystack, " ", "")
			return strings.Contains(haystack, needle)
		},
	}

	index, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	selected := items[index].Site
	return &selected, nil
}

func siteSelectionLabel(site inventory.SiteServer) (string, string) {
	username := strings.TrimSpace(site.Username)
	domain := strings.TrimSpace(site.MainDomain)
	server := strings.TrimSpace(site.ServerName)

	title := domain
	if title == "" {
		title = username
	}
	if title == "" {
		title = server
	}
	if title == "" {
		title = "(unnamed site)"
	}

	metaParts := make([]string, 0, 3)
	if username != "" && !strings.EqualFold(username, title) {
		metaParts = append(metaParts, username)
	}
	if server != "" && !strings.EqualFold(server, title) {
		metaParts = append(metaParts, server)
	}

	return title, strings.Join(metaParts, " | ")
}

func selectPublicKey(keys []localPublicKey) (*localPublicKey, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("no SSH public keys found in ~/.ssh")
	}

	if len(keys) == 1 {
		return &keys[0], nil
	}

	type keyChoice struct {
		Key   localPublicKey
		Title string
		Meta  string
	}

	items := make([]keyChoice, 0, len(keys))
	for _, key := range keys {
		items = append(items, keyChoice{Key: key, Title: key.Name, Meta: key.Comment})
	}

	prompt := promptui.Select{
		Label:     "Select an SSH public key",
		Items:     items,
		Templates: titleMetaSelectTemplates("key"),
		Size:      selectSize(len(items)),
		Stdout:    &bellSkipper{},
		Searcher: func(input string, index int) bool {
			item := items[index].Key
			needle := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(input)), " ", "")
			haystack := strings.ToLower(item.Name + " " + item.Comment + " " + item.Path)
			haystack = strings.ReplaceAll(haystack, " ", "")
			return strings.Contains(haystack, needle)
		},
	}

	index, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	selected := items[index].Key
	return &selected, nil
}

func discoverLocalPublicKeys() ([]localPublicKey, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	paths, err := filepath.Glob(filepath.Join(home, ".ssh", "*.pub"))
	if err != nil {
		return nil, err
	}

	sort.Strings(paths)
	keys := make([]localPublicKey, 0, len(paths))

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		line := strings.TrimSpace(string(content))
		if !looksLikePublicKey(line) {
			continue
		}

		comment := extractKeyComment(line)
		if comment == "" {
			comment = "no comment"
		}

		keys = append(keys, localPublicKey{
			Path:    path,
			Name:    filepath.Base(path),
			Content: line,
			Comment: comment,
		})
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no SSH public keys found in %s", filepath.Join(home, ".ssh"))
	}

	return keys, nil
}

func looksLikePublicKey(value string) bool {
	fields := strings.Fields(value)
	if len(fields) < 2 {
		return false
	}

	keyType := fields[0]
	return strings.HasPrefix(keyType, "ssh-") || strings.HasPrefix(keyType, "ecdsa-") || strings.HasPrefix(keyType, "sk-") || strings.HasPrefix(keyType, "rsa-sha2-")
}

func extractKeyComment(value string) string {
	fields := strings.Fields(value)
	if len(fields) <= 2 {
		return ""
	}

	return strings.Join(fields[2:], " ")
}

func siteDisplayName(site inventory.SiteServer) string {
	if strings.TrimSpace(site.MainDomain) != "" {
		return site.MainDomain
	}

	if strings.TrimSpace(site.Username) != "" {
		return site.Username
	}

	return site.ID
}

func shouldPromptForOrganization(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "organization id not found") || strings.Contains(message, "multiple organizations")
}
