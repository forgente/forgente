// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_23

import (
	"forgente.com/models/db"
	"forgente.com/modules/timeutil"

	"xorm.io/xorm"
)

func AddIndexToActionTaskStoppedLogExpired(x db.EngineMigration) error {
	type ActionTask struct {
		Stopped    timeutil.TimeStamp `xorm:"index(stopped_log_expired)"`
		LogExpired bool               `xorm:"index(stopped_log_expired)"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{
		IgnoreDropIndices: true,
	}, new(ActionTask))
	return err
}
