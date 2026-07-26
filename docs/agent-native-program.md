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

## Where agents run, and why GitHub's answers do not transfer whole

Agent work is dispatched, executed and reported somewhere, and there are four
distinct arrangements. They matter because GitHub operates only the first, and
several of its design choices are consequences of that rather than statements
about the domain.

| Arrangement | Executes on | Obtains forge identity via | Reports work by |
| --- | --- | --- | --- |
| Forge-hosted | The forge operator's own sandbox | Minted internally; never crosses a trust boundary | Writing session records directly |
| Workflow-hosted | The repository's own runners, as a workflow step | A stored app credential exchanged for a short-lived token at job start | Editing a tracking comment |
| Client-side | The developer's machine | The developer's own token, or an OAuth grant | Not at all — the forge sees ordinary API calls |
| External service | A third party's infrastructure | An app token held out of band | Comments, commits, pull requests, statuses |

Strip the runtimes away and every one of them needs the same three things from
the forge: **an identity to act as, a way to obtain a credential for it, and a
record that the work happened.** Only execution differs. That is why supporting
all four needs no runtime abstraction and no pluggable interface — L0 supplies
the first, `AN-IDENT-2` the second, L3 the third, and none of them has to know
where the agent ran. Keep that neutrality in the data model, where it is free,
rather than in an abstraction layer, where it would be expensive and would
almost certainly put the seam in the wrong place.

Client-side is L1's constituency and needs nothing further. Workflow-hosted is
the arrangement a self-hosted forge can actually offer, and the one with a
working third-party reference implementation to study. External services already
work through ordinary app tokens. Forge-hosted is available to whoever operates
the instance — which includes any operator running their own runner fleet, and
Forgente on its own infrastructure; see the deployment posture below.

**The test to apply when copying a GitHub design.** Ask: *would this shape exist
if GitHub did not run the agent themselves?*

If yes, it is modelling of the domain — copy it. Task and session as separate
objects, the eight states, the field names, and assignment as the trigger are
all in this category, and were taken as-is.

If no, it encodes their operating position, and only the vocabulary transfers.
Four examples, each of which misleads if copied literally:

- **Sessions are read-only in their API** because their own runtime writes them
  directly, so no external party exists who could legitimately write one. Copy
  that constraint and Forgente's sessions stay permanently empty.
- **No credential appears anywhere** in the cloud agent, because none crosses a
  trust boundary. Every other arrangement crosses one.
- **Progress is reported by editing a comment** — a workaround third parties
  adopted because no session-write endpoint was ever offered to them.
- **The agent stack is cloud-only**, because there is no self-hosted shape of a
  runtime they operate.

Underneath all four: GitHub can trust the runtime because it is theirs. Forge,
runner and agent share one operator. Everywhere else those are up to three
parties with no obligation to trust each other, and that single difference
generates the rest.

This is the wire-compatibility rule in [AGENTS.md](../AGENTS.md) one level up —
keep the words the ecosystem depends on, own everything underneath.

## Where GitHub actually is (July 2026)

Surveyed from GitHub Docs and the changelog. Recorded because the strategy
below only makes sense against an accurate baseline, and because an outdated
baseline previously produced two wrong conclusions in this very document.

**Which GitHub.** Copilot is not available on GitHub Enterprise Server at
all — 3.20 (March 2026) ships none of it, and the cloud agent requires
Enterprise Cloud. The parity target is therefore **github.com / Enterprise
Cloud**, not GHES, and it should be read that way everywhere below. It also
means a fully self-hosted agent-native forge does not exist from GitHub
today, which is the single most important competitive fact in this document.

**How to re-run this survey.** Walk the product documentation, not the
changelog. A changelog reports what *changed* and will always give a
window-shaped answer; the docs describe current state regardless of when each
piece shipped. Start from the Copilot documentation tree — concepts, the
agents sub-section, and the REST reference — and read the specifications
themselves for anything being implemented rather than a vendor's paraphrase
of them. Use the changelog only afterwards, to date things and catch
deprecations, and read it by **month archive**: the label-filtered view
silently renders empty month headers, which is how an earlier pass at this
document missed June entirely. Survey the ecosystem in the same pass, not
only the target. Re-read prior research before starting.

**Deliberately not surveyed**, recorded so the next pass does not mistake
these for oversights: ~~the exact MCP specification version `gitea-mcp`
targets~~ (checked 2026-07-26 — 2025-11-25, via `mcp-go` v0.56.0; see L1);
whether `tea` and the Gitea SDK have
agent affordances (matters when the MCP-versus-CLI question in the open
questions is decided); GitLab Duo and Bitbucket Rovo, last checked
2026-07-20 (they do not change a GitHub-parity plan); and changelog history
before May 2026 (superseded — the documentation already describes whatever
survived from it).

**On following the ecosystem's conventions.** This document repeatedly says to
take names, paths and wire formats from the Gitea ecosystem. It is worth being
honest about what that means in practice: there is no Gitea documentation on
building agent integrations, and the only in-tree design-document precedent is
`services/actions/token_permission_design.md`. The agent-adjacent work upstream
is a handful of open issues — go-gitea/gitea#36444 (AI code review), #37151
(kanban API for agentic workflows), #35506 (MCP) and #38181 (bot accounts) —
not written guidance. So "the ecosystem's conventions" are usually being
*inferred from code* here rather than followed from a specification, and any
claim in this document about them should be read at that confidence.

Re-checked on 2026-07-25 by that method; nothing below needed correcting.
The entries that postdate the original survey fit the map rather than
breaking it: the GitHub MCP server moved to the next MCP specification
(Jul 23), agent automation controls for issues reached public preview (Jul 23
— this is the approvals queue described below), and code review gained
customization (Jul 17). Two changes do imply work not otherwise tracked here:
agents stopped needing a personal access token inside Actions (Jun 11 for
agentic workflows, Jul 2 for the CLI), and **cloud *and local* sandboxes**
reached public preview (Jun 2), which narrows what "self-hosted" buys us —
see the honest-claims section.

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

## Where the ecosystem already is (July 2026)

Surveyed 2026-07-25. This document originally surveyed the parity target but
never surveyed the substrate, which produced scope that was too large in one
place and too small in another. Both corrections are load-bearing.

**Much of the L3 sandbox contract is already in this tree.** Actions has a
GitHub-compatible `permissions:` key, Permissive and Restricted default
modes, `MaxTokenPermissions` ceilings at both repository and organization
level, a cross-repository allow-list, an automatic downgrade to read-only for
fork pull requests, and an approval gate on fork runs. It is wired end to end,
settings UI included, and documented in
[services/actions/token_permission_design.md](../services/actions/token_permission_design.md).
The organization-level ceiling is stronger than GitHub's equivalent, which has
no such clamp. Read-only-by-default is therefore not work to schedule.

**Scoped Workflows** run workflows from one central repository against many
others, in each consuming repository's own context, configured by
`SCOPED_WORKFLOW_DIRS`. That is the substrate's answer to centrally-mandated
automation, and it is the natural delivery vehicle for organization-mandated
agent policy.

**Upstream has already settled the question L1 answers, the same way.**
go-gitea/gitea#35506 proposed implementing MCP inside the forge; the
maintainers converged on keeping the server standalone precisely because the
protocol moves faster than forge releases, with importing it as an optional
module the furthest anyone was willing to go. Independent agreement is worth
more here than this document's own reasoning.

**Upstream's AI code review request is blocked on the primitive L0 ships.**
go-gitea/gitea#36444 stalled on the view that the right shape is generic bot
management, where a review bot is a bot holding a "review" skill — which is
L0 plus L4 as described below, stated upstream. The same thread's other
suggestion is that agents drive the `tea` CLI rather than MCP; see L2.

**A sibling fork already shipped a machine-identity primitive.** Forgejo v16
(2026-07-16) authenticates API, git and package access with JWTs issued by an
external system — Forgejo Actions, GitHub Actions, GitLab CI, AWS — with no
static credentials, short expiry, rotatable signing keys, and per-integration
validation and capability grants. This is *workload* identity: it authenticates
an external system authenticating inward. It is not an owned principal, and it
carries no profile, provenance or kill switch, so it does not overlap L0's
design. It does mean the claim that this lineage has no machine-identity work
at all is false, and it is the closest prior art to L1's direction.

**Forgejo's position on agents has hardened, not moved.** As of their March
2026 report, AI-generated work is prohibited outright rather than
licence-gated. Their infrastructure work above continues regardless, so
"culturally anti-AI" is not the same as "not building the plumbing".

## Capability spec (frozen, July 2026)

This replaces the earlier parity map. It is a **frozen** snapshot: the target
moves weekly, and a target that moves is one nothing can ever be finished
against. Build to this list; anything GitHub ships after it goes to a dated
backlog and is considered at the next re-survey, not mid-flight.

Rows carry stable IDs so pull requests and issues can cite `AN-IDENT-2`
rather than re-describing the capability. Status is one of **shipped**,
**Ln, part shipped** (a layer has landed some of it — say which pull
requests, and say in the Forgente column what is *not* built), **Ln**
(assigned to a layer, nothing landed), **open** (accepted, unassigned), or
**excluded** (with a reason — those rows are the most useful ones here).

**Update the row in the pull request that ships the capability, not in a
later documentation pass.** These rows went stale within a single session of
shipping once already: the layer sections below recorded what had landed
while six rows here still read `—`, which is worse than a vague row, because
this table is what a pull request cites *instead of* re-reading the code. A
row that under-reports invites rebuilding something that exists.

### Identity and installation

| ID | GitHub's shape | Forgente | Status |
| --- | --- | --- | --- |
| AN-IDENT-1 | Agent apps are GitHub Apps; two-step enable (install, then authorize agent features) | Organization-owned apps with scoped tokens | shipped (#66) |
| AN-IDENT-2 | Partner identity carried by a JWT assertion GitHub issues; no user-managed credentials | A run exchanges the job token it already holds for a short-lived token scoped by an organization's grant. No key or secret is stored anywhere, because the app is owned by the organization and the run is one the forge dispatched | shipped (#89, #90) |
| AN-IDENT-3 | Third-party agents install as hidden GitHub Apps (`anthropic code agent`, `openai code agent`), fully audit-logged | Apps are visible and org-owned by design | shipped (#66) |
| AN-IDENT-4 | Attribution: agent-authored PRs in author search; release notes credit them | Bot label on comments; profile and commits pending | L0 follow-up |
| AN-IDENT-5 | Per-installation permission scoping | Team and collaborator membership | shipped (#66) |

### Runtime and sessions

| ID | GitHub's shape | Forgente | Status |
| --- | --- | --- | --- |
| AN-RUN-1 | Cloud agent researches, plans, commits to a branch, opens a PR | Actions + runner fleet is the substrate | L3 |
| AN-RUN-2 | Confidence rating; low-confidence changes held for review | — | L3 |
| AN-RUN-3 | Cloud **and local** sandboxes (MXC; macOS/Linux/Windows) | Runners are already operator-hosted | shipped (substrate) |
| AN-RUN-4 | Scheduled and event-triggered agent tasks | Assignment to an app records a task; scheduled dispatch not built | L3, part shipped (#84) |
| AN-RUN-5 | Agent tasks REST API: 5 endpoints, task/session objects, 8 states | Task and session model with the same 8 states, plus list and get; the write half and sessions themselves are not built | L3, part shipped (#82, #85) |
| AN-RUN-6 | Session control page, session filters, cross-session insight | The task record on the issue it belongs to; no session view, no cross-session anything | L3, part shipped (#86) |
| AN-RUN-7 | *No agent webhook events exist* — 75 events, none agent-related | — | open — see below |

### Repository configuration

| ID | GitHub's shape | Forgente | Status |
| --- | --- | --- | --- |
| AN-CFG-1 | Custom agents: `.github/agents/*.agent.md`; org in `.github`/`.github-private`; enterprise in a designated `.github-private`; precedence repo → org → enterprise | `.github/agents` and `.claude/agents` read from the default branch; org and enterprise levels not built | L2, part shipped (#77, #79, #80) |
| AN-CFG-2 | Agent profile frontmatter: `description` (required), `name`, `target`, `tools`, `model`, `disable-model-invocation`, `user-invocable`, `mcp-servers`, `metadata`; prompt ≤ 30,000 chars | Parsed, with absent distinguished from false for the two invocation flags; no prompt-length cap | L2, part shipped (#77) |
| AN-CFG-3 | Agent skills: `SKILL.md` folders at `.github/skills`, `.claude/skills`, `.agents/skills`; personal `~/.copilot/skills`, `~/.agents/skills` | All three repository locations read; the personal ones are a client concern, not the forge's | L2, part shipped (#77, #79, #80) |
| AN-CFG-4 | `AGENTS.md` for cross-tool conventions | Already an in-repo convention | shipped |
| AN-CFG-5 | Plugins: `plugin.json` bundling agents, skills, `hooks.json`, `.mcp.json`, `lsp.json`; distributed via `marketplace.json` | — | excluded — plugin economy is a non-goal |
| AN-CFG-6 | Instruction files: `.github/copilot-instructions.md`, `.github/instructions/**/*.instructions.md` | — | excluded — vendor-specific; `AGENTS.md` covers it |

### Agentic workflows

| ID | GitHub's shape | Forgente | Status |
| --- | --- | --- | --- |
| AN-WF-1 | Markdown + YAML frontmatter compiled to a hardened `.lock.yml`, both committed | Actions runs hand-written YAML | open — see the open questions |
| AN-WF-2 | Read-only by default; writes confined to declared **safe outputs** | Token permission system already clamps | shipped (substrate) |
| AN-WF-3 | Proposed outputs scanned before any write is applied | — | L3 |
| AN-WF-4 | Secrets withheld from the agent runtime | — | L3 |
| AN-WF-5 | Central workflow policy across repositories | Scoped Workflows | shipped (substrate) |

### MCP

| ID | GitHub's shape | Forgente | Status |
| --- | --- | --- | --- |
| AN-MCP-1 | Official server, remote and local, with toolset toggles | `gitea-mcp` unforked, validated as an org-owned app | shipped |
| AN-MCP-2 | Remote auth by OAuth or PAT | OAuth 2.1 discovery served; the `401` challenge is the MCP server's half, upstream | L1, part shipped |
| AN-MCP-3 | Repo-level MCP config; org/enterprise policy, disabled by default | — | L2 |
| AN-MCP-4 | Registry, allowlists, runtime discovery (Agent Finder / ARD) | — | excluded — registry is a marketplace concern |

### Governance

| ID | GitHub's shape | Forgente | Status |
| --- | --- | --- | --- |
| AN-GOV-1 | Enterprise → org (selectable by custom property) → repo → IDE, enterprise non-overridable | Instance → org → repo ceilings on Actions tokens | partial, shipped |
| AN-GOV-2 | Agent session audit events; filterable recent view; audit streaming | — | L3 |
| AN-GOV-3 | Agent output scanned by CodeQL, secret scanning, advisory DB — no GHAS licence needed | No code scanning of any kind | excluded — scanning is security work, on its own merits |
| AN-GOV-4 | Branch protection applies; no self-approve, self-merge, or self-ready | — | L3 |
| AN-GOV-5 | Egress firewall on agent runs | Not expressible: the runner protocol has no network field in either direction. Reachable as label-based routing and refusal, which is attestation rather than enforcement | **L3 — the real gap** |
| AN-GOV-6 | Signed agent commits | — | open |
| AN-GOV-7 | Kill switch: installations can be suspended, and policy can disable agents per organization | Org-wide suspend enforced at the single auth choke point, covering every credential type including git-over-SSH | shipped (#66) |

### Tenants

| ID | GitHub's shape | Forgente | Status |
| --- | --- | --- | --- |
| AN-REV-1 | Code review: manual or automatic, effort levels, per-repo MCP servers, reads `AGENTS.md` and skills, self-hosted runner option | — | L4 |
| AN-REV-2 | Issue triage and labelling with rationale and an approvals queue | — | L4 |
| AN-REV-3 | Agentic autofix for scanning alerts | — | excluded — presupposes AN-GOV-3 |

### Excluded wholesale

Client-side tooling (IDE, CLI, desktop app, editor agent pickers), model
hosting and inference resale, and the entire billing surface — credits, cost
centers, per-user budgets, seat management, usage metrics. That last one is
roughly a quarter of GitHub's agent-related changelog volume and is the
scaffolding of a subscription business; its absence is one of the few things
Forgente gets to claim outright.

### The seam worth building into

`AN-RUN-5` and `AN-RUN-7` together describe a structural limit in GitHub's
design. The agent tasks API is authenticated **user-to-server only** —
installation access tokens are explicitly unsupported — so an app cannot
drive the agent API as itself, even though agent apps *are* GitHub Apps. And
no webhook event reports agent activity at all, so sessions cannot be
subscribed to, only polled or streamed to audit tooling.

Forgente's app is a principal holding its own token, with no user delegation
in the path, so neither limit is inherited. An agent API here can accept app
tokens directly, and sessions can emit ordinary webhook events. Take the
object model from GitHub — task and session, the eight states, the field
names — and do not reproduce the authentication constraint. This is the
clearest case in the spec where parity supplies the design and Forgente's
identity model supplies the improvement.

**But do not let the record depend on the agent's cooperation.** The API is
worth building for the reason above, and nothing in the ecosystem will call it
on the day it ships: an agent written against GitHub reports progress by editing
a comment, because a comment is all GitHub ever offered it. A session model that
fills in only when an agent posts to it would stay empty for precisely the
agents Forgente most wants to run. So derive what is derivable from what the
forge already observes — the dispatched run, comments authored by the app, refs
and pull requests it pushed — and treat a posted session as enrichment from
agents that know about Forgente rather than as the source of truth. Deriving
works for all four arrangements above; reporting works for one.

## The layers

Each layer is independently shippable and useful on its own. Later layers
assume earlier ones; nothing in the list requires the layer above it.

### L0 — Installation identity, with agents as its first consumer

Shipped in #66, and validated end to end against a real MCP client — see L1.

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

Shipped in #71, #72 and #73. Serves the client-side arrangement above, and
needs nothing further from the other three.

Do not fork `gitea-mcp`. A validation run (2026-07-25, Forgente `main` at
`58c57191f4`, `gitea-mcp` at `18fcd663e0f0`) drove the upstream server
unmodified against a Forgente instance, authenticating as an
organization-owned app, and it works: a clean handshake, 53 tools, and real
work committed to a repository. Upstream reached the same conclusion about
in-tree MCP servers for its own reasons; see the ecosystem survey above.

With `GITEA_ACCESS_TOKEN` set to a token minted through the organization's
Applications page and `GITEA_HOST` pointed at the instance:

- `get_me` returns the app, not a human.
- An issue filed through the server is attributed to the app, and a file
  committed through it carries the app as commit author, resolved back to
  the app's user.
- The bot label and the operating organization render on the issue and on
  the app's profile.
- A call needing a scope the token lacks is refused server-side, and the
  refusal surfaces intact through MCP.
- Suspending the app stops it immediately, and resuming restores it without
  rotating the token.

That last point is the one worth recording. The kill switch had only ever
been exercised at the moment of authentication; this run held a single MCP
process open across the whole sequence, so the suspension was observed
cutting off an agent already connected and working, not merely refusing a
fresh login.

L0 plus the upstream server therefore already delivers a working agent
principal, and the fork-on-divergence bar in [FORGENTE.md](../FORGENTE.md)
is not met. Forking now would buy nothing and cost a permanent merge burden.

**The validation's own blind spot, and what it makes L1.** That run used the
stdio transport, so it never exercised how a *remote* MCP client connects.
Remote clients — Claude's web, mobile and desktop connectors, ChatGPT,
Copilot — cannot send a static bearer token at all; they support only
unauthenticated servers or the OAuth 2.1 flow in the MCP authorization spec.
A self-hosted operator today must therefore expose an unauthenticated
endpoint or run an external OAuth proxy, and the proxy still talks to the
forge with one shared token, collapsing per-principal identity exactly where
L0 was supposed to provide it.

**Corrected against the MCP authorization specification (2026-07-25).** An
earlier revision of this section assigned the wrong work to the forge. The
spec splits the roles cleanly, and reading it changes what L1 is.

*Which version.* This was first read against the `draft` specification, which
was the wrong document to build from: `gitea-mcp` depends on `mcp-go` v0.56.0,
whose `LATEST_PROTOCOL_VERSION` is **2025-11-25**. Re-checked against
2025-11-25 on 2026-07-26 — every point below holds there too, with one wording
correction noted. This closes the "which MCP specification version does
`gitea-mcp` target" item that the July survey deliberately deferred; pin any
future reading to the version the server actually implements, not to `draft`.

- "MCP servers **MUST** implement OAuth 2.0 Protected Resource Metadata
  (RFC 9728)." The protected-resource document and the `401` with
  `WWW-Authenticate` belong to the **resource server** — that is `gitea-mcp`,
  not Forgente. This is upstream work, and `gitea-mcp` issue #207 is already
  the right place for it.
- "MCP authorization servers **MUST** provide at least one of the following
  discovery mechanisms: OAuth 2.0 Authorization Server Metadata (RFC 8414)
  [or] OpenID Connect Discovery 1.0", and clients **MUST** support both.
  Forgente already serves `/.well-known/openid-configuration`, so it already
  satisfies this MUST. The forge was never the blocker it was described as.
- Dynamic Client Registration is optional and on its way out. 2025-11-25 says
  authorization servers "**MAY** support" it and that it "is included for
  backwards compatibility with earlier versions of the MCP authorization
  spec"; `draft` goes further and calls it deprecated. An earlier revision of
  this document reported the `draft` wording as though it were current — the
  conclusion is unchanged under either, so do not build DCR, but the
  characterisation was drawn from the wrong document.

Two requirements in 2025-11-25 that the `draft` reading did not surface, both
already satisfied: an authorization server serving OpenID Connect Discovery
**MUST** include `code_challenge_methods_supported`, or clients refuse to
proceed for lack of PKCE evidence — Forgente's OIDC document has always
carried it, and the RFC 8414 document carries it too. Support for Client ID
Metadata Documents is advertised by `client_id_metadata_document_supported`;
Forgente does not implement CIMD, and omitting the field correctly says so.

Forgejo's Authorized Integrations is the nearest prior art in this lineage —
short-lived, externally-issued, forge-validated credentials with per-issuer
capability grants — and worth studying before designing this, even though it
solves inbound workload identity rather than delegated user authorization.

**What is actually left on the forge side** is narrower than "make OAuth
work", and one item is a defect rather than a gap:

1. RFC 8414 metadata at `/.well-known/oauth-authorization-server`. Redundant
   against the MUST, since OIDC discovery already satisfies it, but real
   clients probe it first — and unlike the OIDC document it can advertise the
   API scopes a client may request. *Shipped.*
2. **The full-access default is a bad default, not a consent bug.**
   `GrantAdditionalScopes` treats a grant carrying only `openid`, `profile`,
   `email` and `groups` as `AccessTokenScopeAll`, so an MCP client completing
   an ordinary OIDC flow receives a token with full API access. The consent
   screen is honest about it — with no API scopes requested, `grant.tmpl`
   shows "it will be able to access and write to all your account
   information, including private repos and organizations" — so the user is
   told exactly what they are granting. The real problem was discoverability:
   nothing advertised that `read:repository` or `write:issue` existed, so a
   client had no narrower thing to ask for. Item 1 fixes that. Changing the
   default itself is inherited 1.22 behaviour and would need a deprecation
   cycle; with discovery in place it is no longer urgent.
3. **RFC 8707 resource indicators, and audience binding.** Access tokens
   carried `GrantID`, `Kind` and `ExpiresAt` and nothing else — the `aud`
   claim was set only on the `id_token` — so a resource server could not
   satisfy the spec's "MCP servers **MUST** validate that access tokens were
   issued specifically for them as the intended audience". This was the item
   blocking the upstream work rather than a missing nicety. *The token
   endpoint now accepts `resource` and binds it as the audience.* What
   remains is smaller: the authorization endpoint accepts `resource` but does
   not persist it, so a token request is not checked against what was
   authorized, and RFC 9207 (`iss` in the authorization response) is still
   unimplemented.

   Note what this does **not** do: Forgente's own API still ignores `aud`, by
   necessity. `gitea-mcp` is a facade over the same instance and forwards the
   caller's token to the API, so enforcing the audience at the API would
   break the only architecture the ecosystem server actually has.

   2025-11-25 is blunt about this, and more so than the `draft` wording first
   quoted here: "If the MCP server makes requests to upstream APIs, it may act
   as an OAuth client to them. The access token used at the upstream API is a
   separate token, issued by the upstream authorization server. The MCP server
   **MUST NOT** pass through the token it received from the MCP client."

   So the tension is not a reading of ours — a compliant `gitea-mcp` would
   need its own credential at the forge, distinct from the caller's token.
   That is a real design change for upstream and the substance of what to
   raise on issue #207. It is upstream's call, not the forge's, but the forge
   is where the second credential would have to come from.
4. A "Connect an agent" panel on the app's settings page, for the local and
   stdio case that already works. *Shipped, minus the tool list — see below.*
5. **`plain` PKCE is still accepted.** OAuth 2.1 removed it, and the MCP
   specification requires authorization servers to implement OAuth 2.1, so
   advertising and accepting it in `code_challenge_methods_supported` is a
   standing compliance gap. It is narrow rather than alarming: `plain` means
   the challenge equals the verifier, so it only fails against an attacker who
   can observe the authorization request *and* intercept the code, and every
   MCP client is required to use `S256` regardless. Removing it is a breaking
   change for whatever still uses it, so it wants a deprecation cycle or an
   instance setting rather than a quiet deletion — but it should not stay
   undocumented, because both discovery documents currently advertise it as
   supported.

The panel carries the host, the token placeholder, and the step an operator
otherwise has to guess: an app has no repository access until it is added to
a team. That is correct behaviour — access is granted the ordinary way — but
it sits silently between creating an app and having a working agent, and it
was the third finding from the validation run.

**The scope-derived tool list was deliberately not built.** The original plan
was to emit a `GITEA_TOOLS` value matching the token's real scopes, because
the server advertises every tool regardless of what the token can do:
`create_repo` was offered and then refused at call time. The reasoning still
holds — only the forge knows a token's scopes — but the implementation does
not survive contact with the format. `GITEA_TOOLS` is an allowlist of
**individual tool names**, not toolset groups (`pkg/tool/tool.go` filters on
`flag.AllowedTools[st.Tool.Name]`), so honouring it means hardcoding roughly
fifty of another project's internal identifiers, pinned to a commit, in
Forgente. That table would agree with the token by manual synchronisation, not
by construction, and it would break silently the first time upstream renames
or adds a tool.

The rationale for the feature argues against this carrier for it. Filed
upstream as `gitea-mcp` issue #79, with MCP tool annotations (issue #173) the
better vehicle — contribute there rather than keeping a Forgente-only table
that drifts.

Watch item, not a fork trigger: `gitea-mcp` builds on `mark3labs/mcp-go` with
open requests to move to the official Go SDK, while GitHub's server already
tracks the next MCP specification. Protocol drift is worth monitoring.

### L2 — Repository agent configuration and providers

Half shipped. Agent definitions — skill and profile parsing, repository
discovery, and the settings surface — landed in #77, #79 and #80. Model
providers are unstarted.

This is the layer the neutrality position is actually made of: a repository
declares what it wants done in formats the ecosystem already reads, so the
agent executing them is replaceable. Inventing a Forgente-specific schema here
would forfeit the position while looking like ordinary product work.

Two independent pieces of configuration:

- **Agent definitions in the repository.** `AGENTS.md` already carries build,
  test, and style instructions. Named agent personas belong beside it, and
  the forge surfaces them wherever an agent can be assigned. Follow the
  established shape — a Markdown profile with prompt, tool allow-list, and
  MCP server declarations — and adopt an existing convention rather than
  inventing a Forgente-specific one. Interoperating with agents people
  already run is worth more than a bespoke schema.

  **Agent profiles and agent skills are different artifacts.** An earlier
  revision of this document conflated them, and the distinction decides the
  design. A *custom agent* is a persona — `.agent.md`, with a prompt, a tool
  filter and MCP server declarations (`AN-CFG-1`, `AN-CFG-2`). A *skill* is a
  packaged procedure — a folder containing `SKILL.md` plus optional
  `scripts/`, `references/` and `assets/`, loaded by progressive disclosure:
  name and description at discovery, full instructions only once a task
  matches (`AN-CFG-3`). Build both; do not merge them.

  Adopt the Agent Skills standard verbatim. It originated at Anthropic, is
  now openly governed, and is implemented by around forty-five agent products
  including Copilot, VS Code, Cursor, Codex and Gemini CLI — it is the actual
  interoperability layer, not one vendor's convention. A previous revision
  claimed `.agents/skills/` was an ecosystem convention *in conflict* with
  GitHub's; that is wrong and should not be repeated. `.agents/skills` is one
  of the locations GitHub itself reads, so there is no conflict to resolve
  and no compromise to design around.

  `gitea-tea-skill` publishes an official skill teaching an agent to drive
  the `tea` CLI, under `.agents/skills/`. Its file is plain Markdown rather
  than a conformant `SKILL.md`, so treat it as a useful precedent for the
  location and not as a model for the format. It also implies a second,
  cheaper integration path this document had not considered: an agent can
  drive `tea` instead of MCP, needing no server and no new protocol. It is
  the path upstream suggested in #36444, and it is worth costing before
  assuming MCP is the only surface.
- **Model providers.** Endpoint, credential, and model selection at the
  instance and organization level. Credentials are secrets and get handled as
  such: encrypted at rest, never rendered back, never logged, and never
  exposed to repository code.

### L3 — Agent sessions and the sandbox contract

Partly shipped, and the task half works end to end: the task and session model
(#82), a task recorded when an issue is assigned to an app (#84), the read API
(#85), and the record surfaced on the issue it belongs to (#86). What remains is
the harder half — `AN-IDENT-2`, sessions themselves, the egress routing below,
and the no-self-approve rules.

`AN-IDENT-2` is done (#89, #90), and it was first for a reason: until a run
could obtain a short-lived credential for its app, an operator's only option was
to paste a static token into a repository secret, which is worse than what the
parity target offers third parties and defeats the reason L0 exists.

What it settled is worth stating, because the shape differs from the parity
target and the difference is the point. A run exchanges the job token the forge
already gave it for a token belonging to the app, bounded by its task rather
than only by a clock, and scoped by the grant rather than by everything the app
can reach. No key is stored, because there is none to store: GitHub needs one
only because their Apps are external to the repository, and an organization's
own app is not.

What remains in this layer is sessions themselves, the egress routing below,
the no-self-approve rules, and a settings surface for managing grants.

The largest slice, and the one this document originally under-scoped.
Assigning an issue or a review to an agent starts a *session*: a dispatched
Actions run with a session record in front of it, carrying status, live logs,
a steer-or-stop control, and links to whatever it produced. Forgente already
runs the sandbox — Actions with a runner fleet is exactly the substrate
GitHub uses — and GitHub's own choice to compile agentic workflows down to
ordinary Actions YAML is direct evidence this is the right substrate.

**The dispatch trigger already exists (checked 2026-07-26).** This layer was
scoped as though assignment-to-an-agent had to be built. It does not:

- `CanBeAssigned` excludes organizations but **not bot accounts**, so an app —
  a `UserTypeBot` account with write access through team membership — can
  already be assigned to an issue today.
- `IssueChangeAssignee` already emits an Actions notification with action
  `assigned` under `HookEventIssueAssign`.

So a workflow declaring `on: issues: types: [assigned]` already runs when an
issue is assigned to an agent. What L3 adds is not a trigger but a *record*
attached to the run that already happens, plus the runner routing above. That
is the second time checking the substrate has shrunk this layer, after the
permissions half.

**One gap only the end-to-end run found (2026-07-26).** Reading the code
established that assignment dispatches a workflow, and it does. What it did not
establish is that a workflow can tell *who* was assigned: `IssuePayload` carried
no `assignee`, so the GitHub-idiomatic guard
`if: github.event.assignee.login == '<agent>'` silently matched nothing and the
job was skipped rather than failing. Running it end to end is what surfaced
this; four separate readings of the trigger path had not. The field is now
populated on assign and unassign for both issues and pull requests, on the
Actions and webhook paths alike.

The lesson generalises past this one field: a trigger that fires is not the
same as a trigger a workflow can *act* on, and only executing one distinguishes
them.

The sandbox *contract* is not optional polish, and matching it is a
precondition for letting agents near real repositories. Less of it is
outstanding than this document assumed — the permissions half is built, per
the ecosystem survey above — so the list is marked with what remains:

- ~~Read-only permissions by default; write scopes granted explicitly.~~
  **Built.** Actions already clamps job tokens against organization and
  repository ceilings. What an agent session adds is choosing the mode
  deliberately at dispatch rather than inheriting the repository's default.
- Egress control on agent runs — the operator decides what the sandbox may
  reach. **This is the real gap, and it is a different shape than assumed.**
  See the substrate check below; the short version is that the forge cannot
  enforce egress and cannot be made to without an ecosystem-wide protocol
  change, so what it can honestly offer is routing and refusal.
- Agents cannot approve, cannot merge, and cannot mark their own work ready
  for review. Branch protection and required checks apply unchanged.
- Untrusted input is treated as untrusted: the issue body an agent reads is
  data, not instruction.
- Every action is attributable — which agent, which organization, which
  session, on whose behalf — and the session log is queryable and exportable.
  A session running on Actions should authenticate *as its app*, not carry a
  personal access token into the runner. GitHub removed exactly that
  requirement from the Copilot CLI in July 2026, and the same reasoning
  applies here: a long-lived personal credential inside a sandbox is the
  thing L0 exists to avoid.
- Confidence and approvals for low-stakes automation: an agent may propose
  rather than apply, with its reasoning attached, and a human accepts or
  declines. This pattern is cheap, it is proven, and it fits a self-hosted
  audience better than silent automation does.

**Egress: what the substrate actually permits (checked 2026-07-26).** Against
`gitea.dev/actions-proto-go` v0.6.0, the wire protocol between forge and
runner:

- The `Task` message the forge sends has seven fields — `id`,
  `workflow_payload`, `context`, `secrets`, `machine` (marked unused), `needs`
  and `vars`. **There is no network or egress field of any kind.**
- The `Runner` message sent at registration carries `id`, `uuid`, `token`,
  `name`, `status`, `agent_labels`, `custom_labels`, `version`, `labels` and
  `ephemeral`. **No network capability is reported either.**

So the earlier framing — "the runner exposes `network_mode` and `privileged`"
— understated it. Those are act_runner's own configuration; the forge never
sees them and has no field in which to express a policy about them. Egress is
not an unimplemented forge feature. It is **not expressible** in the protocol,
in either direction.

That leaves three routes, and naming them is what makes the layer scopeable:

1. **Extend the protocol upstream.** Correct, and slow: `actions-proto-go` is
   shared with Gitea and the runner, exactly the wire surface
   [AGENTS.md](../AGENTS.md) says not to diverge from unilaterally.
2. **Fork the runner.** The fork-on-divergence bar in
   [FORGENTE.md](../FORGENTE.md) governs, and a permanent merge burden for one
   field is a poor trade.
3. **Use the mechanism that already exists: labels.** An operator configures a
   network-restricted runner — act_runner can already do this — and registers
   it under a label. The forge routes agent sessions to runners carrying that
   label, refuses to dispatch a session when none is available, and shows which
   runners qualify. No protocol change, no fork.

Route 3 is the one to build, and its limit has to be stated plainly rather
than discovered later: **labels are self-asserted, so this is attestation, not
enforcement.** A runner that claims the label without restricting anything
defeats it. That is not a new weakness — the forge already trusts registered
runners to execute arbitrary workflow code — but it means the honest claim is
"agent sessions only dispatch to runners the operator has designated as
network-restricted", never "Forgente enforces an egress firewall". The
enforcement lives in the operator's runner deployment, and the forge's job is
to make the designation explicit, routable and visible.

**How strong that is depends on who operates the runners, and that is what makes
it shippable.** Attestation is weak when the forge operator and the runner
operator are different parties, because the forge is then trusting somebody
else's assertion about somebody else's machines. Where they are the same party —
an organization running its own fleet, or Forgente running the hosted instance —
the label is an operator's assertion about their own infrastructure, and real
enforcement exists; it simply lives in the runner deployment rather than in the
forge. This document previously left open whether shipping sessions without
egress control was defensible. The answer is that it depends on the arrangement,
the forge's obligation is identical in both, and the claim must be worded per
operator rather than per product.

### L4 — Tenants

*Tenants of the substrate, not multi-tenancy.* The name has already been
misread once; nothing in this layer is about hosting several customers on one
instance. See the deployment posture below for that.

The visible features everyone actually names: AI code review, issue triage
and labelling, pull-request summaries. Each is an agent running on L0–L3
rather than a feature compiled into the server, and each ships as a reference
implementation in its own repository. Keeping them out of the tree is
deliberate: it proves the substrate is genuinely usable by third parties, and
it keeps model-specific behavior out of the forge's release cycle.

**One bound worth knowing before promising anything about projects.**
Repository projects — the kanban board — have **no REST API at all**; there
are no project handlers under `routers/api/v1/`. An agent cannot read board
ordering, move an item between columns, or file a new issue into one. Issues,
labels and comments are all API-reachable, so triage that works in those terms
is fine; anything phrased as "the agent manages the board" is not, and would
need that API to exist first.

This is an ecosystem gap rather than a Forgente one, and upstream already
tracks it in go-gitea/gitea#37151 — filed explicitly because agentic workflows
need kanban ordering over the API to pick up work in priority order. Building
it is a reasonable contribution; assuming it exists is not.

## Deployment posture: self-hosted first, hosted as a deployment of it

Forgente may operate forgente.com as a hosted instance alongside self-hosting.
Whether to do so is a business decision rather than a layer — it is a service
obligation, not a feature — but it imposes one architectural constraint worth
settling before anything depends on it, because getting the order backwards is
expensive and hard to reverse.

**Every layer must work when the forge operator, the runner operator and the
agent vendor are three different parties.** Hosted operation is then the
degenerate case where Forgente happens to be more than one of them: the same
code path with different parties behind it, never a separate implementation.

The failure mode is documented in the parity target itself. GitHub built hosted
first, their agent architecture assumes they operate everything, and there is
consequently no self-hosted shape of it to ship — Copilot does not exist on
Enterprise Server at all. A hosted-first Forgente would reproduce exactly that,
with the self-hosted path degrading into a lagging port of the real one.

Two consequences, both checkable rather than aspirational:

- Build the third-party arrangement first and let hosted inherit it. Concretely,
  `AN-IDENT-2` is needed by every arrangement *except* the one Forgente would
  operate itself, so it must not wait on any hosting decision.
- A hosted instance may operate runners, supply a default app, and pre-configure
  a provider for convenience. What it must never do is make that path better
  than the one an operator assembles for themselves — see the inference
  invariant in the non-goals, which is what the neutrality position rests on and
  which binds a hosted instance exactly as it binds the software.

Operating an instance is not the "agent hosting" excluded in the non-goals.
That exclusion is about becoming a marketplace and a platform business for other
people's agents. Running one's own software is an ordinary thing for an
open-source project to do, and it has a second use — it is the standing proof
that the self-hosted path works, but only for as long as it runs the same
artifact everyone else runs.

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

What is left is narrower, and durable. One of these is the position; the others
are what make it credible rather than four unrelated boasts.

**The position — Forgente runs whichever agent you already chose.** Nothing in
the forge requires buying a model or a harness from Forgente, and no capability
is gated behind doing so, which leaves it with nothing to gain by steering
anyone. Every serious competitor in this category has the opposite arrangement:
Copilot is GitHub's product, Duo is GitLab's, Rovo is Atlassian's, and in each
case the agent capabilities are inseparable from the licence. Neutrality is
therefore not merely unclaimed by them, it is structurally unavailable to them —
which is what makes it durable rather than a feature copied next quarter.

Note the claim is "you never have to buy ours", not "we will never sell one".
The first is a property that can be held to and checked; the second is a promise
about the future, and the non-goals below explain why that distinction is the
one doing the work.

State it precisely or it does not survive contact. GitHub's Agent HQ *does* host
third-party agents, Codex and Claude among them. The difference is that theirs
is a **curated** set on infrastructure they operate, and Forgente's is
**whatever you can run, on infrastructure you operate**. "Only Forgente supports
third-party agents" is false and must not be said.

The rest are what that position rests on:

- **Self-hosted — but state it precisely.** GitHub now runs agents in *local*
  sandboxes on the developer's own machine, included in the standard seat,
  and code review can run on self-hosted runners, so "it runs on your
  hardware" is no longer ours alone at the execution layer. What is still
  true, and larger: the **forge** is cloud-only. Copilot does not exist on
  GitHub Enterprise Server. An operator who must keep the forge itself inside
  their network cannot have GitHub's agent stack at any price. For regulated,
  air-gapped and sovereignty-constrained users that is not a preference, it
  is the only option — and it is the *support* to lead with, rather than the
  weaker one about where code executes.
- **No subscription gate.** GitHub's agent capabilities are inseparable from
  Copilot licensing, AI credits, and cost centers. Forgente's agent
  primitives are part of the forge.
- **Open formats over a proprietary stack.** `AGENTS.md`, the Agent Skills
  format, and MCP are open; Marketplace agent apps, plugin standards, and the
  Copilot SDK are GitHub's. Building on the open half is both cheaper and
  more defensible for a fork with Forgente's resources. It is also the
  mechanism the position above depends on: neutrality is a claim about formats
  before it is a claim about intent, and an agent can only be swapped for
  another if what the repository declares is not Forgente's own dialect.
- **A primitive this codebase never had.** Neither Gitea nor Forgejo has an
  owned agent principal: an account belonging to an organization, permissioned
  through ordinary membership, badged where it acts, revocable in one place.
  An earlier draft claimed installation identity was missing from this lineage
  *entirely*; that is wrong and should not be repeated. Forgejo v16 ships
  Authorized Integrations, and upstream #38181 will ship admin-managed bot
  accounts. Both are real machine identity. Neither is *owned* identity, which
  is the narrower and still-true claim.

Forgejo, by contrast, prohibits AI-generated contributions outright — and is
nonetheless building the plumbing agents need, which is worth remembering
before reading their policy as a vacated field. Between a proprietary hosted
stack and a prohibition there is real room, but the room is "self-hosted and
open", not "the only one who governs agents" and not "the only one building
machine identity".

## Non-goals

- **Editor and client tooling.** See above.
- **Requiring Forgente's own inference.** Providers are configured, not
  supplied. What protects the position, though, is the invariant behind that
  rule rather than the rule itself, and it is written as an invariant on
  purpose: **bring-your-own stays fully capable, and no forge capability is
  gated behind buying inference from Forgente.** A flat ban on ever selling
  inference is the kind of commitment that gets reversed under commercial
  pressure and costs credibility when it does; these two conditions can be
  checked instead, and they bind forgente.com exactly as they bind the
  software. A hosted instance pre-configuring a provider for convenience
  breaks neither of them. A managed path that is *better* than what an
  operator can assemble breaks both.
- **An agent marketplace, agent hosting, or a plugin economy.** GitHub has
  Marketplace agent apps and enterprise plugin standards; that is a platform
  business, not what a self-hosted forge is for. Note this is a *choice*, not
  a claim that the category is dead — an earlier draft wrongly recorded
  Copilot Extensions as deprecated in favour of MCP, when in fact GitHub's
  extensibility surface grew. Note also what it does *not* exclude: operating
  an instance of Forgente is not agent hosting, and the deployment posture
  above governs that instead.
- **A new permission system.** Agents use teams and collaborators. The spike
  confirmed the existing machinery has no user-type gate.
- **Autofix.** It presupposes code scanning, which Forgente does not have.
  Scanning may be worth building, but on its own merits.
- **Chasing every surface.** GitHub shipped Copilot Memory, vision,
  repository overviews, desktop parallel workstreams, and a six-language SDK
  in the survey window alone. Parity across all of it is not achievable and
  not the point; the layers above are.

## Open questions

- ~~Naming and positioning.~~ **Settled.** Lead with the neutrality position —
  Forgente runs whichever agent you already chose — and carry installation
  identity as the infrastructure claim underneath it rather than as the
  headline. "Agents" as a word is unavoidable and fine; the hype scepticism it
  invites is answered by being the substrate rather than by avoiding the term.
- Whether L2's provider configuration belongs to the instance only, or to
  organizations as well, and how quota and cost controls work if the latter.
- ~~How much of the L3 sandbox contract is required for a first release.~~
  **Partly settled.** The egress half is answered under L3: the claim is worded
  per operator and is defensible wherever the operator runs the runners. What
  is still open is the rest of the contract, and in particular whether
  propose-and-approve ships alongside the first sessions or after them.
- Whether Forgente operates forgente.com as a hosted instance at all. The
  architecture is settled either way by the deployment posture above, so this
  is a question about appetite for a service obligation — support, uptime,
  abuse, billing — and not an engineering decision waiting on an answer.
  Nothing in the layers should block on it.
- Whether to adopt the natural-language-compiled-to-YAML workflow shape at
  all, or to keep agent invocation explicit and leave workflow authoring to
  humans.
- Whether MCP is the primary agent surface or merely the first one. Driving
  the `tea` CLI through a published skill needs no server, no protocol, and
  no OAuth work, and is what upstream suggests; MCP buys reach into hosted
  clients that cannot run a CLI. The honest answer is probably both, but the
  ordering is not decided.
- Whether Forgente wants inbound workload identity at all — Forgejo's
  Authorized Integrations shape, letting an external CI system authenticate
  with a short-lived JWT — or whether org-owned apps holding scoped tokens
  cover enough of the same ground to skip it.
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
