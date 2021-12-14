// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package debugapi

import (
	"net/http"

	"github.com/holisticode/bee"
	"github.com/holisticode/bee/pkg/api"
	"github.com/holisticode/bee/pkg/jsonhttp"
)

type statusResponse struct {
	Status          string `json:"status"`
	Version         string `json:"version"`
	APIVersion      string `json:"apiVersion"`
	DebugAPIVersion string `json:"debugApiVersion"`
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	jsonhttp.OK(w, statusResponse{
		Status:          "ok",
		Version:         bee.Version,
		APIVersion:      api.Version,
		DebugAPIVersion: Version,
	})
}
