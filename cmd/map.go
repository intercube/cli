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
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Maps files based on config yaml file",
	Run: func(cmd *cobra.Command, args []string) {
		for _, mapping := range config.Mappings {
			symlink(mapping.From, mapping.To)
		}
	},
}

func init() {
	rootCmd.AddCommand(mapCmd)
}

func symlink(from string, to string) {
	if !fileExists(to) {
		err := os.Symlink(from, to)
		if err != nil {
			panic(fmt.Errorf("Unable to map: %s \n", err))
		} else {
			fmt.Printf("Mapped %v to %v\n", from, to)
		}
	} else {
		fmt.Print(fmt.Errorf("Destination file %v already exists\n", to))
	}
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
