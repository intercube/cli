package setup

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
	BaseURL     string
	OrgID       string
	Store       *authutil.SessionStore
	Clerk       *authutil.ClerkClient
	HTTPClient  *http.Client
	GitHubToken string
	mu          sync.Mutex
}

type RepositoryAnalysisRequest struct {
	Owner      string `json:"owner"`
	Repository string `json:"repository"`
	Branch     string `json:"branch,omitempty"`
	Directory  string `json:"directory,omitempty"`
}

type RepositoryAnalysis struct {
	GitHubLinked                bool     `json:"githublinked"`
	GitHubAuthorizationRequired bool     `json:"githubauthorizationrequired"`
	GitHubInstallURL            string   `json:"githubinstallurl"`
	GitHubClientID              string   `json:"githubclientid"`
	Repository                  string   `json:"repository"`
	DefaultBranch               string   `json:"defaultbranch"`
	Branch                      string   `json:"branch"`
	SuggestedDomain             string   `json:"suggesteddomain"`
	Framework                   string   `json:"framework"`
	Runtime                     string   `json:"runtime"`
	RuntimeVersion              string   `json:"runtimeversion"`
	Installations               []string `json:"installations"`
	WorkflowKey                 string   `json:"workflowkey"`
	Evidence                    []string `json:"evidence"`
	Confidence                  string   `json:"confidence"`
}

type RepositoryRequest struct {
	URL                  string `json:"url"`
	Branch               string `json:"branch"`
	Directory            string `json:"directory,omitempty"`
	WorkflowKey          string `json:"workflowKey"`
	ContinuousDeployment bool   `json:"continuousDeployment"`
}

type RuntimeRequest struct {
	Type    string `json:"type"`
	Version string `json:"version,omitempty"`
}

type InstallationRequest struct {
	Type    string `json:"type"`
	Version string `json:"version,omitempty"`
}

type ProvisioningPayload struct {
	ResourceKind     string                `json:"resourceKind"`
	ServerID         int                   `json:"serverId,omitempty"`
	Environment      string                `json:"environment"`
	Domains          []string              `json:"domains"`
	Framework        string                `json:"framework"`
	Runtime          string                `json:"runtime,omitempty"`
	DomainName       string                `json:"domainName"`
	MainDomainName   string                `json:"mainDomainName"`
	ServerPlanKey    string                `json:"serverPlanKey,omitempty"`
	Architecture     string                `json:"architecture,omitempty"`
	RuntimeSelection *RuntimeRequest       `json:"runtimeSelection,omitempty"`
	Installations    []InstallationRequest `json:"installations"`
	Repository       RepositoryRequest     `json:"repository"`
}

type QuoteRequest struct {
	OrganizationID string              `json:"organizationId,omitempty"`
	IdempotencyKey string              `json:"idempotencyKey"`
	BillingPeriod  string              `json:"billingPeriod"`
	NoCharge       bool                `json:"noCharge"`
	Payload        ProvisioningPayload `json:"payload"`
}

type Intent struct {
	ID             string         `json:"id"`
	Revision       string         `json:"revision"`
	Classification Classification `json:"classification"`
}

type Classification struct {
	ServerProfile *ServerProfile `json:"serverProfile"`
}

type ServerProfile struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	Cores        int    `json:"cores"`
	MemoryGB     int    `json:"memoryGb"`
	Architecture string `json:"architecture"`
}

type Quote struct {
	QuoteID       string  `json:"quoteId"`
	ProductName   string  `json:"productName"`
	Amount        float64 `json:"amount"`
	ListAmount    float64 `json:"listAmount"`
	Currency      string  `json:"currency"`
	BillingPeriod string  `json:"billingPeriod"`
}

type QuoteResponse struct {
	Intent Intent `json:"intent"`
	Quote  Quote  `json:"quote"`
}

type CreateRequest struct {
	OrganizationID string `json:"organizationId,omitempty"`
	IdempotencyKey string `json:"idempotencyKey"`
	IntentID       string `json:"intentId"`
	IntentRevision string `json:"intentRevision"`
	QuoteID        string `json:"quoteId,omitempty"`
	BillingPeriod  string `json:"billingPeriod"`
	NoCharge       bool   `json:"noCharge"`
}

type Progress struct {
	ID             string `json:"id"`
	Revision       string `json:"revision"`
	SetupStatus    string `json:"setupstatus"`
	SetupStage     string `json:"setupstage"`
	SetupMessage   string `json:"setupmessage"`
	ServerID       int    `json:"serverid"`
	SiteID         int    `json:"siteid"`
	FailureMessage string `json:"failuremessage"`
}

type envelope[T any] struct {
	Data T `json:"data"`
}

func NewClient(baseURL, organizationID string, store *authutil.SessionStore, clerk *authutil.ClerkClient) *Client {
	return &Client{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		OrgID:   strings.TrimSpace(organizationID),
		Store:   store,
		Clerk:   clerk,
		HTTPClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (c *Client) AnalyzeRepository(ctx context.Context, request RepositoryAnalysisRequest) (*RepositoryAnalysis, error) {
	var response envelope[RepositoryAnalysis]
	if err := c.doJSON(ctx, http.MethodPost, "/api/v2/setups/repository-analysis", request, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (c *Client) Quote(ctx context.Context, request QuoteRequest) (*QuoteResponse, error) {
	var response envelope[QuoteResponse]
	if err := c.doJSON(ctx, http.MethodPost, "/api/v2/setups/quote", request, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (c *Client) Create(ctx context.Context, request CreateRequest) (*Progress, error) {
	var response envelope[Progress]
	if err := c.doJSON(ctx, http.MethodPost, "/api/v2/setups", request, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (c *Client) Progress(ctx context.Context, intentID string) (*Progress, error) {
	var response envelope[Progress]
	if err := c.doJSON(ctx, http.MethodGet, "/api/v2/setups/"+intentID, nil, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (c *Client) Retry(ctx context.Context, intentID string) (*Progress, error) {
	var response envelope[Progress]
	if err := c.doJSON(ctx, http.MethodPost, "/api/v2/setups/"+intentID, nil, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, out any) error {
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
		return fmt.Errorf("setup API %s %s failed with status %d: %s", method, path, response.StatusCode, apiErrorMessage(responseBody))
	}
	if out == nil || len(responseBody) == 0 {
		return nil
	}
	return json.Unmarshal(responseBody, out)
}

func (c *Client) doRequest(ctx context.Context, method, path string, body []byte, forceRefresh bool) (*http.Response, []byte, error) {
	token, err := c.accessToken(ctx, forceRefresh)
	if err != nil {
		return nil, nil, err
	}
	request, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "intercube-cli")
	if c.OrgID != "" {
		request.Header.Set("X-Organization-Id", c.OrgID)
	}
	if strings.Contains(path, "/repository-analysis") && c.GitHubToken != "" {
		request.Header.Set("X-Intercube-GitHub-Token", c.GitHubToken)
	}
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	return response, responseBody, err
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
	if forceRefresh || session.ExpiresSoon(time.Minute) {
		session, err = c.Clerk.RefreshSession(ctx, session)
		if err != nil {
			return "", fmt.Errorf("unable to refresh auth session: %w", err)
		}
		if err := c.Store.Save(ctx, session); err != nil {
			return "", err
		}
	}
	if strings.TrimSpace(session.AccessToken) == "" {
		return "", errors.New("missing access token, run `intercube auth login`")
	}
	return session.AccessToken, nil
}

func marshalPayload(payload any) ([]byte, error) {
	if payload == nil {
		return nil, nil
	}
	return json.Marshal(payload)
}

func apiErrorMessage(payload []byte) string {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(payload, &body) == nil && body.Error.Message != "" {
		return body.Error.Message
	}
	return strings.TrimSpace(string(payload))
}
