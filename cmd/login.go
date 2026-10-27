package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/intercube/cli/util/inventory"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var sshUsername string
var sshLoadSites bool
var sshRoot bool
var sshForward []string
var sshTunnel bool
var sshConnectOnly bool

var sshCmd = &cobra.Command{
	Use: "ssh [site-or-server-filter]", Short: "Connect through Warpgate using Clerk sign-in", Args: cobra.MaximumNArgs(1), RunE: runGatewaySSH,
}
var loginCmd = &cobra.Command{
	Use: "login [site-or-server-filter]", Short: "Connect through Warpgate", Deprecated: "use `intercube ssh` instead", Args: cobra.MaximumNArgs(1), RunE: runGatewaySSH,
}

func runGatewaySSH(cmd *cobra.Command, args []string) error {
	client, _, err := newInventoryClient(cmd, "")
	if err != nil {
		return err
	}
	access, err := client.SSHAccess(cmd.Context())
	if err != nil {
		return err
	}
	query := ""
	if len(args) > 0 {
		query = args[0]
	}
	targets := filterGatewayTargets(access.Targets, query, sshRoot || sshUsername == "root", sshUsername)
	if len(targets) == 0 {
		return fmt.Errorf("no authorized SSH targets match %q", query)
	}
	index := 0
	if len(targets) > 1 {
		if err := ensureInteractiveMode("SSH target selection"); err != nil {
			return err
		}
		prompt := promptui.Select{Label: "Select site or server", Items: targets, Size: selectSize(len(targets)), HideHelp: true,
			Templates: &promptui.SelectTemplates{Label: "{{ . }}", Active: "> {{ .Name | cyan }}: {{ .LinuxUser }} on {{ .ServerName }} {{ .Domains }}", Inactive: "  {{ .Name }}: {{ .LinuxUser }} on {{ .ServerName }} {{ .Domains }}", Selected: "{{ .Name }}"},
			Searcher:  func(input string, index int) bool { return gatewayTargetMatches(targets[index], input) }, StartInSearchMode: true}
		index, _, err = prompt.Run()
		if err != nil {
			return err
		}
	}
	target := targets[index]
	connection := gatewayConnection(*access, target)
	socket, err := gatewayControlPath(connection)
	if err != nil {
		return err
	}
	if sshConnectOnly {
		return ensureGatewayMaster(cmd, connection, socket)
	}
	arguments := gatewaySSHArgs(connection, socket)
	if isNonInteractiveMode() {
		arguments = append(arguments, "-o", "BatchMode=yes")
	}
	for _, forward := range sshForward {
		arguments = append(arguments, "-L", forward)
	}
	if sshTunnel {
		arguments = append(arguments, "-N")
	}
	arguments = append(arguments, "--", connection.Host)
	command := exec.CommandContext(cmd.Context(), "ssh", arguments...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func gatewayTargetMatches(target inventory.SSHTarget, query string) bool {
	value := strings.ToLower(strings.TrimSpace(query))
	fields := []string{target.Name, target.ServerName, target.Address, target.LinuxUser, target.OrganizationID, strconv.Itoa(target.ServerID)}
	fields = append(fields, target.Domains...)
	if target.SiteID != nil {
		fields = append(fields, strconv.Itoa(*target.SiteID))
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), value) {
			return true
		}
	}
	return false
}

func filterGatewayTargets(targets []inventory.SSHTarget, query string, root bool, username string) []inventory.SSHTarget {
	result := make([]inventory.SSHTarget, 0)
	for _, target := range targets {
		if target.Root != root || (username != "" && username != target.LinuxUser) {
			continue
		}
		if gatewayTargetMatches(target, query) {
			result = append(result, target)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// Suppress promptui's terminal bell without changing prompt output.
type bellSkipper struct{}

func (bs *bellSkipper) Write(b []byte) (int, error) {
	filtered := make([]byte, 0, len(b))
	for _, value := range b {
		if value != 7 {
			filtered = append(filtered, value)
		}
	}
	_, err := os.Stderr.Write(filtered)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}
func (bs *bellSkipper) Close() error { return nil }

func init() {
	rootCmd.AddCommand(sshCmd, loginCmd)
	for _, command := range []*cobra.Command{sshCmd, loginCmd} {
		command.Flags().StringVar(&sshUsername, "ssh_username", "", "Select only an authorized Linux account; root requires explicit platform access")
		command.Flags().BoolVarP(&sshLoadSites, "sites", "s", false, "Site metadata is always available")
		command.Flags().BoolVar(&sshRoot, "root", false, "Select platform root targets (platform administrators only)")
		command.Flags().StringArrayVarP(&sshForward, "forward", "L", nil, "Local SSH tunnel, e.g. 127.0.0.1:3307:127.0.0.1:3306")
		command.Flags().BoolVar(&sshTunnel, "tunnel", false, "Keep only the tunnel open without a remote shell")
		command.Flags().BoolVar(&sshConnectOnly, "connect-only", false, "Approve a reusable connection for subsequent non-interactive sync commands")
	}
}
