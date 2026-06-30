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
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// devVersion is reported when the binary was not installed from a tagged
// module version (e.g. built locally with `go build` or `go run`).
const devVersion = "dev"

// currentVersion returns the module version embedded by the Go toolchain.
// For binaries installed via `go install module@version` this is the git tag
// (e.g. "v1.0.20"); for local builds it is empty or "(devel)", in which case
// we report devVersion.
func currentVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return devVersion
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	return devVersion
}

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the Intercube CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(currentVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
