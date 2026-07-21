// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mailer

import (
	"testing"

	"forgente.com/models/unittest"

	_ "forgente.com/models"
	_ "forgente.com/models/actions"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
