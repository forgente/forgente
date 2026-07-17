// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"

	"github.com/stretchr/testify/assert"
)

func TestForgenteIssueRelation(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	issue1 := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	issue5 := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 5})

	assert.NoError(t, issues_model.CreateForgenteIssueRelation(t.Context(), doer, issue1, issue5))

	// duplicate in either direction is rejected
	err := issues_model.CreateForgenteIssueRelation(t.Context(), doer, issue1, issue5)
	assert.True(t, issues_model.IsErrForgenteRelationExists(err))
	err = issues_model.CreateForgenteIssueRelation(t.Context(), doer, issue5, issue1)
	assert.True(t, issues_model.IsErrForgenteRelationExists(err))

	// both issues see the relation
	related1, err := issues_model.GetForgenteRelatedIssues(t.Context(), issue1)
	assert.NoError(t, err)
	if assert.Len(t, related1, 1) {
		assert.Equal(t, issue5.ID, related1[0].ID)
	}
	related5, err := issues_model.GetForgenteRelatedIssues(t.Context(), issue5)
	assert.NoError(t, err)
	if assert.Len(t, related5, 1) {
		assert.Equal(t, issue1.ID, related5[0].ID)
	}

	// timeline comments exist on both issues
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		IssueID: issue1.ID, Type: issues_model.CommentTypeForgenteAddRelated, DependentIssueID: issue5.ID,
	})
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		IssueID: issue5.ID, Type: issues_model.CommentTypeForgenteAddRelated, DependentIssueID: issue1.ID,
	})

	// removal works from the other side too and records comments
	assert.NoError(t, issues_model.RemoveForgenteIssueRelation(t.Context(), doer, issue5, issue1))
	related1, err = issues_model.GetForgenteRelatedIssues(t.Context(), issue1)
	assert.NoError(t, err)
	assert.Empty(t, related1)
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		IssueID: issue1.ID, Type: issues_model.CommentTypeForgenteRemoveRelated, DependentIssueID: issue5.ID,
	})

	// removing a non-existent relation errors
	err = issues_model.RemoveForgenteIssueRelation(t.Context(), doer, issue1, issue5)
	assert.True(t, issues_model.IsErrForgenteRelationNotExists(err))
}
