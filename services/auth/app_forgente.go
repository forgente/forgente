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

// checkForgenteAppSuspended rejects a token holder that is a suspended app.
//
// Token authentication resolves a token straight to its user and never consults
// account state, so suspension cannot be expressed by deactivating the account —
// that would revoke nothing. Every path that turns a token into a user therefore
// passes the result through here.
//
// It takes the preceding lookup's error so call sites stay a single extra line,
// and it short-circuits on anything that cannot be an app: system users have
// negative IDs (Ghost, the Actions user) and humans are not bots, so an ordinary
// API request costs no additional query.
func checkForgenteAppSuspended(ctx context.Context, u *user_model.User, err error) (*user_model.User, error) {
	if err != nil || u == nil || u.ID <= 0 || !u.IsTypeBot() {
		return u, err
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
