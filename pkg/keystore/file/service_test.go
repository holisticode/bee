// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package file_test

import (
	"os"
	"testing"

	"github.com/holisticode/bee/pkg/keystore/file"
	"github.com/holisticode/bee/pkg/keystore/test"
)

func TestService(t *testing.T) {
	dir, err := os.MkdirTemp("", "bzz-keystore-file-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	test.Service(t, file.New(dir))
}
