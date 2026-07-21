// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo_test

import (
	"testing"

	"forgente.com/models/unittest"

	_ "forgente.com/models" // register table model
	_ "forgente.com/models/actions"
	_ "forgente.com/models/activities"
	_ "forgente.com/models/perm/access" // register table model
	_ "forgente.com/models/repo"        // register table model
	_ "forgente.com/models/user"        // register table model
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
