// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	auth_model "forgente.com/models/auth"
	"forgente.com/models/organization"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"
	org_service "forgente.com/services/org"
	"forgente.com/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrgApps(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	const orgName, appName = "org3", "deploybot"
	appsLink := fmt.Sprintf("/org/%s/settings/applications", orgName)

	owner := loginUser(t, "user2")
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: orgName})

	t.Run("OnlyOwnersReachThePage", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// user4 belongs to the organization but does not own it
		loginUser(t, "user4").MakeRequest(t, NewRequest(t, "GET", appsLink), http.StatusNotFound)
		owner.MakeRequest(t, NewRequest(t, "GET", appsLink), http.StatusOK)
	})

	t.Run("Create", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", appsLink+"/apps", map[string]string{
			"name":        appName,
			"description": "ships things",
		})
		owner.MakeRequest(t, req, http.StatusSeeOther)

		botUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: appName})
		assert.True(t, botUser.IsTypeBot())
		// creating an app grants it nothing; access comes from team membership
		assert.False(t, botUser.CanCreateOrganization())

		app := unittest.AssertExistsAndLoadBean(t, &user_model.ForgenteApp{UserID: botUser.ID})
		assert.Equal(t, org.ID, app.OwnerID)
		assert.False(t, app.Suspended)

		resp := owner.MakeRequest(t, NewRequest(t, "GET", appsLink), http.StatusOK)
		assert.Contains(t, resp.Body.String(), appName)
	})

	t.Run("NameIsTakenByAnExistingAccount", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", appsLink+"/apps", map[string]string{"name": "user1"})
		owner.MakeRequest(t, req, http.StatusSeeOther)

		unittest.AssertNotExistsBean(t, &user_model.ForgenteApp{UserID: 1})
	})

	t.Run("Tokens", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		botUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: appName})
		app := unittest.AssertExistsAndLoadBean(t, &user_model.ForgenteApp{UserID: botUser.ID})
		tokensLink := fmt.Sprintf("%s/apps/%d/tokens", appsLink, app.ID)

		req := NewRequestWithValues(t, "POST", tokensLink, map[string]string{
			"name":              "ci",
			"scope-repository":  "write:repository",
			"scope-user":        "read:user",
			"scope-public-only": "",
		})
		owner.MakeRequest(t, req, http.StatusSeeOther)

		// the raw token is shown once, in the flash
		flashes := owner.GetCookieFlashMessage()
		require.NotNil(t, flashes)
		rawToken := flashes.InfoMsg
		assert.NotEmpty(t, rawToken)

		token := unittest.AssertExistsAndLoadBean(t, &auth_model.AccessToken{UID: botUser.ID, Name: "ci"})
		hasScope, err := token.Scope.HasScope(auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeReadUser)
		require.NoError(t, err)
		assert.True(t, hasScope)
		// nothing was granted beyond what the form asked for
		hasScope, err = token.Scope.HasScope(auth_model.AccessTokenScopeWriteIssue)
		require.NoError(t, err)
		assert.False(t, hasScope)

		// the token authenticates as the app, not as the owner who minted it
		req = NewRequest(t, "GET", "/api/v1/user").AddTokenAuth(rawToken)
		resp := MakeRequest(t, req, http.StatusOK)
		var apiUser struct {
			Login string `json:"login"`
		}
		DecodeJSON(t, resp, &apiUser)
		assert.Equal(t, appName, apiUser.Login)

		// a suspended app cannot use it
		suspendReq := NewRequestWithURLValues(t, "POST",
			fmt.Sprintf("%s/apps/%d/suspend", appsLink, app.ID), url.Values{"suspend": {"true"}})
		owner.MakeRequest(t, suspendReq, http.StatusSeeOther)
		MakeRequest(t, NewRequest(t, "GET", "/api/v1/user").AddTokenAuth(rawToken), http.StatusUnauthorized)

		suspendReq = NewRequestWithURLValues(t, "POST",
			fmt.Sprintf("%s/apps/%d/suspend", appsLink, app.ID), url.Values{"suspend": {"false"}})
		owner.MakeRequest(t, suspendReq, http.StatusSeeOther)
		MakeRequest(t, NewRequest(t, "GET", "/api/v1/user").AddTokenAuth(rawToken), http.StatusOK)

		// revoking it is scoped to the app's account
		revokeReq := NewRequestWithURLValues(t, "POST", tokensLink+"/delete",
			url.Values{"token_id": {strconv.FormatInt(token.ID, 10)}})
		owner.MakeRequest(t, revokeReq, http.StatusOK)
		unittest.AssertNotExistsBean(t, &auth_model.AccessToken{ID: token.ID})
	})

	t.Run("Provenance", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		body := MakeRequest(t, NewRequest(t, "GET", "/"+appName), http.StatusOK).Body.String()
		assert.Contains(t, body, "Operated by")
		assert.Contains(t, body, `href="/`+orgName+`"`)
		assert.Contains(t, body, "ships things")
		assert.Contains(t, body, "This is an automated account, not a person.")

		// a person's profile says none of it
		body = MakeRequest(t, NewRequest(t, "GET", "/user2"), http.StatusOK).Body.String()
		assert.NotContains(t, body, "Operated by")
		assert.NotContains(t, body, "This is an automated account, not a person.")
	})

	t.Run("SuspendAndResume", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		botUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: appName})
		app := unittest.AssertExistsAndLoadBean(t, &user_model.ForgenteApp{UserID: botUser.ID})
		appLink := fmt.Sprintf("%s/apps/%d/suspend", appsLink, app.ID)

		for _, suspend := range []bool{true, false, true} {
			req := NewRequestWithURLValues(t, "POST", appLink, url.Values{"suspend": {strconv.FormatBool(suspend)}})
			owner.MakeRequest(t, req, http.StatusSeeOther)

			suspended, err := user_model.IsForgenteAppSuspended(t.Context(), botUser.ID)
			require.NoError(t, err)
			assert.Equal(t, suspend, suspended)
		}

		// the page renders the suspended state and offers the way back
		body := owner.MakeRequest(t, NewRequest(t, "GET", appsLink), http.StatusOK).Body.String()
		assert.Contains(t, body, "Suspended")
		assert.Contains(t, body, "Resume All Apps")

		// the org-wide switch reaches the same app
		req := NewRequestWithURLValues(t, "POST", appsLink+"/apps/suspend", url.Values{"suspend": {"false"}})
		owner.MakeRequest(t, req, http.StatusSeeOther)
		suspended, err := user_model.IsForgenteAppSuspended(t.Context(), botUser.ID)
		require.NoError(t, err)
		assert.False(t, suspended)
	})

	t.Run("AnotherOrgCannotTouchIt", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		botUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: appName})
		app := unittest.AssertExistsAndLoadBean(t, &user_model.ForgenteApp{UserID: botUser.ID})

		// user1 owns private_org35; the app ID is reported as missing, not forbidden
		req := NewRequestWithURLValues(t, "POST",
			fmt.Sprintf("/org/private_org35/settings/applications/apps/%d/suspend", app.ID),
			url.Values{"suspend": {"true"}})
		loginUser(t, "user1").MakeRequest(t, req, http.StatusNotFound)

		suspended, err := user_model.IsForgenteAppSuspended(t.Context(), botUser.ID)
		require.NoError(t, err)
		assert.False(t, suspended)
	})

	t.Run("Delete", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		botUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: appName})
		app := unittest.AssertExistsAndLoadBean(t, &user_model.ForgenteApp{UserID: botUser.ID})

		// an app that has been granted access is a member of the organization,
		// which is what ordinary account deletion refuses to delete
		team := unittest.AssertExistsAndLoadBean(t, &organization.Team{OrgID: org.ID, LowerName: "team1"})
		require.NoError(t, org_service.AddTeamMember(t.Context(), team, botUser))
		unittest.AssertExistsAndLoadBean(t, &organization.OrgUser{UID: botUser.ID, OrgID: org.ID})

		req := NewRequestWithURLValues(t, "POST", appsLink+"/apps/delete", url.Values{"id": {strconv.FormatInt(app.ID, 10)}})
		owner.MakeRequest(t, req, http.StatusOK)

		unittest.AssertNotExistsBean(t, &user_model.User{ID: botUser.ID})
		unittest.AssertNotExistsBean(t, &user_model.ForgenteApp{ID: app.ID})
		unittest.AssertNotExistsBean(t, &organization.OrgUser{UID: botUser.ID, OrgID: org.ID})
	})
}
