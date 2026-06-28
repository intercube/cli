package auth

import "time"

type Session struct {
	AccessToken    string    `json:"access_token"`
	RefreshToken   string    `json:"refresh_token"`
	TokenType      string    `json:"token_type"`
	Scope          string    `json:"scope"`
	ExpiresAt      time.Time `json:"expires_at"`
	OrganizationID string    `json:"organization_id,omitempty"`
	KnownOrgIDs    []string  `json:"known_org_ids,omitempty"`
}

func (s *Session) ExpiresSoon(leeway time.Duration) bool {
	if s == nil || s.AccessToken == "" {
		return true
	}

	if s.ExpiresAt.IsZero() {
		return true
	}

	return time.Now().Add(leeway).After(s.ExpiresAt)
}
