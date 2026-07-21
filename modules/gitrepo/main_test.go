// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitrepo

import (
	"path/filepath"
	"testing"

	"forgente.com/modules/git"
	"forgente.com/modules/git/gitcmd"
	"forgente.com/modules/setting"
)

func mockRepository(repoPath string) gitcmd.RepositoryFacade {
	if !filepath.IsAbs(repoPath) {
		// resolve repository path relative to the unit test fixture directory
		repoPath = filepath.Join(setting.GetGiteaTestSourceRoot(), "modules/git/tests/repos", repoPath)
	}
	return gitcmd.RepositoryManaged(repoPath, repoPath)
}

func TestMain(m *testing.M) {
	setting.SetupGiteaTestEnv()
	git.RunGitTests(m)
}
