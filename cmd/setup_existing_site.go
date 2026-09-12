package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/intercube/cli/util/appconfig"
	authutil "github.com/intercube/cli/util/auth"
	"github.com/intercube/cli/util/inventory"
	"github.com/intercube/cli/util/pipeline"
	setupapi "github.com/intercube/cli/util/setup"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

type existingSiteOption struct {
	Site  inventory.SiteServer
	Title string
	Meta  string
}

func runExistingSiteSetup(
	cmd *cobra.Command,
	setupClient *setupapi.Client,
	organizationID, repositoryRoot, owner, repository, configPath string,
	projectConfig *setupapi.ProjectConfig,
) error {
	if runtimeContext.NonInteractive {
		return errors.New("--existing-site requires an interactive terminal to select a site")
	}

	inventoryClient, _, err := newInventoryClient(cmd, organizationID)
	if err != nil {
		return err
	}
	pipelineClient, err := newPipelineClient(organizationID)
	if err != nil {
		return err
	}

	sites, err := inventoryClient.ListSites(cmd.Context())
	if err != nil {
		return err
	}
	if len(sites) == 0 {
		return errors.New("no sites are available in the selected organization")
	}
	selected, err := selectExistingSite(sites)
	if err != nil {
		return err
	}

	projects, err := pipelineClient.ListProjects(cmd.Context())
	if err != nil {
		return err
	}
	if project := projectForSite(projects, selected.ID); project != nil {
		return fmt.Errorf("site %s already has pipeline project %q; changing its repository, branch, or directory is not supported by `intercube setup`", selected.ID, project.Name)
	}

	site, err := inventoryClient.GetSite(cmd.Context(), selected.ServerID, selected.ID)
	if err != nil {
		return err
	}
	branch, analysis, err := analyzeSetupRepository(cmd, setupClient, repositoryRoot, owner, repository)
	if err != nil {
		return err
	}
	templates, err := pipelineClient.ListWorkflowTemplates(cmd.Context())
	if err != nil {
		return err
	}
	template, err := workflowTemplateForAnalysis(templates, analysis.WorkflowKey, analysis.Framework)
	if err != nil {
		return err
	}

	environment := strings.TrimSpace(setupEnvironment)
	if environment == "" {
		environment = strings.TrimSpace(site.Environment)
	}
	if environment == "" {
		if selected.IsProduction {
			environment = "production"
		} else {
			environment = "development"
		}
	}
	if err := validateEnvironment(environment); err != nil {
		return err
	}
	if configured, exists := projectConfig.Environments[environment]; exists {
		if configured.SiteID > 0 && strconv.Itoa(configured.SiteID) == selected.ID {
			return fmt.Errorf("site %s is already recorded as environment %q in %s", selected.ID, environment, configPath)
		}
		return fmt.Errorf("environment %q is already configured in %s; choose another name with --environment or -e", environment, configPath)
	}

	domain := strings.TrimSpace(site.MainDomain)
	if domain == "" {
		domain = strings.TrimSpace(selected.MainDomain)
	}
	projectName := domain
	if projectName == "" {
		projectName = strings.TrimSpace(site.Username)
	}
	if projectName == "" {
		projectName = owner + "/" + repository
	}
	siteID, err := strconv.Atoi(selected.ID)
	if err != nil {
		return fmt.Errorf("site %q has an invalid identifier", selected.ID)
	}
	serverID, err := strconv.Atoi(selected.ServerID)
	if err != nil {
		return fmt.Errorf("server %q has an invalid identifier", selected.ServerID)
	}

	printExistingSiteSummary(selected, environment, branch, domain, analysis, template)
	if !setupYes {
		confirmed, confirmErr := promptConfirm("Connect this repository and enable deployments?", false)
		if confirmErr != nil {
			return confirmErr
		}
		if !confirmed {
			fmt.Println("No changes were made.")
			return nil
		}
	}

	fmt.Println("Enabling CI/CD for the selected site...")
	if err := pipelineClient.EnableSite(cmd.Context(), siteID); err != nil {
		return err
	}
	created, err := pipelineClient.CreateProject(cmd.Context(), pipeline.CreateProjectRequest{
		Name:               projectName,
		SiteID:             siteID,
		DeployApprovalMode: "success",
		Git: pipeline.GitSettings{
			URL:       fmt.Sprintf("https://github.com/%s/%s.git", owner, repository),
			Type:      "GitHub",
			Branch:    branch,
			Directory: strings.TrimSpace(setupDirectory),
		},
		DefaultBuildPlanID:   template.ID,
		EnvironmentVariables: []any{},
	})
	if err != nil {
		return err
	}

	projectConfig.Version = 1
	projectConfig.Repository = setupapi.ProjectRepository{
		URL:       fmt.Sprintf("https://github.com/%s/%s.git", owner, repository),
		Directory: strings.TrimSpace(setupDirectory),
	}
	projectConfig.Environments[environment] = setupapi.Environment{
		Branch:            branch,
		Domain:            domain,
		ServerID:          serverID,
		SiteID:            siteID,
		PipelineProjectID: created.ID,
		Status:            "pipeline_pending",
	}
	configErr := setupapi.SaveProjectConfig(configPath, projectConfig)

	fmt.Println("Synchronizing the GoCD project...")
	syncResult, err := pipelineClient.SyncProject(cmd.Context(), created.ID)
	if err != nil {
		return existingSiteRecoveryError(created.ID, configPath, configErr, fmt.Errorf("immediate GoCD synchronization failed: %w", err))
	}
	if syncResult.SynchronizedProjects != 1 {
		return existingSiteRecoveryError(created.ID, configPath, configErr, errors.New("GoCD did not synchronize it immediately"))
	}

	configured := projectConfig.Environments[environment]
	configured.Status = "ready"
	projectConfig.Environments[environment] = configured
	if err := setupapi.SaveProjectConfig(configPath, projectConfig); err != nil {
		return fmt.Errorf("pipeline project %d was created and synchronized, but %s could not be updated; inspect the project in Dashboard: %w", created.ID, configPath, err)
	}
	fmt.Printf("Setup complete. Site %s now deploys %s from %s", selected.ID, analysis.Repository, branch)
	if setupDirectory != "" {
		fmt.Printf(" (%s)", setupDirectory)
	}
	fmt.Println(". The first deployment will start automatically in GoCD.")
	return nil
}

func newPipelineClient(organizationID string) (*pipeline.Client, error) {
	appconfig.LoadFromEnv()
	if err := appconfig.ValidateClerk(); err != nil {
		return nil, fmt.Errorf("%w (set via env/.env or build-time)", err)
	}
	if err := appconfig.ValidatePipeline(); err != nil {
		return nil, fmt.Errorf("%w (set via env/.env or build-time)", err)
	}
	store, err := authutil.NewSessionStore("intercube-cli")
	if err != nil {
		return nil, err
	}
	clerk := &authutil.ClerkClient{
		Issuer:       appconfig.ClerkIssuer,
		ClientID:     appconfig.ClerkClientID,
		Audience:     appconfig.ClerkAudience,
		Scopes:       appconfig.ClerkScopes,
		CallbackPort: appconfig.ParsedCallbackPort(),
	}
	return pipeline.NewClient(appconfig.PipelineAPIBaseURL, organizationID, store, clerk), nil
}

func selectExistingSite(sites []inventory.SiteServer) (*inventory.SiteServer, error) {
	options := make([]existingSiteOption, 0, len(sites))
	for _, site := range sites {
		title := strings.TrimSpace(site.MainDomain)
		if title == "" {
			title = site.Username
		}
		environment := "development"
		if site.IsProduction {
			environment = "production"
		}
		options = append(options, existingSiteOption{
			Site:  site,
			Title: title,
			Meta:  fmt.Sprintf("%s · %s · site #%s", environment, site.ServerName, site.ID),
		})
	}
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "> {{ .Title | cyan }} {{ .Meta | faint }}",
		Inactive: "  {{ .Title }} {{ .Meta | faint }}",
		Selected: "Selected site: {{ .Title | cyan }}",
		Details: `
{{ "Environment:" | faint }}	{{ if .Site.IsProduction }}production{{ else }}development{{ end }}
{{ "Server:" | faint }}	{{ .Site.ServerName }} (#{{ .Site.ServerID }})
{{ "Site ID:" | faint }}	{{ .Site.ID }}
{{ "Username:" | faint }}	{{ .Site.Username }}`,
	}
	prompt := promptui.Select{
		Label:             "Search site to connect",
		Items:             options,
		Templates:         templates,
		Size:              selectSize(len(options)),
		StartInSearchMode: true,
		Searcher: func(input string, index int) bool {
			return existingSiteOptionMatches(options[index], input)
		},
		HideHelp: true,
	}
	index, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	return &options[index].Site, nil
}

func existingSiteRecoveryError(projectID int, configPath string, configErr, syncErr error) error {
	message := fmt.Sprintf(
		"pipeline project %d was created, but %v; Nexus will retry synchronization automatically, and the project can be inspected in Dashboard",
		projectID,
		syncErr,
	)
	if configErr != nil {
		message += fmt.Sprintf("; %s also could not be updated: %v", configPath, configErr)
	}
	return errors.New(message)
}

func existingSiteOptionMatches(option existingSiteOption, input string) bool {
	needle := strings.ToLower(strings.TrimSpace(input))
	if needle == "" {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{
		option.Site.ID,
		option.Site.Username,
		option.Site.MainDomain,
		option.Site.ServerID,
		option.Site.ServerName,
		option.Meta,
	}, " "))
	return strings.Contains(haystack, needle)
}

func projectForSite(projects []pipeline.Project, siteID string) *pipeline.Project {
	for index := range projects {
		if projects[index].SiteID != nil && strconv.Itoa(*projects[index].SiteID) == siteID {
			return &projects[index]
		}
	}
	return nil
}

func workflowTemplateForAnalysis(templates []pipeline.WorkflowTemplate, workflowKey, framework string) (*pipeline.WorkflowTemplate, error) {
	for _, latestOnly := range []bool{true, false} {
		for index := range templates {
			template := &templates[index]
			if template.Deprecated || (latestOnly && !template.Latest) {
				continue
			}
			if strings.EqualFold(template.Key, workflowKey) {
				return template, nil
			}
		}
	}
	for _, latestOnly := range []bool{true, false} {
		for index := range templates {
			template := &templates[index]
			if template.Deprecated || (latestOnly && !template.Latest) {
				continue
			}
			if strings.EqualFold(template.FrameworkKey, framework) {
				return template, nil
			}
		}
	}
	return nil, fmt.Errorf("Nexus has no active workflow template for detected application %q", framework)
}

func printExistingSiteSummary(
	site *inventory.SiteServer,
	environment, branch, domain string,
	analysis *setupapi.RepositoryAnalysis,
	template *pipeline.WorkflowTemplate,
) {
	fmt.Println("\nExisting site setup")
	fmt.Printf("  Site:        %s (#%s)\n", domain, site.ID)
	fmt.Printf("  Server:      %s (#%s)\n", site.ServerName, site.ServerID)
	fmt.Printf("  Environment: %s\n", environment)
	fmt.Printf("  Repository:  %s\n", analysis.Repository)
	fmt.Printf("  Branch:      %s\n", branch)
	if setupDirectory != "" {
		fmt.Printf("  Directory:   %s\n", setupDirectory)
	}
	fmt.Printf("  Workflow:    %s %s\n", template.Name, template.Version)
	fmt.Println("  Deployment:  automatic after successful commits")
	fmt.Println()
}
