package sim

import "hash/fnv"

// Hash returns a 64-bit digest of the world's full canonical state: FNV-1a over
// exactly the bytes MarshalBinary produces.
//
// Hashing the byte form rather than the fields is what makes "every field the
// byte form carries enters the digest" true by construction. A second traversal
// over the fields would be a second place to forget one, and the two could drift
// apart without either looking wrong on its own.
//
// It follows that worlds with identical byte forms hash equal, and that the
// digest depends on the logical world alone — entities are encoded in ascending
// id order, so neither insertion nor storage order reaches it.
//
// FNV-1a is a fixed published function with fixed constants, so a digest pinned
// in a test is recomputable from the pinned bytes by someone holding no code of
// ours: the two pins check each other instead of both falling out of one run of
// this package. hash/maphash is the wrong tool here for the precise reason this
// story exists — its seed is randomised per process, so the same world would
// hash differently in the next run.
func (w *World) Hash() uint64 {
	h := fnv.New64a()
	// hash.Hash's Write never returns an error.
	h.Write(w.encode())
	return h.Sum64()
}
