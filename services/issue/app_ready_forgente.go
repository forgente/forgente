// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"context"
	"errors"

	issues_model "forgente.com/models/issues"
	user_model "forgente.com/models/user"
	"forgente.com/modules/log"
)

// ErrAppCannotReadyOwnPull is returned when an app tries to take its own pull
// request out of draft.
var ErrAppCannotReadyOwnPull = errors.New("an app cannot mark a pull request it opened ready for review")

// checkAppNotReadyingOwnPull refuses an app promoting its own draft pull
// request to ready for review.
//
// Marking work ready is the moment it enters human review, and the author of
// the work is the one principal who should not get to decide it has arrived
// there. That holds however the agent is run, so it is a property of the forge
// rather than of any particular harness.
//
// The rule is narrow on purpose: it blocks removing the prefix, not opening a
// pull request without one. Apps serve ordinary automation too, and a release
// bot that opens finished pull requests is doing nothing wrong. An operator who
// wants the stronger guarantee has the agent open its work as a draft, and the
// forge then holds the promotion for a person. What it cannot stop is an app
// opening a second, non-draft pull request — but that is a visible new pull
// request, not a silent promotion of this one.
//
// Like the merge rule this is not configurable, for the same reason: a switch
// to turn it off is the whole guarantee.
func checkAppNotReadyingOwnPull(ctx context.Context, doer *user_model.User, issue *issues_model.Issue, oldTitle, newTitle string) error {
	if doer == nil || issue == nil || !issue.IsPull || !issue.IsPoster(doer.ID) {
		return nil
	}
	// only the draft -> ready transition is a promotion; renaming a draft, or
	// marking finished work as a draft again, is the app's own business
	if !issues_model.HasWorkInProgressPrefix(oldTitle) || issues_model.HasWorkInProgressPrefix(newTitle) {
		return nil
	}

	isApp, err := user_model.IsForgenteApp(ctx, doer)
	if err != nil {
		log.Error("resolve app for title doer %d: %v", doer.ID, err)
		return err
	}
	if !isApp {
		return nil
	}
	return ErrAppCannotReadyOwnPull
}
