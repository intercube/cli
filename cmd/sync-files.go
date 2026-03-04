package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/intercube/cli/util"
)

func runFileSync(cmd *cobra.Command, target ResolvedSyncTarget, settings *SyncSettings, dryRun bool) error {
	items, err := ensureFileSyncItems(settings)
	if err != nil {
		return err
	}

	for _, item := range items {
		source := strings.TrimSpace(item.Source)
		if source == "" {
			return fmt.Errorf("sync file source path is required")
		}

		targetPath := strings.TrimSpace(item.Target)
		if targetPath == "" {
			return fmt.Errorf("sync file target path is required")
		}

		destination := fmt.Sprintf("%s@%s:%s", target.Username, target.Host, targetPath)

		args := []string{"-az", "-e", fmt.Sprintf("ssh -p %d", target.Port)}
		if dryRun {
			args = append(args, "--dry-run")
		}

		for _, pattern := range item.Exclude {
			trimmed := strings.TrimSpace(pattern)
			if trimmed != "" {
				args = append(args, "--exclude", trimmed)
			}
		}

		args = append(args, source, destination)

		fmt.Printf("File sync: %s -> %s\n", source, destination)
		fmt.Printf("Running: rsync %s\n", strings.Join(args, " "))

		if dryRun {
			continue
		}

		command := exec.CommandContext(cmd.Context(), "rsync", args...)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return err
		}
	}

	return nil
}

func ensureFileSyncItems(settings *SyncSettings) ([]util.SyncFileItem, error) {
	if len(settings.Files.Items) > 0 {
		return settings.Files.Items, nil
	}

	if isNonInteractiveMode() {
		return nil, fmt.Errorf("sync.files.items is required in non-interactive mode")
	}

	if err := ensureInteractiveMode("sync file mapping setup"); err != nil {
		return nil, err
	}

	fmt.Println("No file sync paths configured yet. Let's add them now.")

	items := make([]util.SyncFileItem, 0, 1)
	for {
		source, err := promptText("Sync source path", "", requiredValue, 0)
		if err != nil {
			return nil, err
		}

		target, err := promptText("Sync target path", source, requiredValue, 0)
		if err != nil {
			return nil, err
		}

		excludeRaw, err := promptText("Exclude patterns (comma-separated, optional)", "", optionalValue, 0)
		if err != nil {
			return nil, err
		}

		excludes := splitCSV(excludeRaw)
		items = append(items, util.SyncFileItem{Source: source, Target: target, Exclude: excludes})

		addAnother, err := chooseYesNo("Add another file sync mapping?")
		if err != nil {
			return nil, err
		}

		if !addAnother {
			break
		}
	}

	settings.Files.Items = items
	viper.Set("sync.files.items", toMapSlice(items))

	configPath, err := resolveOnboardingConfigPath()
	if err != nil {
		return nil, err
	}

	if err := saveConfigAndReload(configPath); err != nil {
		return nil, err
	}

	fmt.Printf("Saved sync file mappings to %s\n", configPath)
	return items, nil
}

func toMapSlice(items []util.SyncFileItem) []map[string]interface{} {
	encoded := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		entry := map[string]interface{}{
			"source": item.Source,
			"target": item.Target,
		}
		if len(item.Exclude) > 0 {
			entry["exclude"] = item.Exclude
		}
		encoded = append(encoded, entry)
	}

	return encoded
}

func splitCSV(input string) []string {
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
