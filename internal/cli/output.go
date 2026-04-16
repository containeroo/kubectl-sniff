package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	envSniffQuiet   = "SNIFF_QUIET"
	envSniffVerbose = "SNIFF_VERBOSE"
)

// ResolveQuietVerbose resolves CLI flags and environment defaults.
// Explicit --quiet always wins over verbose settings from any source.
func ResolveQuietVerbose(cmd *cobra.Command, quietFlag, verboseFlag bool) (bool, bool) {
	return resolveQuietVerbose(cmd, quietFlag, verboseFlag, os.LookupEnv)
}

// resolveQuietVerbose resolves CLI flags and environment defaults using the provided lookup function.
func resolveQuietVerbose(
	cmd *cobra.Command,
	quietFlag bool,
	verboseFlag bool,
	lookupEnv func(string) (string, bool),
) (bool, bool) {
	quiet := quietFlag
	if !cmd.Flags().Changed("quiet") {
		quiet = envEnabled(envSniffQuiet, lookupEnv)
	}

	verbose := verboseFlag
	if !cmd.Flags().Changed("verbose") {
		verbose = envEnabled(envSniffVerbose, lookupEnv)
	}

	if quiet {
		verbose = false
	}

	return quiet, verbose
}

// envEnabled reports whether the named environment variable enables a boolean option.
func envEnabled(name string, lookupEnv func(string) (string, bool)) bool {
	value, ok := lookupEnv(name)
	if !ok {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
