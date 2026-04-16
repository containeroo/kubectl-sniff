package cli

import (
	"github.com/containeroo/sniff/internal/debugpod"
	"github.com/spf13/cobra"
)

// ValidateProfileFlag validates the user-provided --profile value.
func ValidateProfileFlag(profile string) error {
	return debugpod.ValidateProfile(profile)
}

// RegisterProfileFlagCompletion adds shell completion for the --profile flag.
func RegisterProfileFlagCompletion(cmd *cobra.Command) {
	if err := cmd.RegisterFlagCompletionFunc("profile", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return debugpod.SupportedProfiles(), cobra.ShellCompDirectiveNoFileComp
	}); err != nil {
		panic(err)
	}
}
