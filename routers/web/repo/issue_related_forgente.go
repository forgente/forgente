// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"strconv"
	"strings"

	issues_model "forgente.com/models/issues"
	"forgente.com/services/context"
)

// ForgenteAddRelatedIssue relates another issue of the same repository to this issue.
func ForgenteAddRelatedIssue(ctx *context.Context) {
	issue := GetActionIssue(ctx)
	if ctx.Written() {
		return
	}
	if issue.IsPull {
		ctx.NotFound(nil)
		return
	}
	if !ctx.Repo.Permission.CanWriteIssuesOrPulls(false) {
		ctx.HTTPError(http.StatusForbidden, "CanWriteIssues")
		return
	}

	defer ctx.Redirect(issue.Link())

	ref := strings.TrimPrefix(strings.TrimSpace(ctx.FormString("related_ref")), "#")
	index, err := strconv.ParseInt(ref, 10, 64)
	if err != nil || index <= 0 {
		ctx.Flash.Error(ctx.Tr("repo.issues.related.add_error_invalid"))
		return
	}

	related, err := issues_model.GetIssueByIndex(ctx, ctx.Repo.Repository.ID, index)
	if err != nil {
		ctx.Flash.Error(ctx.Tr("repo.issues.related.add_error_not_exist"))
		return
	}
	if related.ID == issue.ID {
		ctx.Flash.Error(ctx.Tr("repo.issues.related.add_error_same"))
		return
	}

	if err := issues_model.CreateForgenteIssueRelation(ctx, ctx.Doer, issue, related); err != nil {
		if issues_model.IsErrForgenteRelationExists(err) {
			ctx.Flash.Error(ctx.Tr("repo.issues.related.add_error_exists"))
			return
		}
		ctx.ServerError("CreateForgenteIssueRelation", err)
	}
}

// ForgenteRemoveRelatedIssue removes a relation between this issue and another one.
func ForgenteRemoveRelatedIssue(ctx *context.Context) {
	issue := GetActionIssue(ctx)
	if ctx.Written() {
		return
	}
	if issue.IsPull {
		ctx.NotFound(nil)
		return
	}
	if !ctx.Repo.Permission.CanWriteIssuesOrPulls(false) {
		ctx.HTTPError(http.StatusForbidden, "CanWriteIssues")
		return
	}

	related, err := issues_model.GetIssueByID(ctx, ctx.FormInt64("related_id"))
	if err != nil {
		ctx.NotFoundOrServerError("GetIssueByID", issues_model.IsErrIssueNotExist, err)
		return
	}
	if related.RepoID != issue.RepoID {
		ctx.NotFound(nil)
		return
	}

	if err := issues_model.RemoveForgenteIssueRelation(ctx, ctx.Doer, issue, related); err != nil {
		if issues_model.IsErrForgenteRelationNotExists(err) {
			ctx.Flash.Error(ctx.Tr("repo.issues.related.remove_error_not_exist"))
		} else {
			ctx.ServerError("RemoveForgenteIssueRelation", err)
			return
		}
	}
	ctx.Redirect(issue.Link())
}

// forgentePrepareIssueViewSidebarRelated loads the related-issues sidebar data.
func forgentePrepareIssueViewSidebarRelated(ctx *context.Context, issue *issues_model.Issue) {
	if issue.IsPull {
		return
	}
	related, err := issues_model.GetForgenteRelatedIssues(ctx, issue)
	if err != nil {
		ctx.ServerError("GetForgenteRelatedIssues", err)
		return
	}
	ctx.Data["ForgenteRelatedIssues"] = related
	ctx.Data["ForgenteCanRelate"] = ctx.Repo.Permission.CanWriteIssuesOrPulls(false) && !ctx.Repo.Repository.IsArchived
}
