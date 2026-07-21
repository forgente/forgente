// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package markup_test

import (
	"os"
	"testing"

	"forgente.com/modules/markup"
	"forgente.com/modules/setting"
)

func TestMain(m *testing.M) {
	setting.IsInTesting = true
	markup.RenderBehaviorForTesting.DisableAdditionalAttributes = true
	setting.Markdown.FileNamePatterns = []string{"*.md"}
	markup.RefreshFileNamePatterns()
	os.Exit(m.Run())
}
