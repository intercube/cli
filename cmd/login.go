/*
Copyright © 2020 Intercube <opensource@intercube.io>

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
	"github.com/hashicorp/boundary/api/authmethods"
	"github.com/hashicorp/boundary/api/hostcatalogs"
	"github.com/hashicorp/boundary/api/hosts"
	"github.com/hashicorp/boundary/api/scopes"
	"github.com/spf13/cobra"

	"github.com/hashicorp/boundary/api"
)

const boundaryUrl = "http://controller.boundary.intercube.cloud:9200"

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login with your API token",
	Run: func(cmd *cobra.Command, args []string) {
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

		at, err := am.Authenticate(context.Background(), config.Login.AuthMethod, credentials)
		if err != nil {
			panic(err)
		}

		client.SetToken(at.Item.Token)

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

		fmt.Printf("Total of %v hosts found\n", len(hostsList))
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
