// Copyright 2021 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package node

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/holisticode/bee/pkg/logging"
	"github.com/holisticode/bee/pkg/statestore/leveldb"
	"github.com/holisticode/bee/pkg/statestore/mock"
	"github.com/holisticode/bee/pkg/storage"
	"github.com/holisticode/bee/pkg/swarm"
)

// InitStateStore will initialize the stateStore with the given path to the
// data directory. When given an empty directory path, the function will instead
// initialize an in-memory state store that will not be persisted.
func InitStateStore(log logging.Logger, dataDir string) (ret storage.StateStorer, err error) {
	if dataDir == "" {
		ret = mock.NewStateStore()
		log.Warning("using in-mem state store, no node state will be persisted")
		return ret, nil
	}
	return leveldb.NewStateStore(filepath.Join(dataDir, "statestore"), log)
}

const overlayKey = "overlay"
const secureOverlayKey = "non-mineable-overlay"

// CheckOverlayWithStore checks the overlay is the same as stored in the statestore
func CheckOverlayWithStore(overlay swarm.Address, storer storage.StateStorer) error {

	// migrate overlay key to new key
	err := storer.Delete(overlayKey)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}

	var storedOverlay swarm.Address
	err = storer.Get(secureOverlayKey, &storedOverlay)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		return storer.Put(secureOverlayKey, overlay)
	}

	if !storedOverlay.Equal(overlay) {
		return fmt.Errorf("overlay address changed. was %s before but now is %s", storedOverlay, overlay)
	}

	return nil
}
