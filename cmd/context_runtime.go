package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/intercube/cli/util/appconfig"
	authutil "github.com/intercube/cli/util/auth"
	"github.com/intercube/cli/util/contextconfig"
	"github.com/spf13/cobra"
)

var runtimeContext contextconfig.Runtime
var contextOverride string

func isNonInteractiveMode() bool {
	if config.Behavior.NonInteractive {
		return true
	}

	return runtimeContext.NonInteractive
}

func ensureInteractiveMode(action string) error {
	if isNonInteractiveMode() {
		return fmt.Errorf("%s requires interactive input; pass explicit flags or set defaults in context config", action)
	}

	if !stdinIsTerminal() {
		return fmt.Errorf("%s requires a terminal", action)
	}

	return nil
}

func resolveOrganizationID(cmd *cobra.Command, organizationOverride string) (string, string, error) {
	store, err := authutil.NewSessionStore("intercube-cli")
	if err != nil {
		return "", "", err
	}

	sessionOrg := ""
	if session, sessionErr := store.Load(cmd.Context()); sessionErr == nil {
		sessionOrg = strings.TrimSpace(session.OrganizationID)
	}

	resolved := contextconfig.ResolveValue(contextconfig.Inputs{
		FlagValue:           organizationOverride,
		PreferredEnvValue:   strings.TrimSpace(os.Getenv(appconfig.EnvOrganizationIDAlt)),
		LegacyEnvValue:      strings.TrimSpace(os.Getenv(appconfig.EnvOrganizationID)),
		ContextConfigValue:  strings.TrimSpace(config.Context.OrgID),
		SessionDefaultValue: sessionOrg,
		UserDefaultValue:    strings.TrimSpace(appconfig.OrganizationID),
	})

	return resolved.Value, resolved.Source, nil
}

func resolveSiteID(siteOverride string) string {
	resolved := contextconfig.ResolveValue(contextconfig.Inputs{
		FlagValue:          siteOverride,
		PreferredEnvValue:  strings.TrimSpace(os.Getenv(appconfig.EnvSiteID)),
		ContextConfigValue: strings.TrimSpace(config.Context.SiteID),
	})

	return resolved.Value
}

func contextOrgDefault() string {
	resolved := contextconfig.ResolveValue(contextconfig.Inputs{
		PreferredEnvValue:  strings.TrimSpace(os.Getenv(appconfig.EnvOrganizationIDAlt)),
		LegacyEnvValue:     strings.TrimSpace(os.Getenv(appconfig.EnvOrganizationID)),
		ContextConfigValue: strings.TrimSpace(config.Context.OrgID),
		UserDefaultValue:   strings.TrimSpace(appconfig.OrganizationID),
	})

	return resolved.Value
}
