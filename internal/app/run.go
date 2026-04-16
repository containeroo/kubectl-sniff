package app

import (
	"context"
	"fmt"

	"github.com/containeroo/sniff/internal/debugpod"
	"github.com/containeroo/sniff/internal/kube"
	"github.com/containeroo/sniff/internal/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

// RunOptions configures the standalone debug pod workflow.
type RunOptions struct {
	// Namespace overrides the active kubectl namespace.
	Namespace string
	// Image is the container image for the standalone debug pod.
	Image string
	// Name is the explicit pod name to create instead of using GenerateName.
	Name string
	// FromContainer is the regular container used as a copy source.
	FromContainer string
	// Command overrides the debug container entrypoint.
	Command []string
	// Args appends arguments to the debug container command.
	Args []string
	// Stdin enables stdin for the standalone debug container.
	Stdin bool
	// TTY enables TTY allocation for the standalone debug container.
	TTY bool
	// CopyEnv copies env entries from the source container.
	CopyEnv bool
	// CopyEnvFrom copies envFrom entries from the source container.
	CopyEnvFrom bool
	// CopyVolumeMounts copies volume mounts from the source container.
	CopyVolumeMounts bool
	// CopyServiceAccountMounts includes service account token mounts when copying volumes.
	CopyServiceAccountMounts bool
	// ServiceAccount sets the service account on the created debug pod.
	ServiceAccount string
	// DryRun prints the generated manifest instead of creating the pod.
	DryRun bool
	// Output selects the dry-run output format.
	Output string
	// Quiet suppresses non-error informational output.
	Quiet bool
	// Verbose enables detailed informational output.
	Verbose bool
	// Profile applies a predefined security context to the debug container.
	Profile string
}

// RunStandalone creates a standalone debug pod derived from an existing pod.
func RunStandalone(ctx context.Context, streams genericiooptions.IOStreams, sourcePodName string, opts RunOptions) error {
	clientset, namespace, _, err := kube.NewClientset(opts.Namespace)
	if err != nil {
		return fmt.Errorf("build kubernetes client: %w", err)
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, sourcePodName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get pod %q in namespace %q: %w", sourcePodName, namespace, err)
	}

	fromContainer, err := util.ResolveSourceContainer(
		pod,
		opts.FromContainer,
		opts.CopyEnv,
		opts.CopyEnvFrom,
		opts.CopyVolumeMounts,
	)
	if err != nil {
		return err
	}

	serviceAccount := opts.ServiceAccount
	if serviceAccount == "from-pod" {
		serviceAccount = pod.Spec.ServiceAccountName
	}

	buildOpts := debugpod.StandaloneOptions{
		Name:                     opts.Name,
		Image:                    opts.Image,
		FromContainer:            fromContainer,
		Command:                  opts.Command,
		Args:                     opts.Args,
		Stdin:                    opts.Stdin,
		TTY:                      opts.TTY,
		CopyEnv:                  opts.CopyEnv,
		CopyEnvFrom:              opts.CopyEnvFrom,
		CopyVolumeMounts:         opts.CopyVolumeMounts,
		CopyServiceAccountMounts: opts.CopyServiceAccountMounts,
		ServiceAccount:           serviceAccount,
		Profile:                  opts.Profile,
	}

	debugPod, report, err := debugpod.BuildStandalonePod(pod, buildOpts)
	if err != nil {
		return fmt.Errorf("build standalone pod: %w", err)
	}

	if opts.DryRun {
		if shouldPrintVerboseDetails(opts.Quiet, opts.Verbose) {
			writeBuildSummary(streams, report, true)
		}
		return writeDryRunManifest(streams, debugPod, opts.Output)
	}

	created, err := clientset.CoreV1().Pods(namespace).Create(ctx, debugPod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create debug pod: %w", err)
	}

	if opts.Quiet {
		return nil
	}

	fmt.Fprintf(streams.Out, "Created debug pod %s/%s\n", created.Namespace, created.Name) // nolint:errcheck
	if shouldPrintVerboseDetails(opts.Quiet, opts.Verbose) {
		writeBuildSummary(streams, report, false)
	}
	writeStandaloneShellHint(streams, created.Namespace, created.Name)

	return nil
}
