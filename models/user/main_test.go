// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user_test

import (
	"testing"

	"forgente.com/models/unittest"

	_ "forgente.com/models"
	_ "forgente.com/models/actions"
	_ "forgente.com/models/activities"
	_ "forgente.com/models/user"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
