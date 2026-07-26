# Apps: giving an agent its own identity

An **app** is an account an organization owns. It acts as itself rather than on
behalf of a person: what it does is attributed to it, its permissions are
granted the ordinary way, and it can be stopped in one place.

This is what an agent should run as. The alternative — handing an agent a
person's access token — attributes the agent's work to that person, gives it
everything that person can reach, and leaves nothing to revoke short of
rotating their credentials.

Apps are useful to ordinary automation too. Nothing here is agent-specific;
agents are simply the case that makes the difference obvious.

This guide covers using them. The design reasoning is in
[agent-native-program.md](agent-native-program.md).

## Create an app

Organization → **Settings** → **Applications** → **Apps**. Give it a name and a
description. The name becomes a real account name, so it follows the same rules
as a username.

Creating an app grants it nothing. It cannot create repositories or
organizations of its own, and it has no access to anything until you give it
some.

## Give it access

Add the app to a team, the same way you would add a person. Its permissions are
whatever that team grants.

This step is easy to miss: a freshly created app authenticates successfully and
then cannot see a single repository, which reads like a broken token rather
than a missing membership.

## Let a workflow act as the app

A workflow can obtain a short-lived token for an app, without any long-lived
credential being stored in the repository.

### Grant the repository

On the app, open **Repository access for Actions** and add a grant. Name a
repository, or leave it empty to cover every repository the organization owns
including ones created later. Choose the narrowest permissions the work needs.

Read the warning on that form. A grant means **anyone who can add a workflow to
that repository can act as the app**, up to the permissions you chose. That is
the same trust boundary as a repository secret, and it should be given the same
scrutiny.

A grant naming a repository takes precedence over an organization-wide one, so
you can grant broadly and then narrow a single repository.

### Exchange the job token

Every job already receives `secrets.GITEA_TOKEN`, scoped to its own repository.
Post it to the app-token endpoint to receive a token belonging to the app:

```yaml
name: agent
on:
  issues:
    types: [assigned]

jobs:
  work:
    # without this, every assignment to anyone starts the job
    if: github.event.assignee.login == 'myagent'
    runs-on: ubuntu-latest
    steps:
      - name: Get a token for the app
        id: app-token
        run: |
          resp=$(curl -sSf -X POST \
            -H "Authorization: Bearer ${{ secrets.GITEA_TOKEN }}" \
            -H "Content-Type: application/json" \
            -d '{"app": "myagent"}' \
            "${{ github.server_url }}/api/v1/repos/${{ github.repository }}/actions/app-token")
          token=$(printf '%s' "$resp" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
          echo "::add-mask::$token"
          echo "token=$token" >> "$GITHUB_OUTPUT"

      - name: Act as the app
        env:
          TOKEN: ${{ steps.app-token.outputs.token }}
        run: |
          curl -sSf -X POST \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d '{"body": "on it"}' \
            "${{ github.server_url }}/api/v1/repos/${{ github.repository }}/issues/${{ github.event.issue.number }}/comments"
```

This parses with `sed` rather than `jq` on purpose. Common runner images —
including the `node:*` ones act_runner defaults to — do not ship `jq`, and the
step fails with `jq: command not found` before it ever reaches the forge. Use
`jq` if your image has it; do not assume it does.

Mask the token as soon as you have it. It is short-lived, but a log is
permanent.

The same token works for git over HTTP, so the app can commit and push as
itself:

```yaml
      - run: |
          git clone "https://x-access-token:${{ steps.app-token.outputs.token }}@example.com/${{ github.repository }}.git" work
          cd work
          git config user.name myagent
          git config user.email myagent@noreply.example.com
```

**Set the committer address to the app's own, or the commits will not be
attributed to it.** A commit is linked to an account by its email address and
by nothing else, so a plausible-looking address the app does not own produces
commits from an author the forge cannot resolve — they render as a bare name,
with no profile behind them and no bot label. Push access and attribution are
separate things, and getting the first right tells you nothing about the
second.

The address to use is the app's account name at the instance's no-reply domain:
`myagent@noreply.example.com` for an instance serving `example.com`. The domain
comes from `NO_REPLY_ADDRESS`, which defaults to `noreply.` followed by the
instance's own domain — so it is *your* domain, not `example.com`, unless you
set it explicitly.

**You cannot look this address up.** Apps are created with a private email, so
it appears neither on the app's profile nor in the API — construct it from the
app's name and your no-reply domain. Nothing will tell you that you got it
wrong: the push succeeds either way, and the only symptom is a commit whose
author never resolves to an account.

### Keep an app's work on runners you trust

A grant can name a **required runner label**. When it does, a run can act as the
app only while it is executing on a runner carrying that label:

1. Configure a runner however your policy requires — a restricted egress
   network, an isolated host, whatever you enforce — and register it with a
   label such as `egress-restricted`.
2. On the grant, set that label as the required one.
3. Point the workflow at it: `runs-on: egress-restricted`.

Step 3 alone is not a policy. Anyone who can edit the workflow can change
`runs-on` back to an ordinary runner, and before the grant field existed the app
token was minted anyway. Setting the label on the grant is what makes step 3
load-bearing: a run on any other runner is refused the app's identity, so the
workflow cannot quietly relocate itself out of the restriction.

**This is a designation, not an enforcement.** Labels are self-asserted by each
runner, so a runner that claims `egress-restricted` while restricting nothing
defeats it. The forge cannot check the claim — the runner protocol has no field
for network policy in either direction — so what it does is make the designation
explicit, checked at the moment identity is handed over, and visible on the
page. The enforcement lives in your runner's own configuration.

That makes the honest claim *"this app is only claimable from runners we
designated as network-restricted"* — never *"the forge enforces an egress
firewall"*. How much the designation is worth depends on who operates the
runners: an organization designating its own fleet is asserting something about
its own machines, which is meaningful; accepting a label from a runner somebody
else operates is trusting their word.

### What the exchange refuses

- **A fork pull request run.** Such a run executes code the repository's
  collaborators never approved. This refusal is not configurable.
- **A repository with no grant** for that app. An app that does not exist is
  refused identically, so names cannot be probed.
- **A suspended app.**
- **A job that is no longer running.**
- **A runner that does not carry the grant's required label**, when one is set.
  A run whose runner cannot be established is refused too, rather than given the
  benefit of the doubt.

### How long the token lasts

One hour, *or* until the job ends — whichever comes first. A cancelled job
takes its token with it rather than leaving a working credential behind with
nothing left to attribute it to.

The permissions come from the grant, fixed at the moment the token was minted.
Narrowing a grant afterwards does not widen a token already issued, and
revoking one does not strand a job that is already running.

## Connect an agent that runs elsewhere

For an agent on a developer's machine or a hosted service, use an access token
instead. On the app, **Connect an agent** shows the host and the token to
configure.

Mint the token from the app's own page so it belongs to the app rather than to
you.

## Assign work to an app

An app can be assigned to an issue like any other member. Assignment emits the
ordinary `issues.assigned` event, so `on: issues: types: [assigned]` is all a
workflow needs — which is what the example above listens for.

Guard on the assignee, as the example above does, or every assignment to
anyone will start the workflow:

```yaml
    if: github.event.assignee.login == 'myagent'
```

`github.event.assignee` carries the person or app that was just assigned or
unassigned, and only on those actions. Reading the issue's own list instead —
`contains(github.event.issue.assignees.*.login, 'myagent')` — answers a
different question: whether the agent is among the assignees at all, which
stays true when somebody else is later assigned to the same issue. Use the
first unless you specifically want the second.

Where the forge records agent work, it appears on the issue and through
`/api/v1/repos/{owner}/{repo}/agent/tasks`.

## What an app may not do to its own work

Two things are refused outright, whoever the app is and however it is run:

- **Merging a pull request it opened.** Someone else must merge it.
- **Taking a pull request it opened out of draft.** Someone else must mark it
  ready for review.

Approving your own pull request was already refused for everyone. These extend
the same idea to the two other moments where work would otherwise reach the
default branch, or reach reviewers, with no second principal involved.

Neither is configurable. A setting to switch one off is the whole guarantee. If
you need an app to merge unattended, give a second app the job — that at least
leaves two principals and an audit trail rather than one account signing off on
itself.

Both rules are about **who opened the pull request**, not about what the app may
do in general. An app with write access merges other people's pull requests
normally, and marks other people's drafts ready normally.

The draft rule is worth reading carefully, because it is narrower than it
sounds. It stops an app **removing** a work-in-progress prefix, not opening a
pull request without one — an app that opens finished work is doing nothing
unusual, and plenty of ordinary automation does exactly that. If you want the
stronger guarantee, have the agent open its work as a draft:

```yaml
      - run: |
          curl -sSf -X POST \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d '{"title": "WIP: fix for #42", "head": "agent/issue-42", "base": "main"}' \
            "${{ github.server_url }}/api/v1/repos/${{ github.repository }}/pulls"
```

From that point the forge holds the promotion for a person. The app can still
push to the branch and rename its own draft; it cannot decide the work is ready.

## Stop an app

Suspending an app stops it immediately, by every credential it holds — API
tokens, git over HTTP, git over SSH, and any token a running job already
obtained. It is a stop button, not a delete: resuming restores access without
reissuing anything.

Suspend from the app, or use the organization-wide switch to stop every app at
once.

Deleting an app removes the account, its tokens and its grants. Suspending is
almost always what you want first: it is reversible, and it takes effect just
as fast.
