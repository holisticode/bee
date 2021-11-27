package uploadcontext

import (
	"sync"

	"github.com/ethersphere/bee/pkg/contextstore/simplestore"
	"github.com/ethersphere/bee/pkg/storage"
	"github.com/ethersphere/bee/pkg/swarm"
)

type UploadContext struct {
	s     *simplestore.Store
	itemC chan struct{}
	mtx   sync.Mutex
	wg    sync.WaitGroup
	quit  chan struct{}
}

func New(base string) *UploadContext {
	s, err := simplestore.New(base)
	if err != nil {
		panic(err)
	}
	cnt, err := s.Count()
	if err != nil {
		panic(err)
	}

	u := &UploadContext{
		s:     s,
		itemC: make(chan struct{}, 1),
	}
	if cnt > 0 {
		u.itemC <- struct{}{}
	}

	return u
}

// Feed creates a forever loop that feeds chunks continuously
// on the returned channel.
func (u *UploadContext) Feed() chan swarm.Chunk {
	go func() {
		for {
			select {
			case <-u.itemC:
				// if there are items, continue iterating
			case <-u.quit:
				return
			}

			_ = u.s.Iterate(func(swarm.Chunk) (bool, error) {
				return false, nil
			})

			u.mtx.Lock()
			cnt, err := u.s.Count()
			if err != nil {
				panic(err)
			}
			if cnt > 0 {
				select {
				case u.itemC <- struct{}{}:
				default:
				}
			} else {
				// destroy the store and create a new one, so that
				// the storage space can be reclaimed.
			}
			u.mtx.Unlock()
		}
	}()
	return nil
}

// Get a chunk by its swarm.Address. Returns the chunk associated with
// the address alongside with its postage stamp, or a storage.ErrNotFound
// if the chunk is not found.
func (u *UploadContext) Get(addr swarm.Address) (swarm.Chunk, error) {
	return u.s.Get(addr)
}

// Put a chunk into the store alongside with its postage stamp. No duplicates
// are allowed. It returns `exists=true` In case the chunk already exists.
func (u *UploadContext) Put(ch swarm.Chunk) (exists bool, err error) {
	return u.s.Put(ch)
}

// Iterate over chunks in no particular order.
func (u *UploadContext) Iterate(f storage.IterateChunkFn) error {
	return u.s.Iterate(f)
}

// Delete a chunk from the store.
func (u *UploadContext) Delete(addr swarm.Address) error {
	return u.s.Delete(addr)
}

// Has checks whether a chunk exists in the store.
func (u *UploadContext) Has(addr swarm.Address) (bool, error) {
	return u.s.Has(addr)
}
