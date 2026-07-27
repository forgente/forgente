# Model providers: design

**Status: proposed, not built.** This is the last unstarted piece of L2 in
[agent-native-program.md](agent-native-program.md). It exists to be argued
with — one decision in it is not mine to make, and it is marked.

The layer's requirement, as recorded: *endpoint, credential, and model
selection at the instance and organization level. Credentials are secrets and
get handled as such: encrypted at rest, never rendered back, never logged, and
**never exposed to repository code**.*

That last clause is the whole design, because taking it literally forces a
proxy and relaxing it forces nothing. The document works through what each
reading costs and recommends relaxing it — but the clause is a policy, and the
policy is the reader's to set.

## The obvious implementation is not available

A provider credential looks exactly like an organization secret: a name, an
encrypted value, set once by an owner. The table exists, the encryption exists,
the settings UI exists. Reusing it would take an afternoon.

It also breaks the requirement outright. `GetSecretsOfTask`
(`models/secret/secret.go:169`) loads **every** secret belonging to the
repository's owner into **every** task that repository runs:

```go
ownerSecrets, err := db.Find[Secret](ctx, FindSecretsOptions{OwnerID: task.Job.Run.Repo.OwnerID})
```

Organization secrets are exposed to repository code *by design* — that is what
they are for. So an organization's provider key, stored that way, is readable
by anyone who can land a workflow in any repository the organization owns. One
`echo` in one workflow, in one repository, and the key is in a log.

This is not a flaw in the secrets machinery. It is the correct behaviour for
secrets, and the wrong container for this credential.

**Consequence: model providers cannot be a thin wrapper over Actions secrets.**
If that seems like an obstacle to route around, note that it is the same
requirement that made `AN-IDENT-2` worth building rather than telling operators
to paste a token into a secret — and that turned out to be the most valuable
thing in the program so far.

## The fork

Something has to hold the credential at the moment a model is called. There
are only two places, and they are not close together.

### Option A — deliver the credential to the run

The forge hands the provider key to the job, in the environment, and the
harness uses it exactly as it would outside Forgente.

- **For:** nothing to build beyond storage and delivery. Every existing harness
  works unmodified — a credential in the environment, or in the harness's own
  config file, is the arrangement they are all built for. Zero inference
  surface for the forge to maintain. No latency added, no streaming to proxy,
  no provider API shapes to track.
- **Against:** it *is* the thing the requirement forbids. A run is repository
  code. Anyone who can edit a workflow can print the key. Narrowing delivery to
  designated runners (`AN-GOV-5`, shipped) reduces the blast radius but does not
  change the property: the credential is in a place the operator does not
  control.

Option A is honest only if the requirement is rewritten to say what it would
then mean: *credentials are delivered to runs the operator has designated, and
are as exposed as any repository secret.* That is a defensible policy. It is
not the recorded one.

### Option B — the forge proxies inference

The credential never leaves the forge. A run receives a short-lived, scoped
token and points its harness at a Forgente endpoint, which forwards to the
configured provider and returns the response.

- **For:** the requirement holds literally. The run never sees a provider key,
  so a leaked workflow leaks a scoped, expiring token against one
  organization's configured provider rather than the key itself. Revocation is
  real: suspend the app or drop the config and calls stop, without rotating
  anything at the provider.
- **For, structurally:** this is a pattern already built, proven, and running
  in this codebase. A run presents the token the forge gave it and exchanges it
  for a narrower credential bound to its task — that is `AN-IDENT-2` exactly
  (`services/user/app_run_token_forgente.go`). The second use of a shape is
  much cheaper than the first, and every governance property already attached
  to it (fork-PR refusal, suspension, runner-label designation, grant scoping)
  comes along without new work.
- **Against:** the forge joins the inference data path. Streaming, latency,
  timeouts, retries, token accounting, and — as measured below — two provider
  API shapes become the forge's problem. This is a real, ongoing surface, and
  it is the reason to be suspicious of my own preference for it.

### What the harnesses actually send

An earlier draft of this document recommended holding Option B down to *"an
OpenAI-compatible endpoint in, a configured provider out"*, on the reasoning
that it is the one shape the largest number of harnesses can already emit. That
reasoning was from memory, and it was wrong.

Measured on 2026-07-27 by pointing each harness at a recording endpoint and
reading what arrived:

| Harness | Endpoint | Wire shape | Auth | Streaming |
| --- | --- | --- | --- | --- |
| Claude Code 2.1.220 | `POST /v1/messages?beta=true` | Anthropic Messages | `Authorization: Bearer` | `stream: true` |
| Codex 0.145.0 | `POST /v1/responses` | OpenAI **Responses** | `Authorization: Bearer` | `stream: true` |

**Neither speaks OpenAI Chat Completions.** Codex refuses it outright — a
provider configured with `wire_api = "chat"` fails to load with *"`wire_api =
"chat"` is no longer supported"*. So the shape the earlier draft chose as the
common denominator is one that neither of the two most likely harnesses would
have used.

Two things survive the correction, and they are the ones that matter:

- **Both can be pointed at an operator-controlled URL**, and both then send
  *everything* there — Claude Code through `ANTHROPIC_BASE_URL`, Codex through
  a `model_providers` entry in its config. Option B is mechanically possible
  for both.
- **Both authenticate with a bearer token.** That is the same shape as the app
  token a run already exchanges its job token for (`AN-IDENT-2`), so the
  credential half of this needs no new mechanism.

### How the neighbours answer it (surveyed 2026-07-27)

Read from primary sources on the date above, per the standing rule about dated
competitive claims. Both are worth knowing because they converge on the same
rule from opposite directions.

**GitHub runs two mechanisms, split by who owns the runtime.**

- *Enterprise and organization BYOK is a proxy.* The key is stored server-side
  and GitHub's Copilot API calls the provider: "Enterprise BYOK is handled
  server-side and affects the models served to users by the Copilot API."
  Providers accepted are OpenAI-compatible endpoints, Azure OpenAI and
  Anthropic, including local Ollama; models must support tool calling and
  streaming. Note that this proxy serves *their own* harness family, not
  arbitrary agents.
- *Local BYOK is not a proxy.* "These keys are only handled client-side: they
  are stored locally and the associated models aren't available to other
  users." The client calls the provider directly.
- *Third-party agents in Actions get the key.* `anthropics/claude-code-action`
  takes `ANTHROPIC_API_KEY`, or Bedrock/Vertex/Foundry credentials, as an
  ordinary repository or organization secret referenced from workflow YAML.

So GitHub proxies when it owns the runtime and hands the credential over when
somebody else does.

**Gitea has not declined to build a proxy — it has declined the whole feature.**
There is no model provider concept anywhere in the tree. In go-gitea/gitea#36444
(AI code review, open since January 2026) a maintainer writes: "I don't think we
will have a full first-party review feature like GitHub Copilot, that is far too
much work and there already exist implementations that leverage the MCP and/or
API… such loosely coupled integrations will be best." The abuse question this
document exists to answer was raised there on the first day — "If an API key is
instance wide, how do we prevent abuse of it?" — and the answer taken was to
hold no key at all: a bot account, a CI task, and the model configuration
outside the forge. Their lead maintainer later pointed at `gitea-tea-skill`, the
same `tea`-driven path recorded in the L1 notes of the program document.

**Consequence: there is no upstream to inherit or stay compatible with.** Unlike
apps, tokens and webhook payloads, nothing constrains this design. Whatever
Forgente builds here is a divergence by construction.

### What Gitea does with credentials it *is* given

Worth separating from the above, because it is a real house style and it
transfers regardless of which option is chosen. Every third-party credential in
the codebase is encrypted at rest with `SECRET_KEY`, decrypted at exactly one
consumption point, used by the forge itself, and never rendered back:

| Credential | Decrypted at | Who acts |
| --- | --- | --- |
| Webhook `Authorization` header | `services/webhook/deliver.go` — one caller | the forge sends the request |
| Migration token, password, AWS keys | `services/migrations/*` | the forge clones the source |
| LDAP bind password | `services/auth/source/ldap/source.go` | the forge binds |

None of the three is exposed through `services/convert` or the API structs.
**Adopt this posture whichever option wins** — it is what the sketch below
already describes.

But note *why* the credential stays in all three: the forge is intrinsically the
actor. It is the process making the webhook call, the clone, the bind. Nothing
was proxied because nothing needed to be. Inference is the other shape — the
work happens in the runner and the harness is the actor — and Gitea's single
answer for *that* shape is Actions secrets, which is to hand the credential
over. The same split GitHub arrived at, from a codebase that shares none of its
design.

### Recommendation: relax the requirement and take A

**This reverses the previous draft, which recommended B.** Three things changed
it, in order of weight.

*The measurement.* B is not one shape but two, with mandatory streaming — see
above. The earlier recommendation rested on a common denominator that does not
exist.

*The prior art.* Two independent codebases apply the same rule: proxy when you
own the runtime, hand the credential over when you do not. Today every Forgente
operator owns their runtime — their forge, their runners, their organization.
That is the hand-it-over case by both precedents, and `AN-GOV-5`'s runner-label
designation already narrows which runs receive it.

*The asymmetry in what each option commits you to.* A commits you to nothing
external. B publishes an endpoint that agents point at, which means owning that
contract — the wire shapes accepted, the streaming behaviour, and every future
change to either. Codex removing `wire_api = "chat"` is exactly the churn that
would then be absorbed on users' behalf, permanently. **A → B is a feature
addition; B → A is retiring a public endpoint people depend on.**

Set against that, what the credential protects is inference spend — not
repository write access, which is what made `AN-IDENT-2` worth building. The
comparison is not close enough to justify a permanent data-path obligation.

**A is honest only if the requirement is rewritten**, as the fork section says:
*credentials are delivered to runs the operator has designated, and are as
exposed as any repository secret.* Ship A described that way and a later move to
B is a strengthening, which is always safe. Shipping A while wording it as
though it were B is the one outcome to avoid.

**Keeping the door open costs almost nothing**, and should be kept that way. No
speculative abstraction for a proxy that may never exist — the program document
already warns against exactly that for runtimes. Two cheap things only: make
delivery a **per-provider field** so B can later arrive alongside A rather than
replacing it, leaving existing configuration working; and keep the wire-shape
table in the repository, since it is the starting point if hosted ever happens
and will need re-measuring anyway.

### If B is chosen anyway, size it honestly

The requirement as recorded still implies B, and if it is binding then the
following is what it costs — this is the previous draft's recommendation, kept
because it is what B actually takes.

**Two wire APIs, not one.** Anthropic Messages and OpenAI Responses, forwarded
as opaquely as possible. The forge should authenticate the caller, swap in the
configured provider's credential, and get out of the way — it should not parse,
normalise, or unify the two shapes. A request body carrying `thinking`,
`context_management`, `output_config` and 28 tool definitions is not a schema
worth tracking; it is a payload worth forwarding untouched.

**Streaming is mandatory, not an optimisation.** Both harnesses set
`stream: true` on the main path, and Codex fails with *"stream disconnected
before completion"* against a non-streaming endpoint. A buffering proxy does
not work at all. This was listed as a cost in the earlier draft; it is a
requirement.

**Pointing a harness at your endpoint does not confine its egress.** In the
same measurement, Codex independently called
`https://chatgpt.com/backend-api/plugins/featured` — unrelated to the
configured provider. Configuring inference through the forge restricts where
*inference* goes and nothing else. This is worth stating because it is easy to
assume otherwise, and because it is the same distinction `AN-GOV-5` already
makes: what the forge designates, the runner's own configuration enforces.

## What this does *not* decide

**It does not create a Forgente inference business.** The recorded invariant
stands unchanged: *bring-your-own stays fully capable, and no forge capability
is gated behind buying inference from Forgente.* Option B is a proxy for a
credential the operator supplied, to a provider the operator chose.

It is worth being explicit that Option B is *also* the architecture in which a
managed inference option would later be possible, because the forge is already
on the path. That is a consequence, not a motivation, and it is not a reason to
choose B — but pretending not to notice it would be worse than saying it. If
that option were ever taken, the invariant above is what would have to keep
holding, and it is checkable: does bring-your-own still work, and is any forge
feature gated behind buying inference?

## Sketch

Deliberately thin; the decisions above matter more than these details.

**Storage.** A `ForgenteModelProvider` row: owner (organization, or zero for
instance-level), display name, kind, endpoint, encrypted credential, default
model, enabled flag. Encryption reuses `modules/secret`'s
`EncryptSecret`/`DecryptSecret` with `SECRET_KEY` — the same primitive the
secrets table uses, in a table with different delivery rules. Registered via
`db.RegisterModel`, no numbered migration, consistent with the other Forgente
tables.

**Resolution.** Organization provider wins over instance provider, the same
precedence a repository-specific grant already has over an organization-wide
one. An organization with none configured falls back to the instance's, and an
instance with none has the feature simply unavailable rather than half-working.

**Access.** A run exchanges its job token for a scoped inference token, the
same exchange shape as the app token, with the same refusals: not a fork pull
request, task still running, provider enabled, organization not suspended. The
token is bound to the task and dies with it.

**Never rendered back.** The credential is write-only through the UI: set,
replace, clear. No endpoint returns it, decrypted or otherwise. The settings
page shows whether one is set and when, never the value — the same posture as
the app token, which is readable exactly once at creation.

**Configuration surface.** Organization settings, beside Apps. The natural home
is the page that already exists rather than a new section.

## What has to run for this to count as done

Neutrality is a claim until two *different* harnesses run against one
configuration. The measurement above settles which two: **Claude Code and
Codex**, because they span the two wire APIs rather than merely being two
popular names. A proxy that serves both has demonstrated the adapter boundary
is real; a proxy that serves either one alone has demonstrated nothing about
neutrality, however well it works.

## The open question — this one is yours

**Is the recorded requirement the real one?** Option B follows from *"never
exposed to repository code."* If that clause is aspirational rather than
binding, Option A is dramatically cheaper and the whole design collapses to a
table and a delivery step. I would rather be told the requirement is softer
than build a proxy nobody wanted.

Everything found since that question was first asked argues for relaxing it.
Option B costs two wire APIs and a streaming data path rather than one buffered
shape; both GitHub and Gitea hand the credential over whenever somebody else
owns the runtime, which is every Forgente deployment today; and B is the option
with a one-way component, since a published inference endpoint is a contract
that cannot be quietly withdrawn.

None of that decides it. The clause is a policy, and policy is yours: an
operator who intends to run other people's runners, or to offer a hosted
instance, is in the three-party case where the recorded wording is exactly
right and the cost is worth paying. What should not happen is the requirement
staying as written while A gets built underneath it.

## Prior findings worth not repeating

**Check whether it already exists.** Four times in this program, the substrate
already contained what a layer planned to build: assignment-as-trigger,
self-approval refusal, runner label routing, and permissions. The egress
correction (#97) is the worked example: the spec named routing, routing was
already there, and the actual gap was one field away in a different place.

**Measure the thing you are designing around.** The wire-shape table above
replaced a claim this document had already committed to, and the claim was
load-bearing — it was the entire argument for how small Option B could be.
Recalled facts about other people's software age badly and give no signal when
they have. Pointing the two harnesses at a recording endpoint took under an
hour and changed the recommendation; reading about them for a day would not
have, because the earlier draft was written by exactly that method.

**Check what the neighbours actually do before deciding it is unprecedented.**
The survey above cost one afternoon and produced the strongest argument in this
document — two unrelated codebases applying the same rule about when a forge
holds a credential and when it hands one over. That argument was not available
from reasoning about the options, and it reversed the recommendation a second
time. It also settled a question this document had been treating as open:
whether anything upstream constrains the design. Nothing does, and knowing that
for certain is worth more than assuming it.
