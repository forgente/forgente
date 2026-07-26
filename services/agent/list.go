// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"context"

	agent_model "forgente.com/models/agent"
	"forgente.com/models/db"

	"xorm.io/builder"
)

// ListTasksOptions selects tasks for listing.
type ListTasksOptions struct {
	db.ListOptions
	RepoID int64
	// State filters to one state when set. An empty value means all states.
	State agent_model.State
	// IncludeArchived brings back tasks that were archived on completion,
	// which the default listing hides.
	IncludeArchived bool
}

func (opts ListTasksOptions) toConds() builder.Cond {
	cond := builder.NewCond()
	if opts.RepoID != 0 {
		cond = cond.And(builder.Eq{"repo_id": opts.RepoID})
	}
	if opts.State != "" {
		cond = cond.And(builder.Eq{"state": opts.State})
	}
	if !opts.IncludeArchived {
		cond = cond.And(builder.Eq{"archived_at": 0})
	}
	return cond
}

// ListTasks returns tasks newest first, with the total count before paging.
func ListTasks(ctx context.Context, opts ListTasksOptions) ([]*agent_model.Task, int64, error) {
	sess := db.GetEngine(ctx).Where(opts.toConds()).OrderBy("id DESC")
	if opts.Page > 0 {
		db.SetSessionPagination(sess, &opts)
	}
	tasks := make([]*agent_model.Task, 0, opts.PageSize)
	count, err := sess.FindAndCount(&tasks)
	return tasks, count, err
}

// GetTaskByID returns one task, or nil when it does not exist.
func GetTaskByID(ctx context.Context, id int64) (*agent_model.Task, error) {
	task := new(agent_model.Task)
	has, err := db.GetEngine(ctx).ID(id).Get(task)
	if err != nil || !has {
		return nil, err
	}
	return task, nil
}
