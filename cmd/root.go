package cmd

import (
	"errors"
	"os"
	"strings"

	"github.com/containeroo/sniff/internal/app"
	"github.com/containeroo/sniff/internal/cli"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

// rootOptions stores the flags for the preferred root workflow.
type rootOptions struct {
	workflowOptions
	// clone switches from in-place ephemeral attach to standalone debug pod creation.
	clone bool
}

// NewRootCmd creates the root command for the kubectl plugin.
func NewRootCmd() *cobra.Command {
	streams := genericiooptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	opts := &rootOptions{}

	cmd := &cobra.Command{
		Use:   "sniff (POD | -f FILE) --image IMAGE [flags] -- [command...]",
		Short: "Attach an ephemeral debugger or create a cloned debug pod",
		Long: strings.TrimSpace(`
Attach an ephemeral debug container to an existing pod by default.

Use --clone to create a separate standalone debug pod derived from the source
pod instead of modifying the original pod.
`),
		Example: strings.TrimSpace(`
kubectl sniff mypod --image ghcr.io/containeroo/alpine-toolbox

kubectl sniff mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  --from-container app \
  --copy-env \
  --copy-env-from \
  --copy-volume-mounts

kubectl sniff mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  -it -- bash

kubectl sniff mypod \
  --clone \
  --image ghcr.io/containeroo/alpine-toolbox \
  --from-container app \
  --copy-env \
  --copy-env-from \
  --copy-volume-mounts

kubectl get pod mypod -o yaml | kubectl sniff \
  -f - \
  --image ghcr.io/containeroo/alpine-toolbox
`),
		Args: func(cmd *cobra.Command, args []string) error {
			return validateRootArgs(cmd, opts, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRootWorkflow(cmd, streams, opts, args)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.SetOut(streams.Out)
	cmd.SetErr(streams.ErrOut)

	registerCommonWorkflowFlags(cmd, &opts.workflowOptions)
	registerAttachWorkflowFlags(cmd, &opts.workflowOptions)
	registerCloneWorkflowFlags(cmd, &opts.workflowOptions)
	flags := cmd.Flags()
	flags.BoolVar(&opts.clone, "clone", false, "Create a standalone debug pod instead of attaching an ephemeral container")
	flags.BoolVarP(&opts.stdin, "stdin", "i", false, "Pass stdin to the command executed after --; with --clone, keep stdin open on the standalone debug container")
	flags.BoolVarP(&opts.tty, "tty", "t", false, "Allocate a TTY for the command executed after --; with --clone, allocate a TTY on the standalone debug container")

	cmd.AddCommand(NewAttachCmd(streams))
	cmd.AddCommand(NewRunCmd(streams))

	return cmd
}

// validateRootArgs validates the root command flags and positional arguments.
func validateRootArgs(cmd *cobra.Command, opts *rootOptions, args []string) error {
	dash := cmd.ArgsLenAtDash()
	podArgs := len(args)
	if dash != -1 {
		podArgs = dash
		if len(args[dash:]) == 0 {
			return errors.New("a command is required after --")
		}
	}

	if err := cli.ValidateSinglePodSource(opts.filename, podArgs); err != nil {
		return err
	}
	if !cli.IsSupportedOutputFormat(opts.output) {
		return errors.New(`--output must be "yaml", "json", or empty`)
	}
	if err := cli.ValidateProfileFlag(opts.profile); err != nil {
		return err
	}

	if opts.clone {
		if dash != -1 {
			return errors.New("--clone does not accept a command after --; use --command and --arg instead")
		}
		if cmd.Flags().Changed("container") {
			return errors.New("--container is only supported when attaching an ephemeral container")
		}
		if cmd.Flags().Changed("target") {
			return errors.New("--target is only supported when attaching an ephemeral container")
		}
		if cmd.Flags().Changed("rewrite-subpath-mounts") {
			return errors.New("--rewrite-subpath-mounts is only supported when attaching an ephemeral container")
		}
		if !cli.IsSupportedServiceAccountValue(opts.serviceAccount) {
			return errors.New(`--service-account must be empty, "from-pod", or a concrete service account name`)
		}

		return nil
	}

	if cmd.Flags().Changed("name") {
		return errors.New("--name requires --clone")
	}
	if cmd.Flags().Changed("service-account") {
		return errors.New("--service-account requires --clone")
	}
	if cmd.Flags().Changed("command") {
		return errors.New("--command requires --clone")
	}
	if cmd.Flags().Changed("arg") {
		return errors.New("--arg requires --clone")
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

	return nil
}

// runRootWorkflow runs the root command.
func runRootWorkflow(
	cmd *cobra.Command,
	streams genericiooptions.IOStreams,
	opts *rootOptions,
	args []string,
) error {
	dash := cmd.ArgsLenAtDash()
	podName, namespace, err := cli.ResolvePodSource(args, dash, opts.filename, opts.namespace, streams.In)
	if err != nil {
		return err
	}

	quiet, verbose := cli.ResolveQuietVerbose(cmd, opts.quiet, opts.verbose)
	if opts.clone {
		command, commandArgs, err := cli.ResolveRunCommand(cmd, opts.command, opts.args)
		if err != nil {
			return err
		}

		appOpts := app.RunOptions{
			Namespace:                namespace,
			Image:                    opts.image,
			Name:                     opts.name,
			FromContainer:            opts.fromContainer,
			Command:                  command,
			Args:                     commandArgs,
			Stdin:                    resolveRootBoolFlag(cmd, "stdin", true, opts.stdin),
			TTY:                      resolveRootBoolFlag(cmd, "tty", true, opts.tty),
			CopyEnv:                  opts.copyEnv,
			CopyEnvFrom:              opts.copyEnvFrom,
			CopyVolumeMounts:         opts.copyVolumeMounts,
			CopyServiceAccountMounts: opts.copyServiceAccountMounts,
			ServiceAccount:           opts.serviceAccount,
			DryRun:                   opts.dryRun,
			Output:                   opts.output,
			Quiet:                    quiet,
			Verbose:                  verbose,
			Profile:                  opts.profile,
		}

		return app.RunStandalone(cmd.Context(), streams, podName, appOpts)
	}

	var execCommand []string
	if dash != -1 {
		execCommand = args[dash:]
	}

	appOpts := app.AttachOptions{
		Namespace:                namespace,
		Image:                    opts.image,
		ContainerName:            opts.containerName,
		Target:                   opts.target,
		FromContainer:            opts.fromContainer,
		ExecCommand:              execCommand,
		Stdin:                    opts.stdin,
		TTY:                      opts.tty,
		CopyEnv:                  opts.copyEnv,
		CopyEnvFrom:              opts.copyEnvFrom,
		CopyVolumeMounts:         opts.copyVolumeMounts,
		CopyServiceAccountMounts: opts.copyServiceAccountMounts,
		RewriteSubPathMounts:     opts.rewriteSubPathMounts,
		DryRun:                   opts.dryRun,
		Output:                   opts.output,
		Quiet:                    quiet,
		Verbose:                  verbose,
		Profile:                  opts.profile,
		WaitTimeout:              opts.waitTimeout,
	}

	return app.RunAttach(cmd.Context(), streams, podName, appOpts)
}

func resolveRootBoolFlag(cmd *cobra.Command, name string, defaultValue bool, value bool) bool {
	if cmd.Flags().Changed(name) {
		return value
	}

	return defaultValue
}

// markFlagRequired panics only when command construction is internally inconsistent.
func markFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(err)
	}
}
