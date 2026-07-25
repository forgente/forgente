// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package oauth2_provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateResourceIndicator(t *testing.T) {
	valid := []string{
		"https://mcp.example.com",
		"https://mcp.example.com/mcp",
		"https://mcp.example.com:8443",
		"https://mcp.example.com/server/mcp",
		"http://localhost:3000",
	}
	for _, resource := range valid {
		t.Run(resource, func(t *testing.T) {
			require.NoError(t, ValidateResourceIndicator(resource))
		})
	}

	invalid := []struct {
		resource string
		reason   string
	}{
		{"mcp.example.com", "missing scheme"},
		{"/mcp", "relative reference"},
		{"https://mcp.example.com#fragment", "fragment"},
		{"https://mcp.example.com#", "empty fragment"},
	}
	for _, tc := range invalid {
		t.Run(tc.reason, func(t *testing.T) {
			assert.Error(t, ValidateResourceIndicator(tc.resource))
		})
	}
}
