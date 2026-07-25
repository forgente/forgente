// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"errors"

	user_model "forgente.com/models/user"
	"forgente.com/modules/log"
)

// ErrForgenteAppSuspended is returned when a token belonging to a suspended app
// is used. Returning an error rather than a user leaves the request
// unauthenticated, so a suspended app is refused exactly like an unknown token.
var ErrForgenteAppSuspended = errors.New("forgente app is suspended")

// checkForgenteAppSuspended rejects an authenticated user that is a suspended app.
//
// Authentication resolves a credential straight to its user and never consults
// account state, so suspension cannot be expressed by deactivating the account —
// that would revoke nothing. This is called once from Group.Verify, which every
// auth method funnels through, so a method added later cannot miss it. Git over
// SSH does not pass through here and is gated separately in routers/private.
//
// It short-circuits on anything that cannot be an app: system users have
// negative IDs (Ghost, the Actions user) and humans are not bots, so an ordinary
// request costs no additional query.
func checkForgenteAppSuspended(ctx context.Context, u *user_model.User) (*user_model.User, error) {
	if u == nil || u.ID <= 0 || !u.IsTypeBot() {
		return u, nil
	}

	suspended, err := user_model.IsForgenteAppSuspended(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	if suspended {
		log.Trace("Authentication refused: app[%d] %q is suspended", u.ID, u.Name)
		return nil, ErrForgenteAppSuspended
	}
	return u, nil
}
