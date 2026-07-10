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

The release pipeline is adapted from Gitea's:

- **Nightly** (`release-nightly`): every push to `main` or `release/v*` builds
  cross-platform binaries (uploaded as workflow artifacts) and pushes container
  images `ghcr.io/forgente/forgente:{nightly,<major.minor>-nightly}` plus
  `-rootless` variants.
- **Release candidate** (`release-tag-rc`): pushing a `v1*-rc*` tag builds
  binaries, creates a draft GitHub release, and pushes rc container images.
- **Version release** (`release-tag-version`): pushing a `v1.*` tag builds
  binaries, creates a GitHub release with them attached, and pushes container
  images tagged `latest`, `<major>`, `<major.minor>`, `<version>`.

The pipeline mirrors upstream's; external integrations activate automatically as
soon as the matching repository secrets are added (Settings → Secrets and
variables → Actions), and are skipped cleanly while unset:

| External account | Secrets to add | What activates |
|------------------|----------------|----------------|
| GPG release key | `GPGSIGN_KEY`, `GPGSIGN_PASSPHRASE` | `.asc` detached signatures on release binaries (cosign/sigstore signing always runs) |
| AWS S3 bucket (dl.forgente.com) | `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_S3_BUCKET` | binary uploads to `s3://<bucket>/forgente/<version|branch-nightly>` |
| Docker Hub org `forgente` | `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN` | container images also pushed to `docker.io/forgente/forgente` |
| Release PAT (optional) | `RELEASE_TOKEN` | GitHub releases created by that identity instead of the built-in token |
| Snap store (register name `forgente`) | `SNAPCRAFT_STORE_CREDENTIALS` | then enable the workflow: `gh workflow enable release-nightly-snapcraft.yml` (currently disabled; the snap is renamed to `forgente` in `snap/snapcraft.yaml`) |

Remaining differences from upstream: release jobs run on GitHub-hosted
`ubuntu-latest` runners instead of Gitea's Namespace runners, so container
images are `linux/amd64` + `linux/arm64` (riscv64 is too slow under QEMU on
standard runners — restore it in the `release-*` workflows if faster runners
are ever configured), and nightly binaries are additionally uploaded as
workflow artifacts so they are downloadable without S3.

After the first container publish, make the `forgente` package public once under
https://github.com/orgs/forgente/packages (GHCR packages start private).

## Development

Everything from upstream applies unchanged — see [README.md](README.md) and
[CONTRIBUTING.md](CONTRIBUTING.md). Quick reference: `make help`.
