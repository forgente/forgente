// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvWithFallback(t *testing.T) {
	resetDeprecatedEnvWarnings()

	t.Run("NewSetWins", func(t *testing.T) {
		t.Setenv("FORGENTE_TEST_ENV_A", "new-value")
		t.Setenv("GITEA_TEST_ENV_A", "old-value")
		v, ok := EnvWithFallback("FORGENTE_TEST_ENV_A", "GITEA_TEST_ENV_A")
		assert.True(t, ok)
		assert.Equal(t, "new-value", v)
	})

	t.Run("FallbackToOld", func(t *testing.T) {
		t.Setenv("GITEA_TEST_ENV_B", "old-value")
		v, ok := EnvWithFallback("FORGENTE_TEST_ENV_B", "GITEA_TEST_ENV_B")
		assert.True(t, ok)
		assert.Equal(t, "old-value", v)
	})

	t.Run("NeitherSet", func(t *testing.T) {
		v, ok := EnvWithFallback("FORGENTE_TEST_ENV_C", "GITEA_TEST_ENV_C")
		assert.False(t, ok)
		assert.Empty(t, v)
	})
}

func TestEnvWithFallbackFunc(t *testing.T) {
	resetDeprecatedEnvWarnings()

	getEnv := func(vars map[string]string) func(string) string {
		return func(name string) string { return vars[name] }
	}

	t.Run("NewSetWins", func(t *testing.T) {
		v := EnvWithFallbackFunc(getEnv(map[string]string{
			"FORGENTE_TEST_FN_A": "new-value",
			"GITEA_TEST_FN_A":    "old-value",
		}), "FORGENTE_TEST_FN_A", "GITEA_TEST_FN_A")
		assert.Equal(t, "new-value", v)
	})

	t.Run("FallbackToOld", func(t *testing.T) {
		v := EnvWithFallbackFunc(getEnv(map[string]string{
			"GITEA_TEST_FN_B": "old-value",
		}), "FORGENTE_TEST_FN_B", "GITEA_TEST_FN_B")
		assert.Equal(t, "old-value", v)
	})

	t.Run("NeitherSet", func(t *testing.T) {
		v := EnvWithFallbackFunc(getEnv(map[string]string{}), "FORGENTE_TEST_FN_C", "GITEA_TEST_FN_C")
		assert.Empty(t, v)
	})
}
