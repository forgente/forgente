# Forgente roadmap

Forgente is an independent, fully hackable software forge: its own brand,
infrastructure, releases — and, the part being built now, its own
differentiating features. The repository is completely hackable today: any
feature, route, model, or UI change can be built now.

[Gitea](https://github.com/go-gitea/gitea) is Forgente's starting point.
Through Phases 0–1 it was tracked as a daily-merged upstream; at the Phase 2
cutover (2026-07-21) Forgente became a hard fork — deliberately, not by
drift. Upstream security fixes now arrive by watched advisories and
cherry-picks instead of wholesale merges, and the old constraint against
renaming upstream identifiers is gone.

## Phase 0 — infrastructure parity (done)

- Standalone repository (no GitHub fork relation), full Gitea history
- Release pipeline equal to upstream's: signed binaries to S3, containers to
  Docker Hub + GHCR, snap, GitHub releases — under Forgente accounts
- Daily automated upstream sync (merge commits, never squash)
- Build identity rebranded: `forgente` binary, `forgente-*` artifacts,
  `forgente` snap command, compat shims for everything else
- Branch protection, PR workflow, local dev environment

## Phase 1 — differentiate while tracking upstream (done)

The buildout completed: first tagged release shipped (`v1.26.4-1`), live
properties up (forgente.com, dl.forgente.com, docs.forgente.com), brand
applied at the edges, daily sync automation running, and the first Forgente
features landed additively. The Phase 1 rules of engagement (additive code,
compat shims over renames, sacred daily sync, brand at the edges) kept the
eventual cutover mechanical — and ended with Phase 2.

## Phase 2 — hard-fork cutover (executed 2026-07-21)

The cutover ran as one deliberate campaign of five stacked PRs. What shipped
(historical record; details in [FORGENTE.md](FORGENTE.md) and
[docs/migration-hard-fork.md](docs/migration-hard-fork.md)):

1. Go module path renamed to `forgente.com` (from `gitea.dev` — upstream had
   already moved off `code.gitea.io/gitea` itself in 2026-05; there were no
   replace aliases to drop, contrary to this checklist's original wording).
2. Compat shims removed with a fallback window: `gitea` build symlink gone,
   container layout `/app/forgente/` (compat symlink kept), `FORGENTE_*` env
   primary with `GITEA_*` honored + deprecation warning, docker scripts and
   s6 service renamed.
3. Test fixtures regenerated (`tests/gitea-repositories-meta` hooks call
   `forgente`; delegate hook file renamed with self-healing legacy cleanup).
4. Internals mass-rebranded: config defaults, UI strings, templates, docs.
   The API wire surface (X-Gitea headers, webhook types, GITEA_TOKEN) stays
   Gitea-compatible on purpose — that is compatibility, not unfinished work.
5. User migration guide published (docs/migration-hard-fork.md).
6. Sync routine flipped from "merge everything daily" to "watch Gitea
   security advisories + patch tags, cherry-pick fixes"
   (contrib/forgente/pick-upstream.sh).
7. Ecosystem table re-checked: no API divergence, table stands.

Version scheme (decided 2026-07-21, closing the cutover's open item):
Forgente-native semver from **v2.0.0** — the major bump signals the
operator-facing breaking changes of the cutover, and the v1.x namespace is
retired to the pre-fork `v<upstream>-<N>` releases and mirrored upstream
tags. Release mechanics in [FORGENTE.md](FORGENTE.md).

## Phase 3 — agent-native forge (current)

With infrastructure, identity, and release independence settled, Phase 3 is
the first *directional* phase: Forgente becomes the forge where AI agents are
first-class, governed principals. The parity target is GitHub's AI surface —
coding agent, code review, mission control, custom agents, MCP — with two
deliberate divergences: agents are governed rather than exempted from
governance, and the model provider is configured by the operator rather than
bundled and sold.

Design and rationale live in
[docs/agent-native-program.md](docs/agent-native-program.md). The layers, in
build order, each shippable on its own:

- **L0 — agent identity and governance.** Organization-owned agent accounts,
  permissions through ordinary team membership, provenance badges, an
  organization-wide kill switch. A 2026-07-17 spike proved this needs no
  authentication work: `UserTypeBot` already authenticates and acts.
- **L1 — first-party MCP server.** Fork `gitea-mcp` when it needs agent-token
  awareness, per the fork-on-divergence policy.
- **L2 — repository agent configuration and model providers.** Agent
  definitions beside `AGENTS.md`; provider endpoint and credentials at the
  instance and organization level.
- **L3 — agent sessions.** Assigning an issue or review to an agent
  dispatches an Actions run behind a session record with live logs, steering,
  and links to what it produced. Actions and the runner fleet are already the
  sandbox.
- **L4 — tenants.** AI code review, issue triage, pull-request summaries —
  built as agents on L0–L3 in their own repositories, not compiled into the
  server.

Non-goals are listed in the program document and are as load-bearing as the
goals: no editor tooling, no hosted inference, no agent marketplace, no new
permission system.

## Standing rule — security

Forgente serves git over the network. Whatever the phase, there is never a
state where upstream security fixes are neither merged nor consciously
triaged. This rule outranks every other item in this document.
