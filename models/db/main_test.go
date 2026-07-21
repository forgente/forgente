// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package db_test

import (
	"testing"

	"forgente.com/models/unittest"

	_ "forgente.com/models"
	_ "forgente.com/models/repo"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
