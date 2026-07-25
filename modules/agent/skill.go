// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Package agent reads the agent definitions a repository carries alongside its
// code. The formats are not Forgente's: they are the open Agent Skills
// specification (https://agentskills.io/specification) and, later, the custom
// agent profiles the same ecosystem uses. Parsing them rather than inventing a
// format is the point — a repository configured for one agent product works
// here without being rewritten.
package agent

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"forgente.com/modules/markup/markdown"
)

// Limits from the Agent Skills specification. They are counted in characters
// rather than bytes, which only matters for description and compatibility since
// name is restricted to ASCII.
const (
	SkillNameMaxLen          = 64
	SkillDescriptionMaxLen   = 1024
	SkillCompatibilityMaxLen = 500
)

// SkillFileName is the file every skill directory must contain.
const SkillFileName = "SKILL.md"

// Skill is one parsed SKILL.md. Only Name and Description are required by the
// specification; everything else is optional and may be zero.
type Skill struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license"`
	Compatibility string            `yaml:"compatibility"`
	Metadata      map[string]string `yaml:"metadata"`
	// AllowedTools is a space-separated list. The specification marks it
	// experimental and support varies between agents, so it is carried through
	// verbatim rather than parsed into a list.
	AllowedTools string `yaml:"allowed-tools"`

	// Body is the Markdown after the frontmatter: the skill's instructions.
	Body string `yaml:"-"`
}

// ValidateSkillName reports whether a name satisfies the specification: 1-64
// characters, lowercase alphanumerics and hyphens only, no leading, trailing or
// consecutive hyphens.
func ValidateSkillName(name string) error {
	if name == "" {
		return errors.New("name is required")
	}
	if utf8.RuneCountInString(name) > SkillNameMaxLen {
		return fmt.Errorf("name must be at most %d characters", SkillNameMaxLen)
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return fmt.Errorf("name may only contain lowercase letters, digits and hyphens, found %q", r)
		}
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return errors.New("name must not start or end with a hyphen")
	}
	if strings.Contains(name, "--") {
		return errors.New("name must not contain consecutive hyphens")
	}
	return nil
}

// Validate checks a parsed skill against the specification. dirName is the name
// of the directory holding the SKILL.md; the specification requires the name
// field to match it, which is what keeps a skill's identity the same whether an
// agent found it by directory listing or by reading the file.
func (s *Skill) Validate(dirName string) error {
	if err := ValidateSkillName(s.Name); err != nil {
		return err
	}
	if dirName != "" && s.Name != dirName {
		return fmt.Errorf("name %q must match the directory name %q", s.Name, dirName)
	}
	if s.Description == "" {
		return errors.New("description is required")
	}
	if utf8.RuneCountInString(s.Description) > SkillDescriptionMaxLen {
		return fmt.Errorf("description must be at most %d characters", SkillDescriptionMaxLen)
	}
	if utf8.RuneCountInString(s.Compatibility) > SkillCompatibilityMaxLen {
		return fmt.Errorf("compatibility must be at most %d characters", SkillCompatibilityMaxLen)
	}
	return nil
}

// ParseSkill reads a SKILL.md and validates it against the specification.
// dirName is the skill directory's name; pass "" to skip the directory match,
// which is useful when validating content that is not yet on disk.
func ParseSkill(contents []byte, dirName string) (*Skill, error) {
	skill := &Skill{}
	body, err := markdown.ExtractMetadataBytes(contents, skill)
	if err != nil {
		return nil, fmt.Errorf("invalid %s frontmatter: %w", SkillFileName, err)
	}
	skill.Body = string(body)
	if err := skill.Validate(dirName); err != nil {
		return nil, err
	}
	return skill, nil
}
