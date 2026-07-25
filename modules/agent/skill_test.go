// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSkillName(t *testing.T) {
	// the valid and invalid names the specification itself gives as examples
	for _, name := range []string{"pdf-processing", "data-analysis", "code-review", "a", "a1"} {
		assert.NoError(t, ValidateSkillName(name), name)
	}

	for _, tc := range []struct{ name, reason string }{
		{"", "empty"},
		{"PDF-Processing", "uppercase"},
		{"-pdf", "leading hyphen"},
		{"pdf-", "trailing hyphen"},
		{"pdf--processing", "consecutive hyphens"},
		{"pdf_processing", "underscore"},
		{"pdf processing", "space"},
		{strings.Repeat("a", SkillNameMaxLen+1), "too long"},
	} {
		assert.Error(t, ValidateSkillName(tc.name), tc.reason)
	}

	assert.NoError(t, ValidateSkillName(strings.Repeat("a", SkillNameMaxLen)), "exactly at the limit")
}

func TestParseSkill(t *testing.T) {
	t.Run("Minimal", func(t *testing.T) {
		// the specification's minimal example: name and description only
		skill, err := ParseSkill([]byte(`---
name: skill-name
description: A description of what this skill does and when to use it.
---
`), "skill-name")
		require.NoError(t, err)
		assert.Equal(t, "skill-name", skill.Name)
		assert.Equal(t, "A description of what this skill does and when to use it.", skill.Description)
		assert.Empty(t, skill.License)
		assert.Empty(t, skill.Metadata)
	})

	t.Run("OptionalFields", func(t *testing.T) {
		skill, err := ParseSkill([]byte(`---
name: pdf-processing
description: Extract PDF text, fill forms, merge files. Use when handling PDFs.
license: Apache-2.0
compatibility: Requires git, docker, jq, and access to the internet
allowed-tools: Bash(git:*) Bash(jq:*) Read
metadata:
  author: example-org
  version: "1.0"
---
Step one.
`), "pdf-processing")
		require.NoError(t, err)
		assert.Equal(t, "Apache-2.0", skill.License)
		assert.Equal(t, "Requires git, docker, jq, and access to the internet", skill.Compatibility)
		// experimental and agent-specific, so it is carried through unparsed
		assert.Equal(t, "Bash(git:*) Bash(jq:*) Read", skill.AllowedTools)
		assert.Equal(t, map[string]string{"author": "example-org", "version": "1.0"}, skill.Metadata)
		assert.Equal(t, "Step one.\n", skill.Body)
	})

	t.Run("NameMustMatchDirectory", func(t *testing.T) {
		content := []byte(`---
name: pdf-processing
description: Does things with PDFs.
---
`)
		_, err := ParseSkill(content, "pdf-tools")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must match the directory name")

		// an empty directory name skips the check, for content not yet on disk
		_, err = ParseSkill(content, "")
		assert.NoError(t, err)
	})

	t.Run("DescriptionIsRequired", func(t *testing.T) {
		_, err := ParseSkill([]byte(`---
name: pdf-processing
---
`), "pdf-processing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "description is required")
	})

	t.Run("DescriptionLengthIsBoundedInCharactersNotBytes", func(t *testing.T) {
		// a multi-byte description at exactly the limit is valid; one over is not
		at := strings.Repeat("é", SkillDescriptionMaxLen)
		_, err := ParseSkill([]byte("---\nname: x\ndescription: "+at+"\n---\n"), "x")
		assert.NoError(t, err)

		over := strings.Repeat("é", SkillDescriptionMaxLen+1)
		_, err = ParseSkill([]byte("---\nname: x\ndescription: "+over+"\n---\n"), "x")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at most")
	})

	t.Run("MissingFrontmatter", func(t *testing.T) {
		_, err := ParseSkill([]byte("just markdown, no frontmatter\n"), "x")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "frontmatter")
	})

	t.Run("CompatibilityIsBounded", func(t *testing.T) {
		over := strings.Repeat("a", SkillCompatibilityMaxLen+1)
		_, err := ParseSkill([]byte("---\nname: x\ndescription: d\ncompatibility: "+over+"\n---\n"), "x")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "compatibility")
	})
}
