// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"fmt"

	"forgente.com/models/auth"
	"forgente.com/models/db"
	"forgente.com/modules/timeutil"
	"forgente.com/modules/util"
)

// ForgenteAppRunGrant authorizes a repository's Actions runs to act as an app.
//
// Without it the only way to give a workflow an app's identity is to paste a
// long-lived token into a repository secret, which is the credential shape the
// app primitive exists to remove. The grant replaces that with an
// authorization the organization makes once and can withdraw in one place: the
// run presents the job token the forge already issued it, and the forge — which
// dispatched that run and therefore knows which repository it belongs to —
// exchanges it for a short-lived token belonging to the app.
//
// This is deliberately not how GitHub does it. There a third-party App proves
// itself with a private key the repository stores as a secret, because the App
// is external to the repository and the forge has nothing else to go on. An
// app here is owned by the organization and the run is one the forge started,
// so there is no key to distribute, store or leak.
//
// Granting is a real escalation and should be read as one: anyone who can land
// a workflow in the repository can act as the app, within the scope below.
type ForgenteAppRunGrant struct {
	ID int64 `xorm:"pk autoincr"`
	// AppID is the app whose identity is being lent out.
	AppID int64 `xorm:"UNIQUE(app_repo) INDEX NOT NULL"`
	// RepoID is the repository whose runs may assume the app. Zero means every
	// repository the owning organization has, including ones created later —
	// the common case is an app that serves its whole organization, and
	// enumerating repositories would mean re-granting on every new one.
	RepoID int64 `xorm:"UNIQUE(app_repo) NOT NULL"`
	// Scope is the ceiling for a token minted under this grant. It is required
	// rather than defaulted, because a default of "everything" is the mistake
	// this lineage already made once: an OAuth2 grant carrying no API scope was
	// treated as full access, and nothing advertised that narrower scopes
	// existed. A grant that cannot say what it is for should not be created.
	Scope auth.AccessTokenScope `xorm:"NOT NULL"`
	// GrantedByID records which organization owner authorized this, and is kept
	// if that account is deleted — an escalation should not lose its origin.
	GrantedByID int64              `xorm:"NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	// the table is created by SyncAllTables at startup, no migration needed
	db.RegisterModel(new(ForgenteAppRunGrant))
}

// ErrForgenteAppRunGrantNotExist is returned when no grant lets a repository
// act as an app.
type ErrForgenteAppRunGrantNotExist struct {
	AppID  int64
	RepoID int64
}

// IsErrForgenteAppRunGrantNotExist checks if an error is ErrForgenteAppRunGrantNotExist
func IsErrForgenteAppRunGrantNotExist(err error) bool {
	_, ok := err.(ErrForgenteAppRunGrantNotExist)
	return ok
}

func (err ErrForgenteAppRunGrantNotExist) Error() string {
	return fmt.Sprintf("no run grant for app [app id: %d, repo id: %d]", err.AppID, err.RepoID)
}

func (err ErrForgenteAppRunGrantNotExist) Unwrap() error {
	return util.ErrNotExist
}

// GetForgenteAppRunGrant returns the grant that lets the given repository act
// as the given app, preferring one naming the repository over the
// organization-wide one so a narrower scope wins where both exist.
//
// The caller must still establish that the run is entitled to ask — that its
// task is running, and that it is not a fork pull request, which executes
// code the repository's own collaborators never approved.
func GetForgenteAppRunGrant(ctx context.Context, appID, repoID int64) (*ForgenteAppRunGrant, error) {
	grants := make([]*ForgenteAppRunGrant, 0, 2)
	if err := db.GetEngine(ctx).
		Where("app_id = ?", appID).
		In("repo_id", 0, repoID).
		Desc("repo_id").
		Limit(1).
		Find(&grants); err != nil {
		return nil, err
	}
	if len(grants) == 0 {
		return nil, ErrForgenteAppRunGrantNotExist{AppID: appID, RepoID: repoID}
	}
	return grants[0], nil
}

// ListForgenteAppRunGrants returns every grant made for an app, the
// organization-wide one first.
func ListForgenteAppRunGrants(ctx context.Context, appID int64) ([]*ForgenteAppRunGrant, error) {
	grants := make([]*ForgenteAppRunGrant, 0, 8)
	return grants, db.GetEngine(ctx).Where("app_id = ?", appID).Asc("repo_id").Find(&grants)
}

// CountForgenteAppRunGrants returns how many grants an app has, for showing the
// escalation on the app's settings page without loading the rows.
func CountForgenteAppRunGrants(ctx context.Context, appID int64) (int64, error) {
	return db.GetEngine(ctx).Where("app_id = ?", appID).Count(new(ForgenteAppRunGrant))
}

// UpdateForgenteAppRunGrantScope rewrites an existing grant's scope and who
// authorized it. Only those two columns move: re-granting is a change to an
// existing authorization, not a new one, so its creation time stays put.
func UpdateForgenteAppRunGrantScope(ctx context.Context, grant *ForgenteAppRunGrant) error {
	_, err := db.GetEngine(ctx).ID(grant.ID).Cols("scope", "granted_by_id").Update(grant)
	return err
}

// DeleteForgenteAppRunGrant withdraws one grant, checked against the app it is
// supposed to belong to so a grant id from another app cannot be revoked by
// guessing it.
func DeleteForgenteAppRunGrant(ctx context.Context, appID, grantID int64) error {
	deleted, err := db.GetEngine(ctx).
		Where("id = ? AND app_id = ?", grantID, appID).
		Delete(new(ForgenteAppRunGrant))
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrForgenteAppRunGrantNotExist{AppID: appID}
	}
	return nil
}

// Grants naming a deleted repository are removed with it, by the bean list in
// services/repository/delete.go rather than a function here.

// DeleteForgenteAppRunGrantsByAppID withdraws every grant made for an app, for
// when the app itself is deleted.
func DeleteForgenteAppRunGrantsByAppID(ctx context.Context, appID int64) error {
	_, err := db.GetEngine(ctx).Where("app_id = ?", appID).Delete(new(ForgenteAppRunGrant))
	return err
}
