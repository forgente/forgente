// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package avatars_test

import (
	"testing"

	"forgente.com/models/unittest"

	_ "forgente.com/models"
	_ "forgente.com/models/activities"
	_ "forgente.com/models/perm/access"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
