// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package incoming

import (
	"testing"

	issues_model "forgente.com/models/issues"
	repo_model "forgente.com/models/repo"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"
	incoming_payload "forgente.com/services/mailer/incoming/payload"

	"github.com/stretchr/testify/assert"
)

func TestForgenteNewIssueHandler_Handle(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // user2/repo1
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // user2, owner of repo1, has write access

	payload, err := incoming_payload.CreateForgenteNewIssuePayload(repo)
	assert.NoError(t, err)

	content := &MailContent{
		Subject: "New issue via mail",
		Content: "issue body from mail",
	}

	before, err := issues_model.CountIssues(t.Context(), &issues_model.IssuesOptions{RepoIDs: []int64{repo.ID}})
	assert.NoError(t, err)

	h := &ForgenteNewIssueHandler{}
	assert.NoError(t, h.Handle(t.Context(), content, doer, payload))

	after, err := issues_model.CountIssues(t.Context(), &issues_model.IssuesOptions{RepoIDs: []int64{repo.ID}})
	assert.NoError(t, err)
	assert.Equal(t, before+1, after)

	issues, err := issues_model.Issues(t.Context(), &issues_model.IssuesOptions{RepoIDs: []int64{repo.ID}, SortType: "newest"})
	assert.NoError(t, err)
	if assert.NotEmpty(t, issues) {
		issue := issues[0]
		assert.Equal(t, "New issue via mail", issue.Title)
		assert.Equal(t, "issue body from mail", issue.Content)
		assert.Equal(t, doer.ID, issue.PosterID)
	}
}

func TestForgenteNewIssueHandler_Handle_NoPermission(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // user2/repo1
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})       // user4, no collaboration on repo1

	payload, err := incoming_payload.CreateForgenteNewIssuePayload(repo)
	assert.NoError(t, err)

	content := &MailContent{
		Subject: "Should not be created",
		Content: "issue body from mail",
	}

	before, err := issues_model.CountIssues(t.Context(), &issues_model.IssuesOptions{RepoIDs: []int64{repo.ID}})
	assert.NoError(t, err)

	h := &ForgenteNewIssueHandler{}
	assert.NoError(t, h.Handle(t.Context(), content, doer, payload))

	after, err := issues_model.CountIssues(t.Context(), &issues_model.IssuesOptions{RepoIDs: []int64{repo.ID}})
	assert.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestForgenteNewIssueHandler_Handle_EmptySubject(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	payload, err := incoming_payload.CreateForgenteNewIssuePayload(repo)
	assert.NoError(t, err)

	content := &MailContent{
		Subject: "   ",
		Content: "issue body from mail",
	}

	before, err := issues_model.CountIssues(t.Context(), &issues_model.IssuesOptions{RepoIDs: []int64{repo.ID}})
	assert.NoError(t, err)

	// Blank subject is dropped without error so the mail still counts as handled.
	h := &ForgenteNewIssueHandler{}
	assert.NoError(t, h.Handle(t.Context(), content, doer, payload))

	after, err := issues_model.CountIssues(t.Context(), &issues_model.IssuesOptions{RepoIDs: []int64{repo.ID}})
	assert.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestForgenteNewIssueHandler_Handle_SubjectSanitized(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	payload, err := incoming_payload.CreateForgenteNewIssuePayload(repo)
	assert.NoError(t, err)

	content := &MailContent{
		Subject: " broken\r\n subject\x07 line ",
		Content: "body",
	}

	h := &ForgenteNewIssueHandler{}
	assert.NoError(t, h.Handle(t.Context(), content, doer, payload))

	issues, err := issues_model.Issues(t.Context(), &issues_model.IssuesOptions{RepoIDs: []int64{repo.ID}, SortType: "newest"})
	assert.NoError(t, err)
	if assert.NotEmpty(t, issues) {
		assert.Equal(t, "broken subject line", issues[0].Title)
	}
}
