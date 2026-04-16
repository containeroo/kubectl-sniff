package debugpod

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// ToYAML marshals a pod manifest to YAML.
func ToYAML(pod *corev1.Pod) ([]byte, error) {
	return yaml.Marshal(pod)
}

// ToJSON marshals a pod manifest to indented JSON.
func ToJSON(pod *corev1.Pod) ([]byte, error) {
	return json.MarshalIndent(pod, "", "  ")
}
