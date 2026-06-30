/*
Copyright © 2026 Intercube <opensource@intercube.io>

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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const (
	// modulePath is the Go module that hosts the CLI.
	modulePath = "github.com/intercube/cli"
	// installPath is the package built into the `intercube` binary.
	installPath = modulePath + "/cmd/intercube"
	// goProxy is the public Go module proxy used to discover the latest tag.
	// The repository is public, so no authentication is required.
	goProxy = "https://proxy.golang.org"
)

var forceUpdate bool

// selfUpdateCmd updates the running intercube binary to the latest (or a
// specified) tagged version by re-running `go install`.
var selfUpdateCmd = &cobra.Command{
	Use:   "self-update [version]",
	Short: "Update the Intercube CLI to the latest version",
	Long: `Update the Intercube CLI by reinstalling it with the Go toolchain.

By default this installs the latest published version. Pass an explicit
version (e.g. "intercube self-update v1.0.19") to pin or roll back.

Requires the Go toolchain to be installed and on your PATH.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSelfUpdate,
}

func init() {
	selfUpdateCmd.Flags().BoolVarP(&forceUpdate, "force", "f", false, "reinstall even if already up to date")
	rootCmd.AddCommand(selfUpdateCmd)
}

func runSelfUpdate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	current := currentVersion()

	// Resolve the target version: an explicit argument, or "latest".
	target := "latest"
	if len(args) == 1 {
		target = strings.TrimSpace(args[0])
	}

	// When targeting latest, look up the newest tag so we can report it and
	// skip needless reinstalls. A lookup failure is non-fatal: we warn and
	// proceed to let `go install` do the work.
	if target == "latest" {
		latest, err := latestVersion(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not check for the latest version: %v\n", err)
		} else {
			target = latest
			if !forceUpdate && current == latest {
				fmt.Printf("Intercube CLI is already up to date (%s)\n", current)
				return nil
			}
			if !forceUpdate && semver.IsValid(current) && semver.Compare(current, latest) > 0 {
				fmt.Printf("Intercube CLI %s is newer than the latest published version (%s); nothing to do\n", current, latest)
				return nil
			}
		}
	}

	goBin, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("the Go toolchain is required to self-update but `go` was not found on your PATH.\n"+
			"Install Go (https://go.dev/dl/) or update manually with:\n  go install %s@latest", installPath)
	}

	spec := installPath + "@" + target
	fmt.Printf("Updating Intercube CLI (%s -> %s)...\n", current, target)

	install := exec.CommandContext(ctx, goBin, "install", spec)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	install.Env = os.Environ()
	if err := install.Run(); err != nil {
		return fmt.Errorf("`go install %s` failed: %w", spec, err)
	}

	fmt.Printf("Updated Intercube CLI to %s.\n", target)
	warnIfShadowed(ctx, goBin)
	return nil
}

// latestVersion queries the public Go module proxy for the most recent tagged
// version of the module.
func latestVersion(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/%s/@latest", goProxy, modulePath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s from %s", resp.Status, url)
	}

	var payload struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Version == "" {
		return "", fmt.Errorf("no version returned by %s", url)
	}
	return payload.Version, nil
}

// warnIfShadowed alerts the user when `go install` wrote the new binary to a
// directory other than the one the currently-running binary lives in, which
// can happen when an older copy appears earlier on PATH.
func warnIfShadowed(ctx context.Context, goBin string) {
	installDir := goInstallDir(ctx, goBin)
	if installDir == "" {
		return
	}

	self, err := os.Executable()
	if err != nil {
		return
	}
	if selfPath, err := filepath.EvalSymlinks(self); err == nil {
		self = selfPath
	}

	if filepath.Dir(self) != installDir {
		fmt.Fprintf(os.Stderr,
			"\nNote: the updated binary was installed to %s,\n"+
				"but the Intercube CLI you just ran is at %s.\n"+
				"Make sure %s is early on your PATH, or run the updated binary directly.\n",
			installDir, self, installDir)
	}
}

// goInstallDir returns the directory `go install` writes binaries to,
// i.e. $GOBIN if set, otherwise $GOPATH/bin.
func goInstallDir(ctx context.Context, goBin string) string {
	if gobin := strings.TrimSpace(goEnv(ctx, goBin, "GOBIN")); gobin != "" {
		return gobin
	}
	if gopath := strings.TrimSpace(goEnv(ctx, goBin, "GOPATH")); gopath != "" {
		// GOPATH may contain a list; the first entry receives `go install` output.
		first := filepath.SplitList(gopath)[0]
		if first != "" {
			return filepath.Join(first, "bin")
		}
	}
	return ""
}

// goEnv reads a single value from `go env`.
func goEnv(ctx context.Context, goBin, key string) string {
	out, err := exec.CommandContext(ctx, goBin, "env", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
