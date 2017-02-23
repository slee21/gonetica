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
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/slee21/gonetica"
)

var (
	netList   []*gonetica.Network
	netLookup map[string]*gonetica.Network

	serveLock sync.RWMutex

	apiPrefix string
	apiRoutes []map[string]string
)

// serveCmd represents the server command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve HTTP requests for Bayesian inference with Netica",
	Long: `Serve starts a long-running server process that loads Bayesnets once on startup
then performs Bayesian inference in response to HTTP requests indicating the
target Bayesnet and case data. It does not support HTTPS and should be 
proxied behind a real webserver such as Apache or Nginx if desired.
Serves JSON by default.`,
	RunE: serveJSON,
}

func init() {
	RootCmd.AddCommand(serveCmd)

	// Get Bayesnets directory
	netDir := "."
	if exeDir != "" {
		netDir = exeDir
	}

	// Initialise flags
	serveCmd.PersistentFlags().String("dir", filepath.Join(netDir, "bayesnets"), "directory where Netica Bayesnet files are located")
	serveCmd.PersistentFlags().String("bind", "127.0.0.1", "interface to which the server will bind")
	serveCmd.PersistentFlags().Int("port", 8080, "port on which the server will listen")
	serveCmd.PersistentFlags().String("prefix", "api", "path prefix from which requests will be served")

	// Bind flags to 12 factor interface
	viper.BindPFlag("dir", serveCmd.PersistentFlags().Lookup("dir"))
	viper.BindPFlag("port", serveCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("bind", serveCmd.PersistentFlags().Lookup("bind"))
	viper.BindPFlag("prefix", serveCmd.PersistentFlags().Lookup("prefix"))

	// Add subcommands based on request format
	serveCmd.AddCommand(serveJSONCmd)
}

// initAPIPrefix initialises path prefix to serve HTTP requests from.
func initAPIPrefix(prefix string) string {
	path := "/"
	if prefix != "" {
		if !strings.HasPrefix(prefix, "/") {
			path = path + prefix
		} else {
			path = prefix
		}
	}
	return path
}

// initServe initialises Netica and reads available Bayesnets before server start.
func initServe() error {
	// Initialise Netica and check for errors
	err := initNetica(viper.GetString("license"))
	if err != nil {
		return err
	}
	// Read Bayesnets in dir, index them by relative path and check for errors
	serveLock.Lock()
	netList, netLookup, err = indexNets(neticaEnv, viper.GetString("dir"))
	serveLock.Unlock()
	if err != nil {
		return err
	}
	// Initialise path prefix
	apiPrefix = initAPIPrefix(viper.GetString("prefix"))
	return nil
}

// indexNets reads Netica Bayesnets in dir into env and index them in a list and map.
func indexNets(env *gonetica.Environment, dir string) ([]*gonetica.Network, map[string]*gonetica.Network, error) {
	var nets []*gonetica.Network
	var lookup = make(map[string]*gonetica.Network)
	root := filepath.Clean(dir)
	// Recursively iterate over files in dir and check for errors
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// Only process .dne and .neta files
		if !info.IsDir() && (filepath.Ext(path) == ".dne" || filepath.Ext(path) == ".neta") {
			// Read file into Netica Bayesnet and check for errors
			net, err := gonetica.NewNetwork(neticaEnv, path)
			if err != nil {
				// If error reading net, log error and skip
				log.Println(err)
				return nil
			}
			name := net.Name()
			// Get relative path of path from root
			relPath, _ := filepath.Rel(root, path)
			// Check if network with name already exists
			if lookup[name] != nil {
				net.CloseNetwork()
				err = fmt.Errorf("In function serve: network named %s already loaded from path %s", name, relPath)
				log.Println(err)
				return nil
			}
			// Index network in lists and map
			nets = append(nets, net)
			lookup[name] = net
			lookup[strconv.Itoa(len(nets)-1)] = net
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return nets, lookup, nil
}
