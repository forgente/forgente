// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/url"
	"testing"

	agent_model "forgente.com/models/agent"
	issues_model "forgente.com/models/issues"
	"forgente.com/models/organization"
	repo_model "forgente.com/models/repo"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"
	agent_service "forgente.com/services/agent"
	issue_service "forgente.com/services/issue"
	org_service "forgente.com/services/org"
	user_service "forgente.com/services/user"
	"forgente.com/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentDispatchOnAssignment(t *testing.T) {
	onGiteaRun(t, testAgentDispatchOnAssignment)
}

func testAgentDispatchOnAssignment(t *testing.T, _ *url.URL) {
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org3"})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: owner.ID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 1})
	require.NoError(t, issue.LoadRepo(t.Context()))

	botUser, app, err := user_service.CreateOrgApp(t.Context(), owner, org, "dispatchbot", "runs errands")
	require.NoError(t, err)

	// an app only becomes assignable once it has write access, granted the
	// ordinary way — this is the undocumented step the connect panel names
	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{OrgID: org.ID, LowerName: "owners"})
	require.NoError(t, org_service.AddTeamMember(t.Context(), team, botUser))

	t.Run("AssignmentStartsATask", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		_, _, err = issue_service.ToggleAssigneeWithNotify(t.Context(), issue, owner, botUser.ID)
		require.NoError(t, err)

		task, err := agent_service.TaskForIssue(t.Context(), issue.ID, app.ID)
		require.NoError(t, err)
		require.NotNil(t, task, "assigning an app to an issue should record a task")
		assert.Equal(t, agent_model.StateQueued, task.State)
		assert.Equal(t, issue.ID, task.IssueID)
		assert.Equal(t, app.ID, task.AppID)
		assert.Equal(t, org.ID, task.OwnerID)
		assert.Equal(t, owner.ID, task.CreatorID)
	})

	t.Run("UnassignmentCancelsIt", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		_, _, err = issue_service.ToggleAssigneeWithNotify(t.Context(), issue, owner, botUser.ID)
		require.NoError(t, err)

		task, err := agent_service.TaskForIssue(t.Context(), issue.ID, app.ID)
		require.NoError(t, err)
		require.NotNil(t, task)
		assert.Equal(t, agent_model.StateCancelled, task.State)
	})

	t.Run("ReassignmentRevivesRatherThanDuplicates", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		before, err := agent_service.TaskForIssue(t.Context(), issue.ID, app.ID)
		require.NoError(t, err)
		require.NotNil(t, before)

		_, _, err = issue_service.ToggleAssigneeWithNotify(t.Context(), issue, owner, botUser.ID)
		require.NoError(t, err)

		after, err := agent_service.TaskForIssue(t.Context(), issue.ID, app.ID)
		require.NoError(t, err)
		require.NotNil(t, after)
		// the same task comes back to life; asking twice is not two requests
		assert.Equal(t, before.ID, after.ID)
		assert.Equal(t, agent_model.StateQueued, after.State)
	})

	t.Run("SuspendedAppRecordsNothing", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// the kill switch should stop new work being recorded, not only new
		// tokens being accepted
		require.NoError(t, user_model.SetForgenteAppSuspended(t.Context(), app.ID, true))
		defer func() {
			require.NoError(t, user_model.SetForgenteAppSuspended(t.Context(), app.ID, false))
		}()

		third := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 4})
		require.NoError(t, third.LoadRepo(t.Context()))
		_, _, err = issue_service.ToggleAssigneeWithNotify(t.Context(), third, owner, botUser.ID)
		require.NoError(t, err)

		task, err := agent_service.TaskForIssue(t.Context(), third.ID, app.ID)
		require.NoError(t, err)
		assert.Nil(t, task, "a suspended app should not pick up new work")
	})

	t.Run("AssigningAPersonRecordsNothing", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		other := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 2})
		require.NoError(t, other.LoadRepo(t.Context()))
		_, _, err = issue_service.ToggleAssigneeWithNotify(t.Context(), other, owner, owner.ID)
		require.NoError(t, err)

		task, err := agent_service.TaskForIssue(t.Context(), other.ID, app.ID)
		require.NoError(t, err)
		assert.Nil(t, task)
	})
}
