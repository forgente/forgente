// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

import "time"

// MintAppRunTokenOption asks for a short-lived token letting the calling
// Actions run act as an app.
type MintAppRunTokenOption struct {
	// App is the app's account name.
	// required: true
	App string `json:"app" binding:"Required"`
}

// AppRunToken is a short-lived credential for one Actions run. The token is
// returned once and never retrievable again.
type AppRunToken struct {
	Token string `json:"token"`
	// App is the account the token acts as, so a workflow can attribute what it
	// does without a second call.
	App   string `json:"app"`
	Scope string `json:"scope"`
	// swagger:strfmt date-time
	ExpiresAt time.Time `json:"expires_at"`
}
