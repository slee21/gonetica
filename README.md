# gonetica: Unofficial Go wrapper for Netica C API

Gonetica provides a rough interface for interacting with Netica Bayesian networks and performing Bayesian inference with the Go programming language. Go allows easy distribution of binary executables without bothering users with the compilation process.

## Binary Executable Download

Binary executables are available for download at the gonetica Github [releases](https://github.com/slee21/gonetica/releases) page.

## Compiling the Source Code
### Requirements
* `go version` >= 1.5
* On Windows, a gcc compiler (for instance, mingw-w64) and gcc.exe in PATH environment variable
### Instructions
Get a copy of the package source code:
`$go get github.com/slee21/gonetica`

Compile the package executable module:
`$go install github.com/slee21/gncli`

The compiled binary executable should be available in the bin directory under the GOPATH environment variable.

On Windows, copy the appropriate DLL from `github.com/slee21/gonetica/cgo/bin/windows` into the executable directory:
* `386/Netica.dll` for 32-bit systems
* `amd64/Netica.dll` for 64-bit systems

## Running the Executable
To start serving with default configuration:
`$gncli serve json` or shortcut `$gncli serve`

For description of configurable options/flags:
`gncli serve json --help`

## JSON API Consumption
Source code excerpt describing the available API endpoints:
```
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
	"description": "Perform Bayesian inference on #netid with #nodeid as target node and JSON payload as cases."}
```

## Limitations
* Bayesian networks must meet requirements to be loaded
    - supported file extension
        * `.dne` or `.neta`
    - name must be unique
    - Able to be compiled
        * dynamic links must be expanded
        * continuous nodes must be discretised
        * no inconsistencies or conflicts
        
* Only accepts deterministic findings
* Only checks for conflicts after all findings in a case have been entered
    - Conflict unreported if inference target has finding entered
* Only accepts deterministic findings
* Only Netica is supported as backend for Bayesian inference