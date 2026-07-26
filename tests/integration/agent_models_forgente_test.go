// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"testing"

	"forgente.com/models/db"
	"forgente.com/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentTablesAreRegistered guards a silent failure. The agent models are
// created by SyncAllTables from init() registration, which only runs if
// something links the package — currently a single blank import in
// models/agent_forgente.go. Remove that import and everything still compiles,
// every unit test still passes, and the tables simply never exist.
func TestAgentTablesAreRegistered(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	for _, table := range []string{"agent_task", "agent_session"} {
		exist, err := db.GetEngine(t.Context()).IsTableExist(table)
		require.NoError(t, err)
		assert.True(t, exist, "table %q is missing: is models/agent still imported?", table)
	}
}
