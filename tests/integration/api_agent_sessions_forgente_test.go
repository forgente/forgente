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
	org_service "forgente.com/services/org"
	user_service "forgente.com/services/user"
	"forgente.com/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIAgentSessions(t *testing.T) {
	onGiteaRun(t, testAPIAgentSessions)
}

func testAPIAgentSessions(t *testing.T, _ *url.URL) {
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org3"})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: owner.ID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 1})
	require.NoError(t, issue.LoadRepo(t.Context()))

	botUser, app, err := user_service.CreateOrgApp(t.Context(), owner, org, "apisessionbot", "attempts things")
	require.NoError(t, err)
	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{OrgID: org.ID, LowerName: "owners"})
	require.NoError(t, org_service.AddTeamMember(t.Context(), team, botUser))

	task, err := agent_service.StartTaskForIssue(t.Context(), owner, issue, app)
	require.NoError(t, err)
	require.NotNil(t, task)

	// finish the first attempt, then start a second, so the listing has to show
	// two outcomes rather than one
	first, err := agent_service.ActiveSessionForTask(t.Context(), task.ID)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NoError(t, agent_service.SetSessionState(t.Context(), first, agent_model.StateCompleted))

	second, err := agent_service.StartSession(t.Context(), task, "another go")
	require.NoError(t, err)
	require.NoError(t, agent_service.SetSessionState(t.Context(), second, agent_model.StateFailed))

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadIssue)

	t.Run("ListShowsEveryAttempt", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/agent/tasks/%d/sessions", repo.FullName(), task.ID)).
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var got []*api.AgentSession
		DecodeJSON(t, resp, &got)
		require.Len(t, got, 2)
		// newest first, matching the task listing
		assert.Equal(t, second.ID, got[0].ID)
		assert.Equal(t, string(agent_model.StateFailed), got[0].State)
		assert.Equal(t, first.ID, got[1].ID)
		assert.Equal(t, string(agent_model.StateCompleted), got[1].State,
			"the attempt that succeeded must still say so after a later one failed")
		assert.Equal(t, task.ID, got[0].TaskID)
	})

	t.Run("GetReturnsOneAttempt", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/agent/sessions/%d", repo.FullName(), first.ID)).
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var got *api.AgentSession
		DecodeJSON(t, resp, &got)
		assert.Equal(t, first.ID, got.ID)
		assert.Equal(t, string(agent_model.StateCompleted), got.State)
		assert.NotNil(t, got.CompletedAt, "a finished attempt records when it finished")
	})

	t.Run("ASessionFromAnotherRepositoryIsNotFound", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// reported missing rather than forbidden, so an id cannot be probed
		// across repositories — the same rule the task endpoints follow
		other := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/agent/sessions/%d", other.FullName(), first.ID)).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("SessionsOfAnotherRepositorysTaskAreNotFound", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		other := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/agent/tasks/%d/sessions", other.FullName(), task.ID)).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("AMissingTaskIsNotFound", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/agent/tasks/999999/sessions", repo.FullName())).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("TheTaskEndpointsStillWork", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// the routes were regrouped to hang sessions off the same prefix, so the
		// endpoints that already existed are worth re-checking
		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/agent/tasks/%d", repo.FullName(), task.ID)).
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var got *api.AgentTask
		DecodeJSON(t, resp, &got)
		assert.Equal(t, task.ID, got.ID)
		assert.Equal(t, string(agent_model.StateFailed), got.State)
	})
}
