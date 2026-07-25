# The agent-native program

Forgente's directional bet: **agents are first-class, governed principals in
the forge.** This document is the program-level design for that bet — what
gets built, in what order, and what deliberately does not get built. Phase 3
of [ROADMAP.md](../ROADMAP.md) tracks its execution; individual slices get
their own design notes and pull requests.

The parity target is GitHub's AI surface as of mid-2026 (Copilot coding
agent, Copilot code review, Agent HQ / mission control, custom agents, the
GitHub MCP server, GitHub Models). The stance is *parity in forge surface,
divergence in governance and in who supplies the model*.

## What is and is not forge work

GitHub's AI surface is one product name over three very different things.
Only the third is a forge concern.

**Client-side — not built here.** IDE completion, chat, the Copilot CLI, the
Copilot desktop app. Editors and agent CLIs already work against Forgente
repositories over ordinary git and the API; nothing in the forge gates them.
Competing on editor integration is a client race, not forge work.

**Bundled inference — inverted, not cloned.** GitHub sells Copilot seats and
runs GitHub Models, so its AI is welded to its billing. Forgente instead
configures *providers*: an operator or organization points Forgente at an
OpenAI-compatible endpoint, an Anthropic endpoint, or a local Ollama / vLLM
server, with its own credentials. A self-hosted forge that talks to any
model — including one that never leaves the network — is a better answer for
Forgente's users than a bundled model it would have to fund.

**Forge-side — the build.** Agent identity, agent sessions, repository agent
configuration, a first-party MCP server, and the features that ride on them.
The rest of this document is about those.

## Parity map

| GitHub | Forgente today | What is missing |
| ---- | ---- | ---- |
| Agent accounts (as branch-protection "bypass actors") | `UserTypeBot` exists in the schema, dormant, no UI | Product surface only — token auth, permissions, and the bot sign-in block already work |
| Coding agent: issue → sandboxed run → draft PR | Actions and a runner fleet already run untrusted workloads | Assignment, a session record, draft-PR wiring |
| Agent HQ / mission control | — | Session model plus the UI to assign, watch, steer, and stop |
| Custom agents (`.github/agents/*.agent.md`) | `AGENTS.md` is already an in-repo convention | Read agent definitions and offer them at assignment time |
| First-party MCP server | `gitea-mcp` works unforked against Forgente | Fork trigger is agent-token awareness, not API divergence |
| Copilot code review | — | A *tenant* of the layers above, not a layer itself |
| Copilot Autofix | no code scanning of any kind | Code scanning is the prerequisite; that is security work, not AI work |

## The layers

Each layer is independently shippable and useful on its own. Later layers
assume earlier ones; nothing in the list requires the layer above it.

### L0 — Agent identity and governance

Agents become organization-owned principals: created from organization
settings, permissioned through ordinary team and collaborator membership,
token-authenticated, badged wherever they act, and revocable in one place.

An empirical spike (2026-07-17, dev instance) established that this needs
**no authentication work at all**. A `UserTypeBot` row already authenticates
by access token, acts through the API and git-over-HTTP, and gains and loses
permissions through the normal membership path. Password sign-in for
non-individual users is already refused in `services/auth/signin.go` — agents
are token-only, enforced by code that already exists. That block is correct
behavior; do not "fix" it.

What remains is product surface: the creation flow, the organization
ownership link (the one new table), provenance badges on profiles, comments,
commits and pull requests, and an organization-level kill switch that
suspends every agent token at once.

### L1 — First-party MCP server

Fork `gitea-mcp` into the Forgente organization once it needs to know about
agent identity — minting and scoping tokens for an agent principal rather
than reusing a human's. Until that divergence, the upstream server works
against Forgente unmodified, consistent with the fork-on-divergence policy in
[FORGENTE.md](../FORGENTE.md).

### L2 — Repository agent configuration and providers

Two independent pieces of configuration:

- **Agent definitions in the repository.** `AGENTS.md` already carries
  build, test, and style instructions. Named agent personas — a reviewer
  restricted to read-only tools, a triager that only labels — belong beside
  it, and the forge surfaces them wherever an agent can be assigned.
- **Model providers.** Endpoint, credential, and model selection at the
  instance and organization level, so that in-forge features can call a model
  without every user wiring a workflow first. Credentials are secrets and get
  handled as such: encrypted at rest, never rendered back, never logged, and
  never exposed to repository code.

### L3 — Agent sessions

The mission-control layer, and the largest slice. Assigning an issue or a
review to an agent starts a *session*: a dispatched Actions workflow with a
session record in front of it, carrying status, live logs, a steer-or-stop
control, and links to whatever it produced. Forgente already runs the
sandbox — Actions with a runner fleet is exactly the substrate GitHub's
coding agent uses — so this layer is mostly plumbing and UI over machinery
that exists.

### L4 — Tenants

The visible features everyone actually names: AI code review, issue triage
and labelling, pull-request summaries. Each is an agent running on L0–L3
rather than a feature compiled into the server, and each ships as a reference
implementation in its own repository. Keeping them out of the tree is
deliberate: it proves the substrate is genuinely usable by third parties, and
it keeps model-specific behavior out of the forge's release cycle.

## Governance is the differentiator

GitHub made agents *ungovernable*: cloud-only, subscription-gated, and
exempted from branch protection as bypass actors. Forgejo went the other way
and banned agent contributions outright. Neither leaves a self-hosted
operator anywhere to stand.

Forgente's position is the empty middle — agents are allowed, and they are
governed:

- Branch protection, required reviews, and required checks apply to agents
  exactly as they apply to humans. There is no bypass path.
- Every agent action is attributable: which agent, owned by which
  organization, in which session, on whose behalf.
- One switch suspends every agent in an organization.
- It all runs on the operator's own hardware, against the operator's choice
  of model.

This is also the honest answer to agent-hype skepticism: the same primitive
serves ordinary CI bots and integrations, which have needed a real identity
in this codebase for years.

## Non-goals

- **Editor and client tooling.** See above.
- **Hosting or reselling model inference.** Providers are configured, not
  supplied.
- **An agent marketplace or agent hosting.** That is GitHub's cloud
  business, and it is not what a self-hosted forge is for.
- **A new permission system.** Agents use teams and collaborators. The spike
  confirmed the existing machinery has no user-type gate.
- **Copilot-Extensions-style plugins.** GitHub deprecated them in favour of
  MCP; there is no reason to build the thing that lost.
- **Autofix.** It presupposes code scanning, which Forgente does not have.
  Scanning may be worth building, but on its own merits.

## Open questions

- Naming and positioning: "agents" reads forward-leaning and invites hype
  scepticism; "service accounts done right" reads conservative and undersells
  the direction. This affects landing copy, docs, and the announcement post.
- Whether L2's provider configuration belongs to the instance only, or to
  organizations as well, and how quota and cost controls work if the latter.
- Whether upstream Gitea activating `UserTypeBot` would collide. The hedge is
  the same additive discipline used for every Forgente feature so far:
  explicit enum values, separate tables, no numbered migrations.
