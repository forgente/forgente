// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileIDFromFileName(t *testing.T) {
	// the longer extension is stripped first, or "review.agent.md" would
	// deduplicate as "review.agent" and never collide with "review.md"
	assert.Equal(t, "review", ProfileIDFromFileName("review.agent.md"))
	assert.Equal(t, "review", ProfileIDFromFileName("review.md"))
	assert.Equal(t, "triage-issues", ProfileIDFromFileName("triage-issues.agent.md"))
	assert.Empty(t, ProfileIDFromFileName("review.txt"))
	assert.Empty(t, ProfileIDFromFileName("review"))
}

func TestParseProfile(t *testing.T) {
	t.Run("Minimal", func(t *testing.T) {
		profile, err := ParseProfile([]byte(`---
description: Reviews pull requests.
---
Review the diff.
`), "review.agent.md")
		require.NoError(t, err)
		assert.Equal(t, "review", profile.ID)
		// name is optional and falls back to the file identifier
		assert.Empty(t, profile.Name)
		assert.Equal(t, "review", profile.DisplayName())
		assert.Equal(t, "Review the diff.\n", profile.Prompt)
		// the defaults are not the zero values
		assert.True(t, profile.IsUserInvocable())
		assert.True(t, profile.IsModelInvocable())
		assert.True(t, profile.AllowsAllTools())
	})

	t.Run("DescriptionIsRequired", func(t *testing.T) {
		_, err := ParseProfile([]byte("---\nname: review\n---\n"), "review.agent.md")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "description is required")
	})

	t.Run("ToolsAsList", func(t *testing.T) {
		profile, err := ParseProfile([]byte(`---
description: d
tools: [read, write]
---
`), "a.md")
		require.NoError(t, err)
		assert.Equal(t, ProfileTools{"read", "write"}, profile.Tools)
		assert.False(t, profile.AllowsAllTools())
	})

	t.Run("ToolsAsCommaSeparatedString", func(t *testing.T) {
		profile, err := ParseProfile([]byte(`---
description: d
tools: "read, write ,search"
---
`), "a.md")
		require.NoError(t, err)
		assert.Equal(t, ProfileTools{"read", "write", "search"}, profile.Tools)
	})

	t.Run("ToolsWildcardMeansAll", func(t *testing.T) {
		profile, err := ParseProfile([]byte("---\ndescription: d\ntools: [\"*\"]\n---\n"), "a.md")
		require.NoError(t, err)
		assert.True(t, profile.AllowsAllTools())
	})

	t.Run("EmptyToolsListDisablesTools", func(t *testing.T) {
		// an empty list is not the same as an omitted field
		profile, err := ParseProfile([]byte("---\ndescription: d\ntools: []\n---\n"), "a.md")
		require.NoError(t, err)
		assert.False(t, profile.AllowsAllTools())
		assert.Empty(t, profile.Tools)
	})

	t.Run("InvocationFlags", func(t *testing.T) {
		profile, err := ParseProfile([]byte(`---
description: d
disable-model-invocation: true
user-invocable: false
---
`), "a.md")
		require.NoError(t, err)
		assert.False(t, profile.IsModelInvocable())
		assert.False(t, profile.IsUserInvocable())
	})

	t.Run("Target", func(t *testing.T) {
		profile, err := ParseProfile([]byte("---\ndescription: d\ntarget: vscode\n---\n"), "a.md")
		require.NoError(t, err)
		assert.Equal(t, ProfileTargetVSCode, profile.Target)

		_, err = ParseProfile([]byte("---\ndescription: d\ntarget: emacs\n---\n"), "a.md")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target must be")
	})

	t.Run("PromptIsBounded", func(t *testing.T) {
		body := strings.Repeat("a", ProfilePromptMaxLen+1)
		_, err := ParseProfile([]byte("---\ndescription: d\n---\n"+body), "a.md")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at most")
	})

	t.Run("FileNameConstraints", func(t *testing.T) {
		content := []byte("---\ndescription: d\n---\n")

		_, err := ParseProfile(content, "review.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), ".agent.md")

		_, err = ParseProfile(content, "my agent.md")
		require.Error(t, err)

		// dots, hyphens and underscores are all accepted
		_, err = ParseProfile(content, "my-agent_v2.agent.md")
		assert.NoError(t, err)
	})

	t.Run("MCPServersAndMetadata", func(t *testing.T) {
		profile, err := ParseProfile([]byte(`---
description: d
mcp-servers:
  forgente:
    command: gitea-mcp
metadata:
  owner: platform
---
`), "a.md")
		require.NoError(t, err)
		assert.Contains(t, profile.MCPServers, "forgente")
		assert.Equal(t, map[string]string{"owner": "platform"}, profile.Metadata)
	})
}
