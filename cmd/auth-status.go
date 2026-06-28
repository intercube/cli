package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	authutil "github.com/intercube/cli/util/auth"
	"github.com/spf13/cobra"
)

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current API auth session status",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := authutil.NewSessionStore("intercube-cli")
		if err != nil {
			return err
		}

		session, err := store.Load(cmd.Context())
		if err != nil {
			if errors.Is(err, authutil.ErrNoSession) {
				fmt.Println("API auth session: not signed in")
				fmt.Println("Run `intercube auth login` to sign in.")
				return nil
			}

			return err
		}

		remaining := time.Until(session.ExpiresAt)
		if remaining <= 0 {
			fmt.Println("API auth session: expired")
		} else {
			fmt.Println("API auth session: active")
			fmt.Printf("Expires in: %s\n", remaining.Round(time.Second))
		}

		fmt.Printf("Expires at: %s\n", session.ExpiresAt.Format(time.RFC3339))
		if session.RefreshToken != "" {
			fmt.Println("Refresh token: present")
		} else {
			fmt.Println("Refresh token: missing")
		}

		selectedOrg := strings.TrimSpace(session.OrganizationID)
		if selectedOrg == "" {
			fmt.Println("Selected org: none")
		} else {
			fmt.Printf("Selected org: %s\n", selectedOrg)
		}

		fmt.Printf("Access token: %s\n", session.AccessToken)

		return nil
	},
}

func init() {
	authCmd.AddCommand(authStatusCmd)
}
