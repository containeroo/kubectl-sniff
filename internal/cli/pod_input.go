package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

var (
	podManifestScheme = runtime.NewScheme()
	podManifestCodecs = serializer.NewCodecFactory(podManifestScheme)
)

func init() {
	if err := corev1.AddToScheme(podManifestScheme); err != nil {
		panic(err)
	}
}

// ValidateSinglePodSource ensures the command receives exactly one pod source:
// either a positional pod name or a manifest file via -f/--filename.
func ValidateSinglePodSource(filename string, podArgs int) error {
	switch {
	case filename == "" && podArgs != 1:
		return errors.New("exactly one pod name must be provided")
	case filename != "" && podArgs != 0:
		return errors.New("a pod name cannot be provided together with -f/--filename")
	default:
		return nil
	}
}

// ResolvePodSource returns the pod name and namespace selected by the caller.
// The namespace override wins over the manifest namespace when both are set.
func ResolvePodSource(
	args []string,
	dash int,
	filename string,
	namespaceOverride string,
	stdin io.Reader,
) (string, string, error) {
	podArgs := args
	if dash != -1 {
		podArgs = args[:dash]
	}

	if filename == "" {
		return podArgs[0], namespaceOverride, nil
	}

	pod, err := loadPodFromFileOrStdin(filename, stdin)
	if err != nil {
		return "", "", err
	}

	namespace := namespaceOverride
	if namespace == "" {
		namespace = pod.Namespace
	}

	return pod.Name, namespace, nil
}

// loadPodFromFileOrStdin loads a single Pod manifest from a file path or stdin.
func loadPodFromFileOrStdin(filename string, stdin io.Reader) (*corev1.Pod, error) {
	reader := stdin
	if filename != "-" {
		file, err := os.Open(filename)
		if err != nil {
			return nil, fmt.Errorf("open pod manifest %q: %w", filename, err)
		}
		defer file.Close() // nolint:errcheck
		reader = file
	}

	pod, err := decodeSinglePod(reader)
	if err != nil {
		if filename == "-" {
			return nil, fmt.Errorf("read pod manifest from stdin: %w", err)
		}
		return nil, fmt.Errorf("read pod manifest from %q: %w", filename, err)
	}

	if strings.TrimSpace(pod.Name) == "" {
		return nil, errors.New("pod manifest metadata.name is required")
	}

	return pod, nil
}

func decodeSinglePod(reader io.Reader) (*corev1.Pod, error) {
	decoder := yamlutil.NewYAMLOrJSONDecoder(reader, 4096)

	raw, err := decodeNextDocument(decoder)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, errors.New("input did not contain a Pod manifest")
	}

	obj, gvk, err := podManifestCodecs.UniversalDeserializer().Decode(raw, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("manifest must be a Pod, got %s", gvk.String())
	}

	if extra, err := decodeNextDocument(decoder); err != nil {
		return nil, err
	} else if len(extra) > 0 {
		return nil, errors.New("manifest input must contain exactly one Pod document")
	}

	return pod, nil
}

func decodeNextDocument(decoder *yamlutil.YAMLOrJSONDecoder) ([]byte, error) {
	for {
		var raw runtime.RawExtension
		if err := decoder.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				return nil, nil
			}
			return nil, err
		}

		if len(bytes.TrimSpace(raw.Raw)) == 0 {
			continue
		}

		return raw.Raw, nil
	}
}
