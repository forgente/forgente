// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	git_model "forgente.com/models/git"
	issues_model "forgente.com/models/issues"
	"forgente.com/models/unittest"

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

func TestForgenteCreateBranchFromIssueWithBaseBranch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user2")

		base, err := git_model.GetBranch(t.Context(), 1, "branch2")
		assert.NoError(t, err)

		req := NewRequestWithValues(t, "POST", "/user2/repo1/issues/1/create_branch", map[string]string{
			"new_branch_name":  "1-branch-from-branch2",
			"base_branch_name": "branch2",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		branch, err := git_model.GetBranch(t.Context(), 1, "1-branch-from-branch2")
		assert.NoError(t, err)
		assert.NotNil(t, branch)
		assert.Equal(t, base.CommitID, branch.CommitID) // created from branch2's head, not the default branch
	})
}

func TestForgenteCreateBranchFromIssueInvalidBaseBranch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user2")
		req := NewRequestWithValues(t, "POST", "/user2/repo1/issues/1/create_branch", map[string]string{
			"new_branch_name":  "1-branch-should-not-exist",
			"base_branch_name": "does-not-exist",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		flashMsg := session.GetCookieFlashMessage()
		assert.NotEmpty(t, flashMsg.ErrorMsg)

		unittest.AssertNotExistsBean(t, &git_model.Branch{RepoID: 1, Name: "1-branch-should-not-exist"})
	})
}

func TestForgenteIssueSidebarShowsCreatedBranches(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user2")
		req := NewRequestWithValues(t, "POST", "/user2/repo1/issues/1/create_branch", map[string]string{
			"new_branch_name": "1-branch-in-sidebar",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		req = NewRequest(t, "GET", "/user2/repo1/issues/1")
		resp := session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)
		link := htmlDoc.Find(".issue-content-right a[href$='/user2/repo1/src/branch/1-branch-in-sidebar']")
		assert.Equal(t, 1, link.Length())
	})
}
