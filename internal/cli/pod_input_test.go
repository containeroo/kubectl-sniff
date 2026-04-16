package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPodManifest = `
apiVersion: v1
kind: Pod
metadata:
  name: demo-pod
  namespace: demo-ns
`

// TestValidateSinglePodSource verifies positional and filename-based pod selection.
func TestValidateSinglePodSource(t *testing.T) {
	t.Parallel()

	t.Run("positional pod", func(t *testing.T) {
		t.Parallel()

		err := ValidateSinglePodSource("", 1)
		require.NoError(t, err)
	})

	t.Run("filename pod", func(t *testing.T) {
		t.Parallel()

		err := ValidateSinglePodSource("pod.yaml", 0)
		require.NoError(t, err)
	})

	t.Run("missing source", func(t *testing.T) {
		t.Parallel()

		err := ValidateSinglePodSource("", 0)
		require.Error(t, err)
		assert.ErrorContains(t, err, "exactly one pod name must be provided")
	})

	t.Run("both provided", func(t *testing.T) {
		t.Parallel()

		err := ValidateSinglePodSource("pod.yaml", 1)
		require.Error(t, err)
		assert.ErrorContains(t, err, "a pod name cannot be provided together with -f/--filename")
	})
}

// TestValidationHelpers verifies shared CLI validation helpers.
func TestValidationHelpers(t *testing.T) {
	t.Parallel()

	t.Run("supports known output formats", func(t *testing.T) {
		t.Parallel()

		assert.True(t, IsSupportedOutputFormat(""))
		assert.True(t, IsSupportedOutputFormat("yaml"))
		assert.True(t, IsSupportedOutputFormat("json"))
		assert.False(t, IsSupportedOutputFormat("wide"))
	})

	t.Run("supports known service account values", func(t *testing.T) {
		t.Parallel()

		assert.True(t, IsSupportedServiceAccountValue(""))
		assert.True(t, IsSupportedServiceAccountValue("from-pod"))
		assert.False(t, IsSupportedServiceAccountValue("default"))
	})

	t.Run("requires command after dash when stdin or tty is enabled", func(t *testing.T) {
		t.Parallel()

		assert.True(t, RequiresCommandAfterDash(true, false, -1))
		assert.True(t, RequiresCommandAfterDash(false, true, -1))
		assert.False(t, RequiresCommandAfterDash(false, false, -1))
		assert.False(t, RequiresCommandAfterDash(true, false, 1))
	})

	t.Run("only rewrites subPath mounts when copy-volume-mounts is enabled", func(t *testing.T) {
		t.Parallel()

		assert.True(t, CanRewriteSubPathMounts(false, false))
		assert.True(t, CanRewriteSubPathMounts(true, false))
		assert.True(t, CanRewriteSubPathMounts(true, true))
		assert.False(t, CanRewriteSubPathMounts(false, true))
	})

	t.Run("stdin manifest conflicts with exec stdin", func(t *testing.T) {
		t.Parallel()

		assert.True(t, CanUseManifestStdin("", true))
		assert.True(t, CanUseManifestStdin("-", false))
		assert.False(t, CanUseManifestStdin("-", true))
	})
}

// TestResolvePodSource verifies pod resolution from args, files, and stdin.
func TestResolvePodSource(t *testing.T) {
	t.Parallel()

	t.Run("uses positional pod name", func(t *testing.T) {
		t.Parallel()

		podName, namespace, err := ResolvePodSource([]string{"demo-pod"}, -1, "", "override-ns", strings.NewReader(""))
		require.NoError(t, err)
		assert.Equal(t, "demo-pod", podName)
		assert.Equal(t, "override-ns", namespace)
	})

	t.Run("uses manifest namespace when no override is provided", func(t *testing.T) {
		t.Parallel()

		path := writeTestManifest(t, testPodManifest)
		podName, namespace, err := ResolvePodSource(nil, -1, path, "", strings.NewReader(""))
		require.NoError(t, err)
		assert.Equal(t, "demo-pod", podName)
		assert.Equal(t, "demo-ns", namespace)
	})

	t.Run("namespace flag overrides manifest namespace", func(t *testing.T) {
		t.Parallel()

		path := writeTestManifest(t, testPodManifest)
		podName, namespace, err := ResolvePodSource(nil, -1, path, "flag-ns", strings.NewReader(""))
		require.NoError(t, err)
		assert.Equal(t, "demo-pod", podName)
		assert.Equal(t, "flag-ns", namespace)
	})

	t.Run("reads manifest from stdin", func(t *testing.T) {
		t.Parallel()

		podName, namespace, err := ResolvePodSource(nil, -1, "-", "", strings.NewReader(testPodManifest))
		require.NoError(t, err)
		assert.Equal(t, "demo-pod", podName)
		assert.Equal(t, "demo-ns", namespace)
	})

	t.Run("rejects non-pod manifests", func(t *testing.T) {
		t.Parallel()

		path := writeTestManifest(t, `
apiVersion: v1
kind: ConfigMap
metadata:
  name: demo
`)
		_, _, err := ResolvePodSource(nil, -1, path, "", strings.NewReader(""))
		require.Error(t, err)
		assert.ErrorContains(t, err, "manifest must be a Pod")
	})

	t.Run("rejects multiple pod documents", func(t *testing.T) {
		t.Parallel()

		path := writeTestManifest(t, `
apiVersion: v1
kind: Pod
metadata:
  name: first
---
apiVersion: v1
kind: Pod
metadata:
  name: second
`)
		_, _, err := ResolvePodSource(nil, -1, path, "", strings.NewReader(""))
		require.Error(t, err)
		assert.ErrorContains(t, err, "exactly one Pod document")
	})
}

func writeTestManifest(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "pod.yaml")
	require.NoError(t, os.WriteFile(path, []byte(strings.TrimSpace(content)), 0o600))
	return path
}
