package main

import (
	"github.com/hupe1980/mwaacli/cmd"
)

var (
	version = "dev"
)

func main() {
	cmd.Execute(version)
}
