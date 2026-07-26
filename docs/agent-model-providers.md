# Model providers: design

**Status: proposed, not built.** This is the last unstarted piece of L2 in
[agent-native-program.md](agent-native-program.md). It exists to be argued
with — two decisions in it are not mine to make, and they are marked.

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
  works unmodified — they already read `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`
  and friends from the environment. Zero inference surface for the forge to
  maintain. No latency added, no streaming to proxy, no provider API shapes to
  track.
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
  timeouts, retries, token accounting, and at least one provider API shape
  become the forge's problem. This is a real, ongoing surface, and it is the
  reason to be suspicious of my own preference for it.

### Recommendation: B, deliberately small

Take Option B, and hold its surface down to one thing: **an
OpenAI-compatible endpoint in, a configured provider out.**

The forge should not attempt to normalise every provider's API. It should
accept the one request shape that the largest number of harnesses can already
emit, and forward. Where a configured provider speaks something else, that is a
per-provider adapter, added when a real user needs it — not a framework built
in advance.

> **Verify before building.** My understanding is that most current providers
> either expose an OpenAI-compatible endpoint or are commonly fronted by
> something that does, and that some — Anthropic's own API, Bedrock — do not
> natively. That claim is from training data, not from reading their docs
> today, and it is load-bearing for how small this stays. Check it against
> primary sources first; if it is wrong, Option B gets more expensive and the
> fork deserves re-litigating.

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

## Open questions — these are yours, not mine

1. **Is the recorded requirement the real one?** Option B follows from *"never
   exposed to repository code."* If that clause is aspirational rather than
   binding, Option A is dramatically cheaper and the whole design collapses to
   a table and a delivery step. I would rather be told the requirement is
   softer than build a proxy nobody wanted.

2. **What has to run for this to count as done?** Neutrality is a claim until
   two *different* harnesses run against one configuration. Which two is a
   product decision. My instinct is one harness that speaks OpenAI-compatible
   natively and one that does not, because the pair proves the adapter boundary
   is real rather than assumed — but the specific choices should be agents you
   expect early users to actually bring.

## Prior finding worth not repeating

Three times in this program, the substrate already contained what a layer
planned to build: assignment-as-trigger, self-approval refusal, and runner
label routing. Before building any of the above, check whether some part of it
already exists — the check is cheap and it has paid every time. The egress
correction (#97) is the worked example: the spec named routing, routing was
already there, and the actual gap was one field away in a different place.
