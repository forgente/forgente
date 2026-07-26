// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Package agent discovers the agent definitions a repository carries, in the
// locations the wider ecosystem already uses. Parsing lives in
// forgente.com/modules/agent; this package only decides where to look and what
// wins when the same definition appears twice.
package agent

import (
	"context"
	"path"

	repo_model "forgente.com/models/repo"
	agent_module "forgente.com/modules/agent"
	"forgente.com/modules/git"
	"forgente.com/modules/log"
	"forgente.com/modules/util"
)

// definitionSizeLimit caps how much of a definition file is read. Skills are
// meant to stay well under this and profile prompts are capped at 30,000
// characters, so anything approaching it is malformed rather than ambitious.
const definitionSizeLimit = 1024 * 1024

// SkillDirCandidates are searched in order, and the first directory to define a
// given skill name wins.
//
// GitHub reads all three. It publishes no precedence between them, so the order
// here is Forgente's choice: `.agents/skills` first because it is the
// vendor-neutral location, and the one the roadmap commits to.
var SkillDirCandidates = []string{
	".agents/skills",
	".github/skills",
	".claude/skills",
}

// ProfileDirCandidates are searched in order. Unlike the skill locations this
// order *is* specified: `.github/agents` takes precedence over `.claude/agents`
// at the same level.
var ProfileDirCandidates = []string{
	".github/agents",
	".claude/agents",
}

// Definitions is what a repository declares. A malformed definition is recorded
// in Errors rather than dropped, so a repository owner can be told why theirs
// did not load instead of finding it silently absent.
type Definitions struct {
	Skills   []*agent_module.Skill
	Profiles []*agent_module.Profile
	Errors   map[string]error
}

func (d *Definitions) addError(fullPath string, err error) {
	if d.Errors == nil {
		d.Errors = make(map[string]error)
	}
	d.Errors[fullPath] = err
}

// IsEmpty reports whether the repository declared nothing at all, errors
// included.
func (d *Definitions) IsEmpty() bool {
	return len(d.Skills) == 0 && len(d.Profiles) == 0 && len(d.Errors) == 0
}

// readBlob reads one file from the commit, or returns nil if it is absent.
func readBlob(ctx context.Context, gitRepo *git.Repository, commit *git.Commit, filePath string) ([]byte, error) {
	entry, err := commit.GetTreeEntryByPath(ctx, gitRepo, filePath)
	if err != nil {
		return nil, nil // absent, which is not an error worth reporting
	}
	reader, err := entry.Blob(gitRepo).DataAsync(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return util.ReadWithLimit(reader, definitionSizeLimit)
}

// LoadFromDefaultBranch reads every agent definition the repository declares on
// its default branch. It never returns nil.
func LoadFromDefaultBranch(ctx context.Context, repo *repo_model.Repository, gitRepo *git.Repository) *Definitions {
	found := &Definitions{}
	if repo == nil || gitRepo == nil || repo.IsEmpty {
		return found
	}

	commit, err := gitRepo.GetBranchCommit(ctx, repo.DefaultBranch)
	if err != nil {
		log.Debug("agent discovery: get commit for %s: %v", repo.FullName(), err)
		return found
	}

	loadSkills(ctx, gitRepo, commit, found)
	loadProfiles(ctx, gitRepo, commit, found)
	return found
}

// loadSkills walks the skill directories. A skill is a directory containing
// SKILL.md, so this descends one level further than the profile search.
func loadSkills(ctx context.Context, gitRepo *git.Repository, commit *git.Commit, found *Definitions) {
	seen := make(map[string]bool)

	for _, dirName := range SkillDirCandidates {
		tree, err := commit.SubTree(ctx, gitRepo, dirName)
		if err != nil {
			continue // directory absent
		}
		entries, err := tree.ListEntries(ctx, gitRepo)
		if err != nil {
			log.Debug("agent discovery: list %s: %v", dirName, err)
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillDir := entry.Name()
			fullPath := path.Join(dirName, skillDir, agent_module.SkillFileName)

			contents, err := readBlob(ctx, gitRepo, commit, fullPath)
			if err != nil {
				found.addError(fullPath, err)
				continue
			}
			if contents == nil {
				continue // a directory without SKILL.md is simply not a skill
			}

			skill, err := agent_module.ParseSkill(contents, skillDir)
			if err != nil {
				found.addError(fullPath, err)
				continue
			}
			// an earlier candidate directory already defined this name
			if seen[skill.Name] {
				continue
			}
			seen[skill.Name] = true
			found.Skills = append(found.Skills, skill)
		}
	}
}

// loadProfiles walks the custom agent directories.
func loadProfiles(ctx context.Context, gitRepo *git.Repository, commit *git.Commit, found *Definitions) {
	seen := make(map[string]bool)

	for _, dirName := range ProfileDirCandidates {
		tree, err := commit.SubTree(ctx, gitRepo, dirName)
		if err != nil {
			continue
		}
		entries, err := tree.ListEntries(ctx, gitRepo)
		if err != nil {
			log.Debug("agent discovery: list %s: %v", dirName, err)
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fileName := entry.Name()
			// skip anything that is not a profile before reading it
			if agent_module.ProfileIDFromFileName(fileName) == "" {
				continue
			}
			fullPath := path.Join(dirName, fileName)

			contents, err := readBlob(ctx, gitRepo, commit, fullPath)
			if err != nil {
				found.addError(fullPath, err)
				continue
			}
			if contents == nil {
				continue
			}

			profile, err := agent_module.ParseProfile(contents, fileName)
			if err != nil {
				found.addError(fullPath, err)
				continue
			}
			// deduplication keys on the identifier, not the name field
			if seen[profile.ID] {
				continue
			}
			seen[profile.ID] = true
			found.Profiles = append(found.Profiles, profile)
		}
	}
}
