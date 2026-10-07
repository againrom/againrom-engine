// Package mapedit is the headless edit model over one accepted ROM1 .alm map:
// typed mutations — paint a grid cell, edit a type-0 field, place, move and
// delete a unit — with undo and redo, and every byte the caller did not ask to
// change carried out of the file it was loaded from.
//
// Preservation by carrying, not by regeneration. A mutation partitions the
// loaded bytes into the target region it deliberately edits and the carried
// remainder, and the remainder is copied, never re-encoded from an interpreted
// value: a move rewrites a unit record's two coordinate words and leaves the
// other 62 bytes exactly as they arrived, the seven 64-byte text slots and the
// bytes past a string field's first NUL ride along untouched, and the file
// header's own size word is carried rather than recomputed. That is the whole
// reason this package edits bytes instead of re-serializing a decoded map: the
// interpreted view does not expose everything the file holds, so anything
// regenerated from it would be an invention where the original had data.
//
// The model is therefore the bytes: an accepted stream, an ordered log of the
// edits applied to it, and a cursor into that log for undo and redo. Nothing
// derived is stored — offsets, record spans, the map dimensions and the unit
// count are computed from the current bytes at each use — so a length-changing
// edit has no cached index to invalidate. Acceptance is alm.OpenDocument's
// alone and is taken once, at load; every mutation validates its own arguments
// first and is atomic, leaving the bytes, the view and the history untouched
// when it rejects them. The serialized bytes and the interpreted view are both
// functions of the buffer, each handed out as a fresh value that no later edit
// disturbs.
//
// A map need not carry every record. alm accepts three and up with only
// type-0, type-1 and type-2 guaranteed, so the overlay plane and the unit
// roster may simply not be in the file — and an absent record is not an empty
// one. A setter that addresses a section the map does not have is rejected like
// any other bad argument, atomically, rather than resolved against some other
// record's bytes; the unit count of a map with no type-6 record is zero however
// many that map's own count word advertises; and no mutation here adds or
// removes a record, so the file header's record count never changes.
//
// Units cross this boundary as bytes and nothing else: a record is read as a
// 70-byte copy and placed as a complete 70-byte record from the caller, so no
// call here can invent the 50 bytes the decoded view does not expose. The
// cheap way to a valid record is to clone one. A unit is addressed by its
// file-order index, and a delete shifts the later indices down.
//
// Scope is the version-990 dialect pkg/formats/alm accepts. The editor is a
// stateful single-goroutine session; it validates nothing against the game — a
// coordinate outside the grid, an owner slot that resolves to no roster entry,
// a duplicated id word are all the caller's business.
//
// Tier: pkg/mapedit sits directly above pkg/formats/alm and imports it plus
// the standard library, and no external module — the golang.org/x/text grant
// belongs to the formats tier, which is why both halves of the type-0 string
// field codecs live down there. See docs/ARCHITECTURE.md and
// docs/0025-mapedit-model/spec.md.
package mapedit
