// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"context"

	issues_model "forgente.com/models/issues"
	user_model "forgente.com/models/user"
	"forgente.com/modules/log"
	notify_service "forgente.com/services/notify"
)

type agentNotifier struct {
	notify_service.NullNotifier
}

var _ notify_service.Notifier = &agentNotifier{}

// NewNotifier records agent work as issues are assigned and unassigned.
//
// Assignment is deliberately the whole trigger. Bot accounts are already
// assignable and assignment already emits an Actions event, so subscribing
// here adds the record without inventing a second way to ask an agent for
// something.
func NewNotifier() notify_service.Notifier {
	return &agentNotifier{}
}

func (n *agentNotifier) IssueChangeAssignee(ctx context.Context, doer *user_model.User, issue *issues_model.Issue, assignee *user_model.User, removed bool, _ *issues_model.Comment) {
	app := appForAssignee(ctx, assignee)
	if app == nil {
		return
	}

	if removed {
		if err := CancelTaskForIssue(ctx, issue.ID, app.ID); err != nil {
			log.Error("agent: cancel task for issue %d app %d: %v", issue.ID, app.ID, err)
		}
		return
	}

	if _, err := StartTaskForIssue(ctx, doer, issue, app); err != nil {
		log.Error("agent: start task for issue %d app %d: %v", issue.ID, app.ID, err)
	}
}

// Init registers the notifier. Called from routers/init.go, which is also what
// links this package into the binary.
func Init() error {
	notify_service.RegisterNotifier(NewNotifier())
	return nil
}
