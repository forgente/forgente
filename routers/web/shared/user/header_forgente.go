// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	user_model "forgente.com/models/user"
	"forgente.com/modules/log"
	"forgente.com/services/context"
)

// prepareContextForForgenteApp adds the provenance shown on an app's profile:
// which organization operates it, and whether it is currently suspended.
//
// The profile is where provenance belongs in full — activity elsewhere only
// carries the bot label, and the name beside it links here. A missing ownership
// row is normal: built-in accounts like the Actions user are bots that no
// organization owns, and they keep the label without an operator.
func prepareContextForForgenteApp(ctx *context.Context) {
	if !ctx.ContextUser.IsTypeBot() {
		return
	}

	app, err := user_model.GetForgenteAppByUserID(ctx, ctx.ContextUser.ID)
	if err != nil {
		if !user_model.IsErrForgenteAppNotExist(err) {
			log.Error("GetForgenteAppByUserID: %v", err)
		}
		return
	}

	owner, err := user_model.GetUserByID(ctx, app.OwnerID)
	if err != nil {
		log.Error("GetUserByID: %v", err)
		return
	}

	ctx.Data["ForgenteAppOwner"] = owner
	ctx.Data["ForgenteAppSuspended"] = app.Suspended
	if app.Description != "" {
		ctx.Data["ForgenteAppDescription"] = app.Description
	}
}
