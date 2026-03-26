// Package main is the entry point for the FactoryAI CLI.
package main

import (
	"os"

	"github.com/uttufy/FactoryAI/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
