package app

import (
	"errors"
	"io"
	"testing"

	"github.com/containeroo/sniff/internal/debugpod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

// TestSummarizeBuildReport verifies the human-readable build summary format.
func TestSummarizeBuildReport(t *testing.T) {
	t.Parallel()

	report := debugpod.BuildReport{
		SourceContainer:             "app",
		Profile:                     "netadmin",
		CopiedEnv:                   2,
		CopiedEnvFrom:               1,
		CopiedVolumeMounts:          3,
		RewrittenSubPathMounts:      1,
		SkippedServiceAccountMounts: 2,
	}

	got := summarizeBuildReport(report)
	want := `applied "netadmin" profile; copied from container "app": 2 env entries, 1 envFrom source, 3 volume mounts; rewrote 1 subPath mount; skipped 2 service account mounts`
	assert.Equal(t, want, got)
}

func TestSummaryWritersPropagateErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("broken output")
	streams := genericiooptions.IOStreams{Out: failingWriter{err: wantErr}, ErrOut: io.Discard}

	err := writeAttachShellHint(streams, "default", "app", "debugger")
	require.ErrorIs(t, err, wantErr)

	err = writeStandaloneShellHint(streams, "default", "debugger")
	require.ErrorIs(t, err, wantErr)

	err = writeBuildSummary(streams, debugpod.BuildReport{Profile: "general"}, false)
	require.ErrorIs(t, err, wantErr)
}

type failingWriter struct {
	err error
}

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}
