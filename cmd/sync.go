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
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type syncMode struct {
	runFiles    bool
	runDatabase bool
	dryRun      bool
	autoApprove bool
}

var (
	syncOnlyFiles    bool
	syncOnlyDatabase bool
	syncAll          bool
	syncDryRun       bool
	syncYes          bool
	syncSiteID       string
	syncOrgID        string
)

var syncCmd = &cobra.Command{
	Use:   "sync [env-or-host]",
	Short: "Sync files and MySQL data to another environment",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := ""
		if len(args) == 1 {
			query = strings.TrimSpace(args[0])
		}

		settings, err := loadSyncSettings()
		if err != nil {
			return err
		}

		mode, err := resolveSyncMode()
		if err != nil {
			return err
		}

		inventoryClient, _, err := newInventoryClient(cmd, syncOrgID)
		if err != nil {
			return err
		}

		target, source, err := resolveSyncTarget(cmd, inventoryClient, query, strings.TrimSpace(syncSiteID))
		if err != nil {
			return err
		}

		fmt.Printf("Source: %s\n", source.DisplayName)
		fmt.Printf("Target: %s (%s@%s:%d)\n", target.DisplayName, target.Username, target.Host, target.Port)

		if mode.runFiles {
			if err := runFileSync(cmd, target, &settings, mode.dryRun); err != nil {
				return err
			}
		}

		if mode.runDatabase {
			if err := runDatabaseSync(cmd, target, &settings, mode.dryRun, mode.autoApprove); err != nil {
				return err
			}
		}

		fmt.Println("Sync finished.")
		return nil
	},
}

func resolveSyncMode() (syncMode, error) {
	mode := syncMode{
		dryRun:      syncDryRun,
		autoApprove: syncYes,
	}

	if syncAll {
		mode.runFiles = true
		mode.runDatabase = true
		return mode, nil
	}

	if syncOnlyFiles {
		mode.runFiles = true
	}

	if syncOnlyDatabase {
		mode.runDatabase = true
	}

	if mode.runFiles || mode.runDatabase {
		return mode, nil
	}

	mode.runFiles = true
	mode.runDatabase = true

	return mode, nil
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().BoolVar(&syncOnlyFiles, "files", false, "sync files only")
	syncCmd.Flags().BoolVar(&syncOnlyDatabase, "database", false, "sync database only")
	syncCmd.Flags().BoolVar(&syncAll, "all", false, "sync both files and database")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "print planned commands without executing")
	syncCmd.Flags().BoolVar(&syncYes, "yes", false, "skip confirmation prompts")
	syncCmd.Flags().StringVar(&syncSiteID, "site-id", "", "target site id override")
	syncCmd.Flags().StringVar(&syncOrgID, "org-id", "", "organization id for inventory requests")
}
