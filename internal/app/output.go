package app

import (
	"fmt"

	"github.com/containeroo/sniff/internal/debugpod"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

// writeDryRunManifest prints the generated pod manifest in the requested format.
func writeDryRunManifest(streams genericiooptions.IOStreams, pod *corev1.Pod, output string) error {
	switch output {
	case "", "yaml":
		out, err := debugpod.ToYAML(pod)
		if err != nil {
			return fmt.Errorf("marshal pod to yaml: %w", err)
		}

		_, err = fmt.Fprintln(streams.Out, string(out))
		return err
	case "json":
		out, err := debugpod.ToJSON(pod)
		if err != nil {
			return fmt.Errorf("marshal pod to json: %w", err)
		}

		_, err = fmt.Fprintln(streams.Out, string(out))
		return err
	default:
		return fmt.Errorf(`unsupported output format %q (supported: "yaml", "json")`, output)
	}
}
