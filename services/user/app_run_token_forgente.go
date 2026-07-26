// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"errors"

	actions_model "forgente.com/models/actions"
	user_model "forgente.com/models/user"
	"forgente.com/modules/util"
)

// ErrForgenteAppRunTokenRefused is returned when a run may not act as the app
// it asked for. The reason is carried for the log and the response, but every
// case is the same refusal: this run has not been authorized to be this app.
type ErrForgenteAppRunTokenRefused struct {
	Reason string
}

// IsErrForgenteAppRunTokenRefused checks if an error is ErrForgenteAppRunTokenRefused
func IsErrForgenteAppRunTokenRefused(err error) bool {
	_, ok := err.(ErrForgenteAppRunTokenRefused)
	return ok
}

func (err ErrForgenteAppRunTokenRefused) Error() string {
	return "run may not act as this app: " + err.Reason
}

func (err ErrForgenteAppRunTokenRefused) Unwrap() error {
	return util.ErrPermissionDenied
}

// checkGrantRunnerLabel refuses a run that is not executing on a runner the
// organization designated for this grant.
//
// The check belongs here rather than at scheduling because scheduling is not a
// boundary anyone has to cross: a job's `runs-on` already routes it to a
// labelled runner, but whoever can edit the workflow can edit that line, and
// the app token would be minted regardless. Asking for the app's identity is
// the boundary, so the designation is checked at the point the run tries to
// cross it.
//
// It fails closed. A grant that names a label and a run whose runner cannot be
// established is refused, because the alternative — issuing the app's identity
// to a run whose location is unknown — is the thing the restriction exists to
// prevent.
func checkGrantRunnerLabel(ctx context.Context, task *actions_model.ActionTask, grant *user_model.ForgenteAppRunGrant) error {
	if grant.RunnerLabel == "" {
		return nil
	}
	if task.RunnerID == 0 {
		return ErrForgenteAppRunTokenRefused{Reason: "the run is not attached to a runner"}
	}

	runner, err := actions_model.GetRunnerByID(ctx, task.RunnerID)
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			return ErrForgenteAppRunTokenRefused{Reason: "the runner this job is on no longer exists"}
		}
		return err
	}
	if !runner.CanMatchLabels([]string{grant.RunnerLabel}) {
		return ErrForgenteAppRunTokenRefused{
			Reason: "this app may only be claimed from a runner labelled " + grant.RunnerLabel,
		}
	}
	return nil
}

// MintAppRunToken exchanges a running Actions task for a short-lived token
// belonging to an app, if the app's organization has granted that repository
// the right to act as it.
//
// The task is the authority on which repository is asking. It is loaded from
// the id the job token carried rather than taken from the request, so a run
// cannot name a repository other than its own.
func MintAppRunToken(ctx context.Context, taskID, repoID int64, appName string) (*user_model.ForgenteAppRunToken, error) {
	task, err := actions_model.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// The route says which repository this is; the task says which repository is
	// actually running. They should never disagree, and a run asking under
	// another repository's path is refused rather than quietly served against
	// its own grant.
	if task.RepoID != repoID {
		return nil, ErrForgenteAppRunTokenRefused{Reason: "the run does not belong to this repository"}
	}

	// A finished job has no business minting anything, and a job token outlives
	// its usefulness the moment the task stops.
	if task.Status != actions_model.StatusRunning {
		return nil, ErrForgenteAppRunTokenRefused{Reason: "the task is not running"}
	}

	// The decisive refusal. A fork pull request runs code that the repository's
	// own collaborators never approved, so letting one assume an app would hand
	// the organization's identity to anyone who can open a pull request.
	if task.IsForkPullRequest {
		return nil, ErrForgenteAppRunTokenRefused{Reason: "a fork pull request run cannot act as an app"}
	}

	botUser, err := user_model.GetUserByName(ctx, appName)
	if err != nil {
		return nil, err
	}
	app, err := user_model.GetForgenteAppByUserID(ctx, botUser.ID)
	if err != nil {
		return nil, err
	}
	// The kill switch is enforced in the auth path for every credential type,
	// but refusing here too means suspending an app stops new runs from picking
	// it up rather than only stopping the tokens they would have used.
	if app.Suspended {
		return nil, ErrForgenteAppRunTokenRefused{Reason: "the app is suspended"}
	}

	grant, err := user_model.GetForgenteAppRunGrant(ctx, app.ID, task.RepoID)
	if err != nil {
		if user_model.IsErrForgenteAppRunGrantNotExist(err) {
			return nil, ErrForgenteAppRunTokenRefused{Reason: "this repository has not been granted the app"}
		}
		return nil, err
	}

	if err := checkGrantRunnerLabel(ctx, task, grant); err != nil {
		return nil, err
	}

	token := &user_model.ForgenteAppRunToken{
		AppID:  app.ID,
		UserID: app.UserID,
		TaskID: task.ID,
		Scope:  grant.Scope,
	}
	if err := user_model.NewForgenteAppRunToken(ctx, token); err != nil {
		return nil, err
	}
	return token, nil
}
