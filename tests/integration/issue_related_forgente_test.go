// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	issues_model "forgente.com/models/issues"
	"forgente.com/models/unittest"
	"forgente.com/tests"
)

func TestForgenteRelatedIssues(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")

	// relate repo1 issue #1 to issue #4 (issue id 5)
	req := NewRequestWithValues(t, "POST", "/user2/repo1/issues/1/related/add", map[string]string{
		"related_ref": "#4",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	issue1 := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: 1, Index: 1})
	issue4 := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: 1, Index: 4})
	unittest.AssertExistsAndLoadBean(t, &issues_model.ForgenteIssueRelated{IssueID: issue1.ID, RelatedID: issue4.ID})
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		IssueID: issue4.ID, Type: issues_model.CommentTypeForgenteAddRelated, DependentIssueID: issue1.ID,
	})

	// the issue page renders with the relation in the sidebar
	req = NewRequest(t, "GET", "/user2/repo1/issues/1")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	assert1 := htmlDoc.Find(".issue-content-right a[href$='/user2/repo1/issues/4']")
	if assert1.Length() == 0 {
		t.Error("related issue link not found in sidebar")
	}

	// remove the relation from the other side
	req = NewRequestWithValues(t, "POST", "/user2/repo1/issues/4/related/delete", map[string]string{
		"related_id": "1",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)
	unittest.AssertNotExistsBean(t, &issues_model.ForgenteIssueRelated{IssueID: issue1.ID, RelatedID: issue4.ID})
}
