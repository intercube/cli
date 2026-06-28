package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/intercube/cli/util/appconfig"
	authutil "github.com/intercube/cli/util/auth"
	"github.com/spf13/cobra"
)

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in with Clerk in your browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := appconfig.ValidateClerk(); err != nil {
			return fmt.Errorf("%w (set via env/.env or build-time)", err)
		}

		store, err := authutil.NewSessionStore("intercube-cli")
		if err != nil {
			return err
		}

		var previousSession *authutil.Session
		storedSession, loadErr := store.Load(cmd.Context())
		if loadErr == nil {
			previousSession = storedSession
		} else if !errors.Is(loadErr, authutil.ErrNoSession) {
			return loadErr
		}

		clerkClient := &authutil.ClerkClient{
			Issuer:       appconfig.ClerkIssuer,
			ClientID:     appconfig.ClerkClientID,
			Audience:     appconfig.ClerkAudience,
			Scopes:       appconfig.ClerkScopes,
			CallbackPort: appconfig.ParsedCallbackPort(),
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
		defer cancel()

		fmt.Println("Opening browser for Clerk sign-in...")
		session, err := clerkClient.Login(ctx)
		if err != nil {
			return err
		}

		if previousSession != nil {
			session.OrganizationID = strings.TrimSpace(previousSession.OrganizationID)
			for _, known := range previousSession.KnownOrgIDs {
				session.KnownOrgIDs = addKnownOrganizationID(session.KnownOrgIDs, known)
			}
		}

		if appconfig.OrganizationID != "" {
			session.KnownOrgIDs = addKnownOrganizationID(session.KnownOrgIDs, appconfig.OrganizationID)
		}

		session.KnownOrgIDs = addKnownOrganizationID(session.KnownOrgIDs, session.OrganizationID)

		if err := store.Save(ctx, session); err != nil {
			return err
		}

		fmt.Printf("Authenticated. Session expires at %s\n", session.ExpiresAt.Format(time.RFC3339))

		if selectErr := runAuthOrgSelect(cmd, nil, true); selectErr != nil {
			return selectErr
		}

		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
}
