package cmd

import (
	"fmt"

	"github.com/intercube/cli/util/appconfig"
	authutil "github.com/intercube/cli/util/auth"
	"github.com/spf13/cobra"
)

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear local API auth session",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := authutil.NewSessionStore("intercube-cli")
		if err != nil {
			return err
		}

		session, err := store.Load(cmd.Context())
		if err == nil && session != nil {
			if appconfig.ValidateClerk() == nil {
				clerkClient := &authutil.ClerkClient{
					Issuer:   appconfig.ClerkIssuer,
					ClientID: appconfig.ClerkClientID,
				}
				_ = clerkClient.RevokeRefreshToken(cmd.Context(), session.RefreshToken)
			}
		}

		if err := store.Clear(cmd.Context()); err != nil {
			return err
		}

		fmt.Println("Signed out.")
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLogoutCmd)
}
