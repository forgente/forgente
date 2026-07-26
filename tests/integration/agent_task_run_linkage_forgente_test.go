// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/url"
	"testing"

	actions_model "forgente.com/models/actions"
	agent_model "forgente.com/models/agent"
	"forgente.com/models/db"
	issues_model "forgente.com/models/issues"
	"forgente.com/models/organization"
	repo_model "forgente.com/models/repo"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"
	"forgente.com/modules/json"
	api "forgente.com/modules/structs"
	webhook_module "forgente.com/modules/webhook"
	agent_service "forgente.com/services/agent"
	issue_service "forgente.com/services/issue"
	org_service "forgente.com/services/org"
	user_service "forgente.com/services/user"
	"forgente.com/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentTaskFollowsItsRun(t *testing.T) {
	onGiteaRun(t, testAgentTaskFollowsItsRun)
}

func testAgentTaskFollowsItsRun(t *testing.T, _ *url.URL) {
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org3"})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: owner.ID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 1})
	require.NoError(t, issue.LoadRepo(t.Context()))

	botUser, app, err := user_service.CreateOrgApp(t.Context(), owner, org, "runlinkbot", "finishes things")
	require.NoError(t, err)
	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{OrgID: org.ID, LowerName: "owners"})
	require.NoError(t, org_service.AddTeamMember(t.Context(), team, botUser))

	_, _, err = issue_service.ToggleAssigneeWithNotify(t.Context(), issue, owner, botUser.ID)
	require.NoError(t, err)

	task, err := agent_service.TaskForIssue(t.Context(), issue.ID, app.ID)
	require.NoError(t, err)
	require.NotNil(t, task)
	require.Equal(t, agent_model.StateQueued, task.State, "assignment records the work as queued")

	// a run shaped like the one assignment starts: the assignee is what says
	// which app it belongs to, so a run without one must be ignored
	runIndex := int64(700)
	newRun := func(t *testing.T, status actions_model.Status, assignee *api.User) *actions_model.ActionRun {
		t.Helper()
		runIndex++
		payload, err := json.Marshal(&api.IssuePayload{
			Action:   api.HookIssueAssigned,
			Index:    issue.Index,
			Assignee: assignee,
		})
		require.NoError(t, err)

		run := &actions_model.ActionRun{
			Title:         "agent",
			RepoID:        repo.ID,
			OwnerID:       repo.OwnerID,
			WorkflowID:    "agent.yaml",
			Index:         runIndex,
			TriggerUserID: owner.ID,
			Ref:           "refs/heads/master",
			Event:         webhook_module.HookEventIssueAssign,
			EventPayload:  string(payload),
			Status:        status,
		}
		require.NoError(t, db.Insert(t.Context(), run))
		run.Repo = repo
		return run
	}
	notifier := agent_service.NewNotifier()

	reload := func(t *testing.T) *agent_model.Task {
		t.Helper()
		got, err := agent_service.TaskForIssue(t.Context(), issue.ID, app.ID)
		require.NoError(t, err)
		return got
	}

	t.Run("ARunStillGoingLeavesTheTaskAlone", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		run := newRun(t, actions_model.StatusRunning, &api.User{UserName: botUser.Name})
		notifier.WorkflowRunStatusUpdate(t.Context(), repo, owner, run)
		assert.Equal(t, agent_model.StateQueued, reload(t).State)
	})

	t.Run("AnUnrelatedEventIsIgnored", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// the run carries the specific hook event, issue_assign; the tasks API
		// reports the coarser "issues", and believing that listing is what made
		// the first version of this match nothing at all
		run := newRun(t, actions_model.StatusSuccess, &api.User{UserName: botUser.Name})
		run.Event = webhook_module.HookEventIssues
		notifier.WorkflowRunStatusUpdate(t.Context(), repo, owner, run)
		assert.Equal(t, agent_model.StateQueued, reload(t).State)
	})

	t.Run("ARunWithoutAnAssigneeIsNotOurs", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// before the assignee reached the payload this was every run, which is
		// why the task could never advance
		run := newRun(t, actions_model.StatusSuccess, nil)
		notifier.WorkflowRunStatusUpdate(t.Context(), repo, owner, run)
		assert.Equal(t, agent_model.StateQueued, reload(t).State)
	})

	t.Run("SuccessCompletesTheTask", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		run := newRun(t, actions_model.StatusSuccess, &api.User{UserName: botUser.Name})
		notifier.WorkflowRunStatusUpdate(t.Context(), repo, owner, run)

		got := reload(t)
		assert.Equal(t, agent_model.StateCompleted, got.State)
		// completing is not archiving: the record should still be visible where
		// somebody would go to look at what the agent did
		assert.True(t, got.ArchivedAt.IsZero(), "a finished task must not vanish from default listings")
	})

	t.Run("AFinishedTaskIsNotWalkedBack", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		run := newRun(t, actions_model.StatusFailure, &api.User{UserName: botUser.Name})
		notifier.WorkflowRunStatusUpdate(t.Context(), repo, owner, run)
		assert.Equal(t, agent_model.StateCompleted, reload(t).State)
	})
}
