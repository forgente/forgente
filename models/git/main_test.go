// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git_test

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
