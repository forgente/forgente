// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

import "time"

// AgentTask is a unit of work given to an agent. The field names follow
// GitHub's agent task object so a client written against that surface reads
// this one, minus the usage and billing fields Forgente does not have.
type AgentTask struct {
	// ID is the unique identifier for the task
	ID int64 `json:"id"`
	// Name is the human-readable title, usually the issue's
	Name string `json:"name"`
	// State is one of queued, in_progress, completed, failed, idle,
	// waiting_for_user, timed_out, cancelled
	State string `json:"state"`
	// Issue is the issue or pull request the work came from, if any
	Issue *Issue `json:"issue,omitempty"`
	// Agent is the app acting as the principal
	Agent *User `json:"agent,omitempty"`
	// Creator is who asked for the work
	Creator *User `json:"creator,omitempty"`
	// ProfileID names the repository agent profile the task runs under, by its
	// file identifier, or is empty when no profile was named
	ProfileID string `json:"profile_id,omitempty"`
	// swagger:strfmt date-time
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
	// swagger:strfmt date-time
	Created time.Time `json:"created_at"`
	// swagger:strfmt date-time
	Updated time.Time `json:"updated_at"`
}

// AgentSession is one attempt at a task. A task has many over time — a retry,
// or a continuation after a human answered a question — and each keeps its own
// outcome, so an earlier attempt stays readable after a later one ends
// differently.
type AgentSession struct {
	// ID is the unique identifier for the session
	ID int64 `json:"id"`
	// TaskID is the task this is an attempt at
	TaskID int64 `json:"task_id"`
	// State is one of queued, in_progress, completed, failed, idle,
	// waiting_for_user, timed_out, cancelled
	State string `json:"state"`
	// RunID is the Actions run that carried out the attempt, or 0 before one
	// has been attached. Actions is the sandbox, so logs, timing and
	// cancellation live on the run rather than being duplicated here.
	RunID int64 `json:"run_id,omitempty"`
	// Prompt is what the agent was asked
	Prompt string `json:"prompt,omitempty"`
	// HeadRef is the branch the agent works on, BaseRef what it targets
	HeadRef string `json:"head_ref,omitempty"`
	BaseRef string `json:"base_ref,omitempty"`
	// Model records which model served the session, for provenance
	Model string `json:"model,omitempty"`
	// ErrorMessage explains a failed state to a human
	ErrorMessage string `json:"error_message,omitempty"`
	// swagger:strfmt date-time
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	// swagger:strfmt date-time
	Created time.Time `json:"created_at"`
	// swagger:strfmt date-time
	Updated time.Time `json:"updated_at"`
}
