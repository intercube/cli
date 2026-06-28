package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadSyncSettingsDecodesDatabaseTargetSSH(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(`
sync:
  database:
    target_ssh:
      host: production-sql.example.com
      user: deploy
      port: 2200
`))
	if err != nil {
		t.Fatalf("unable to read config: %v", err)
	}

	settings, err := loadSyncSettings()
	if err != nil {
		t.Fatalf("unable to load sync settings: %v", err)
	}

	if settings.Database.TargetSSH.Host != "production-sql.example.com" {
		t.Fatalf("unexpected database SSH host: %q", settings.Database.TargetSSH.Host)
	}
	if settings.Database.TargetSSH.User != "deploy" {
		t.Fatalf("unexpected database SSH user: %q", settings.Database.TargetSSH.User)
	}
	if settings.Database.TargetSSH.Port != 2200 {
		t.Fatalf("unexpected database SSH port: %d", settings.Database.TargetSSH.Port)
	}
	if settings.Files.Items == nil {
		t.Fatalf("expected file sync items to default to an empty slice")
	}
}
