# Forgente

Forgente is a soft fork of [Gitea](https://github.com/go-gitea/gitea), hosted at
[github.com/forgente/forgente](https://github.com/forgente/forgente) with its home
at [forgente.com](https://forgente.com).

"Soft fork" means: Forgente tracks upstream Gitea closely and regularly merges its
changes, while carrying its own commits on top. The GitHub repository is a regular
repository (not a GitHub fork), so it has no "forked from" association.

## Repository layout

| Remote     | URL                                        | Purpose                     |
|------------|--------------------------------------------|-----------------------------|
| `origin`   | https://github.com/forgente/forgente.git   | Forgente (this project)     |
| `upstream` | https://github.com/go-gitea/gitea.git      | Gitea (sync source)         |

Clone and set up:

```bash
git clone https://github.com/forgente/forgente.git
cd forgente
git remote add upstream https://github.com/go-gitea/gitea.git
```

## Syncing from upstream Gitea

Run the helper script:

```bash
contrib/forgente/sync-upstream.sh
```

It fetches `upstream`, merges `upstream/main` into `main`, and syncs tags and
release branches. Resolve any merge conflicts (they will be in files Forgente has
modified), then push:

```bash
git push origin main --tags
```

To keep merges tractable, prefer additive changes (new files under `contrib/forgente/`,
new packages) over edits to upstream files where possible.

## CI/CD and releases

CI works like upstream Gitea: the `pull-*` workflows (compliance, db tests, e2e,
docker dry-run) run on every pull request on standard GitHub runners with no
secrets needed. Gitea-specific automation (giteabot, crowdin translations,
renovate, license crons) is guarded by `if: github.repository == 'go-gitea/gitea'`
upstream and skips automatically here.

The release pipeline is upstream Gitea's with Forgente substitutions:

- **Nightly** (`release-nightly`): every push to `main` or `release/v*` builds
  cross-platform binaries (signed, uploaded to S3) and pushes container images
  `forgente/forgente` (Docker Hub) and `ghcr.io/forgente/forgente` tagged
  `{nightly,<major.minor>-nightly}` plus `-rootless` variants.
- **Release candidate** (`release-tag-rc`): pushing a `v1*-rc*` tag builds
  binaries, creates a draft GitHub release, and pushes rc container images.
- **Version release** (`release-tag-version`): pushing a `v1.*` tag builds
  binaries, creates a GitHub release with them attached, and pushes container
  images tagged `latest`, `<major>`, `<major.minor>`, `<version>`.

Configured repository secrets (Settings → Secrets and variables → Actions):
`GPGSIGN_KEY`/`GPGSIGN_PASSPHRASE` (release signing key
`67129BAD57A2C8D2186032489D6FD2FD6E0B9BA5`, `Forgente <maintainers@forgente.com>`),
`DOCKERHUB_USERNAME`/`DOCKERHUB_TOKEN` (Docker Hub account `forgente`),
`AWS_REGION`/`AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`/`AWS_S3_BUCKET`
(bucket `forgente-dl` in `eu-central-1`, IAM user `forgente-release-ci`,
binaries land under `s3://forgente-dl/forgente/<version or branch-nightly>`),
and `RELEASE_TOKEN` (PAT used to create GitHub releases, as upstream does).

Differences from upstream: release jobs run on GitHub-hosted `ubuntu-latest`
runners instead of Gitea's Namespace runners, so container images are
`linux/amd64` + `linux/arm64` (riscv64 is too slow under QEMU on standard
runners — restore it in the `release-*` workflows if faster runners are ever
configured). The snapcraft workflow is disabled and the snap renamed to
`forgente`; register the name on the Snap store, add
`SNAPCRAFT_STORE_CREDENTIALS`, then run
`gh workflow enable release-nightly-snapcraft.yml`.

After the first container publish, make the `forgente` package public once under
https://github.com/orgs/forgente/packages (GHCR packages start private).

## Rebranding state

The build identity is Forgente; the runtime/compat surface deliberately stays
Gitea's:

- Binary and release artifacts are named `forgente` / `forgente-<version>-*`
  (`EXECUTABLE` in the Makefile, xgo `-out`, src tarball, man page, `.air.toml`).
- The snap installs and exposes a `forgente` command.
- Container images keep the upstream-compatible internals on purpose:
  `/app/gitea/gitea` path, `gitea` wrapper on PATH, `GITEA_*` environment
  variables, s6 service names, volumes. Existing Gitea container setups work
  unchanged, and `docker/` needs no fork-side edits.
- Not yet rebranded: logo and UI branding (needs a Forgente logo — note
  Gitea's name/logo are upstream trademarks), config defaults (`APP_NAME`),
  Go module path (`code.gitea.io/gitea` — deep fork territory, avoid).

Files that now differ from upstream and may conflict on sync (re-apply the
same renames): the 3 `release-*` workflows (see above), `Makefile`
(`EXECUTABLE`, `-out forgente-$(VERSION)`, `forgente-src-`, docs target),
`Dockerfile` + `Dockerfile.rootless` (two `/go/src/gitea.dev/forgente` lines
each), `.air.toml`, `.gitignore` (`/forgente`), `snap/*`.

## Gitea ecosystem tools

The Gitea ecosystem is hosted mostly at [gitea.com/gitea](https://gitea.com/gitea)
(exception: [giteabot](https://github.com/go-gitea/giteabot) lives on GitHub,
since it operates on GitHub's PR/label APIs) and talks to the server through
its API. Because Forgente stays API-compatible with Gitea, **upstream tools
work against Forgente unforked** — the strategy is fork-on-divergence, not
fork-in-advance. Per tool:

| Tool | Works with Forgente today | Fork trigger |
| ---- | ---- | ---- |
| `tea` (CLI) | yes — point it at a Forgente instance | Forgente-specific API additions or branding requirements |
| `runner` (act_runner) | yes — registers against Forgente Actions | changes to the Actions protocol |
| `helm-gitea` | yes — override `image.repository=forgente/forgente` in values | wanting a published `forgente` chart with our defaults (earliest sensible fork) |
| `go-sdk` | yes — API-compatible | API divergence (also implies maintaining a `code.forgente.com`-style module path) |
| `terraform-provider-gitea` | yes | API divergence |
| `giteabot` | n/a — automates go-gitea/gitea team workflow (backports, merge queue, labels); its in-repo workflows skip here via repo guard | own release branches + contributor-scale PR volume; interim: a generic backport GitHub Action; prerequisites: bot account + hosting (it is a deployed service, not just a repo) |

When a trigger fires, follow the same soft-fork playbook as this repository:
regular repo (no fork relation), `origin` = forgente, `upstream` = gitea.com
source, CI adapted by pure substitutions, sync via merge commits.

## Development

Everything from upstream applies unchanged — see [README.md](README.md) and
[CONTRIBUTING.md](CONTRIBUTING.md). Quick reference: `make help`.
