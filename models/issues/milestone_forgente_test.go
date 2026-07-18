// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForgenteGroupMilestonesByName(t *testing.T) {
	cases := []struct {
		name        string
		miles       MilestoneList
		wantGroups  int
		checkGroup  int // index of group to inspect
		wantName    string
		wantOpen    int
		wantClosed  int
		wantPercent int
		wantDead    int64
	}{
		{
			name: "case-insensitive grouping merges by lower-cased name, display keeps first-seen casing",
			miles: MilestoneList{
				{Name: "Q1 Release", RepoID: 1, NumOpenIssues: 2, NumClosedIssues: 1},
				{Name: "q1 release", RepoID: 2, NumOpenIssues: 0, NumClosedIssues: 3},
			},
			wantGroups:  1,
			checkGroup:  0,
			wantName:    "Q1 Release",
			wantOpen:    2,
			wantClosed:  4,
			wantPercent: 66, // 4 closed / 6 total
		},
		{
			name: "distinct names produce distinct groups in first-seen order",
			miles: MilestoneList{
				{Name: "Alpha", RepoID: 1, NumOpenIssues: 1},
				{Name: "Beta", RepoID: 2, NumOpenIssues: 1},
			},
			wantGroups: 2,
			checkGroup: 1,
			wantName:   "Beta",
			wantOpen:   1,
			wantClosed: 0,
		},
		{
			name: "earliest non-zero deadline wins across members",
			miles: MilestoneList{
				{Name: "Sprint", RepoID: 1, DeadlineUnix: 200},
				{Name: "sprint", RepoID: 2, DeadlineUnix: 0},
				{Name: "SPRINT", RepoID: 3, DeadlineUnix: 100},
			},
			wantGroups: 1,
			checkGroup: 0,
			wantName:   "Sprint",
			wantDead:   100,
		},
		{
			name: "display casing follows the lowest-ID member regardless of input order",
			miles: MilestoneList{
				{ID: 7, Name: "forgente q1", RepoID: 2, NumOpenIssues: 1},
				{ID: 4, Name: "Forgente Q1", RepoID: 1, NumClosedIssues: 1},
			},
			wantGroups:  1,
			checkGroup:  0,
			wantName:    "Forgente Q1",
			wantOpen:    1,
			wantClosed:  1,
			wantPercent: 50,
		},
		{
			name: "all-closed group with zero issues is 100% complete",
			miles: MilestoneList{
				{Name: "Done", RepoID: 1, IsClosed: true},
				{Name: "done", RepoID: 2, IsClosed: true},
			},
			wantGroups:  1,
			checkGroup:  0,
			wantName:    "Done",
			wantPercent: 100,
		},
		{
			name: "mixed open/closed with zero issues is 0% complete",
			miles: MilestoneList{
				{Name: "Mixed", RepoID: 1, IsClosed: true},
				{Name: "mixed", RepoID: 2, IsClosed: false},
			},
			wantGroups:  1,
			checkGroup:  0,
			wantName:    "Mixed",
			wantPercent: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			groups := ForgenteGroupMilestonesByName(c.miles)
			assert.Len(t, groups, c.wantGroups)
			g := groups[c.checkGroup]
			assert.Equal(t, c.wantName, g.Name)
			assert.Equal(t, c.wantOpen, g.NumOpenIssues)
			assert.Equal(t, c.wantClosed, g.NumClosedIssues)
			assert.Equal(t, c.wantPercent, g.Completeness)
			assert.Equal(t, c.wantDead, g.DeadlineUnix)
		})
	}
}
