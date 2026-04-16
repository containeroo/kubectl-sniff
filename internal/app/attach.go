package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/containeroo/sniff/internal/debugpod"
	"github.com/containeroo/sniff/internal/kube"
	"github.com/containeroo/sniff/internal/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

const ephemeralContainerNamePrefix = "sniff-"

// AttachOptions configures the ephemeral debug container workflow.
type AttachOptions struct {
	// Namespace overrides the active kubectl namespace.
	Namespace string
	// Image is the container image for the ephemeral debugger.
	Image string
	// ContainerName is the name assigned to the new ephemeral container.
	ContainerName string
	// Target is the regular container whose namespaces should be targeted.
	Target string
	// FromContainer is the regular container used as a copy source.
	FromContainer string
	// ExecCommand runs inside the created ephemeral container after it starts.
	ExecCommand []string
	// Stdin enables stdin for the post-create exec session.
	Stdin bool
	// TTY enables TTY allocation for the post-create exec session.
	TTY bool
	// CopyEnv copies env entries from the source container.
	CopyEnv bool
	// CopyEnvFrom copies envFrom entries from the source container.
	CopyEnvFrom bool
	// CopyVolumeMounts copies volume mounts from the source container.
	CopyVolumeMounts bool
	// CopyServiceAccountMounts includes service account token mounts when copying volumes.
	CopyServiceAccountMounts bool
	// RewriteSubPathMounts rewrites subPath mounts into debug-friendly direct mounts.
	RewriteSubPathMounts bool
	// DryRun prints the updated manifest instead of patching the pod.
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

// RunAttach executes the attach flow against an existing pod.
func RunAttach(ctx context.Context, streams genericiooptions.IOStreams, podName string, opts AttachOptions) error {
	clientset, namespace, restConfig, err := kube.NewClientset(opts.Namespace)
	if err != nil {
		return fmt.Errorf("build kubernetes client: %w", err)
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get pod %q in namespace %q: %w", podName, namespace, err)
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

	target, err := util.ResolveTargetContainer(pod, opts.Target)
	if err != nil {
		return err
	}

	containerName := opts.ContainerName
	if strings.TrimSpace(containerName) == "" {
		containerName = defaultEphemeralContainerName()
	}

	buildOpts := debugpod.EphemeralOptions{
		Image:                    opts.Image,
		ContainerName:            containerName,
		TargetContainer:          target,
		FromContainer:            fromContainer,
		CopyEnv:                  opts.CopyEnv,
		CopyEnvFrom:              opts.CopyEnvFrom,
		CopyVolumeMounts:         opts.CopyVolumeMounts,
		CopyServiceAccountMounts: opts.CopyServiceAccountMounts,
		RewriteSubPathMounts:     opts.RewriteSubPathMounts,
		Profile:                  opts.Profile,
	}

	updatedPod, report, err := debugpod.BuildUpdatedPod(pod, buildOpts)
	if err != nil {
		return fmt.Errorf("build updated pod: %w", err)
	}

	if opts.DryRun {
		if shouldPrintVerboseDetails(opts.Quiet, opts.Verbose) {
			writeBuildSummary(streams, report, true)
		}
		return writeDryRunManifest(streams, updatedPod, opts.Output)
	}

	if shouldPrintGeneratedContainerName(opts.Quiet, opts.Verbose, opts.ContainerName) {
		fmt.Fprintf(streams.Out, "Defaulting debug container name to %q\n", containerName) // nolint:errcheck
	}

	created, err := clientset.CoreV1().Pods(namespace).UpdateEphemeralContainers(ctx, pod.Name, updatedPod, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update ephemeral containers for pod %q: %w", pod.Name, err)
	}

	if opts.Quiet {
		return nil
	}

	fmt.Fprintf(streams.Out, "Added ephemeral container %q to pod %s/%s\n", containerName, created.Namespace, created.Name) // nolint:errcheck
	if shouldPrintVerboseDetails(opts.Quiet, opts.Verbose) {
		writeBuildSummary(streams, report, false)
	}
	writeAttachShellHint(streams, created.Namespace, created.Name, containerName)

	if len(opts.ExecCommand) == 0 {
		return nil
	}

	if err := kube.WaitForEphemeralContainerRunning(ctx, clientset, namespace, pod.Name, containerName); err != nil {
		return fmt.Errorf("wait for ephemeral container %q to be running: %w", containerName, err)
	}

	return kube.ExecInPod(
		ctx,
		restConfig,
		clientset,
		namespace,
		pod.Name,
		containerName,
		opts.ExecCommand,
		streams.In,
		streams.Out,
		streams.ErrOut,
		opts.Stdin,
		opts.TTY,
	)
}

// defaultEphemeralContainerName returns a kubectl-debug-style generated container name.
func defaultEphemeralContainerName() string {
	return ephemeralContainerNamePrefix + utilrand.String(5)
}
