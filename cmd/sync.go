/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

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
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var syncType string

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("sync called with %v\n", args[0])
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("The type argument is required")
		} else {
			if syncType != "database" && syncType != "files" {
				return errors.New("The type argument either has to be 'database' or 'files'")
			}
			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	//syncCmd.

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	syncCmd.PersistentFlags().StringVar(&syncType, "type", "", "Either 'database' or 'files'")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	syncCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
