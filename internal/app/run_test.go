package app

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRunStandaloneDryRunSeparatesManifestAndSummary(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset(testSourcePod())
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runStandalone(
		context.Background(),
		genericiooptions.IOStreams{In: bytes.NewReader(nil), Out: &stdout, ErrOut: &stderr},
		"app",
		RunOptions{
			Image:         "alpine",
			FromContainer: "app",
			CopyEnv:       true,
			DryRun:        true,
			Verbose:       true,
		},
		runtimeDependencies{client: client, namespace: "default"},
	)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "kind: Pod")
	assert.NotContains(t, stdout.String(), "Summary:")
	assert.Contains(t, stderr.String(), "Summary: copied from container \"app\": 1 env entry")

	pods, err := client.CoreV1().Pods("default").List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
}

func TestRunStandaloneCreatesDebugPod(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset(testSourcePod())
	var stdout bytes.Buffer

	err := runStandalone(
		context.Background(),
		genericiooptions.IOStreams{In: bytes.NewReader(nil), Out: &stdout, ErrOut: io.Discard},
		"app",
		RunOptions{Image: "alpine", Name: "app-debug"},
		runtimeDependencies{client: client, namespace: "default"},
	)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Created debug pod default/app-debug")

	created, err := client.CoreV1().Pods("default").Get(context.Background(), "app-debug", metav1.GetOptions{})
	require.NoError(t, err)
	require.Len(t, created.Spec.Containers, 1)
	assert.Equal(t, "alpine", created.Spec.Containers[0].Image)
}

func testSourcePod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "app",
				Env:  []corev1.EnvVar{{Name: "MODE", Value: "test"}},
			}},
		},
	}
}
