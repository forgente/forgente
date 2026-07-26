// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	issues_model "forgente.com/models/issues"
	"forgente.com/models/organization"
	repo_model "forgente.com/models/repo"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"
	issue_service "forgente.com/services/issue"
	org_service "forgente.com/services/org"
	user_service "forgente.com/services/user"
	"forgente.com/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueAgentTaskSidebar(t *testing.T) {
	onGiteaRun(t, testIssueAgentTaskSidebar)
}

func testIssueAgentTaskSidebar(t *testing.T, _ *url.URL) {
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org3"})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: owner.ID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 1})
	require.NoError(t, issue.LoadRepo(t.Context()))

	session := loginUser(t, owner.Name)
	issueLink := fmt.Sprintf("/%s/%s/issues/%d", owner.Name, repo.Name, issue.Index)

	t.Run("NoAgentWorkShowsNothing", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		body := session.MakeRequest(t, NewRequest(t, "GET", issueLink), http.StatusOK).Body.String()
		// the section is absent entirely rather than rendering an empty block
		assert.NotContains(t, body, "Agent Work")
	})

	botUser, _, err := user_service.CreateOrgApp(t.Context(), owner, org, "sidebarbot", "does the work")
	require.NoError(t, err)
	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{OrgID: org.ID, LowerName: "owners"})
	require.NoError(t, org_service.AddTeamMember(t.Context(), team, botUser))

	t.Run("AssignedAgentAppears", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		_, _, err := issue_service.ToggleAssigneeWithNotify(t.Context(), issue, owner, botUser.ID)
		require.NoError(t, err)

		body := session.MakeRequest(t, NewRequest(t, "GET", issueLink), http.StatusOK).Body.String()
		assert.Contains(t, body, "Agent Work")
		assert.Contains(t, body, "sidebarbot")
		assert.Contains(t, body, "Queued")
	})

	t.Run("UnassignedAgentShowsCancelled", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		_, _, err := issue_service.ToggleAssigneeWithNotify(t.Context(), issue, owner, botUser.ID)
		require.NoError(t, err)

		// the record stays visible after unassignment: what an agent did to an
		// issue is history, not something that disappears with the assignment
		body := session.MakeRequest(t, NewRequest(t, "GET", issueLink), http.StatusOK).Body.String()
		assert.Contains(t, body, "Agent Work")
		assert.Contains(t, body, "Cancelled")
	})
}
