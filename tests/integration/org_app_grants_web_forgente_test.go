// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	auth_model "forgente.com/models/auth"
	"forgente.com/models/organization"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"
	user_service "forgente.com/services/user"
	"forgente.com/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrgAppGrantsWeb(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org3"})
	appsLink := "/org/org3/settings/applications"

	_, app, err := user_service.CreateOrgApp(t.Context(), doer, org, "grantuibot", "runs the work")
	require.NoError(t, err)
	grantsLink := fmt.Sprintf("%s/apps/%d/grants", appsLink, app.ID)

	owner := loginUser(t, doer.Name)

	t.Run("TheSectionIsOnThePage", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		body := owner.MakeRequest(t, NewRequest(t, "GET", appsLink), http.StatusOK).Body.String()
		assert.Contains(t, body, "Repository access for Actions")
		// the escalation has to be stated where the grant is made, not only in
		// the design document
		assert.Contains(t, body, "can act as")
	})

	t.Run("GrantAllRepositories", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", grantsLink, map[string]string{
			"repo_name":   "",
			"scope-issue": "read:issue",
		})
		owner.MakeRequest(t, req, http.StatusSeeOther)

		grant := unittest.AssertExistsAndLoadBean(t, &user_model.ForgenteAppRunGrant{AppID: app.ID, RepoID: 0})
		assert.Equal(t, auth_model.AccessTokenScopeReadIssue, grant.Scope)
		assert.Equal(t, doer.ID, grant.GrantedByID)
	})

	t.Run("GrantOneRepositoryNarrows", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", grantsLink, map[string]string{
			"repo_name":        "repo3",
			"scope-repository": "read:repository",
		})
		owner.MakeRequest(t, req, http.StatusSeeOther)

		grant := unittest.AssertExistsAndLoadBean(t, &user_model.ForgenteAppRunGrant{AppID: app.ID, RepoID: 3})
		assert.Equal(t, auth_model.AccessTokenScopeReadRepository, grant.Scope)

		body := owner.MakeRequest(t, NewRequest(t, "GET", appsLink), http.StatusOK).Body.String()
		assert.Contains(t, body, "org3/repo3")
	})

	t.Run("AnUnknownRepositoryIsRejected", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", grantsLink, map[string]string{
			"repo_name":   "no-such-repo",
			"scope-issue": "read:issue",
		})
		owner.MakeRequest(t, req, http.StatusSeeOther)

		count, err := user_model.CountForgenteAppRunGrants(t.Context(), app.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count, "the rejected grant must not have been created")
	})

	t.Run("AScopeWithNoPermissionIsRejected", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", grantsLink, map[string]string{"repo_name": "repo5"})
		owner.MakeRequest(t, req, http.StatusSeeOther)

		count, err := user_model.CountForgenteAppRunGrants(t.Context(), app.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("OnlyOwnersMayGrant", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// user4 is in the organization but does not own it
		req := NewRequestWithValues(t, "POST", grantsLink, map[string]string{
			"repo_name":   "repo3",
			"scope-issue": "write:issue",
		})
		loginUser(t, "user4").MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("Revoke", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		grant := unittest.AssertExistsAndLoadBean(t, &user_model.ForgenteAppRunGrant{AppID: app.ID, RepoID: 3})
		req := NewRequestWithValues(t, "POST", grantsLink+"/delete", map[string]string{
			"id": strconv.FormatInt(grant.ID, 10),
		})
		owner.MakeRequest(t, req, http.StatusSeeOther)

		unittest.AssertNotExistsBean(t, &user_model.ForgenteAppRunGrant{AppID: app.ID, RepoID: 3})
	})
}
