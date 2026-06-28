package cmd

import "github.com/spf13/cobra"

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate for API calls",
}

func init() {
	rootCmd.AddCommand(authCmd)
}
