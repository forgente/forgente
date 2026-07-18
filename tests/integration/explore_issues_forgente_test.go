// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"gitea.dev/models/db"
	issue_indexer "gitea.dev/modules/indexer/issues"
	"gitea.dev/tests"

	"github.com/stretchr/testify/assert"
)

// waitIssueIndexed queues an issue for indexing and polls until a scoped search finds
// it, so the assertions below never race the async indexer queue.
func waitIssueIndexed(t *testing.T, issueID, repoID int64, keyword string) {
	t.Helper()
	issue_indexer.UpdateIssueIndexer(t.Context(), issueID)
	deadline := time.Now().Add(10 * time.Second)
	for {
		ids, _, err := issue_indexer.SearchIssues(t.Context(), &issue_indexer.SearchOptions{
			Keyword:   keyword,
			RepoIDs:   []int64{repoID},
			Paginator: &db.ListOptions{Page: 1, PageSize: 20},
		})
		assert.NoError(t, err)
		if slices.Contains(ids, issueID) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("issue %d was not indexed in time", issueID)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// TestForgenteExploreIssuesPublic checks that the new /explore/issues page finds a
// known open issue from a public fixture repo (repo1, is_private: false) for an
// anonymous (not signed-in) visitor.
func TestForgenteExploreIssuesPublic(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// fixture issue id 1 ("issue1") belongs to repo1, which is public
	waitIssueIndexed(t, 1, 1, "issue1")

	req := NewRequest(t, "GET", "/explore/issues?q=issue1")
	resp := MakeRequest(t, req, http.StatusOK)

	// assert on the result list only; the page always echoes the query elsewhere
	listText := NewHTMLParser(t, resp.Body).Find("#issue-list").Text()
	assert.Contains(t, listText, "issue1")
	assert.Contains(t, listText, "user2/repo1")
}

// TestForgenteExploreIssuesPrivacy is the non-negotiable privacy check: an issue that
// lives in a private fixture repo (repo2, is_private: true) must never surface on the
// global explore/issues page for an anonymous visitor. waitIssueIndexed proves the
// issue IS in the index (via a repo-scoped search), so the absence below is enforced
// by the AllPublic filter, not by an empty index.
func TestForgenteExploreIssuesPrivacy(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// fixture issue id 7 ("issue7") belongs to repo2, which is private
	waitIssueIndexed(t, 7, 2, "issue7")

	req := NewRequest(t, "GET", "/explore/issues?q=issue7")
	resp := MakeRequest(t, req, http.StatusOK)

	// the query is echoed in the search box, so only the result list must be clean
	listText := NewHTMLParser(t, resp.Body).Find("#issue-list").Text()
	assert.NotContains(t, listText, "issue7")
	assert.NotContains(t, listText, "user2/repo2")
}
