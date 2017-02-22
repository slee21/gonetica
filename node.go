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
	"strconv"
	"strings"
	"unsafe"
)

// Node is a node or variable in Netica's Bayesnet.
type Node struct {
	c *C.node_bn

	Net *Network
}

// Errors returns all Netica errors of severity level error since it was last called.
func (node *Node) Errors() error {
	return node.Net.Errors()
}

// Name returns the name of the Node.
func (node *Node) Name() string {
	return C.GoString(C.GetNodeName_bn(node.c))
}

// Title returns the title of the Node.
func (node *Node) Title() string {
	return C.GoString(C.GetNodeTitle_bn(node.c))
}

// Comment returns the comment of the Node.
func (node *Node) Comment() string {
	return C.GoString(C.GetNodeComment_bn(node.c))
}

// IsDiscreteType returns bool whether node is discrete type.
func (node *Node) IsDiscreteType() bool {
	return C.GetNodeType_bn(node.c) == C.DISCRETE_TYPE
}

// IsContinuousType returns bool whether node is continuous type.
func (node *Node) IsContinuousType() bool {
	return C.GetNodeType_bn(node.c) == C.CONTINUOUS_TYPE
}

// StateNamed returns index of state with name if exists error otherwise.
func (node *Node) StateNamed(name string) (int, error) {
	var index int
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	cIndex := int(C.GetStateNamed_bn(cName, node.c))
	// Check for errors
	if err := node.Errors(); err != nil {
		return 0, err
	}
	// Check for undefined expected value
	if cIndex == C.UNDEF_STATE {
		return 0, fmt.Errorf("In function Node.StateNamed: state %s not defined for node %s", name, node.Name())
	}
	index = int(cIndex)
	return index, nil
}

// StateNameList returns a Slice of state name strings in order.
func (node *Node) StateNameList() ([]string, error) {
	var stateNames []string
	length := C.GetNodeNumberStates_bn(node.c)
	// Iterate over node states saving name strings in stateNames
	for index := C.int(0); index < length; index++ {
		stateName := C.GoString(C.GetNodeStateName_bn(node.c, C.state_bn(index)))
		stateNames = append(stateNames, stateName)
	}
	// Check for errors
	if err := node.Errors(); err != nil {
		return nil, err
	}
	return stateNames, nil
}

// LevelList returns a Slice of level floats in order.
func (node *Node) LevelList() ([]float64, error) {
	var levels []float64
	// Return empty slice for node with no levels
	cLevels := C.GetNodeLevels_bn(node.c)
	if cLevels == nil {
		return nil, nil
	}
	// Get number of levels based on node type
	length := C.GetNodeNumberStates_bn(node.c)
	if node.IsContinuousType() {
		length++
	}
	// Iterate over node levels saving level floats in levels
	for index := C.int(0); index < length; index++ {
		levels = append(levels, float64(C.NthLevel_bn(cLevels, C.state_bn(index))))
	}
	// Check for errors
	if err := node.Errors(); err != nil {
		return nil, err
	}
	return levels, nil
}

// SetState enters a state finding for a discrete type node.
func (node *Node) SetState(state int) error {
	C.EnterFinding_bn(node.c, C.state_bn(state))
	// Check for errors, clear node findings on error
	if err := node.Errors(); err != nil {
		node.ClearFindings()
		return err
	}
	return nil
}

// SetValue enters a real value finding for a continuous type node.
func (node *Node) SetValue(value float64) error {
	C.EnterNodeValue_bn(node.c, C.double(value))
	// Check for errors, clear node findings on error
	if err := node.Errors(); err != nil {
		node.ClearFindings()
		return err
	}
	return nil
}

// EnterFinding enters an evidence string which may be a discrete state or real value.
func (node *Node) EnterFinding(evidence string) error {
	// Try to enter evidence as real value and check for errors
	value, err := strconv.ParseFloat(evidence, 64)
	if err == nil {
		return node.SetValue(value)
	}
	// Try to enter evidence as state index
	if strings.HasPrefix(evidence, "#") {
		state := strings.TrimPrefix(evidence, "#")
		index, err := strconv.Atoi(state)
		if err == nil {
			return node.SetState(index)
		}
	}
	// Try to enter evidence as state name and check for errors
	index, err := node.StateNamed(evidence)
	if err != nil {
		return err
	}
	return node.SetState(index)
}

// ClearFindings retracts all findings for the node.
func (node *Node) ClearFindings() error {
	// Retract any findings in node and check for errors
	C.RetractNodeFindings_bn(node.c)
	return node.Errors()
}

// BeliefList returns Slice of belief floats in order of states.
func (node *Node) BeliefList() ([]float64, error) {
	var beliefs []float64
	cBeliefs := C.GetNodeBeliefs_bn(node.c)
	length := C.GetNodeNumberStates_bn(node.c)
	// Iterate over node beliefs saving belief floats in beliefs
	for index := C.int(0); index < length; index++ {
		beliefs = append(beliefs, float64(C.NthProb_bn(cBeliefs, C.state_bn(index))))
	}
	// Check for errors
	if err := node.Errors(); err != nil {
		return nil, err
	}
	return beliefs, nil
}

// State returns most likely state of node
func (node *Node) State() (int, error) {
	// Get list of node beliefs
	beliefList, err := node.BeliefList()
	if err != nil {
		return 0, err
	}
	// Return first state with max belief
	maxIndex := 0
	maxBelief := beliefList[maxIndex]
	for index, belief := range beliefList {
		if belief > maxBelief {
			maxIndex = index
			maxBelief = belief
		}
	}
	return maxIndex, nil
}

// Value returns expected value of node
func (node *Node) Value() (float64, float64, error) {
	var value float64
	var stdDev float64
	// Allocate memory for std dev
	cStdDev := (*C.double)(C.malloc(C.sizeof_double))
	defer C.free(unsafe.Pointer(cStdDev))
	// Calculate expected value with std dev
	cValue := C.GetNodeExpectedValue_bn(node.c, cStdDev, nil, nil)
	// Check for errors
	if err := node.Errors(); err != nil {
		return 0, 0, err
	}
	// Check for undefined expected value
	if cValue == C.GetUndefDbl_ns() {
		return 0, 0, fmt.Errorf("%s: %s", "In function Node.Value", "undefined expected value")
	}
	value = float64(cValue)
	stdDev = float64(*cStdDev)
	return value, stdDev, nil
}

// Infer attempts to infer the value or state of the node.
func (node *Node) Infer() (string, error) {
	// Try to return a real value estimate
	value, _, err := node.Value()
	if err == nil {
		return strconv.FormatFloat(value, 'E', -1, 64), nil
	}
	// Try to return a discrete state estimate
	index, err := node.State()
	if err == nil {
		// Try to return state name
		cName := C.GetNodeStateName_bn(node.c, C.state_bn(index))
		// Check for errors
		if err := node.Errors(); err != nil {
			return "", err
		}
		name := C.GoString(cName)
		if name != "" {
			return name, nil
		}
		// Return state index
		return fmt.Sprintf("#%d", index), nil
	}
	return "", err
}
