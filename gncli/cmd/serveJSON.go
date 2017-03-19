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
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// netJSON is the JSON representation of a Network.
type netJSON struct {
	Index   int         `json:"index"`
	Name    string      `json:"name"`
	Title   string      `json:"title"`
	Comment string      `json:"comment"`
	Nodes   []*nodeJSON `json:"nodes"`
}

// nodeJSON is the JSON respresentation of a Node.
type nodeJSON struct {
	Index   int       `json:"index"`
	Name    string    `json:"name"`
	Title   string    `json:"title"`
	Comment string    `json:"comment"`
	States  []string  `json:"states"`
	Levels  []float64 `json:"levels"`
}

// caseJSON is the JSON respresentation of a Case for Bayesian inference.
type caseJSON struct {
	ID    string              `json:"id"`
	Cases []map[string]string `json:"cases"`
}

// batchJSON is the JSON respresentation of the batch results of Bayesian inference.
type batchJSON struct {
	ID      string        `json:"id"`
	Results []*singleJSON `json:"results"`
}

// singleJSON is the JSON respresentation of a single result of Bayesian inference.
type singleJSON struct {
	Index int    `json:"index"`
	Error string `json:"error"`
	Value string `json:"value"`
}

var (
	netJSONList []*netJSON
	netsJSON    map[string]*netJSON

	serveJSONLock sync.RWMutex
)

// serveJSONCmd represents the JSON API server command
var serveJSONCmd = &cobra.Command{
	Use:   "json",
	Short: "Serve JSON requests for Bayesian inference with Netica",
	Long: `A JSON API server process that performs Bayesian inference in response to JSON 
requests indicating the target Bayesnet and case data. It does not support 
delayed result retreival and hence reasonable rate limits should be enforced.`,
	RunE: serveJSON,
}

// json starts the JSON API server.
func serveJSON(cmd *cobra.Command, args []string) error {
	// Initialise common server resources and check for errors
	err := initServe()
	if err != nil {
		return err
	}
	// Build structs for JSON outputs and check for errors
	serveJSONLock.Lock()
	netJSONList, netsJSON, err = buildJSON()
	serveJSONLock.Unlock()
	if err != nil {
		return err
	}
	// Start JSON api using go-json-rest framework and check for errors
	host := net.JoinHostPort(viper.GetString("bind"), strconv.Itoa(viper.GetInt("port")))
	api := initMiddleware(rest.NewApi())
	api, err = initRouter(api, apiPrefix)
	if err != nil {
		return err
	}
	return http.ListenAndServe(host, api.MakeHandler())
}

// buildJSON constructs the JSON representation of loaded Networks and Nodes.
func buildJSON() ([]*netJSON, map[string]*netJSON, error) {
	var list []*netJSON
	var nodes []*nodeJSON
	var nets = make(map[string]*netJSON)
	serveLock.RLock()
	defer serveLock.RUnlock()
	// Iterate over Networks in neticaEnv, building JSON representation and check for errors
	for netIndex, net := range netList {
		netRepr := &netJSON{netIndex, net.Name(), net.Title(), net.Comment(), nil}
		nodeList, err := net.NodeList()
		// If error building net JSON representation, log error and skip
		if err != nil {
			log.Println(err)
			continue
		}
		nodes = nil
		// Iterate over Nodes in net, building JSON representationo and check for errors
		for index, node := range nodeList {
			repr := &nodeJSON{index, node.Name(), node.Title(), node.Comment(), nil, nil}
			names, err := node.StateNameList()
			// Check for errors, break out of Node loop on error
			if err != nil {
				break
			}
			repr.States = names
			levels, err := node.LevelList()
			// Check for errors, break out of Node loop on error
			if err != nil {
				break
			}
			repr.Levels = levels
			nodes = append(nodes, repr)
		}
		// If error building net JSON representation, log error and skip
		if err != nil {
			log.Println(err)
			continue
		}
		list = append(list, netRepr)
		modRepr := *netRepr
		modRepr.Nodes = nodes
		nets[modRepr.Name] = &modRepr
		nets[strconv.Itoa(netIndex)] = &modRepr
	}
	return list, nets, nil
}

// initMiddleware initialises Middleware to add functionality to the JSON API.
func initMiddleware(api *rest.Api) *rest.Api {
	api.Use(rest.DefaultProdStack...)
	// allow cross-origin resource sharing
	api.Use(&rest.CorsMiddleware{
		RejectNonCorsRequests: false,
		OriginValidator: func(origin string, request *rest.Request) bool {
			return true
		},
		AllowedMethods: []string{"GET", "POST", "PUT"},
		AllowedHeaders: []string{
			"Accept", "Content-Type", "X-Custom-Header", "Origin"},
		AccessControlAllowCredentials: true,
		AccessControlMaxAge:           3600,
	})
	return api
}

// initRouter initialises the JSON API request router.
func initRouter(api *rest.Api, prefix string) (*rest.Api, error) {
	// Initialise router and check for errors
	router, err := rest.MakeRouter(
		rest.Get(apiPrefix, getAPI),
		rest.Get(apiPrefix+"/nets", getNets),
		rest.Get(apiPrefix+"/nets/#netid", getNet),
		rest.Get(apiPrefix+"/nets/#netid/nodes", getNetNodes),
		rest.Get(apiPrefix+"/nets/#netid/nodes/#nodeid", getNetNode),
		rest.Post(apiPrefix+"/nets/#netid/nodes/#nodeid", postNetNode),
	)
	api.SetApp(router)
	if err != nil {
		return nil, err
	}
	apiRoutes = []map[string]string{
		{"path": apiPrefix + "/nets",
			"method":      "GET",
			"description": "List all loaded Bayesian networks."},
		{"path": apiPrefix + "/nets/#netid",
			"method":      "GET",
			"description": "Describe #netid and list contained nodes."},
		{"path": apiPrefix + "/nets/#netid/nodes",
			"method":      "GET",
			"description": "List all nodes contained in #netid."},
		{"path": apiPrefix + "/nets/#netid/nodes/#nodeid",
			"method":      "GET",
			"description": "Describe #nodeid in #netid."},
		{"path": apiPrefix + "/nets/#netid/nodes/#nodeid",
			"method":      "POST",
			"description": "Perform Bayesian inference on #netid with #nodeid as target node and JSON payload as cases."},
	}
	return api, nil
}

// getAPI returns JSON listing all valid api paths.
func getAPI(w rest.ResponseWriter, r *rest.Request) {
	w.WriteJson(apiRoutes)
}

// getNets returns JSON listing all loaded Networks.
func getNets(w rest.ResponseWriter, r *rest.Request) {
	w.WriteJson(netJSONList)
}

// getNet returns JSON detailing specific Network and contained nodes.
func getNet(w rest.ResponseWriter, r *rest.Request) {
	netID := r.PathParam("netid")
	// Return Network JSON representation if loaded, NotFound otherwise
	if repr, ok := netsJSON[netID]; ok {
		w.WriteJson(repr)
	} else {
		rest.NotFound(w, r)
	}
}

// getNetNodes returns JSON nodes contained in a specific Network.
func getNetNodes(w rest.ResponseWriter, r *rest.Request) {
	netID := r.PathParam("netid")
	// Return Network JSON representation if loaded, NotFound otherwise
	if repr, ok := netsJSON[netID]; ok {
		w.WriteJson(repr.Nodes)
	} else {
		rest.NotFound(w, r)
	}
}

// getNetNode returns JSON a specific node contained in a specific Network.
func getNetNode(w rest.ResponseWriter, r *rest.Request) {
	netID := r.PathParam("netid")
	// Return Network JSON representation if loaded, NotFound otherwise
	if repr, ok := netsJSON[netID]; ok {
		nodeID := r.PathParam("nodeid")
		for index, node := range repr.Nodes {
			if strconv.Itoa(index) == nodeID || node.Name == nodeID {
				w.WriteJson(node)
				return
			}
		}
		rest.NotFound(w, r)
	} else {
		rest.NotFound(w, r)
	}
}

// postNetNode returns JSON Bayesian inference results of a specific node in a specific network given JSON payload case.
func postNetNode(w rest.ResponseWriter, r *rest.Request) {
	netID := r.PathParam("netid")
	// Validated target network and node and check for errors
	if repr, ok := netsJSON[netID]; ok {
		net := netLookup[netID]
		// Attempt to lookup node by name
		nodeID := r.PathParam("nodeid")
		node, err := net.NodeNamed(nodeID)
		if err != nil {
			// Attempt to lookup node by index
			index, err := strconv.Atoi(nodeID)
			if err != nil {
				rest.NotFound(w, r)
				return
			}
			node, err = net.NodeNamed(repr.Nodes[index].Name)
			if err != nil {
				rest.NotFound(w, r)
				return
			}
		}
		// Decode case data from JSON payload and check for errors
		infer := new(caseJSON)
		err = r.DecodeJsonPayload(infer)
		if err != nil {
			rest.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		batch := &batchJSON{infer.ID, nil}
		// Iterate over case data and build up results and check for errors
		for index, evidence := range infer.Cases {
			// Enter case data and check for errors
			net.Lock()
			err = net.EnterCase(evidence)
			if err != nil {
				net.Unlock()
				log.Println(err)
				batch.Results = append(batch.Results, &singleJSON{index, err.Error(), ""})
				continue
			}
			// Infer value of target node and check for errors
			result, err := node.Infer()
			if err != nil {
				net.ClearCases()
				net.Unlock()
				log.Println(err)
				batch.Results = append(batch.Results, &singleJSON{index, err.Error(), ""})
				continue
			}
			// Clear cases from network and append result to batch
			net.ClearCases()
			net.Unlock()
			batch.Results = append(batch.Results, &singleJSON{index, "", result})
		}
		w.WriteJson(batch)
	} else {
		rest.NotFound(w, r)
	}
}
