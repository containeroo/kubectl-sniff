package util

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveSourceContainer verifies source-container resolution and ambiguity handling.
func TestResolveSourceContainer(t *testing.T) {
	t.Parallel()

	t.Run("returns requested container", func(t *testing.T) {
		t.Parallel()
		pod := newTestPod(
			[]string{"app", "sidecar"},
			nil,
		)

		got, err := ResolveSourceContainer(pod, "sidecar", true, false, false)
		require.NoError(t, err)
		assert.Equal(t, "sidecar", got)
	})

	t.Run("returns requested container when no source is needed", func(t *testing.T) {
		t.Parallel()
		pod := newTestPod(
			[]string{"app", "sidecar"},
			map[string]string{defaultContainerAnnotation: "app"},
		)

		got, err := ResolveSourceContainer(pod, "sidecar", false, false, false)
		require.NoError(t, err)
		assert.Equal(t, "sidecar", got)
	})

	t.Run("uses default-container annotation when present", func(t *testing.T) {
		t.Parallel()
		pod := newTestPod(
			[]string{"app", "sidecar"},
			map[string]string{defaultContainerAnnotation: "sidecar"},
		)

		got, err := ResolveSourceContainer(pod, "", true, false, false)
		require.NoError(t, err)
		assert.Equal(t, "sidecar", got)
	})

	t.Run("ignores default-container annotation if not a regular container", func(t *testing.T) {
		t.Parallel()
		pod := newTestPod(
			[]string{"app", "sidecar"},
			map[string]string{defaultContainerAnnotation: "missing"},
		)

		_, err := ResolveSourceContainer(pod, "", true, false, false)
		require.Error(t, err)
		assert.ErrorContains(t, err, "--from-container is required")
	})

	t.Run("uses the only regular container", func(t *testing.T) {
		t.Parallel()
		pod := newTestPod(
			[]string{"app"},
			nil,
		)

		got, err := ResolveSourceContainer(pod, "", true, false, false)
		require.NoError(t, err)
		assert.Equal(t, "app", got)
	})

	t.Run("fails when multiple regular containers are ambiguous", func(t *testing.T) {
		t.Parallel()
		pod := newTestPod(
			[]string{"app", "sidecar"},
			nil,
		)

		_, err := ResolveSourceContainer(pod, "", true, true, true)
		require.Error(t, err)
		assert.ErrorContains(t, err, "--from-container is required")
	})
}

// newTestPod builds a pod fixture with the provided container names and annotations.
func newTestPod(containerNames []string, annotations map[string]string) *corev1.Pod {
	containers := make([]corev1.Container, 0, len(containerNames))
	for _, name := range containerNames {
		containers = append(containers, corev1.Container{Name: name})
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "mypod",
			Namespace:   "default",
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: containers,
		},
	}
}
