// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package incoming

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	issues_model "forgente.com/models/issues"
	access_model "forgente.com/models/perm/access"
	"forgente.com/models/unit"
	user_model "forgente.com/models/user"
	"forgente.com/modules/log"
	"forgente.com/modules/util"
	issue_service "forgente.com/services/issue"
	incoming_payload "forgente.com/services/mailer/incoming/payload"
	"forgente.com/services/mailer/token"
)

// init registers ForgenteNewIssueHandler additively, without touching the upstream
// `handlers` map literal in incoming_handler.go.
func init() {
	handlers[token.ForgenteNewIssueHandlerType] = &ForgenteNewIssueHandler{}
}

// ForgenteNewIssueHandler handles incoming emails that create a new issue in a repository,
// closing the "create issue via email" gap (upstream go-gitea/gitea#6226).
type ForgenteNewIssueHandler struct{}

func (h *ForgenteNewIssueHandler) Handle(ctx context.Context, content *MailContent, doer *user_model.User, payload []byte) error {
	if doer == nil {
		return util.NewInvalidArgumentErrorf("doer can't be nil")
	}

	repo, err := incoming_payload.GetForgenteRepoFromPayload(ctx, payload)
	if err != nil {
		return err
	}

	if repo.IsArchived || !repo.UnitEnabled(ctx, unit.TypeIssues) {
		log.Debug("can't create issue via mail: repo archived or issues unit disabled")
		return nil
	}

	perm, err := access_model.GetDoerRepoPermission(ctx, repo, doer)
	if err != nil {
		return err
	}

	// Deliberately stricter than the web UI (which lets readers open issues): the address is
	// only handed out to writers, so mail arriving without write access is treated as stale.
	if !perm.CanWriteIssuesOrPulls(false) {
		log.Debug("can't create issue via mail: doer lacks write access to issues")
		return nil
	}

	// Collapse whitespace and strip control chars: RFC 2047 subjects can decode to multi-line strings.
	title := strings.Join(strings.FieldsFunc(content.Subject, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}), " ")
	if title == "" {
		// Log-and-drop instead of returning an error so the message still counts as handled
		// and gets expunged; an error here would strand the mail in the mailbox forever.
		log.Error("can't create issue via mail: empty subject, refusing to create a titleless issue")
		return nil
	}

	issue := &issues_model.Issue{
		RepoID:   repo.ID,
		Repo:     repo,
		Title:    title,
		PosterID: doer.ID,
		Poster:   doer,
		Content:  content.Content,
	}

	// Attachment support is deferred: ReplyHandler's upload path assumes an existing issue/comment
	// target and reusing it here would meaningfully grow this slice; noted in the PR description.
	if err := issue_service.NewIssue(ctx, repo, issue, nil, nil, nil, nil); err != nil {
		return fmt.Errorf("NewIssue failed: %w", err)
	}

	return nil
}
