// Package data turns a parsed graphics registry into typed classes.
//
// LoadUnitClasses, LoadObjectClasses and LoadStructureClasses each take a
// registry already parsed by pkg/formats/reg and return that registry's classes
// with inheritance resolved: every field holds the class's own value at that
// key, the value the nearest ancestor that sets it held, or — where no section
// on the chain sets it at all — that key's own default, which is the type's zero
// only where the registry says so. A loaded class keeps no reference to its
// parent. Loading performs no IO, and a class is looked up by its ID — never by
// its section index.
//
// NewUnitDef turns one row of the definition table's Units collection into a
// stat block, by replaying the spawn path's own slot order over the
// constructor's defaults: slots 0 to 37 in ascending order, an empty cell
// leaving the default standing and being consumed all the same. Those two files
// are the ONLY place in this tree that reads a meaning into a definition-table
// parameter — the format tier frames the table and names nothing — and the
// registry loaders above share nothing with them but the tier.
//
// Tier: pkg/data may import pkg/vfs and pkg/formats/reg; this loader imports
// only pkg/formats/reg, and the definition half imports nothing at all. It takes
// the table's rows as plain values, so the format tier that parses them is not
// on this package's import list. See docs/ARCHITECTURE.md.
package data
