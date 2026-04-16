package main

import (
	"fmt"
	"os"

	"github.com/containeroo/sniff/cmd"
)

// main runs the sniff command-line entrypoint.
func main() {
	rootCmd := cmd.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
