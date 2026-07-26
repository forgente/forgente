// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	agent_model "forgente.com/models/agent"
	issues_model "forgente.com/models/issues"
	user_model "forgente.com/models/user"
	"forgente.com/modules/log"
	agent_service "forgente.com/services/agent"
	"forgente.com/services/context"
)

// issueAgentTaskView pairs a task with the account it runs as, so the sidebar
// can name the agent rather than showing an opaque id.
type issueAgentTaskView struct {
	*agent_model.Task
	Agent *user_model.User
}

// prepareIssueAgentTasks exposes any agent work recorded against the issue.
//
// This is read-only and best-effort: a failure here should not take down the
// issue page, which is why it logs rather than erroring. The tasks are created
// by assignment, so an issue with no agent assignee simply has none.
func prepareIssueAgentTasks(ctx *context.Context, issue *issues_model.Issue) {
	tasks, err := agent_service.TasksForIssue(ctx, issue.ID)
	if err != nil {
		log.Error("agent: load tasks for issue %d: %v", issue.ID, err)
		return
	}
	if len(tasks) == 0 {
		return
	}

	views := make([]*issueAgentTaskView, 0, len(tasks))
	for _, task := range tasks {
		view := &issueAgentTaskView{Task: task}
		if app, err := user_model.GetForgenteAppByID(ctx, task.AppID); err == nil {
			if botUser, err := user_model.GetUserByID(ctx, app.UserID); err == nil {
				view.Agent = botUser
			}
		}
		views = append(views, view)
	}
	ctx.Data["IssueAgentTasks"] = views
}
