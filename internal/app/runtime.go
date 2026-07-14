package app

import (
	"context"
	"io"

	"github.com/containeroo/sniff/internal/kube"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type runtimeDependencies struct {
	client                           kubernetes.Interface
	namespace                        string
	restConfig                       *rest.Config
	waitForEphemeralContainerRunning func(
		context.Context,
		kubernetes.Interface,
		string,
		string,
		string,
	) error
	execInPod func(
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
	) error
}

func newRuntimeDependencies(namespaceOverride string) (runtimeDependencies, error) {
	client, namespace, restConfig, err := kube.NewClientset(namespaceOverride)
	if err != nil {
		return runtimeDependencies{}, err
	}

	return runtimeDependencies{
		client:                           client,
		namespace:                        namespace,
		restConfig:                       restConfig,
		waitForEphemeralContainerRunning: kube.WaitForEphemeralContainerRunning,
		execInPod:                        kube.ExecInPod,
	}, nil
}
