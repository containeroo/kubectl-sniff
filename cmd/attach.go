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
	// namespace overrides the active kubectl namespace.
	namespace string
	// filename selects a pod manifest to use instead of a positional pod name.
	filename string
	// image is the container image for the ephemeral debugger.
	image string
	// containerName is the name assigned to the new ephemeral container.
	containerName string
	// target is the regular container whose namespaces should be targeted.
	target string
	// fromContainer is the regular container used as a copy source.
	fromContainer string
	// stdin enables stdin for the post-create exec session.
	stdin bool
	// tty enables TTY allocation for the post-create exec session.
	tty bool
	// copyEnv copies env entries from the source container.
	copyEnv bool
	// copyEnvFrom copies envFrom entries from the source container.
	copyEnvFrom bool
	// copyVolumeMounts copies volume mounts from the source container.
	copyVolumeMounts bool
	// copyServiceAccountMounts includes service account token mounts when copying volumes.
	copyServiceAccountMounts bool
	// rewriteSubPathMounts rewrites subPath mounts into debug-friendly direct mounts.
	rewriteSubPathMounts bool
	// dryRun prints the updated manifest instead of patching the pod.
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

			podName, namespace, err := cli.ResolvePodSource(args, dash, opts.filename, opts.namespace, streams.In)
			if err != nil {
				return err
			}

			quiet, verbose := cli.ResolveQuietVerbose(cmd, opts.quiet, opts.verbose)
			appOpts := opts.toAppOptions(execCommand, quiet, verbose)
			appOpts.Namespace = namespace
			return app.RunAttach(cmd.Context(), streams, podName, appOpts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.namespace, "namespace", "n", "", "Namespace of the target pod (defaults to current namespace)")
	flags.StringVarP(&opts.filename, "filename", "f", "", "Path to a Pod manifest to use as input; use - for stdin")
	flags.StringVar(&opts.image, "image", "", "Image for the new ephemeral debug container")
	flags.StringVarP(&opts.containerName, "container", "c", opts.containerName, "Name of the new ephemeral debug container (defaults to a generated sniff-xxxxx name)")
	flags.StringVar(&opts.target, "target", "", "Target container name whose namespaces should be targeted when supported")
	flags.StringVar(&opts.fromContainer, "from-container", "", "Source regular container in the pod to copy fields from")
	flags.BoolVarP(&opts.stdin, "stdin", "i", false, "Pass stdin to the command executed after --")
	flags.BoolVarP(&opts.tty, "tty", "t", false, "Allocate a TTY for the command executed after --")
	flags.BoolVar(&opts.copyEnv, "copy-env", false, "Copy env entries from --from-container")
	flags.BoolVar(&opts.copyEnvFrom, "copy-env-from", false, "Copy envFrom entries from --from-container")
	flags.BoolVar(&opts.copyVolumeMounts, "copy-volume-mounts", false, "Copy volumeMounts from --from-container")
	flags.BoolVar(&opts.copyServiceAccountMounts, "copy-service-account-mounts", false, "When copying volume mounts, include service account token mounts")
	flags.BoolVar(&opts.rewriteSubPathMounts, "rewrite-subpath-mounts", false, "Rewrite subPath and subPathExpr mounts to debug-friendly directory mounts under /mnt/sniff/volumes")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "Print the updated pod manifest instead of patching the pod")
	flags.StringVarP(&opts.output, "output", "o", "", `Output format for --dry-run (supported: "yaml", "json")`)
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress non-error informational output")
	flags.BoolVarP(&opts.verbose, "verbose", "v", false, "Show detailed informational output")
	flags.StringVar(&opts.profile, "profile", "", `Apply a predefined debug profile ("general", "netadmin", "sysadmin", "privileged")`)
	markFlagRequired(cmd, "image")
	cli.RegisterProfileFlagCompletion(cmd)

	return cmd
}
