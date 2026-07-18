// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package payload

import (
	"testing"

	"gitea.dev/models/unittest"

	_ "gitea.dev/models"
	_ "gitea.dev/models/actions"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
