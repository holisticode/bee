// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package localstore

import (
	"errors"
	"time"

	"github.com/holisticode/bee/pkg/shed"
	"github.com/holisticode/bee/pkg/swarm"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	// gcTargetRatio defines the target number of items
	// in garbage collection index that will not be removed
	// on garbage collection. The target number of items
	// is calculated by gcTarget function. This value must be
	// in range (0,1]. For example, with 0.9 value,
	// garbage collection will leave 90% of defined capacity
	// in database after its run. This prevents frequent
	// garbage collection runs.
	gcTargetRatio = 0.9
	// gcBatchSize limits the number of chunks in a single
	// transaction on garbage collection.
	gcBatchSize uint64 = 2000

	// reserveCollectionRatio is the ratio of the cache to evict from
	// the reserve every time it hits the limit. If the cache size is
	// 1000 chunks then we will evict 500 chunks from the reserve, this is
	// not to overwhelm the cache with too many chunks which it will flush
	// anyway.
	reserveCollectionRatio = 0.5
	// reserveEvictionBatch limits the number of chunks collected in
	// a single reserve eviction run.
	reserveEvictionBatch uint64 = 200
	// maxPurgeablePercentageOfReserve is a ceiling of size of the reserve
	// to evict in case the cache size is bigger than the reserve
	maxPurgeablePercentageOfReserve = 0.1
)

// collectGarbageWorker is a long running function that waits for
// collectGarbageTrigger channel to signal a garbage collection
// run. GC run iterates on gcIndex and removes older items
// form retrieval and other indexes.
func (db *DB) collectGarbageWorker() {
	defer close(db.collectGarbageWorkerDone)

	for {
		select {
		case <-db.collectGarbageTrigger:
			// run a single collect garbage run and
			// if done is false, gcBatchSize is reached and
			// another collect garbage run is needed
			collectedCount, done, err := db.collectGarbage()
			if err != nil {
				db.logger.Errorf("localstore: collect garbage: %v", err)
			}
			// check if another gc run is needed
			if !done {
				db.triggerGarbageCollection()
			}

			if testHookCollectGarbage != nil {
				testHookCollectGarbage(collectedCount)
			}
		case <-db.close:
			return
		}
	}
}

// collectGarbage removes chunks from retrieval and other
// indexes if maximal number of chunks in database is reached.
// This function returns the number of removed chunks. If done
// is false, another call to this function is needed to collect
// the rest of the garbage as the batch size limit is reached.
// This function is called in collectGarbageWorker.
func (db *DB) collectGarbage() (collectedCount uint64, done bool, err error) {
	db.metrics.GCCounter.Inc()
	defer func(start time.Time) {
		if err != nil {
			db.metrics.GCErrorCounter.Inc()
		}
		totalTimeMetric(db.metrics.TotalTimeCollectGarbage, start)
	}(time.Now())
	batch := new(leveldb.Batch)
	target := db.gcTarget()

	// tell the localstore to start logging dirty addresses
	db.batchMu.Lock()
	db.gcRunning = true
	db.batchMu.Unlock()

	defer func() {
		db.batchMu.Lock()
		db.gcRunning = false
		db.dirtyAddresses = nil
		db.batchMu.Unlock()
	}()

	gcSize, err := db.gcSize.Get()
	if err != nil {
		return 0, true, err
	}
	if gcSize == target {
		return 0, true, nil
	}
	db.metrics.GCSize.Set(float64(gcSize))
	done = true
	first := true
	start := time.Now()
	candidates := make([]shed.Item, 0)
	err = db.gcIndex.Iterate(func(item shed.Item) (stop bool, err error) {
		if first {
			totalTimeMetric(db.metrics.TotalTimeGCFirstItem, start)
			first = false
		}
		if gcSize-collectedCount <= target {
			return true, nil
		}

		candidates = append(candidates, item)

		collectedCount++
		if collectedCount >= gcBatchSize {
			// batch size limit reached, however we don't
			// know whether another gc run is needed until
			// we weed out the dirty entries below
			return true, nil
		}
		return false, nil
	}, nil)
	if err != nil {
		return 0, false, err
	}
	db.metrics.GCCollectedCounter.Add(float64(collectedCount))
	if testHookGCIteratorDone != nil {
		testHookGCIteratorDone()
	}

	// protect database from changing idexes and gcSize
	db.batchMu.Lock()
	defer totalTimeMetric(db.metrics.TotalTimeGCLock, time.Now())
	defer db.batchMu.Unlock()

	// refresh gcSize value, since it might have
	// changed in the meanwhile
	gcSize, err = db.gcSize.Get()
	if err != nil {
		return 0, false, err
	}

	// get rid of dirty entries
	for _, item := range candidates {
		if swarm.NewAddress(item.Address).MemberOf(db.dirtyAddresses) {
			collectedCount--
			continue
		}

		db.metrics.GCStoreTimeStamps.Set(float64(item.StoreTimestamp))
		db.metrics.GCStoreAccessTimeStamps.Set(float64(item.AccessTimestamp))

		// delete from retrieve, pull, gc
		err = db.retrievalDataIndex.DeleteInBatch(batch, item)
		if err != nil {
			return 0, false, err
		}
		err = db.retrievalAccessIndex.DeleteInBatch(batch, item)
		if err != nil {
			return 0, false, err
		}
		err = db.pushIndex.DeleteInBatch(batch, item)
		if err != nil {
			return 0, false, err
		}
		err = db.pullIndex.DeleteInBatch(batch, item)
		if err != nil {
			return 0, false, err
		}
		err = db.gcIndex.DeleteInBatch(batch, item)
		if err != nil {
			return 0, false, err
		}
		err = db.postageChunksIndex.DeleteInBatch(batch, item)
		if err != nil {
			return 0, false, err
		}
		err = db.postageIndexIndex.DeleteInBatch(batch, item)
		if err != nil {
			return 0, false, err
		}

	}
	if gcSize-collectedCount > target {
		done = false
	}

	db.metrics.GCCommittedCounter.Add(float64(collectedCount))
	db.gcSize.PutInBatch(batch, gcSize-collectedCount)

	err = db.shed.WriteBatch(batch)
	if err != nil {
		db.metrics.GCErrorCounter.Inc()
		return 0, false, err
	}
	return collectedCount, done, nil
}

// gcTrigger retruns the absolute value for garbage collection
// target value, calculated from db.capacity and gcTargetRatio.
func (db *DB) gcTarget() (target uint64) {
	return uint64(float64(db.cacheCapacity) * gcTargetRatio)
}

func (db *DB) reserveEvictionTarget() (target uint64) {
	targetCache := db.reserveCapacity - uint64(float64(db.cacheCapacity)*reserveCollectionRatio)
	targetCeiling := db.reserveCapacity - uint64(float64(db.reserveCapacity)*maxPurgeablePercentageOfReserve)
	if targetCeiling > targetCache {
		return targetCeiling
	}
	return targetCache
}

// triggerGarbageCollection signals collectGarbageWorker
// to call collectGarbage.
func (db *DB) triggerGarbageCollection() {
	select {
	case db.collectGarbageTrigger <- struct{}{}:
	case <-db.close:
	default:
	}
}

// triggerGarbageCollection signals collectGarbageWorker
// to call collectGarbage.
func (db *DB) triggerReserveEviction() {
	select {
	case db.reserveEvictionTrigger <- struct{}{}:
	case <-db.close:
	default:
	}
}

// incGCSizeInBatch changes gcSize field value
// by change which can be negative. This function
// must be called under batchMu lock.
func (db *DB) incGCSizeInBatch(batch *leveldb.Batch, change int64) (err error) {
	if change == 0 {
		return nil
	}
	gcSize, err := db.gcSize.Get()
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return err
	}

	var newSize uint64
	if change > 0 {
		newSize = gcSize + uint64(change)
	} else {
		// 'change' is an int64 and is negative
		// a conversion is needed with correct sign
		c := uint64(-change)
		if c > gcSize {
			// protect uint64 undeflow
			return nil
		}
		newSize = gcSize - c
	}
	db.gcSize.PutInBatch(batch, newSize)
	db.metrics.GCSize.Set(float64(newSize))

	// trigger garbage collection if we reached the capacity
	if newSize >= db.cacheCapacity {
		db.triggerGarbageCollection()
	}
	return nil
}

// incReserveSizeInBatch changes reserveSize field value
// by change which can be negative. This function
// must be called under batchMu lock.
func (db *DB) incReserveSizeInBatch(batch *leveldb.Batch, change int64) (err error) {
	if change == 0 {
		return nil
	}
	reserveSize, err := db.reserveSize.Get()
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return err
	}

	var newSize uint64
	if change > 0 {
		newSize = reserveSize + uint64(change)
	} else {
		// 'change' is an int64 and is negative
		// a conversion is needed with correct sign
		c := uint64(-change)
		if c > reserveSize {
			// protect uint64 undeflow
			return nil
		}
		newSize = reserveSize - c
	}
	db.reserveSize.PutInBatch(batch, newSize)
	db.metrics.ReserveSize.Set(float64(newSize))
	// trigger garbage collection if we reached the capacity
	if newSize >= db.reserveCapacity {
		db.triggerReserveEviction()
	}
	return nil
}

func (db *DB) reserveEvictionWorker() {
	defer close(db.reserveEvictionWorkerDone)
	for {
		select {
		case <-db.reserveEvictionTrigger:
			evictedCount, done, err := db.evictReserve()
			if err != nil {
				db.logger.Errorf("localstore: evict reserve: %v", err)
			}

			if !done {
				db.triggerReserveEviction()
			}

			if testHookEviction != nil {
				testHookEviction(evictedCount)
			}
		case <-db.close:
			return
		}
	}
}

func (db *DB) evictReserve() (totalEvicted uint64, done bool, err error) {
	var target uint64
	db.metrics.EvictReserveCounter.Inc()
	defer func(start time.Time) {
		if err != nil {
			db.metrics.EvictReserveErrorCounter.Inc()
		}
		totalTimeMetric(db.metrics.TotalTimeEvictReserve, start)
	}(time.Now())
	target = db.reserveEvictionTarget()
	db.batchMu.Lock()
	defer db.batchMu.Unlock()
	reserveSizeStart, err := db.reserveSize.Get()
	if err != nil {
		return 0, false, err
	}
	if reserveSizeStart == target {
		return 0, true, nil
	}

	// if we dont get any entries at all then there's no use
	// of triggering subsequent runs in case we're not done
	totalCallbacks := 0
	err = db.unreserveFunc(func(batchID []byte, radius uint8) (bool, error) {
		totalCallbacks++
		e, err := db.UnreserveBatch(batchID, radius)
		if err != nil {
			return true, err
		}
		totalEvicted += e
		if reserveSizeStart-totalEvicted <= target {
			done = true
			return true, nil
		}
		if totalEvicted >= reserveEvictionBatch {
			// stop collecting when we reach the eviction
			// batch size so that we can avoid lock contention
			// on localstore.
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return 0, false, err
	}

	if totalCallbacks == 0 {
		// if we did not get any items from the batchstore
		// it means there's no point of trigerring a subsequent
		// round
		done = true
	}

	return totalEvicted, done, nil
}

// testHookCollectGarbage is a hook that can provide
// information when a garbage collection run is done
// and how many items it removed.
var testHookCollectGarbage func(collectedCount uint64)

// testHookGCIteratorDone is a hook which is called
// when the GC is done collecting candidate items for
// eviction.
var testHookGCIteratorDone func()

var withinRadiusFn func(*DB, shed.Item) bool

var testHookEviction func(count uint64)
