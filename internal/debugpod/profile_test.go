package debugpod

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestValidateProfile verifies accepted and rejected debug profile values.
func TestValidateProfile(t *testing.T) {
	t.Parallel()

	t.Run("accepts empty profile", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, ValidateProfile(""))
	})

	t.Run("accepts general profile", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, ValidateProfile("general"))
	})

	t.Run("accepts mixed-case general profile", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, ValidateProfile("GENERAL"))
	})

	t.Run("accepts trimmed netadmin profile", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, ValidateProfile(" netadmin "))
	})

	t.Run("rejects unsupported profile", func(t *testing.T) {
		t.Parallel()
		require.Error(t, ValidateProfile("nope"))
	})
}

// TestBuildUpdatedPodAppliesProfileAndReport verifies profile application and reporting.
func TestBuildUpdatedPodAppliesProfileAndReport(t *testing.T) {
	t.Parallel()

	sourcePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mypod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
					Env: []corev1.EnvVar{
						{Name: "A", Value: "1"},
						{Name: "B", Value: "2"},
					},
				},
			},
		},
	}

	updated, report, err := BuildUpdatedPod(sourcePod, EphemeralOptions{
		Image:         "alpine",
		ContainerName: "debugger",
		FromContainer: "app",
		CopyEnv:       true,
		Profile:       ProfileNetAdmin,
	})
	require.NoError(t, err)

	require.Len(t, updated.Spec.EphemeralContainers, 1)

	container := updated.Spec.EphemeralContainers[0]
	require.NotNil(t, container.SecurityContext)
	require.NotNil(t, container.SecurityContext.Capabilities)
	assert.Equal(t, []corev1.Capability{"NET_ADMIN", "NET_RAW"}, container.SecurityContext.Capabilities.Add)
	assert.Equal(t, ProfileNetAdmin, report.Profile)
	assert.Equal(t, "app", report.SourceContainer)
	assert.Equal(t, 2, report.CopiedEnv)
}
