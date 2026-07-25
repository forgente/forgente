// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemUser(t *testing.T) {
	uid, u, err := GetPossibleUserByID(t.Context(), -1)
	require.NoError(t, err)
	assert.Equal(t, int64(-1), uid)
	assert.Equal(t, "Ghost", u.Name)
	assert.Equal(t, "ghost", u.LowerName)
	assert.True(t, u.IsGhost())

	u = GetSystemUserByName("gHost")
	require.NotNil(t, u)
	assert.Equal(t, "Ghost", u.Name)

	uid, u, err = GetPossibleUserByID(t.Context(), -2)
	require.NoError(t, err)
	assert.Equal(t, int64(-2), uid)
	assert.Equal(t, "forgente-actions", u.Name)
	assert.Equal(t, "forgente-actions", u.LowerName)
	assert.True(t, u.IsGiteaActions())

	u = GetSystemUserByName("Forgente-actionS")
	require.NotNil(t, u)
	assert.Equal(t, "Forgente Actions", u.FullName)

	// the pre-hard-fork name still resolves, so old links and payloads keep working
	u = GetSystemUserByName("Gitea-actionS")
	require.NotNil(t, u)
	assert.Equal(t, "forgente-actions", u.Name)

	uid, u, err = GetPossibleUserByID(t.Context(), 999999)
	require.NoError(t, err)
	assert.Equal(t, int64(-1), uid)
	assert.Equal(t, "Ghost", u.Name)
}
