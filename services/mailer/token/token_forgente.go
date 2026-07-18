// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package token

// ForgenteNewIssueHandlerType handles incoming emails that create a new issue.
// Given an explicit high value (rather than continuing the upstream iota sequence)
// so future upstream HandlerType additions can never collide with Forgente extensions.
const ForgenteNewIssueHandlerType HandlerType = 100
