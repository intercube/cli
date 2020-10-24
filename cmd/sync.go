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
	"errors"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)
import "github.com/asaskevich/govalidator"

var syncType string
var fromServer string
var filesPath string
var remoteUser string

var syncCmd = &cobra.Command{
	Use:   "sync [files|database] [destination]",
	Short: "Syncs files or database from one of your servers to another",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		sync()
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("The type argument is required")
		} else {
			syncType = args[0]
			if len(args) > 1 {
				fromServer = args[1]
			} else {
				fromServer = viper.GetStringMapString("file_syncing")["from_server"]
			}

			if len(args) > 2 {
				filesPath = args[2]
			} else {
				filesPath = viper.GetStringMapString("file_syncing")["path"]
			}

			if len(args) > 3 {
				remoteUser = args[3]
			} else {
				remoteUser = viper.GetString("remote_user")
			}

			if syncType != "database" && syncType != "files" {
				return errors.New("The type argument either has to be 'database' or 'files'")
			}

			if !govalidator.IsDNSName(fromServer) {
				return errors.New("Provide a valid destination hostname")
			}

			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.PersistentFlags().StringVar(&syncType, "type", "", "Either 'database' or 'files'")
	syncCmd.PersistentFlags().StringVar(&fromServer, "from_server", "", "Provide the hostname of the server to pull the data from")
	syncCmd.PersistentFlags().StringVar(&filesPath, "files_path", "", "Provide the location of where the files are located")
	syncCmd.PersistentFlags().StringVar(&remoteUser, "remote_user", "", "Provide the user to connect to the server")
}

func sync() {
	if syncType == "files" {
		syncFiles(fromServer, filesPath, remoteUser)
	} else {
		//syncDatabase()
	}
}
