package kube

import (
	"context"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// ephemeralContainerPollInterval is the cadence used while waiting for runtime status.
const ephemeralContainerPollInterval = 500 * time.Millisecond

// WaitForEphemeralContainerRunning waits until the named ephemeral container is running.
func WaitForEphemeralContainerRunning(
	ctx context.Context,
	clientset *kubernetes.Clientset,
	namespace, podName, containerName string,
) error {
	ticker := time.NewTicker(ephemeralContainerPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// The pod status is updated asynchronously after the ephemeral container
			// is added, so poll until the runtime reports it as running.
			pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			for _, status := range pod.Status.EphemeralContainerStatuses {
				if status.Name != containerName {
					continue
				}

				if status.State.Running != nil {
					return nil
				}

				if status.State.Terminated != nil {
					return fmt.Errorf("ephemeral container %q terminated with exit code %d", containerName, status.State.Terminated.ExitCode)
				}
			}
		}
	}
}

// ExecInPod streams an exec session into the requested container.
func ExecInPod(
	ctx context.Context,
	restConfig *rest.Config,
	clientset *kubernetes.Clientset,
	namespace, podName, containerName string,
	command []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	enableStdin, tty bool,
) error {
	req := clientset.CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("exec")

	req.VersionedParams(&corev1.PodExecOptions{
		Container: containerName,
		Command:   command,
		Stdin:     enableStdin,
		Stdout:    stdout != nil,
		Stderr:    !tty && stderr != nil,
		TTY:       tty,
	}, kubescheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
	if err != nil {
		return err
	}

	return executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}
