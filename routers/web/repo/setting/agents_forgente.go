// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"net/http"
	"sort"

	"forgente.com/modules/log"
	"forgente.com/modules/templates"
	agent_service "forgente.com/services/agent"
	"forgente.com/services/context"
)

const tplSettingsAgents templates.TplName = "repo/settings/agents_forgente"

// agentDefinitionError pairs a path with why its definition was rejected. The
// template needs a sorted slice rather than a map so the order is stable.
type agentDefinitionError struct {
	Path   string
	Reason string
}

// Agents lists the agent definitions the repository declares, including the
// ones that failed to load. The failures are the point: a definition that is
// silently absent is the hardest kind to debug.
func Agents(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.settings.agents")
	ctx.Data["PageIsSettingsAgents"] = true
	ctx.Data["SkillDirCandidates"] = agent_service.SkillDirCandidates
	ctx.Data["ProfileDirCandidates"] = agent_service.ProfileDirCandidates

	gitRepo := ctx.Repo.GitRepo
	if ctx.Repo.Repository.IsEmpty || gitRepo == nil {
		ctx.Data["AgentDefinitions"] = &agent_service.Definitions{}
		ctx.Data["AgentDefinitionErrors"] = []agentDefinitionError{}
		ctx.HTML(http.StatusOK, tplSettingsAgents)
		return
	}

	found := agent_service.LoadFromDefaultBranch(ctx, ctx.Repo.Repository, gitRepo)
	ctx.Data["AgentDefinitions"] = found

	errs := make([]agentDefinitionError, 0, len(found.Errors))
	for path, err := range found.Errors {
		errs = append(errs, agentDefinitionError{Path: path, Reason: err.Error()})
	}
	sort.Slice(errs, func(i, j int) bool { return errs[i].Path < errs[j].Path })
	ctx.Data["AgentDefinitionErrors"] = errs

	log.Trace("agent definitions for %s: %d skills, %d profiles, %d errors",
		ctx.Repo.Repository.FullName(), len(found.Skills), len(found.Profiles), len(errs))

	ctx.HTML(http.StatusOK, tplSettingsAgents)
}
