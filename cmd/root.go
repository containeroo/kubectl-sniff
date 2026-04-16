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
	// clone switches from in-place ephemeral attach to standalone debug pod creation.
	clone bool
	// namespace overrides the active kubectl namespace.
	namespace string
	// filename selects a pod manifest to use instead of a positional pod name.
	filename string
	// image is the debug container image for either workflow.
	image string
	// name is the explicit pod name to create in clone mode.
	name string
	// containerName is the name assigned to the new ephemeral container.
	containerName string
	// target is the regular container whose namespaces should be targeted.
	target string
	// fromContainer is the regular container used as a copy source.
	fromContainer string
	// command overrides the standalone debug container entrypoint in clone mode.
	command []string
	// args appends arguments to the standalone debug container command in clone mode.
	args []string
	// stdin controls exec stdin for attach mode and container stdin for clone mode.
	stdin bool
	// tty controls exec TTY for attach mode and container TTY for clone mode.
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
	// serviceAccount sets the service account on the created debug pod in clone mode.
	serviceAccount string
	// dryRun prints the generated manifest instead of creating resources.
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

	flags := cmd.Flags()
	flags.BoolVar(&opts.clone, "clone", false, "Create a standalone debug pod instead of attaching an ephemeral container")
	flags.StringVarP(&opts.namespace, "namespace", "n", "", "Namespace of the source pod (defaults to current namespace)")
	flags.StringVarP(&opts.filename, "filename", "f", "", "Path to a Pod manifest to use as input; use - for stdin")
	flags.StringVar(&opts.image, "image", "", "Image for the debug container")
	flags.StringVar(&opts.name, "name", "", "Name of the created debug pod in clone mode (defaults to generated name)")
	flags.StringVarP(&opts.containerName, "container", "c", "", "Name of the new ephemeral debug container")
	flags.StringVar(&opts.target, "target", "", "Target container name whose namespaces should be targeted when supported")
	flags.StringVar(&opts.fromContainer, "from-container", "", "Source regular container in the pod to copy fields from")
	flags.StringSliceVar(&opts.command, "command", nil, "Command for the standalone debug container in clone mode")
	flags.StringSliceVar(&opts.args, "arg", nil, "Argument for --command in clone mode; repeat for multiple arguments")
	flags.BoolVarP(&opts.stdin, "stdin", "i", false, "Pass stdin to the command executed after --; with --clone, keep stdin open on the standalone debug container")
	flags.BoolVarP(&opts.tty, "tty", "t", false, "Allocate a TTY for the command executed after --; with --clone, allocate a TTY on the standalone debug container")
	flags.BoolVar(&opts.copyEnv, "copy-env", false, "Copy env entries from --from-container")
	flags.BoolVar(&opts.copyEnvFrom, "copy-env-from", false, "Copy envFrom entries from --from-container")
	flags.BoolVar(&opts.copyVolumeMounts, "copy-volume-mounts", false, "Copy volumeMounts from --from-container")
	flags.BoolVar(&opts.copyServiceAccountMounts, "copy-service-account-mounts", false, "When copying volume mounts, include service account token mounts")
	flags.BoolVar(&opts.rewriteSubPathMounts, "rewrite-subpath-mounts", false, "Rewrite subPath and subPathExpr mounts to debug-friendly directory mounts under /mnt/sniff/volumes")
	flags.StringVar(&opts.serviceAccount, "service-account", "", `Service account for the cloned debug pod; use "from-pod" to copy from the source pod`)
	flags.BoolVar(&opts.dryRun, "dry-run", false, "Print the generated manifest instead of creating it")
	flags.StringVarP(&opts.output, "output", "o", "", `Output format for --dry-run (supported: "yaml", "json")`)
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress non-error informational output")
	flags.BoolVarP(&opts.verbose, "verbose", "v", false, "Show detailed informational output")
	flags.StringVar(&opts.profile, "profile", "", `Apply a predefined debug profile ("general", "netadmin", "sysadmin", "privileged")`)
	markFlagRequired(cmd, "image")
	cli.RegisterProfileFlagCompletion(cmd)

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
