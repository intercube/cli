package pipeline

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestClientExistingSiteSetupRequests(t *testing.T) {
	type receivedRequest struct {
		Method string
		Path   string
		Body   string
	}
	received := make([]receivedRequest, 0, 4)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		received = append(received, receivedRequest{Method: request.Method, Path: request.URL.RequestURI(), Body: string(body)})
		if request.Header.Get("Authorization") != "Bearer local-token" {
			t.Errorf("authorization header = %q", request.Header.Get("Authorization"))
		}
		if request.Header.Get("X-Organization-Id") != "org-123" {
			t.Errorf("organization header = %q", request.Header.Get("X-Organization-Id"))
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/projects":
			if request.Method == http.MethodGet {
				_, _ = writer.Write([]byte(`[]`))
				return
			}
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"id":91,"name":"shop.example.com","projectName":"shop-example-com","siteId":456}`))
		case "/projects/workflow-templates":
			_, _ = writer.Write([]byte(`[{"id":-20,"key":"wordpress","name":"WordPress","version":"1.0.0","frameworkKey":"wordpress","latest":true}]`))
		case "/pipeline-sites/456/eligibility":
			writer.WriteHeader(http.StatusNoContent)
		case "/projects/templates/sync":
			_, _ = writer.Write([]byte(`{"evaluatedProjects":1,"synchronizedProjects":1,"skippedProjects":0}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "org-123", nil, nil)
	client.token = func(context.Context, bool) (string, error) { return "local-token", nil }

	ctx := context.Background()
	if _, err := client.ListProjects(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListWorkflowTemplates(ctx); err != nil {
		t.Fatal(err)
	}
	if err := client.EnableSite(ctx, 456); err != nil {
		t.Fatal(err)
	}
	created, err := client.CreateProject(ctx, CreateProjectRequest{
		Name:               "shop.example.com",
		SiteID:             456,
		DeployApprovalMode: "success",
		Git: GitSettings{
			URL:       "https://github.com/intercube/shop.git",
			Type:      "GitHub",
			Branch:    "develop",
			Directory: "wordpress",
		},
		DefaultBuildPlanID:   -20,
		EnvironmentVariables: []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != 91 {
		t.Fatalf("created project id = %d, want 91", created.ID)
	}
	if _, err := client.SyncProject(ctx, created.ID); err != nil {
		t.Fatal(err)
	}

	wantMethodsAndPaths := [][2]string{
		{http.MethodGet, "/projects"},
		{http.MethodGet, "/projects/workflow-templates"},
		{http.MethodPut, "/pipeline-sites/456/eligibility"},
		{http.MethodPost, "/projects"},
		{http.MethodPost, "/projects/templates/sync?projectId=91&onlyReadyForPush=true"},
	}
	gotMethodsAndPaths := make([][2]string, 0, len(received))
	for _, request := range received {
		gotMethodsAndPaths = append(gotMethodsAndPaths, [2]string{request.Method, request.Path})
	}
	if !reflect.DeepEqual(gotMethodsAndPaths, wantMethodsAndPaths) {
		t.Fatalf("requests = %#v, want %#v", gotMethodsAndPaths, wantMethodsAndPaths)
	}

	var createPayload map[string]any
	if err := json.Unmarshal([]byte(received[3].Body), &createPayload); err != nil {
		t.Fatal(err)
	}
	gitPayload := createPayload["git"].(map[string]any)
	if createPayload["siteId"] != float64(456) || createPayload["deployApprovalMode"] != "success" || gitPayload["branch"] != "develop" || gitPayload["directory"] != "wordpress" {
		t.Fatalf("unexpected create payload: %#v", createPayload)
	}
}
