package cmd

import (
	"errors"
	"strings"

	"github.com/containeroo/sniff/internal/app"
	"github.com/containeroo/sniff/internal/cli"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

// attachOptions stores the flags for the attach command.
type attachOptions struct {
	workflowOptions
}

// toAppOptions converts attach flags into application options.
func (o *attachOptions) toAppOptions(execCommand []string, quiet bool, verbose bool) app.AttachOptions {
	return app.AttachOptions{
		Namespace:                o.namespace,
		Image:                    o.image,
		ContainerName:            o.containerName,
		Target:                   o.target,
		FromContainer:            o.fromContainer,
		ExecCommand:              execCommand,
		Stdin:                    o.stdin,
		TTY:                      o.tty,
		CopyEnv:                  o.copyEnv,
		CopyEnvFrom:              o.copyEnvFrom,
		CopyVolumeMounts:         o.copyVolumeMounts,
		CopyServiceAccountMounts: o.copyServiceAccountMounts,
		RewriteSubPathMounts:     o.rewriteSubPathMounts,
		DryRun:                   o.dryRun,
		Output:                   o.output,
		Quiet:                    quiet,
		Verbose:                  verbose,
		Profile:                  o.profile,
		WaitTimeout:              o.waitTimeout,
	}
}

// NewAttachCmd builds the command that adds ephemeral debug containers to pods.
func NewAttachCmd(streams genericiooptions.IOStreams) *cobra.Command {
	opts := &attachOptions{}

	cmd := &cobra.Command{
		Use:   "attach (POD | -f FILE) --image IMAGE [flags] -- [command...]",
		Short: "Attach an ephemeral debug container to an existing pod",
		Long: strings.TrimSpace(`
Attach a new ephemeral debug container to an existing pod.

The debug container is added to the pod's ephemeralcontainers subresource.
Optional copy flags let you copy selected fields from an existing regular
container in the same pod into the new debug container.

If a command is provided after --, the plugin waits for the ephemeral
container to be running and then execs that command inside it.
`),
		Example: strings.TrimSpace(`
kubectl sniff attach mypod --image ghcr.io/containeroo/alpine-toolbox

kubectl sniff attach mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  --container debugger \
  --target app

kubectl sniff attach mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  --copy-env \
  --copy-env-from \
  --copy-volume-mounts

kubectl sniff attach mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  --copy-volume-mounts \
  --rewrite-subpath-mounts

kubectl sniff attach mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  -it -- /bin/bash

kubectl sniff attach mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  --copy-env \
  --copy-env-from \
  --copy-volume-mounts \
  --dry-run -o yaml

kubectl get pod mypod -o yaml | kubectl sniff attach \
  -f - \
  --image ghcr.io/containeroo/alpine-toolbox
`),
		Args: func(cmd *cobra.Command, args []string) error {
			dash := cmd.ArgsLenAtDash()
			podArgs := len(args)
			if dash != -1 {
				podArgs = dash
			}

			switch dash {
			case -1:
			case 1:
				if len(args[dash:]) == 0 {
					return errors.New("a command is required after --")
				}
			case 0:
				if len(args[dash:]) == 0 {
					return errors.New("a command is required after --")
				}
			default:
				if dash > 1 {
					return errors.New("exactly one pod source must be provided before --")
				}
			}

			if err := cli.ValidateSinglePodSource(opts.filename, podArgs); err != nil {
				return err
			}
			if !cli.IsSupportedOutputFormat(opts.output) {
				return errors.New(`--output must be "yaml", "json", or empty`)
			}
			if cli.RequiresCommandAfterDash(opts.stdin, opts.tty, dash) {
				return errors.New("-i/--stdin and -t/--tty require a command after --")
			}
			if !cli.CanUseManifestStdin(opts.filename, opts.stdin) {
				return errors.New("-f - cannot be combined with -i/--stdin because stdin is used for manifest input")
			}
			if !cli.CanRewriteSubPathMounts(opts.copyVolumeMounts, opts.rewriteSubPathMounts) {
				return errors.New("--rewrite-subpath-mounts requires --copy-volume-mounts")
			}
			if err := cli.ValidateProfileFlag(opts.profile); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			dash := cmd.ArgsLenAtDash()

			var execCommand []string
			if dash != -1 {
				execCommand = args[dash:]
			}

			podName, namespace, err := cli.ResolvePodReference(args, dash, opts.filename, opts.namespace, streams.In)
			if err != nil {
				return err
			}

			quiet, verbose := cli.ResolveQuietVerbose(cmd, opts.quiet, opts.verbose)
			appOpts := opts.toAppOptions(execCommand, quiet, verbose)
			appOpts.Namespace = namespace
			return app.RunAttach(cmd.Context(), streams, podName, appOpts)
		},
	}

	registerCommonWorkflowFlags(cmd, &opts.workflowOptions)
	registerAttachWorkflowFlags(cmd, &opts.workflowOptions)
	flags := cmd.Flags()
	flags.BoolVarP(&opts.stdin, "stdin", "i", false, "Pass stdin to the command executed after --")
	flags.BoolVarP(&opts.tty, "tty", "t", false, "Allocate a TTY for the command executed after --")

	return cmd
}
