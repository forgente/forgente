// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"
	repo_service "forgente.com/services/repository"
	"forgente.com/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoSettingsAgents(t *testing.T) {
	onGiteaRun(t, testRepoSettingsAgents)
}

func testRepoSettingsAgents(t *testing.T, _ *url.URL) {
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	// its own repository, since this pushes files: a shared fixture would
	// change results for every other test that reads it
	repo, err := repo_service.CreateRepositoryDirectly(t.Context(), user, user, repo_service.CreateRepoOptions{
		Name:          "agent-definitions-page",
		Readme:        "Default",
		AutoInit:      true,
		DefaultBranch: "main",
	}, true)
	require.NoError(t, err)

	session := loginUser(t, user.Name)
	settingsLink := fmt.Sprintf("/%s/%s/settings/agents", user.Name, repo.Name)

	t.Run("EmptyStateNamesTheLocations", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		body := session.MakeRequest(t, NewRequest(t, "GET", settingsLink), http.StatusOK).Body.String()
		assert.Contains(t, body, "No skills declared")
		assert.Contains(t, body, "No agent profiles declared")
		// an owner with nothing declared still learns where to put it
		assert.Contains(t, body, ".agents/skills/")
		assert.Contains(t, body, ".github/agents/")
	})

	t.Run("OnlyCollaboratorsCanSeeIt", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// user4 has no access to user2's repository
		loginUser(t, "user4").MakeRequest(t, NewRequest(t, "GET", settingsLink), http.StatusNotFound)
	})

	_, err = createFileInBranch(user, repo, createFileInBranchOptions{OldBranch: repo.DefaultBranch}, map[string]string{
		".agents/skills/pdf-processing/SKILL.md": "---\nname: pdf-processing\ndescription: Extracts text from PDFs.\ncompatibility: Requires python\n---\nStep one.\n",
		".github/agents/review.agent.md":         "---\ndescription: Reviews pull requests.\ntools: [read, search]\n---\nReview it.\n",
		".github/agents/quiet.agent.md":          "---\ndescription: Runs on its own.\nuser-invocable: false\n---\n",
		// malformed: the name disagrees with the directory
		".agents/skills/mismatched/SKILL.md": "---\nname: something-else\ndescription: Wrong name.\n---\n",
	})
	require.NoError(t, err)

	t.Run("ListsDefinitions", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		body := session.MakeRequest(t, NewRequest(t, "GET", settingsLink), http.StatusOK).Body.String()
		assert.Contains(t, body, "pdf-processing")
		assert.Contains(t, body, "Extracts text from PDFs")
		assert.Contains(t, body, "Requires python")
		// a profile with no name field falls back to its file identifier
		assert.Contains(t, body, "review")
		assert.Contains(t, body, "read, search")
		assert.Contains(t, body, "Not user-invocable")
	})

	t.Run("ShowsWhyADefinitionFailedToLoad", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		body := session.MakeRequest(t, NewRequest(t, "GET", settingsLink), http.StatusOK).Body.String()
		// the whole point of the page: a silently-absent definition is the
		// hardest kind to debug, so the reason is shown next to the path
		assert.Contains(t, body, ".agents/skills/mismatched/SKILL.md")
		assert.Contains(t, body, "must match the directory name")
	})
}
