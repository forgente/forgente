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
    runs-on: ubuntu-latest
    steps:
      - name: Get a token for the app
        id: app-token
        run: |
          token=$(curl -sSf -X POST \
            -H "Authorization: Bearer ${{ secrets.GITEA_TOKEN }}" \
            -H "Content-Type: application/json" \
            -d '{"app": "myagent"}' \
            "${{ github.server_url }}/api/v1/repos/${{ github.repository }}/actions/app-token" \
            | jq -r .token)
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

Mask the token as soon as you have it. It is short-lived, but a log is
permanent.

The same token works for git over HTTP, so the app can commit as itself:

```yaml
      - run: |
          git clone "https://x-access-token:${{ steps.app-token.outputs.token }}@example.com/${{ github.repository }}.git" work
```

### What the exchange refuses

- **A fork pull request run.** Such a run executes code the repository's
  collaborators never approved. This refusal is not configurable.
- **A repository with no grant** for that app. An app that does not exist is
  refused identically, so names cannot be probed.
- **A suspended app.**
- **A job that is no longer running.**

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

Guard on the assignee, or every assignment will start the workflow:

```yaml
    if: github.event.assignee.login == 'myagent'
```

Where the forge records agent work, it appears on the issue and through
`/api/v1/repos/{owner}/{repo}/agent/tasks`.

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
