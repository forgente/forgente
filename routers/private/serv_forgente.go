// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package private

import (
	"net/http"

	user_model "forgente.com/models/user"
	"forgente.com/services/context"
)

// rejectSuspendedForgenteApp stops a suspended app from acting over SSH.
//
// An app can hold SSH keys as well as tokens, and a key keeps working after the
// app's tokens start being refused, so the kill switch has to be enforced on
// this path as well as in the token auth paths. Anything that cannot be an app
// costs no query: system users have negative IDs and humans are not bots.
//
// It reports true once it has written the response, so callers only need to return.
func rejectSuspendedForgenteApp(ctx *context.PrivateContext, u *user_model.User) bool {
	if u == nil || u.ID <= 0 || !u.IsTypeBot() {
		return false
	}

	suspended, err := user_model.IsForgenteAppSuspended(ctx, u.ID)
	if err != nil {
		ctx.PrivateInternalErrorf("Unable to check app suspension for user %d: %v", u.ID, err)
		return true
	}
	if suspended {
		ctx.PrivateUserErrorf(http.StatusForbidden, "This app is suspended.")
		return true
	}
	return false
}
