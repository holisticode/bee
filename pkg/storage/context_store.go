package storage

import "github.com/ethersphere/bee/pkg/swarm"

type ContextStorer interface {
	GetContext(ctx ...Context)
	SyncContext()
}

type Context int

const (
	ContextSync Context = iota
	ContextCache
	ContextPin
	ContextUpload
)

type SimpleChunkStorer interface {
	// Get a chunk by its swarm.Address. Returns the chunk associated with
	// the address alongside with its postage stamp, or a storage.ErrNotFound
	// if the chunk is not found.
	Get(addr swarm.Address) (swarm.Chunk, error)
	// Put a chunk into the store alongside with its postage stamp. No duplicates
	// are allowed. It returns `exists=true` In case the chunk already exists.
	Put(ch swarm.Chunk) (exists bool, err error)
	// Iterate over chunks in no particular order.
	Iterate(IterateChunkFn) error
	// Delete a chunk from the store.
	Delete(swarm.Address) error
	// Has checks whether a chunk exists in the store.
	Has(swarm.Address) (bool, error)
}

type IterateChunkFn func(swarm.Chunk) (stop bool, err error)

/*

protocols:
- context for retrieval, push, pull

api:
- uploads
- downloads
- pinning

stewardship

*/
