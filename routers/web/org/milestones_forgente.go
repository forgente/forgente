// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"net/http"

	"forgente.com/models/db"
	issues_model "forgente.com/models/issues"
	repo_model "forgente.com/models/repo"
	"forgente.com/models/unit"
	"forgente.com/modules/optional"
	"forgente.com/modules/setting"
	"forgente.com/modules/templates"
	shared_user "forgente.com/routers/web/shared/user"
	"forgente.com/services/context"
)

const tplForgenteMilestonesOverview templates.TplName = "org/forgente_milestones_overview"

// ForgenteMilestonesOverview renders a GitLab-style "group milestone" roll-up:
// milestones sharing the same name across the org's repositories are combined
// into a single card with summed progress. Milestones themselves stay
// repo-scoped (see issues_model.Milestone.RepoID); this is a read-only,
// presentational view computed over the same data as the existing
// per-repo-row org milestones dashboard (see user.Milestones).
func ForgenteMilestonesOverview(ctx *context.Context) {
	// same availability rule as user.Milestones: the page is meaningless when
	// both issues and pull requests are globally disabled
	if unit.TypeIssues.UnitGlobalDisabled() && unit.TypePullRequests.UnitGlobalDisabled() {
		ctx.NotFound(nil)
		return
	}

	if _, err := shared_user.RenderUserOrgHeader(ctx); err != nil {
		ctx.ServerError("RenderUserOrgHeader", err)
		return
	}

	org := ctx.Org.Organization
	ctx.Data["Title"] = ctx.Tr("milestones")
	ctx.Data["PageIsMilestonesDashboard"] = true

	isShowClosed := ctx.FormString("state") == "closed"

	repoOpts := repo_model.SearchRepoOptions{
		Actor:         ctx.Doer,
		OwnerID:       org.ID,
		Private:       true,
		AllPublic:     false,
		AllLimited:    false,
		Archived:      optional.Some(false),
		HasMilestones: optional.Some(true),
	}
	repoCond := repo_model.SearchRepositoryCondition(repoOpts)

	milestones, err := db.Find[issues_model.Milestone](ctx, issues_model.FindMilestoneOptions{
		ListOptions: db.ListOptionsAll,
		RepoCond:    repoCond,
		IsClosed:    optional.Some(isShowClosed),
	})
	if err != nil {
		ctx.ServerError("FindMilestones", err)
		return
	}

	showRepos, _, err := repo_model.SearchRepositoryByCondition(ctx, repoOpts, repoCond, false)
	if err != nil {
		ctx.ServerError("SearchRepositoryByCondition", err)
		return
	}
	repoByID := make(map[int64]*repo_model.Repository, len(showRepos))
	for _, repo := range showRepos {
		repoByID[repo.ID] = repo
	}

	for i := 0; i < len(milestones); {
		milestones[i].Repo = repoByID[milestones[i].RepoID]
		if milestones[i].Repo == nil {
			milestones = append(milestones[:i], milestones[i+1:]...)
			continue
		}
		i++
	}

	milestoneStats, err := issues_model.GetMilestonesStatsByRepoCondAndKw(ctx, repoCond, "")
	if err != nil {
		ctx.ServerError("GetMilestonesStatsByRepoCondAndKw", err)
		return
	}

	ctx.Data["MilestoneGroups"] = issues_model.ForgenteGroupMilestonesByName(milestones)
	ctx.Data["MilestoneStats"] = milestoneStats
	ctx.Data["IsShowClosed"] = isShowClosed
	// read by base/head_navbar via base/head
	ctx.Data["ShowMilestonesDashboardPage"] = setting.Service.ShowMilestonesDashboardPage

	ctx.HTML(http.StatusOK, tplForgenteMilestonesOverview)
}
