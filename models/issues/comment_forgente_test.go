// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	issues_model "forgente.com/models/issues"
	repo_model "forgente.com/models/repo"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"

	"github.com/stretchr/testify/assert"
)

func TestGetForgenteCreateBranchNames(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})

	// no comments yet
	names, err := issues_model.GetForgenteCreateBranchNames(t.Context(), issue.ID)
	assert.NoError(t, err)
	assert.Empty(t, names)

	assert.NoError(t, issues_model.AddCreateBranchComment(t.Context(), doer, repo, issue.ID, "branch-a"))
	assert.NoError(t, issues_model.AddCreateBranchComment(t.Context(), doer, repo, issue.ID, "branch-b"))
	// duplicate name (e.g. branch recreated after deletion) must be deduplicated
	assert.NoError(t, issues_model.AddCreateBranchComment(t.Context(), doer, repo, issue.ID, "branch-a"))

	names, err = issues_model.GetForgenteCreateBranchNames(t.Context(), issue.ID)
	assert.NoError(t, err)
	assert.Equal(t, []string{"branch-a", "branch-b"}, names)
}
