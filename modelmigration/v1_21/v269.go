// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_21

import "forgente.com/modelmigration/base"

func DropDeletedBranchTable(x base.EngineMigration) error {
	return x.DropTables("deleted_branch")
}
