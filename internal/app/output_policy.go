package app

import "strings"

// shouldPrintVerboseDetails reports whether verbose informational output should be shown.
func shouldPrintVerboseDetails(quiet bool, verbose bool) bool {
	return verbose && !quiet
}

// shouldPrintGeneratedContainerName reports whether the generated container name hint should be shown.
func shouldPrintGeneratedContainerName(quiet bool, verbose bool, requestedContainerName string) bool {
	return shouldPrintVerboseDetails(quiet, verbose) && strings.TrimSpace(requestedContainerName) == ""
}
