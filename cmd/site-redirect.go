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
	siteRedirectListSiteID   string
	siteRedirectAddSiteID    string
	siteRedirectAddDomain    string
	siteRedirectAddCode      int
	siteRedirectAddLocation  string
	siteRedirectAddValue     string
	siteRedirectRemoveSiteID string
	siteRedirectRemoveID     string
	siteRedirectRemoveYes    bool
)

var siteRedirectCmd = &cobra.Command{
	Use:   "redirect",
	Short: "Manage site redirects",
}

var siteRedirectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List redirects for a site",
	RunE: func(cmd *cobra.Command, args []string) error {
		inventoryClient, _, err := newInventoryClient(cmd, "")
		if err != nil {
			return err
		}

		site, err := resolveSiteSelection(cmd, inventoryClient, siteRedirectListSiteID)
		if err != nil {
			return err
		}

		redirects, err := inventoryClient.ListSiteRedirects(cmd.Context(), site.ID)
		if err != nil {
			return err
		}

		if len(redirects) == 0 {
			fmt.Printf("No redirects found for site %s (%s)\n", siteDisplayName(*site), site.ID)
			return nil
		}

		sort.SliceStable(redirects, func(i, j int) bool {
			left := strings.ToLower(redirects[i].Domain + redirects[i].Location)
			right := strings.ToLower(redirects[j].Domain + redirects[j].Location)
			return left < right
		})

		for _, redirect := range redirects {
			fmt.Printf("[%s] %s %d %s -> %s\n", redirect.ID, redirect.Domain, redirect.ReturnCode, redirect.Location, redirect.Value)
		}

		return nil
	},
}

var siteRedirectAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a redirect to a site",
	RunE: func(cmd *cobra.Command, args []string) error {
		inventoryClient, _, err := newInventoryClient(cmd, "")
		if err != nil {
			return err
		}

		site, err := resolveSiteSelection(cmd, inventoryClient, siteRedirectAddSiteID)
		if err != nil {
			return err
		}

		domain := strings.TrimSpace(siteRedirectAddDomain)
		if domain == "" {
			domain, err = promptRequiredText("Redirect domain", site.MainDomain)
			if err != nil {
				return err
			}
		}

		location := strings.TrimSpace(siteRedirectAddLocation)
		if location == "" {
			location, err = promptRequiredText("Redirect source location", "/")
			if err != nil {
				return err
			}
		}

		value := strings.TrimSpace(siteRedirectAddValue)
		if value == "" {
			value, err = promptRequiredText("Redirect destination", "")
			if err != nil {
				return err
			}
		}

		returnCode := siteRedirectAddCode
		if returnCode == 0 {
			returnCode = 301
		}
		if !isValidRedirectCode(returnCode) {
			return fmt.Errorf("redirect return code must be one of 301, 302, 307, 308")
		}

		created, err := inventoryClient.CreateSiteRedirect(cmd.Context(), site.ID, inventory.RedirectMutate{
			Domain:     domain,
			ReturnCode: returnCode,
			Location:   location,
			Value:      value,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Added redirect [%s] %s %d %s -> %s\n", created.ID, created.Domain, created.ReturnCode, created.Location, created.Value)
		return nil
	},
}

var siteRedirectRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a redirect from a site",
	RunE: func(cmd *cobra.Command, args []string) error {
		inventoryClient, _, err := newInventoryClient(cmd, "")
		if err != nil {
			return err
		}

		site, err := resolveSiteSelection(cmd, inventoryClient, siteRedirectRemoveSiteID)
		if err != nil {
			return err
		}

		redirects, err := inventoryClient.ListSiteRedirects(cmd.Context(), site.ID)
		if err != nil {
			return err
		}

		if len(redirects) == 0 {
			return fmt.Errorf("no redirects found for site %s (%s)", siteDisplayName(*site), site.ID)
		}

		selected, err := resolveRedirectSelection(redirects, siteRedirectRemoveID)
		if err != nil {
			return err
		}

		if !siteRedirectRemoveYes {
			confirmed, confirmErr := promptYesNo(fmt.Sprintf("Delete redirect %s %d %s -> %s?", selected.Domain, selected.ReturnCode, selected.Location, selected.Value))
			if confirmErr != nil {
				return confirmErr
			}

			if !confirmed {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		if err := inventoryClient.DeleteSiteRedirect(cmd.Context(), site.ID, selected.ID); err != nil {
			return err
		}

		fmt.Printf("Deleted redirect %s (%s)\n", selected.Location, selected.ID)
		return nil
	},
}

func init() {
	siteCmd.AddCommand(siteRedirectCmd)
	siteRedirectCmd.AddCommand(siteRedirectListCmd)
	siteRedirectCmd.AddCommand(siteRedirectAddCmd)
	siteRedirectCmd.AddCommand(siteRedirectRemoveCmd)

	siteRedirectListCmd.Flags().StringVar(&siteRedirectListSiteID, "site-id", "", "site id")

	siteRedirectAddCmd.Flags().StringVar(&siteRedirectAddSiteID, "site-id", "", "site id")
	siteRedirectAddCmd.Flags().StringVar(&siteRedirectAddDomain, "domain", "", "domain for redirect")
	siteRedirectAddCmd.Flags().IntVar(&siteRedirectAddCode, "code", 301, "redirect return code (301,302,307,308)")
	siteRedirectAddCmd.Flags().StringVar(&siteRedirectAddLocation, "location", "", "source location path")
	siteRedirectAddCmd.Flags().StringVar(&siteRedirectAddValue, "value", "", "destination URL or path")

	siteRedirectRemoveCmd.Flags().StringVar(&siteRedirectRemoveSiteID, "site-id", "", "site id")
	siteRedirectRemoveCmd.Flags().StringVar(&siteRedirectRemoveID, "id", "", "redirect id")
	siteRedirectRemoveCmd.Flags().BoolVar(&siteRedirectRemoveYes, "yes", false, "delete without confirmation")
}

func resolveRedirectSelection(redirects []inventory.Redirect, redirectID string) (*inventory.Redirect, error) {
	id := strings.TrimSpace(redirectID)
	if id != "" {
		for i := range redirects {
			if strings.EqualFold(strings.TrimSpace(redirects[i].ID), id) {
				return &redirects[i], nil
			}
		}

		return nil, fmt.Errorf("redirect id %q not found", redirectID)
	}

	return selectRedirect(redirects)
}

func selectRedirect(redirects []inventory.Redirect) (*inventory.Redirect, error) {
	if len(redirects) == 1 {
		return &redirects[0], nil
	}

	sort.SliceStable(redirects, func(i, j int) bool {
		left := strings.ToLower(redirects[i].Domain + redirects[i].Location)
		right := strings.ToLower(redirects[j].Domain + redirects[j].Location)
		return left < right
	})

	type redirectChoice struct {
		Redirect inventory.Redirect
		Title    string
		Meta     string
	}

	items := make([]redirectChoice, 0, len(redirects))
	for _, redirect := range redirects {
		title := fmt.Sprintf("%s %d %s", strings.TrimSpace(redirect.Domain), redirect.ReturnCode, strings.TrimSpace(redirect.Location))
		meta := strings.TrimSpace(redirect.Value)
		items = append(items, redirectChoice{Redirect: redirect, Title: title, Meta: meta})
	}

	prompt := promptui.Select{
		Label:     "Select redirect",
		Items:     items,
		Size:      selectSize(len(items)),
		Stdout:    &bellSkipper{},
		Templates: titleMetaSelectTemplates("redirect"),
	}

	index, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	selected := items[index].Redirect
	return &selected, nil
}

func isValidRedirectCode(code int) bool {
	return code == 301 || code == 302 || code == 307 || code == 308
}
