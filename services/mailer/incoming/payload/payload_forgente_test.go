// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package payload

import (
	"testing"

	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unittest"

	"github.com/stretchr/testify/assert"
)

func TestForgenteNewIssuePayloadRoundTrip(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	data, err := CreateForgenteNewIssuePayload(repo)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	got, err := GetForgenteRepoFromPayload(t.Context(), data)
	assert.NoError(t, err)
	assert.Equal(t, repo.ID, got.ID)
}

func TestForgenteNewIssuePayloadInvalid(t *testing.T) {
	_, err := GetForgenteRepoFromPayload(t.Context(), nil)
	assert.Error(t, err)

	_, err = GetForgenteRepoFromPayload(t.Context(), []byte{99, 1})
	assert.Error(t, err)
}
