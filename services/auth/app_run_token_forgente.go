// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"

	actions_model "forgente.com/models/actions"
	user_model "forgente.com/models/user"
	"forgente.com/modules/log"
)

// resolveForgenteAppRunToken resolves a short-lived app run token to the
// account it acts as, or returns nil when the token is not one.
//
// The token is only honoured while the task it was minted for is still
// running. Expiry alone would leave a cancelled or finished job holding a
// working credential for the rest of its hour, with nothing left to attribute
// it to; tying it to the task is what makes the lifetime a ceiling rather than
// the actual bound.
//
// Nothing here checks whether the app is suspended. That is deliberate: the
// kill switch is applied once in Group.Verify, after any method returns a user,
// precisely so a method added later cannot miss it.
func resolveForgenteAppRunToken(ctx context.Context, tokenSHA string, store DataStore) *user_model.User {
	runToken, err := user_model.FindForgenteAppRunTokenByToken(ctx, tokenSHA)
	if err != nil {
		return nil
	}

	task, err := actions_model.GetTaskByID(ctx, runToken.TaskID)
	if err != nil || task.Status != actions_model.StatusRunning {
		log.Trace("app run token refused: task[%d] is no longer running", runToken.TaskID)
		return nil
	}

	appUser, err := user_model.GetUserByID(ctx, runToken.UserID)
	if err != nil {
		log.Error("GetUserByID for app run token: %v", err)
		return nil
	}

	store.GetData()["IsApiToken"] = true
	store.GetData()["ApiTokenScope"] = runToken.Scope
	log.Trace("Valid app run token for app[%d] on task[%d]", runToken.AppID, runToken.TaskID)
	return appUser
}
