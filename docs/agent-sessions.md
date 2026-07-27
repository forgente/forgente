# Agent sessions: design

**Status: built in #102.** This is the design it was built to, kept because the
reasoning outlives the change. What shipped is the record itself; sessions are
not yet exposed over the API, which is the next slice.

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

First, a correction to how this was originally framed here. Derived-versus-
stored does not decide whether history survives. Once sessions exist at all,
each attempt's outcome lives on its own row and is safe either way; what loses
it today is the *revive*, and giving an attempt somewhere to go removes the
revive on its own. The real question is narrower: **what does `task.state`
mean, and who is allowed to write it.**

### Decided

`agent_task.state` is a **derived value, materialised into the existing column
by exactly one writer, in the same transaction that changes a session's state.**
No other code path writes it.

Derivation, in order:

1. Any session active → the task is active (`in_progress` if any session is,
   otherwise `queued`).
2. Otherwise → the state of the **most recent** session.
3. No sessions yet → `queued`.

And the change that actually fixes the loss: **`StartTaskForIssue` stops
reviving.** Re-assignment creates a new session and the task's state follows
from it. The task row becomes immutable after creation apart from `ArchivedAt`
and the derived column.

### Why this shape

**The column stays because the listing needs it.** `ListTasks`
(`services/agent/list.go:31`) filters state with a plain `WHERE` and pages with
`FindAndCount`. Computing state per row turns that into a join plus an
aggregate, and pagination over an aggregate, which is a real cost imposed on a
listing that works today.

**Single writer in one transaction is the load-bearing part, not the caching.**
The original loss was not caused by storing state. It was caused by two paths
writing one column with neither owning it — `CompleteTask` writing outcomes and
`StartTaskForIssue` writing revivals, each correct in isolation. Funnelling
every write through the function that changes a session makes a task state that
contradicts its sessions unrepresentable, which is the same choke-point
discipline `CheckPullMergeable` and `MintAppRunToken` already use.

**Most recent wins, not sticky-completed.** If attempt one succeeded and
attempt two failed, the task reads `failed`. That is honest: the last attempt
did fail, whatever attempt one produced still exists, and session one still
records `completed`. A best-outcome rule would hide real failures, which is a
worse fault than the one it fixes.

**No schema change.** Both tables exist and `Session.RunID` is already there for
the run linkage. This is entirely code.

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

## Retry trigger

**Re-assignment creates a session.** The alternative was an explicit retry
action, which is more deliberate and less discoverable; this keeps one trigger
rather than two, and it is what the current code is already reaching for when
it revives the row.

The cost accepted: an accidental double-assign creates an attempt. That is
visible in the session list rather than silent, which is the property that
matters.

## Prior finding worth not repeating

The substrate check has now paid five times — assignment-as-trigger,
self-approval refusal, runner label routing, permissions, and here. This
instance inverted it: the check usually finds that something is *already built*
and shrinks the work. This time it found a table that exists, is registered,
and is entirely dead, which is a different failure mode and a more expensive
one. **Check both directions**: whether the thing exists, and whether anything
actually uses it.
