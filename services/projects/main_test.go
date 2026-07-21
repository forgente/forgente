// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"testing"

	"forgente.com/models/unittest"

	_ "forgente.com/models/actions"
	_ "forgente.com/models/activities"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
