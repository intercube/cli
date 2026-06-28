package cmd

import (
	"errors"
	"fmt"
	"strings"

	authutil "github.com/intercube/cli/util/auth"
	"github.com/spf13/cobra"
)

var authOrgShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show selected organization id",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := authutil.NewSessionStore("intercube-cli")
		if err != nil {
			return err
		}

		session, err := store.Load(cmd.Context())
		if err != nil {
			if errors.Is(err, authutil.ErrNoSession) {
				return errors.New("you are not authenticated, run `intercube auth login` first")
			}

			return err
		}

		orgID := strings.TrimSpace(session.OrganizationID)
		if orgID == "" {
			fmt.Println("No organization selected. Run `intercube auth org select`.")
			return nil
		}

		fmt.Println(orgID)
		return nil
	},
}

func init() {
	authOrgCmd.AddCommand(authOrgShowCmd)
}
