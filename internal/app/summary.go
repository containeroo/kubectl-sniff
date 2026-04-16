package app

import (
	"fmt"
	"strings"

	"github.com/containeroo/sniff/internal/debugpod"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

// writeBuildSummary prints a human-friendly summary of the generated debug spec.
func writeBuildSummary(streams genericiooptions.IOStreams, report debugpod.BuildReport, dryRun bool) {
	if !report.HasDetails() {
		return
	}

	writer := streams.Out
	if dryRun {
		writer = streams.ErrOut
	}

	fmt.Fprintf(writer, "Summary: %s\n", summarizeBuildReport(report)) // nolint:errcheck
}

// writeAttachShellHint prints the command to open a shell in the new ephemeral container.
func writeAttachShellHint(streams genericiooptions.IOStreams, namespace string, podName string, containerName string) {
	fmt.Fprintf(streams.Out, "Open a shell with:\n")                                                              // nolint:errcheck
	fmt.Fprintf(streams.Out, "  kubectl exec -it -n %s %s -c %s -- /bin/sh\n", namespace, podName, containerName) // nolint:errcheck
}

// writeStandaloneShellHint prints the command to open a shell in the standalone debug pod.
func writeStandaloneShellHint(streams genericiooptions.IOStreams, namespace string, podName string) {
	fmt.Fprintf(streams.Out, "Open a shell with:\n")                                         // nolint:errcheck
	fmt.Fprintf(streams.Out, "  kubectl exec -it -n %s %s -- /bin/sh\n", namespace, podName) // nolint:errcheck
}

// summarizeBuildReport renders the build report as a single readable summary line.
func summarizeBuildReport(report debugpod.BuildReport) string {
	parts := make([]string, 0, 8)
	copyParts := make([]string, 0, 4)

	if report.Profile != "" {
		parts = append(parts, fmt.Sprintf("applied %q profile", report.Profile))
	}

	if report.CopiedEnv != 0 {
		copyParts = append(copyParts, fmt.Sprintf("%d env %s", report.CopiedEnv, pluralize("entry", report.CopiedEnv)))
	}
	if report.CopiedEnvFrom != 0 {
		copyParts = append(copyParts, fmt.Sprintf("%d envFrom %s", report.CopiedEnvFrom, pluralize("source", report.CopiedEnvFrom)))
	}
	if report.CopiedVolumeMounts != 0 {
		copyParts = append(copyParts, fmt.Sprintf("%d volume %s", report.CopiedVolumeMounts, pluralize("mount", report.CopiedVolumeMounts)))
	}
	if len(copyParts) != 0 {
		if report.SourceContainer != "" {
			parts = append(parts, fmt.Sprintf("copied from container %q: %s", report.SourceContainer, strings.Join(copyParts, ", ")))
		} else {
			parts = append(parts, fmt.Sprintf("copied %s", strings.Join(copyParts, ", ")))
		}
	}

	appendCountPart(&parts, report.RewrittenSubPathMounts, "rewrote", "subPath mount")
	appendCountPart(&parts, report.SkippedSubPathMounts, "skipped", "subPath mount")
	appendCountPart(&parts, report.SkippedServiceAccountMounts, "skipped", "service account mount")

	return strings.Join(parts, "; ")
}

// appendCountPart appends a counted summary fragment when the count is non-zero.
func appendCountPart(parts *[]string, count int, verb string, noun string) {
	if count == 0 {
		return
	}

	*parts = append(*parts, fmt.Sprintf("%s %d %s", verb, count, pluralize(noun, count)))
}

// pluralize returns the noun in singular or plural form for the given count.
func pluralize(noun string, count int) string {
	if count == 1 {
		return noun
	}
	if stem, ok := strings.CutSuffix(noun, "y"); ok {
		return stem + "ies"
	}

	return noun + "s"
}
