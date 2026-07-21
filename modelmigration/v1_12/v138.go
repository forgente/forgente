// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_12

import (
	"fmt"

	"forgente.com/modelmigration/base"
)

func AddResolveDoerIDCommentColumn(x base.EngineMigration) error {
	type Comment struct {
		ResolveDoerID int64
	}

	if err := x.Sync(new(Comment)); err != nil {
		return fmt.Errorf("Sync: %w", err)
	}
	return nil
}
