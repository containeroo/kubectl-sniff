package kube

import (
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NewClientset builds a Kubernetes clientset and resolves the active namespace.
// If namespaceOverride is empty, the current kubectl namespace is used.
func NewClientset(namespaceOverride string) (*kubernetes.Clientset, string, *rest.Config, error) {
	configFlags := genericclioptions.NewConfigFlags(true)
	if namespaceOverride != "" {
		configFlags.Namespace = &namespaceOverride
	}

	restConfig, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, "", nil, fmt.Errorf("build rest config: %w", err)
	}

	rawKubeConfigLoader := configFlags.ToRawKubeConfigLoader()
	namespace, _, err := rawKubeConfigLoader.Namespace()
	if err != nil {
		return nil, "", nil, fmt.Errorf("resolve namespace: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, "", nil, fmt.Errorf("create clientset: %w", err)
	}

	return clientset, namespace, restConfig, nil
}
