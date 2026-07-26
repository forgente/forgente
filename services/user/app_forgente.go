// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"fmt"
	"strings"

	"forgente.com/models/db"
	"forgente.com/models/organization"
	user_model "forgente.com/models/user"
	"forgente.com/modules/optional"
	"forgente.com/modules/setting"
	"forgente.com/modules/structs"
	"forgente.com/modules/util"
	org_service "forgente.com/services/org"
)

// maxAppsPerOrg bounds how many apps one organization may own. Apps are cheap
// to create and each one is a principal that can be granted access, so the
// limit exists to keep a compromised or careless owner from minting an
// unbounded number of them.
const maxAppsPerOrg = 50

// ErrForgenteAppLimitReached is returned when an organization is at its app limit.
type ErrForgenteAppLimitReached struct {
	OwnerID int64
	Limit   int64
}

// IsErrForgenteAppLimitReached checks if an error is ErrForgenteAppLimitReached
func IsErrForgenteAppLimitReached(err error) bool {
	_, ok := err.(ErrForgenteAppLimitReached)
	return ok
}

func (err ErrForgenteAppLimitReached) Error() string {
	return fmt.Sprintf("organization has reached its app limit [owner id: %d, limit: %d]", err.OwnerID, err.Limit)
}

func (err ErrForgenteAppLimitReached) Unwrap() error {
	return util.ErrPermissionDenied
}

// CreateOrgApp creates an organization-owned app: a UserTypeBot account plus
// the ownership row recording which organization administers it, in one
// transaction.
//
// The account is created without a password and is never linked to an
// authentication source — it acts only through access tokens, which the
// existing sign-in paths already enforce for non-individual users. Creating
// the app grants it no access to anything; like any other principal it is
// added to teams or made a collaborator afterwards.
func CreateOrgApp(ctx context.Context, doer *user_model.User, org *organization.Organization, name, description string) (*user_model.User, *user_model.ForgenteApp, error) {
	if err := user_model.IsUsableUsername(name); err != nil {
		return nil, nil, err
	}

	var botUser *user_model.User
	var app *user_model.ForgenteApp

	err := db.WithTx(ctx, func(ctx context.Context) error {
		count, err := user_model.CountForgenteAppsByOwnerID(ctx, org.ID)
		if err != nil {
			return err
		}
		if count >= maxAppsPerOrg {
			return ErrForgenteAppLimitReached{OwnerID: org.ID, Limit: maxAppsPerOrg}
		}

		botUser = &user_model.User{
			Name:      name,
			LowerName: strings.ToLower(name),
			// Apps never receive mail; the no-reply domain keeps the address
			// unique and undeliverable without inventing a real mailbox.
			Email:      fmt.Sprintf("%s@%s", strings.ToLower(name), setting.Service.NoReplyAddress),
			Type:       user_model.UserTypeBot,
			FullName:   name,
			IsActive:   true,
			Visibility: structs.VisibleTypePublic,
		}

		if err := user_model.CreateUser(ctx, botUser, &user_model.Meta{}, &user_model.CreateUserOverwriteOptions{
			IsActive: optional.Some(true),
			// An app is a principal, not an account holder: it cannot create
			// organizations or repositories of its own, only act where it has
			// been granted access.
			AllowCreateOrganization: optional.Some(false),
			KeepEmailPrivate:        optional.Some(true),
		}); err != nil {
			return err
		}

		app = &user_model.ForgenteApp{
			UserID:      botUser.ID,
			OwnerID:     org.ID,
			CreatorID:   doer.ID,
			Description: description,
		}
		return db.Insert(ctx, app)
	})
	if err != nil {
		return nil, nil, err
	}

	return botUser, app, nil
}

// GetOrgApp resolves one of an organization's apps together with the account it
// acts as.
//
// An app belonging to another organization is reported as missing rather than
// forbidden, so app IDs cannot be probed across organizations.
func GetOrgApp(ctx context.Context, org *organization.Organization, id int64) (*user_model.ForgenteApp, *user_model.User, error) {
	app, err := user_model.GetForgenteAppByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if app.OwnerID != org.ID {
		return nil, nil, user_model.ErrForgenteAppNotExist{ID: id}
	}

	botUser, err := user_model.GetUserByID(ctx, app.UserID)
	if err != nil {
		return nil, nil, err
	}
	return app, botUser, nil
}

// DeleteOrgApp removes an app and the account behind it.
func DeleteOrgApp(ctx context.Context, org *organization.Organization, app *user_model.ForgenteApp, botUser *user_model.User) error {
	if app.UserID != botUser.ID || !botUser.IsTypeBot() {
		return fmt.Errorf("user %s[%d] is not the account of app %d", botUser.Name, botUser.ID, app.ID)
	}

	// Suspend first: deleting an account touches many tables and can fail
	// partway, and a suspended app cannot act while that is unresolved.
	if err := user_model.SetForgenteAppSuspended(ctx, app.ID, true); err != nil {
		return err
	}

	// An app granted access through a team is a member of the organization, and
	// DeleteUser refuses to delete a user that still belongs to one. Removing it
	// here rather than purging keeps the failure modes honest: RemoveOrgUser
	// reports the last-owner case instead of deleting the organization.
	if err := org_service.RemoveOrgUser(ctx, org, botUser); err != nil {
		return err
	}

	// Run grants key on the app rather than on its account, so deleting the
	// account does not take them with it. A left-behind grant would authorize
	// the next app to reuse this id.
	if err := user_model.DeleteForgenteAppRunGrantsByAppID(ctx, app.ID); err != nil {
		return err
	}

	// Deleting the account drops the ownership row with it.
	return DeleteUser(ctx, botUser, false)
}

// SuspendOrgApps flips the kill switch for every app an organization owns and
// reports how many rows changed. Suspension leaves the account and its tokens
// in place: it is a stop button, not a delete, so access can be restored
// without reissuing credentials.
func SuspendOrgApps(ctx context.Context, org *organization.Organization, suspended bool) (int64, error) {
	return user_model.SetForgenteAppsSuspendedByOwnerID(ctx, org.ID, suspended)
}
