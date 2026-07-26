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
