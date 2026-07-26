// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	"context"
	"errors"

	issues_model "forgente.com/models/issues"
	user_model "forgente.com/models/user"
	"forgente.com/modules/log"
)

// ErrAppCannotMergeOwnPull is returned when an organization-owned app tries to
// merge a pull request it opened.
var ErrAppCannotMergeOwnPull = errors.New("an app cannot merge a pull request it opened")

// checkAppNotMergingOwnPull refuses an app merging its own pull request.
//
// Approving your own work is already refused for everyone, on both the web and
// API paths. Merging it is the same conflict of interest with a larger blast
// radius: an agent that can open a pull request and then merge it has removed
// the human from the loop entirely, which is the one thing that makes running
// agents against a real repository defensible.
//
// This is deliberately not configurable. A setting to switch it off is the
// whole guarantee, and an operator who wants an app to merge unattended can
// give a second app the job — which at least leaves two principals and an
// audit trail rather than one account approving itself.
//
// It is scoped to apps rather than to every bot account. Other bots are
// somebody else's automation, governed by whatever their operator decided;
// apps are the principal this forge issues and vouches for.
func checkAppNotMergingOwnPull(ctx context.Context, doer *user_model.User, issue *issues_model.Issue) error {
	if doer == nil || issue == nil || !issue.IsPoster(doer.ID) {
		return nil
	}

	isApp, err := user_model.IsForgenteApp(ctx, doer)
	if err != nil {
		log.Error("resolve app for merge doer %d: %v", doer.ID, err)
		return err
	}
	if !isApp {
		return nil
	}
	return ErrAppCannotMergeOwnPull
}
