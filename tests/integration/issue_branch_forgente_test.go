// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	git_model "gitea.dev/models/git"
	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"

	"github.com/stretchr/testify/assert"
)

func TestForgenteCreateBranchFromIssue(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user2")
		req := NewRequestWithValues(t, "POST", "/user2/repo1/issues/1/create_branch", map[string]string{
			"new_branch_name": "1-branch-from-issue",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		branch, err := git_model.GetBranch(t.Context(), 1, "1-branch-from-issue")
		assert.NoError(t, err)
		assert.NotNil(t, branch)

		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: 1, Index: 1})
		unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
			IssueID: issue.ID,
			Type:    issues_model.CommentTypeForgenteCreateBranch,
			NewRef:  "1-branch-from-issue",
		})
	})
}
