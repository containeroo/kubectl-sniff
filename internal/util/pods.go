package util

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const defaultContainerAnnotation = "kubectl.kubernetes.io/default-container"

// ResolveSourceContainer returns the container to copy settings from.
func ResolveSourceContainer(
	pod *corev1.Pod,
	requested string,
	copyEnv bool,
	copyEnvFrom bool,
	copyVolumeMounts bool,
) (string, error) {
	if !needsSourceContainer(copyEnv, copyEnvFrom, copyVolumeMounts) {
		return strings.TrimSpace(requested), nil
	}

	// If the user provided a container name, use it as the source.
	if trimmed := strings.TrimSpace(requested); trimmed != "" {
		return trimmed, nil
	}

	if defaultContainer := defaultRegularContainer(pod); defaultContainer != "" {
		return defaultContainer, nil
	}

	// If the pod has only one container, use it as the source.
	if len(pod.Spec.Containers) == 1 {
		return pod.Spec.Containers[0].Name, nil
	}

	return "", fmt.Errorf(
		"--from-container is required when using any --copy-* flag because pod %s/%s has %d regular containers",
		pod.Namespace,
		pod.Name,
		len(pod.Spec.Containers),
	)
}

// ResolveTargetContainer returns the default target container when one is obvious.
func ResolveTargetContainer(pod *corev1.Pod, requested string) (string, error) {
	if trimmed := strings.TrimSpace(requested); trimmed != "" {
		return trimmed, nil
	}

	if len(pod.Spec.Containers) == 1 {
		return pod.Spec.Containers[0].Name, nil
	}

	return "", nil
}

// needsSourceContainer reports whether any copy flag requires a source container.
func needsSourceContainer(copyEnv bool, copyEnvFrom bool, copyVolumeMounts bool) bool {
	return copyEnv || copyEnvFrom || copyVolumeMounts
}

// defaultRegularContainer returns the annotated default regular container when it exists.
func defaultRegularContainer(pod *corev1.Pod) string {
	if pod == nil || len(pod.Spec.Containers) == 0 {
		return ""
	}

	name := strings.TrimSpace(pod.Annotations[defaultContainerAnnotation])
	if name == "" {
		return ""
	}

	for _, container := range pod.Spec.Containers {
		if container.Name == name {
			return name
		}
	}

	return ""
}
