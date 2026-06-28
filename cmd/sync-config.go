package cmd

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/intercube/cli/util"
)

type SyncSettings = util.Sync

func loadSyncSettings() (SyncSettings, error) {
	var settings SyncSettings
	if err := viper.UnmarshalKey("sync", &settings); err != nil {
		return SyncSettings{}, fmt.Errorf("unable to decode sync config: %w", err)
	}

	if settings.Files.Items == nil {
		settings.Files.Items = []util.SyncFileItem{}
	}

	return settings, nil
}
