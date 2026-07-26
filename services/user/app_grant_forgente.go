// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"fmt"

	auth_model "forgente.com/models/auth"
	"forgente.com/models/db"
	"forgente.com/models/organization"
	repo_model "forgente.com/models/repo"
	user_model "forgente.com/models/user"
	"forgente.com/modules/util"
)

// ErrForgenteAppRunGrantScope is returned when a grant is asked for with a
// scope that grants no API permission at all.
type ErrForgenteAppRunGrantScope struct {
	Scope auth_model.AccessTokenScope
}

// IsErrForgenteAppRunGrantScope checks if an error is ErrForgenteAppRunGrantScope
func IsErrForgenteAppRunGrantScope(err error) bool {
	_, ok := err.(ErrForgenteAppRunGrantScope)
	return ok
}

func (err ErrForgenteAppRunGrantScope) Error() string {
	return fmt.Sprintf("run grant scope %q carries no permission", err.Scope)
}

func (err ErrForgenteAppRunGrantScope) Unwrap() error {
	return util.ErrInvalidArgument
}

// GrantAppToRepoRuns authorizes a repository's Actions runs to act as one of
// the organization's apps, or every repository it owns when repoID is zero.
// Granting again for the same pair replaces the scope, so the settings page can
// offer one control rather than a revoke-then-grant dance.
//
// Two things are refused rather than validated leniently, because both would
// hand the app's identity to people the organization did not choose:
//
//   - A repository the organization does not own. Whoever can land a workflow
//     there could then act as the app, and that is not the organization's call
//     to make on another owner's behalf.
//   - A scope carrying no permission. An empty scope reads like "harmless" and
//     would be the second time this lineage treated "unspecified" as "all" —
//     see the OAuth2 grant default corrected during L1.
//
// runnerLabel is optional and restricts which runners may claim the app under
// this grant. It is not validated against the runners that exist today: a
// designated runner may legitimately be registered after the policy is set,
// and refusing the grant until then would invert the order operators work in.
// The settings page shows which runners currently match so the mismatch is
// visible rather than silent.
func GrantAppToRepoRuns(ctx context.Context, doer *user_model.User, org *organization.Organization, appID, repoID int64, scope auth_model.AccessTokenScope, runnerLabel string) (*user_model.ForgenteAppRunGrant, error) {
	normalized, err := scope.Normalize()
	if err != nil {
		return nil, err
	}
	if hasPermission := normalized.HasPermissionScope(); !hasPermission {
		return nil, ErrForgenteAppRunGrantScope{Scope: scope}
	}

	var grant *user_model.ForgenteAppRunGrant
	err = db.WithTx(ctx, func(ctx context.Context) error {
		// resolves through the organization, so an app belonging to another one
		// reports missing rather than forbidden
		app, _, err := GetOrgApp(ctx, org, appID)
		if err != nil {
			return err
		}

		if repoID != 0 {
			repo, err := repo_model.GetRepositoryByID(ctx, repoID)
			if err != nil {
				return err
			}
			if repo.OwnerID != org.ID {
				return fmt.Errorf("repository %d is not owned by organization %d: %w", repoID, org.ID, util.ErrPermissionDenied)
			}
		}

		existing, err := user_model.GetForgenteAppRunGrant(ctx, app.ID, repoID)
		switch {
		case err == nil && existing.RepoID == repoID:
			existing.Scope = normalized
			existing.RunnerLabel = runnerLabel
			existing.GrantedByID = doer.ID
			grant = existing
			return user_model.UpdateForgenteAppRunGrantScope(ctx, existing)
		case err != nil && !user_model.IsErrForgenteAppRunGrantNotExist(err):
			return err
		}

		grant = &user_model.ForgenteAppRunGrant{
			AppID:       app.ID,
			RepoID:      repoID,
			Scope:       normalized,
			RunnerLabel: runnerLabel,
			GrantedByID: doer.ID,
		}
		return db.Insert(ctx, grant)
	})
	if err != nil {
		return nil, err
	}
	return grant, nil
}

// RevokeAppRunGrant withdraws one grant. Runs already holding a minted token
// keep it until it expires — revocation stops the next exchange, and suspending
// the app is what stops one already in flight.
func RevokeAppRunGrant(ctx context.Context, org *organization.Organization, appID, grantID int64) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		app, _, err := GetOrgApp(ctx, org, appID)
		if err != nil {
			return err
		}
		return user_model.DeleteForgenteAppRunGrant(ctx, app.ID, grantID)
	})
}
