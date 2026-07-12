---
name: babysit-ci
description: Watch Forgente CI after merging PRs to main — confirm the nightly release build survives, cancel forever-queued release-branch runs, report failures. Use after a batch merge, typically via /loop.
---

# Babysitting CI after merges

Two repo-specific behaviors make post-merge CI need supervision:

1. **Merges to main cancel in-flight nightlies** (workflow concurrency). The
   `release-nightly` binary build takes ~1h; every merge restarts it. So batch
   merges first, then watch the LAST nightly run through to completion.
2. **Release-branch pushes run upstream's workflow file** (the file at that
   ref), which targets Gitea's private Namespace runners — those runs queue
   forever and must be cancelled manually.

## Loop body

Each iteration (suggested `/loop` interval: 10–15m while a nightly is running):

1. `gh run list --branch main --workflow release-nightly --limit 3` — is the
   newest run still in progress, completed, or was it cancelled by a newer
   merge? If cancelled and no newer run exists, re-trigger is NOT possible
   (push-triggered); report it so a follow-up merge or empty commit can be
   decided by the user.
2. `gh run list --status queued` — cancel anything queued >15m on a
   `release/v*` branch (`gh run cancel <id>`). Leave main's queued runs alone.
3. `gh run list --branch main --status failure --limit 5` — for real failures,
   fetch the failing job log (`gh run view <id> --log-failed`) and report the
   root cause; do not auto-rerun more than once (flaky-test retry only).

## Stop condition

Stop when the newest main nightly run has concluded `success` and nothing is
stuck in queue. On success, spot-check the artifacts: main builds are tagged
`main-nightly` (the workflow uses the cleaned branch name, NOT plain
`nightly`), so check that tag was pushed within the last hours
(`curl -s https://hub.docker.com/v2/repositories/forgente/forgente/tags/main-nightly | jq -r '.last_updated'`)
— validate real external state, not just the green check.
