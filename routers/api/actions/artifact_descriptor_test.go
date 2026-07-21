// Copyright 2026 The Forgente Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/descriptorpb"
)

// The raw descriptor in artifact.pb.go is hand-patched (no protoc in the
// build); a wrong length byte would only fail at runtime, so verify it parses
// and carries the expected go_package.
func TestArtifactDescriptorGoPackage(t *testing.T) {
	opts, ok := File_artifact_proto.Options().(*descriptorpb.FileOptions)
	assert.True(t, ok)
	assert.Equal(t, "forgente.com/routers/api/actions", opts.GetGoPackage())
}
