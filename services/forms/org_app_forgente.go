// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forms

import (
	"net/http"

	"forgente.com/modules/web/middleware"
	"forgente.com/services/context"

	"gitea.com/go-chi/binding"
)

// CreateOrgAppForm is the form for creating an organization-owned app. The name
// becomes a real account name, so it is bound by the same rules as a username.
type CreateOrgAppForm struct {
	Name        string `binding:"Required;Username;MaxSize(40)" locale:"settings.app_name"`
	Description string `binding:"MaxSize(255)"`
}

// Validate validates the fields
func (f *CreateOrgAppForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
