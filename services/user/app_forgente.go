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

// SuspendOrgApps flips the kill switch for every app an organization owns and
// reports how many rows changed. Suspension leaves the account and its tokens
// in place: it is a stop button, not a delete, so access can be restored
// without reissuing credentials.
func SuspendOrgApps(ctx context.Context, org *organization.Organization, suspended bool) (int64, error) {
	return user_model.SetForgenteAppsSuspendedByOwnerID(ctx, org.ID, suspended)
}
