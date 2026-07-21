// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"context"
	"fmt"

	"forgente.com/models/db"
	user_model "forgente.com/models/user"
	"forgente.com/modules/timeutil"
	"forgente.com/modules/util"

	"xorm.io/builder"
)

// ForgenteIssueRelated links two issues as "related" — a symmetric reference
// without the blocking semantics of issue dependencies. One row represents the
// relation in both directions.
type ForgenteIssueRelated struct {
	ID int64 `xorm:"pk autoincr"`
	// UserID is kept on user deletion: it only records who created the relation, and
	// (matching issue_dependency, which upstream's deleteUser also leaves untouched)
	// the relation's validity doesn't depend on the creator still existing.
	UserID      int64              `xorm:"NOT NULL"`
	IssueID     int64              `xorm:"UNIQUE(forgente_related) NOT NULL"`
	RelatedID   int64              `xorm:"UNIQUE(forgente_related) NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
}

func init() {
	// the table is created by SyncAllTables at startup, no migration needed
	db.RegisterModel(new(ForgenteIssueRelated))
}

// ErrForgenteRelationExists represents an error where the relation already exists
type ErrForgenteRelationExists struct {
	IssueID   int64
	RelatedID int64
}

// IsErrForgenteRelationExists checks if an error is ErrForgenteRelationExists
func IsErrForgenteRelationExists(err error) bool {
	_, ok := err.(ErrForgenteRelationExists)
	return ok
}

func (err ErrForgenteRelationExists) Error() string {
	return fmt.Sprintf("issues are already related [issue id: %d, related id: %d]", err.IssueID, err.RelatedID)
}

func (err ErrForgenteRelationExists) Unwrap() error {
	return util.ErrAlreadyExist
}

// ErrForgenteRelationNotExists represents an error where the relation does not exist
type ErrForgenteRelationNotExists struct {
	IssueID   int64
	RelatedID int64
}

// IsErrForgenteRelationNotExists checks if an error is ErrForgenteRelationNotExists
func IsErrForgenteRelationNotExists(err error) bool {
	_, ok := err.(ErrForgenteRelationNotExists)
	return ok
}

func (err ErrForgenteRelationNotExists) Error() string {
	return fmt.Sprintf("issues are not related [issue id: %d, related id: %d]", err.IssueID, err.RelatedID)
}

func (err ErrForgenteRelationNotExists) Unwrap() error {
	return util.ErrNotExist
}

func forgenteRelationCond(issueID, relatedID int64) builder.Cond {
	return builder.Or(
		builder.Eq{"issue_id": issueID, "related_id": relatedID},
		builder.Eq{"issue_id": relatedID, "related_id": issueID},
	)
}

// CreateForgenteIssueRelation relates two issues and records it on both timelines.
func CreateForgenteIssueRelation(ctx context.Context, doer *user_model.User, issue, related *Issue) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		exists, err := db.GetEngine(ctx).Where(forgenteRelationCond(issue.ID, related.ID)).Exist(&ForgenteIssueRelated{})
		if err != nil {
			return err
		}
		if exists {
			return ErrForgenteRelationExists{IssueID: issue.ID, RelatedID: related.ID}
		}
		if err := db.Insert(ctx, &ForgenteIssueRelated{UserID: doer.ID, IssueID: issue.ID, RelatedID: related.ID}); err != nil {
			return err
		}
		return createForgenteRelatedComment(ctx, doer, issue, related, true)
	})
}

// RemoveForgenteIssueRelation removes the relation between two issues and records it on both timelines.
func RemoveForgenteIssueRelation(ctx context.Context, doer *user_model.User, issue, related *Issue) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		affected, err := db.GetEngine(ctx).Where(forgenteRelationCond(issue.ID, related.ID)).Delete(&ForgenteIssueRelated{})
		if err != nil {
			return err
		}
		if affected == 0 {
			return ErrForgenteRelationNotExists{IssueID: issue.ID, RelatedID: related.ID}
		}
		return createForgenteRelatedComment(ctx, doer, issue, related, false)
	})
}

// GetForgenteRelatedIssues returns all issues related to the given issue, with repositories loaded.
func GetForgenteRelatedIssues(ctx context.Context, issue *Issue) (IssueList, error) {
	relations := make([]*ForgenteIssueRelated, 0, 8)
	if err := db.GetEngine(ctx).Where("issue_id = ? OR related_id = ?", issue.ID, issue.ID).Find(&relations); err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(relations))
	for _, rel := range relations {
		if rel.IssueID == issue.ID {
			ids = append(ids, rel.RelatedID)
		} else {
			ids = append(ids, rel.IssueID)
		}
	}
	issues, err := GetIssuesByIDs(ctx, ids, true)
	if err != nil {
		return nil, err
	}
	if _, err := issues.LoadRepositories(ctx); err != nil {
		return nil, err
	}
	return issues, nil
}

// createForgenteRelatedComment records the relation change on both issue timelines.
func createForgenteRelatedComment(ctx context.Context, doer *user_model.User, issue, related *Issue, add bool) error {
	cType := CommentTypeForgenteAddRelated
	if !add {
		cType = CommentTypeForgenteRemoveRelated
	}
	if err := issue.LoadRepo(ctx); err != nil {
		return err
	}
	if err := related.LoadRepo(ctx); err != nil {
		return err
	}
	if _, err := CreateComment(ctx, &CreateCommentOptions{
		Type:             cType,
		Doer:             doer,
		Repo:             issue.Repo,
		Issue:            issue,
		DependentIssueID: related.ID,
	}); err != nil {
		return err
	}
	_, err := CreateComment(ctx, &CreateCommentOptions{
		Type:             cType,
		Doer:             doer,
		Repo:             related.Repo,
		Issue:            related,
		DependentIssueID: issue.ID,
	})
	return err
}
