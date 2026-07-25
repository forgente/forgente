// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	auth_model "forgente.com/models/auth"
	"forgente.com/models/db"
	user_model "forgente.com/models/user"
	"forgente.com/modules/setting"
	"forgente.com/modules/util"
	"forgente.com/modules/web"
	user_setting "forgente.com/routers/web/user/setting"
	"forgente.com/services/context"
	"forgente.com/services/forms"
	user_service "forgente.com/services/user"
)

// orgAppView pairs an app with the account it acts as: the ownership row holds
// the administration state, the account holds the name and avatar the list shows.
type orgAppView struct {
	*user_model.ForgenteApp
	User   *user_model.User
	Tokens []*auth_model.AccessToken
}

// loadOrgAppsData fills in the apps section of the organization applications page.
func loadOrgAppsData(ctx *context.Context) {
	apps, err := user_model.ListForgenteAppsByOwnerID(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.ServerError("ListForgenteAppsByOwnerID", err)
		return
	}

	views := make([]*orgAppView, 0, len(apps))
	anyActive := false
	for _, app := range apps {
		botUser, err := user_model.GetUserByID(ctx, app.UserID)
		if err != nil {
			ctx.ServerError("GetUserByID", err)
			return
		}
		tokens, err := db.Find[auth_model.AccessToken](ctx, auth_model.ListAccessTokensOptions{UserID: app.UserID})
		if err != nil {
			ctx.ServerError("ListAccessTokens", err)
			return
		}

		views = append(views, &orgAppView{ForgenteApp: app, User: botUser, Tokens: tokens})
		anyActive = anyActive || !app.Suspended
	}

	ctx.Data["OrgApps"] = views
	// the org-wide switch offers whichever direction is useful: stop everything
	// while anything still runs, otherwise bring everything back
	ctx.Data["OrgAppsAnyActive"] = anyActive

	// the connect panel shows the host an MCP client points at
	ctx.Data["AppURL"] = setting.AppURL
	ctx.Data["AccessTokenScopePublicOnly"] = auth_model.AccessTokenScopePublicOnly
	// an app is never an administrator, so the admin scope is not on offer
	ctx.Data["TokenCategories"] = util.SliceRemoveAll(auth_model.GetAccessTokenCategories(), "admin")
}

func redirectToOrgApplications(ctx *context.Context) {
	ctx.Redirect(ctx.Org.OrgLink + "/settings/applications")
}

// getOrgAppFromPath resolves the {id} of an app route. It writes the response
// itself on failure, so callers only need to return.
func getOrgAppFromPath(ctx *context.Context) (*user_model.ForgenteApp, error) {
	app, _, err := user_service.GetOrgApp(ctx, ctx.Org.Organization, ctx.PathParamInt64("id"))
	if err != nil {
		if user_model.IsErrForgenteAppNotExist(err) {
			ctx.NotFound(err)
		} else {
			ctx.ServerError("GetOrgApp", err)
		}
		return nil, err
	}
	return app, nil
}

// AppsPost creates an organization-owned app.
func AppsPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateOrgAppForm)

	if ctx.HasError() {
		ctx.Flash.Error(ctx.GetErrMsg())
		redirectToOrgApplications(ctx)
		return
	}

	botUser, _, err := user_service.CreateOrgApp(ctx, ctx.Doer, ctx.Org.Organization, form.Name, form.Description)
	if err != nil {
		switch {
		case user_model.IsErrUserAlreadyExist(err):
			ctx.Flash.Error(ctx.Tr("form.username_been_taken"))
		case user_model.IsErrEmailAlreadyUsed(err):
			ctx.Flash.Error(ctx.Tr("form.email_been_used"))
		case user_service.IsErrForgenteAppLimitReached(err):
			ctx.Flash.Error(ctx.Tr("settings.app_limit_reached"))
		case db.IsErrNameReserved(err), db.IsErrNamePatternNotAllowed(err), db.IsErrNameCharsNotAllowed(err):
			ctx.Flash.Error(ctx.Tr("user.form.name_reserved", form.Name))
		default:
			ctx.ServerError("CreateOrgApp", err)
			return
		}
		redirectToOrgApplications(ctx)
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.create_app_success", botUser.Name))
	redirectToOrgApplications(ctx)
}

// AppSuspendPost flips the kill switch for a single app.
func AppSuspendPost(ctx *context.Context) {
	app, err := getOrgAppFromPath(ctx)
	if err != nil {
		return
	}

	suspended := ctx.FormBool("suspend")
	if err := user_model.SetForgenteAppSuspended(ctx, app.ID, suspended); err != nil {
		ctx.ServerError("SetForgenteAppSuspended", err)
		return
	}

	if suspended {
		ctx.Flash.Success(ctx.Tr("settings.app_suspend_success"))
	} else {
		ctx.Flash.Success(ctx.Tr("settings.app_resume_success"))
	}
	redirectToOrgApplications(ctx)
}

// AppsSuspendAllPost flips the kill switch for every app the organization owns.
func AppsSuspendAllPost(ctx *context.Context) {
	suspended := ctx.FormBool("suspend")
	count, err := user_service.SuspendOrgApps(ctx, ctx.Org.Organization, suspended)
	if err != nil {
		ctx.ServerError("SuspendOrgApps", err)
		return
	}

	if suspended {
		ctx.Flash.Success(ctx.Tr("settings.apps_suspend_all_success", count))
	} else {
		ctx.Flash.Success(ctx.Tr("settings.apps_resume_all_success", count))
	}
	redirectToOrgApplications(ctx)
}

// AppTokenPost mints an access token for one of the organization's apps.
//
// The token belongs to the app's account, so it carries the app's identity and
// whatever access the app has been granted; the scope chosen here only narrows
// which of that the token may use.
func AppTokenPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.NewAccessTokenForm)

	app, err := getOrgAppFromPath(ctx)
	if err != nil {
		return
	}

	scope, ok := user_setting.ParseAccessTokenScopeFromForm(ctx)
	if !ok {
		return
	}
	if !scope.HasPermissionScope() {
		ctx.Flash.Error(ctx.Tr("settings.at_least_one_permission"))
		redirectToOrgApplications(ctx)
		return
	}
	if ctx.HasError() {
		ctx.Flash.Error(ctx.GetErrMsg())
		redirectToOrgApplications(ctx)
		return
	}

	t := &auth_model.AccessToken{
		UID:   app.UserID,
		Name:  form.Name,
		Scope: scope,
	}

	exist, err := auth_model.AccessTokenByNameExists(ctx, t)
	if err != nil {
		ctx.ServerError("AccessTokenByNameExists", err)
		return
	}
	if exist {
		ctx.Flash.Error(ctx.Tr("settings.generate_token_name_duplicate", t.Name))
		redirectToOrgApplications(ctx)
		return
	}

	// an owner acting through a limited token must not mint a broader one for
	// an app it controls, which would launder the restriction away
	if t.Scope, ok = user_setting.RestrictAccessTokenScopeToRequest(ctx, t.Scope); !ok {
		return
	}

	if err := auth_model.NewAccessToken(ctx, t); err != nil {
		ctx.ServerError("NewAccessToken", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.generate_token_success"))
	// the only time the token is ever readable
	ctx.Flash.Info(t.Token)
	redirectToOrgApplications(ctx)
}

// AppTokenDelete revokes one of an app's access tokens.
func AppTokenDelete(ctx *context.Context) {
	app, err := getOrgAppFromPath(ctx)
	if err != nil {
		return
	}

	// scoping the delete to the app's account is what keeps one organization
	// from revoking another's tokens by guessing an ID
	if err := auth_model.DeleteAccessTokenByID(ctx, ctx.FormInt64("token_id"), app.UserID); err != nil {
		ctx.Flash.Error("DeleteAccessTokenByID: " + err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("settings.delete_token_success"))
	}

	ctx.JSONRedirect(ctx.Org.OrgLink + "/settings/applications")
}

// DeleteApp deletes an app and the account behind it.
func DeleteApp(ctx *context.Context) {
	app, botUser, err := user_service.GetOrgApp(ctx, ctx.Org.Organization, ctx.FormInt64("id"))
	if err != nil {
		if user_model.IsErrForgenteAppNotExist(err) {
			ctx.NotFound(err)
		} else {
			ctx.ServerError("GetOrgApp", err)
		}
		return
	}

	if err := user_service.DeleteOrgApp(ctx, ctx.Org.Organization, app, botUser); err != nil {
		// the app stays suspended after a failed delete, so reporting the reason
		// is enough — it cannot act in the meantime
		ctx.Flash.Error(ctx.Tr("settings.delete_app_failed", err))
	} else {
		ctx.Flash.Success(ctx.Tr("settings.delete_app_success", botUser.Name))
	}

	ctx.JSONRedirect(ctx.Org.OrgLink + "/settings/applications")
}
