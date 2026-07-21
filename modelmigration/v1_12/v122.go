// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_12

import "forgente.com/modelmigration/base"

func AddRequireSignedCommits(x base.EngineMigration) error {
	type ProtectedBranch struct {
		RequireSignedCommits bool `xorm:"NOT NULL DEFAULT false"`
	}

	return x.Sync(new(ProtectedBranch))
}
