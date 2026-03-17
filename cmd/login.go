/*
Copyright © 2023 Intercube <opensource@intercube.io>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/boundary/api/hostcatalogs"
	"github.com/hashicorp/boundary/api/hosts"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"sort"
)

var sshUsername = "root"

var sshCmd = &cobra.Command{
	Use:   "ssh [host-filter]",
	Short: "Connect to a host over Boundary SSH",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runBoundarySSH(cmd, args, false)
	},
}

// loginCmd is deprecated; use sshCmd.
var loginCmd = &cobra.Command{
	Use:        "login [host-filter]",
	Short:      "Deprecated: use `intercube ssh`",
	Args:       cobra.MaximumNArgs(1),
	Deprecated: "use `intercube ssh` instead",
	Run: func(cmd *cobra.Command, args []string) {
		runBoundarySSH(cmd, args, true)
	},
}

func runBoundarySSH(cmd *cobra.Command, args []string, fromDeprecatedLogin bool) {
	if fromDeprecatedLogin {
		fmt.Println("Warning: `intercube login` is deprecated. Use `intercube ssh` instead.")
	}

	if err := ensureLoginConfiguration(); err != nil {
		fmt.Printf("Unable to continue SSH session: %v\n", err)
		return
	}

	boundaryUrl := config.Login.InstanceUrl
	if boundaryUrl == "" {
		boundaryUrl = "https://controller.boundary.intercube.cloud"
	}

	boundaryPath, err := exec.LookPath("boundary")
	if err != nil {
		panic("Boundary is not installed. Download and install Boundary before using `intercube ssh` (https://learn.hashicorp.com/tutorials/boundary/getting-started-install)")
	}

	apiConfig, _ := api.DefaultConfig()
	apiConfig.Addr = boundaryUrl

	client, err := api.NewClient(apiConfig)
	if err != nil {
		panic(err)
	}

	credentials := map[string]interface{}{
		"login_name": config.Login.Username,
		"password":   config.Login.Password,
	}

	am := authmethods.NewClient(client)

	at, err := am.Authenticate(context.Background(), config.Login.AuthMethod, "login", credentials)
	if err != nil {
		panic(err)
	}

	var token string
	if x, found := at.Attributes["token"]; found {
		token, _ = x.(string)
	}

	err = os.Setenv("BOUNDARY_TOKEN", token)
	if err != nil {
		panic(err)
	}
	client.SetToken(token)

	scopes := scopes.NewClient(client)
	ctx := context.Background()
	scopeResults, err := scopes.List(ctx, config.Login.Scope)
	if err != nil {
		panic(err)
	}

	catalogsClient := hostcatalogs.NewClient(client)

	var catalogs []*hostcatalogs.HostCatalog

	for _, item := range scopeResults.Items {
		catalogsResult, listErr := catalogsClient.List(ctx, item.Id)
		if listErr != nil {
			panic(listErr)
		}

		catalogs = append(catalogs, catalogsResult.Items...)
	}

	var hostsList []*hosts.Host

	hostClient := hosts.NewClient(client)
	for _, hostCatalog := range catalogs {
		hostsResult, listErr := hostClient.List(ctx, hostCatalog.Id)
		if listErr != nil {
			panic(listErr)
		}

		hostsList = append(hostsList, hostsResult.Items...)
	}

	if len(hostsList) == 0 {
		fmt.Println("No Boundary hosts available")
		return
	}

	sort.Slice(hostsList[:], func(i, j int) bool {
		return hostsList[i].Name < hostsList[j].Name
	})

	sites, inventoryErr := fetchInventorySites(cmd)
	if inventoryErr != nil {
		fmt.Println("Inventory metadata unavailable; searching Boundary host names only.")
		if Verbose {
			fmt.Printf("Inventory lookup error: %v\n", inventoryErr)
		}
	}

	targetOptions := buildSSHTargetOptions(hostsList, sites)
	filteredTargets := targetOptions
	if len(args) == 1 {
		filteredTargets = filterAndRankSSHTargets(targetOptions, args[0])

		switch len(filteredTargets) {
		case 0:
			fmt.Printf("No hosts matched %q\n", args[0])
			return
		case 1:
			fmt.Printf("Connecting to host: %s\n", filteredTargets[0].HostName)
			connectToHost(boundaryPath, boundaryUrl, filteredTargets[0].Host)
			return
		}
	}

	fmt.Printf("Total of %v hosts available\n", len(hostsList))
	if len(args) == 1 {
		fmt.Printf("%v hosts match %q\n\n", len(filteredTargets), args[0])
	} else {
		fmt.Println()
	}

	detailsTemplate := `
{{ "Server:" | faint }}	{{ .ServerName }}
{{ "Boundary host:" | faint }}	{{ .HostName }}
{{ "Host ID:" | faint }}	{{ .HostID }}
{{ "Sites:" | faint }}	{{ .SitePreview }}
{{ "Metadata:" | faint }}	{{ .JoinStatus }}
`

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "> {{ .Title | cyan }}{{ if .Meta }} {{ .Meta | faint }}{{ end }}",
		Inactive: "  {{ .Title }}{{ if .Meta }} {{ .Meta | faint }}{{ end }}",
		Selected: "Selected host: {{ .HostName | cyan }}",
		Details:  detailsTemplate,
	}

	searcher := func(input string, index int) bool {
		target := filteredTargets[index]
		return sshTargetMatchesInput(target, input)
	}

	prompt := promptui.Select{
		Label:     "Search by site or server to connect",
		Items:     filteredTargets,
		Templates: templates,
		Size:      selectSize(len(filteredTargets)),
		Searcher:  searcher,
		Stdout:    &bellSkipper{},
	}

	i, _, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	fmt.Printf("Connecting to host: %s\n", filteredTargets[i].HostName)
	connectToHost(boundaryPath, boundaryUrl, filteredTargets[i].Host)
}

func connectToHost(boundaryPath, boundaryURL string, host *hosts.Host) {
	command := exec.Command(
		boundaryPath,
		"connect",
		"ssh",
		"-target-name=ssh",
		"-target-scope-id="+host.Scope.Id,
		"-addr="+boundaryURL,
		"-username="+sshUsername,
		"-host-id="+host.Id,
		"-token=env://BOUNDARY_TOKEN",
	)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	_ = command.Run()
}

type bellSkipper struct{}

// This solves the issue where manifoldco/promptui sends out an annoying bell sound whenever you hit a key (on MacOS)
// Write implements an io.WriterCloser over os.Stderr, but it skips the terminal bell character.
func (bs *bellSkipper) Write(b []byte) (int, error) {
	const charBell = 7 // c.f. readline.CharBell
	if len(b) == 1 && b[0] == charBell {
		return 0, nil
	}
	return os.Stderr.Write(b)
}

// Close implements an io.WriterCloser over os.Stderr.
func (bs *bellSkipper) Close() error {
	return nil
}

func init() {
	rootCmd.AddCommand(sshCmd)
	rootCmd.AddCommand(loginCmd)

	sshCmd.PersistentFlags().StringVar(
		&sshUsername,
		"ssh_username",
		"root",
		"Username used to connect with the server",
	)

	loginCmd.PersistentFlags().StringVar(
		&sshUsername,
		"ssh_username",
		"root",
		"Username used to connect with the server",
	)
}
