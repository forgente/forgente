// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"context"

	agent_model "forgente.com/models/agent"
	"forgente.com/models/db"
	issues_model "forgente.com/models/issues"
	user_model "forgente.com/models/user"
	"forgente.com/modules/log"
	"forgente.com/modules/util"
)

// TaskForIssue returns the task an app is running for an issue, or nil if it
// has none. A task is unique per issue and app: assigning the same app twice
// continues the existing work rather than starting a second copy of it.
func TaskForIssue(ctx context.Context, issueID, appID int64) (*agent_model.Task, error) {
	task := new(agent_model.Task)
	has, err := db.GetEngine(ctx).
		Where("issue_id = ? AND app_id = ?", issueID, appID).
		OrderBy("id DESC").
		Get(task)
	if err != nil || !has {
		return nil, err
	}
	return task, nil
}

// StartTaskForIssue records that an app has been asked to work on an issue.
//
// It does not dispatch anything. Assignment already emits an Actions event, so
// a workflow subscribed to `issues: [assigned]` runs on its own; this gives
// that run an agent-shaped record to be attached to, and is the thing the
// forge knows that Actions does not — which app, on whose behalf, for which
// issue.
func StartTaskForIssue(ctx context.Context, doer *user_model.User, issue *issues_model.Issue, app *user_model.ForgenteApp) (*agent_model.Task, error) {
	existing, err := TaskForIssue(ctx, issue.ID, app.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// re-assignment of an app already working is not a new request; only
		// a finished task is worth reviving
		if !existing.State.IsTerminal() {
			return existing, nil
		}
		if err := setTaskState(ctx, existing, agent_model.StateQueued); err != nil {
			return nil, err
		}
		return existing, nil
	}

	task := &agent_model.Task{
		RepoID:    issue.RepoID,
		OwnerID:   app.OwnerID,
		Name:      issue.Title,
		IssueID:   issue.ID,
		AppID:     app.ID,
		CreatorID: doer.ID,
		State:     agent_model.StateQueued,
	}
	if err := db.Insert(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

// CancelTaskForIssue ends an app's work on an issue, which is what unassigning
// it means. A task that already finished is left alone: cancelling a completed
// task would rewrite history rather than stop anything.
func CancelTaskForIssue(ctx context.Context, issueID, appID int64) error {
	task, err := TaskForIssue(ctx, issueID, appID)
	if err != nil || task == nil {
		return err
	}
	if task.State.IsTerminal() {
		return nil
	}
	return setTaskState(ctx, task, agent_model.StateCancelled)
}

func setTaskState(ctx context.Context, task *agent_model.Task, state agent_model.State) error {
	task.State = state
	_, err := db.GetEngine(ctx).ID(task.ID).Cols("state", "updated_unix").Update(task)
	return err
}

// appForAssignee resolves an assignee to the app it acts as, or nil when the
// assignee is an ordinary person. A suspended app is treated as no app at all:
// the kill switch should stop new work being recorded, not only new tokens.
func appForAssignee(ctx context.Context, assignee *user_model.User) *user_model.ForgenteApp {
	if assignee == nil || !assignee.IsTypeBot() {
		return nil
	}
	app, err := user_model.GetForgenteAppByUserID(ctx, assignee.ID)
	if err != nil {
		if !user_model.IsErrForgenteAppNotExist(err) {
			log.Error("agent dispatch: resolve app for user %d: %v", assignee.ID, err)
		}
		// a bot account that is not an app is someone else's automation
		return nil
	}
	if app.Suspended {
		return nil
	}
	return app
}

// CompleteTask marks a task finished.
//
// It deliberately does not archive. An earlier draft did, on the reasoning that
// finished work is done with — but archiving hides a task from default
// listings, so completing one would make it vanish at the moment somebody wants
// to look at what the agent did. Archiving stays a separate, deliberate action.
func CompleteTask(ctx context.Context, task *agent_model.Task, state agent_model.State) error {
	if !state.IsTerminal() {
		return util.NewInvalidArgumentErrorf("cannot complete a task into non-terminal state %q", state)
	}
	if task.State.IsTerminal() {
		// a run that reports twice, or a task cancelled by unassignment while
		// its run was still going, must not be walked back
		return nil
	}
	return setTaskState(ctx, task, state)
}
