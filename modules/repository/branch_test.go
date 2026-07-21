// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"testing"

	"forgente.com/models/db"
	git_model "forgente.com/models/git"
	repo_model "forgente.com/models/repo"
	"forgente.com/models/unittest"

	"github.com/stretchr/testify/assert"
)

func TestSyncRepoBranches(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	_, err := db.GetEngine(t.Context()).ID(1).Update(&repo_model.Repository{ObjectFormatName: "bad-fmt"})
	assert.NoError(t, db.TruncateBeans(t.Context(), &git_model.Branch{}))
	assert.NoError(t, err)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	assert.Equal(t, "bad-fmt", repo.ObjectFormatName)
	_, err = SyncRepoBranches(t.Context(), 1, 0)
	assert.NoError(t, err)
	repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	assert.Equal(t, "sha1", repo.ObjectFormatName)
	branch, err := git_model.GetBranch(t.Context(), 1, "master")
	assert.NoError(t, err)
	assert.Equal(t, "master", branch.Name)
}
