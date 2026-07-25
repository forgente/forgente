// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"fmt"

	"forgente.com/models/db"
	"forgente.com/modules/timeutil"
	"forgente.com/modules/util"
)

// ForgenteApp gives an organization an owned principal: a UserTypeBot account
// the organization creates, permissions through ordinary team membership, and
// can suspend in one action. It is the installation-identity primitive —
// unlike an OAuth2 application, which acts as the user that authorized it, an
// app acts as itself and its actions are attributed to it.
//
// The bot account carries the identity (name, avatar, tokens, permissions);
// this row records who owns it and whether it is currently allowed to act.
type ForgenteApp struct {
	ID int64 `xorm:"pk autoincr"`
	// UserID is the UserTypeBot account the app acts as. One account per app.
	UserID int64 `xorm:"UNIQUE NOT NULL"`
	// OwnerID is the organization the app belongs to. Its owners administer
	// the app; the app's permissions are separate and granted like anyone's.
	// User-owned apps are a later extension and need no schema change here —
	// only the creation path and the UI decide what may be an owner.
	OwnerID int64 `xorm:"INDEX NOT NULL"`
	// CreatorID only records who created the app, so it is kept on user
	// deletion — the app's validity does not depend on the creator existing.
	CreatorID   int64 `xorm:"NOT NULL"`
	Description string
	// Suspended is the kill switch. Token auth resolves a token straight to its
	// user without consulting user activity state, so this is enforced
	// explicitly in the auth path rather than by deactivating the account.
	Suspended   bool               `xorm:"NOT NULL DEFAULT false"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	// the table is created by SyncAllTables at startup, no migration needed
	db.RegisterModel(new(ForgenteApp))
}

// ErrForgenteAppNotExist represents a "app does not exist" error
type ErrForgenteAppNotExist struct {
	ID     int64
	UserID int64
}

// IsErrForgenteAppNotExist checks if an error is ErrForgenteAppNotExist
func IsErrForgenteAppNotExist(err error) bool {
	_, ok := err.(ErrForgenteAppNotExist)
	return ok
}

func (err ErrForgenteAppNotExist) Error() string {
	return fmt.Sprintf("app does not exist [id: %d, user id: %d]", err.ID, err.UserID)
}

func (err ErrForgenteAppNotExist) Unwrap() error {
	return util.ErrNotExist
}

// GetForgenteAppByUserID returns the app backed by the given bot account.
func GetForgenteAppByUserID(ctx context.Context, userID int64) (*ForgenteApp, error) {
	app := new(ForgenteApp)
	has, err := db.GetEngine(ctx).Where("user_id = ?", userID).Get(app)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrForgenteAppNotExist{UserID: userID}
	}
	return app, nil
}

// GetForgenteAppByID returns an app by its own ID.
func GetForgenteAppByID(ctx context.Context, id int64) (*ForgenteApp, error) {
	app := new(ForgenteApp)
	has, err := db.GetEngine(ctx).ID(id).Get(app)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrForgenteAppNotExist{ID: id}
	}
	return app, nil
}

// ListForgenteAppsByOwnerID returns every app owned by an organization, oldest first.
func ListForgenteAppsByOwnerID(ctx context.Context, ownerID int64) ([]*ForgenteApp, error) {
	apps := make([]*ForgenteApp, 0, 8)
	return apps, db.GetEngine(ctx).Where("owner_id = ?", ownerID).Asc("id").Find(&apps)
}

// CountForgenteAppsByOwnerID returns how many apps an organization owns, for
// enforcing the per-organization limit at creation time.
func CountForgenteAppsByOwnerID(ctx context.Context, ownerID int64) (int64, error) {
	return db.GetEngine(ctx).Where("owner_id = ?", ownerID).Count(new(ForgenteApp))
}

// IsForgenteAppSuspended reports whether the given account is a suspended app.
// A non-app account is never suspended by this check, so the auth path can call
// it for any token holder.
func IsForgenteAppSuspended(ctx context.Context, userID int64) (bool, error) {
	return db.GetEngine(ctx).Where("user_id = ? AND suspended = ?", userID, true).Exist(new(ForgenteApp))
}

// SetForgenteAppSuspended flips the kill switch for a single app.
func SetForgenteAppSuspended(ctx context.Context, id int64, suspended bool) error {
	_, err := db.GetEngine(ctx).ID(id).Cols("suspended").Update(&ForgenteApp{Suspended: suspended})
	return err
}

// SetForgenteAppsSuspendedByOwnerID flips the kill switch for every app an
// organization owns — the "suspend all agents" action.
func SetForgenteAppsSuspendedByOwnerID(ctx context.Context, ownerID int64, suspended bool) (int64, error) {
	return db.GetEngine(ctx).Where("owner_id = ?", ownerID).Cols("suspended").Update(&ForgenteApp{Suspended: suspended})
}
