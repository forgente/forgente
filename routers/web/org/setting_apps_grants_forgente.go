// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"strings"

	repo_model "forgente.com/models/repo"
	user_model "forgente.com/models/user"
	"forgente.com/modules/web"
	user_setting "forgente.com/routers/web/user/setting"
	"forgente.com/services/context"
	"forgente.com/services/forms"
	user_service "forgente.com/services/user"
)

// grantView pairs a run grant with the repository it names, so the list can show
// a name rather than an id. Repo is nil for an organization-wide grant.
type grantView struct {
	*user_model.ForgenteAppRunGrant
	Repo *repo_model.Repository
}

// loadAppRunGrants resolves an app's run grants for display.
func loadAppRunGrants(ctx *context.Context, appID int64) ([]*grantView, error) {
	grants, err := user_model.ListForgenteAppRunGrants(ctx, appID)
	if err != nil {
		return nil, err
	}

	views := make([]*grantView, 0, len(grants))
	for _, grant := range grants {
		view := &grantView{ForgenteAppRunGrant: grant}
		if grant.RepoID != 0 {
			// a repository deleted out from under a grant should not blank the
			// page; the grant goes with it, so this is only a race
			if repo, err := repo_model.GetRepositoryByID(ctx, grant.RepoID); err == nil {
				view.Repo = repo
			}
		}
		views = append(views, view)
	}
	return views, nil
}

// AppGrantPost authorizes a repository's Actions runs to act as the app, or
// every repository the organization owns when no repository is named.
//
// This is an escalation and the page says so: anyone who can land a workflow in
// the repository can then act as the app, up to the scope chosen here.
func AppGrantPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.GrantAppRunForm)

	app, err := getOrgAppFromPath(ctx)
	if err != nil {
		return
	}

	scope, ok := user_setting.ParseAccessTokenScopeFromForm(ctx)
	if !ok {
		return
	}
	if ctx.HasError() {
		ctx.Flash.Error(ctx.GetErrMsg())
		redirectToOrgApplications(ctx)
		return
	}

	var repoID int64
	if form.RepoName != "" {
		repo, err := repo_model.GetRepositoryByName(ctx, ctx.Org.Organization.ID, form.RepoName)
		if err != nil {
			if repo_model.IsErrRepoNotExist(err) {
				ctx.Flash.Error(ctx.Tr("org.settings.app_grant_repo_not_found", form.RepoName))
				redirectToOrgApplications(ctx)
				return
			}
			ctx.ServerError("GetRepositoryByName", err)
			return
		}
		repoID = repo.ID
	}

	// the same guard the token path uses: an owner acting through a limited
	// token must not grant a broader one, which would launder the restriction
	if scope, ok = user_setting.RestrictAccessTokenScopeToRequest(ctx, scope); !ok {
		return
	}

	runnerLabel := strings.TrimSpace(form.RunnerLabel)

	if _, err := user_service.GrantAppToRepoRuns(ctx, ctx.Doer, ctx.Org.Organization, app.ID, repoID, scope, runnerLabel); err != nil {
		if user_service.IsErrForgenteAppRunGrantScope(err) {
			ctx.Flash.Error(ctx.Tr("settings.at_least_one_permission"))
		} else {
			ctx.ServerError("GrantAppToRepoRuns", err)
			return
		}
		redirectToOrgApplications(ctx)
		return
	}

	ctx.Flash.Success(ctx.Tr("org.settings.app_grant_success"))
	redirectToOrgApplications(ctx)
}

// AppGrantDelete withdraws one grant. A run already holding a minted token keeps
// it until its job ends; suspending the app is what stops one in flight.
func AppGrantDelete(ctx *context.Context) {
	app, err := getOrgAppFromPath(ctx)
	if err != nil {
		return
	}

	if err := user_service.RevokeAppRunGrant(ctx, ctx.Org.Organization, app.ID, ctx.FormInt64("id")); err != nil {
		if user_model.IsErrForgenteAppRunGrantNotExist(err) {
			ctx.NotFound(err)
			return
		}
		ctx.ServerError("RevokeAppRunGrant", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("org.settings.app_grant_revoked"))
	redirectToOrgApplications(ctx)
}
