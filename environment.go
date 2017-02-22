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
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"
)

// Environment is Netica's global execution context.
type Environment struct {
	c    *C.environ_ns
	cMsg *C.char

	netlocks map[*C.net_bn]*sync.RWMutex
}

// NewEnvironment returns a new initialised Environment with optional license string.
// The program will block and must be killed manually if an invalid license is entered.
// Presumably this is to discourage bruteforcing Netica license keys.
func NewEnvironment(license string) (*Environment, error) {
	var env = new(Environment)
	var cLic *C.char
	// Load license if provided
	if license != "" {
		cLic = C.CString(license)
		defer C.free(unsafe.Pointer(cLic))
	}
	// Initialise environment
	env.c = C.NewNeticaEnviron_ns(cLic, nil, nil)
	// Allocate message
	env.cMsg = (*C.char)(C.malloc(C.MESG_LEN_ns * C.sizeof_char))
	res := C.InitNetica2_bn(env.c, env.cMsg)
	// Check for errors
	if res < 0 {
		env.CloseEnvironment()
		return nil, fmt.Errorf("%d - %s: %s", res, "In function InitNetica2_bn", "error initialising environment")
	}
	C.ArgumentChecking_ns(C.QUICK_CHECK, env.c)
	// Initialise synchronisation map
	env.netlocks = make(map[*C.net_bn]*sync.RWMutex)
	return env, nil
}

// CloseEnvironment closes the Environment, freeing resources.
func (env *Environment) CloseEnvironment() error {
	// Free up allocated resources
	defer C.free(unsafe.Pointer(env.cMsg))
	res := C.CloseNetica_bn(env.c, env.cMsg)
	// Check for errors
	if res < 0 {
		return fmt.Errorf("%d - %s: %s", res, "In function CloseNetica_bn", "error closing environment")
	}
	return nil
}

// Errors returns all Netica errors of severity level error since it was last called.
func (env *Environment) Errors() error {
	var messages []string
	var cRep *C.report_ns
	// Iterate over Netica errors and clear them after saving in messages
	for cRep = C.GetError_ns(env.c, C.ERROR_ERR, nil); cRep != nil; cRep = C.GetError_ns(env.c, C.ERROR_ERR, nil) {
		messages = append(messages, fmt.Sprintf("%d - %s", C.ErrorNumber_ns(cRep), C.GoString(C.ErrorMessage_ns(cRep))))
		C.ClearError_ns(cRep)
	}
	// Return messages as errors if not empty
	if messages != nil {
		return errors.New(strings.Join(messages, "\n"))
	}
	return nil
}

// Message returns Netica message for non-error purposes.
func (env *Environment) Message() string {
	return C.GoString(env.cMsg)
}

// NetworkList returns a Slice of Networks in the order they were read.
func (env *Environment) NetworkList() ([]*Network, error) {
	var networks []*Network
	// Iterate over Netica nets and save them as Network in networks
	for index := C.int(0); C.GetNthNet_bn(index, env.c) != nil; index++ {
		networks = append(networks, &Network{C.GetNthNet_bn(index, env.c), env})
	}
	// Check for errors
	if err := env.Errors(); err != nil {
		return nil, err
	}
	return networks, nil
}
