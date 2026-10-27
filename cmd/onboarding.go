package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/intercube/cli/util"
	"github.com/intercube/cli/util/contextconfig"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/manifoldco/promptui"
)

var onboardingCmd = &cobra.Command{
	Use: "onboarding", Short: "Check SSH and sync tools and sign in with Clerk",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := exec.LookPath("ssh"); err != nil {
			return fmt.Errorf("OpenSSH is required: %w", err)
		}
		if _, err := exec.LookPath("rsync"); err != nil {
			fmt.Println("Install rsync before using file synchronization.")
		}
		if err := authLoginCmd.RunE(cmd, nil); err != nil {
			return err
		}
		path, err := resolveOnboardingConfigPath()
		if err != nil {
			return err
		}
		if _, err := os.Stat(path); err == nil {
			if err := writeOnboardingConfig(path); err != nil {
				return err
			}
		}
		return nil
	},
}

func ensureMappingsConfiguration() error {
	if len(config.Mappings) > 0 {
		return nil
	}

	configPath, err := resolveOnboardingConfigPath()
	if err != nil {
		return err
	}

	fmt.Println("No mappings configured. Let's add one for `intercube map`.")

	mappings := make([]map[string]string, 0, 1)
	for {
		from, err := promptText("Mapping source path", "", requiredValue, 0)
		if err != nil {
			return err
		}

		to, err := promptText("Mapping destination path", "", requiredValue, 0)
		if err != nil {
			return err
		}

		mappings = append(mappings, map[string]string{
			"from": from,
			"to":   to,
		})

		addAnother, err := chooseYesNo("Add another mapping?")
		if err != nil {
			return err
		}

		if !addAnother {
			break
		}
	}

	viper.Set("mappings", mappings)
	if err := saveConfigAndReload(configPath); err != nil {
		return err
	}

	fmt.Printf("Saved mappings to %s\n", configPath)
	fmt.Println()

	return nil
}

func promptSyncFileItems(current []util.SyncFileItem) ([]util.SyncFileItem, error) {
	items := make([]util.SyncFileItem, 0, len(current))
	items = append(items, current...)

	if len(items) > 0 {
		keepExisting, err := chooseYesNo("Keep existing sync file mappings?")
		if err != nil {
			return nil, err
		}

		if !keepExisting {
			items = []util.SyncFileItem{}
		}
	}

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

		items = append(items, util.SyncFileItem{Source: source, Target: target, Exclude: splitCSV(excludeRaw)})

		addAnother, err := chooseYesNo("Add another sync mapping?")
		if err != nil {
			return nil, err
		}

		if !addAnother {
			break
		}
	}

	return items, nil
}

func saveConfigAndReload(configPath string) error {
	if err := writeOnboardingConfig(configPath); err != nil {
		return err
	}

	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("unable to decode config: %w", err)
	}

	return nil
}

func resolveOnboardingConfigPath() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	return contextconfig.ResolveConfigPath(home), nil
}

func writeOnboardingConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	settings := viper.AllSettings()
	delete(settings, "login") // Boundary passwords have no role in Clerk-backed access.
	writer := viper.New()
	writer.SetConfigFile(path)
	if err := writer.MergeConfigMap(settings); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return writer.WriteConfigAs(path)
	} else if os.IsNotExist(err) {
		return writer.SafeWriteConfigAs(path)
	} else {
		return err
	}
}

func chooseYesNo(label string) (bool, error) {
	items := []string{"Yes", "No"}
	prompt := promptui.Select{
		Label:     label,
		Items:     items,
		Size:      selectSize(len(items)),
		Stdout:    &bellSkipper{},
		Templates: simpleSelectTemplates("option"),
	}

	index, _, err := prompt.Run()
	if err != nil {
		return false, err
	}

	return index == 0, nil
}

func promptText(label, defaultValue string, validator func(string) error, mask rune) (string, error) {
	prompt := promptui.Prompt{
		Label:     label,
		Default:   defaultValue,
		AllowEdit: true,
		Mask:      mask,
		Validate: func(input string) error {
			if validator == nil {
				return nil
			}

			return validator(strings.TrimSpace(input))
		},
	}

	if mask != 0 {
		prompt.HideEntered = true
	}

	value, err := prompt.Run()
	if err != nil {
		return "", err
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return strings.TrimSpace(defaultValue), nil
	}

	return value, nil
}

func requiredValue(input string) error {
	if strings.TrimSpace(input) == "" {
		return errors.New("value is required")
	}

	return nil
}

func optionalValue(_ string) error {
	return nil
}

func init() {
	rootCmd.AddCommand(onboardingCmd)
}
