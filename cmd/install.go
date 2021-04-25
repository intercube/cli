/*
Copyright © 2021 Intercube <opensource@intercube.io>

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
	"io/ioutil"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Installs intercube requirements",
	Run: func(cmd *cobra.Command, args []string) {
		_, err := os.Stat("/usr/local/bin/intercube-rsync")
		if err != nil {
			response, err := http.Get("https://gist.githubusercontent.com/JKetelaar/cb8d31729c1f21d0f67734aeb029748d/raw/3e14950edf53d9e4a943af95ad137fe4537d6c7c/intercube-sync.sh") //use package "net/http"

			if err != nil {
				panic(err)
			}

			body, err := ioutil.ReadAll(response.Body)

			err = ioutil.WriteFile("/usr/local/bin/intercube-rsync", body, 0770)
			if err != nil {
				panic(err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
