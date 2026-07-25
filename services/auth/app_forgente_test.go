// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth

import (
	"errors"
	"testing"

	"forgente.com/models/db"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckForgenteAppSuspended(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx := t.Context()

	const activeAppUserID, suspendedAppUserID, unownedBotID = 1001, 1002, 1003

	require.NoError(t, db.Insert(ctx, &user_model.ForgenteApp{
		UserID: activeAppUserID, OwnerID: 3, CreatorID: 2,
	}))
	require.NoError(t, db.Insert(ctx, &user_model.ForgenteApp{
		UserID: suspendedAppUserID, OwnerID: 3, CreatorID: 2, Suspended: true,
	}))

	bot := func(id int64) *user_model.User {
		return &user_model.User{ID: id, Type: user_model.UserTypeBot}
	}

	lookupErr := errors.New("lookup failed")

	cases := []struct {
		name    string
		user    *user_model.User
		inErr   error
		wantErr error
		wantNil bool
	}{
		{name: "nil user passes through", user: nil, wantNil: true},
		{name: "preceding error is preserved", user: bot(suspendedAppUserID), inErr: lookupErr, wantErr: lookupErr},
		{name: "individual is never an app", user: &user_model.User{ID: 2, Type: user_model.UserTypeIndividual}},
		{name: "system bot is never an app", user: bot(user_model.ActionsUserID)},
		{name: "bot without an app row is allowed", user: bot(unownedBotID)},
		{name: "active app is allowed", user: bot(activeAppUserID)},
		{name: "suspended app is refused", user: bot(suspendedAppUserID), wantErr: ErrForgenteAppSuspended, wantNil: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			u, err := checkForgenteAppSuspended(ctx, c.user, c.inErr)
			if c.wantErr != nil {
				assert.ErrorIs(t, err, c.wantErr)
			} else {
				assert.NoError(t, err)
			}
			if c.wantNil {
				assert.Nil(t, u)
			} else {
				assert.Equal(t, c.user, u)
			}
		})
	}
}

func TestForgenteAppSuspendAllByOwner(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx := t.Context()

	const ownerID, otherOwnerID = 3, 6
	require.NoError(t, db.Insert(ctx, &user_model.ForgenteApp{UserID: 2001, OwnerID: ownerID, CreatorID: 2}))
	require.NoError(t, db.Insert(ctx, &user_model.ForgenteApp{UserID: 2002, OwnerID: ownerID, CreatorID: 2}))
	require.NoError(t, db.Insert(ctx, &user_model.ForgenteApp{UserID: 2003, OwnerID: otherOwnerID, CreatorID: 2}))

	count, err := user_model.SetForgenteAppsSuspendedByOwnerID(ctx, ownerID, true)
	require.NoError(t, err)
	assert.EqualValues(t, 2, count)

	// the kill switch stops at the organization boundary
	suspended, err := user_model.IsForgenteAppSuspended(ctx, 2003)
	require.NoError(t, err)
	assert.False(t, suspended)

	for _, id := range []int64{2001, 2002} {
		suspended, err := user_model.IsForgenteAppSuspended(ctx, id)
		require.NoError(t, err)
		assert.True(t, suspended)
	}
}
