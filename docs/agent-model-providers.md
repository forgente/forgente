# Model providers: design

**Status: proposed, not built.** This is the last unstarted piece of L2 in
[agent-native-program.md](agent-native-program.md). It exists to be argued
with — one decision in it is not mine to make, and it is marked.

The layer's requirement, as recorded: *endpoint, credential, and model
selection at the instance and organization level. Credentials are secrets and
get handled as such: encrypted at rest, never rendered back, never logged, and
**never exposed to repository code**.*

That last clause is the whole design. Everything below follows from taking it
literally.

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
|---|---|---|---|---|
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

### Recommendation: B, sized honestly

Take Option B. It is still the right side of the fork — but it is bigger than
the earlier draft claimed, and the difference should be budgeted rather than
discovered.

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

The measurement makes this question sharper rather than softer. Option B now
costs two wire APIs and a streaming data path, not one buffered shape — so the
gap between the two options is wider than when the question was first asked,
and the answer decides more.

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
