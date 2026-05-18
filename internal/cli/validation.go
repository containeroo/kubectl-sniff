package cli

import "strings"

// IsSupportedOutputFormat reports whether the dry-run output format is supported.
func IsSupportedOutputFormat(output string) bool {
	return output == "" || output == "yaml" || output == "json"
}

// IsSupportedServiceAccountValue reports whether the service-account flag value is supported.
func IsSupportedServiceAccountValue(serviceAccount string) bool {
	return serviceAccount == "" || strings.TrimSpace(serviceAccount) != ""
}

// RequiresCommandAfterDash reports whether stdin or tty flags require an exec command.
func RequiresCommandAfterDash(stdin bool, tty bool, dash int) bool {
	return (stdin || tty) && dash == -1
}

// CanRewriteSubPathMounts reports whether subPath rewrite is valid for the selected flags.
func CanRewriteSubPathMounts(copyVolumeMounts bool, rewriteSubPathMounts bool) bool {
	return !rewriteSubPathMounts || copyVolumeMounts
}

// CanUseManifestStdin reports whether stdin can be used for manifest input without
// conflicting with the attach exec session stdin.
func CanUseManifestStdin(filename string, stdin bool) bool {
	return filename != "-" || !stdin
}
