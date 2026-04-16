package debugpod

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// defaultDebugContainerName is the container name used for standalone debug pods.
const defaultDebugContainerName = "debugger"

// StandaloneOptions describes how the standalone debug pod should be created.
type StandaloneOptions struct {
	// Name is the explicit pod name to create instead of using GenerateName.
	Name string
	// Image is the image used for the standalone debug container.
	Image string
	// FromContainer is the regular container used as a copy source.
	FromContainer string
	// Command overrides the debug container entrypoint.
	Command []string
	// Args appends arguments to the debug container command.
	Args []string
	// Stdin enables stdin for the debug container.
	Stdin bool
	// TTY enables TTY allocation for the debug container.
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
	// Profile applies a predefined security context to the debug container.
	Profile string
}

// BuildStandalonePod returns a standalone debug pod derived from the source pod.
func BuildStandalonePod(sourcePod *corev1.Pod, opts StandaloneOptions) (*corev1.Pod, BuildReport, error) {
	sourceContainer, err := findOptionalRegularContainer(sourcePod, opts.FromContainer, "source")
	if err != nil {
		return nil, BuildReport{}, err
	}

	report := BuildReport{
		Profile: NormalizeProfile(opts.Profile),
	}
	if sourceContainer != nil {
		report.SourceContainer = sourceContainer.Name
	}

	container := buildStandaloneContainer(opts)
	podSpec := corev1.PodSpec{
		RestartPolicy:      corev1.RestartPolicyNever,
		ServiceAccountName: opts.ServiceAccount,
		Containers:         []corev1.Container{container},
	}

	if sourceContainer != nil {
		if opts.CopyEnv {
			podSpec.Containers[0].Env = cloneEnvVars(sourceContainer.Env)
			report.CopiedEnv = len(sourceContainer.Env)
		}
		if opts.CopyEnvFrom {
			podSpec.Containers[0].EnvFrom = cloneEnvFromSources(sourceContainer.EnvFrom)
			report.CopiedEnvFrom = len(sourceContainer.EnvFrom)
		}
		if opts.CopyVolumeMounts {
			mountResult := filterAndCloneVolumeMounts(sourcePod, sourceContainer.VolumeMounts, opts.CopyServiceAccountMounts)
			podSpec.Containers[0].VolumeMounts = mountResult.mounts
			podSpec.Volumes = copyMountedVolumes(sourcePod, mountResult.mounts, opts.CopyServiceAccountMounts)
			report.CopiedVolumeMounts = len(mountResult.mounts)
			report.RewrittenSubPathMounts = mountResult.rewrittenSubPathMounts
			report.SkippedSubPathMounts = mountResult.skippedSubPathMounts
			report.SkippedServiceAccountMounts = mountResult.skippedServiceAccountMount
		}
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: sourcePod.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "sniff",
				"sniff.k8s.io/source-pod":      sourcePod.Name,
			},
			Annotations: map[string]string{
				"sniff.k8s.io/source-namespace": sourcePod.Namespace,
			},
		},
		Spec: podSpec,
	}

	if opts.Name != "" {
		pod.Name = opts.Name
	} else {
		pod.GenerateName = fmt.Sprintf("%s-debug-", sourcePod.Name)
	}

	return pod, report, nil
}

// buildStandaloneContainer returns the standalone debug container spec.
func buildStandaloneContainer(opts StandaloneOptions) corev1.Container {
	command, args := standaloneCommandAndArgs(opts)

	container := corev1.Container{
		Name:    defaultDebugContainerName,
		Image:   opts.Image,
		Command: command,
		Args:    args,
		Stdin:   opts.Stdin,
		TTY:     opts.TTY,
	}

	applyProfileToContainer(&container, opts.Profile)
	return container
}

// standaloneCommandAndArgs returns the default keep-alive command when none is provided.
func standaloneCommandAndArgs(opts StandaloneOptions) ([]string, []string) {
	if len(opts.Command) != 0 || len(opts.Args) != 0 {
		return opts.Command, opts.Args
	}

	return []string{"sleep"}, []string{"infinity"}
}
