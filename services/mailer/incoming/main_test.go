// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package incoming

import (
	"testing"

	"forgente.com/models/unittest"

	_ "forgente.com/models"
	_ "forgente.com/models/actions"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
