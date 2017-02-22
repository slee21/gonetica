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

package gonetica

/*
#cgo darwin CFLAGS: -I"${SRCDIR}/cgo/lib/darwin"
#cgo darwin,amd64 LDFLAGS: -L"${SRCDIR}/cgo/lib/darwin/amd64"
#cgo darwin LDFLAGS: -lm -lnetica -lpthread -lstdc++
#cgo linux CFLAGS: -I"${SRCDIR}/cgo/lib/linux"
#cgo linux,386 LDFLAGS: -L"${SRCDIR}/cgo/lib/linux/386"
#cgo linux,amd64 LDFLAGS: -L"${SRCDIR}/cgo/lib/linux/amd64"
#cgo linux LDFLAGS: -lm -lrt -lnetica -lpthread -lstdc++
#cgo windows CFLAGS: -I"${SRCDIR}/cgo/lib/windows"
#cgo windows,386 LDFLAGS: -L"${SRCDIR}/cgo/lib/windows/386"
#cgo windows,amd64 LDFLAGS: -L"${SRCDIR}/cgo/lib/windows/amd64"
#cgo windows LDFLAGS: -lm -llibNetica -lpthread -lstdc++
#include "stdlib.h"
#include "Netica.h"
*/
import "C"

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"unsafe"
)

// Network is Netica's Bayesnet.
type Network struct {
	c *C.net_bn

	env *Environment
}

// NewNetwork parses file at path into a new Network and index with key.
func NewNetwork(environment *Environment, path string) (*Network, error) {
	var net = new(Network)
	var err error
	net.env = environment
	// Open file for reading and check for errors
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	// Allocate name string and check for errors
	name := filepath.Base(path)
	cName := (*C.char)(C.CString(name))
	defer C.free(unsafe.Pointer(cName))
	if err = net.Errors(); err != nil {
		return nil, err
	}
	// Allocate file stream and check for errors
	cStrm := C.NewMemoryStream_ns(cName, net.env.c, nil)
	defer C.DeleteStream_ns(cStrm)
	if err = net.Errors(); err != nil {
		return nil, err
	}
	// Read file and check for errors
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	// Stream file into Netica and check for errors
	cBuf := (*C.char)(C.CBytes(buf)) // C.CBytes causes spurious report prior to Go1.8: https://github.com/golang/go/issues/17563
	defer C.free(unsafe.Pointer(cBuf))
	C.SetStreamContents_ns(cStrm, cBuf, C.long(len(buf)), C.FALSE)
	if err = net.Errors(); err != nil {
		return nil, err
	}
	// Parse file into Netica network and check for errors
	net.c = C.ReadNet_bn(cStrm, C.NO_VISUAL_INFO)
	if err = net.Errors(); err != nil {
		return nil, err
	}
	// Compile network in Netica and check for errors
	C.CompileNet_bn(net.c)
	if err = net.Errors(); err != nil {
		return nil, err
	}
	// Retract any findings in network and check for errors
	C.RetractNetFindings_bn(net.c)
	if err = net.Errors(); err != nil {
		return nil, err
	}
	// Turn off automatic updating for network and check for errors
	C.SetNetAutoUpdate_bn(net.c, C.int(0))
	if err = net.Errors(); err != nil {
		return nil, err
	}
	// Register in synchronization map
	net.env.netlocks[net.c] = new(sync.RWMutex)
	return net, nil
}

// CloseNetwork closes the Network, freeing resources.
func (net *Network) CloseNetwork() error {
	// Delete network from Environment
	C.DeleteNet_bn(net.c)
	// Delete from synchronization map
	delete(net.env.netlocks, net.c)
	return net.Errors()
}

// Errors returns all Netica errors of severity level error since it was last called.
func (net *Network) Errors() error {
	return net.env.Errors()
}

// Name returns the name of the Network.
func (net *Network) Name() string {
	return C.GoString(C.GetNetName_bn(net.c))
}

// Title returns the title of the Network.
func (net *Network) Title() string {
	return C.GoString(C.GetNetTitle_bn(net.c))
}

// Comment returns the title of the Network.
func (net *Network) Comment() string {
	return C.GoString(C.GetNetComment_bn(net.c))
}

// NodeNamed returns Node in net with name.
func (net *Network) NodeNamed(name string) (*Node, error) {
	var node *Node
	// Allocate string name
	cName := (*C.char)(C.CString(name))
	defer C.free(unsafe.Pointer(cName))
	// Search underlying c network for node with name, error if not found
	cNode := C.GetNodeNamed_bn(cName, net.c)
	if cNode == nil {
		return nil, fmt.Errorf("In function Network.NodeNamed: node %s not defined for network %s", name, net.Name())
	}
	node = &Node{cNode, net}
	return node, nil
}

// NodeMap returns a Map of Nodes in the Network indexed by name.
func (net *Network) NodeMap() (map[string]*Node, error) {
	var nodes = make(map[string]*Node)
	var cNodes *C.nodelist_bn
	// Read and duplicate list of Netica nodes in Network
	cNodes = C.DupNodeList_bn(C.GetNetNodes2_bn(net.c, nil))
	defer C.DeleteNodeList_bn(cNodes)
	// Iterate over Netica nodes and save as Node in nodes
	for index := C.LengthNodeList_bn(cNodes) - 1; index >= 0; index-- {
		node := &Node{C.NthNode_bn(cNodes, index), net}
		nodes[node.Name()] = node
	}
	// Check for errors
	if err := net.Errors(); err != nil {
		return nil, err
	}
	return nodes, nil
}

// NodeList returns a Slice of Nodes sorted by name lexicographically ascending.
func (net *Network) NodeList() ([]*Node, error) {
	var nodes []*Node
	var names []string
	// Get nodes mapped by name
	nodeMap, err := net.NodeMap()
	if err != nil {
		return nil, err
	}
	for name := range nodeMap {
		names = append(names, name)
	}
	// Sort names
	sort.Strings(names)
	// Construct sorted node list
	for _, name := range names {
		nodes = append(nodes, nodeMap[name])
	}
	return nodes, nil
}

// EnterCase enters a set of findings into the network.
func (net *Network) EnterCase(caseMap map[string]string) error {
	// Get network nodes mapped by name and check for errors
	nodeMap, err := net.NodeMap()
	if err != nil {
		return err
	}
	for name, node := range nodeMap {
		if evidence, ok := caseMap[name]; ok {
			// Enter findings for each node in case
			err := node.EnterFinding(evidence)
			// Check for errors, retract all findings on error
			if err != nil {
				net.ClearCases()
				return err
			}
		}
	}
	return nil
}

// ClearCases retracts all findings in the network.
func (net *Network) ClearCases() error {
	// Retract any findings in network and check for errors
	C.RetractNetFindings_bn(net.c)
	return net.Errors()
}

// Lock acquires lock for writing to underlying C network.
func (net *Network) Lock() {
	net.env.netlocks[net.c].Lock()
}

// RLock acquires lock for reading from underlying C network.
func (net *Network) RLock() {
	net.env.netlocks[net.c].RLock()
}

// RUnlock releases lock for reading from underlying C network.
func (net *Network) RUnlock() {
	net.env.netlocks[net.c].RUnlock()
}

// Unlock releases lock for writing to underlying C network.
func (net *Network) Unlock() {
	net.env.netlocks[net.c].Unlock()
}
