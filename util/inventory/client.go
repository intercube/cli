package inventory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	authutil "github.com/intercube/cli/util/auth"
)

type Client struct {
	BaseURL    string
	OrgID      string
	Store      *authutil.SessionStore
	Clerk      *authutil.ClerkClient
	HTTPClient *http.Client
	mu         sync.Mutex
}

type SiteServer struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	MainDomain string `json:"maindomain"`
	ServerID   string `json:"serverid"`
	ServerName string `json:"servername"`
}

type AuthorizationKey struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Comment string `json:"comment"`
}

type OrganizationSSHKey struct {
	ID             string `json:"id"`
	Content        string `json:"content"`
	Comment        string `json:"comment"`
	ExpirationDate string `json:"expirationdate"`
	CreatedAtUTC   string `json:"createdatutc"`
	UpdatedAtUTC   string `json:"updatedatutc"`
	SiteIDs        []int  `json:"siteids"`
}

type EnvironmentVariable struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

type Redirect struct {
	ID         string `json:"id"`
	Domain     string `json:"domain"`
	ReturnCode int    `json:"returncode"`
	Location   string `json:"location"`
	Value      string `json:"value"`
}

type CurrentUserOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Role string `json:"role"`
}

type CreateAuthorizationKeyRequest struct {
	Content string `json:"content"`
	Comment string `json:"comment,omitempty"`
}

type OrganizationSSHKeyRequest struct {
	Content        string `json:"content"`
	Comment        string `json:"comment,omitempty"`
	ExpirationDate string `json:"expirationdate,omitempty"`
}

type EnvironmentVariableMutate struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

type RedirectMutate struct {
	Domain     string `json:"domain"`
	ReturnCode int    `json:"returncode"`
	Location   string `json:"location"`
	Value      string `json:"value"`
}

type pagedMeta struct {
	Offset      int  `json:"offset"`
	Limit       int  `json:"limit"`
	Count       int  `json:"count"`
	Total       int  `json:"total"`
	HasNext     bool `json:"hasnext"`
	HasPrevious bool `json:"hasprevious"`
}

type pagedResponse[T any] struct {
	Items []T       `json:"items"`
	Meta  pagedMeta `json:"meta"`
}

func NewClient(baseURL, organizationID string, store *authutil.SessionStore, clerk *authutil.ClerkClient) *Client {
	return &Client{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		OrgID:   strings.TrimSpace(organizationID),
		Store:   store,
		Clerk:   clerk,
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *Client) ListSites(ctx context.Context) ([]SiteServer, error) {
	sites := make([]SiteServer, 0)
	offset := 0
	limit := 100

	for {
		var page pagedResponse[SiteServer]
		path := fmt.Sprintf("/sites?offset=%d&limit=%d", offset, limit)
		if err := c.doJSON(ctx, http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}

		sites = append(sites, page.Items...)
		if !page.Meta.HasNext {
			break
		}

		offset += limit
	}

	return sites, nil
}

func (c *Client) ListSiteAuthorizationKeys(ctx context.Context, siteID string) ([]AuthorizationKey, error) {
	var keys []AuthorizationKey
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/site/%s/authorization-keys", siteID), nil, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}

func (c *Client) ListOrganizationSSHKeys(ctx context.Context) ([]OrganizationSSHKey, error) {
	keys := make([]OrganizationSSHKey, 0)
	offset := 0
	limit := 100

	for {
		var page pagedResponse[OrganizationSSHKey]
		path := fmt.Sprintf("/organization/ssh-keys?offset=%d&limit=%d", offset, limit)
		if err := c.doJSON(ctx, http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}

		keys = append(keys, page.Items...)
		if !page.Meta.HasNext {
			break
		}

		offset += limit
	}

	return keys, nil
}

func (c *Client) CreateOrganizationSSHKey(ctx context.Context, request OrganizationSSHKeyRequest) (*OrganizationSSHKey, error) {
	var key OrganizationSSHKey
	if err := c.doJSON(ctx, http.MethodPost, "/organization/ssh-keys", request, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

func (c *Client) AssignOrganizationSSHKeyToSite(ctx context.Context, keyID, siteID string) error {
	path := fmt.Sprintf("/organization/ssh-keys/%s/sites/%s", keyID, siteID)
	return c.doJSON(ctx, http.MethodPost, path, nil, nil)
}

func (c *Client) UnassignOrganizationSSHKeyFromSite(ctx context.Context, keyID, siteID string, deleteIfUnassigned bool) error {
	path := fmt.Sprintf("/organization/ssh-keys/%s/sites/%s", keyID, siteID)
	if deleteIfUnassigned {
		path += "?deleteIfUnassigned=true"
	}

	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) ListCurrentUserOrganizations(ctx context.Context) ([]CurrentUserOrganization, error) {
	var organizations []CurrentUserOrganization
	if err := c.doJSON(ctx, http.MethodGet, "/me/organizations", nil, &organizations); err != nil {
		return nil, err
	}

	return organizations, nil
}

func (c *Client) ListSiteEnvironmentVariables(ctx context.Context, siteID string) ([]EnvironmentVariable, error) {
	var variables []EnvironmentVariable
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/site/%s/environment-variables", siteID), nil, &variables); err != nil {
		return nil, err
	}

	return variables, nil
}

func (c *Client) CreateSiteEnvironmentVariable(ctx context.Context, siteID string, request EnvironmentVariableMutate) (*EnvironmentVariable, error) {
	var variable EnvironmentVariable
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/site/%s/environment-variables", siteID), request, &variable); err != nil {
		return nil, err
	}

	return &variable, nil
}

func (c *Client) UpdateSiteEnvironmentVariable(ctx context.Context, siteID, variableID string, request EnvironmentVariableMutate) error {
	path := fmt.Sprintf("/site/%s/environment-variables/%s", siteID, variableID)
	return c.doJSON(ctx, http.MethodPut, path, request, nil)
}

func (c *Client) DeleteSiteEnvironmentVariable(ctx context.Context, siteID, variableID string) error {
	path := fmt.Sprintf("/site/%s/environment-variables/%s", siteID, variableID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) ListSiteRedirects(ctx context.Context, siteID string) ([]Redirect, error) {
	redirects := make([]Redirect, 0)
	offset := 0
	limit := 100

	for {
		var page pagedResponse[Redirect]
		path := fmt.Sprintf("/site/%s/redirects?offset=%d&limit=%d", siteID, offset, limit)
		if err := c.doJSON(ctx, http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}

		redirects = append(redirects, page.Items...)
		if !page.Meta.HasNext {
			break
		}

		offset += limit
	}

	return redirects, nil
}

func (c *Client) CreateSiteRedirect(ctx context.Context, siteID string, request RedirectMutate) (*Redirect, error) {
	var redirect Redirect
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/site/%s/redirects", siteID), request, &redirect); err != nil {
		return nil, err
	}

	return &redirect, nil
}

func (c *Client) DeleteSiteRedirect(ctx context.Context, siteID, redirectID string) error {
	path := fmt.Sprintf("/site/%s/redirects/%s", siteID, redirectID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) CreateSiteAuthorizationKey(ctx context.Context, siteID string, request CreateAuthorizationKeyRequest) (*AuthorizationKey, error) {
	var key AuthorizationKey
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/site/%s/authorization-keys", siteID), request, &key); err != nil {
		return nil, err
	}

	return &key, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload interface{}, out interface{}) error {
	body, err := marshalPayload(payload)
	if err != nil {
		return err
	}

	response, responseBody, err := c.doRequest(ctx, method, path, body, false)
	if err != nil {
		return err
	}

	if response.StatusCode == http.StatusUnauthorized {
		response, responseBody, err = c.doRequest(ctx, method, path, body, true)
		if err != nil {
			return err
		}
	}

	if response.StatusCode >= 300 {
		return fmt.Errorf("inventory API %s %s failed with status %d: %s", method, path, response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	if out == nil || len(responseBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBody, out); err != nil {
		return err
	}

	return nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body []byte, forceRefresh bool) (*http.Response, []byte, error) {
	token, err := c.accessToken(ctx, forceRefresh)
	if err != nil {
		return nil, nil, err
	}

	url := c.BaseURL + ensureLeadingSlash(path)
	request, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}

	request.Header.Set("Authorization", "Bearer "+token)
	if c.OrgID != "" {
		request.Header.Set("X-Organization-Id", c.OrgID)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "intercube-cli")
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, nil, err
	}

	return response, responseBody, nil
}

func (c *Client) accessToken(ctx context.Context, forceRefresh bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	session, err := c.Store.Load(ctx)
	if err != nil {
		if errors.Is(err, authutil.ErrNoSession) {
			return "", errors.New("you are not authenticated, run `intercube auth login`")
		}

		return "", err
	}

	if forceRefresh || session.ExpiresSoon(60*time.Second) {
		refreshed, refreshErr := c.Clerk.RefreshSession(ctx, session)
		if refreshErr != nil {
			return "", fmt.Errorf("unable to refresh auth session: %w", refreshErr)
		}

		if saveErr := c.Store.Save(ctx, refreshed); saveErr != nil {
			return "", saveErr
		}

		session = refreshed
	}

	if strings.TrimSpace(session.AccessToken) == "" {
		return "", errors.New("missing access token, run `intercube auth login`")
	}

	return session.AccessToken, nil
}

func marshalPayload(payload interface{}) ([]byte, error) {
	if payload == nil {
		return nil, nil
	}

	return json.Marshal(payload)
}

func ensureLeadingSlash(path string) string {
	trimmed := strings.TrimSpace(path)
	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}

	return "/" + trimmed
}
