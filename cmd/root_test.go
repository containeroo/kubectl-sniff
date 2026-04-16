package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRootValidation verifies the preferred root workflow validation rules.
func TestRootValidation(t *testing.T) {
	t.Parallel()

	t.Run("attach mode rejects clone-only command flag", func(t *testing.T) {
		t.Parallel()

		command := NewRootCmd()
		command.SetArgs([]string{"demo-pod", "--image", "alpine", "--command", "sh"})

		err := command.Execute()
		require.Error(t, err)
		assert.ErrorContains(t, err, "--command requires --clone")
	})

	t.Run("attach mode rejects interactive flags without command", func(t *testing.T) {
		t.Parallel()

		command := NewRootCmd()
		command.SetArgs([]string{"demo-pod", "--image", "alpine", "-it"})

		err := command.Execute()
		require.Error(t, err)
		assert.ErrorContains(t, err, "-i/--stdin and -t/--tty require a command after --")
	})

	t.Run("clone mode rejects attach-only flags", func(t *testing.T) {
		t.Parallel()

		command := NewRootCmd()
		command.SetArgs([]string{"demo-pod", "--image", "alpine", "--clone", "--target", "app"})

		err := command.Execute()
		require.Error(t, err)
		assert.ErrorContains(t, err, "--target is only supported when attaching an ephemeral container")
	})

	t.Run("clone mode rejects exec command after dash", func(t *testing.T) {
		t.Parallel()

		command := NewRootCmd()
		command.SetArgs([]string{"demo-pod", "--image", "alpine", "--clone", "--", "sh"})

		err := command.Execute()
		require.Error(t, err)
		assert.ErrorContains(t, err, "--clone does not accept a command after --; use --command and --arg instead")
	})
}

// TestResolveRootBoolFlag verifies the mode-dependent default handling used by the root command.
func TestResolveRootBoolFlag(t *testing.T) {
	t.Parallel()

	t.Run("uses default when the flag is unchanged", func(t *testing.T) {
		t.Parallel()

		command := newRootBoolCommand(t)
		assert.True(t, resolveRootBoolFlag(command, "stdin", true, false))
		assert.False(t, resolveRootBoolFlag(command, "stdin", false, true))
	})

	t.Run("keeps the explicit flag value when the flag is set", func(t *testing.T) {
		t.Parallel()

		command := newRootBoolCommand(t)
		require.NoError(t, command.Flags().Set("stdin", "false"))
		assert.False(t, resolveRootBoolFlag(command, "stdin", true, false))

		command = newRootBoolCommand(t)
		require.NoError(t, command.Flags().Set("stdin", "true"))
		assert.True(t, resolveRootBoolFlag(command, "stdin", false, true))
	})
}

func newRootBoolCommand(t *testing.T) *cobra.Command {
	t.Helper()

	command := &cobra.Command{}
	command.Flags().Bool("stdin", false, "")
	return command
}
