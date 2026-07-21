// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitdiff

import (
	"testing"

	"forgente.com/models/unittest"

	_ "forgente.com/models"
	_ "forgente.com/models/actions"
	_ "forgente.com/models/activities"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
