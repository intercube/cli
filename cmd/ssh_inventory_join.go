package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/boundary/api/hosts"
	"github.com/intercube/cli/util/inventory"
	"github.com/spf13/cobra"
)

type sshTargetOption struct {
	Host         *hosts.Host
	HostName     string
	HostID       string
	ServerName   string
	Title        string
	Meta         string
	SitePreview  string
	JoinStatus   string
	SearchFields []string
}

func fetchInventorySites(cmd *cobra.Command) ([]inventory.SiteServer, error) {
	inventoryClient, _, err := newInventoryClient(cmd, "")
	if err != nil {
		return nil, err
	}

	sites, err := inventoryClient.ListSites(cmd.Context())
	if err != nil {
		return nil, err
	}

	return sites, nil
}

func buildSSHTargetOptions(hostsList []*hosts.Host, sites []inventory.SiteServer) []sshTargetOption {
	if len(hostsList) == 0 {
		return nil
	}

	hostIndexByExactKey := make(map[string][]int, len(hostsList))
	hostIndexByShortKey := make(map[string][]int, len(hostsList))
	for i, host := range hostsList {
		exactKey := sshJoinKey(host.Name)
		if exactKey != "" {
			hostIndexByExactKey[exactKey] = appendUniqueHostIndex(hostIndexByExactKey[exactKey], i)
		}

		shortKey := sshJoinShortKey(host.Name)
		if shortKey != "" && shortKey != exactKey {
			hostIndexByShortKey[shortKey] = appendUniqueHostIndex(hostIndexByShortKey[shortKey], i)
		}
	}

	sitesByHostIndex := make(map[int][]inventory.SiteServer)
	for _, site := range sites {
		matchedHostIndexes := make(map[int]struct{})

		exactServerKey := sshJoinKey(site.ServerName)
		for _, hostIndex := range hostIndexByExactKey[exactServerKey] {
			matchedHostIndexes[hostIndex] = struct{}{}
		}

		if len(matchedHostIndexes) == 0 {
			shortServerKey := sshJoinShortKey(site.ServerName)
			shortCandidates := hostIndexByShortKey[shortServerKey]
			if len(shortCandidates) == 1 {
				hostIndex := shortCandidates[0]
				matchedHostIndexes[hostIndex] = struct{}{}
			}
		}

		for hostIndex := range matchedHostIndexes {
			sitesByHostIndex[hostIndex] = append(sitesByHostIndex[hostIndex], site)
		}
	}

	options := make([]sshTargetOption, 0, len(hostsList))
	for i, host := range hostsList {
		associatedSites := dedupeAssociatedSites(sitesByHostIndex[i])
		siteLabels := associatedSiteLabels(associatedSites)

		serverName := strings.TrimSpace(host.Name)
		if inventoryServerName := chooseInventoryServerName(associatedSites); inventoryServerName != "" {
			serverName = inventoryServerName
		}

		title := serverName
		if strings.TrimSpace(title) == "" {
			title = strings.TrimSpace(host.Name)
		}
		if title == "" {
			title = "(unnamed host)"
		}

		meta := ""
		sitePreview := "-"
		joinStatus := "boundary_only"
		if len(siteLabels) > 0 {
			meta = buildMetaLabel(title, siteLabels)
			sitePreview = summarizeSiteLabels(siteLabels, 5)
			joinStatus = "inventory_enriched"
		}

		options = append(options, sshTargetOption{
			Host:         host,
			HostName:     strings.TrimSpace(host.Name),
			HostID:       strings.TrimSpace(host.Id),
			ServerName:   serverName,
			Title:        title,
			Meta:         meta,
			SitePreview:  sitePreview,
			JoinStatus:   joinStatus,
			SearchFields: buildSearchFieldsForTarget(host, associatedSites, siteLabels, serverName),
		})
	}

	return dedupeSSHTargetOptions(options)
}

func buildSearchFieldsForTarget(host *hosts.Host, associatedSites []inventory.SiteServer, siteLabels []string, serverName string) []string {
	raw := []string{host.Name, serverName, sshJoinShortKey(host.Name), sshJoinShortKey(serverName)}
	for _, site := range associatedSites {
		raw = append(raw,
			site.ID,
			site.ServerID,
			site.ServerName,
			site.MainDomain,
			site.Username,
		)
	}
	raw = append(raw, siteLabels...)

	fields := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, value := range raw {
		normalized := sshNormalizeSearchValue(value)
		if normalized == "" {
			continue
		}

		if _, exists := seen[normalized]; exists {
			continue
		}

		seen[normalized] = struct{}{}
		fields = append(fields, normalized)
	}

	return fields
}

func dedupeAssociatedSites(sites []inventory.SiteServer) []inventory.SiteServer {
	if len(sites) < 2 {
		return sites
	}

	result := make([]inventory.SiteServer, 0, len(sites))
	seen := make(map[string]struct{}, len(sites))
	for _, site := range sites {
		key := strings.TrimSpace(site.ID)
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(site.MainDomain)) + "|" + strings.ToLower(strings.TrimSpace(site.Username)) + "|" + strings.ToLower(strings.TrimSpace(site.ServerName))
		}

		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		result = append(result, site)
	}

	return result
}

func associatedSiteLabels(sites []inventory.SiteServer) []string {
	labels := make([]string, 0, len(sites))
	seen := make(map[string]struct{}, len(sites))
	for _, site := range sites {
		label := strings.TrimSpace(site.MainDomain)
		if label == "" {
			label = strings.TrimSpace(site.Username)
		}
		if label == "" {
			label = strings.TrimSpace(site.ID)
		}
		if label == "" {
			continue
		}

		normalized := strings.ToLower(label)
		if _, exists := seen[normalized]; exists {
			continue
		}

		seen[normalized] = struct{}{}
		labels = append(labels, label)
	}

	sort.SliceStable(labels, func(i, j int) bool {
		return strings.ToLower(labels[i]) < strings.ToLower(labels[j])
	})

	return labels
}

func chooseInventoryServerName(sites []inventory.SiteServer) string {
	serverNames := make([]string, 0, len(sites))
	seen := make(map[string]struct{}, len(sites))
	for _, site := range sites {
		serverName := strings.TrimSpace(site.ServerName)
		if serverName == "" {
			continue
		}

		normalized := strings.ToLower(serverName)
		if _, exists := seen[normalized]; exists {
			continue
		}

		seen[normalized] = struct{}{}
		serverNames = append(serverNames, serverName)
	}

	if len(serverNames) == 0 {
		return ""
	}

	sort.SliceStable(serverNames, func(i, j int) bool {
		return strings.ToLower(serverNames[i]) < strings.ToLower(serverNames[j])
	})

	return serverNames[0]
}

func sshJoinKeys(value string) []string {
	primary := sshJoinKey(value)
	if primary == "" {
		return nil
	}

	keys := []string{primary}
	if short := sshJoinShortKey(primary); short != "" && short != primary {
		keys = append(keys, short)
	}

	return keys
}

func sshJoinKey(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = strings.TrimSuffix(normalized, ".")
	normalized = strings.ReplaceAll(normalized, " ", "")
	return normalized
}

func sshJoinShortKey(value string) string {
	normalized := sshJoinKey(value)
	if normalized == "" {
		return ""
	}

	if separator := strings.Index(normalized, "."); separator > 0 {
		return normalized[:separator]
	}

	return normalized
}

func appendUniqueHostIndex(indexes []int, value int) []int {
	for _, current := range indexes {
		if current == value {
			return indexes
		}
	}

	return append(indexes, value)
}

func summarizeSiteLabels(labels []string, max int) string {
	if len(labels) == 0 {
		return "-"
	}

	if max <= 0 || len(labels) <= max {
		return strings.Join(labels, ", ")
	}

	visible := strings.Join(labels[:max], ", ")
	return fmt.Sprintf("%s (+%d more)", visible, len(labels)-max)
}

func buildMetaLabel(title string, siteLabels []string) string {
	if len(siteLabels) == 0 {
		return ""
	}

	primary := siteLabels[0]
	if len(siteLabels) == 1 {
		if strings.EqualFold(strings.TrimSpace(primary), strings.TrimSpace(title)) {
			return ""
		}

		return primary
	}

	if strings.EqualFold(strings.TrimSpace(primary), strings.TrimSpace(title)) {
		return fmt.Sprintf("%d sites", len(siteLabels))
	}

	return fmt.Sprintf("%s (+%d more)", primary, len(siteLabels)-1)
}

func dedupeSSHTargetOptions(options []sshTargetOption) []sshTargetOption {
	if len(options) < 2 {
		sortSSHTargetsByTitle(options)
		return options
	}

	indexByKey := make(map[string]int, len(options))
	result := make([]sshTargetOption, 0, len(options))
	for _, option := range options {
		key := strings.ToLower(strings.TrimSpace(option.HostName))
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(option.Title))
		}
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(option.HostID))
		}
		if key == "" {
			result = append(result, option)
			continue
		}

		if index, exists := indexByKey[key]; exists {
			if shouldReplaceSSHTargetOption(result[index], option) {
				result[index] = option
			}
			continue
		}

		indexByKey[key] = len(result)
		result = append(result, option)
	}

	sortSSHTargetsByTitle(result)
	return result
}

func shouldReplaceSSHTargetOption(current sshTargetOption, candidate sshTargetOption) bool {
	if current.JoinStatus != candidate.JoinStatus {
		return candidate.JoinStatus == "inventory_enriched"
	}

	if len(candidate.SitePreview) > len(current.SitePreview) {
		return true
	}

	return false
}
