// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/url"
	"testing"

	repo_model "forgente.com/models/repo"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"
	"forgente.com/modules/git"
	agent_service "forgente.com/services/agent"
	"forgente.com/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentDefinitionDiscovery(t *testing.T) {
	// the push that creates the fixture files runs the pre-receive hook, which
	// calls back into the server, so this needs a running instance
	onGiteaRun(t, testAgentDefinitionDiscovery)
}

func testAgentDefinitionDiscovery(t *testing.T, _ *url.URL) {
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: user.ID})

	load := func(t *testing.T) *agent_service.Definitions {
		t.Helper()
		gitRepo, err := git.OpenRepository(repo)
		require.NoError(t, err)
		defer gitRepo.Close()
		return agent_service.LoadFromDefaultBranch(t.Context(), repo, gitRepo)
	}

	t.Run("NothingDeclared", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		found := load(t)
		assert.True(t, found.IsEmpty())
	})

	_, err := createFileInBranch(user, repo, createFileInBranchOptions{OldBranch: repo.DefaultBranch}, map[string]string{
		// a well-formed skill in the vendor-neutral location
		".agents/skills/pdf-processing/SKILL.md": "---\nname: pdf-processing\ndescription: Extracts text from PDFs. Use when handling PDFs.\n---\nStep one.\n",
		// the same skill name in a lower-precedence location, which must lose
		".claude/skills/pdf-processing/SKILL.md": "---\nname: pdf-processing\ndescription: A different description that must not win.\n---\n",
		// a skill whose name disagrees with its directory: invalid per the spec
		".agents/skills/mismatched/SKILL.md": "---\nname: something-else\ndescription: Name does not match the directory.\n---\n",
		// a directory with no SKILL.md is simply not a skill, not an error
		".agents/skills/not-a-skill/README.md": "nothing here\n",

		// a well-formed profile, plus one using the bare .md extension
		".github/agents/review.agent.md": "---\ndescription: Reviews pull requests.\n---\nReview the diff.\n",
		".github/agents/triage.md":       "---\nname: Triage Bot\ndescription: Triages issues.\n---\n",
		// same identifier in the lower-precedence directory, which must lose
		".claude/agents/review.agent.md": "---\ndescription: Must not win.\n---\n",
		// missing the required description
		".github/agents/broken.agent.md": "---\nname: broken\n---\n",
		// not a profile at all, and must not be reported as an error
		".github/agents/notes.txt": "just notes\n",
	})
	require.NoError(t, err)

	t.Run("Skills", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		found := load(t)
		require.Len(t, found.Skills, 1)
		skill := found.Skills[0]
		assert.Equal(t, "pdf-processing", skill.Name)
		// .agents/skills wins over .claude/skills
		assert.Contains(t, skill.Description, "Extracts text from PDFs")
		assert.Equal(t, "Step one.\n", skill.Body)
	})

	t.Run("Profiles", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		found := load(t)
		require.Len(t, found.Profiles, 2)

		byID := map[string]string{}
		for _, p := range found.Profiles {
			byID[p.ID] = p.Description
		}
		// .github/agents wins over .claude/agents for the same identifier
		assert.Contains(t, byID["review"], "Reviews pull requests")
		// the bare .md extension is valid too
		assert.Contains(t, byID["triage"], "Triages issues")
	})

	t.Run("MalformedDefinitionsAreReportedNotDropped", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		found := load(t)

		mismatch := found.Errors[".agents/skills/mismatched/SKILL.md"]
		require.Error(t, mismatch)
		assert.Contains(t, mismatch.Error(), "must match the directory name")

		broken := found.Errors[".github/agents/broken.agent.md"]
		require.Error(t, broken)
		assert.Contains(t, broken.Error(), "description is required")

		// a directory without SKILL.md and a non-profile file are both silent
		assert.NotContains(t, found.Errors, ".agents/skills/not-a-skill/SKILL.md")
		assert.NotContains(t, found.Errors, ".github/agents/notes.txt")
	})
}
