# Forgente operations

This is the operational handbook for Forgente
([github.com/forgente/forgente](https://github.com/forgente/forgente), home at
[forgente.com](https://forgente.com)): how the project tracks its upstream,
ships releases, and runs its live properties. For what Forgente is and where
it is heading, see [README.md](README.md) and [ROADMAP.md](ROADMAP.md).

Forgente builds on [Gitea](https://github.com/go-gitea/gitea) and tracks it as
an upstream: upstream changes are merged regularly while Forgente carries its
own commits on top. The GitHub repository is a regular repository (not a
GitHub fork), so it has no "forked from" association.

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

Syncing is automated: a daily scheduled agent (06:00, maintainer-side) fetches
`upstream`, opens a `chore: sync upstream gitea` PR when `main` has new upstream
commits, merges it with a **merge commit once checks pass — sync PRs are never
squashed** (squashing flattens upstream history and makes every future sync
re-conflict), fast-forwards `release/v*` branches and tags, and cancels
release-branch nightlies stuck on upstream's private runners. The same daily run
also sweeps the ecosystem forks (docs, blog, helm-forgente, homebrew-forgente),
merging their gitea.com upstreams with merge commits; conflicts there keep the
Forgente identity surfaces (chart appVersion/icon, docusaurus title/urls,
`.github/` workflows, formula versions). Feature/fix PRs, by contrast, are
squash-merged with the `(#N)` title suffix, matching upstream convention.

The daily run also watches upstream release **tags**: branches sync silently,
but a release is triggered by a tag, so any new stable `vX.Y.Z` upstream tag
without a matching Forgente `vX.Y.Z-N` release gets an issue opened in this
repo with the release checklist ("upstream tagged vX.Y.Z — cut vX.Y.Z-1").
Release branches themselves are not merged into `forgente/vX.Y` between
releases — drift there is free until upstream tags, and the tag-time merge is
near-guaranteed clean because our release-line commits are additive.

For a manual sync, the helper script does the same:

```bash
contrib/forgente/sync-upstream.sh
git push origin main --tags
```

Merge conflicts, when they happen, are in the files Forgente has modified — the
list is at the end of the "Rebranding state" section. To keep merges tractable,
prefer additive changes (new files under `contrib/forgente/`, new packages) over
edits to upstream files where possible.

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
`RELEASE_TOKEN` (PAT used to create GitHub releases, as upstream does), and
`SNAPCRAFT_STORE_CREDENTIALS` (publishes the `forgente` snap to `latest/edge`;
the store name is registered, listing goes public after Canonical's review).

Differences from upstream: release jobs run on GitHub-hosted `ubuntu-latest`
runners instead of Gitea's Namespace runners, so container images are
`linux/amd64` + `linux/arm64` (riscv64 is too slow under QEMU on standard
runners — restore it in the `release-*` workflows if faster runners are ever
configured).

### Shipping a tagged release

Forgente versions are `v<upstream version>-<forgente release>`, e.g.
`v1.26.4-1` is the first Forgente release of Gitea 1.26.4 (the Forgejo
convention). Upstream's own tags are mirrored into this repository by the
daily sync, so plain `vX.Y.Z` names are taken — and pushing a mirrored
upstream tag is harmless by design: it runs the workflow file *at that tag*,
which targets upstream's private runners and just queues (cancel it).

Release procedure:

1. Branch `forgente/vX.Y` from the upstream tag being shipped (NOT
   `release/vX.Y*` — the sync routine owns `release/v*` as pure upstream
   mirrors, and `release-nightly` triggers on that pattern). Cherry-pick the
   rebrand commits (the conflict-file list below is the checklist) onto it.
2. Dry-run with an rc tag first (`vX.Y.Z-N-rc1`): `release-tag-rc` builds
   binaries and rc images and creates a *draft* GitHub release — delete the
   draft after verifying.
3. Push an **annotated** tag `vX.Y.Z-N` whose message is the release notes
   (`gh release create --notes-from-tag`): "based on Gitea X.Y.Z" + link to
   upstream's changelog + the Forgente delta.
4. `release-tag-version` publishes binaries to GitHub + S3 and container
   images tagged `latest`, `<major>`, `<major.minor>`, `<upstream version>`,
   `<full version>` (+ `-rootless`). The workflow uses `type=match` docker
   tag patterns because semver types would treat the `-N` suffix as a
   prerelease and skip `latest`/major/minor.
5. Fan out: `forgente/deployment` version.json (update checker),
   `homebrew-forgente` versioned formula, `helm-forgente` chart pin, and
   pin the forgente.com instance to the new tag (compose image bump —
   procedure in the forgente/infra runbook).
6. Snap: dispatch the `release-snap-stable` workflow with the tag — it
   builds the tag's source with main's `snap/` scripts (the vX.Y.Z-N
   grade/version handling lives there) and publishes to `latest/candidate`
   (read back by `snap/part-gitea-pull.sh` as the last-released marker)
   and `latest/stable`.

## Live properties

| URL | What | Source of truth |
| ---- | ---- | ---- |
| https://forgente.com | hosted instance (anonymous visitors redirect to about) | forgente/infra (private runbook) |
| https://about.forgente.com | landing page | served from the instance host |
| https://docs.forgente.com | documentation | [forgente/docs](https://github.com/forgente/docs) |
| https://blog.forgente.com | blog (infra live; content policy pending — currently builds upstream posts) | [forgente/blog](https://github.com/forgente/blog) |
| https://dl.forgente.com | signed binaries + `forgente/version.json` (update checker) + `charts/` (helm repo index) | release workflows + [forgente/deployment](https://github.com/forgente/deployment) + [forgente/helm-forgente](https://github.com/forgente/helm-forgente) |

The private [forgente/infra](https://github.com/forgente/infra) repo holds the
server provisioning, CDN/DNS/cert inventory, and operational lessons.

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
- Default `APP_NAME` and `[ui.meta]` author/description/keywords are Forgente
  (`modules/setting/server.go`, `modules/setting/ui.go`, `app.example.ini`).
- Logo/favicon are a placeholder Forgente mark (`assets/logo.svg`,
  `assets/favicon.svg`; regenerate derived files with `make generate-images`).
  Replace with a real brand design later — same two files + regenerate.
  `public/assets/img/gitea.svg` stays Gitea's mark on purpose (it represents
  Gitea as an external service in migration screens).
- Not yet rebranded: Go module path (`code.gitea.io/gitea` — deep fork
  territory, avoid).

Files that now differ from upstream and may conflict on sync (re-apply the
same renames): the 3 `release-*` workflows (see above), `Makefile`
(`EXECUTABLE`, `-out forgente-$(VERSION)`, `forgente-src-`, docs target),
`Dockerfile` + `Dockerfile.rootless` (two `/go/src/gitea.dev/forgente` lines
each), `.air.toml`, `.gitignore` (`/forgente`), `snap/*`,
`modules/setting/testenv.go` (AppPath), `modules/setting/server.go`
(APP_NAME default), `modules/setting/ui.go` (meta defaults),
`custom/conf/app.example.ini`, `services/cron/tasks_extended.go`
(update-checker endpoint), `web_src/js/modules/favicon-status.test.ts`
(asserts the Forgente favicon color).

## Gitea ecosystem tools

The Gitea ecosystem is hosted mostly at [gitea.com/gitea](https://gitea.com/gitea)
(exception: [giteabot](https://github.com/go-gitea/giteabot) lives on GitHub,
since it operates on GitHub's PR/label APIs) and talks to the server through
its API. Because Forgente stays API-compatible with Gitea, **upstream tools
work against Forgente unforked** — the strategy is fork-on-divergence, not
fork-in-advance. Per tool:

Complete disposition of all active gitea.com/gitea repos (audited 2026-07-11):

**Forked under the forgente org:** `gitea` (this repo),
`docs` → [forgente/docs](https://github.com/forgente/docs) (docs.forgente.com),
`blog` → [forgente/blog](https://github.com/forgente/blog) (blog.forgente.com;
forked 2026-07-16 ahead of its first-post trigger to stand up the S3+CloudFront
publish infra — Forgente-only content policy still to apply),
`helm-gitea` → [forgente/helm-forgente](https://github.com/forgente/helm-forgente),
`homebrew-gitea` → [forgente/homebrew-forgente](https://github.com/forgente/homebrew-forgente),
and `infrastructure`/`deployment` → forgente/infra (private; includes the
dl CDN, mirroring upstream's `infrastructure/dl-gitea-com`).

**Work against Forgente unforked (fork trigger: API divergence):** `tea`,
`go-sdk`, `sdk.js`, `terraform-provider-gitea` (can also terraform the
forgente.com instance itself), `gitea-mcp`, `runner` (act_runner — Actions
protocol change is its trigger), `git-lfs-transfer`, `gitea-mirror`,
`importer`, `daggerverse-gitea`.

**Internal build deps, consumed as-is:** `actions-proto-def`,
`actions-proto-go`, `go-xsd-duration`, `go-fed-activity`, `runner-images`,
`renovate-config`.

**Trigger-based, not yet forked:**

| Repo | Trigger |
| ---- | ---- |
| `changelog` | consciously deferred at v1.26.4-1: release notes live in the annotated tag; fork when Forgente's own PR volume justifies generated notes |
| `design` | a real Forgente logo/brand exists |
| `awesome-gitea` | community exists |
| `government` | enterprise/compliance docs needed |
| `giteabot` (GitHub) | own release branches + contributor-scale PR volume; interim: generic backport action; needs bot account + hosting |

**Site plumbing where Forgente differs by design:** `gitea.com`, `redirects`,
`website-pr-preview`, `pr-deployer` — replaced by our CloudFront
distributions and the docs repo's publish Action. Test fixtures
(`test-openldap`, `test_repo`, `*-skill`) are consumed by CI as-is.

When a trigger fires, follow the same upstream-tracking playbook as this
repository:
regular repo (no fork relation), `origin` = forgente, `upstream` = gitea.com
source, CI adapted by pure substitutions, sync via merge commits.

## Development

Everything from upstream applies unchanged — see [README.md](README.md) and
[CONTRIBUTING.md](CONTRIBUTING.md). Quick reference: `make help`.
