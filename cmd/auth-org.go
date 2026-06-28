package cmd

import "github.com/spf13/cobra"

var authOrgCmd = &cobra.Command{
	Use:   "org",
	Short: "Manage selected organization context",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuthOrgSelect(cmd, nil, true)
	},
}

func init() {
	authCmd.AddCommand(authOrgCmd)
}
