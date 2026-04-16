package cmd

import (
	"errors"
	"strings"

	"github.com/containeroo/sniff/internal/app"
	"github.com/containeroo/sniff/internal/cli"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

// runOptions stores the flags for the run command.
type runOptions struct {
	// namespace overrides the active kubectl namespace.
	namespace string
	// filename selects a pod manifest to use instead of a positional pod name.
	filename string
	// image is the container image for the standalone debug pod.
	image string
	// name is the explicit pod name to create instead of using GenerateName.
	name string
	// fromContainer is the regular container used as a copy source.
	fromContainer string
	// command overrides the debug container entrypoint.
	command []string
	// args appends arguments to the debug container command.
	args []string
	// stdin enables stdin for the standalone debug container.
	stdin bool
	// tty enables TTY allocation for the standalone debug container.
	tty bool
	// copyEnv copies env entries from the source container.
	copyEnv bool
	// copyEnvFrom copies envFrom entries from the source container.
	copyEnvFrom bool
	// copyVolumeMounts copies volume mounts from the source container.
	copyVolumeMounts bool
	// copyServiceAccountMounts includes service account token mounts when copying volumes.
	copyServiceAccountMounts bool
	// serviceAccount sets the service account on the created debug pod.
	serviceAccount string
	// dryRun prints the generated manifest instead of creating the pod.
	dryRun bool
	// output selects the dry-run output format.
	output string
	// quiet suppresses informational output.
	quiet bool
	// verbose enables detailed informational output.
	verbose bool
	// profile applies a predefined security context to the debug container.
	profile string
}

// toAppOptions converts run flags into application options.
func (o *runOptions) toAppOptions(command []string, args []string, quiet bool, verbose bool) app.RunOptions {
	return app.RunOptions{
		Namespace:                o.namespace,
		Image:                    o.image,
		Name:                     o.name,
		FromContainer:            o.fromContainer,
		Command:                  command,
		Args:                     args,
		Stdin:                    o.stdin,
		TTY:                      o.tty,
		CopyEnv:                  o.copyEnv,
		CopyEnvFrom:              o.copyEnvFrom,
		CopyVolumeMounts:         o.copyVolumeMounts,
		CopyServiceAccountMounts: o.copyServiceAccountMounts,
		ServiceAccount:           o.serviceAccount,
		DryRun:                   o.dryRun,
		Output:                   o.output,
		Quiet:                    quiet,
		Verbose:                  verbose,
		Profile:                  o.profile,
	}
}

// NewRunCmd builds the command that creates standalone debug pods.
func NewRunCmd(streams genericiooptions.IOStreams) *cobra.Command {
	opts := &runOptions{
		stdin: true,
		tty:   true,
	}

	cmd := &cobra.Command{
		Use:   "run (POD | -f FILE) --image IMAGE",
		Short: "Run a standalone debug pod based on an existing pod",
		Long: strings.TrimSpace(`
Create a new standalone debug pod in the same namespace as an existing pod.

The new pod contains only the debug container. Optional copy flags let you
copy selected fields from one regular container in the source pod.
`),
		Example: strings.TrimSpace(`
kubectl sniff run mypod --image ghcr.io/containeroo/alpine-toolbox

kubectl sniff run mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  --from-container app \
  --copy-env \
  --copy-env-from \
  --copy-volume-mounts

kubectl sniff run mypod \
  --image alpine \
  --command sh \
  --arg -c \
  --arg "sleep 3600"

kubectl sniff run mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  --service-account from-pod \
  --dry-run -o yaml

kubectl get pod mypod -o yaml | kubectl sniff run \
  -f - \
  --image ghcr.io/containeroo/alpine-toolbox
`),
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cli.ValidateSinglePodSource(opts.filename, len(args)); err != nil {
				return err
			}
			if !cli.IsSupportedOutputFormat(opts.output) {
				return errors.New(`--output must be "yaml", "json", or empty`)
			}
			if !cli.IsSupportedServiceAccountValue(opts.serviceAccount) {
				return errors.New(`--service-account must be empty, "from-pod", or a concrete service account name`)
			}
			if err := cli.ValidateProfileFlag(opts.profile); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			quiet, verbose := cli.ResolveQuietVerbose(cmd, opts.quiet, opts.verbose)
			command, commandArgs, err := cli.ResolveRunCommand(cmd, opts.command, opts.args)
			if err != nil {
				return err
			}

			podName, namespace, err := cli.ResolvePodSource(args, -1, opts.filename, opts.namespace, streams.In)
			if err != nil {
				return err
			}

			appOpts := opts.toAppOptions(command, commandArgs, quiet, verbose)
			appOpts.Namespace = namespace
			return app.RunStandalone(cmd.Context(), streams, podName, appOpts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.namespace, "namespace", "n", "", "Namespace of the source pod (defaults to current namespace)")
	flags.StringVarP(&opts.filename, "filename", "f", "", "Path to a Pod manifest to use as input; use - for stdin")
	flags.StringVar(&opts.image, "image", "", "Image for the debug container")
	flags.StringVar(&opts.name, "name", "", "Name of the created debug pod (defaults to generated name)")
	flags.StringVar(&opts.fromContainer, "from-container", "", "Source regular container in the pod to copy fields from")
	flags.StringSliceVar(&opts.command, "command", nil, "Command for the debug container")
	flags.StringSliceVar(&opts.args, "arg", nil, "Argument for the debug container; repeat for multiple arguments")
	flags.BoolVar(&opts.stdin, "stdin", true, "Enable stdin for the debug container")
	flags.BoolVar(&opts.tty, "tty", true, "Enable TTY for the debug container")
	flags.BoolVar(&opts.copyEnv, "copy-env", false, "Copy env entries from --from-container")
	flags.BoolVar(&opts.copyEnvFrom, "copy-env-from", false, "Copy envFrom entries from --from-container")
	flags.BoolVar(&opts.copyVolumeMounts, "copy-volume-mounts", false, "Copy volumeMounts from --from-container")
	flags.BoolVar(&opts.copyServiceAccountMounts, "copy-service-account-mounts", false, "When copying volume mounts, include service account token mounts")
	flags.StringVar(&opts.serviceAccount, "service-account", "", `Service account for the debug pod; use "from-pod" to copy from the source pod`)
	flags.BoolVar(&opts.dryRun, "dry-run", false, "Print the generated pod manifest instead of creating it")
	flags.StringVarP(&opts.output, "output", "o", "", `Output format for --dry-run (supported: "yaml", "json")`)
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress non-error informational output")
	flags.BoolVarP(&opts.verbose, "verbose", "v", false, "Show detailed informational output")
	flags.StringVar(&opts.profile, "profile", "", `Apply a predefined debug profile ("general", "netadmin", "sysadmin", "privileged")`)
	markFlagRequired(cmd, "image")
	cli.RegisterProfileFlagCompletion(cmd)

	return cmd
}
