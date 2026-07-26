// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package models

import (
	// The agent models register themselves in init(), and registration only
	// runs if something links the package. Nothing does yet — the services
	// that will use it are not built — so without this import SyncAllTables
	// would never see the tables and they would silently not exist.
	//
	// This lives in its own file rather than joining the imports of an
	// inherited one, so upstream cherry-picks never conflict with it.
	_ "forgente.com/models/agent"
)
