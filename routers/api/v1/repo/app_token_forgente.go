// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	user_model "forgente.com/models/user"
	api "forgente.com/modules/structs"
	"forgente.com/modules/web"
	"forgente.com/services/context"
	user_service "forgente.com/services/user"
)

// MintAppRunToken issues a short-lived token letting the calling Actions run
// act as an organization-owned app.
func MintAppRunToken(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/actions/app-token repository repoMintAppRunToken
	// ---
	// summary: Exchange a running job's token for a short-lived app token
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/MintAppRunTokenOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/AppRunToken"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	// Only a running job may ask. The task id rides on the account the job
	// token authenticates as, so there is nothing for the caller to assert and
	// nothing to spoof — a request without it is simply not from a run.
	taskID, ok := user_model.GetActionsUserTaskID(ctx.Doer)
	if !ok {
		ctx.APIError(http.StatusForbidden, "only a running Actions job may request an app token")
		return
	}

	form := web.GetForm(ctx).(*api.MintAppRunTokenOption)

	token, err := user_service.MintAppRunToken(ctx, taskID, ctx.Repo.Repository.ID, form.App)
	if err != nil {
		switch {
		case user_service.IsErrForgenteAppRunTokenRefused(err):
			ctx.APIError(http.StatusForbidden, err.Error())
		case user_model.IsErrForgenteAppNotExist(err) || user_model.IsErrUserNotExist(err):
			// an app that does not exist and one that exists but was never
			// granted are reported the same way, so names cannot be probed
			ctx.APIError(http.StatusForbidden, "run may not act as this app")
		default:
			ctx.APIErrorInternal(err)
		}
		return
	}

	ctx.JSON(http.StatusCreated, &api.AppRunToken{
		Token:     token.Token,
		App:       form.App,
		Scope:     string(token.Scope),
		ExpiresAt: token.ExpiresUnix.AsTime(),
	})
}
