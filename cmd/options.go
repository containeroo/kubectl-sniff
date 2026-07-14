package cmd

import (
	"time"

	"github.com/containeroo/sniff/internal/cli"
	"github.com/spf13/cobra"
)

// workflowOptions contains the complete option set shared by the command forms.
// Each command registers only the fields that apply to its workflow.
type workflowOptions struct {
	namespace                string
	filename                 string
	image                    string
	name                     string
	containerName            string
	target                   string
	fromContainer            string
	command                  []string
	args                     []string
	stdin                    bool
	tty                      bool
	copyEnv                  bool
	copyEnvFrom              bool
	copyVolumeMounts         bool
	copyServiceAccountMounts bool
	rewriteSubPathMounts     bool
	serviceAccount           string
	dryRun                   bool
	output                   string
	quiet                    bool
	verbose                  bool
	profile                  string
	waitTimeout              time.Duration
}

func registerCommonWorkflowFlags(cmd *cobra.Command, opts *workflowOptions) {
	flags := cmd.Flags()
	flags.StringVarP(&opts.namespace, "namespace", "n", "", "Namespace of the source pod (defaults to current namespace)")
	flags.StringVarP(&opts.filename, "filename", "f", "", "Path to a Pod manifest to use as input; use - for stdin")
	flags.StringVar(&opts.image, "image", "", "Image for the debug container")
	flags.StringVar(&opts.fromContainer, "from-container", "", "Source regular container in the pod to copy fields from")
	flags.BoolVar(&opts.copyEnv, "copy-env", false, "Copy env entries from --from-container")
	flags.BoolVar(&opts.copyEnvFrom, "copy-env-from", false, "Copy envFrom entries from --from-container")
	flags.BoolVar(&opts.copyVolumeMounts, "copy-volume-mounts", false, "Copy volumeMounts from --from-container")
	flags.BoolVar(&opts.copyServiceAccountMounts, "copy-service-account-mounts", false, "When copying volume mounts, include service account token mounts")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "Print the generated pod manifest instead of applying it")
	flags.StringVarP(&opts.output, "output", "o", "", `Output format for --dry-run (supported: "yaml", "json")`)
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress non-error informational output")
	flags.BoolVarP(&opts.verbose, "verbose", "v", false, "Show detailed informational output")
	flags.StringVar(&opts.profile, "profile", "", `Apply a predefined debug profile ("general", "netadmin", "sysadmin", "privileged")`)
	markFlagRequired(cmd, "image")
	cli.RegisterProfileFlagCompletion(cmd)
}

func registerAttachWorkflowFlags(cmd *cobra.Command, opts *workflowOptions) {
	flags := cmd.Flags()
	flags.StringVarP(&opts.containerName, "container", "c", "", "Name of the new ephemeral debug container (defaults to a generated sniff-xxxxx name)")
	flags.StringVar(&opts.target, "target", "", "Target container name whose namespaces should be targeted when supported")
	flags.BoolVar(&opts.rewriteSubPathMounts, "rewrite-subpath-mounts", false, "Rewrite subPath and subPathExpr mounts to debug-friendly directory mounts under /mnt/sniff/volumes")
	flags.DurationVar(&opts.waitTimeout, "wait-timeout", 2*time.Minute, "Maximum time to wait for the ephemeral container before exec")
}

func registerCloneWorkflowFlags(cmd *cobra.Command, opts *workflowOptions) {
	flags := cmd.Flags()
	flags.StringVar(&opts.name, "name", "", "Name of the created debug pod (defaults to generated name)")
	flags.StringSliceVar(&opts.command, "command", nil, "Command for the standalone debug container")
	flags.StringSliceVar(&opts.args, "arg", nil, "Argument for --command; repeat for multiple arguments")
	flags.StringVar(&opts.serviceAccount, "service-account", "", `Service account for the cloned debug pod; use "from-pod" to copy from the source pod`)
}
