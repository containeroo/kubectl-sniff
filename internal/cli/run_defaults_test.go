package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveRunCommand verifies standalone-run command precedence across flags and env values.
func TestResolveRunCommand(t *testing.T) {
	t.Parallel()

	t.Run("keeps explicit flag values", func(t *testing.T) {
		t.Parallel()
		command := newRunCommand(t)
		require.NoError(t, command.Flags().Set("command", "sh"))
		require.NoError(t, command.Flags().Set("arg", "-c"))

		gotCommand, gotArgs, err := resolveRunCommand(
			command,
			[]string{"sh"},
			[]string{"-c"},
			lookupEnvFromMap(map[string]string{
				envSniffRunCommand: "/bin/bash",
				envSniffRunArgs:    `["-lc","sleep infinity"]`,
			}),
		)
		require.NoError(t, err)
		assert.Equal(t, []string{"sh"}, gotCommand)
		assert.Equal(t, []string{"-c"}, gotArgs)
	})

	t.Run("uses command from env when flag is unset", func(t *testing.T) {
		t.Parallel()
		command := newRunCommand(t)

		gotCommand, gotArgs, err := resolveRunCommand(
			command,
			nil,
			nil,
			lookupEnvFromMap(map[string]string{
				envSniffRunCommand: "/bin/sh",
			}),
		)
		require.NoError(t, err)
		assert.Equal(t, []string{"/bin/sh"}, gotCommand)
		assert.Nil(t, gotArgs)
	})

	t.Run("uses args from env when flag is unset", func(t *testing.T) {
		t.Parallel()
		command := newRunCommand(t)

		gotCommand, gotArgs, err := resolveRunCommand(
			command,
			nil,
			nil,
			lookupEnvFromMap(map[string]string{
				envSniffRunArgs: `["-lc","sleep infinity"]`,
			}),
		)
		require.NoError(t, err)
		assert.Nil(t, gotCommand)
		assert.Equal(t, []string{"-lc", "sleep infinity"}, gotArgs)
	})

	t.Run("overrides env command only when command flag is set", func(t *testing.T) {
		t.Parallel()
		command := newRunCommand(t)
		require.NoError(t, command.Flags().Set("command", "sh"))

		gotCommand, gotArgs, err := resolveRunCommand(
			command,
			[]string{"sh"},
			nil,
			lookupEnvFromMap(map[string]string{
				envSniffRunCommand: "/bin/bash",
				envSniffRunArgs:    `["-lc","sleep infinity"]`,
			}),
		)
		require.NoError(t, err)
		assert.Equal(t, []string{"sh"}, gotCommand)
		assert.Equal(t, []string{"-lc", "sleep infinity"}, gotArgs)
	})

	t.Run("returns an error for invalid env args", func(t *testing.T) {
		t.Parallel()
		command := newRunCommand(t)

		_, _, err := resolveRunCommand(
			command,
			nil,
			nil,
			lookupEnvFromMap(map[string]string{
				envSniffRunArgs: "not-json",
			}),
		)
		require.Error(t, err)
		assert.ErrorContains(t, err, envSniffRunArgs)
	})
}

// newRunCommand constructs a test command with standalone-run command flags.
func newRunCommand(t *testing.T) *cobra.Command {
	t.Helper()

	command := newOutputCommand(t)
	flags := command.Flags()
	flags.StringSlice("command", nil, "")
	flags.StringSlice("arg", nil, "")
	return command
}
