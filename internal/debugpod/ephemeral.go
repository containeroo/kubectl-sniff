package debugpod

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// EphemeralOptions describes how the ephemeral debug container should be created.
type EphemeralOptions struct {
	// Image is the image used for the ephemeral debug container.
	Image string
	// ContainerName is the name assigned to the ephemeral container.
	ContainerName string
	// TargetContainer is the regular container whose namespaces should be targeted.
	TargetContainer string
	// FromContainer is the regular container used as a copy source.
	FromContainer string
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
	// Profile applies a predefined security context to the debug container.
	Profile string
}

// BuildUpdatedPod returns a deep-copied pod with one additional ephemeral container.
func BuildUpdatedPod(pod *corev1.Pod, opts EphemeralOptions) (*corev1.Pod, BuildReport, error) {
	if err := validateContainerNameUniqueness(pod, opts.ContainerName); err != nil {
		return nil, BuildReport{}, err
	}

	if opts.TargetContainer != "" {
		if _, err := findRegularContainer(pod, opts.TargetContainer); err != nil {
			return nil, BuildReport{}, fmt.Errorf("target container validation failed: %w", err)
		}
	}

	sourceContainer, err := findOptionalRegularContainer(pod, opts.FromContainer, "source")
	if err != nil {
		return nil, BuildReport{}, err
	}

	report := BuildReport{
		Profile: NormalizeProfile(opts.Profile),
	}
	if sourceContainer != nil {
		report.SourceContainer = sourceContainer.Name
	}

	ec := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:  opts.ContainerName,
			Image: opts.Image,
		},
		TargetContainerName: opts.TargetContainer,
	}
	applyProfileToEphemeralContainer(&ec, opts.Profile)

	if sourceContainer != nil {
		if opts.CopyEnv {
			ec.Env = cloneEnvVars(sourceContainer.Env)
			report.CopiedEnv = len(sourceContainer.Env)
		}
		if opts.CopyEnvFrom {
			ec.EnvFrom = cloneEnvFromSources(sourceContainer.EnvFrom)
			report.CopiedEnvFrom = len(sourceContainer.EnvFrom)
		}
		if opts.CopyVolumeMounts {
			mountResult := filterAndCloneEphemeralVolumeMounts(
				pod,
				sourceContainer.VolumeMounts,
				opts.CopyServiceAccountMounts,
				opts.RewriteSubPathMounts,
			)
			ec.VolumeMounts = mountResult.mounts
			report.CopiedVolumeMounts = len(mountResult.mounts)
			report.RewrittenSubPathMounts = mountResult.rewrittenSubPathMounts
			report.SkippedSubPathMounts = mountResult.skippedSubPathMounts
			report.SkippedServiceAccountMounts = mountResult.skippedServiceAccountMount
		}
	}

	updated := pod.DeepCopy()
	updated.Spec.EphemeralContainers = append(updated.Spec.EphemeralContainers, ec)
	return updated, report, nil
}

// validateContainerNameUniqueness ensures the new container name does not already exist.
func validateContainerNameUniqueness(pod *corev1.Pod, name string) error {
	for _, container := range pod.Spec.InitContainers {
		if container.Name == name {
			return fmt.Errorf("container name %q already exists as an init container", name)
		}
	}

	for _, container := range pod.Spec.Containers {
		if container.Name == name {
			return fmt.Errorf("container name %q already exists as a regular container", name)
		}
	}

	for _, container := range pod.Spec.EphemeralContainers {
		if container.Name == name {
			return fmt.Errorf("container name %q already exists as an ephemeral container", name)
		}
	}

	return nil
}
