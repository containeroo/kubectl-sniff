package debugpod

import (
	"fmt"
	"path"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// rewrittenMountRoot is the base mount path used for rewritten subPath mounts.
const rewrittenMountRoot = "/mnt/sniff/volumes"

// findRegularContainer returns a regular container by name.
func findRegularContainer(pod *corev1.Pod, name string) (*corev1.Container, error) {
	for index := range pod.Spec.Containers {
		container := &pod.Spec.Containers[index]
		if container.Name == name {
			return container, nil
		}
	}

	return nil, fmt.Errorf("regular container %q not found in pod %s/%s", name, pod.Namespace, pod.Name)
}

// findOptionalRegularContainer resolves the named regular container when present.
func findOptionalRegularContainer(pod *corev1.Pod, name string, role string) (*corev1.Container, error) {
	if strings.TrimSpace(name) == "" {
		return nil, nil
	}

	container, err := findRegularContainer(pod, name)
	if err != nil {
		return nil, fmt.Errorf("%s container validation failed: %w", role, err)
	}

	return container, nil
}

// cloneEnvVars copies env entries so the source pod is never mutated.
func cloneEnvVars(envs []corev1.EnvVar) []corev1.EnvVar {
	if len(envs) == 0 {
		return nil
	}

	cloned := make([]corev1.EnvVar, len(envs))
	copy(cloned, envs)
	return cloned
}

// cloneEnvFromSources copies envFrom entries so the source pod is never mutated.
func cloneEnvFromSources(envFrom []corev1.EnvFromSource) []corev1.EnvFromSource {
	if len(envFrom) == 0 {
		return nil
	}

	cloned := make([]corev1.EnvFromSource, len(envFrom))
	copy(cloned, envFrom)
	return cloned
}

type volumeMountCopyResult struct {
	mounts                     []corev1.VolumeMount
	rewrittenSubPathMounts     int
	skippedSubPathMounts       int
	skippedServiceAccountMount int
}

// filterAndCloneVolumeMounts copies supported regular container mounts.
func filterAndCloneVolumeMounts(pod *corev1.Pod, mounts []corev1.VolumeMount, includeServiceAccountMounts bool) volumeMountCopyResult {
	if len(mounts) == 0 {
		return volumeMountCopyResult{}
	}

	volumeIndex := make(map[string]corev1.Volume, len(pod.Spec.Volumes))
	for _, volume := range pod.Spec.Volumes {
		volumeIndex[volume.Name] = volume
	}

	cloned := make([]corev1.VolumeMount, 0, len(mounts))
	result := volumeMountCopyResult{}
	seenMountPaths := make(map[string]struct{}, len(mounts))

	for _, mount := range mounts {
		if _, exists := seenMountPaths[mount.MountPath]; exists {
			continue
		}

		volume, hasVolume := volumeIndex[mount.Name]
		if hasVolume && isServiceAccountVolume(volume, mount.MountPath) && !includeServiceAccountMounts {
			result.skippedServiceAccountMount++
			continue
		}

		cloned = append(cloned, mount)
		seenMountPaths[mount.MountPath] = struct{}{}
	}

	result.mounts = cloned
	return result
}

// filterAndCloneEphemeralVolumeMounts copies supported ephemeral container mounts.
func filterAndCloneEphemeralVolumeMounts(
	pod *corev1.Pod,
	mounts []corev1.VolumeMount,
	includeServiceAccountMounts bool,
	rewriteSubPathMounts bool,
) volumeMountCopyResult {
	if len(mounts) == 0 {
		return volumeMountCopyResult{}
	}

	volumeIndex := make(map[string]corev1.Volume, len(pod.Spec.Volumes))
	for _, volume := range pod.Spec.Volumes {
		volumeIndex[volume.Name] = volume
	}

	cloned := make([]corev1.VolumeMount, 0, len(mounts))
	result := volumeMountCopyResult{}
	seenMountPaths := make(map[string]struct{}, len(mounts))

	for _, mount := range mounts {
		volume, hasVolume := volumeIndex[mount.Name]
		if hasVolume && isServiceAccountVolume(volume, mount.MountPath) && !includeServiceAccountMounts {
			result.skippedServiceAccountMount++
			continue
		}

		rewritten := mount
		if mount.SubPath != "" || mount.SubPathExpr != "" {
			if !rewriteSubPathMounts {
				result.skippedSubPathMounts++
				continue
			}

			// Rewriting keeps the full parent mount visible when the original container
			// relied on subPath-specific filesystem layout.
			rewritten.MountPath = rewrittenSubPathMountPath(mount)
			rewritten.SubPath = ""
			rewritten.SubPathExpr = ""
			result.rewrittenSubPathMounts++
		}

		if _, exists := seenMountPaths[rewritten.MountPath]; exists {
			continue
		}

		cloned = append(cloned, rewritten)
		seenMountPaths[rewritten.MountPath] = struct{}{}
	}

	result.mounts = cloned
	return result
}

// rewrittenSubPathMountPath returns the mount path used for rewritten subPath mounts.
func rewrittenSubPathMountPath(mount corev1.VolumeMount) string {
	base := path.Join(rewrittenMountRoot, mount.Name)

	switch {
	case mount.SubPath != "":
		return path.Join(base, mount.SubPath)
	case mount.SubPathExpr != "":
		return path.Join(base, sanitizePathForMount(mount.SubPathExpr))
	default:
		return base
	}
}

// sanitizePathForMount removes path separators from subPathExpr-derived directories.
func sanitizePathForMount(value string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"$", "_",
		"{", "_",
		"}", "_",
		"(", "_",
		")", "_",
	)
	return replacer.Replace(value)
}

// copyMountedVolumes copies the volume definitions referenced by the selected mounts.
func copyMountedVolumes(pod *corev1.Pod, mounts []corev1.VolumeMount, includeServiceAccountMounts bool) []corev1.Volume {
	if len(mounts) == 0 {
		return nil
	}

	needed := make(map[string]struct{}, len(mounts))
	for _, mount := range mounts {
		needed[mount.Name] = struct{}{}
	}

	copied := make([]corev1.Volume, 0, len(needed))
	for _, volume := range pod.Spec.Volumes {
		if _, ok := needed[volume.Name]; !ok {
			continue
		}
		if isServiceAccountVolume(volume, "") && !includeServiceAccountMounts {
			continue
		}
		copied = append(copied, volume)
	}

	if len(copied) == 0 {
		return nil
	}

	return copied
}

// isServiceAccountVolume reports whether the mount is derived from service account tokens.
func isServiceAccountVolume(volume corev1.Volume, mountPath string) bool {
	if strings.HasPrefix(mountPath, "/var/run/secrets/kubernetes.io/serviceaccount") {
		return true
	}
	if volume.Projected != nil {
		for _, source := range volume.Projected.Sources {
			if source.ServiceAccountToken != nil {
				return true
			}
		}
	}
	if volume.Secret != nil && strings.HasPrefix(volume.Secret.SecretName, "default-token-") {
		return true
	}
	return false
}
