package cmd

import (
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewritingConfigurationRemovesBoundaryPasswordAndPreservesContext(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	path := filepath.Join(t.TempDir(), ".intercube.yaml")
	viper.Set("login.username", "legacy")
	viper.Set("login.password", "fixture-old-boundary-password")
	viper.Set("context.org_id", "org_keep")
	if err := writeOnboardingConfig(path); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contents), "fixture-old-boundary-password") || strings.Contains(string(contents), "login:") {
		t.Fatal("legacy Boundary credentials were retained")
	}
	restored := viper.New()
	restored.SetConfigFile(path)
	if err := restored.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	if restored.GetString("context.org_id") != "org_keep" {
		t.Fatal("organization context was lost")
	}
}
