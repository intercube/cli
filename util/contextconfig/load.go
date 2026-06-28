package contextconfig

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type LoadResult struct {
	LoadedPaths []string
}

func LoadLayeredConfig(v *viper.Viper, runtime Runtime, explicitConfigPath string) (LoadResult, error) {
	loaded := make([]string, 0, 2)

	if settings, path, err := readSettings(runtime.UserConfigPath); err != nil {
		return LoadResult{}, err
	} else if settings != nil {
		if err := v.MergeConfigMap(settings); err != nil {
			return LoadResult{}, fmt.Errorf("unable to merge config from %s: %w", path, err)
		}
		loaded = append(loaded, path)
	}

	activePath := strings.TrimSpace(explicitConfigPath)
	if activePath == "" {
		activePath = strings.TrimSpace(runtime.ActiveConfigPath)
	}

	if activePath != "" && activePath == strings.TrimSpace(runtime.UserConfigPath) {
		return LoadResult{LoadedPaths: loaded}, nil
	}

	if settings, path, err := readSettings(activePath); err != nil {
		return LoadResult{}, err
	} else if settings != nil {
		if err := v.MergeConfigMap(settings); err != nil {
			return LoadResult{}, fmt.Errorf("unable to merge config from %s: %w", path, err)
		}
		loaded = append(loaded, path)
	}

	return LoadResult{LoadedPaths: loaded}, nil
}

func readSettings(path string) (map[string]interface{}, string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, "", nil
	}

	if _, err := os.Stat(trimmed); err != nil {
		if os.IsNotExist(err) {
			return nil, "", nil
		}

		return nil, "", err
	}

	temp := viper.New()
	temp.SetConfigFile(trimmed)
	if err := temp.ReadInConfig(); err != nil {
		return nil, "", fmt.Errorf("unable to read config file %s: %w", trimmed, err)
	}

	return temp.AllSettings(), trimmed, nil
}
