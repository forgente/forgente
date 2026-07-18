// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	issues_model "gitea.dev/models/issues"

	"github.com/stretchr/testify/assert"
)

// TestForgenteOrgMilestonesOverview verifies the grouped "group milestone"
// roll-up view combines same-named milestones from different org repos into
// a single card with summed progress (closes the docs-comparison gap with
// GitLab's group milestones; see upstream go-gitea/gitea#14622).
func TestForgenteOrgMilestonesOverview(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		// org3's repo3 and repo5 (both owner_id: 3) don't have milestones in the
		// fixtures, so create matching-named ones at runtime rather than editing
		// milestone.yml/repository.yml (which would need num_milestones kept in sync).
		assert.NoError(t, issues_model.NewMilestone(t.Context(), &issues_model.Milestone{
			RepoID:          3,
			Name:            "Forgente Q1",
			NumIssues:       2,
			NumClosedIssues: 1,
		}))
		assert.NoError(t, issues_model.NewMilestone(t.Context(), &issues_model.Milestone{
			RepoID:          5,
			Name:            "forgente q1", // different casing, same group
			NumIssues:       0,
			NumClosedIssues: 0,
		}))

		session := loginUser(t, "user2") // member of org3
		req := NewRequest(t, "GET", "/org/org3/milestones-overview")
		resp := session.MakeRequest(t, req, http.StatusOK)

		htmlDoc := NewHTMLParser(t, resp.Body)
		text := htmlDoc.Find(".page-content").Text()
		assert.Contains(t, text, "Forgente Q1")
		// group combines: repo3 (2 issues/1 closed -> 1 open) + repo5 (0 issues) = 1 open, 1 closed, 50% complete
		assert.Contains(t, text, "50%")
		assert.Contains(t, text, "repo3")
		assert.Contains(t, text, "repo5")
	})
}
