// Copyright © 2017 Lee Sheng Long <s.lee.21@warwick.ac.uk>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kardianos/osext"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/slee21/gonetica"
)

var (
	cfgFile string
	exeDir  string

	neticaEnv *gonetica.Environment
)

// RootCmd represents the base command when called without any subcommands.
var RootCmd = &cobra.Command{
	Use:   "gncli",
	Short: "CLI to perform Bayesian inference with Netica given Bayesnets and cases",
	Long: `Gonetica is an unofficial Go wrapper around the Netica C API. 
The executable provides tools to perform Bayesian inference with Netica 
given input Bayesnets and cases. The top-level program does nothing as 
functionality is grouped under commands.`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Get executable directory
	cfgDir := "."
	_ = initExeDir()
	if exeDir != "" {
		cfgDir = exeDir
	}

	// Initialise flags
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default \"%s\")", filepath.Join(cfgDir, ".gonetica")))
	RootCmd.PersistentFlags().String("license", "", "Netica license key to remove trial version limits (default no license)")

	// Bind flags to 12 factor interface
	viper.BindPFlag("license", RootCmd.PersistentFlags().Lookup("license"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}
	viper.SetConfigName(".gonetica") // name of config file (without extension)
	viper.AddConfigPath(exeDir)      // adding executable directory as first search path
	viper.AddConfigPath(".")         // adding current working directory as second search path
	viper.AutomaticEnv()             // read in environment variables that match
	viper.SetEnvPrefix("gonetica")   // only read environment variable prefixed with GONETICA_

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

}

// initExeDir initialises the executable directory and checks for errors.
func initExeDir() error {
	// Get executable directory and check for errors
	dir, err := osext.ExecutableFolder()
	if err != nil {
		return err
	}
	exeDir = dir
	return nil
}

// initNetica initialises netica with license and checks for errors.
func initNetica(license string) error {
	// Initialise netica and check for errors
	env, err := gonetica.NewEnvironment(license)
	if err != nil {
		return err
	}
	neticaEnv = env
	return nil
}
