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

Re-checked against the changelog on 2026-07-25; nothing below needed
correcting. The entries that postdate the original survey all fit the map
rather than breaking it: the GitHub MCP server moved to the next MCP
specification (Jul 23), agent automation controls for issues reached public
preview (Jul 23 — this is the approvals queue described below), the Copilot
CLI stopped needing a personal access token inside Actions (Jul 2), and code
review gained customization (Jul 17). Only the third implies work that is not
already tracked here; see L3.

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

## Parity map

| GitHub | Forgente today | What is missing |
| ---- | ---- | ---- |
| Agent apps installed as GitHub Apps; cloud agent acts under Copilot identity with the requester as co-author | Organization-owned apps shipped in #66; OAuth2 apps still act *as the authorizing user* | Per-installation scoping and user-owned apps; the primitive itself now exists |
| Cloud agent: issue → sandboxed run → draft PR | Actions and a runner fleet already run untrusted workloads | Assignment, a session record, draft-PR wiring |
| Agentic Workflows: Markdown compiled to Actions YAML | Actions runs hand-written YAML | The compile step and the safety envelope around it |
| Session streaming and audit | Actions logs only | A session as a first-class, queryable object |
| Custom agents (`.github/agents/NAME.md`, org and enterprise levels) | `AGENTS.md` is already an in-repo convention | Read agent profiles and offer them at assignment time |
| Plugins, Agent Skills, Hooks | — | Adopt the open Agent Skills format rather than inventing one |
| First-party MCP server | `gitea-mcp` works unforked against Forgente, validated as an org-owned app | Not a fork: OAuth 2.1 resource-server support so remote MCP clients can connect at all |
| Enterprise plugin standards: central policy for agent tooling | Scoped Workflows already push central workflows across repositories | A policy vocabulary for agents, not only for workflows |
| Cloud agent runs under least privilege | Actions token permissions, org and repo ceilings, fork downgrade, fork-run approval | Nothing — this is built; egress control is the gap (see L3) |
| Copilot code review | — | A *tenant* of the layers above, not a layer itself |
| Agentic autofix for scanning alerts (preview, July 2026) | no code scanning of any kind | Code scanning is the prerequisite; that is security work, not AI work |

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

The forge is the right place to fix this and the standalone server is not,
because the missing pieces are the forge's to serve. `gitea-mcp` issue #207
proposes the server act as an OAuth 2.1 *resource server* delegating to the
instance as authorization server; that requires the instance to publish
`/.well-known/oauth-protected-resource` (RFC 9728), which Forgente does not,
and it works around Dynamic Client Registration (RFC 7591) only because this
lineage lacks it. Forgente already is an OAuth2/OIDC provider with PKCE, and
its API already accepts OAuth2 bearer tokens, so the distance is short.

Forgejo's Authorized Integrations is the nearest prior art in this lineage —
short-lived, externally-issued, forge-validated credentials with per-issuer
capability grants — and worth studying before designing this, even though it
solves inbound workload identity rather than delegated user authorization.

**L1 is therefore forge-side authorization work, not a server.** In order:

1. Serve RFC 9728 metadata and answer unauthenticated MCP requests with `401`
   and `WWW-Authenticate`, so a compliant client can discover where to
   authenticate. This unblocks remote clients generally, not only MCP.
2. Decide whether Dynamic Client Registration is worth supporting, or whether
   pre-registered OAuth applications are enough. Pre-registration is enough
   for Claude today and needs no new machinery.
3. A "Connect an agent" panel on the app's settings page, for the local and
   stdio case that already works — host, token, and a `GITEA_TOOLS` value
   derived from the token's real scopes, with an integration test pinning the
   scope-to-tools mapping.

Item 3 addresses the second finding from the run: the server advertises all
of its tools regardless of what the token can do, so `create_repo` was
offered and then refused at call time for a missing scope. Token scopes are
enforced by the forge, `GITEA_TOOLS` filters tools in the client, and nothing
reconciles the two, so an agent discovers its limits by failing and an
operator reading the tool list gets a false picture of what the token
permits. Only the forge knows a token's real scopes, so only the forge can
emit a list that agrees with them by construction. This is already filed
upstream as `gitea-mcp` issue #79, and MCP tool annotations (issue #173) may
be a better carrier than an environment variable — prefer contributing there
over keeping a Forgente-only workaround.

A third, smaller finding: an app has no repository access until it is added
to a team, which is correct — access is granted the ordinary way — but it is
an undocumented step between creating an app and having a working agent. The
panel should name it or handle it.

Watch item, not a fork trigger: `gitea-mcp` builds on `mark3labs/mcp-go` with
open requests to move to the official Go SDK, while GitHub's server already
tracks the next MCP specification. Protocol drift is worth monitoring.

### L2 — Repository agent configuration and providers

Two independent pieces of configuration:

- **Agent definitions in the repository.** `AGENTS.md` already carries build,
  test, and style instructions. Named agent personas belong beside it, and
  the forge surfaces them wherever an agent can be assigned. Follow the
  established shape — a Markdown profile with prompt, tool allow-list, and
  MCP server declarations — and adopt an existing convention rather than
  inventing a Forgente-specific one. Interoperating with agents people
  already run is worth more than a bespoke schema.

  Two ecosystem facts should settle which convention. `gitea-tea-skill`
  publishes an official skill teaching an agent to drive the `tea` CLI, laid
  out under `.agents/skills/` — so the directory convention here is
  `.agents/skills/`, not GitHub's `.github/agents/`, and the file is plain
  Markdown with no frontmatter, so it is *not* Anthropic's `SKILL.md` schema
  despite sharing the "Agent Skills" name. Do not conflate the two.

  That skill also implies a second, cheaper integration path this document
  had not considered: an agent can drive `tea` instead of MCP, which needs no
  server and no new protocol. It is the path upstream suggested in #36444.
  Worth evaluating on cost before assuming MCP is the only surface.
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
precondition for letting agents near real repositories. Less of it is
outstanding than this document assumed — the permissions half is built, per
the ecosystem survey above — so the list is marked with what remains:

- ~~Read-only permissions by default; write scopes granted explicitly.~~
  **Built.** Actions already clamps job tokens against organization and
  repository ceilings. What an agent session adds is choosing the mode
  deliberately at dispatch rather than inheriting the repository's default.
- Egress control on agent runs — the operator decides what the sandbox may
  reach. **This is the real gap.** Nothing in the tree expresses an egress
  policy; the runner exposes container `network_mode` and `privileged`, which
  are deployment settings, not policy the forge can state or enforce. Of the
  whole contract this is the piece that most needs designing, and shipping
  agent sessions without it is probably not defensible.
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
