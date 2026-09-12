package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
	token      func(context.Context, bool) (string, error)
}

type Project struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	SiteID *int   `json:"siteId"`
}

type WorkflowTemplate struct {
	ID           int    `json:"id"`
	Key          string `json:"key"`
	Name         string `json:"name"`
	Version      string `json:"version"`
	FrameworkKey string `json:"frameworkKey"`
	Deprecated   bool   `json:"deprecated"`
	Latest       bool   `json:"latest"`
}

type GitSettings struct {
	URL       string `json:"url"`
	Type      string `json:"type"`
	Branch    string `json:"branch"`
	Directory string `json:"directory,omitempty"`
}

type CreateProjectRequest struct {
	Name                 string      `json:"name"`
	SiteID               int         `json:"siteId"`
	DeployApprovalMode   string      `json:"deployApprovalMode"`
	Git                  GitSettings `json:"git"`
	DefaultBuildPlanID   int         `json:"defaultBuildPlanId"`
	EnvironmentVariables []any       `json:"environmentVariables"`
}

type CreatedProject struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ProjectName string `json:"projectName"`
	SiteID      *int   `json:"siteId"`
}

type SyncResult struct {
	EvaluatedProjects    int `json:"evaluatedProjects"`
	SynchronizedProjects int `json:"synchronizedProjects"`
	SkippedProjects      int `json:"skippedProjects"`
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

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var projects []Project
	if err := c.doJSON(ctx, http.MethodGet, "/projects", nil, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (c *Client) ListWorkflowTemplates(ctx context.Context) ([]WorkflowTemplate, error) {
	var templates []WorkflowTemplate
	if err := c.doJSON(ctx, http.MethodGet, "/projects/workflow-templates", nil, &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

func (c *Client) EnableSite(ctx context.Context, siteID int) error {
	return c.doJSON(ctx, http.MethodPut, fmt.Sprintf("/pipeline-sites/%d/eligibility", siteID), map[string]bool{
		"pipelineEnabled": true,
	}, nil)
}

func (c *Client) CreateProject(ctx context.Context, request CreateProjectRequest) (*CreatedProject, error) {
	var project CreatedProject
	if err := c.doJSON(ctx, http.MethodPost, "/projects", request, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

func (c *Client) SyncProject(ctx context.Context, projectID int) (*SyncResult, error) {
	path := "/projects/templates/sync?projectId=" + url.QueryEscape(strconv.Itoa(projectID)) + "&onlyReadyForPush=true"
	var result SyncResult
	if err := c.doJSON(ctx, http.MethodPost, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload, out any) error {
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
	if response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("pipeline API %s %s failed with status %d: %s", method, path, response.StatusCode, apiErrorMessage(responseBody))
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
	if c.token != nil {
		return c.token(ctx, forceRefresh)
	}
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
	var problem struct {
		Detail string `json:"detail"`
		Title  string `json:"title"`
		Error  struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(payload, &problem) == nil {
		for _, message := range []string{problem.Error.Message, problem.Detail, problem.Title} {
			if strings.TrimSpace(message) != "" {
				return message
			}
		}
	}
	return strings.TrimSpace(string(payload))
}
