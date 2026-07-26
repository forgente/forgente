// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
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

func TestOrgAppRunGrants(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org3"})
	// repo3 belongs to org3; repo1 belongs to user2, so it is foreign to the app
	const ownRepoID, foreignRepoID = 3, 1

	_, app, err := user_service.CreateOrgApp(t.Context(), doer, org, "grantbot", "runs the work")
	require.NoError(t, err)

	t.Run("AScopeWithNoPermissionIsRefused", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// "unspecified" must not read as "everything" — the OAuth2 grant default
		// corrected during L1 was exactly this mistake
		_, err := user_service.GrantAppToRepoRuns(t.Context(), doer, org, app.ID, ownRepoID, "")
		assert.True(t, user_service.IsErrForgenteAppRunGrantScope(err), "got %v", err)
	})

	t.Run("AForeignRepositoryIsRefused", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// otherwise an organization could hand its app's identity to whoever can
		// land a workflow in somebody else's repository
		_, err := user_service.GrantAppToRepoRuns(t.Context(), doer, org, app.ID, foreignRepoID, auth_model.AccessTokenScopeReadIssue)
		require.Error(t, err)
		assert.ErrorContains(t, err, "not owned by organization")
	})

	t.Run("Grant", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		grant, err := user_service.GrantAppToRepoRuns(t.Context(), doer, org, app.ID, ownRepoID, auth_model.AccessTokenScopeReadIssue)
		require.NoError(t, err)
		assert.Equal(t, auth_model.AccessTokenScopeReadIssue, grant.Scope)
		assert.Equal(t, doer.ID, grant.GrantedByID)
	})

	t.Run("GrantingAgainReplacesTheScope", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		before, err := user_model.GetForgenteAppRunGrant(t.Context(), app.ID, ownRepoID)
		require.NoError(t, err)

		after, err := user_service.GrantAppToRepoRuns(t.Context(), doer, org, app.ID, ownRepoID, auth_model.AccessTokenScopeWriteIssue)
		require.NoError(t, err)

		// the same authorization changed, not a second one added
		assert.Equal(t, before.ID, after.ID)
		assert.Equal(t, auth_model.AccessTokenScopeWriteIssue, after.Scope)

		count, err := user_model.CountForgenteAppRunGrants(t.Context(), app.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("AnAppFromAnotherOrganizationIsNotFound", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		otherOrg := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org6"})
		_, err := user_service.GrantAppToRepoRuns(t.Context(), doer, otherOrg, app.ID, 0, auth_model.AccessTokenScopeReadIssue)
		assert.True(t, user_model.IsErrForgenteAppNotExist(err), "got %v", err)
	})

	t.Run("Revoke", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		grant, err := user_model.GetForgenteAppRunGrant(t.Context(), app.ID, ownRepoID)
		require.NoError(t, err)

		require.NoError(t, user_service.RevokeAppRunGrant(t.Context(), org, app.ID, grant.ID))

		_, err = user_model.GetForgenteAppRunGrant(t.Context(), app.ID, ownRepoID)
		assert.True(t, user_model.IsErrForgenteAppRunGrantNotExist(err), "got %v", err)
	})

	t.Run("DeletingTheAppTakesItsGrants", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		_, err := user_service.GrantAppToRepoRuns(t.Context(), doer, org, app.ID, 0, auth_model.AccessTokenScopeReadIssue)
		require.NoError(t, err)

		botUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: app.UserID})
		require.NoError(t, user_service.DeleteOrgApp(t.Context(), org, app, botUser))

		count, err := user_model.CountForgenteAppRunGrants(t.Context(), app.ID)
		require.NoError(t, err)
		assert.Zero(t, count, "a grant left behind would authorize the next app to reuse this id")
	})
}
