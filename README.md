# sniff

`sniff` is a `kubectl` plugin for debugging pods when plain `kubectl debug` is not enough.

It can:

- attach an ephemeral debug container to an existing pod by default
- copy `env` and `envFrom` from a regular container
- copy `volumeMounts` from a regular container
- explain what was copied or skipped after building the debug spec
- optionally include service account token mounts
- create a standalone debug pod derived from an existing pod with `--clone`
- apply predefined debug security profiles such as `netadmin` or `privileged`
- print the generated manifest with `--dry-run -o yaml` or `--dry-run -o json`

## TL;DR

`sniff` now prefers a single top-level workflow:

- `kubectl sniff ...`: add an ephemeral debug container to the existing pod
- `kubectl sniff --clone ...`: create a separate standalone debug pod derived from the existing pod

Use the default mode when:

- you want to debug the existing pod in place
- you want to target the namespaces of a running container
- you want to open a shell immediately with `-it -- bash` or `-it -- sh`

```bash
kubectl sniff mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  --from-container app \
  --copy-env \
  --copy-env-from \
  --copy-volume-mounts \
  -it -- bash
```

Use `--clone` when:

- you want a separate debug pod instead of touching the original pod
- you want a safer working copy for experiments
- you want to keep the original pod unchanged and `exec` into the copy afterward

```bash
kubectl sniff mypod \
  --clone \
  --image ghcr.io/containeroo/alpine-toolbox \
  --from-container app \
  --copy-env \
  --copy-env-from \
  --copy-volume-mounts

kubectl exec -it <debug-pod-name> -- bash
```

Quick rule:

- if you want a shell inside the existing pod now, use `kubectl sniff`
- if you want a separate debug pod first, add `--clone`

## Why sniff?

`kubectl debug` is great for many cases, but it does not cover the common situation where your debug container also needs:

- the same environment variables as the app container
- the same `envFrom` config and secret references
- the same mounted volumes

`sniff` focuses on that workflow.

## Install

`kubectl` discovers plugins from executables on your `PATH` named `kubectl-<name>`.

To use this plugin, place the binary on your `PATH` as:

```bash
kubectl-sniff
```

Then run it as:

```bash
kubectl sniff
```

## Common Flags

| Flag                            | Applies to                    | What it does                                                 |
| ------------------------------- | ----------------------------- | ------------------------------------------------------------ |
| `--image`                       | default, `--clone`           | Sets the debug container image                               |
| `--clone`                       | top-level                     | Creates a standalone debug pod instead of attaching in place |
| `--from-container`              | default, `--clone`           | Selects the regular container to copy fields from            |
| `--copy-env`                    | default, `--clone`           | Copies `env` entries from `--from-container`                 |
| `--copy-env-from`               | default, `--clone`           | Copies `envFrom` entries from `--from-container`             |
| `--copy-volume-mounts`          | default, `--clone`           | Copies `volumeMounts` from `--from-container`                |
| `--copy-service-account-mounts` | default, `--clone`           | Includes service account token mounts when copying volumes   |
| `--profile`                     | default, `--clone`           | Applies a predefined security profile                        |
| `--dry-run`                     | default, `--clone`           | Prints the generated manifest instead of applying it         |
| `-o, --output`                  | default, `--clone`           | Selects the dry-run format: yaml or json                     |
| `--target`                      | default                       | Targets another container's namespaces when supported        |
| `--rewrite-subpath-mounts`      | default                       | Rewrites copied `subPath` mounts into debug-friendly mounts  |
| `--command`                     | `--clone`                     | Sets the standalone debug container command                  |
| `--arg`                         | `--clone`                     | Appends arguments to `--command`; repeat for multiple values |
| `--service-account`             | `--clone`                     | Sets the service account on the standalone debug pod         |
| `--quiet`                       | default, `--clone`           | Suppresses non-error output                                  |
| `--verbose`                     | default, `--clone`           | Shows detailed copy summaries                                |

## Usage

### Default: attach an ephemeral container

Use the top-level command when you want to debug the existing pod in place.

This adds an ephemeral container to the target pod. If you pass a command after `--`,
`sniff` waits for that ephemeral container to start and then opens an interactive exec
session into it, similar to the `kubectl debug ... -it -- sh` workflow.

If that command is long-running, the ephemeral container also stays attached to the pod
until the process exits or the pod is replaced. This can be useful, but unlike
`--clone` it adds more long-lived state to the original pod.

Minimal attach:

```bash
kubectl sniff mypod --image ghcr.io/containeroo/alpine-toolbox
```

Attach and target the app container namespaces:

```bash
kubectl sniff mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  --target app
```

Attach and copy environment plus mounted volumes from the source container:

```bash
kubectl sniff mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  --from-container app \
  --copy-env \
  --copy-env-from \
  --copy-volume-mounts
```

Attach with a predefined debug profile:

```bash
kubectl sniff mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  --profile netadmin
```

Attach and immediately run a shell:

```bash
kubectl sniff mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  -it -- bash
```

Attach and open `sh` instead:

```bash
kubectl sniff mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  -it -- sh
```

Render the generated pod update without applying it:

```bash
kubectl sniff mypod \
  --image ghcr.io/containeroo/alpine-toolbox \
  --from-container app \
  --copy-env \
  --copy-env-from \
  --copy-volume-mounts \
  --dry-run -o yaml
```

### `--clone`: run a standalone debug pod

Use `--clone` when you want a separate debug pod derived from the source pod.

This creates a new pod with one debug container. Unlike the default attach workflow, `--clone` does not open
an immediate shell with `-- bash`; instead it creates the standalone pod and keeps it
running so you can `kubectl exec` into it afterward.

If you know `kubectl debug`, think of `--clone` as the "create a copy of this pod for
debugging" workflow, similar in spirit to `kubectl debug --copy-to=...`.

Minimal standalone run:

```bash
kubectl sniff mypod --clone --image ghcr.io/containeroo/alpine-toolbox
```

Copy selected fields from one container:

```bash
kubectl sniff mypod \
  --clone \
  --image ghcr.io/containeroo/alpine-toolbox \
  --from-container app \
  --copy-env \
  --copy-env-from \
  --copy-volume-mounts
```

Run with a predefined debug profile:

```bash
kubectl sniff mypod \
  --clone \
  --image ghcr.io/containeroo/alpine-toolbox \
  --profile privileged
```

Create a debug pod and then enter a shell:

```bash
kubectl sniff mypod \
  --clone \
  --image ghcr.io/containeroo/alpine-toolbox \
  --from-container app \
  --copy-env \
  --copy-env-from \
  --copy-volume-mounts

kubectl exec -it <debug-pod-name> -- bash
```

Run a custom command instead of the default long-running sleep:

```bash
kubectl sniff mypod \
  --clone \
  --image alpine \
  --command sh \
  --arg -c \
  --arg "sleep 3600"
```

Copy the service account from the source pod:

```bash
kubectl sniff mypod \
  --clone \
  --image ghcr.io/containeroo/alpine-toolbox \
  --service-account from-pod
```

Render the generated pod manifest as JSON:

```bash
kubectl sniff mypod \
  --clone \
  --image ghcr.io/containeroo/alpine-toolbox \
  --from-container app \
  --copy-env \
  --copy-volume-mounts \
  --dry-run -o json
```

## Output formats

When `--dry-run` is used, `sniff` supports:

- `-o yaml`
- `-o json`

If `-o` is omitted with `--dry-run`, YAML is used.

Use `-q` or `--quiet` to suppress informational output such as creation messages
and shell hints. Dry-run manifests are still printed.

Use `-v` or `--verbose` to print detailed copy summaries.

You can also configure defaults from your shell profile:

```bash
export SNIFF_VERBOSE=1
export SNIFF_QUIET=1
export SNIFF_RUN_COMMAND=/bin/sh
export SNIFF_RUN_ARGS='["-lc","sleep infinity"]'
```

If both are enabled, quiet wins.
For standalone debug pods, `SNIFF_RUN_COMMAND` and `SNIFF_RUN_ARGS` provide env-based
defaults for `--command` and `--arg`. CLI flags win when both are set.

## Important behavior

### `--from-container`

When you use copy flags such as:

- `--copy-env`
- `--copy-env-from`
- `--copy-volume-mounts`

`sniff` needs a source regular container.

Resolution order:

- explicit `--from-container`
- the pod's `kubectl.kubernetes.io/default-container` annotation, if it matches a regular container
- the only regular container, if there is exactly one
- otherwise `sniff` asks you to pass `--from-container`

### `--target`

For ephemeral containers, `--target` selects the container whose namespaces should be targeted.

If the pod has only one regular container, `sniff` uses it automatically.

### Rewriting `subPath` mounts

Ephemeral containers cannot always use copied `subPath` mounts as-is.

If you want volume mounts copied from a container that uses `subPath` or `subPathExpr`, use:

```bash
--copy-volume-mounts --rewrite-subpath-mounts
```

This rewrites those mounts under:

```text
/mnt/sniff/volumes
```

### `--profile`

`sniff` supports these predefined profiles:

- `general`
- `netadmin`
- `sysadmin`
- `privileged`

Profiles apply a predefined container security context to the debug container.
Use `--dry-run -o yaml` to inspect the exact manifest that a profile produces.

### Shell behavior

For the default attach flow, you can open a shell directly by passing a command after `--`:

```bash
kubectl sniff mypod --image alpine -it -- sh
kubectl sniff mypod --image alpine -it -- bash
```

If you omit the command after `--`, `sniff` only adds the ephemeral container and then
prints a `kubectl exec` command you can run later.

If you start the ephemeral container with a command that never exits, such as
`sleep infinity`, it will remain attached to the original pod until that process exits
or the pod is recreated. Use that deliberately; if you want a long-lived debug target
without modifying the original pod, prefer `--clone`.

For `--clone`, the standalone debug pod defaults to `sleep infinity` so it stays alive
for later `kubectl exec` sessions:

```bash
kubectl sniff mypod --clone --image alpine
kubectl exec -it <debug-pod-name> -- sh
```

Use `--command` and repeated `--arg` flags to override that default, or set
`SNIFF_RUN_COMMAND` and `SNIFF_RUN_ARGS` for a shell-profile default.

## Safety notes

- Copying `envFrom` may expose config and secret references inside the debug container.
- Copying service account mounts gives the debug container access to the pod's service account token.
- A standalone debug pod is a separate pod, not an ephemeral container inside the original pod.
- An attached ephemeral container shares more context with the original pod, but it still depends on cluster support for ephemeral containers.

## Quick help

```bash
kubectl sniff --help
kubectl sniff attach --help
kubectl sniff run --help
```

## License

This project is licensed under the Apache License 2.0. See [LICENSE](./LICENSE).
