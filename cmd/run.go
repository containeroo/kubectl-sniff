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
	workflowOptions
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
		workflowOptions: workflowOptions{
			stdin: true,
			tty:   true,
		},
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
			if opts.filename == "" {
				return validateExplicitNames(&opts.workflowOptions, args[0], true)
			}
			if err := validateExplicitNames(&opts.workflowOptions, "", true); err != nil {
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

			podName, namespace, err := cli.ResolvePodReference(args, -1, opts.filename, opts.namespace, streams.In)
			if err != nil {
				return err
			}

			appOpts := opts.toAppOptions(command, commandArgs, quiet, verbose)
			appOpts.Namespace = namespace
			return app.RunStandalone(cmd.Context(), streams, podName, appOpts)
		},
	}

	registerCommonWorkflowFlags(cmd, &opts.workflowOptions)
	registerCloneWorkflowFlags(cmd, &opts.workflowOptions)
	flags := cmd.Flags()
	flags.BoolVar(&opts.stdin, "stdin", true, "Enable stdin for the debug container")
	flags.BoolVar(&opts.tty, "tty", true, "Enable TTY for the debug container")

	return cmd
}
