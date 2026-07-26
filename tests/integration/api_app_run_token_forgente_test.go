// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"encoding/hex"
	"net/http"
	"testing"

	actions_model "forgente.com/models/actions"
	auth_model "forgente.com/models/auth"
	"forgente.com/models/db"
	"forgente.com/models/organization"
	repo_model "forgente.com/models/repo"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"
	api "forgente.com/modules/structs"
	"forgente.com/modules/util"
	org_service "forgente.com/services/org"
	user_service "forgente.com/services/user"
	"forgente.com/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runIndex keeps each inserted run unique per repository, which the schema
// requires.
var runIndex int64 = 900

// newRunningTask inserts a whole run, job and task of its own rather than
// reusing a fixture. Fixture 47 is the obvious candidate but belongs to a
// user-owned repository, which cannot carry an organization's grant — and
// mutating a shared fixture to suit one test has broken unrelated suites here
// before.
//
// The job is not optional scaffolding: repoAssignment resolves an Actions
// caller's permissions through task.LoadJob, so a task without one fails long
// before reaching the handler.
func newRunningTask(t *testing.T, repoID, ownerID int64, isFork bool) string {
	t.Helper()

	runIndex++
	run := &actions_model.ActionRun{
		Title:             "app run token test",
		RepoID:            repoID,
		OwnerID:           ownerID,
		WorkflowID:        "test.yaml",
		Index:             runIndex,
		TriggerUserID:     ownerID,
		Ref:               "refs/heads/master",
		IsForkPullRequest: isFork,
		Event:             "push",
		Status:            actions_model.StatusRunning,
	}
	require.NoError(t, db.Insert(t.Context(), run))

	job := &actions_model.ActionRunJob{
		RunID:             run.ID,
		RepoID:            repoID,
		OwnerID:           ownerID,
		IsForkPullRequest: isFork,
		Name:              "act",
		JobID:             "act",
		Attempt:           1,
		Status:            actions_model.StatusRunning,
	}
	require.NoError(t, db.Insert(t.Context(), job))

	token := hex.EncodeToString(util.CryptoRandomBytes(20))
	salt := util.CryptoRandomString(10)
	task := &actions_model.ActionTask{
		JobID:             job.ID,
		RepoID:            repoID,
		OwnerID:           ownerID,
		Attempt:           1,
		Status:            actions_model.StatusRunning,
		IsForkPullRequest: isFork,
		TokenHash:         auth_model.HashToken(token, salt),
		TokenSalt:         salt,
		TokenLastEight:    token[len(token)-8:],
	}
	require.NoError(t, db.Insert(t.Context(), task))
	return token
}

func TestAPIAppRunToken(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org3"})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3, OwnerID: org.ID})
	mintLink := "/api/v1/repos/" + repo.OwnerName + "/" + repo.Name + "/actions/app-token"

	botUser, app, err := user_service.CreateOrgApp(t.Context(), doer, org, "tokenbot", "acts inside runs")
	require.NoError(t, err)
	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{OrgID: org.ID, LowerName: "owners"})
	require.NoError(t, org_service.AddTeamMember(t.Context(), team, botUser))

	taskToken := newRunningTask(t, repo.ID, org.ID, false)
	body := map[string]string{"app": botUser.Name}

	t.Run("WithoutAGrantItIsRefused", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST", mintLink, body).AddTokenAuth(taskToken)
		MakeRequest(t, req, http.StatusForbidden)
	})

	// read:user only, so the ceiling can be demonstrated as well as the identity
	_, err = user_service.GrantAppToRepoRuns(t.Context(), doer, org, app.ID, repo.ID, auth_model.AccessTokenScopeReadUser)
	require.NoError(t, err)

	var minted api.AppRunToken

	t.Run("AGrantedRunGetsAToken", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST", mintLink, body).AddTokenAuth(taskToken)
		resp := MakeRequest(t, req, http.StatusCreated)
		DecodeJSON(t, resp, &minted)

		assert.NotEmpty(t, minted.Token)
		assert.Equal(t, botUser.Name, minted.App)
		// the scope comes from the grant, not from what the app can reach
		assert.Equal(t, string(auth_model.AccessTokenScopeReadUser), minted.Scope)
		assert.False(t, minted.ExpiresAt.IsZero(), "an expiry is set")
	})

	t.Run("TheGrantIsACeiling", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// the app is an organization owner and could write issues with a token
		// of its own; this run may not, because the grant did not say so
		req := NewRequestWithJSON(t, "POST",
			"/api/v1/repos/"+repo.OwnerName+"/"+repo.Name+"/issues",
			&api.CreateIssueOption{Title: "from a run"}).AddTokenAuth(minted.Token)
		MakeRequest(t, req, http.StatusForbidden)
	})

	t.Run("TheTokenActsAsTheApp", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// the whole point: what the run does is attributed to the app, not to
		// the person who wrote the workflow and not to the Actions user
		req := NewRequest(t, "GET", "/api/v1/user").AddTokenAuth(minted.Token)
		resp := MakeRequest(t, req, http.StatusOK)
		var whoami api.User
		DecodeJSON(t, resp, &whoami)
		assert.Equal(t, botUser.Name, whoami.UserName)
	})

	t.Run("AnOrdinaryUserTokenCannotMint", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// only a run may ask, and a personal token carries no task
		session := loginUser(t, doer.Name)
		userToken := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)
		req := NewRequestWithJSON(t, "POST", mintLink, body).AddTokenAuth(userToken)
		MakeRequest(t, req, http.StatusForbidden)
	})

	t.Run("AForkPullRequestRunIsRefused", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// a fork run executes code the repository's collaborators never
		// approved, so it must not be able to borrow the organization's identity
		forkToken := newRunningTask(t, repo.ID, org.ID, true)
		req := NewRequestWithJSON(t, "POST", mintLink, body).AddTokenAuth(forkToken)
		MakeRequest(t, req, http.StatusForbidden)
	})

	t.Run("ARunFromAnotherRepositoryIsRefused", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// the task decides which repository is asking, not the route. This is
		// refused before the handler runs at all: an Actions caller's
		// permissions are computed against the route's repository through its
		// own task, so a run from elsewhere simply cannot see this one — hence
		// not-found rather than forbidden. The handler's own repository check
		// stays as defence in depth for any path that does not go through
		// repoAssignment.
		otherToken := newRunningTask(t, 5, org.ID, false)
		req := NewRequestWithJSON(t, "POST", mintLink, body).AddTokenAuth(otherToken)
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("SuspendingTheAppStopsTheMintedToken", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		require.NoError(t, user_model.SetForgenteAppSuspended(t.Context(), app.ID, true))
		defer func() {
			require.NoError(t, user_model.SetForgenteAppSuspended(t.Context(), app.ID, false))
		}()

		// the kill switch is enforced once, after any auth method resolves a
		// user, so a credential type added later cannot slip past it
		req := NewRequest(t, "GET", "/api/v1/user").AddTokenAuth(minted.Token)
		MakeRequest(t, req, http.StatusUnauthorized)

		// and a suspended app cannot be picked up by a fresh run either
		req = NewRequestWithJSON(t, "POST", mintLink, body).AddTokenAuth(taskToken)
		MakeRequest(t, req, http.StatusForbidden)
	})

	t.Run("AStoppedTaskTakesItsTokenWithIt", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		stoppedToken := newRunningTask(t, repo.ID, org.ID, false)
		req := NewRequestWithJSON(t, "POST", mintLink, body).AddTokenAuth(stoppedToken)
		resp := MakeRequest(t, req, http.StatusCreated)
		var short api.AppRunToken
		DecodeJSON(t, resp, &short)

		req = NewRequest(t, "GET", "/api/v1/user").AddTokenAuth(short.Token)
		MakeRequest(t, req, http.StatusOK)

		// the run finishes; its credential should stop working immediately
		// rather than staying live for the rest of its hour with nothing left
		// to attribute it to
		task := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{TokenLastEight: stoppedToken[len(stoppedToken)-8:]})
		_, err := db.GetEngine(t.Context()).ID(task.ID).Cols("status").Update(&actions_model.ActionTask{Status: actions_model.StatusSuccess})
		require.NoError(t, err)

		req = NewRequest(t, "GET", "/api/v1/user").AddTokenAuth(short.Token)
		MakeRequest(t, req, http.StatusUnauthorized)
	})
}
