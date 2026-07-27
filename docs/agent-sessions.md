# Agent sessions: design

**Status: proposed, not built.** This is the remaining slice of L3 in
[agent-native-program.md](agent-native-program.md).

It is not a new feature. The task and session pair was designed together and
half of it shipped; this document is about finishing the half that did not, and
about the cost of leaving it unfinished, which is larger than "a missing
feature" suggests.

## The shipped half is lossy without the other

`models/agent/session.go` defines both objects and registers both tables.
`Task` is written, read, surfaced on the issue, and moved to a final state when
its run ends (#82, #84, #85, #86, #93). `Session` is a live table that **no
code reads or writes** — it has no references anywhere outside its own
definition.

The model states the intended division of labour:

> Task is a unit of work given to an agent … It outlives any single attempt at
> it: a task that fails and is retried keeps its identity, and the attempts are
> Sessions.

The identity is kept. The attempts are lost. With nowhere to put an attempt,
`StartTaskForIssue` revives the existing row back to `queued`
(`services/agent/dispatch.go:50`), so a second attempt overwrites the outcome
of the first.

Observed on a live instance on 2026-07-27. One issue, one app, seven runs. The
first run succeeded: it opened a draft pull request, was correctly refused
self-ready and self-merge, and commented as the app. Later runs were
deliberately broken to test a refusal. Afterwards:

```
agent_task    -> 1 row,  state = failed
agent_session -> 0 rows
```

The record reports `failed` for a task whose real work succeeded, and there is
no way to recover that the successful attempt ever happened. The API reports
the same thing, because it reads the same field.

**No individual function is wrong.** `CompleteTask` correctly refuses to walk
back a terminal state. `CancelTaskForIssue` correctly leaves finished work
alone. The revive is deliberate and commented. Each behaves exactly as
documented; the composition loses history, because every one of them is written
against a Session layer that is not there.

That reframes this slice. It is not "add sessions so users can see attempts" —
it is "finish the pair, because the shipped half reports the wrong outcome on
the second attempt."

## What a session has to do

The minimum, in order of how much each is load-bearing:

1. **Be the thing an attempt is recorded on.** One row per dispatch. A retry
   makes a new session and never mutates an old one. This alone fixes the
   lossiness above.
2. **Own the run linkage.** `Session.RunID` already exists for this. Today
   `WorkflowRunStatusUpdate` resolves a run straight to a task
   (`services/agent/run_linkage_forgente.go:69`) and calls `CompleteTask`; it
   should resolve to the *session* the run belongs to, and the task's state
   should be derived from its sessions rather than written directly.
3. **Carry what the task cannot.** Prompt, head and base ref, model label,
   error message, completion time — all already fields on the struct, all
   per-attempt rather than per-task.

Everything past that — a session view, cross-session filtering, steer-or-stop —
is `AN-RUN-6` and can follow. None of it is needed to stop losing history.

## The seam

The decision that matters is **where a task's state comes from once sessions
exist.**

Two candidates:

- **Derived.** A task's state is a function of its sessions: active if any
  session is active, otherwise the outcome of the most recent one. No task
  state is ever written directly.
- **Stored, updated by the session.** The task keeps its `State` column and a
  session's terminal transition writes it.

**Recommend derived**, for one reason: stored state is what produced the bug
above. A column that two code paths write and neither owns is exactly the shape
that lost the successful run. Deriving it makes the lossy case unrepresentable
rather than merely fixed.

The cost is real and should be stated: derivation means a listing cannot filter
on `agent_task.state` with a plain `WHERE`, and the task table already indexes
that column. Either the listing joins, or the column stays as a maintained
cache with the sessions as the source of truth. The second is a middle path and
is probably where this lands, but it should be chosen deliberately, because a
cache that drifts is the original bug wearing a different hat.

## What this does not decide

**It does not build the write half of the API.** `AN-RUN-5` counts five
endpoints; two exist (list, get). Sessions being writable by an external agent
is a separate question with its own security surface — an agent reporting its
own progress is an agent that can lie about it — and it should not ride along
with fixing the record.

**It does not decide how progress gets reported.** The current design derives
everything the forge already knows from the run it dispatched, deliberately:
an agent written against another forge reports progress by editing comments,
because that is all it was ever offered, so a record that filled in only when
an agent posted to it would stay empty for exactly the agents worth attracting.
That reasoning is unchanged by anything here.

**It does not touch the eight states.** They are GitHub's, taken verbatim, and
the `IsTerminal` reading — `idle` and `waiting_for_user` are pauses, not
endings — is already recorded in `models/agent/state.go` as ours rather than
theirs.

## Open question

**Should a re-assignment start a new session, or is unassign-then-reassign the
only way to retry?** Today re-assigning an app whose task is finished revives
it. With sessions, that becomes "new session on the same task", which is
probably right — but it means the assignment event is the retry trigger, and an
accidental double-assign creates an attempt. The alternative is an explicit
retry action, which is more deliberate and less discoverable.

I lean towards re-assignment creating a session, because it keeps one trigger
rather than two and matches what the current code already tries to express.

## Prior finding worth not repeating

The substrate check has now paid five times — assignment-as-trigger,
self-approval refusal, runner label routing, permissions, and here. This
instance inverted it: the check usually finds that something is *already built*
and shrinks the work. This time it found a table that exists, is registered,
and is entirely dead, which is a different failure mode and a more expensive
one. **Check both directions**: whether the thing exists, and whether anything
actually uses it.
