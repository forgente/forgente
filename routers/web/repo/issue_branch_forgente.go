// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	git_model "gitea.dev/models/git"
	issues_model "gitea.dev/models/issues"
	"gitea.dev/modules/git"
	"gitea.dev/modules/log"
	"gitea.dev/modules/web"
	"gitea.dev/services/context"
	"gitea.dev/services/forms"
	repo_service "gitea.dev/services/repository"
)

// ForgenteCreateBranchFromIssue creates a branch from the repository's default
// branch and records it on the issue timeline.
func ForgenteCreateBranchFromIssue(ctx *context.Context) {
	issue := GetActionIssue(ctx)
	if ctx.Written() {
		return
	}
	if issue.IsPull || ctx.Repo.Repository.IsEmpty || !ctx.Repo.CanCreateBranch() {
		ctx.NotFound(nil)
		return
	}

	form := web.GetForm(ctx).(*forms.NewBranchForm)
	if ctx.HasError() {
		ctx.Flash.Error(ctx.GetErrMsg())
		ctx.Redirect(issue.Link())
		return
	}

	if err := repo_service.CreateNewBranch(ctx, ctx.Doer, ctx.Repo.Repository, ctx.Repo.GitRepo, ctx.Repo.Repository.DefaultBranch, form.NewBranchName); err != nil {
		switch {
		case git_model.IsErrBranchAlreadyExists(err) || git.IsErrPushOutOfDate(err):
			ctx.Flash.Error(ctx.Tr("repo.branch.branch_already_exists", form.NewBranchName))
		case git_model.IsErrBranchNameConflict(err):
			e := err.(git_model.ErrBranchNameConflict)
			ctx.Flash.Error(ctx.Tr("repo.branch.branch_name_conflict", form.NewBranchName, e.BranchName))
		case git.IsErrPushRejected(err):
			ctx.Flash.Error(ctx.Tr("repo.editor.push_rejected_no_message"))
		default:
			ctx.ServerError("CreateNewBranch", err)
			return
		}
		ctx.Redirect(issue.Link())
		return
	}

	if err := issues_model.AddCreateBranchComment(ctx, ctx.Doer, ctx.Repo.Repository, issue.ID, form.NewBranchName); err != nil {
		log.Error("AddCreateBranchComment: %v", err) // branch already created, do not fail the request
	}

	ctx.Flash.Success(ctx.Tr("repo.branch.create_success", form.NewBranchName))
	ctx.Redirect(issue.Link())
}
