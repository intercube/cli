/*
Copyright © 2023 Intercube <opensource@intercube.io>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var override = false
var interactiveMapSetup = false

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Maps files based on config yaml file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(config.Mappings) == 0 {
			if !interactiveMapSetup {
				return fmt.Errorf("no mappings configured. set `mappings` in config or run `intercube onboarding` interactively")
			}

			if !stdinIsTerminal() {
				return fmt.Errorf("interactive setup requires a terminal")
			}

			if err := ensureMappingsConfiguration(); err != nil {
				return fmt.Errorf("unable to continue map: %w", err)
			}
		}

		for _, mapping := range config.Mappings {
			if err := symlink(mapping.From, mapping.To); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(mapCmd)
	mapCmd.PersistentFlags().BoolVarP(&override, "override", "o", false, "Overrides existing destination file")
	mapCmd.PersistentFlags().BoolVar(&interactiveMapSetup, "interactive", false, "Prompt to create mappings when missing")
}

func symlink(from string, to string) error {
	resolvedFrom, err := expandHomePath(from)
	if err != nil {
		return err
	}

	resolvedTo, err := expandHomePath(to)
	if err != nil {
		return err
	}

	if strings.TrimSpace(resolvedFrom) == "" || strings.TrimSpace(resolvedTo) == "" {
		return fmt.Errorf("mapping source and destination are required")
	}

	if !fileExists(resolvedFrom) {
		return fmt.Errorf("origin file %v does not exist", resolvedFrom)
	}

	alreadyMapped, mappedErr := destinationMatchesSource(resolvedFrom, resolvedTo)
	if mappedErr == nil && alreadyMapped {
		fmt.Printf("Already mapped %v to %v\n", resolvedFrom, resolvedTo)
		return nil
	}

	if !fileExists(resolvedTo) || override {
		if override {
			if destination, err := os.Lstat(resolvedTo); err == nil {
				if destination.IsDir() {
					if err := os.RemoveAll(resolvedTo); err != nil {
						return err
					}
					fmt.Printf("Removed directory %v\n", resolvedTo)
				} else {
					if err := os.Remove(resolvedTo); err != nil {
						return err
					}
					fmt.Printf("Removed file %v\n", resolvedTo)
				}
			}
		}

		err := os.Symlink(resolvedFrom, resolvedTo)
		if err != nil {
			return fmt.Errorf("unable to map %v -> %v: %w", resolvedFrom, resolvedTo, err)
		}

		fmt.Printf("Mapped %v to %v\n", resolvedFrom, resolvedTo)
		return nil
	}

	return fmt.Errorf("destination file %v already exists", resolvedTo)
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func stdinIsTerminal() bool {
	if os.Stdin == nil {
		return false
	}

	return term.IsTerminal(int(os.Stdin.Fd()))
}

func expandHomePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", nil
	}

	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		if trimmed == "~" {
			return home, nil
		}

		return filepath.Join(home, strings.TrimPrefix(trimmed, "~/")), nil
	}

	return trimmed, nil
}

func destinationMatchesSource(source string, destination string) (bool, error) {
	info, err := os.Lstat(destination)
	if err != nil {
		return false, err
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return false, nil
	}

	linkTarget, err := os.Readlink(destination)
	if err != nil {
		return false, err
	}

	if !filepath.IsAbs(linkTarget) {
		linkTarget = filepath.Join(filepath.Dir(destination), linkTarget)
	}

	absSource, err := filepath.Abs(source)
	if err != nil {
		return false, err
	}

	absTarget, err := filepath.Abs(linkTarget)
	if err != nil {
		return false, err
	}

	return filepath.Clean(absSource) == filepath.Clean(absTarget), nil
}
