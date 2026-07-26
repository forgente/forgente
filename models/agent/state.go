// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Package agent records agent work: the task an agent is given and the
// sessions that attempt it. The object model is GitHub's — task and session as
// separate objects, one to many, sharing a state vocabulary — because that
// surface is a proven design and reimplementing it differently would cost
// interoperability for no gain. What is deliberately absent is the usage and
// billing half, which the program document excludes wholesale.
package agent

import "slices"

// State is the lifecycle of a task or a session. Both use the same vocabulary,
// which is GitHub's, taken verbatim so the values mean the same thing here.
type State string

const (
	StateQueued         State = "queued"
	StateInProgress     State = "in_progress"
	StateCompleted      State = "completed"
	StateFailed         State = "failed"
	StateIdle           State = "idle"
	StateWaitingForUser State = "waiting_for_user"
	StateTimedOut       State = "timed_out"
	StateCancelled      State = "cancelled"
)

// AllStates is every valid value, in the order the upstream enumeration lists
// them.
var AllStates = []State{
	StateQueued,
	StateInProgress,
	StateCompleted,
	StateFailed,
	StateIdle,
	StateWaitingForUser,
	StateTimedOut,
	StateCancelled,
}

// IsValid reports whether s is one of the eight known states. Anything else
// came from a client or a database row that should not be trusted.
func (s State) IsValid() bool {
	return slices.Contains(AllStates, s)
}

// IsTerminal reports whether no further work will happen.
//
// Only the four unambiguously final states count. `idle` and
// `waiting_for_user` are pauses rather than endings — a session waiting for a
// human approval is expected to continue once it gets one — so treating them
// as terminal would garbage-collect work that is merely blocked. That reading
// is ours; the upstream enumeration documents the values but not their
// transitions.
func (s State) IsTerminal() bool {
	switch s {
	case StateCompleted, StateFailed, StateTimedOut, StateCancelled:
		return true
	}
	return false
}

// IsActive reports whether the agent is currently working.
func (s State) IsActive() bool {
	return s == StateQueued || s == StateInProgress
}
