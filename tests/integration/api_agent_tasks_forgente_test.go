// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	agent_model "forgente.com/models/agent"
	auth_model "forgente.com/models/auth"
	issues_model "forgente.com/models/issues"
	"forgente.com/models/organization"
	repo_model "forgente.com/models/repo"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"
	api "forgente.com/modules/structs"
	agent_service "forgente.com/services/agent"
	issue_service "forgente.com/services/issue"
	org_service "forgente.com/services/org"
	user_service "forgente.com/services/user"
	"forgente.com/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIAgentTasks(t *testing.T) {
	onGiteaRun(t, testAPIAgentTasks)
}

func testAPIAgentTasks(t *testing.T, _ *url.URL) {
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org3"})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: owner.ID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 1})
	require.NoError(t, issue.LoadRepo(t.Context()))

	botUser, app, err := user_service.CreateOrgApp(t.Context(), owner, org, "apitaskbot", "answers questions")
	require.NoError(t, err)
	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{OrgID: org.ID, LowerName: "owners"})
	require.NoError(t, org_service.AddTeamMember(t.Context(), team, botUser))

	_, _, err = issue_service.ToggleAssigneeWithNotify(t.Context(), issue, owner, botUser.ID)
	require.NoError(t, err)
	task, err := agent_service.TaskForIssue(t.Context(), issue.ID, app.ID)
	require.NoError(t, err)
	require.NotNil(t, task)

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadIssue)
	listLink := fmt.Sprintf("/api/v1/repos/%s/%s/agent/tasks", owner.Name, repo.Name)

	t.Run("List", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", listLink).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)
		var tasks []*api.AgentTask
		DecodeJSON(t, resp, &tasks)

		require.Len(t, tasks, 1)
		assert.Equal(t, task.ID, tasks[0].ID)
		assert.Equal(t, string(agent_model.StateQueued), tasks[0].State)
		// the expansions are the reason to call this rather than read the table
		require.NotNil(t, tasks[0].Issue)
		assert.Equal(t, issue.Title, tasks[0].Issue.Title)
		require.NotNil(t, tasks[0].Agent)
		assert.Equal(t, botUser.Name, tasks[0].Agent.UserName)
		require.NotNil(t, tasks[0].Creator)
		assert.Equal(t, owner.Name, tasks[0].Creator.UserName)
	})

	t.Run("FilterByState", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", listLink+"?state=queued").AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)
		var tasks []*api.AgentTask
		DecodeJSON(t, resp, &tasks)
		assert.Len(t, tasks, 1)

		req = NewRequest(t, "GET", listLink+"?state=completed").AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		DecodeJSON(t, resp, &tasks)
		assert.Empty(t, tasks)
	})

	t.Run("UnknownStateIsRejected", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// a typo should not silently return everything
		req := NewRequest(t, "GET", listLink+"?state=running").AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
	})

	t.Run("Get", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("%s/%d", listLink, task.ID)).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)
		var got api.AgentTask
		DecodeJSON(t, resp, &got)
		assert.Equal(t, task.ID, got.ID)
	})

	t.Run("ATaskFromAnotherRepositoryIsNotFound", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		other := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/agent/tasks/%d",
			other.OwnerName, other.Name, task.ID)).AddTokenAuth(token)
		// not found rather than forbidden, so ids cannot be probed across repos
		MakeRequest(t, req, http.StatusNotFound)
	})
}
