// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	"forgente.com/models/unit"
	"forgente.com/modules/optional"
	"forgente.com/services/context"
	issue_service "forgente.com/services/issue"
)

// IssueSuggestions returns a list of issue suggestions
func IssueSuggestions(ctx *context.Context) {
	keyword := ctx.Req.FormValue("q")

	canReadIssues := ctx.Repo.Permission.CanRead(unit.TypeIssues)
	canReadPulls := ctx.Repo.Permission.CanRead(unit.TypePullRequests)

	var isPull optional.Option[bool]
	if canReadPulls && !canReadIssues {
		isPull = optional.Some(true)
	} else if canReadIssues && !canReadPulls {
		isPull = optional.Some(false)
	}

	suggestions, err := issue_service.GetSuggestion(ctx, ctx.Repo.Repository, isPull, keyword)
	if err != nil {
		ctx.ServerError("GetSuggestion", err)
		return
	}

	ctx.JSON(http.StatusOK, suggestions)
}
