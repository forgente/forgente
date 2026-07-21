// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"strings"

	"forgente.com/models/db"
	git_model "forgente.com/models/git"
	issues_model "forgente.com/models/issues"
	"forgente.com/modules/git"
	"forgente.com/modules/log"
	"forgente.com/modules/optional"
	"forgente.com/modules/web"
	"forgente.com/services/context"
	"forgente.com/services/forms"
	repo_service "forgente.com/services/repository"
)

// forgenteBranchOptionsLimit caps how many non-default branch names are loaded for the
// base-branch selector, to bound page cost on repositories with many branches.
const forgenteBranchOptionsLimit = 100

// ForgenteCreateBranchFromIssue creates a branch from an optional base branch (defaulting to
// the repository's default branch) and records it on the issue timeline.
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

	baseBranchName := strings.TrimSpace(ctx.FormString("base_branch_name"))
	if baseBranchName == "" {
		baseBranchName = ctx.Repo.Repository.DefaultBranch
	} else if exists, err := git_model.IsBranchExist(ctx, ctx.Repo.Repository.ID, baseBranchName); err != nil {
		ctx.ServerError("IsBranchExist", err)
		return
	} else if !exists {
		ctx.Flash.Error(ctx.Tr("repo.editor.branch_does_not_exist", baseBranchName))
		ctx.Redirect(issue.Link())
		return
	}

	if err := repo_service.CreateNewBranch(ctx, ctx.Doer, ctx.Repo.Repository, ctx.Repo.GitRepo, baseBranchName, form.NewBranchName); err != nil {
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

// forgentePrepareIssueViewSidebarBranch loads the base-branch selector options and the list of
// branches previously created from this issue. It mirrors the render conditions of the
// create-branch button/modal so the queries are skipped when the button wouldn't show.
func forgentePrepareIssueViewSidebarBranch(ctx *context.Context, issue *issues_model.Issue) {
	if issue.IsPull || ctx.Repo.Repository.IsArchived || ctx.Repo.Repository.IsEmpty || !ctx.Repo.CanCreateBranch() {
		return
	}

	defaultBranch := ctx.Repo.Repository.DefaultBranch
	names, err := git_model.FindBranchNames(ctx, git_model.FindBranchOptions{
		RepoID:             ctx.Repo.Repository.ID,
		IsDeletedBranch:    optional.Some(false),
		ExcludeBranchNames: []string{defaultBranch},
		ListOptions:        db.ListOptions{Page: 1, PageSize: forgenteBranchOptionsLimit},
	})
	if err != nil {
		ctx.ServerError("FindBranchNames", err)
		return
	}
	// default branch is always first, so it is selected by default in the <select>
	ctx.Data["ForgenteBranchOptions"] = append([]string{defaultBranch}, names...)

	createdNames, err := issues_model.GetForgenteCreateBranchNames(ctx, issue.ID)
	if err != nil {
		ctx.ServerError("GetForgenteCreateBranchNames", err)
		return
	}
	if len(createdNames) == 0 {
		return
	}
	existingBranches, err := git_model.GetBranches(ctx, ctx.Repo.Repository.ID, createdNames, false)
	if err != nil {
		ctx.ServerError("GetBranches", err)
		return
	}
	existing := git_model.BranchesToNamesSet(existingBranches)
	createdBranches := make([]string, 0, len(createdNames))
	for _, name := range createdNames {
		if existing.Contains(name) {
			createdBranches = append(createdBranches, name)
		}
	}
	ctx.Data["ForgenteCreatedBranches"] = createdBranches
}
