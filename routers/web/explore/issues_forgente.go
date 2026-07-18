// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package explore

import (
	"net/http"
	"strings"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	issue_indexer "gitea.dev/modules/indexer/issues"
	"gitea.dev/modules/optional"
	"gitea.dev/modules/setting"
	"gitea.dev/modules/templates"
	"gitea.dev/services/context"
	issue_service "gitea.dev/services/issue"
	pull_service "gitea.dev/services/pull"
)

// tplExploreIssues explore issues page template
const tplExploreIssues templates.TplName = "explore/issues_forgente"

// ForgenteIssues renders a cross-repository issue/pull search page covering all public repos,
// filling the gap where Gitea's global search only exists on the signed-in user's dashboard.
func ForgenteIssues(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("explore_title")
	ctx.Data["PageIsExplore"] = true
	ctx.Data["PageIsExploreIssues"] = true
	ctx.Data["UsersPageIsDisabled"] = setting.Service.Explore.DisableUsersPage
	ctx.Data["OrganizationsPageIsDisabled"] = setting.Service.Explore.DisableOrganizationsPage
	ctx.Data["IsRepoIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled

	keyword := strings.TrimSpace(ctx.FormString("q"))
	ctx.Data["Keyword"] = keyword

	viewType := ctx.FormString("type")
	if viewType != "pulls" {
		viewType = "issues"
	}
	ctx.Data["ViewType"] = viewType
	isPullList := viewType == "pulls"
	ctx.Data["PageIsPulls"] = isPullList

	isShowClosed := ctx.FormString("state") == "closed"
	if isShowClosed {
		ctx.Data["State"] = "closed"
	} else {
		ctx.Data["State"] = "open"
	}
	ctx.Data["IsShowClosed"] = isShowClosed

	page := max(ctx.FormInt("page"), 1)

	issueIDs, total, err := issue_indexer.SearchIssues(ctx, &issue_indexer.SearchOptions{
		Keyword:    keyword,
		AllPublic:  true, // search across all public repos, not just the ones the doer belongs to
		IsPull:     optional.Some(isPullList),
		IsClosed:   optional.Some(isShowClosed),
		IsArchived: optional.Some(false), // same as the dashboard: archived repos are noise here
		SortBy:     issue_indexer.SortByCreatedDesc,
		Paginator: &db.ListOptions{
			Page:     page,
			PageSize: setting.UI.IssuePagingNum,
		},
	})
	if err != nil {
		if issue_indexer.IsAvailable(ctx) {
			ctx.ServerError("SearchIssues", err)
			return
		}
		ctx.Data["IssueIndexerUnavailable"] = true
	} else {
		ctx.Data["IssueIndexerUnavailable"] = !issue_indexer.IsAvailable(ctx)
	}

	issues, err := issues_model.GetIssuesByIDs(ctx, issueIDs, true)
	if err != nil {
		ctx.ServerError("GetIssuesByIDs", err)
		return
	}

	if err := issues.LoadAttributes(ctx); err != nil {
		ctx.ServerError("issues.LoadAttributes", err)
		return
	}

	// The db indexer backend only checks repository.is_private, while the other backends
	// also require a publicly visible owner. Re-check both here so limited/private-visibility
	// owners never leak to anonymous visitors regardless of the backend in use.
	filtered := make(issues_model.IssueList, 0, len(issues))
	for _, issue := range issues {
		if issue.Repo == nil || issue.Repo.IsPrivate {
			continue
		}
		if err := issue.Repo.LoadOwner(ctx); err != nil {
			ctx.ServerError("LoadOwner", err)
			return
		}
		visible := issue.Repo.Owner.Visibility.IsPublic()
		if !visible && ctx.Doer != nil {
			visible = issue.Repo.Owner.Visibility.IsLimited()
		}
		if visible {
			filtered = append(filtered, issue)
		}
	}
	issues = filtered
	ctx.Data["Issues"] = issues

	commitStatuses, lastStatus, err := pull_service.GetIssuesAllCommitStatus(ctx, issues)
	if err != nil {
		ctx.ServerError("GetIssuesAllCommitStatus", err)
		return
	}
	ctx.Data["CommitStatuses"] = commitStatuses
	ctx.Data["CommitLastStatus"] = lastStatus

	// RepoLink is intentionally left unset: issues span multiple repos, so
	// shared/issuelist.tmpl falls back to each issue's own .Repo.Link.
	ctx.Data["IssueRefEndNames"], ctx.Data["IssueRefURLs"] = issue_service.GetRefEndNamesAndURLs(issues, "")

	approvalCounts, err := issues.GetApprovalCounts(ctx)
	if err != nil {
		ctx.ServerError("GetApprovalCounts", err)
		return
	}
	ctx.Data["ApprovalCounts"] = func(issueID int64, typ string) int64 {
		counts, ok := approvalCounts[issueID]
		if !ok || len(counts) == 0 {
			return 0
		}
		reviewTyp := issues_model.ReviewTypeApprove
		switch typ {
		case "reject":
			reviewTyp = issues_model.ReviewTypeReject
		case "waiting":
			reviewTyp = issues_model.ReviewTypeRequest
		}
		for _, count := range counts {
			if count.Type == reviewTyp {
				return count.Count
			}
		}
		return 0
	}

	ctx.Data["SearchModes"] = issue_indexer.SupportedSearchModes()
	ctx.Data["SelectedSearchMode"] = ctx.FormTrim("search_mode")

	pager := context.NewPagination(total, setting.UI.IssuePagingNum, page, 5)
	pager.AddParamFromRequest(ctx.Req)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplExploreIssues)
}
