# Migrating an existing instance across the hard-fork cutover

Forgente completed its hard-fork cutover from Gitea (Phase 2 of
[ROADMAP.md](../ROADMAP.md)) in July 2026. Runtime surfaces that previously
kept Gitea-compatible names are now Forgente-named. This guide maps every
change for operators upgrading an existing instance. Most renames ship with a
compatibility fallback, so an unmodified deployment keeps working — but it
will log deprecation warnings until you migrate the names, and a few items
(marked **action required**) need a one-time step.

## Environment variables

Renamed, with the legacy name still honored as a fallback (a one-time
deprecation warning is logged per variable):

| New (primary) | Legacy (deprecated fallback) |
| ---- | ---- |
| `FORGENTE_WORK_DIR` | `GITEA_WORK_DIR` |
| `FORGENTE_CUSTOM` | `GITEA_CUSTOM` |
| `FORGENTE_RUN_MODE` | `GITEA_RUN_MODE` |
| `FORGENTE_I_AM_BEING_UNSAFE_RUNNING_AS_ROOT` | `GITEA_I_AM_BEING_UNSAFE_RUNNING_AS_ROOT` |
| `FORGENTE_RUNNER_REGISTRATION_TOKEN` / `..._FILE` | `GITEA_RUNNER_REGISTRATION_TOKEN` / `..._FILE` |
| `FORGENTE__section__key` config overrides | `GITEA__section__key` (both prefixes accepted) |

Emitted to subprocesses under **both** names forever (no action needed;
custom git hooks and external renderers reading `GITEA_*` keep working):
`FORGENTE_REPO_NAME`/`GITEA_REPO_NAME` and the rest of the git-hook
environment (`*_REPO_ID`, `*_PUSHER_*`, `*_PR_ID`, `*_PUSH_TRIGGER`,
`*_ROOT_URL`, …), plus `FORGENTE_PREFIX_SRC`/`GITEA_PREFIX_SRC` and
`FORGENTE_PREFIX_RAW`/`GITEA_PREFIX_RAW` for external renderers.

Intentionally **not** renamed (wire/ecosystem compatibility; do not change):

- `GITEA_TOKEN` — Actions runner authentication secret name
- `X-Gitea-*` HTTP headers and webhook payloads (`gitea` webhook type)
- `ONLY_ALLOW_PUSH_IF_GITEA_ENVIRONMENT_SET` ini key

## Container images

Image names and tags are unchanged (`forgente/forgente`,
`ghcr.io/forgente/forgente`).

**Root image** (default): the binary moved to `/app/forgente/forgente`; a
compat symlink keeps `/app/gitea/gitea` resolvable, and a `gitea` wrapper
remains on `PATH` alongside the new `forgente` wrapper. Data stays under
`/data` (including the existing `/data/gitea/...` layout — deliberately kept
so existing volumes work unchanged). No action required.

**Rootless image — action required.** The image's volumes and defaults moved:

| Old | New |
| ---- | ---- |
| `/var/lib/gitea` | `/var/lib/forgente` |
| `/etc/gitea` | `/etc/forgente` |

Volume mounts are host-side, so no code-level fallback is possible. Update
your compose/k8s mounts to the new container paths. The simplest migration is
to keep your existing host volume and just remap it:

```yaml
volumes:
  - forgente-data:/var/lib/forgente   # was .../gitea
  - forgente-config:/etc/forgente     # was /etc/gitea
```

Your data is never touched by the rename — if the instance comes up empty,
the volume is simply still mounted at the old path; fix the mapping.

Also note: the rootless image now bakes `FORGENTE_WORK_DIR` etc. as image
defaults. An old-style `-e GITEA_WORK_DIR=...` override is shadowed by those
defaults — rename such overrides to `FORGENTE_*` when upgrading.

## Git hooks and authorized_keys — action required

The server-side delegate hook file inside each repository changed name
(`hooks/<name>.d/gitea` → `hooks/<name>.d/forgente`). After upgrading, run
once:

```bash
forgente admin regenerate hooks
forgente admin regenerate keys   # only if using the authorized_keys file
```

This rewrites every repository's hooks (removing the legacy delegate file)
and the SSH `authorized_keys` command lines. Skipping it does not break
pushes — old hooks still point at the correct binary path — but
`forgente doctor check --run hooks --fix` will flag and repair repositories
until it is done.

## Init scripts / packages

`contrib/service/*` units are now named `forgente` (e.g.
`systemd/forgente.service`, launchd `com.forgente.web.plist`). For systemd:

```bash
systemctl disable --now gitea
# install contrib/service/systemd/forgente.service, then
systemctl enable --now forgente
```

The snap already installs the `forgente` command; binary release names
(`forgente-<version>-*`) are unchanged since v1.26.4-1.

## Go module path (developers)

The module is now `forgente.com` (imports like
`forgente.com/modules/setting`). Anything vendoring the old
`code.gitea.io/gitea` or `gitea.dev` module path must update imports;
`-ldflags -X` symbol paths change accordingly (see
[build-source.md](build-source.md)).

## Upstream relationship after the cutover

Forgente no longer merges upstream Gitea wholesale. Gitea security advisories
and patch releases are watched daily, and relevant fixes are cherry-picked via
`contrib/forgente/pick-upstream.sh` (see [FORGENTE.md](../FORGENTE.md)).
The API surface intentionally remains Gitea-compatible, so existing
integrations, SDKs, `tea`, and `act_runner` continue to work.
