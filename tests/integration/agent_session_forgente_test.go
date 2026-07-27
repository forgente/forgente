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

// TestAgentSessionsRecordEveryAttempt is the regression for the defect that
// motivated sessions: with nowhere to record an attempt, a retry overwrote the
// previous one's outcome, so a task whose first run succeeded ended up
// reporting the failure of a later one and the success became unrecoverable.
func TestAgentSessionsRecordEveryAttempt(t *testing.T) {
	onGiteaRun(t, testAgentSessionsRecordEveryAttempt)
}

func testAgentSessionsRecordEveryAttempt(t *testing.T, _ *url.URL) {
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org3"})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: owner.ID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 1})
	require.NoError(t, issue.LoadRepo(t.Context()))

	botUser, app, err := user_service.CreateOrgApp(t.Context(), owner, org, "sessionbot", "tries twice")
	require.NoError(t, err)
	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{OrgID: org.ID, LowerName: "owners"})
	require.NoError(t, org_service.AddTeamMember(t.Context(), team, botUser))

	runIndex := int64(900)
	finishRun := func(t *testing.T, status actions_model.Status) {
		t.Helper()
		runIndex++
		payload, err := json.Marshal(&api.IssuePayload{
			Action:   api.HookIssueAssigned,
			Index:    issue.Index,
			Assignee: &api.User{UserName: botUser.Name},
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
		agent_service.NewNotifier().WorkflowRunStatusUpdate(t.Context(), repo, owner, run)
	}

	toggleAssignee := func(t *testing.T) {
		t.Helper()
		_, _, err := issue_service.ToggleAssigneeWithNotify(t.Context(), issue, owner, botUser.ID)
		require.NoError(t, err)
	}

	reloadTask := func(t *testing.T) *agent_model.Task {
		t.Helper()
		got, err := agent_service.TaskForIssue(t.Context(), issue.ID, app.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		return got
	}

	// first attempt: assigned, and its run succeeds
	toggleAssignee(t)
	task := reloadTask(t)
	finishRun(t, actions_model.StatusSuccess)
	require.Equal(t, agent_model.StateCompleted, reloadTask(t).State)

	// second attempt: unassigned, assigned again, and this run fails
	toggleAssignee(t)
	toggleAssignee(t)
	finishRun(t, actions_model.StatusFailure)

	t.Run("BothAttemptsAreRecorded", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		sessions, err := agent_service.SessionsForTask(t.Context(), task.ID)
		require.NoError(t, err)
		require.Len(t, sessions, 2, "each attempt at a task needs its own record")
	})

	t.Run("TheSuccessfulAttemptSurvivesTheFailedOne", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		sessions, err := agent_service.SessionsForTask(t.Context(), task.ID)
		require.NoError(t, err)
		require.Len(t, sessions, 2)

		// newest first, so the retry is [0] and the original attempt is [1]
		assert.Equal(t, agent_model.StateFailed, sessions[0].State)
		assert.Equal(t, agent_model.StateCompleted, sessions[1].State,
			"the attempt that succeeded must still say so after a later one failed")
	})

	t.Run("TheTaskReportsItsMostRecentAttempt", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// honest rather than flattering: the last attempt failed, and whatever
		// the first one produced is still findable on its own session
		assert.Equal(t, agent_model.StateFailed, reloadTask(t).State)
	})

	t.Run("TheTaskKeepsItsIdentityAcrossAttempts", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		assert.Equal(t, task.ID, reloadTask(t).ID, "a retry is another attempt, not another task")
	})
}
