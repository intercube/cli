package appconfig

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

var ClerkIssuer = "https://clerk.intercube.io"
var ClerkClientID = "Oi68oAK1xuK1088Z"
var ClerkAudience = ""
var ClerkScopes = "openid profile email offline_access"
var ClerkCallbackPort = "8976"
var InventoryAPIBaseURL = "http://inventory-nexus.dev-c8s.intercube.dev/"
var OrganizationID = ""

const (
	EnvClerkIssuer       = "INTERCUBE_AUTH_CLERK_ISSUER"
	EnvClerkClientID     = "INTERCUBE_AUTH_CLERK_CLIENT_ID"
	EnvClerkAudience     = "INTERCUBE_AUTH_CLERK_AUDIENCE"
	EnvClerkScopes       = "INTERCUBE_AUTH_CLERK_SCOPES"
	EnvClerkCallbackPort = "INTERCUBE_AUTH_CLERK_CALLBACK_PORT"
	EnvInventoryAPIURL   = "INTERCUBE_INVENTORY_API_BASE_URL"
	EnvOrganizationID    = "INTERCUBE_ORGANIZATION_ID"
)

func LoadFromEnv() {
	if value := strings.TrimSpace(os.Getenv(EnvClerkIssuer)); value != "" {
		ClerkIssuer = value
	}

	if value := strings.TrimSpace(os.Getenv(EnvClerkClientID)); value != "" {
		ClerkClientID = value
	}

	if value := strings.TrimSpace(os.Getenv(EnvClerkAudience)); value != "" {
		ClerkAudience = value
	}

	if value := strings.TrimSpace(os.Getenv(EnvClerkScopes)); value != "" {
		ClerkScopes = value
	}

	if value := strings.TrimSpace(os.Getenv(EnvClerkCallbackPort)); value != "" {
		ClerkCallbackPort = value
	}

	if value := strings.TrimSpace(os.Getenv(EnvInventoryAPIURL)); value != "" {
		InventoryAPIBaseURL = value
	}

	if value := strings.TrimSpace(os.Getenv(EnvOrganizationID)); value != "" {
		OrganizationID = value
	}
}

func ValidateClerk() error {
	missing := make([]string, 0, 2)
	if strings.TrimSpace(ClerkIssuer) == "" {
		missing = append(missing, "ClerkIssuer")
	}

	if strings.TrimSpace(ClerkClientID) == "" {
		missing = append(missing, "ClerkClientID")
	}

	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf("missing internal auth config: %s", strings.Join(missing, ", "))
}

func ValidateInventory() error {
	if strings.TrimSpace(InventoryAPIBaseURL) == "" {
		return fmt.Errorf("missing internal inventory config: InventoryAPIBaseURL")
	}

	return nil
}
func ParsedCallbackPort() int {
	port, err := strconv.Atoi(strings.TrimSpace(ClerkCallbackPort))
	if err != nil || port < 1 || port > 65535 {
		return 8976
	}

	return port
}
