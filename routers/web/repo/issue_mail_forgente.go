// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"strings"

	"forgente.com/modules/log"
	"forgente.com/modules/setting"
	"forgente.com/services/context"
	incoming_payload "forgente.com/services/mailer/incoming/payload"
	"forgente.com/services/mailer/token"
)

// forgentePrepareNewIssueMailAddress exposes a "New issue via email" mailto address on the issue
// list page when incoming mail processing is enabled, closing the "create issue via email" gap
// (upstream go-gitea/gitea#6226). The address embeds a per-user, per-repo token so mail delivered
// there is attributed and permission-checked the same way ReplyHandler handles mail replies.
func forgentePrepareNewIssueMailAddress(ctx *context.Context) {
	if !setting.IncomingEmail.Enabled || !ctx.IsSigned || ctx.Repo.Repository.IsArchived {
		return
	}
	if !ctx.Repo.Permission.CanWriteIssuesOrPulls(false) {
		return
	}

	payload, err := incoming_payload.CreateForgenteNewIssuePayload(ctx.Repo.Repository)
	if err != nil {
		log.Error("CreateForgenteNewIssuePayload: %v", err)
		return
	}

	tok, err := token.CreateToken(token.ForgenteNewIssueHandlerType, ctx.Doer, payload)
	if err != nil {
		log.Error("CreateToken: %v", err)
		return
	}

	ctx.Data["ForgenteNewIssueMailAddress"] = strings.Replace(setting.IncomingEmail.ReplyToAddress, setting.IncomingEmailTokenPlaceholder, tok, 1)
}
