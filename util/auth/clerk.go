package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type ClerkClient struct {
	Issuer       string
	ClientID     string
	Audience     string
	Scopes       string
	CallbackPort int
	HTTPClient   *http.Client
}

type oidcMetadata struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	RevocationEndpoint    string `json:"revocation_endpoint"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
	Error        string `json:"error"`
}

type callbackResult struct {
	Code  string
	Error string
	State string
}

func (c *ClerkClient) Login(ctx context.Context) (*Session, error) {
	if strings.TrimSpace(c.Issuer) == "" || strings.TrimSpace(c.ClientID) == "" {
		return nil, errors.New("missing Clerk issuer or client ID")
	}

	metadata, err := c.fetchMetadata(ctx)
	if err != nil {
		return nil, err
	}

	state, err := randomString(24)
	if err != nil {
		return nil, err
	}

	verifier, err := randomString(48)
	if err != nil {
		return nil, err
	}

	challenge := pkceChallenge(verifier)
	redirectURI := c.redirectURI()

	resultCh, shutdown, err := startCallbackServer(c.callbackPort(), state)
	if err != nil {
		return nil, err
	}
	defer shutdown(context.Background())

	authURL, err := c.buildAuthURL(metadata.AuthorizationEndpoint, redirectURI, state, challenge)
	if err != nil {
		return nil, err
	}

	if err := openBrowser(authURL); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		if result.Error != "" {
			return nil, fmt.Errorf("clerk login failed: %s", result.Error)
		}

		if result.State != state {
			return nil, errors.New("received invalid auth state")
		}

		return c.exchangeCode(ctx, metadata.TokenEndpoint, result.Code, verifier, redirectURI)
	}
}

func (c *ClerkClient) RefreshSession(ctx context.Context, session *Session) (*Session, error) {
	if session == nil || strings.TrimSpace(session.RefreshToken) == "" {
		return nil, errors.New("missing refresh token")
	}

	metadata, err := c.fetchMetadata(ctx)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", session.RefreshToken)
	values.Set("client_id", c.ClientID)

	body, err := c.postForm(ctx, metadata.TokenEndpoint, values)
	if err != nil {
		return nil, err
	}

	if body.Error != "" {
		return nil, errors.New(body.Error)
	}

	if body.AccessToken == "" {
		return nil, errors.New("token refresh returned empty access token")
	}

	if body.RefreshToken == "" {
		body.RefreshToken = session.RefreshToken
	}

	updated := tokenResponseToSession(body)
	updated.OrganizationID = session.OrganizationID
	updated.KnownOrgIDs = append([]string(nil), session.KnownOrgIDs...)

	return updated, nil
}

func (c *ClerkClient) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return nil
	}

	metadata, err := c.fetchMetadata(ctx)
	if err != nil {
		return err
	}

	if strings.TrimSpace(metadata.RevocationEndpoint) == "" {
		return nil
	}

	values := url.Values{}
	values.Set("token", refreshToken)
	values.Set("token_type_hint", "refresh_token")
	values.Set("client_id", c.ClientID)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, metadata.RevocationEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.client().Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return fmt.Errorf("revocation failed with status %d", response.StatusCode)
	}

	return nil
}

func (c *ClerkClient) exchangeCode(ctx context.Context, tokenEndpoint, code, verifier, redirectURI string) (*Session, error) {
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("client_id", c.ClientID)
	values.Set("redirect_uri", redirectURI)
	values.Set("code_verifier", verifier)

	body, err := c.postForm(ctx, tokenEndpoint, values)
	if err != nil {
		return nil, err
	}

	if body.Error != "" {
		return nil, errors.New(body.Error)
	}

	if body.AccessToken == "" {
		return nil, errors.New("token exchange returned empty access token")
	}

	return tokenResponseToSession(body), nil
}

func (c *ClerkClient) postForm(ctx context.Context, endpoint string, values url.Values) (*tokenResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.client().Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var body tokenResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 && body.Error == "" {
		body.Error = fmt.Sprintf("request failed with status %d", response.StatusCode)
	}

	return &body, nil
}

func (c *ClerkClient) fetchMetadata(ctx context.Context) (*oidcMetadata, error) {
	issuer := strings.TrimRight(strings.TrimSpace(c.Issuer), "/")
	if issuer == "" {
		return nil, errors.New("missing Clerk issuer")
	}

	metadataURL := issuer + "/.well-known/openid-configuration"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, err
	}

	response, err := c.client().Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unable to load oidc metadata: status %d", response.StatusCode)
	}

	var metadata oidcMetadata
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return nil, err
	}

	if metadata.AuthorizationEndpoint == "" || metadata.TokenEndpoint == "" {
		return nil, errors.New("oidc metadata missing authorization or token endpoint")
	}

	return &metadata, nil
}

func (c *ClerkClient) buildAuthURL(endpoint, redirectURI, state, challenge string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	query.Set("response_type", "code")
	query.Set("client_id", c.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("scope", c.scopeString())
	query.Set("state", state)
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")

	if strings.TrimSpace(c.Audience) != "" {
		query.Set("audience", c.Audience)
	}

	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (c *ClerkClient) scopeString() string {
	scopes := strings.TrimSpace(c.Scopes)
	if scopes == "" {
		return "openid profile email offline_access public_metadata"
	}

	return scopes
}

func (c *ClerkClient) callbackPort() int {
	if c.CallbackPort > 0 {
		return c.CallbackPort
	}

	return 8976
}

func (c *ClerkClient) redirectURI() string {
	return fmt.Sprintf("http://127.0.0.1:%d/callback", c.callbackPort())
}

func (c *ClerkClient) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}

	return &http.Client{Timeout: 15 * time.Second}
}

func startCallbackServer(port int, expectedState string) (<-chan callbackResult, func(context.Context) error, error) {
	resultCh := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(writer http.ResponseWriter, request *http.Request) {
		sendResult := func(result callbackResult) {
			select {
			case resultCh <- result:
			default:
			}
		}

		query := request.URL.Query()
		state := query.Get("state")

		if query.Get("error") != "" {
			_, _ = writer.Write([]byte("Authentication failed. You can close this tab."))
			sendResult(callbackResult{Error: query.Get("error"), State: state})
			return
		}

		if query.Get("code") == "" {
			_, _ = writer.Write([]byte("Missing authorization code. You can close this tab."))
			sendResult(callbackResult{Error: "missing authorization code", State: state})
			return
		}

		if state != expectedState {
			_, _ = writer.Write([]byte("State mismatch. You can close this tab."))
			sendResult(callbackResult{Error: "state mismatch", State: state})
			return
		}

		_, _ = writer.Write([]byte("Authentication complete. You can close this tab and return to the terminal."))
		sendResult(callbackResult{Code: query.Get("code"), State: state})
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, nil, err
	}

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()

	return resultCh, server.Shutdown, nil
}

func randomString(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func pkceChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func openBrowser(url string) error {
	var command *exec.Cmd
	if runtime.GOOS == "darwin" {
		command = exec.Command("open", url)
	} else if runtime.GOOS == "windows" {
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	} else {
		command = exec.Command("xdg-open", url)
	}

	return command.Start()
}

func tokenResponseToSession(body *tokenResponse) *Session {
	expiresIn := body.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}

	return &Session{
		AccessToken:  body.AccessToken,
		RefreshToken: body.RefreshToken,
		TokenType:    body.TokenType,
		Scope:        body.Scope,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
	}
}
