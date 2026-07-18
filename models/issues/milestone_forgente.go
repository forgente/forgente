// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"sort"
	"strings"
)

// ForgenteMilestoneGroup is a GitLab-style "group milestone": one milestone
// name spanning multiple repositories, with combined progress. Gitea's
// milestones are strictly repo-scoped (see Milestone.RepoID), so this is a
// purely presentational roll-up computed over already-loaded milestones —
// it does not introduce a new database concept.
type ForgenteMilestoneGroup struct {
	Name            string // display name, using the casing of the oldest (lowest-ID) member milestone
	Milestones      MilestoneList
	NumOpenIssues   int
	NumClosedIssues int
	Completeness    int   // percentage (0-100), combined across all member milestones
	DeadlineUnix    int64 // earliest non-zero deadline among member milestones, 0 if none set
}

// NumIssues returns the combined issue count (open + closed) of the group.
func (g *ForgenteMilestoneGroup) NumIssues() int {
	return g.NumOpenIssues + g.NumClosedIssues
}

// ForgenteGroupMilestonesByName groups already-loaded milestones by name
// (case-insensitive) and computes combined stats for each group. Milestones
// are expected to already have their Repo field populated by the caller.
// Group order follows first-seen order of each name in the input list. The
// display casing comes from the lowest-ID member (input order is not
// deterministic across databases: case-insensitive collations make
// same-named rows compare equal in ORDER BY).
func ForgenteGroupMilestonesByName(miles MilestoneList) []*ForgenteMilestoneGroup {
	groups := make([]*ForgenteMilestoneGroup, 0, len(miles))
	index := make(map[string]int, len(miles)) // lower-cased name -> index into groups

	for _, m := range miles {
		key := strings.ToLower(m.Name)
		idx, ok := index[key]
		if !ok {
			idx = len(groups)
			index[key] = idx
			groups = append(groups, &ForgenteMilestoneGroup{Name: m.Name})
		}
		g := groups[idx]
		g.Milestones = append(g.Milestones, m)
		g.NumOpenIssues += m.NumOpenIssues
		g.NumClosedIssues += m.NumClosedIssues

		if int64(m.DeadlineUnix) > 0 && (g.DeadlineUnix == 0 || int64(m.DeadlineUnix) < g.DeadlineUnix) {
			g.DeadlineUnix = int64(m.DeadlineUnix)
		}
	}

	for _, g := range groups {
		// sort members by ID: same-named rows compare equal under case-insensitive
		// DB collations, so the query's ORDER BY leaves their order backend-dependent
		sort.SliceStable(g.Milestones, func(i, j int) bool { return g.Milestones[i].ID < g.Milestones[j].ID })
		g.Name = g.Milestones[0].Name // display casing: the oldest member's name

		if total := g.NumIssues(); total > 0 {
			g.Completeness = g.NumClosedIssues * 100 / total
		} else if allClosed(g.Milestones) {
			// mirrors Milestone.BeforeUpdate: a closed milestone with no issues counts as 100% complete
			g.Completeness = 100
		}
	}

	return groups
}

func allClosed(miles MilestoneList) bool {
	for _, m := range miles {
		if !m.IsClosed {
			return false
		}
	}
	return len(miles) > 0
}
