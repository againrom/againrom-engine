package sim

// SavedObjectID is a stable native object identity. It is never a source
// address, CArchive ordinal, document index, cell or container position.
// Zero is explicit legacy/unbound state, not an invitation to match by value.
type SavedObjectID uint64
