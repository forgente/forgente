// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"os"
	"sync"

	"forgente.com/modules/log"
)

// deprecatedEnvWarnOnce tracks which legacy (old) env var names have already
// been warned about, so the deprecation warning is only logged once per name per process.
var deprecatedEnvWarnOnce sync.Map // map[string]*sync.Once

func warnDeprecatedEnvOnce(oldName, newName string) {
	onceAny, _ := deprecatedEnvWarnOnce.LoadOrStore(oldName, &sync.Once{})
	onceAny.(*sync.Once).Do(func() {
		log.Warn("%s is deprecated, use %s instead", oldName, newName)
	})
}

// resetDeprecatedEnvWarnings clears the one-time-warning state; for tests only, so
// repeated fallback reads in the same test binary each produce a fresh warning check.
func resetDeprecatedEnvWarnings() {
	deprecatedEnvWarnOnce = sync.Map{}
}

// EnvWithFallback reads the FORGENTE_-prefixed env var first; if unset, falls
// back to the legacy GITEA_-prefixed name with a one-time deprecation warning.
func EnvWithFallback(newName, oldName string) (value string, ok bool) {
	if v, ok := os.LookupEnv(newName); ok {
		return v, true
	}
	if v, ok := os.LookupEnv(oldName); ok {
		warnDeprecatedEnvOnce(oldName, newName)
		return v, true
	}
	return "", false
}

// EnvWithFallbackFunc is like EnvWithFallback, but reads through a caller-supplied
// lookup function instead of os.LookupEnv. It exists for callers (e.g.
// InitWorkPathAndCfgProvider) that accept an injectable getEnvFn for testability;
// getEnvFn is expected to return "" for unset variables.
func EnvWithFallbackFunc(getEnvFn func(name string) string, newName, oldName string) (value string) {
	if v := getEnvFn(newName); v != "" {
		return v
	}
	if v := getEnvFn(oldName); v != "" {
		warnDeprecatedEnvOnce(oldName, newName)
		return v
	}
	return ""
}
