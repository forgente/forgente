// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package system_test

import (
	"testing"

	"forgente.com/models/unittest"

	_ "forgente.com/models" // register models
	_ "forgente.com/models/actions"
	_ "forgente.com/models/activities"
	_ "forgente.com/models/system" // register models of system
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
