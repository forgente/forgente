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
the first *directional* phase: Forgente becomes the forge where agents are
first-class principals, on infrastructure the operator owns. The parity
target is GitHub's agent surface as surveyed in July 2026 — cloud agent,
agentic workflows, agent sessions, custom agents, MCP, code review.

**Reach parity first, then extend.** GitHub has already paid the
product-design cost of this category — what an agent may do, what it must
not, what configuration looks like, what a session is. That work is more
expensive and riskier than the code. So their surface is treated as a proven
specification to implement, not a set of ideas to improve on, and the
novelty budget is spent afterwards. Four rules make that workable:

- **The target is frozen.** GitHub ships faster than we can build, so parity
  against a live target is permanently unreachable. The capability spec in
  the program document is a dated snapshot; anything shipped after it waits
  for the next re-survey rather than reopening scope mid-flight.
- **"All capabilities" is smaller than it sounds.** Client tooling, model
  hosting, and the entire billing surface are excluded, which leaves roughly
  six forge-side areas. Naming the exclusions is what makes the goal
  finishable.
- **GitHub's semantics, the ecosystem's spelling.** Where the two conflict,
  take behaviour, defaults and guardrails from GitHub, and names, paths and
  wire formats from the Gitea ecosystem, so `tea`, the SDKs and the runner
  keep working. The Actions token-permission system already does exactly
  this: GitHub's `permissions:` semantics under `GITEA_TOKEN`.
- **Our differentiators are constraints, not a later phase.** Self-hosting,
  no subscription gate, open formats and owned agent identity are properties
  of *how* each parity item gets built. L0 is the proof — agent apps are a
  parity item and owned identity is the differentiator, shipped together.

Design and rationale live in
[docs/agent-native-program.md](docs/agent-native-program.md), including the
frozen capability spec, the exclusions and their reasons, and the method for
re-running the survey before any layer is scoped in detail. The layers, in
build order, each shippable on its own:

- **L0 — installation identity, agents as its first consumer.** *Shipped in
  #66.* The real gap was not "agents": Forgente had no per-installation
  principal at all (OAuth2 apps act as the authorizing user).
  Organization-owned agent accounts, permissions through ordinary team
  membership, provenance badges, an organization-wide kill switch. Upstream is
  activating bot accounts at the admin level in go-gitea/gitea#38181 and
  explicitly defers organization ownership; Forgente builds the deferred layer
  and cherry-picks the rest. Demand is long-standing and quantified (upstream
  #25900, #13044, #26754, #33469).
- **L1 — connecting agents, not a server.** *Shipped in #71, #72 and #73.*
  Do not fork `gitea-mcp`: it was validated unforked against Forgente as an
  organization-owned app, and upstream independently settled on keeping the
  server standalone. The forge work was authorization, and reading the MCP
  specification rather than paraphrasing it shrank the layer twice: the
  resource-server half (RFC 9728, the `401` challenge) belongs to the MCP
  server and is an upstream contribution, and Dynamic Client Registration is
  optional and on its way out, so it will not be built. What shipped is the
  forge's half — RFC 8414 authorization-server metadata advertising the API
  scopes, RFC 8707 resource indicators bound to the token audience so a
  resource server can validate a token was issued for it, and a connect
  panel. One tension is
  recorded rather than resolved: `gitea-mcp` forwards the caller's token to
  the API, which the specification forbids, and that is upstream's design
  question to settle.
- **L2 — repository agent configuration and model providers.** Agent
  definitions beside `AGENTS.md`, following the `.agents/skills/` convention
  the ecosystem already uses; provider endpoint and credentials at the
  instance and organization level.
- **L3 — agent sessions and the sandbox contract.** Assigning an issue or
  review to an agent dispatches an Actions run behind a session record with
  live logs, steering, and links to what it produced. Actions and the runner
  fleet are already the sandbox, and its token-permission system already
  delivers least privilege. The rest of the contract — egress control, no
  self-approval or self-merge, attributable sessions, propose-and-approve for
  low-confidence actions — is a precondition for the layer, not later polish.
  Egress control is the piece with nothing behind it yet, and checking the
  runner protocol showed why: it carries no network field in either
  direction, so the forge cannot enforce egress at all. What it can offer is
  routing agent sessions to runners an operator has designated as
  network-restricted, and refusing to dispatch when none exists —
  attestation, not enforcement, and the program document says so in those
  words.
- **L4 — tenants.** AI code review, issue triage, pull-request summaries —
  built as agents on L0–L3 in their own repositories, not compiled into the
  server.

Non-goals are listed in the program document and are as load-bearing as the
goals: no editor tooling, no hosted inference, no agent marketplace or plugin
economy, no new permission system, and no attempt at parity across every
surface GitHub ships.

## Standing rule — security

Forgente serves git over the network. Whatever the phase, there is never a
state where upstream security fixes are neither merged nor consciously
triaged. This rule outranks every other item in this document.
