// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"fmt"
	"net/http"

	agent_model "forgente.com/models/agent"
	issues_model "forgente.com/models/issues"
	user_model "forgente.com/models/user"
	api "forgente.com/modules/structs"
	"forgente.com/routers/api/v1/utils"
	agent_service "forgente.com/services/agent"
	"forgente.com/services/context"
	"forgente.com/services/convert"
)

// toAPIAgentTask expands a task for the API. The issue, agent and creator are
// looked up individually rather than joined: a listing is small, and a task
// whose issue or creator has been deleted should still appear rather than
// disappear from the list with it.
func toAPIAgentTask(ctx *context.APIContext, task *agent_model.Task) *api.AgentTask {
	out := &api.AgentTask{
		ID:        task.ID,
		Name:      task.Name,
		State:     string(task.State),
		ProfileID: task.ProfileID,
		Created:   task.CreatedUnix.AsTime(),
		Updated:   task.UpdatedUnix.AsTime(),
	}
	if !task.ArchivedAt.IsZero() {
		archived := task.ArchivedAt.AsTime()
		out.ArchivedAt = &archived
	}

	if task.IssueID != 0 {
		if issue, err := issues_model.GetIssueByID(ctx, task.IssueID); err == nil {
			if err := issue.LoadRepo(ctx); err == nil {
				out.Issue = convert.ToAPIIssue(ctx, ctx.Doer, issue)
			}
		}
	}
	if app, err := user_model.GetForgenteAppByID(ctx, task.AppID); err == nil {
		if botUser, err := user_model.GetUserByID(ctx, app.UserID); err == nil {
			out.Agent = convert.ToUser(ctx, botUser, ctx.Doer)
		}
	}
	if creator, err := user_model.GetUserByID(ctx, task.CreatorID); err == nil {
		out.Creator = convert.ToUser(ctx, creator, ctx.Doer)
	}
	return out
}

// ListAgentTasks lists the agent tasks recorded for a repository.
func ListAgentTasks(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/agent/tasks repository repoListAgentTasks
	// ---
	// summary: List a repository's agent tasks
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: state
	//   in: query
	//   description: filter by state, one of queued, in_progress, completed, failed, idle, waiting_for_user, timed_out, cancelled
	//   type: string
	// - name: include_archived
	//   in: query
	//   description: include tasks archived on completion
	//   type: boolean
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/AgentTaskList"
	//   "422":
	//     "$ref": "#/responses/validationError"

	state := agent_model.State(ctx.FormString("state"))
	if state != "" && !state.IsValid() {
		ctx.APIError(http.StatusUnprocessableEntity, fmt.Sprintf("unknown state %q", state))
		return
	}

	tasks, total, err := agent_service.ListTasks(ctx, agent_service.ListTasksOptions{
		ListOptions:     utils.GetListOptions(ctx),
		RepoID:          ctx.Repo.Repository.ID,
		State:           state,
		IncludeArchived: ctx.FormBool("include_archived"),
	})
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}

	out := make([]*api.AgentTask, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, toAPIAgentTask(ctx, task))
	}

	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, out)
}

// GetAgentTask returns one agent task.
func GetAgentTask(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/agent/tasks/{id} repository repoGetAgentTask
	// ---
	// summary: Get an agent task
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the task
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/AgentTask"
	//   "404":
	//     "$ref": "#/responses/notFound"

	task, err := agent_service.GetTaskByID(ctx, ctx.PathParamInt64("id"))
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	// a task belonging to another repository is reported missing rather than
	// forbidden, so an id cannot be probed across repositories
	if task == nil || task.RepoID != ctx.Repo.Repository.ID {
		ctx.APIErrorNotFound()
		return
	}

	ctx.JSON(http.StatusOK, toAPIAgentTask(ctx, task))
}
