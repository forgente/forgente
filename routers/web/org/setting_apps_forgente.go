// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"forgente.com/models/db"
	user_model "forgente.com/models/user"
	"forgente.com/modules/web"
	"forgente.com/services/context"
	"forgente.com/services/forms"
	user_service "forgente.com/services/user"
)

// orgAppView pairs an app with the account it acts as: the ownership row holds
// the administration state, the account holds the name and avatar the list shows.
type orgAppView struct {
	*user_model.ForgenteApp
	User *user_model.User
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
		views = append(views, &orgAppView{ForgenteApp: app, User: botUser})
		anyActive = anyActive || !app.Suspended
	}

	ctx.Data["OrgApps"] = views
	// the org-wide switch offers whichever direction is useful: stop everything
	// while anything still runs, otherwise bring everything back
	ctx.Data["OrgAppsAnyActive"] = anyActive
}

func redirectToOrgApplications(ctx *context.Context) {
	ctx.Redirect(ctx.Org.OrgLink + "/settings/applications")
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
	app, _, err := user_service.GetOrgApp(ctx, ctx.Org.Organization, ctx.PathParamInt64("id"))
	if err != nil {
		if user_model.IsErrForgenteAppNotExist(err) {
			ctx.NotFound(err)
		} else {
			ctx.ServerError("GetOrgApp", err)
		}
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
