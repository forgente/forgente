// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	auth_model "forgente.com/models/auth"
	"forgente.com/models/organization"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"
	api "forgente.com/modules/structs"
	"forgente.com/services/forms"
	org_service "forgente.com/services/org"
	user_service "forgente.com/services/user"
	"forgente.com/tests"

	"github.com/stretchr/testify/require"
)

// An app opening a pull request and then merging it removes the human from the
// loop entirely, which is the one thing that makes running agents against a
// real repository defensible. Approving your own work is already refused for
// everyone; this is the same conflict with a larger blast radius.
func TestAPIAppCannotMergeItsOwnPull(t *testing.T) {
	onGiteaRun(t, testAPIAppCannotMergeItsOwnPull)
}

func testAPIAppCannotMergeItsOwnPull(t *testing.T, _ *url.URL) {
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org3"})

	botUser, _, err := user_service.CreateOrgApp(t.Context(), owner, org, "mergebot", "opens things")
	require.NoError(t, err)
	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{OrgID: org.ID, LowerName: "owners"})
	require.NoError(t, org_service.AddTeamMember(t.Context(), team, botUser))

	// the app acts as itself, never as a person, so it needs its own token
	appAccessToken := &auth_model.AccessToken{UID: botUser.ID, Name: "merge-test", Scope: auth_model.AccessTokenScopeAll}
	require.NoError(t, auth_model.NewAccessToken(t.Context(), appAccessToken))
	appToken := appAccessToken.Token

	ownerToken := getTokenForLoggedInUser(t, loginUser(t, owner.Name), auth_model.AccessTokenScopeAll)

	// a branch and a commit, made by the app
	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/org3/repo3/contents/AGENT.md", &api.CreateFileOptions{
		FileOptions: api.FileOptions{
			BranchName:    "master",
			NewBranchName: "app-work",
			Message:       "work by the app",
		},
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("done\n")),
	}).AddTokenAuth(appToken)
	MakeRequest(t, req, http.StatusCreated)

	req = NewRequestWithJSON(t, "POST", "/api/v1/repos/org3/repo3/pulls", &api.CreatePullRequestOption{
		Title: "work from the app", Head: "app-work", Base: "master",
	}).AddTokenAuth(appToken)
	resp := MakeRequest(t, req, http.StatusCreated)
	var pull api.PullRequest
	DecodeJSON(t, resp, &pull)
	require.Equal(t, botUser.Name, pull.Poster.UserName, "the app must be the author for this test to mean anything")

	mergeLink := fmt.Sprintf("/api/v1/repos/org3/repo3/pulls/%d/merge", pull.Index)

	t.Run("TheAppIsRefused", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST", mergeLink,
			&forms.MergePullRequestForm{Do: "merge"}).AddTokenAuth(appToken)
		MakeRequest(t, req, http.StatusMethodNotAllowed)
	})

	t.Run("SomebodyElseStillCan", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// the refusal is about who opened it, not about what the app may do:
		// the same pull request merges fine for a person
		req := NewRequestWithJSON(t, "POST", mergeLink,
			&forms.MergePullRequestForm{Do: "merge"}).AddTokenAuth(ownerToken)
		MakeRequest(t, req, http.StatusOK)
	})
}
