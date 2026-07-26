// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cron

import (
	"context"

	user_model "forgente.com/models/user"
	"forgente.com/modules/log"
)

// registerCleanupAppRunTokens drops expired app run tokens.
//
// This is housekeeping, not enforcement: an expired token is already refused at
// authentication, and so is one whose task has stopped. Left alone the table
// would still grow by one row per job per app forever, which is why it runs
// enabled by default and often — the rows are worthless within the hour.
func registerCleanupAppRunTokens() {
	RegisterTaskFatal("cleanup_app_run_tokens", &BaseConfig{
		Enabled:    true,
		RunAtStart: false,
		Schedule:   "@hourly",
	}, func(ctx context.Context, _ *user_model.User, _ Config) error {
		deleted, err := user_model.DeleteExpiredForgenteAppRunTokens(ctx)
		if err != nil {
			return err
		}
		if deleted > 0 {
			log.Trace("cleanup_app_run_tokens: removed %d expired token(s)", deleted)
		}
		return nil
	})
}
