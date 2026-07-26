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
	org_service "forgente.com/services/org"
	user_service "forgente.com/services/user"
	"forgente.com/tests"

	"github.com/stretchr/testify/require"
)

// Marking work ready for review is the moment it enters human review, and the
// author of the work is the one principal who should not decide it has arrived
// there. An agent that opens a draft and then promotes it has reviewed itself.
func TestAPIAppCannotReadyItsOwnPull(t *testing.T) {
	onGiteaRun(t, testAPIAppCannotReadyItsOwnPull)
}

func testAPIAppCannotReadyItsOwnPull(t *testing.T, _ *url.URL) {
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org3"})

	botUser, _, err := user_service.CreateOrgApp(t.Context(), owner, org, "draftbot", "opens drafts")
	require.NoError(t, err)
	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{OrgID: org.ID, LowerName: "owners"})
	require.NoError(t, org_service.AddTeamMember(t.Context(), team, botUser))

	appAccessToken := &auth_model.AccessToken{UID: botUser.ID, Name: "ready-test", Scope: auth_model.AccessTokenScopeAll}
	require.NoError(t, auth_model.NewAccessToken(t.Context(), appAccessToken))
	appToken := appAccessToken.Token

	ownerToken := getTokenForLoggedInUser(t, loginUser(t, owner.Name), auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/org3/repo3/contents/DRAFT.md", &api.CreateFileOptions{
		FileOptions: api.FileOptions{
			BranchName:    "master",
			NewBranchName: "app-draft",
			Message:       "work in progress by the app",
		},
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("wip\n")),
	}).AddTokenAuth(appToken)
	MakeRequest(t, req, http.StatusCreated)

	// the app opens its work as a draft, which is the posture that earns it the
	// guarantee: the forge holds the promotion for a person
	req = NewRequestWithJSON(t, "POST", "/api/v1/repos/org3/repo3/pulls", &api.CreatePullRequestOption{
		Title: "WIP: work from the app", Head: "app-draft", Base: "master",
	}).AddTokenAuth(appToken)
	resp := MakeRequest(t, req, http.StatusCreated)
	var pull api.PullRequest
	DecodeJSON(t, resp, &pull)
	require.Equal(t, botUser.Name, pull.Poster.UserName, "the app must be the author for this test to mean anything")

	pullLink := fmt.Sprintf("/api/v1/repos/org3/repo3/pulls/%d", pull.Index)

	t.Run("TheAppIsRefused", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "PATCH", pullLink,
			&api.EditPullRequestOption{Title: "work from the app"}).AddTokenAuth(appToken)
		MakeRequest(t, req, http.StatusForbidden)

		// and the refusal actually held, rather than being reported after the fact
		req = NewRequest(t, "GET", pullLink).AddTokenAuth(appToken)
		resp := MakeRequest(t, req, http.StatusOK)
		var after api.PullRequest
		DecodeJSON(t, resp, &after)
		require.Equal(t, "WIP: work from the app", after.Title)
	})

	t.Run("TheAppMayStillRenameItsDraft", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// only the promotion is refused; the app owns its work while it is a draft
		req := NewRequestWithJSON(t, "PATCH", pullLink,
			&api.EditPullRequestOption{Title: "WIP: better description"}).AddTokenAuth(appToken)
		MakeRequest(t, req, http.StatusCreated)
	})

	t.Run("APersonStillCan", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// the refusal is about who opened it, not about the title itself
		req := NewRequestWithJSON(t, "PATCH", pullLink,
			&api.EditPullRequestOption{Title: "better description"}).AddTokenAuth(ownerToken)
		MakeRequest(t, req, http.StatusCreated)
	})
}

// An ordinary bot is somebody else's automation, governed by whoever runs it.
// The rule is scoped to apps: the principal this forge issues and vouches for.
func TestAPIAppReadyRuleIsScopedToApps(t *testing.T) {
	onGiteaRun(t, testAPIAppReadyRuleIsScopedToApps)
}

func testAPIAppReadyRuleIsScopedToApps(t *testing.T, _ *url.URL) {
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	ownerToken := getTokenForLoggedInUser(t, loginUser(t, owner.Name), auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/contents/PLAIN.md", &api.CreateFileOptions{
		FileOptions: api.FileOptions{
			BranchName:    "master",
			NewBranchName: "plain-draft",
			Message:       "a draft from a person",
		},
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("wip\n")),
	}).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusCreated)

	req = NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/pulls", &api.CreatePullRequestOption{
		Title: "WIP: from a person", Head: "plain-draft", Base: "master",
	}).AddTokenAuth(ownerToken)
	resp := MakeRequest(t, req, http.StatusCreated)
	var pull api.PullRequest
	DecodeJSON(t, resp, &pull)

	// a person promoting their own draft is ordinary, and must stay ordinary
	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/user2/repo1/pulls/%d", pull.Index),
		&api.EditPullRequestOption{Title: "from a person"}).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusCreated)
}
