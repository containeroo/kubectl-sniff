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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestRunAttachQuietStillExecutesCommand(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app"}},
		},
	})
	var stdout bytes.Buffer
	waitCalled := false
	execCalled := false

	deps := runtimeDependencies{
		client:     client,
		namespace:  "default",
		restConfig: &rest.Config{},
		waitForEphemeralContainerRunning: func(
			context.Context,
			kubernetes.Interface,
			string,
			string,
			string,
		) error {
			waitCalled = true
			return nil
		},
		execInPod: func(
			context.Context,
			*rest.Config,
			kubernetes.Interface,
			string,
			string,
			string,
			[]string,
			io.Reader,
			io.Writer,
			io.Writer,
			bool,
			bool,
		) error {
			execCalled = true
			return nil
		},
	}

	err := runAttach(
		context.Background(),
		genericiooptions.IOStreams{In: bytes.NewReader(nil), Out: &stdout, ErrOut: io.Discard},
		"app",
		AttachOptions{
			Image:         "alpine",
			ContainerName: "debugger",
			ExecCommand:   []string{"sh"},
			Quiet:         true,
		},
		deps,
	)
	require.NoError(t, err)
	assert.True(t, waitCalled)
	assert.True(t, execCalled)
	assert.Empty(t, stdout.String())
}
