// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"context"

	actions_model "forgente.com/models/actions"
	agent_model "forgente.com/models/agent"
	issues_model "forgente.com/models/issues"
	repo_model "forgente.com/models/repo"
	user_model "forgente.com/models/user"
	"forgente.com/modules/json"
	"forgente.com/modules/log"
	api "forgente.com/modules/structs"
	webhook_module "forgente.com/modules/webhook"
)

// WorkflowRunStatusUpdate moves an agent task to its final state when the run
// that assignment started finishes.
//
// The forge dispatched the run and therefore already knows how it ended; the
// task record is derived from that rather than reported by the agent. That
// matters more than it sounds: an agent written against another forge reports
// progress by writing comments, because that is all it was ever offered, so a
// record that filled in only when an agent posted to it would stay empty for
// exactly the agents worth attracting.
func (n *agentNotifier) WorkflowRunStatusUpdate(ctx context.Context, repo *repo_model.Repository, _ *user_model.User, run *actions_model.ActionRun) {
	if run == nil {
		return
	}
	// A run that has merely been queued or blocked has not started work, and a
	// cancelling run is on its way to a terminal status that will arrive here
	// anyway. Only starting and finishing are worth recording.
	if !run.Status.IsRunning() && !run.Status.IsDone() {
		return
	}
	// Only a run an issue assignment started can belong to a task. Note this
	// is issue_assign, not issues: the run carries the specific hook event,
	// while the tasks API reports the coarser "issues" — reading that listing
	// rather than the model is what made the first version of this silently
	// match nothing.
	if run.Event != webhook_module.HookEventIssueAssign {
		return
	}

	var payload api.IssuePayload
	if err := json.Unmarshal([]byte(run.EventPayload), &payload); err != nil {
		log.Error("agent: parse run %d payload: %v", run.ID, err)
		return
	}
	// the assignee is what identifies which app the run was for, and it only
	// arrives on the assign action
	if payload.Action != api.HookIssueAssigned || payload.Assignee == nil {
		return
	}

	assignee, err := user_model.GetUserByName(ctx, payload.Assignee.UserName)
	if err != nil {
		log.Error("agent: resolve assignee %q for run %d: %v", payload.Assignee.UserName, run.ID, err)
		return
	}
	app := appForAssignee(ctx, assignee)
	if app == nil {
		return
	}

	issue, err := issues_model.GetIssueByIndex(ctx, repo.ID, payload.Index)
	if err != nil {
		log.Error("agent: resolve issue %d of repo %d for run %d: %v", payload.Index, repo.ID, run.ID, err)
		return
	}

	task, err := TaskForIssue(ctx, issue.ID, app.ID)
	if err != nil {
		log.Error("agent: find task for issue %d app %d: %v", issue.ID, app.ID, err)
		return
	}
	if task == nil {
		return
	}

	if run.Status.IsRunning() {
		if err := BeginAttempt(ctx, task, run.ID); err != nil {
			log.Error("agent: begin attempt on task %d from run %d: %v", task.ID, run.ID, err)
		}
		return
	}

	if err := CompleteAttempt(ctx, task, stateForRunStatus(run.Status), run.ID); err != nil {
		log.Error("agent: complete attempt on task %d from run %d: %v", task.ID, run.ID, err)
	}
}

// stateForRunStatus maps how a run ended onto the task vocabulary. Anything that
// is neither success nor cancellation is a failure: a task that stopped without
// finishing its work has failed, whatever the runner called it.
func stateForRunStatus(status actions_model.Status) agent_model.State {
	switch {
	case status.IsSuccess():
		return agent_model.StateCompleted
	case status.IsCancelled():
		return agent_model.StateCancelled
	default:
		return agent_model.StateFailed
	}
}
