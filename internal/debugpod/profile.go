package debugpod

import (
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	// ProfileGeneral keeps the debug container close to the default runtime posture.
	ProfileGeneral = "general"
	// ProfileNetAdmin adds network-focused capabilities.
	ProfileNetAdmin = "netadmin"
	// ProfileSysAdmin adds broader system-debugging capabilities.
	ProfileSysAdmin = "sysadmin"
	// ProfilePrivileged enables a fully privileged debug container.
	ProfilePrivileged = "privileged"
)

var supportedProfiles = []string{
	ProfileGeneral,
	ProfileNetAdmin,
	ProfileSysAdmin,
	ProfilePrivileged,
}

// BuildReport summarizes which debug-oriented settings were applied.
type BuildReport struct {
	SourceContainer             string
	Profile                     string
	CopiedEnv                   int
	CopiedEnvFrom               int
	CopiedVolumeMounts          int
	RewrittenSubPathMounts      int
	SkippedSubPathMounts        int
	SkippedServiceAccountMounts int
}

// HasDetails reports whether the build added anything worth summarizing.
func (r BuildReport) HasDetails() bool {
	return r.Profile != "" ||
		r.CopiedEnv != 0 ||
		r.CopiedEnvFrom != 0 ||
		r.CopiedVolumeMounts != 0 ||
		r.RewrittenSubPathMounts != 0 ||
		r.SkippedSubPathMounts != 0 ||
		r.SkippedServiceAccountMounts != 0
}

// SupportedProfiles returns the allowed --profile values.
func SupportedProfiles() []string {
	profiles := make([]string, len(supportedProfiles))
	copy(profiles, supportedProfiles)
	return profiles
}

// NormalizeProfile canonicalizes a user-provided profile name.
func NormalizeProfile(profile string) string {
	return strings.ToLower(strings.TrimSpace(profile))
}

// ValidateProfile returns an error when the named profile is unsupported.
func ValidateProfile(profile string) error {
	normalized := NormalizeProfile(profile)
	if normalized == "" {
		return nil
	}

	if slices.Contains(supportedProfiles, normalized) {
		return nil
	}

	return fmt.Errorf(`--profile must be one of "%s", "%s", "%s", or "%s"`,
		ProfileGeneral,
		ProfileNetAdmin,
		ProfileSysAdmin,
		ProfilePrivileged,
	)
}

// applyProfileToContainer mutates a regular container to match the requested profile.
func applyProfileToContainer(container *corev1.Container, profile string) {
	container.SecurityContext = buildSecurityContextForProfile(profile)
}

// applyProfileToEphemeralContainer mutates an ephemeral container to match the requested profile.
func applyProfileToEphemeralContainer(container *corev1.EphemeralContainer, profile string) {
	container.SecurityContext = buildSecurityContextForProfile(profile)
}

// buildSecurityContextForProfile returns the debug security context for the named profile.
func buildSecurityContextForProfile(profile string) *corev1.SecurityContext {
	switch NormalizeProfile(profile) {
	case "":
		return nil
	case ProfileGeneral:
		return &corev1.SecurityContext{
			AllowPrivilegeEscalation: new(false),
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		}
	case ProfileNetAdmin:
		return &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"NET_ADMIN",
					"NET_RAW",
				},
			},
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		}
	case ProfileSysAdmin:
		return &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					"SYS_ADMIN",
					"SYS_PTRACE",
				},
			},
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeUnconfined,
			},
		}
	case ProfilePrivileged:
		return &corev1.SecurityContext{
			AllowPrivilegeEscalation: new(true),
			Privileged:               new(true),
		}
	default:
		return nil
	}
}
