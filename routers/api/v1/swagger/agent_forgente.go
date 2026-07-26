// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package swagger

import (
	api "forgente.com/modules/structs"
)

// AgentTask
// swagger:response AgentTask
type swaggerResponseAgentTask struct {
	// in:body
	Body api.AgentTask `json:"body"`
}

// AgentTaskList
// swagger:response AgentTaskList
type swaggerResponseAgentTaskList struct {
	// in:body
	Body []api.AgentTask `json:"body"`
}
