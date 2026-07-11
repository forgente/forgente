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

## Development

Everything from upstream applies unchanged — see [README.md](README.md) and
[CONTRIBUTING.md](CONTRIBUTING.md). Quick reference: `make help`.
