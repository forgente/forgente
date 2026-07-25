// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"forgente.com/modules/markup/markdown"

	"go.yaml.in/yaml/v4"
)

// ProfilePromptMaxLen bounds the Markdown body, matching the published limit.
const ProfilePromptMaxLen = 30000

// Profile targets. A profile with no target applies to both environments.
const (
	ProfileTargetVSCode        = "vscode"
	ProfileTargetGitHubCopilot = "github-copilot"
)

// ProfileToolsAll is the wildcard entry meaning "every available tool", which
// is also what an omitted tools field means.
const ProfileToolsAll = "*"

// ProfileTools is the tools field, which may be written either as a YAML list
// or as a single comma-separated string. Both spellings are in use, so both
// parse; the distinction is not preserved because nothing depends on it.
type ProfileTools []string

// UnmarshalYAML accepts either a sequence or a comma-separated scalar.
func (t *ProfileTools) UnmarshalYAML(node *yaml.Node) error {
	var list []string
	if err := node.Decode(&list); err == nil {
		*t = list
		return nil
	}

	var joined string
	if err := node.Decode(&joined); err != nil {
		return errors.New("tools must be a list or a comma-separated string")
	}
	parsed := ProfileTools{}
	for part := range strings.SplitSeq(joined, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parsed = append(parsed, trimmed)
		}
	}
	*t = parsed
	return nil
}

// AllowsAllTools reports whether the profile leaves every tool enabled. An
// omitted field and an explicit ["*"] mean the same thing; an empty list does
// not, and disables tools entirely.
func (p *Profile) AllowsAllTools() bool {
	if p.Tools == nil {
		return true
	}
	return len(p.Tools) == 1 && p.Tools[0] == ProfileToolsAll
}

// Profile is one custom agent definition. Only Description is required; an
// omitted Name falls back to the file's identifier.
type Profile struct {
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Target      string       `yaml:"target"`
	Tools       ProfileTools `yaml:"tools"`
	Model       string       `yaml:"model"`
	// DisableModelInvocation and UserInvocable are pointers so that "absent"
	// is distinguishable from "false"; the defaults differ between them.
	DisableModelInvocation *bool             `yaml:"disable-model-invocation"`
	UserInvocable          *bool             `yaml:"user-invocable"`
	MCPServers             map[string]any    `yaml:"mcp-servers"`
	Metadata               map[string]string `yaml:"metadata"`

	// ID is the file's identifier: its name with the extension removed. It is
	// what deduplicates a profile across repository, organization and instance
	// levels, so two files with different name fields but the same filename
	// are the same profile with the lowest level winning.
	ID string `yaml:"-"`
	// Prompt is the Markdown body: the agent's instructions.
	Prompt string `yaml:"-"`
}

// DisplayName is the name to show, falling back to the file identifier.
func (p *Profile) DisplayName() string {
	if p.Name != "" {
		return p.Name
	}
	return p.ID
}

// IsUserInvocable defaults to true when the field is absent.
func (p *Profile) IsUserInvocable() bool {
	return p.UserInvocable == nil || *p.UserInvocable
}

// IsModelInvocable defaults to true: the field that turns it off defaults off.
func (p *Profile) IsModelInvocable() bool {
	return p.DisableModelInvocation == nil || !*p.DisableModelInvocation
}

// ProfileIDFromFileName strips the recognised extensions to get the identifier
// used for deduplication. Both ".agent.md" and ".md" are valid; the longer one
// is tried first so "review.agent.md" does not become "review.agent".
// An unrecognised extension yields an empty string.
func ProfileIDFromFileName(fileName string) string {
	for _, ext := range []string{".agent.md", ".md"} {
		if strings.HasSuffix(fileName, ext) {
			return strings.TrimSuffix(fileName, ext)
		}
	}
	return ""
}

// validProfileFileName reports whether a file name uses only the characters
// the cloud agent accepts.
func validProfileFileName(fileName string) bool {
	for _, r := range fileName {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '-', r == '_':
		default:
			return false
		}
	}
	return fileName != ""
}

// Validate checks a parsed profile.
func (p *Profile) Validate() error {
	if p.Description == "" {
		return errors.New("description is required")
	}
	if p.Target != "" && p.Target != ProfileTargetVSCode && p.Target != ProfileTargetGitHubCopilot {
		return fmt.Errorf("target must be %q or %q", ProfileTargetVSCode, ProfileTargetGitHubCopilot)
	}
	if utf8.RuneCountInString(p.Prompt) > ProfilePromptMaxLen {
		return fmt.Errorf("prompt must be at most %d characters", ProfilePromptMaxLen)
	}
	return nil
}

// ParseProfile reads a custom agent definition. fileName is the file's base
// name, which supplies the identifier and the display name when none is given.
func ParseProfile(contents []byte, fileName string) (*Profile, error) {
	id := ProfileIDFromFileName(fileName)
	if id == "" {
		return nil, fmt.Errorf("%q must end in .agent.md or .md", fileName)
	}
	if !validProfileFileName(fileName) {
		return nil, fmt.Errorf("%q may only contain letters, digits, %q, %q and %q", fileName, ".", "-", "_")
	}

	profile := &Profile{}
	body, err := markdown.ExtractMetadataBytes(contents, profile)
	if err != nil {
		return nil, fmt.Errorf("invalid frontmatter in %q: %w", fileName, err)
	}
	profile.ID = id
	profile.Prompt = string(body)
	if err := profile.Validate(); err != nil {
		return nil, err
	}
	return profile, nil
}
