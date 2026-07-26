// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"encoding/hex"
	"time"

	"forgente.com/models/auth"
	"forgente.com/models/db"
	"forgente.com/modules/timeutil"
	"forgente.com/modules/util"
)

// ForgenteAppRunTokenLifetime caps how long a minted token stays usable. The
// task it was minted for is checked as well, so a run that finishes early takes
// its credential with it — this is only the ceiling for a run that does not.
const ForgenteAppRunTokenLifetime = time.Hour

// ForgenteAppRunToken is a short-lived credential issued to one Actions run so
// it can act as an app.
//
// It exists so an app's identity can reach a workflow without a long-lived
// token being stored anywhere. Nothing here is presented by a human, and
// nothing is stored by the repository: the run proves which repository it
// belongs to with the job token the forge already gave it, and receives this
// in exchange.
//
// Two things bound it, and both matter more than the lifetime. It names the
// task it was minted for, so it dies when that run does rather than outliving
// it as an unowned credential. And its scope comes from the grant rather than
// from the app's own access, so lending an identity is not the same as lending
// everything that identity can reach.
type ForgenteAppRunToken struct {
	ID int64 `xorm:"pk autoincr"`
	// AppID is the app being acted as; UserID is its account, denormalised so
	// the auth path resolves a token without a second lookup.
	AppID  int64 `xorm:"INDEX NOT NULL"`
	UserID int64 `xorm:"NOT NULL"`
	// TaskID is the Actions task this was minted for. A token is only valid
	// while that task is still running, which is what keeps a cancelled job
	// from leaving a working credential behind.
	TaskID         int64  `xorm:"INDEX NOT NULL"`
	TokenHash      string `xorm:"UNIQUE"`
	TokenSalt      string
	TokenLastEight string `xorm:"INDEX token_last_eight"`
	// Token is the plaintext, set only when minting and never stored.
	Token string `xorm:"-"`
	// Scope is copied from the grant at mint time rather than read through it,
	// so narrowing a grant does not silently widen a token already issued, and
	// revoking one does not strand a run mid-job.
	Scope       auth.AccessTokenScope `xorm:"NOT NULL"`
	ExpiresUnix timeutil.TimeStamp    `xorm:"INDEX NOT NULL"`
	CreatedUnix timeutil.TimeStamp    `xorm:"created"`
}

func init() {
	// the table is created by SyncAllTables at startup, no migration needed
	db.RegisterModel(new(ForgenteAppRunToken))
}

// NewForgenteAppRunToken mints and stores a token, filling in Token with the
// plaintext for the one response that carries it.
func NewForgenteAppRunToken(ctx context.Context, t *ForgenteAppRunToken) error {
	t.TokenSalt = util.CryptoRandomString(10)
	t.Token = hex.EncodeToString(util.CryptoRandomBytes(20))
	t.TokenHash = auth.HashToken(t.Token, t.TokenSalt)
	t.TokenLastEight = t.Token[len(t.Token)-8:]
	t.ExpiresUnix = timeutil.TimeStampNow().Add(int64(ForgenteAppRunTokenLifetime / time.Second))
	return db.Insert(ctx, t)
}

// FindForgenteAppRunTokenByToken resolves a plaintext token to an unexpired
// row, or reports not-exist. Candidates are narrowed by the last eight
// characters and then confirmed by hash, the same way access tokens are, so a
// salted hash never has to be searched directly.
//
// Whether the token's task is still running is deliberately not checked here:
// that is the auth path's business and needs the actions model, which this
// package must not import.
func FindForgenteAppRunTokenByToken(ctx context.Context, token string) (*ForgenteAppRunToken, error) {
	if len(token) != 40 {
		return nil, util.ErrNotExist
	}

	candidates := make([]*ForgenteAppRunToken, 0, 2)
	if err := db.GetEngine(ctx).
		Where("token_last_eight = ? AND expires_unix > ?", token[len(token)-8:], timeutil.TimeStampNow()).
		Find(&candidates); err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		if auth.HashToken(token, candidate.TokenSalt) == candidate.TokenHash {
			return candidate, nil
		}
	}
	return nil, util.ErrNotExist
}

// DeleteExpiredForgenteAppRunTokens drops rows past their expiry and reports
// how many went. Expired rows are already refused at authentication; this only
// keeps the table from growing without bound.
func DeleteExpiredForgenteAppRunTokens(ctx context.Context) (int64, error) {
	return db.GetEngine(ctx).
		Where("expires_unix <= ?", timeutil.TimeStampNow()).
		Delete(new(ForgenteAppRunToken))
}

// DeleteForgenteAppRunTokensByAppID drops every token minted for an app, so
// deleting or suspending one does not leave a usable credential behind.
func DeleteForgenteAppRunTokensByAppID(ctx context.Context, appID int64) error {
	_, err := db.GetEngine(ctx).Where("app_id = ?", appID).Delete(new(ForgenteAppRunToken))
	return err
}
