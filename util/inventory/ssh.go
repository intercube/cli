package inventory

import (
	"context"
	"net/http"
)

type SSHAccess struct {
	Host     string      `json:"host"`
	Port     int         `json:"port"`
	Username string      `json:"username"`
	Targets  []SSHTarget `json:"targets"`
}

type SSHTarget struct {
	Address        string   `json:"address"`
	Name           string   `json:"name"`
	SiteID         *int     `json:"site_id"`
	ServerID       int      `json:"server_id"`
	ServerName     string   `json:"server_name"`
	OrganizationID string   `json:"organization_id"`
	LinuxUser      string   `json:"linux_user"`
	Domains        []string `json:"domains"`
	Root           bool     `json:"root"`
}

// SSHAccess uses identity-scoped discovery, independent of the selected organization.
func (c *Client) SSHAccess(ctx context.Context) (*SSHAccess, error) {
	var access SSHAccess
	if err := c.doJSON(ctx, http.MethodPost, "/me/ssh-access", nil, &access); err != nil {
		return nil, err
	}
	return &access, nil
}
