# Forgente operations

This is the operational handbook for Forgente
([github.com/forgente/forgente](https://github.com/forgente/forgente), home at
[forgente.com](https://forgente.com)): how the project tracks its upstream,
ships releases, and runs its live properties. For what Forgente is and where
it is heading, see [README.md](README.md) and [ROADMAP.md](ROADMAP.md).

Forgente builds on [Gitea](https://github.com/go-gitea/gitea). Since the
Phase 2 hard-fork cutover (2026-07, see [ROADMAP.md](ROADMAP.md)) Forgente is
an independent codebase: upstream is no longer merged wholesale; instead,
Gitea security advisories and patch releases are watched and relevant fixes
are cherry-picked. The GitHub repository is a regular repository (not a
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

## Tracking upstream Gitea after the hard fork

Forgente stopped merging upstream wholesale at the Phase 2 cutover
(2026-07-21; the last full sync was upstream main as of #47). The standing
security rule in [ROADMAP.md](ROADMAP.md) still applies — upstream security
fixes are never ignored — but the mechanism changed:

- A daily scheduled agent (maintainer-side) watches go-gitea/gitea GitHub
  **security advisories** and new upstream **patch-release tags** on release
  lines Forgente has shipped from. Each new advisory or patch tag gets an
  issue in this repo (`security: candidate cherry-pick — <id/tag>`) linking
  the upstream commits. Nothing is cherry-picked automatically.
- A human triages the issue and runs the pick helper on a branch:

```bash
contrib/forgente/pick-upstream.sh <upstream-sha>...
```

Upstream commits still use the `gitea.dev` module path internally, so picks
touching import lines conflict mechanically; the helper rewrites the module
path (`gitea.dev/` → `forgente.com/`, sparing the external `gitea.dev/sdk`
and `gitea.dev/actions-proto-go` modules) and leaves real conflicts for
manual resolution. Cherry-pick PRs follow the normal feature-PR flow
(squash-merged with the `(#N)` title suffix).

The same daily run still sweeps the ecosystem forks (docs, helm-forgente,
homebrew-forgente), which remain soft forks of their gitea.com upstreams:
merged with merge commits, conflicts keep the Forgente identity surfaces
(chart appVersion/icon, docusaurus title/urls, `.github/` workflows, formula
versions). The blog is **no longer** in this sweep — as of 2026-07-22 it
publishes Forgente-native content only (see the blog table row below), so it
is a normal Forgente repo, not a soft fork.

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

Forgente versions are Forgente-native semver starting at **v2.0.0** (decided
2026-07-21 with the hard fork): the pre-fork `v<upstream>-<N>` releases
(`v1.26.4-1`, `v1.27.0-1`) remain valid history, and v2.0.0 will be the
first post-fork release — a semver **major** on purpose, since the cutover
changes operator-facing surfaces (see
[docs/migration-hard-fork.md](docs/migration-hard-fork.md)). The v1.x tag
namespace is retired: upstream tags mirrored before the cutover live there,
so plain `v1.Y.Z` names are taken — pushing such a mirrored tag is harmless
by design: it runs the workflow file *at that tag*, which targets upstream's
private runners and just queues (cancel it).

Release procedure:

1. Branch `forgente/vX.Y` (e.g. `forgente/v2.0`) from `main` at the chosen
   release point (NOT `release/vX.Y*` — those are frozen pre-fork upstream
   mirrors, and `release-nightly` triggers on that pattern). Main is the
   single source of truth now; there are no rebrand commits to cherry-pick.
2. Dry-run with an rc tag first (`vX.Y.Z-rc1`): `release-tag-rc` builds
   binaries and rc images and creates a *draft* GitHub release — delete the
   draft after verifying.
3. Push an **annotated** tag `vX.Y.Z` whose message is the release notes
   (`gh release create --notes-from-tag`): the Forgente changelog, the
   upstream baseline it forked from (for v2.0.0: Gitea main as of
   2026-07-21 / #47), and for v2.0.0 a prominent link to the migration
   guide.
4. `release-tag-version` publishes binaries to GitHub + S3 and container
   images tagged `latest`, `<major>`, `<major.minor>`, `<full version>`
   (+ `-rootless`). The workflow keeps `type=match` docker tag patterns
   (they cover both the new plain-semver tags and the historical `-N`
   suffixed ones, which semver types would treat as prereleases).
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
| https://blog.forgente.com | blog (Forgente-native content only since 2026-07-22; no longer tracks the upstream Gitea blog) | [forgente/blog](https://github.com/forgente/blog) |
| https://dl.forgente.com | signed binaries + `forgente/version.json` (update checker) + `charts/` (helm repo index) | release workflows + [forgente/deployment](https://github.com/forgente/deployment) + [forgente/helm-forgente](https://github.com/forgente/helm-forgente) |

The private [forgente/infra](https://github.com/forgente/infra) repo holds the
server provisioning, CDN/DNS/cert inventory, and operational lessons.

## Rebranding state (post-hard-fork)

Since the Phase 2 cutover, build identity AND runtime surface are Forgente:

- Binary and release artifacts are named `forgente` / `forgente-<version>-*`;
  the snap installs a `forgente` command.
- Go module path is `forgente.com` (bare-domain, gitea.dev-style; the
  go-import vanity meta at forgente.com only matters for `go get`, not
  builds).
- Containers: binary at `/app/forgente/forgente`, `forgente` wrapper primary;
  compat shims kept for a deprecation window (`/app/gitea` symlink, `gitea`
  wrapper shim). Rootless image volumes are `/var/lib/forgente` +
  `/etc/forgente`; the root image keeps `/data/gitea` paths so existing
  volumes work.
- Env: `FORGENTE_*` primary, legacy `GITEA_*` honored with a one-time
  deprecation warning; git-hook env dual-emitted under both prefixes
  (kept forever for users' custom hooks). See
  [docs/migration-hard-fork.md](docs/migration-hard-fork.md) for the full
  mapping.
- Server-side delegate hooks are `hooks/<name>.d/forgente`; regeneration
  removes the legacy `.d/gitea` file (`admin regenerate hooks`).
- Default `APP_NAME` and `[ui.meta]` are Forgente; logo/favicon are a
  placeholder Forgente mark (`assets/logo.svg`, `assets/favicon.svg`;
  regenerate derived files with `make generate-images`).
  `public/assets/img/gitea.svg` stays Gitea's mark on purpose (it represents
  Gitea as an external service in migration screens).

**Deliberately kept Gitea-compatible (wire/ecosystem surface — do not
"finish" these renames):** the API routes and `X-Gitea-*` headers, webhook
type `gitea` and payload shape, `GITEA_TOKEN` runner secret, the `GITEA__*`
config-override prefix (accepted alongside `FORGENTE__*`), and the
`ONLY_ALLOW_PUSH_IF_GITEA_ENVIRONMENT_SET` ini key. This is what keeps the
unforked ecosystem tools working (see the table below). Each such site
carries an intent comment in code.

Cherry-picks from upstream conflict mechanically on the module path
(`gitea.dev/...` imports) — `contrib/forgente/pick-upstream.sh` handles that
rewrite. An upstream fix can also compile-break against Forgente's diverged
APIs with no textual conflict; cherry-pick PRs must build (`go build ./...`)
before merge — CI enforces this.

## Gitea ecosystem tools

The Gitea ecosystem is hosted mostly at [gitea.com/gitea](https://gitea.com/gitea)
(exception: [giteabot](https://github.com/go-gitea/giteabot) lives on GitHub,
since it operates on GitHub's PR/label APIs) and talks to the server through
its API. Because Forgente stays API-compatible with Gitea, **upstream tools
work against Forgente unforked** — the strategy is fork-on-divergence, not
fork-in-advance. Re-checked at the Phase 2 hard-fork cutover (2026-07-21):
the cutover renamed identity/runtime surfaces only, not the API, so the
table stands unchanged. Per tool:

Complete disposition of all active gitea.com/gitea repos (audited 2026-07-11):

**Forked under the forgente org:** `gitea` (this repo),
`docs` → [forgente/docs](https://github.com/forgente/docs) (docs.forgente.com),
`blog` → [forgente/blog](https://github.com/forgente/blog) (blog.forgente.com;
forked 2026-07-16 to stand up the S3+CloudFront publish infra, then cut over to
Forgente-native content only on 2026-07-22 — the inherited upstream posts were
removed and it dropped out of the daily sync, so it no longer tracks
gitea.com/gitea/blog),
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
