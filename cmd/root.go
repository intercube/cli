/*
Copyright © 2021 Intercube <opensource@intercube.io>

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
	"github.com/intercube/intercube/util"
	"github.com/spf13/cobra"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string
var Verbose bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "intercube",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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

	cobra.OnInitialize(initConfig)
}

var config util.Configuration

func initConfig() {
	viper.SetConfigName("config")
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".intercube")
	}

	viper.AutomaticEnv()

	viper.SetDefault("file_syncing", map[string]string{})
	viper.SetDefault("remote_user", "")

	if err := viper.ReadInConfig(); err == nil {
		if Verbose {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}

	err := viper.Unmarshal(&config)
	if err != nil {
		panic(fmt.Errorf("Unable to decode Config: %s \n", err))
	}
}

func displayMagentoCommands(commands []*cobra.Command) {
	fmt.Println("This command has the following sub commands:")
	for _, command := range commands {
		fmt.Printf("\t%v (%v)\n", command.Use, command.Short)
	}
}
