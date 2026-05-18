package kube

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateEphemeralContainerStatus(t *testing.T) {
	t.Parallel()

	t.Run("returns done when container is running", func(t *testing.T) {
		t.Parallel()

		done, err := evaluateEphemeralContainerStatus([]corev1.ContainerStatus{
			{
				Name: "debugger",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}, "debugger")

		require.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("returns error when container terminated", func(t *testing.T) {
		t.Parallel()

		done, err := evaluateEphemeralContainerStatus([]corev1.ContainerStatus{
			{
				Name: "debugger",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{ExitCode: 42},
				},
			},
		}, "debugger")

		require.Error(t, err)
		assert.False(t, done)
		assert.ErrorContains(t, err, `ephemeral container "debugger" terminated with exit code 42`)
	})

	t.Run("returns error for fatal waiting reasons", func(t *testing.T) {
		t.Parallel()

		done, err := evaluateEphemeralContainerStatus([]corev1.ContainerStatus{
			{
				Name: "debugger",
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason:  "ImagePullBackOff",
						Message: "Back-off pulling image",
					},
				},
			},
		}, "debugger")

		require.Error(t, err)
		assert.False(t, done)
		assert.ErrorContains(t, err, `ephemeral container "debugger" is waiting with reason "ImagePullBackOff": Back-off pulling image`)
	})

	t.Run("keeps polling for non-fatal waiting reasons", func(t *testing.T) {
		t.Parallel()

		done, err := evaluateEphemeralContainerStatus([]corev1.ContainerStatus{
			{
				Name: "debugger",
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason:  "ContainerCreating",
						Message: "still starting",
					},
				},
			},
		}, "debugger")

		require.NoError(t, err)
		assert.False(t, done)
	})
}
