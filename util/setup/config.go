package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v3"
)

type ProjectConfig struct {
	Version      int                    `yaml:"version"`
	Repository   ProjectRepository      `yaml:"repository"`
	Environments map[string]Environment `yaml:"environments"`
}

type ProjectRepository struct {
	URL       string `yaml:"url"`
	Directory string `yaml:"directory,omitempty"`
}

type Environment struct {
	Branch         string `yaml:"branch"`
	Domain         string `yaml:"domain"`
	ManagedDomain  string `yaml:"managed_domain"`
	IntentID       string `yaml:"intent_id"`
	IntentRevision string `yaml:"intent_revision"`
	QuoteID        string `yaml:"quote_id,omitempty"`
	IdempotencyKey string `yaml:"idempotency_key"`
	NoCharge       bool   `yaml:"no_charge,omitempty"`
	ServerID       int    `yaml:"server_id,omitempty"`
	SiteID         int    `yaml:"site_id,omitempty"`
	Status         string `yaml:"status"`
}

func LoadProjectConfig(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProjectConfig{Version: 1, Environments: map[string]Environment{}}, nil
		}
		return nil, err
	}
	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unable to read %s: %w", path, err)
	}
	if config.Environments == nil {
		config.Environments = map[string]Environment{}
	}
	if config.Version == 0 {
		config.Version = 1
	}
	return &config, nil
}

func SaveProjectConfig(path string, config *ProjectConfig) error {
	if config == nil {
		return fmt.Errorf("project config is required")
	}
	var document yaml.Node
	if existing, readErr := os.ReadFile(path); readErr == nil {
		if err := yaml.Unmarshal(existing, &document); err != nil {
			return fmt.Errorf("unable to preserve %s: %w", path, err)
		}
	} else if !os.IsNotExist(readErr) {
		return readErr
	}
	if len(document.Content) == 0 {
		document.Kind = yaml.DocumentNode
		document.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}

	encoded, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	var setupDocument yaml.Node
	if err := yaml.Unmarshal(encoded, &setupDocument); err != nil {
		return err
	}
	root := document.Content[0]
	setupRoot := setupDocument.Content[0]
	for _, key := range []string{"version", "repository", "environments"} {
		setMappingValue(root, key, mappingValue(setupRoot, key))
	}
	data, err := yaml.Marshal(&document)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".intercube-*.yaml")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0644); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1]
		}
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"}
}

func setMappingValue(mapping *yaml.Node, key string, value *yaml.Node) {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			mapping.Content[index+1] = value
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value)
}

func EnvironmentNames(config *ProjectConfig) []string {
	names := make([]string, 0, len(config.Environments))
	for name := range config.Environments {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
