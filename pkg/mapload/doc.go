// Package mapload assembles a playable map from .alm data.
//
// A world takes up to three inputs: the map, a definition table, and the
// difficulty a spawner applies to what it builds. Only the map is a file this
// package interprets; the table arrives as an argument, because what a class key
// is worth is a property of the installed table rather than a fact of the map,
// and reading one here would give a single map two worlds depending on what is
// installed. Resolve is the ONE place a placement record's flag word, definition
// id and class keys are read, and it selects between the searches pkg/data
// offers without any of them seeing a record.
//
// The block plane a world routes on is derived in two stages and never one. The
// five terrain arms are unioned first (passability.go); a placed structure is
// then applied on top, per footprint cell, and may SUBTRACT a block the arms put
// there — which is what lets a bridge cross water (structures.go). Both stages
// read the map and the table alone, so the plane stays a function of its inputs;
// with no table the second stage resolves nothing and the plane is the first
// stage's, byte for byte.
//
// Tier: pkg/mapload may import pkg/formats/alm, pkg/data and pkg/sim. The
// package that PARSES the definition table is not on that list and is not
// reached from here: a collection arrives through the interface pkg/data
// declares, so the table crosses the tier boundary as behaviour rather than as a
// type. See docs/ARCHITECTURE.md.
package mapload
