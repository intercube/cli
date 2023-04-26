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

const boundaryUrl = "https://controller.boundary.intercube.cloud"

var sshUsername = "root"

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login with your API token",
	Run: func(cmd *cobra.Command, args []string) {
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

		fmt.Printf("Total of %v hosts available\n\n", len(hostsList))

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
			host := hostsList[index]
			name := strings.Replace(strings.ToLower(host.Name), " ", "", -1)
			input = strings.Replace(strings.ToLower(input), " ", "", -1)

			return strings.Contains(name, input)
		}

		prompt := promptui.Select{
			Label:     "Which host would you like to connect to?",
			Items:     hostsList,
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

		fmt.Printf("Connecting to host: %s\n", hostsList[i].Name)

		command := exec.Command(
			boundaryPath,
			"connect",
			"ssh",
			"-target-name=ssh",
			"-target-scope-id="+hostsList[i].Scope.Id,
			"-addr="+boundaryUrl,
			"-username="+sshUsername,
			"-host-id="+hostsList[i].Id,
			"-token=env://BOUNDARY_TOKEN",
		)
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		_ = command.Run()
	},
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
