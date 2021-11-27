package contextstore

import (
	"context"
	"path/filepath"

	"github.com/ethersphere/bee/pkg/contextstore/uploadcontext"
	"github.com/ethersphere/bee/pkg/storage"
	"github.com/ethersphere/bee/pkg/swarm"
)

type ContextStore struct {
	uploadContext *uploadcontext.UploadContext
	ls            storage.Storer
}

func New(path string) *ContextStore {
	up := uploadcontext.New(filepath.Join(path, "upload"))
	c := &ContextStore{uploadContext: up}
	return c
}

func (c *ContextStore) UploadContext() *uploadcontext.UploadContext {
	return c.uploadContext
}

func (c *ContextStore) SyncContext() storage.SimpleChunkStorer {
	return wrap(c.ls, storage.ModeGetRequest, storage.ModePutRequest)
}

func (c *ContextStore) PinContext() storage.SimpleChunkStorer {
	return wrap(c.ls, storage.ModeGetRequest, storage.ModePutRequest)
}

type modeWrapper struct {
	modeGet storage.ModeGet
	modePut storage.ModePut
	ls      storage.Storer
}

func wrap(l storage.Storer, g storage.ModeGet, p storage.ModePut) storage.SimpleChunkStorer {
	return &modeWrapper{
		modeGet: g,
		modePut: p,
		ls:      l,
	}
}

// Get a chunk by its swarm.Address. Returns the chunk associated with
// the address alongside with its postage stamp, or a storage.ErrNotFound
// if the chunk is not found.
func (m *modeWrapper) Get(addr swarm.Address) (swarm.Chunk, error) {
	return m.ls.Get(context.TODO(), m.modeGet, addr)
}

// Put a chunk into the store alongside with its postage stamp. No duplicates
// are allowed. It returns `exists=true` In case the chunk already exists.
func (m *modeWrapper) Put(ch swarm.Chunk) (exists bool, err error) {
	e, err := m.ls.Put(context.TODO(), m.modePut, ch)
	return e[0], err
}

// Iterate over chunks in no particular order.
func (m *modeWrapper) Iterate(_ storage.IterateChunkFn) error {
	panic("not implemented") // TODO: Implement
}

// Delete a chunk from the store.
func (m *modeWrapper) Delete(addr swarm.Address) error {
	return m.ls.Set(context.TODO(), storage.ModeSetRemove, addr)
}

// Has checks whether a chunk exists in the store.
func (m *modeWrapper) Has(addr swarm.Address) (bool, error) {
	return m.ls.Has(context.TODO(), addr)
}
