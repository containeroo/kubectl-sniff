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
