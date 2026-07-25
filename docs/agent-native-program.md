# The agent-native program

Forgente's directional bet: **agents are first-class principals in the
forge, on infrastructure the operator owns.** This document is the
program-level design for that bet — what gets built, in what order, and what
deliberately does not get built. Phase 3 of [ROADMAP.md](../ROADMAP.md)
tracks its execution; individual slices get their own design notes and pull
requests.

The parity target is GitHub's agent surface as surveyed in **July 2026**.
That surface moves fast — it changed substantially in the six months before
this document was written — so the survey below is dated on purpose and
should be re-run before any layer is scoped in detail.

## What is and is not forge work

GitHub's agent surface is one product name over three very different things.
Only the third is a forge concern.

**Client-side — not built here.** IDE completion and chat, the Copilot CLI,
the Copilot desktop app, editor agent pickers. Editors and agent CLIs already
work against Forgente repositories over ordinary git and the API; nothing in
the forge gates them. Competing on editor integration is a client race, not
forge work.

**Model access — parity, not a differentiator.** Forgente configures
*providers*: an operator or organization points it at an OpenAI-compatible
endpoint, an Anthropic endpoint, or a local Ollama / vLLM server, with its
own credentials. This is the right design, but it must not be sold as a
Forgente invention — GitHub shipped enterprise BYOK in late 2025 and by
mid-2026 supports Bedrock, Google AI Studio, Azure OpenAI, Anthropic, any
OpenAI-compatible gateway, and locally-running Ollama or LM Studio, with
per-organization keys and admin context-window caps. Forgente's honest claim
is narrower and still real: no per-seat AI subscription is required to use
any of it.

**Forge-side — the build.** Agent identity, agent sessions and their sandbox
contract, repository agent configuration, a first-party MCP server, and the
features that ride on them. The rest of this document is about those.

## Where GitHub actually is (July 2026)

Surveyed from GitHub Docs and the changelog. Recorded because the strategy
below only makes sense against an accurate baseline, and because an outdated
baseline previously produced two wrong conclusions in this very document.

**The agent runtime.** "Copilot coding agent" is now **Copilot cloud
agent**: it researches, plans, and pushes to a `copilot/` branch or the
pull request's own branch, in a sandbox, opening a draft pull request.
**GitHub Agentic Workflows** (public preview, June 2026) let repositories
define automations as natural-language Markdown that *compiles to ordinary
Actions YAML* and runs on existing runner groups. **Agent session streaming**
(preview, July 2026) exposes prompts, responses, and tool calls to enterprise
admins by REST and by push to SIEM tooling.

**Extensibility.** Not MCP-only, as previously recorded here. GitHub now
ships **Plugins** (installable bundles of skills, tools, hooks, and MCP
servers), **Agent Skills** (built on the open Agent Skills format originated
by Anthropic), **Hooks**, **Enterprise Plugin Standards** for central policy,
a **Copilot SDK** with sub-agent orchestration, and **agent apps** — partner
agents installed from the Marketplace as GitHub Apps and assignable to issues
or invoked from pull request comments. Third-party agents (Codex, Claude) are
first-class providers, not workarounds.

**Configuration.** Custom agents are Markdown profiles with `name`,
`description`, `prompt`, optional `tools`, and optional `mcp-servers`, read
from `.github/agents/NAME.md` at repository level and `/agents/NAME.md` in
the `.github` or `.github-private` repository at organization and enterprise
level. `AGENTS.md` carries repository conventions.

**Governance — stronger than this document originally claimed.** The cloud
agent is *subject to* branch protections and required checks. It cannot mark
a pull request ready for review, cannot approve, and cannot merge; the person
who triggered it cannot approve the resulting pull request either. It signs
its commits (April 2026). Its network access is restricted by an egress
firewall, and hidden characters are stripped from input to blunt prompt
injection. Agentic workflows default to read-only permissions, validate
"safe outputs", and run a threat-detection job over proposed changes before
they apply. Admins enable it per organization, target by custom property, and
choose which security tools it may use. Issue automation carries **rationale,
confidence levels, and an approvals queue** (preview, July 2026) so
low-confidence actions wait for a human.

Adding the agent as a ruleset **bypass actor** is available, but it is an
opt-in escape hatch, not the default posture. Any claim that GitHub exempts
agents from governance is false and must not be repeated.

## Parity map

| GitHub | Forgente today | What is missing |
| ---- | ---- | ---- |
| Agent apps installed as GitHub Apps; cloud agent acts under Copilot identity with the requester as co-author | `UserTypeBot` exists but dormant; OAuth2 apps act *as the authorizing user*, so there is no installation identity at all | The App-installation primitive itself — this is the real gap, and it predates AI |
| Cloud agent: issue → sandboxed run → draft PR | Actions and a runner fleet already run untrusted workloads | Assignment, a session record, draft-PR wiring |
| Agentic Workflows: Markdown compiled to Actions YAML | Actions runs hand-written YAML | The compile step and the safety envelope around it |
| Session streaming and audit | Actions logs only | A session as a first-class, queryable object |
| Custom agents (`.github/agents/NAME.md`, org and enterprise levels) | `AGENTS.md` is already an in-repo convention | Read agent profiles and offer them at assignment time |
| Plugins, Agent Skills, Hooks | — | Adopt the open Agent Skills format rather than inventing one |
| First-party MCP server | `gitea-mcp` works unforked against Forgente | Fork trigger is agent-token awareness, not API divergence |
| Copilot code review | — | A *tenant* of the layers above, not a layer itself |
| Agentic autofix for scanning alerts (preview, July 2026) | no code scanning of any kind | Code scanning is the prerequisite; that is security work, not AI work |

## The layers

Each layer is independently shippable and useful on its own. Later layers
assume earlier ones; nothing in the list requires the layer above it.

### L0 — Installation identity, with agents as its first consumer

The framing correction that matters most: Forgente's gap is not "agents". It
is that Forgente has **no installation-identity primitive**. OAuth2
applications act as the authorizing user, and `UserTypeBot` rows have no
product surface. GitHub Apps have provided per-installation identity —
distinct principal, own permissions, own tokens, attributable actions — for
close to a decade, and *that* is what agent apps are built on. Build the
primitive; agents are its first and most visible consumer, and ordinary CI
bots and integrations have needed it here for years.

Concretely: organization-owned agent principals, created from organization
settings, permissioned through ordinary team and collaborator membership,
token-authenticated, badged wherever they act, revocable in one place.

**Upstream is activating the same primitive, and Forgente complements it
rather than duplicating it.** go-gitea/gitea#38181 (approved, awaiting a
final review at the time of writing) gives site admins bot-account creation,
individual-to-bot conversion, admin-managed scoped tokens, and — importantly
— closes real holes where a bot could obtain an interactive session through
LDAP/SMTP/PAM fallback or reverse-proxy headers. It ships its own design
note at `models/user/bot_user_design.md`.

That work is cherry-picked when it merges, not reimplemented. What it
deliberately leaves out is precisely Forgente's layer: it declares
organization-managed bots out of scope because ownership "expands the
permission and ownership model considerably (who owns the bot, who can
rotate its tokens, visibility across orgs)", while noting the layer can be
added later without breaking its model. The review thread goes further,
sketching global, user-owned and org-owned levels and asking whether
ownership needs its own table — the same conclusion reached here
independently.

The practical consequence for scope: build the ownership table, the
organization-side management surface, the kill switch, and provenance.
Do not build admin creation, conversion, or the interactive-auth hardening.
The demand behind this is long-standing and quantified — upstream #25900
(organization-level access tokens, 87 reactions since 2023), #13044 (bot
accounts owned by a user or org, 38 reactions since 2020), #26754 (service
accounts to shrink CI blast radius, 17), and #33469 (bot display, 4).

The mechanism is already present: access tokens are per-user with a full
scope system, so an organization-level access token *is* an org-owned bot
holding a scoped token — the same way GitLab implements group tokens. No new
token machinery is needed, and "organization-level access tokens" is the
framing with the most demand behind it.

One gap the spike did not catch: token authentication resolves a token
straight to its user with no active-or-suspended check, so deactivating an
account does not revoke its tokens. The kill switch therefore needs an
explicit gate in the token path — the one piece of L0 that is not purely
product surface, and worth reviewing on its own.

A spike (2026-07-17, dev instance) established that this needs **no
authentication work at all**. A `UserTypeBot` row already authenticates by
access token, acts through the API and git-over-HTTP, and gains and loses
permissions through the normal membership path. Password sign-in for
non-individual users is already refused in `services/auth/signin.go` — agents
are token-only, enforced by code that already exists. That block is correct
behavior; do not "fix" it.

### L1 — First-party MCP server

Fork `gitea-mcp` into the Forgente organization once it needs to know about
agent identity — minting and scoping tokens for an agent principal rather
than reusing a human's. Until that divergence, the upstream server works
against Forgente unmodified, consistent with the fork-on-divergence policy in
[FORGENTE.md](../FORGENTE.md).

### L2 — Repository agent configuration and providers

Two independent pieces of configuration:

- **Agent definitions in the repository.** `AGENTS.md` already carries build,
  test, and style instructions. Named agent personas belong beside it, and
  the forge surfaces them wherever an agent can be assigned. Follow the
  established shape — a Markdown profile with prompt, tool allow-list, and
  MCP server declarations — and adopt the open Agent Skills format rather
  than inventing a Forgente-specific one. Interoperating with agents people
  already run is worth more than a bespoke schema.
- **Model providers.** Endpoint, credential, and model selection at the
  instance and organization level. Credentials are secrets and get handled as
  such: encrypted at rest, never rendered back, never logged, and never
  exposed to repository code.

### L3 — Agent sessions and the sandbox contract

The largest slice, and the one this document originally under-scoped.
Assigning an issue or a review to an agent starts a *session*: a dispatched
Actions run with a session record in front of it, carrying status, live logs,
a steer-or-stop control, and links to whatever it produced. Forgente already
runs the sandbox — Actions with a runner fleet is exactly the substrate
GitHub uses — and GitHub's own choice to compile agentic workflows down to
ordinary Actions YAML is direct evidence this is the right substrate.

The sandbox *contract* is not optional polish, and matching it is a
precondition for letting agents near real repositories:

- Read-only permissions by default; write scopes granted explicitly.
- Egress control on agent runs — the operator decides what the sandbox may
  reach.
- Agents cannot approve, cannot merge, and cannot mark their own work ready
  for review. Branch protection and required checks apply unchanged.
- Untrusted input is treated as untrusted: the issue body an agent reads is
  data, not instruction.
- Every action is attributable — which agent, which organization, which
  session, on whose behalf — and the session log is queryable and exportable.
- Confidence and approvals for low-stakes automation: an agent may propose
  rather than apply, with its reasoning attached, and a human accepts or
  declines. This pattern is cheap, it is proven, and it fits a self-hosted
  audience better than silent automation does.

### L4 — Tenants

The visible features everyone actually names: AI code review, issue triage
and labelling, pull-request summaries. Each is an agent running on L0–L3
rather than a feature compiled into the server, and each ships as a reference
implementation in its own repository. Keeping them out of the tree is
deliberate: it proves the substrate is genuinely usable by third parties, and
it keeps model-specific behavior out of the forge's release cycle.

## What Forgente can honestly claim

An earlier draft of this document claimed the differentiator was governance —
that GitHub exempts agents while Forgente would govern them. **That was
wrong**, and it is recorded here so the mistake is not made again in landing
copy or an announcement post. GitHub's agent governance in mid-2026 is
serious: signed commits, protection rules enforced, no self-approval, egress
firewalls, threat detection, per-organization enablement, and audit
streaming. Forgente does not get to win on that axis by default; it has to
build most of it just to be credible, which is why the sandbox contract sits
inside L3 rather than in a "later" pile.

What is left is narrower, and durable:

- **Self-hosted.** GitHub's agent stack runs in GitHub's cloud. Forgente's
  runs on the operator's hardware, against a model of their choosing,
  including one that never leaves the network. For regulated, air-gapped, and
  sovereignty-constrained users this is not a preference, it is the only
  option.
- **No subscription gate.** GitHub's agent capabilities are inseparable from
  Copilot licensing, AI credits, and cost centers. Forgente's agent
  primitives are part of the forge.
- **Open formats over a proprietary stack.** `AGENTS.md`, the Agent Skills
  format, and MCP are open; Marketplace agent apps, plugin standards, and the
  Copilot SDK are GitHub's. Building on the open half is both cheaper and
  more defensible for a fork with Forgente's resources.
- **A primitive the codebase never had.** Installation identity is missing
  from this lineage entirely. Whoever adds it defines how bots, integrations,
  and agents work here.

Forgejo, by contrast, banned agent contributions outright. Between a
proprietary hosted stack and a prohibition there is real room — but the room
is "self-hosted and open", not "the only one who governs agents".

## Non-goals

- **Editor and client tooling.** See above.
- **Hosting or reselling model inference.** Providers are configured, not
  supplied.
- **An agent marketplace, agent hosting, or a plugin economy.** GitHub has
  Marketplace agent apps and enterprise plugin standards; that is a platform
  business, not what a self-hosted forge is for. Note this is a *choice*, not
  a claim that the category is dead — an earlier draft wrongly recorded
  Copilot Extensions as deprecated in favour of MCP, when in fact GitHub's
  extensibility surface grew.
- **A new permission system.** Agents use teams and collaborators. The spike
  confirmed the existing machinery has no user-type gate.
- **Autofix.** It presupposes code scanning, which Forgente does not have.
  Scanning may be worth building, but on its own merits.
- **Chasing every surface.** GitHub shipped Copilot Memory, vision,
  repository overviews, desktop parallel workstreams, and a six-language SDK
  in the survey window alone. Parity across all of it is not achievable and
  not the point; the layers above are.

## Open questions

- Naming and positioning: "agents" reads forward-leaning and invites hype
  scepticism; "service accounts done right" reads conservative and undersells
  the direction. A third framing is now available and may be strongest —
  lead with **installation identity** as infrastructure, with agents as the
  headline consumer.
- Whether L2's provider configuration belongs to the instance only, or to
  organizations as well, and how quota and cost controls work if the latter.
- How much of the L3 sandbox contract is required for a first release versus
  a credible one. Shipping agent sessions without egress control is probably
  not defensible.
- Whether to adopt the natural-language-compiled-to-YAML workflow shape at
  all, or to keep agent invocation explicit and leave workflow authoring to
  humans.
- Upstream activating `UserTypeBot` is no longer hypothetical — see L0. The
  open question is now narrower: if upstream later adds its own ownership
  model, does Forgente converge on it or keep its own? Converging is
  cheaper, and the review thread suggests upstream would land somewhere
  close to this design anyway.
- Whether apps may be owned by individual users as well as organizations.
  Upstream #13044 asks for both and the review thread sketches three levels;
  the first release is organizations only, which needs no schema change to
  extend later.
- Whether a bot should be capped at its owner's access, as proposed in the
  upstream thread. It is a clean principle, but it interacts awkwardly with
  organization ownership, where the "owner" is not a single principal.
