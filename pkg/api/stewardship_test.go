// Copyright 2021 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api_test

import (
	"encoding/hex"
	"io"
	"net/http"
	"testing"

	"github.com/holisticode/bee/pkg/api"
	"github.com/holisticode/bee/pkg/jsonhttp"
	"github.com/holisticode/bee/pkg/jsonhttp/jsonhttptest"
	"github.com/holisticode/bee/pkg/logging"
	statestore "github.com/holisticode/bee/pkg/statestore/mock"
	"github.com/holisticode/bee/pkg/steward/mock"
	smock "github.com/holisticode/bee/pkg/storage/mock"
	"github.com/holisticode/bee/pkg/swarm"
	"github.com/holisticode/bee/pkg/tags"
)

func TestStewardship(t *testing.T) {
	var (
		logger         = logging.New(io.Discard, 0)
		statestoreMock = statestore.NewStateStore()
		stewardMock    = &mock.Steward{}
		storer         = smock.NewStorer()
		addr           = swarm.NewAddress([]byte{31: 128})
	)
	client, _, _, _ := newTestServer(t, testServerOptions{
		Storer:  storer,
		Tags:    tags.NewTags(statestoreMock, logger),
		Logger:  logger,
		Steward: stewardMock,
	})

	t.Run("re-upload", func(t *testing.T) {
		jsonhttptest.Request(t, client, http.MethodPut, "/v1/stewardship/"+addr.String(), http.StatusOK,
			jsonhttptest.WithExpectedJSONResponse(jsonhttp.StatusResponse{
				Message: http.StatusText(http.StatusOK),
				Code:    http.StatusOK,
			}),
		)
		if !stewardMock.LastAddress().Equal(addr) {
			t.Fatalf("\nhave address: %q\nwant address: %q", stewardMock.LastAddress().String(), addr.String())
		}
	})

	t.Run("is-retrievable", func(t *testing.T) {
		jsonhttptest.Request(t, client, http.MethodGet, "/v1/stewardship/"+addr.String(), http.StatusOK,
			jsonhttptest.WithExpectedJSONResponse(api.IsRetrievableResponse{IsRetrievable: true}),
		)
		jsonhttptest.Request(t, client, http.MethodGet, "/v1/stewardship/"+hex.EncodeToString([]byte{}), http.StatusNotFound,
			jsonhttptest.WithExpectedJSONResponse(&jsonhttp.StatusResponse{
				Code:    http.StatusNotFound,
				Message: http.StatusText(http.StatusNotFound),
			}),
		)
	})
}
