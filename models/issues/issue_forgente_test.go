// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	issues_model "forgente.com/models/issues"

	"github.com/stretchr/testify/assert"
)

func TestForgenteDefaultBranchName(t *testing.T) {
	cases := []struct {
		index    int64
		title    string
		expected string
	}{
		{123, "Fix crash on login", "123-fix-crash-on-login"},
		{7, "  Weird!!  title??  ", "7-weird-title"},
		{1, "改进 中文 标题", "1-改进-中文-标题"},
		{9, "!!!", "issue-9"},
		{5, "", "issue-5"},
		{42, "word word word word word word word word word word word word word word", "42-word-word-word-word-word-word-word-word-word-word-word"},
	}
	for _, c := range cases {
		issue := &issues_model.Issue{Index: c.index, Title: c.title}
		assert.Equal(t, c.expected, issue.ForgenteDefaultBranchName(), "title %q", c.title)
	}
}
