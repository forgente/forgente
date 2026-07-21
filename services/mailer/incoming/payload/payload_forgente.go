// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package payload

import (
	"context"

	repo_model "forgente.com/models/repo"
	"forgente.com/modules/util"
)

const forgenteNewIssuePayloadVersion1 byte = 1

// CreateForgenteNewIssuePayload creates payload data which GetForgenteRepoFromPayload resolves
// back to the repository. Kept as a sibling to CreateReferencePayload (issue/comment references
// only) rather than extending it, since a repository is not an issue reference.
func CreateForgenteNewIssuePayload(repo *repo_model.Repository) ([]byte, error) {
	payload, err := util.PackData(repo.ID)
	if err != nil {
		return nil, err
	}

	return append([]byte{forgenteNewIssuePayloadVersion1}, payload...), nil
}

// GetForgenteRepoFromPayload resolves the repository from a payload created by
// CreateForgenteNewIssuePayload.
func GetForgenteRepoFromPayload(ctx context.Context, payload []byte) (*repo_model.Repository, error) {
	if len(payload) < 1 {
		return nil, util.NewInvalidArgumentErrorf("payload too small")
	}

	if payload[0] != forgenteNewIssuePayloadVersion1 {
		return nil, util.NewInvalidArgumentErrorf("unsupported payload version")
	}

	var repoID int64
	if err := util.UnpackData(payload[1:], &repoID); err != nil {
		return nil, err
	}

	return repo_model.GetRepositoryByID(ctx, repoID)
}
