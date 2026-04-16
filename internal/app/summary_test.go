package app

import (
	"testing"

	"github.com/containeroo/sniff/internal/debugpod"
	"github.com/stretchr/testify/assert"
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
