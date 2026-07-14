package debugpod

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildStandalonePodServiceAccountTokenPolicy(t *testing.T) {
	t.Parallel()

	sourcePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app"}},
		},
	}

	tests := []struct {
		name                   string
		opts                   StandaloneOptions
		wantAutomount          bool
		wantServiceAccountName string
	}{
		{
			name:          "disables implicit credentials by default",
			opts:          StandaloneOptions{Image: "alpine"},
			wantAutomount: false,
		},
		{
			name: "enables credentials for an explicit service account",
			opts: StandaloneOptions{
				Image:          "alpine",
				ServiceAccount: "debugger",
			},
			wantAutomount:          true,
			wantServiceAccountName: "debugger",
		},
		{
			name: "enables credentials when service account mounts are requested",
			opts: StandaloneOptions{
				Image:                    "alpine",
				CopyServiceAccountMounts: true,
			},
			wantAutomount: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pod, _, err := BuildStandalonePod(sourcePod, tt.opts)
			require.NoError(t, err)
			require.NotNil(t, pod.Spec.AutomountServiceAccountToken)
			assert.Equal(t, tt.wantAutomount, *pod.Spec.AutomountServiceAccountToken)
			assert.Equal(t, tt.wantServiceAccountName, pod.Spec.ServiceAccountName)
		})
	}
}

func TestBuildStandalonePodFiltersAndDeduplicatesMounts(t *testing.T) {
	t.Parallel()

	sourcePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "app",
				VolumeMounts: []corev1.VolumeMount{
					{Name: "config", MountPath: "/etc/config"},
					{Name: "duplicate", MountPath: "/etc/config"},
					{Name: "token", MountPath: "/var/run/secrets/kubernetes.io/serviceaccount"},
				},
			}},
			Volumes: []corev1.Volume{
				{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{}}},
				{Name: "duplicate", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{
					Name: "token",
					VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{
						Sources: []corev1.VolumeProjection{{ServiceAccountToken: &corev1.ServiceAccountTokenProjection{Path: "token"}}},
					}},
				},
			},
		},
	}

	pod, report, err := BuildStandalonePod(sourcePod, StandaloneOptions{
		Image:            "alpine",
		FromContainer:    "app",
		CopyVolumeMounts: true,
	})
	require.NoError(t, err)
	require.Len(t, pod.Spec.Containers, 1)
	assert.Equal(t, []corev1.VolumeMount{{Name: "config", MountPath: "/etc/config"}}, pod.Spec.Containers[0].VolumeMounts)
	require.Len(t, pod.Spec.Volumes, 1)
	assert.Equal(t, "config", pod.Spec.Volumes[0].Name)
	assert.Equal(t, 1, report.CopiedVolumeMounts)
	assert.Equal(t, 1, report.SkippedServiceAccountMounts)
}
