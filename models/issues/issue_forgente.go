// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"fmt"
	"strings"
	"unicode"
)

// ForgenteDefaultBranchName returns the suggested name for a branch created
// from this issue, e.g. "123-fix-crash-on-login".
func (issue *Issue) ForgenteDefaultBranchName() string {
	const maxSlugLen = 60
	var b strings.Builder
	lastDash := true // swallow leading separators
	for _, r := range strings.ToLower(issue.Title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if runes := []rune(slug); len(runes) > maxSlugLen {
		slug = strings.TrimRight(string(runes[:maxSlugLen]), "-")
		if i := strings.LastIndexByte(slug, '-'); i > 0 {
			slug = slug[:i] // cut at a word boundary
		}
	}
	if slug == "" {
		return fmt.Sprintf("issue-%d", issue.Index)
	}
	return fmt.Sprintf("%d-%s", issue.Index, slug)
}
