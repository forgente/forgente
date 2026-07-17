# Forgente roadmap

Forgente is an independent, fully hackable software forge: its own brand,
infrastructure, releases — and, the part being built now, its own
differentiating features. The repository is completely hackable today: any
feature, route, model, or UI change can be built now.

[Gitea](https://github.com/go-gitea/gitea) is Forgente's starting point and
current upstream. Tracking it is an engineering choice, not an identity:
daily merges keep upstream's fixes and security patches flowing for free
while Forgente builds what makes it worth existing. The only thing this
choice constrains is the renaming of upstream identifiers (paths, env vars,
module names), because that is what keeps the merges cheap. When divergence
becomes worth more than upstream's flow of fixes, Forgente cuts over to a
hard fork (Phase 2) — deliberately, not by drift.

## Phase 0 — infrastructure parity (done)

- Standalone repository (no GitHub fork relation), full Gitea history
- Release pipeline equal to upstream's: signed binaries to S3, containers to
  Docker Hub + GHCR, snap, GitHub releases — under Forgente accounts
- Daily automated upstream sync (merge commits, never squash)
- Build identity rebranded: `forgente` binary, `forgente-*` artifacts,
  `forgente` snap command, compat shims for everything else
- Branch protection, PR workflow, local dev environment

## Phase 1 — differentiate while tracking upstream (current)

The buildout is complete: first tagged release shipped (`v1.26.4-1`), live
properties up (forgente.com, dl.forgente.com, docs.forgente.com), brand
applied at the edges, sync automation running. Forgente's own value — the
features that make it more than its starting point — is built here, and
these rules of engagement remain the standing operating mode until a Phase 2
trigger fires:

- **Additive code.** New features live in new packages/files where possible
  (e.g. `forgente/`-namespaced modules, new routes, feature flags). Bounded
  divergence keeps the eventual cutover mechanical.
- **Compat shims over renames.** Runtime surface stays Gitea-compatible:
  container internals (`/app/gitea/`, `GITEA_*` env), fixture hooks
  (`gitea -> forgente` build symlink), Go module path (`code.gitea.io/gitea`).
  Every shim is documented in [FORGENTE.md](FORGENTE.md).
- **Sync is sacred.** The daily upstream sync keeps running; a sync PR that
  rots for a week is a process failure. Upstream security fixes arrive
  through it — Forgente does not yet have its own security triage.
- **Brand at the edges.** Everything users see can become Forgente without
  merge cost: logo (placeholder Forgente mark shipped; the final design swaps
  the same two asset files — Gitea's name and logo are upstream trademarks
  and stay out), `APP_NAME` default, forgente.com, dl.forgente.com, docs,
  screenshots, community channels.

## Phase 2 trigger — when to hard fork

Cut over when **any** of these holds:

1. Features need such broad changes to core models/services that weekly
   upstream merges cost more than they deliver.
2. Upstream makes a structural decision Forgente must not follow.
3. The project has the capacity (maintainers, process) to triage Gitea
   security advisories and patch independently.

Until a trigger fires, resist partial hard-forking: half-renamed internals
pay both costs (merge friction and shim upkeep) and get neither benefit.

## Phase 2 — hard-fork cutover checklist

Executed as one deliberate campaign, roughly a week of work if Phase 1
discipline held:

1. Rename the Go module path (`code.gitea.io/gitea` → a forgente.com path)
   and all imports; drop the `gitea.dev` replace aliases.
2. Remove the compat shims: `gitea` build symlink, container layout
   (`/app/gitea/` → `/app/forgente/`, `GITEA_*` → `FORGENTE_*` env with a
   compatibility fallback window for users), docker scripts, s6 service names.
3. Regenerate test fixtures (`tests/gitea-repositories-meta` hooks) for the
   new names.
4. Mass rebrand internals: config defaults, UI strings, templates, docs.
5. Publish a user migration guide (env var mapping, volume paths, image tags).
6. Flip the sync routine from "merge everything daily" to "watch Gitea
   security advisories and cherry-pick fixes" — and subscribe to their
   advisory feed before the flip, not after.
7. Re-evaluate the ecosystem table in FORGENTE.md — API divergence is the
   fork trigger for tea, runner, helm chart, and SDK.

## Standing rule — security

Forgente serves git over the network. Whatever the phase, there is never a
state where upstream security fixes are neither merged nor consciously
triaged. This rule outranks every other item in this document.
