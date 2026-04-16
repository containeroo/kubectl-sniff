package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveQuietVerbose verifies quiet and verbose resolution across flags and env values.
func TestResolveQuietVerbose(t *testing.T) {
	t.Parallel()

	t.Run("defaults disabled", func(t *testing.T) {
		t.Parallel()
		command := newOutputCommand(t)
		gotQuiet, gotVerbose := resolveQuietVerbose(command, false, false, lookupEnvFromMap(nil))
		assert.False(t, gotQuiet)
		assert.False(t, gotVerbose)
	})

	t.Run("verbose from env", func(t *testing.T) {
		t.Parallel()
		command := newOutputCommand(t)
		gotQuiet, gotVerbose := resolveQuietVerbose(command, false, false, lookupEnvFromMap(map[string]string{
			envSniffVerbose: "1",
		}))
		assert.False(t, gotQuiet)
		assert.True(t, gotVerbose)
	})

	t.Run("quiet from env overrides verbose env", func(t *testing.T) {
		t.Parallel()
		command := newOutputCommand(t)
		gotQuiet, gotVerbose := resolveQuietVerbose(command, false, false, lookupEnvFromMap(map[string]string{
			envSniffQuiet:   "1",
			envSniffVerbose: "1",
		}))
		assert.True(t, gotQuiet)
		assert.False(t, gotVerbose)
	})

	t.Run("quiet flag false overrides quiet env", func(t *testing.T) {
		t.Parallel()
		command := newOutputCommand(t)
		require.NoError(t, command.Flags().Set("quiet", "false"))
		gotQuiet, gotVerbose := resolveQuietVerbose(command, false, false, lookupEnvFromMap(map[string]string{
			envSniffQuiet: "1",
		}))
		assert.False(t, gotQuiet)
		assert.False(t, gotVerbose)
	})

	t.Run("verbose flag true overridden by quiet env", func(t *testing.T) {
		t.Parallel()
		command := newOutputCommand(t)
		require.NoError(t, command.Flags().Set("verbose", "true"))
		gotQuiet, gotVerbose := resolveQuietVerbose(command, false, true, lookupEnvFromMap(map[string]string{
			envSniffQuiet: "1",
		}))
		assert.True(t, gotQuiet)
		assert.False(t, gotVerbose)
	})

	t.Run("quiet flag wins over verbose flag", func(t *testing.T) {
		t.Parallel()
		command := newOutputCommand(t)
		require.NoError(t, command.Flags().Set("quiet", "true"))
		require.NoError(t, command.Flags().Set("verbose", "true"))
		gotQuiet, gotVerbose := resolveQuietVerbose(command, true, true, lookupEnvFromMap(nil))
		assert.True(t, gotQuiet)
		assert.False(t, gotVerbose)
	})
}

// newOutputCommand constructs a test command with quiet and verbose flags.
func newOutputCommand(t *testing.T) *cobra.Command {
	t.Helper()

	command := &cobra.Command{Use: "test"}
	flags := command.Flags()
	flags.Bool("quiet", false, "")
	flags.Bool("verbose", false, "")
	return command
}

// lookupEnvFromMap builds an environment lookup function backed by a static map.
func lookupEnvFromMap(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		if values == nil {
			return "", false
		}

		value, ok := values[name]
		return value, ok
	}
}
