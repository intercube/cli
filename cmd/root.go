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
	"github.com/intercube/cli/util"
	"github.com/intercube/cli/util/contextconfig"
	"github.com/spf13/cobra"
	"os"
	"strings"

	"github.com/spf13/viper"
)

var cfgFile string
var Verbose bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "intercube",
	Short: "Intercube CLI",
	Long: `Intercube CLI for host access, API operations, and environment sync.

Tip: use "intercube ssh" for host access.
The "intercube login" command is kept as a deprecated alias.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.intercube.yaml)")
	rootCmd.PersistentFlags().StringVar(&contextOverride, "context", "", "execution context override (pipeline,server,repository,global)")

	cobra.OnInitialize(initConfig)
}

var config util.Configuration

func initConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.BindEnv("context", "INTERCUBE_CONTEXT")
	viper.BindEnv("behavior.non_interactive", "INTERCUBE_NON_INTERACTIVE")
	viper.BindEnv("context.org_id", "INTERCUBE_ORG_ID", "INTERCUBE_ORGANIZATION_ID")
	viper.BindEnv("context.site_id", "INTERCUBE_SITE_ID")
	viper.BindEnv("context.server_id", "INTERCUBE_SERVER_ID")
	viper.AutomaticEnv()

	workingDir, _ := os.Getwd()
	explicitContext := strings.TrimSpace(contextOverride)
	if explicitContext == "" {
		explicitContext = strings.TrimSpace(viper.GetString("context"))
	}
	runtimeContext = contextconfig.DetectRuntime(explicitContext, workingDir)

	viper.SetDefault("behavior.non_interactive", false)
	viper.SetDefault("sync", map[string]interface{}{})
	loadResult, err := contextconfig.LoadLayeredConfig(viper.GetViper(), runtimeContext, cfgFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if Verbose {
		for _, path := range loadResult.LoadedPaths {
			fmt.Println("Using config file:", path)
		}
		fmt.Printf("Using context: %s\n", runtimeContext.Kind)
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		panic(fmt.Errorf("Unable to decode Config: %s \n", err))
	}

	if config.Behavior.NonInteractive {
		runtimeContext.NonInteractive = true
	}
}

func displaySubCommands(commands []*cobra.Command) {
	fmt.Println("This command has the following sub commands:")
	for _, command := range commands {
		fmt.Printf("\t%v (%v)\n", command.Use, command.Short)
	}
}
