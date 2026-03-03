package cmd

import "github.com/spf13/cobra"

var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Manage organization resources",
}

func init() {
	rootCmd.AddCommand(orgCmd)
}
