// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"forgente.com/models/db"
	"forgente.com/modules/timeutil"
)

// Task is a unit of work given to an agent — "address this issue", "review
// this pull request". It outlives any single attempt at it: a task that fails
// and is retried keeps its identity, and the attempts are Sessions.
type Task struct {
	ID int64 `xorm:"pk autoincr"`
	// RepoID is the repository the work belongs to. OwnerID is denormalised
	// from it so an organization-wide listing does not need a join.
	RepoID  int64 `xorm:"INDEX NOT NULL"`
	OwnerID int64 `xorm:"INDEX NOT NULL"`
	// Name is the human-readable title, usually taken from the issue or pull
	// request that started the task.
	Name string
	// IssueID is the issue or pull request the work came from. Assignment is
	// the trigger — an app is a bot account, CanBeAssigned does not exclude
	// bots, and assigning one already fires an Actions event — so a task
	// normally has one. Zero means it was created directly rather than by
	// assignment, which the API path will allow.
	IssueID int64 `xorm:"INDEX"`
	// AppID is the organization-owned app acting as the principal, the L0
	// primitive. A task always runs as an app rather than as a person, which
	// is what makes its actions attributable.
	AppID int64 `xorm:"INDEX NOT NULL"`
	// ProfileID is the repository agent profile this task runs under, by its
	// file identifier — the L2 discovery key, not a foreign key, because the
	// profile lives in the repository and can change or disappear between
	// runs. Empty means no profile was named.
	ProfileID string
	// CreatorID is who asked for the work. It is kept if that account is
	// deleted: the task's history should not silently lose its origin.
	CreatorID int64 `xorm:"NOT NULL"`
	State     State `xorm:"INDEX NOT NULL"`
	// ArchivedAt hides a finished task from default listings without deleting
	// it. Zero means not archived.
	ArchivedAt  timeutil.TimeStamp
	CreatedUnix timeutil.TimeStamp `xorm:"created INDEX"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

// Session is one attempt at a Task: a dispatched Actions run with a record in
// front of it. A task has many sessions over time — a retry, or a continuation
// after a human answered a question — and each carries its own state.
type Session struct {
	ID     int64 `xorm:"pk autoincr"`
	TaskID int64 `xorm:"INDEX NOT NULL"`
	// RepoID and OwnerID are denormalised from the task so a session can be
	// authorised and listed without loading it.
	RepoID  int64 `xorm:"INDEX NOT NULL"`
	OwnerID int64 `xorm:"INDEX NOT NULL"`
	State   State `xorm:"INDEX NOT NULL"`
	// RunID is the Actions run executing this session. Actions is the sandbox,
	// so the run is where logs, timing and cancellation actually live; this
	// record exists to give that run an agent-shaped identity rather than to
	// duplicate it. Zero until the run is dispatched.
	RunID int64 `xorm:"INDEX"`
	// Prompt is what the agent was asked, recorded so a session can be
	// understood after the fact without reconstructing it from the issue.
	Prompt string `xorm:"TEXT"`
	// HeadRef is the branch the agent works on, BaseRef what it targets.
	HeadRef string
	BaseRef string
	// Model records which model served the session, for provenance. Forgente
	// does not host models, so this is a label rather than a configuration.
	Model string
	// ErrorMessage explains a failed state to a human. Empty otherwise.
	ErrorMessage string `xorm:"TEXT"`
	CompletedAt  timeutil.TimeStamp
	CreatedUnix  timeutil.TimeStamp `xorm:"created INDEX"`
	UpdatedUnix  timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	// new tables, created by SyncAllTables at startup. A numbered migration
	// would take a version slot upstream also uses, which would conflict on
	// every future cherry-pick; #66 established this pattern for the same
	// reason.
	db.RegisterModel(new(Task))
	db.RegisterModel(new(Session))
}

// TableName keeps the table names explicit and namespaced, since "task" and
// "session" are far too generic to claim on their own.
func (Task) TableName() string { return "agent_task" }

// TableName is namespaced for the same reason as Task's.
func (Session) TableName() string { return "agent_session" }

// IsArchived reports whether the task has been archived.
func (t *Task) IsArchived() bool { return !t.ArchivedAt.IsZero() }
