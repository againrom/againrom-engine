// Package sav reads original Asg& saves and preserves their bytes through edits.
//
// Open decompresses the container and walks the complete document envelope
// under one shared archive class/object index. Player nulls/back-references,
// the world selector, counted terrain, session, Sacks and common trailer are
// framed by serialization order, never by searching raw members. The first
// non-null Player remains the party source; no multiplayer policy is inferred.
//
// Body remains authoritative for in-place edits and Marshal. Exact structural
// framing does not imply gameplay completeness or a from-scratch world writer.
// The detached CityProvenance writer remains limited to its supported no-world
// grammar. The general reader retains its short ANSI CString boundary.
// CellTriggers projects the six persisted trigger bytes in archive order;
// it does not restore the other terrain fields or enact a gameplay entry.
//
// ClassRecords and Chain are legacy diagnostic scan helpers, not Open's index
// and not production gameplay authority.
package sav
