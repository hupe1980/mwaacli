// Package main is the entry point for the MWAA CLI application.
// It initializes and executes the command-line interface.
package main

import (
	"github.com/hupe1980/mwaacli/cmd"
)

var (
	// version specifies the current version of the application.
	// It is set during the build process.
	version = "dev"
)

// main is the entry point of the application.
// It executes the command-line interface with the provided version.
func main() {
	cmd.Execute(version)
}
