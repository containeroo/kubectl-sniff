package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	envSniffRunCommand = "SNIFF_RUN_COMMAND"
	envSniffRunArgs    = "SNIFF_RUN_ARGS"
)

// ResolveRunCommand resolves standalone-run command defaults from flags and environment.
func ResolveRunCommand(cmd *cobra.Command, command []string, args []string) ([]string, []string, error) {
	return resolveRunCommand(cmd, command, args, os.LookupEnv)
}

// resolveRunCommand resolves standalone-run command defaults using the provided lookup function.
func resolveRunCommand(
	cmd *cobra.Command,
	command []string,
	args []string,
	lookupEnv func(string) (string, bool),
) ([]string, []string, error) {
	if !cmd.Flags().Changed("command") {
		command = commandFromEnv(lookupEnv)
	}

	if !cmd.Flags().Changed("arg") {
		var err error
		args, err = argsFromEnv(lookupEnv)
		if err != nil {
			return nil, nil, err
		}
	}

	return command, args, nil
}

// commandFromEnv returns the standalone-run command from the environment when configured.
func commandFromEnv(lookupEnv func(string) (string, bool)) []string {
	value, ok := lookupEnv(envSniffRunCommand)
	if !ok {
		return nil
	}

	command := strings.TrimSpace(value)
	if command == "" {
		return nil
	}

	return []string{command}
}

// argsFromEnv returns the standalone-run args from the environment when configured.
func argsFromEnv(lookupEnv func(string) (string, bool)) ([]string, error) {
	value, ok := lookupEnv(envSniffRunArgs)
	if !ok {
		return nil, nil
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	var args []string
	if err := json.Unmarshal([]byte(trimmed), &args); err != nil {
		return nil, fmt.Errorf("%s must be a JSON array of strings: %w", envSniffRunArgs, err)
	}

	return args, nil
}
