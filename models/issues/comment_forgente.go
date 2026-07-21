// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"context"

	"forgente.com/models/db"
	repo_model "forgente.com/models/repo"
	user_model "forgente.com/models/user"
	"forgente.com/modules/container"
)

// Forgente comment types use explicit values starting at 1000 so upstream
// additions to the iota-based CommentType list can never collide with values
// already stored in the database.
const (
	// CommentTypeForgenteCreateBranch records that a branch was created from the issue
	CommentTypeForgenteCreateBranch CommentType = 1000
	// CommentTypeForgenteAddRelated records that another issue was marked as related
	CommentTypeForgenteAddRelated CommentType = 1001
	// CommentTypeForgenteRemoveRelated records that a related issue was removed
	CommentTypeForgenteRemoveRelated CommentType = 1002
)

var forgenteCommentStrings = map[CommentType]string{
	CommentTypeForgenteCreateBranch:  "forgente_create_branch",
	CommentTypeForgenteAddRelated:    "forgente_add_related",
	CommentTypeForgenteRemoveRelated: "forgente_remove_related",
}

// IsForgenteRelatedCommentType reports whether t is one of the related-issue comment types,
// which carry a DependentIssueID like the dependency comment types do.
func IsForgenteRelatedCommentType(t CommentType) bool {
	return t == CommentTypeForgenteAddRelated || t == CommentTypeForgenteRemoveRelated
}

func asForgenteCommentType(typeName string) CommentType {
	for t, name := range forgenteCommentStrings {
		if name == typeName {
			return t
		}
	}
	return CommentTypeUndefined
}

// AddCreateBranchComment records on the issue timeline that doer created branchName from the issue.
func AddCreateBranchComment(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, issueID int64, branchName string) error {
	issue, err := GetIssueByID(ctx, issueID)
	if err != nil {
		return err
	}
	_, err = CreateComment(ctx, &CreateCommentOptions{
		Type:   CommentTypeForgenteCreateBranch,
		Doer:   doer,
		Repo:   repo,
		Issue:  issue,
		NewRef: branchName,
	})
	return err
}

// forgenteCreateBranchNamesLimit caps how many create-branch comments are read per issue, to
// bound page cost on long-lived issues (matches the base-branch selector's cap).
const forgenteCreateBranchNamesLimit = 100

// GetForgenteCreateBranchNames returns the distinct branch names recorded as created-from-issue
// comments for issueID, in the order they were first created. Callers should filter the result
// against the repository's live branches, since a comment persists after its branch is deleted.
func GetForgenteCreateBranchNames(ctx context.Context, issueID int64) ([]string, error) {
	comments, err := FindComments(ctx, &FindCommentsOptions{
		IssueID:     issueID,
		Type:        CommentTypeForgenteCreateBranch,
		ListOptions: db.ListOptions{Page: 1, PageSize: forgenteCreateBranchNamesLimit},
	})
	if err != nil {
		return nil, err
	}
	return container.FilterSlice(comments, func(c *Comment) (string, bool) {
		return c.NewRef, c.NewRef != ""
	}), nil
}
