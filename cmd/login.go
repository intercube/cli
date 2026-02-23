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
	"strings"
)

var sshUsername = "root"

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login [host-filter]",
	Short: "Login with your API token",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		boundaryUrl := config.Login.InstanceUrl
		if boundaryUrl == "" {
			boundaryUrl = "https://controller.boundary.intercube.cloud"
		}

		boundaryPath, err := exec.LookPath("boundary")
		if err != nil {
			panic("Boundary not installed on this machine. Download & install boundary before using the login function (https://learn.hashicorp.com/tutorials/boundary/getting-started-install)")
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
		scopeResults, _ := scopes.List(ctx, config.Login.Scope)

		catalogsClient := hostcatalogs.NewClient(client)

		var catalogs []*hostcatalogs.HostCatalog

		for _, item := range scopeResults.Items {
			catalogsResult, _ := catalogsClient.List(ctx, item.Id)
			catalogs = append(catalogs, catalogsResult.Items...)
		}

		var hostsList []*hosts.Host

		hostClient := hosts.NewClient(client)
		for _, hostCatalog := range catalogs {
			hostsResult, _ := hostClient.List(ctx, hostCatalog.Id)
			hostsList = append(hostsList, hostsResult.Items...)
		}

		sort.Slice(hostsList[:], func(i, j int) bool {
			return hostsList[i].Name < hostsList[j].Name
		})

		filteredHosts := hostsList
		if len(args) == 1 {
			searchTerm := strings.ToLower(strings.TrimSpace(args[0]))
			filteredHosts = make([]*hosts.Host, 0, len(hostsList))

			for _, host := range hostsList {
				if strings.Contains(strings.ToLower(host.Name), searchTerm) {
					filteredHosts = append(filteredHosts, host)
				}
			}

			switch len(filteredHosts) {
			case 0:
				fmt.Printf("No hosts matched %q\n", args[0])
				return
			case 1:
				fmt.Printf("Connecting to host: %s\n", filteredHosts[0].Name)
				connectToHost(boundaryPath, boundaryUrl, filteredHosts[0])
				return
			}
		}

		fmt.Printf("Total of %v hosts available\n", len(hostsList))
		if len(args) == 1 {
			fmt.Printf("%v hosts match %q\n\n", len(filteredHosts), args[0])
		} else {
			fmt.Println()
		}

		templates := &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "\U0001F9CA {{ .Name | red }}",
			Inactive: "  {{ .Name | cyan }}",
			Selected: "\U0001F9CA {{ .Name | red | cyan }}",
			Details: `
--------- Host ----------
{{ "Name:" | faint }}	{{ .Name }}
{{range $key, $value := .Attributes}}{{ $key }}{{ ":" | faint }}	{{ $value }}{{end}}
`,
		}

		searcher := func(input string, index int) bool {
			host := filteredHosts[index]
			name := strings.Replace(strings.ToLower(host.Name), " ", "", -1)
			input = strings.Replace(strings.ToLower(input), " ", "", -1)

			return strings.Contains(name, input)
		}

		prompt := promptui.Select{
			Label:     "Which host would you like to connect to?",
			Items:     filteredHosts,
			Templates: templates,
			Size:      8,
			Searcher:  searcher,
			Stdout:    &bellSkipper{},
		}

		i, _, err := prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		fmt.Printf("Connecting to host: %s\n", filteredHosts[i].Name)
		connectToHost(boundaryPath, boundaryUrl, filteredHosts[i])
	},
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
	return os.Stderr.Close()
}

func init() {
	rootCmd.AddCommand(loginCmd)

	loginCmd.PersistentFlags().StringVar(
		&sshUsername,
		"ssh_username",
		"root",
		"Username used to connect with the server",
	)
}
