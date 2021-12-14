// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package debugapi_test

import (
	"net/http"
	"testing"

	"github.com/holisticode/bee"
	"github.com/holisticode/bee/pkg/api"
	"github.com/holisticode/bee/pkg/debugapi"
	"github.com/holisticode/bee/pkg/jsonhttp/jsonhttptest"
)

func TestHealth(t *testing.T) {
	testServer := newTestServer(t, testServerOptions{})

	jsonhttptest.Request(t, testServer.Client, http.MethodGet, "/health", http.StatusOK,
		jsonhttptest.WithExpectedJSONResponse(debugapi.StatusResponse{
			Status:          "ok",
			Version:         bee.Version,
			APIVersion:      api.Version,
			DebugAPIVersion: debugapi.Version,
		}),
	)
}

func TestReadiness(t *testing.T) {
	testServer := newTestServer(t, testServerOptions{})

	jsonhttptest.Request(t, testServer.Client, http.MethodGet, "/readiness", http.StatusOK,
		jsonhttptest.WithExpectedJSONResponse(debugapi.StatusResponse{
			Status:          "ok",
			Version:         bee.Version,
			APIVersion:      api.Version,
			DebugAPIVersion: debugapi.Version,
		}),
	)
}
