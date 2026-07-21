// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgente.com/modules/setting"
	"forgente.com/modules/structs"
	"forgente.com/modules/test"
	"forgente.com/tests"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.AppVer, "test-version-1")()
	req := NewRequest(t, "GET", "/api/v1/version")
	resp := MakeRequest(t, req, http.StatusOK)

	version := DecodeJSON(t, resp, &structs.ServerVersion{})
	assert.Equal(t, setting.AppVer, version.Version)
}
