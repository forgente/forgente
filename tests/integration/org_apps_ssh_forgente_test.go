// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/url"
	"testing"

	asymkey_model "forgente.com/models/asymkey"
	"forgente.com/models/organization"
	"forgente.com/models/perm"
	"forgente.com/models/unittest"
	user_model "forgente.com/models/user"
	"forgente.com/modules/private"
	user_service "forgente.com/services/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrgAppSuspensionBlocksSSH covers the kill switch on the SSH path.
//
// An app can hold SSH keys as well as tokens, and a key is resolved to its owner
// without ever consulting a token, so gating only token auth would leave a
// suspended app able to push and pull. This drives the same internal endpoint
// the SSH server calls, so it fails if the gate is dropped from either
// ServCommand or ServNoCommand.
func TestOrgAppSuspensionBlocksSSH(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, _ *url.URL) {
		ctx := t.Context()

		doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})
		org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{Name: "org3"})

		botUser, app, err := user_service.CreateOrgApp(ctx, doer, org, "sshbot", "pushes things")
		require.NoError(t, err)

		key, err := asymkey_model.AddPublicKey(ctx, botUser.ID, "app-key",
			"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKJ1Ya0w3y5dERmgOVPkf1EyTOHudNet/AqsyxXEIba2 app-key", 0, true)
		require.NoError(t, err)

		// repo21 is public and owned by org3, so the app can read it without a team
		servRead := func() error {
			_, extra := private.ServCommand(ctx, key.ID, "org3", "repo21", perm.AccessModeRead, "git-upload-pack", "")
			return extra.Error
		}

		t.Run("ActiveAppCanUseItsKey", func(t *testing.T) {
			results, extra := private.ServCommand(ctx, key.ID, "org3", "repo21", perm.AccessModeRead, "git-upload-pack", "")
			require.NoError(t, extra.Error)
			assert.Equal(t, botUser.Name, results.UserName)
			assert.Equal(t, botUser.ID, results.UserID)
		})

		t.Run("SuspendedAppIsRefused", func(t *testing.T) {
			require.NoError(t, user_model.SetForgenteAppSuspended(ctx, app.ID, true))

			require.Error(t, servRead(), "a suspended app must not reach a repository over SSH")

			// the key must not resolve to an owner either, which is what the SSH
			// server asks before it runs any command at all
			_, owner, err := private.ServNoCommand(ctx, key.ID)
			require.Error(t, err)
			assert.Empty(t, owner)
		})

		t.Run("ResumeRestoresAccess", func(t *testing.T) {
			require.NoError(t, user_model.SetForgenteAppSuspended(ctx, app.ID, false))

			// suspension is a stop button, not a revocation: the same key works again
			require.NoError(t, servRead())
		})

		t.Run("SuspensionDoesNotAffectPeople", func(t *testing.T) {
			require.NoError(t, user_model.SetForgenteAppSuspended(ctx, app.ID, true))

			// key 1 belongs to user2, who owns the org that just suspended everything
			_, owner, err := private.ServNoCommand(ctx, 1)
			require.NoError(t, err)
			assert.Equal(t, doer.ID, owner.ID)
		})
	})
}
