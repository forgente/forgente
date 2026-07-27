// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"context"

	agent_model "forgente.com/models/agent"
	"forgente.com/models/db"
	"forgente.com/modules/timeutil"
	"forgente.com/modules/util"
)

// SessionsForTask returns every attempt at a task, newest first.
func SessionsForTask(ctx context.Context, taskID int64) ([]*agent_model.Session, error) {
	sessions := make([]*agent_model.Session, 0, 1)
	err := db.GetEngine(ctx).Where("task_id = ?", taskID).OrderBy("id DESC").Find(&sessions)
	return sessions, err
}

// LatestSessionForTask returns the most recent attempt, or nil when a task has
// none yet.
func LatestSessionForTask(ctx context.Context, taskID int64) (*agent_model.Session, error) {
	session := new(agent_model.Session)
	has, err := db.GetEngine(ctx).Where("task_id = ?", taskID).OrderBy("id DESC").Get(session)
	if err != nil || !has {
		return nil, err
	}
	return session, nil
}

// ActiveSessionForTask returns the attempt currently under way, or nil. Only
// one should ever be active, so the newest wins if that is somehow untrue.
func ActiveSessionForTask(ctx context.Context, taskID int64) (*agent_model.Session, error) {
	session := new(agent_model.Session)
	has, err := db.GetEngine(ctx).
		Where("task_id = ?", taskID).
		In("state", agent_model.StateQueued, agent_model.StateInProgress).
		OrderBy("id DESC").
		Get(session)
	if err != nil || !has {
		return nil, err
	}
	return session, nil
}

// StartSession records a new attempt at a task.
//
// A retry is a new session rather than a change to an old one, which is what
// keeps the outcome of every earlier attempt readable afterwards.
func StartSession(ctx context.Context, task *agent_model.Task, prompt string) (*agent_model.Session, error) {
	session := &agent_model.Session{
		TaskID: task.ID,
		// denormalised from the task so a session can be authorised and listed
		// without loading the task first
		RepoID:  task.RepoID,
		OwnerID: task.OwnerID,
		State:   agent_model.StateQueued,
		Prompt:  prompt,
	}
	if err := db.WithTx(ctx, func(ctx context.Context) error {
		if err := db.Insert(ctx, session); err != nil {
			return err
		}
		return syncTaskState(ctx, task.ID)
	}); err != nil {
		return nil, err
	}
	return session, nil
}

// SetSessionState moves an attempt to a new state and brings its task's state
// with it.
//
// This is the only writer of agent_task.state. The defect that made sessions
// worth building was not that the task's state was stored — it was that two
// paths wrote it and neither owned it, each correct on its own, so a retry
// erased the outcome of the attempt before it. Funnelling every write through
// here, in one transaction, makes a task that contradicts its own sessions
// unrepresentable rather than merely unlikely.
func SetSessionState(ctx context.Context, session *agent_model.Session, state agent_model.State) error {
	if !state.IsValid() {
		return util.NewInvalidArgumentErrorf("unknown agent state %q", state)
	}
	if session.State.IsTerminal() {
		// a run reporting twice, or a session cancelled by unassignment while
		// its run was still going, must not be walked back
		return nil
	}

	return db.WithTx(ctx, func(ctx context.Context) error {
		session.State = state
		cols := []string{"state", "updated_unix"}
		if state.IsTerminal() && session.CompletedAt.IsZero() {
			session.CompletedAt = timeutil.TimeStampNow()
			cols = append(cols, "completed_at")
		}
		if _, err := db.GetEngine(ctx).ID(session.ID).Cols(cols...).Update(session); err != nil {
			return err
		}
		return syncTaskState(ctx, session.TaskID)
	})
}

// AttachRunToSession records which Actions run carried out an attempt.
//
// Actions is the sandbox, so the run is where logs, timing and cancellation
// actually live; the session points at it rather than duplicating it.
func AttachRunToSession(ctx context.Context, session *agent_model.Session, runID int64) error {
	if session.RunID == runID {
		return nil
	}
	session.RunID = runID
	_, err := db.GetEngine(ctx).ID(session.ID).Cols("run_id", "updated_unix").Update(session)
	return err
}

// deriveTaskState computes a task's state from its attempts.
//
// An attempt still under way decides the answer whatever earlier ones ended
// as, because the task is being worked on right now. Otherwise the most recent
// attempt speaks for the task: reporting the last failure rather than an
// earlier success is the honest reading, and the success is still readable on
// its own session.
func deriveTaskState(ctx context.Context, taskID int64) (agent_model.State, error) {
	sessions := make([]*agent_model.Session, 0, 2)
	if err := db.GetEngine(ctx).
		Where("task_id = ?", taskID).
		Cols("id", "state").
		OrderBy("id DESC").
		Find(&sessions); err != nil {
		return "", err
	}
	if len(sessions) == 0 {
		// a task exists before its first attempt is recorded
		return agent_model.StateQueued, nil
	}

	queued := false
	for _, session := range sessions {
		switch session.State {
		case agent_model.StateInProgress:
			return agent_model.StateInProgress, nil
		case agent_model.StateQueued:
			queued = true
		}
	}
	if queued {
		return agent_model.StateQueued, nil
	}
	return sessions[0].State, nil
}

// syncTaskState writes the derived state onto the task. Callers must already
// be inside the transaction that changed the sessions it is derived from.
func syncTaskState(ctx context.Context, taskID int64) error {
	state, err := deriveTaskState(ctx, taskID)
	if err != nil {
		return err
	}
	_, err = db.GetEngine(ctx).ID(taskID).
		Cols("state", "updated_unix").
		Update(&agent_model.Task{State: state})
	return err
}
