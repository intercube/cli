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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"net/url"
	"os/exec"
	"strings"
)

// magentoCmd represents the magento command
var magentoCmd = &cobra.Command{
	Use:   "magento",
	Short: "Executes Magento 2 intercube commands",
	Run: func(cmd *cobra.Command, args []string) {
		displaySubCommands(cmd.Commands())
	},
}

var appendDomain = ".mycube.dev"

var baseUrlCmd = &cobra.Command{
	Use:   "base-urls",
	Short: "Sets base URL based on the current web/secure/base_url value",
	Run: func(cmd *cobra.Command, args []string) {
		command := executeN98Command(
			"config:store:get",
			"web/secure/base_url",
			"--format=json",
		)

		var result map[string]json.RawMessage
		err := json.Unmarshal(command.Bytes(), &result)

		if err != nil {
			panic(err)
		}

		for _, item := range result {
			var baseUrl BaseUrl
			err = json.Unmarshal(item, &baseUrl)

			if err != nil {
				panic(err)
			}

			fragments, _ := url.Parse(baseUrl.Value)

			if !strings.HasSuffix(fragments.Host, ".mycube.dev") {
				fragments.Host = fragments.Host + ".mycube.dev"
				fmt.Printf(
					"/usr/local/bin/n98-magerun2 config:store:set web/secure/base_url --scope=%v --scope-id=%v %v\n",
					baseUrl.Scope,
					baseUrl.ScopeId,
					fragments.String())

				executeN98Command(
					"config:store:set",
					"web/secure/base_url",
					fmt.Sprintf("--scope=%v", baseUrl.Scope),
					fmt.Sprintf("--scope-id=%v", baseUrl.ScopeId),
					fmt.Sprintf("%v", fragments.String()))

				executeN98Command(
					"config:store:set",
					"web/unsecure/base_url",
					fmt.Sprintf("--scope=%v", baseUrl.Scope),
					fmt.Sprintf("--scope-id=%v", baseUrl.ScopeId),
					fmt.Sprintf("%v", fragments.String()))
			}
		}
	},
}

func executeN98Command(args ...string) bytes.Buffer {
	command := exec.Command("n98-magerun2", args...)

	var out bytes.Buffer
	var outErr bytes.Buffer

	command.Stdout = &out
	command.Stderr = &outErr

	err := command.Run()
	if err != nil {
		if Verbose {
			println(command.String())
		}

		log.Fatal(outErr.String())
	}

	return out
}

func init() {
	rootCmd.AddCommand(magentoCmd)
	magentoCmd.AddCommand(baseUrlCmd)

	magentoCmd.PersistentFlags().StringVar(
		&appendDomain,
		"append_domain",
		".mycube.dev",
		"Domain that will be appended to current value of web/secure/base_url")
}

type BaseUrl struct {
	Path    string
	Scope   string
	ScopeId string `json:"Scope-ID"`
	Value   string
}
