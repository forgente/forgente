// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	actions_model "forgente.com/models/actions"
	auth_model "forgente.com/models/auth"
	"forgente.com/models/db"
	"forgente.com/models/organization"
	repo_model "forgente.com/models/repo"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"
	org_service "forgente.com/services/org"
	user_service "forgente.com/services/user"
	"forgente.com/tests"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newLabelledRunner(t *testing.T, ownerID int64, name string, labels []string) *actions_model.ActionRunner {
	t.Helper()

	runner := &actions_model.ActionRunner{
		// the column is unique and nothing fills it in for a direct insert
		UUID:        uuid.NewString(),
		Name:        name,
		OwnerID:     ownerID,
		AgentLabels: labels,
	}
	runner.GenerateAndFillToken()
	require.NoError(t, db.Insert(t.Context(), runner))
	return runner
}

// A grant may name a runner label, and then a run can only claim the app while
// it is executing on a runner carrying that label.
//
// Routing by label already existed: a job's `runs-on` decides which runner
// picks it up. What did not exist was anything holding it there — whoever can
// edit a workflow can edit that line, and the app token was minted regardless.
// So the designation is checked where the run asks to become the app.
func TestAPIAppRunTokenRunnerLabel(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org3"})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3, OwnerID: org.ID})
	mintLink := "/api/v1/repos/" + repo.OwnerName + "/" + repo.Name + "/actions/app-token"

	botUser, app, err := user_service.CreateOrgApp(t.Context(), doer, org, "labelbot", "runs somewhere specific")
	require.NoError(t, err)
	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{OrgID: org.ID, LowerName: "owners"})
	require.NoError(t, org_service.AddTeamMember(t.Context(), team, botUser))

	restricted := newLabelledRunner(t, org.ID, "restricted", []string{"ubuntu-latest", "egress-restricted"})
	ordinary := newLabelledRunner(t, org.ID, "ordinary", []string{"ubuntu-latest"})

	body := map[string]string{"app": botUser.Name}

	_, err = user_service.GrantAppToRepoRuns(t.Context(), doer, org, app.ID, repo.ID,
		auth_model.AccessTokenScopeReadUser, "egress-restricted")
	require.NoError(t, err)

	t.Run("OnTheDesignatedRunnerItIsMinted", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		token := newRunningTaskOnRunner(t, repo.ID, org.ID, false, restricted.ID)
		req := NewRequestWithJSON(t, "POST", mintLink, body).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusCreated)
	})

	t.Run("OnAnotherRunnerItIsRefused", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// the same repository, the same grant, the same app — only the runner
		// differs, which is the whole point of the restriction
		token := newRunningTaskOnRunner(t, repo.ID, org.ID, false, ordinary.ID)
		req := NewRequestWithJSON(t, "POST", mintLink, body).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusForbidden)
	})

	t.Run("WithNoRunnerItIsRefused", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// fails closed: a run whose location cannot be established must not be
		// handed an identity that was restricted by location
		token := newRunningTaskOnRunner(t, repo.ID, org.ID, false, 0)
		req := NewRequestWithJSON(t, "POST", mintLink, body).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusForbidden)
	})

	t.Run("ClearingTheLabelAllowsAnyRunnerAgain", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// re-granting with an empty label must actually clear the column rather
		// than leave the old restriction in place, which is what a non-zero
		// column update would have done
		_, err := user_service.GrantAppToRepoRuns(t.Context(), doer, org, app.ID, repo.ID,
			auth_model.AccessTokenScopeReadUser, "")
		require.NoError(t, err)

		grant, err := user_model.GetForgenteAppRunGrant(t.Context(), app.ID, repo.ID)
		require.NoError(t, err)
		require.Empty(t, grant.RunnerLabel)

		token := newRunningTaskOnRunner(t, repo.ID, org.ID, false, ordinary.ID)
		req := NewRequestWithJSON(t, "POST", mintLink, body).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusCreated)
	})
}
