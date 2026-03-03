package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/intercube/cli/util"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/manifoldco/promptui"
)

const defaultBoundaryURL = "https://controller.boundary.intercube.cloud"

var onboardingCmd = &cobra.Command{
	Use:   "onboarding",
	Short: "Guides first-time setup for Intercube CLI",
	Run: func(cmd *cobra.Command, args []string) {
		runOnboarding()
	},
}

func runOnboarding() {
	configPath, err := resolveOnboardingConfigPath()
	if err != nil {
		panic(err)
	}

	boundaryPath, boundaryErr := exec.LookPath("boundary")
	boundaryInstalled := boundaryErr == nil

	rsyncPath, rsyncErr := exec.LookPath("rsync")
	rsyncInstalled := rsyncErr == nil

	fmt.Println("Intercube CLI onboarding")
	fmt.Println()
	fmt.Printf("Config file: %s\n", configPath)

	if boundaryInstalled {
		fmt.Printf("Boundary CLI: installed (%s)\n", boundaryPath)
	} else {
		fmt.Println("Boundary CLI: not found (required for `intercube ssh`)")
	}

	if rsyncInstalled {
		fmt.Printf("rsync: installed (%s)\n", rsyncPath)
	} else {
		fmt.Println("rsync: not found (required for `intercube sync --files`)")
	}

	fmt.Println()
	configureNow, err := chooseYesNo("Configure login defaults now?")
	if err != nil {
		fmt.Printf("Onboarding cancelled: %v\n", err)
		return
	}

	if !configureNow {
		fmt.Println("Skipped configuration. You can run `intercube onboarding` any time.")
		return
	}

	username, err := promptText(
		"Boundary username",
		config.Login.Username,
		requiredValue,
		0,
	)
	if err != nil {
		fmt.Printf("Onboarding cancelled: %v\n", err)
		return
	}

	passwordLabel := "Boundary password"
	passwordValidator := requiredValue
	if strings.TrimSpace(config.Login.Password) != "" {
		passwordLabel = "Boundary password (leave blank to keep current)"
		passwordValidator = optionalValue
	}

	password, err := promptText(
		passwordLabel,
		"",
		passwordValidator,
		'*',
	)
	if err != nil {
		fmt.Printf("Onboarding cancelled: %v\n", err)
		return
	}
	if strings.TrimSpace(password) == "" {
		password = config.Login.Password
	}

	scope, err := promptText(
		"Boundary scope",
		config.Login.Scope,
		requiredValue,
		0,
	)
	if err != nil {
		fmt.Printf("Onboarding cancelled: %v\n", err)
		return
	}

	authMethod, err := promptText(
		"Boundary auth method ID",
		config.Login.AuthMethod,
		requiredValue,
		0,
	)
	if err != nil {
		fmt.Printf("Onboarding cancelled: %v\n", err)
		return
	}

	defaultInstanceURL := config.Login.InstanceUrl
	if strings.TrimSpace(defaultInstanceURL) == "" {
		defaultInstanceURL = defaultBoundaryURL
	}

	instanceURL, err := promptText(
		"Boundary controller URL",
		defaultInstanceURL,
		requiredValue,
		0,
	)
	if err != nil {
		fmt.Printf("Onboarding cancelled: %v\n", err)
		return
	}

	configureSyncDefaults, err := chooseYesNo("Configure sync defaults too?")
	if err != nil {
		fmt.Printf("Onboarding cancelled: %v\n", err)
		return
	}

	syncItems := config.Sync.Files.Items

	if configureSyncDefaults {
		syncItems, err = promptSyncFileItems(syncItems)
		if err != nil {
			fmt.Printf("Onboarding cancelled: %v\n", err)
			return
		}
	}

	viper.Set("login.username", username)
	viper.Set("login.password", password)
	viper.Set("login.scope", scope)
	viper.Set("login.auth_method", authMethod)
	viper.Set("login.instance_url", instanceURL)
	viper.Set("sync.files.items", toMapSlice(syncItems))

	if err := writeOnboardingConfig(configPath); err != nil {
		panic(err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Errorf("Unable to decode Config: %s \n", err))
	}

	fmt.Println()
	fmt.Printf("Saved configuration to %s\n", configPath)
	fmt.Println("Next steps:")
	fmt.Println("- Run `intercube ssh` to connect to a host")
	fmt.Println("- Run `intercube auth login` to sign in for API calls")
	fmt.Println("- Run `intercube auth status` to inspect local API session state")
	if !boundaryInstalled {
		fmt.Println("- Install Boundary CLI first: https://developer.hashicorp.com/boundary/downloads")
	}
	if !rsyncInstalled {
		fmt.Println("- Install rsync to enable `intercube sync --files`")
	}
}

func ensureLoginConfiguration() error {
	configPath, err := resolveOnboardingConfigPath()
	if err != nil {
		return err
	}

	configFileExists := true
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configFileExists = false
	} else if err != nil {
		return err
	}

	missingFields := make([]string, 0, 4)
	if strings.TrimSpace(config.Login.Username) == "" {
		missingFields = append(missingFields, "login.username")
	}
	if strings.TrimSpace(config.Login.Password) == "" {
		missingFields = append(missingFields, "login.password")
	}
	if strings.TrimSpace(config.Login.Scope) == "" {
		missingFields = append(missingFields, "login.scope")
	}
	if strings.TrimSpace(config.Login.AuthMethod) == "" {
		missingFields = append(missingFields, "login.auth_method")
	}

	if configFileExists && len(missingFields) == 0 {
		return nil
	}

	fmt.Println("Login configuration is missing. Let's set it up.")
	if !configFileExists {
		fmt.Printf("Config file not found, creating %s\n", configPath)
	} else {
		fmt.Printf("Missing required values: %s\n", strings.Join(missingFields, ", "))
	}
	fmt.Println()

	username, err := promptText(
		"Boundary username",
		config.Login.Username,
		requiredValue,
		0,
	)
	if err != nil {
		return err
	}

	passwordLabel := "Boundary password"
	passwordValidator := requiredValue
	if strings.TrimSpace(config.Login.Password) != "" {
		passwordLabel = "Boundary password (leave blank to keep current)"
		passwordValidator = optionalValue
	}

	password, err := promptText(
		passwordLabel,
		"",
		passwordValidator,
		'*',
	)
	if err != nil {
		return err
	}
	if strings.TrimSpace(password) == "" {
		password = config.Login.Password
	}

	scope, err := promptText(
		"Boundary scope",
		config.Login.Scope,
		requiredValue,
		0,
	)
	if err != nil {
		return err
	}

	authMethod, err := promptText(
		"Boundary auth method ID",
		config.Login.AuthMethod,
		requiredValue,
		0,
	)
	if err != nil {
		return err
	}

	defaultInstanceURL := config.Login.InstanceUrl
	if strings.TrimSpace(defaultInstanceURL) == "" {
		defaultInstanceURL = defaultBoundaryURL
	}

	instanceURL, err := promptText(
		"Boundary controller URL",
		defaultInstanceURL,
		requiredValue,
		0,
	)
	if err != nil {
		return err
	}

	viper.Set("login.username", username)
	viper.Set("login.password", password)
	viper.Set("login.scope", scope)
	viper.Set("login.auth_method", authMethod)
	viper.Set("login.instance_url", instanceURL)

	if err := saveConfigAndReload(configPath); err != nil {
		return err
	}

	fmt.Printf("Saved login configuration to %s\n", configPath)
	fmt.Println()

	return nil
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

	if used := viper.ConfigFileUsed(); used != "" {
		return used, nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".intercube.yaml"), nil
}

func writeOnboardingConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		return viper.WriteConfigAs(path)
	} else if os.IsNotExist(err) {
		return viper.SafeWriteConfigAs(path)
	} else {
		return err
	}
}

func chooseYesNo(label string) (bool, error) {
	items := []string{"Yes", "No"}
	prompt := promptui.Select{
		Label:  label,
		Items:  items,
		Size:   len(items),
		Stdout: &bellSkipper{},
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
