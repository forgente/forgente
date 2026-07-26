// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user_test

import (
	"testing"

	auth_model "forgente.com/models/auth"
	"forgente.com/models/db"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// grants are inserted rather than fixtured: the shared fixtures are read by
// every package's tests, and a row added there for one case has broken
// unrelated suites before.
func insertGrant(t *testing.T, appID, repoID int64, scope auth_model.AccessTokenScope) *user_model.ForgenteAppRunGrant {
	t.Helper()
	grant := &user_model.ForgenteAppRunGrant{
		AppID:       appID,
		RepoID:      repoID,
		Scope:       scope,
		GrantedByID: 1,
	}
	require.NoError(t, db.Insert(t.Context(), grant))
	return grant
}

func TestForgenteAppRunGrantLookup(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	const appID, repoID, otherRepoID = 900, 910, 911

	t.Run("NoGrantIsNotExist", func(t *testing.T) {
		_, err := user_model.GetForgenteAppRunGrant(t.Context(), appID, repoID)
		assert.True(t, user_model.IsErrForgenteAppRunGrantNotExist(err), "got %v", err)
	})

	orgWide := insertGrant(t, appID, 0, auth_model.AccessTokenScopeReadIssue)

	t.Run("OrgWideGrantCoversAnyRepository", func(t *testing.T) {
		for _, id := range []int64{repoID, otherRepoID} {
			got, err := user_model.GetForgenteAppRunGrant(t.Context(), appID, id)
			require.NoError(t, err)
			assert.Equal(t, orgWide.ID, got.ID)
		}
	})

	specific := insertGrant(t, appID, repoID, auth_model.AccessTokenScopeReadRepository)

	t.Run("RepositoryGrantWinsOverOrgWide", func(t *testing.T) {
		// the narrower authorization has to win, or naming a repository could
		// only ever widen what an organization-wide grant already allowed
		got, err := user_model.GetForgenteAppRunGrant(t.Context(), appID, repoID)
		require.NoError(t, err)
		assert.Equal(t, specific.ID, got.ID)
		assert.Equal(t, auth_model.AccessTokenScopeReadRepository, got.Scope)

		// a repository without its own grant still falls back
		got, err = user_model.GetForgenteAppRunGrant(t.Context(), appID, otherRepoID)
		require.NoError(t, err)
		assert.Equal(t, orgWide.ID, got.ID)
	})

	t.Run("AnotherAppIsUnaffected", func(t *testing.T) {
		_, err := user_model.GetForgenteAppRunGrant(t.Context(), appID+1, repoID)
		assert.True(t, user_model.IsErrForgenteAppRunGrantNotExist(err), "got %v", err)
	})
}

func TestForgenteAppRunGrantDelete(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	const appID, otherAppID, repoID = 920, 921, 930
	grant := insertGrant(t, appID, repoID, auth_model.AccessTokenScopeReadIssue)

	t.Run("RevokingUnderTheWrongAppDoesNothing", func(t *testing.T) {
		// the id alone must not be enough, or a grant could be revoked from an
		// app whose settings the doer has no access to
		err := user_model.DeleteForgenteAppRunGrant(t.Context(), otherAppID, grant.ID)
		assert.True(t, user_model.IsErrForgenteAppRunGrantNotExist(err), "got %v", err)

		_, err = user_model.GetForgenteAppRunGrant(t.Context(), appID, repoID)
		assert.NoError(t, err, "the grant should still be there")
	})

	t.Run("Revoke", func(t *testing.T) {
		require.NoError(t, user_model.DeleteForgenteAppRunGrant(t.Context(), appID, grant.ID))
		_, err := user_model.GetForgenteAppRunGrant(t.Context(), appID, repoID)
		assert.True(t, user_model.IsErrForgenteAppRunGrantNotExist(err), "got %v", err)
	})
}

func TestForgenteAppRunGrantDeleteByApp(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	const appID, otherAppID = 940, 941
	insertGrant(t, appID, 0, auth_model.AccessTokenScopeReadIssue)
	insertGrant(t, appID, 950, auth_model.AccessTokenScopeReadIssue)
	insertGrant(t, otherAppID, 0, auth_model.AccessTokenScopeReadIssue)

	require.NoError(t, user_model.DeleteForgenteAppRunGrantsByAppID(t.Context(), appID))

	count, err := user_model.CountForgenteAppRunGrants(t.Context(), appID)
	require.NoError(t, err)
	assert.Zero(t, count, "deleting an app must take every grant made for it")

	count, err = user_model.CountForgenteAppRunGrants(t.Context(), otherAppID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "and must leave other apps alone")
}
