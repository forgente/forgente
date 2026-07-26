// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateIsValid(t *testing.T) {
	// the eight values of the upstream enumeration, spelled out rather than
	// ranged over, so a typo in AllStates cannot make this pass
	for _, s := range []State{
		"queued", "in_progress", "completed", "failed",
		"idle", "waiting_for_user", "timed_out", "cancelled",
	} {
		assert.True(t, s.IsValid(), s)
	}
	assert.Len(t, AllStates, 8)

	for _, s := range []State{"", "running", "success", "Queued", "in-progress"} {
		assert.False(t, s.IsValid(), s)
	}
}

func TestStateIsTerminal(t *testing.T) {
	for _, s := range []State{StateCompleted, StateFailed, StateTimedOut, StateCancelled} {
		assert.True(t, s.IsTerminal(), s)
	}

	// a pause is not an ending: a session waiting on a human is expected to
	// continue, and treating it as finished would discard blocked work
	for _, s := range []State{StateQueued, StateInProgress, StateIdle, StateWaitingForUser} {
		assert.False(t, s.IsTerminal(), s)
	}
}

func TestStateIsActive(t *testing.T) {
	assert.True(t, StateQueued.IsActive())
	assert.True(t, StateInProgress.IsActive())

	for _, s := range []State{StateIdle, StateWaitingForUser, StateCompleted, StateCancelled} {
		assert.False(t, s.IsActive(), s)
	}
}

func TestStateCategoriesArePartitioned(t *testing.T) {
	// every state is active, terminal, or a pause — never two at once
	for _, s := range AllStates {
		assert.False(t, s.IsActive() && s.IsTerminal(), s)
	}
}
