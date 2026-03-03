package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/intercube/cli/util/inventory"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

type ResolvedSyncTarget struct {
	SiteID      string
	DisplayName string
	Host        string
	Username    string
	Port        int
}

type ResolvedSyncSource struct {
	SiteID      string
	DisplayName string
}

func resolveSyncTarget(cmd *cobra.Command, inventoryClient *inventory.Client, query string, explicitSiteID string) (ResolvedSyncTarget, ResolvedSyncSource, error) {
	sites, err := inventoryClient.ListSites(cmd.Context())
	if err != nil {
		if shouldPromptForOrganizationError(err) {
			return ResolvedSyncTarget{}, ResolvedSyncSource{}, fmt.Errorf("organization context is required. Run `intercube auth org` (or pass --org-id)")
		}

		return ResolvedSyncTarget{}, ResolvedSyncSource{}, err
	}

	if len(sites) == 0 {
		return ResolvedSyncTarget{}, ResolvedSyncSource{}, fmt.Errorf("no sites available for your account")
	}

	sourceSite := resolveSourceSite(sites)
	source := ResolvedSyncSource{DisplayName: "unknown"}
	if sourceSite != nil {
		source = ResolvedSyncSource{SiteID: sourceSite.ID, DisplayName: syncSiteDisplayName(*sourceSite)}
	}

	candidates := excludeSourceSite(sites, sourceSite)
	if len(candidates) == 0 {
		return ResolvedSyncTarget{}, source, fmt.Errorf("no target sites available")
	}

	selectedSite, err := selectTargetSite(candidates, sites, query, explicitSiteID)
	if err != nil {
		return ResolvedSyncTarget{}, source, err
	}

	target, err := promptTargetAccessDetails(selectedSite)
	if err != nil {
		return ResolvedSyncTarget{}, source, err
	}

	return target, source, nil
}

func shouldPromptForOrganizationError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "x-organization-id") || strings.Contains(message, "organization")
}

func resolveSourceSite(sites []inventory.SiteServer) *inventory.SiteServer {
	hostname, err := os.Hostname()
	if err != nil {
		return nil
	}

	trimmed := strings.TrimSpace(hostname)
	if trimmed == "" {
		return nil
	}

	matches := findSiteMatches(sites, trimmed)
	if len(matches) == 1 {
		return matches[0]
	}

	return nil
}
func selectTargetSite(candidates []inventory.SiteServer, allSites []inventory.SiteServer, query string, explicitSiteID string) (*inventory.SiteServer, error) {
	if strings.TrimSpace(explicitSiteID) != "" {
		selected, found := findSiteByID(allSites, explicitSiteID)
		if !found {
			return nil, fmt.Errorf("site %q not found", explicitSiteID)
		}

		return selected, nil
	}

	if strings.TrimSpace(query) != "" {
		matches := findSiteMatches(candidates, query)
		if len(matches) == 1 {
			return matches[0], nil
		}

		if len(matches) > 1 {
			matchedSites := make([]inventory.SiteServer, 0, len(matches))
			for _, match := range matches {
				matchedSites = append(matchedSites, *match)
			}

			fmt.Printf("Query %q matched multiple sites, please choose:\n", query)
			selected, err := selectSiteFromList(matchedSites)
			if err != nil {
				return nil, err
			}

			return selected, nil
		}

		return nil, fmt.Errorf("no target site matched %q", query)
	}

	selected, err := selectSiteFromList(candidates)
	if err != nil {
		return nil, err
	}

	return selected, nil
}

func selectSiteFromList(sites []inventory.SiteServer) (*inventory.SiteServer, error) {
	if len(sites) == 0 {
		return nil, fmt.Errorf("no sites available")
	}

	if len(sites) == 1 {
		return &sites[0], nil
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "\U0001F9CA {{ . | red }}",
		Inactive: "  {{ . | cyan }}",
		Selected: "\U0001F9CA {{ . | cyan }}",
	}

	labels := make([]string, 0, len(sites))
	for _, site := range sites {
		labels = append(labels, syncSiteDisplayName(site))
	}

	prompt := promptui.Select{
		Label:     "Select target site",
		Items:     labels,
		Templates: templates,
		Size:      minInt(10, len(labels)),
		Stdout:    &bellSkipper{},
	}

	index, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	return &sites[index], nil
}

func syncSiteDisplayName(site inventory.SiteServer) string {
	parts := make([]string, 0, 3)
	if strings.TrimSpace(site.MainDomain) != "" {
		parts = append(parts, strings.TrimSpace(site.MainDomain))
	}
	if strings.TrimSpace(site.ServerName) != "" {
		parts = append(parts, strings.TrimSpace(site.ServerName))
	}
	if strings.TrimSpace(site.ID) != "" {
		parts = append(parts, strings.TrimSpace(site.ID))
	}

	if len(parts) == 0 {
		return "(unnamed site)"
	}

	return strings.Join(parts, " | ")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func findSiteMatches(sites []inventory.SiteServer, value string) []*inventory.SiteServer {
	needle := normalizeToken(value)
	if needle == "" {
		return nil
	}

	exact := make([]*inventory.SiteServer, 0, 1)
	contains := make([]*inventory.SiteServer, 0, 2)

	for i := range sites {
		candidates := []string{sites[i].ID, sites[i].MainDomain, sites[i].ServerName, sites[i].Username}
		matchedExact := false
		matchedContains := false
		for _, candidate := range candidates {
			normalized := normalizeToken(candidate)
			if normalized == "" {
				continue
			}

			if normalized == needle {
				matchedExact = true
				break
			}

			if strings.Contains(normalized, needle) {
				matchedContains = true
			}
		}

		if matchedExact {
			exact = append(exact, &sites[i])
			continue
		}

		if matchedContains {
			contains = append(contains, &sites[i])
		}
	}

	if len(exact) > 0 {
		return exact
	}

	return contains
}

func normalizeToken(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	trimmed = strings.ReplaceAll(trimmed, " ", "")
	return trimmed
}

func excludeSourceSite(sites []inventory.SiteServer, sourceSite *inventory.SiteServer) []inventory.SiteServer {
	excludedID := ""
	if sourceSite != nil {
		excludedID = strings.TrimSpace(sourceSite.ID)
	}

	result := make([]inventory.SiteServer, 0, len(sites))
	for _, site := range sites {
		if excludedID != "" && strings.EqualFold(strings.TrimSpace(site.ID), excludedID) {
			continue
		}

		result = append(result, site)
	}

	sort.SliceStable(result, func(i, j int) bool {
		return syncSiteDisplayName(result[i]) < syncSiteDisplayName(result[j])
	})

	return result
}

func promptTargetAccessDetails(site *inventory.SiteServer) (ResolvedSyncTarget, error) {
	hostDefault := strings.TrimSpace(site.MainDomain)
	userDefault := strings.TrimSpace(site.Username)
	portDefault := "22"

	host, err := promptText("Target SSH host", hostDefault, requiredValue, 0)
	if err != nil {
		return ResolvedSyncTarget{}, err
	}

	username, err := promptText("Target SSH user", userDefault, requiredValue, 0)
	if err != nil {
		return ResolvedSyncTarget{}, err
	}

	portString, err := promptText("Target SSH port", portDefault, requiredValue, 0)
	if err != nil {
		return ResolvedSyncTarget{}, err
	}

	port, err := strconv.Atoi(strings.TrimSpace(portString))
	if err != nil || port <= 0 {
		return ResolvedSyncTarget{}, fmt.Errorf("invalid SSH port %q", portString)
	}

	return ResolvedSyncTarget{
		SiteID:      strings.TrimSpace(site.ID),
		DisplayName: syncSiteDisplayName(*site),
		Host:        strings.TrimSpace(host),
		Username:    strings.TrimSpace(username),
		Port:        port,
	}, nil
}
