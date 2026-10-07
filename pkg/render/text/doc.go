// Package text draws a string in the game's own bitmap font: one placement rule,
// read once by measurement and once by drawing.
//
// A font here is PLAIN DATA a loader outside this tier fills — the glyph grids,
// each glyph's own pen advance, and one letter spacing. Decoding the two nodes a
// font is made of needs packages this tier may not import, so the types sit
// where both this package and its callers already reach and the loading sits in
// pkg/game. This is the same seam pkg/render/terrain's sprite bundles and
// pkg/render/menu's asset set travel on. Nothing here opens an archive, reads a
// file, consults a clock or needs a graphics context.
//
// # The two things a reader should know before changing anything here
//
// THE CELL IS NOT THE ADVANCE. A glyph record is a fixed cell and the
// proportional metric is the sidecar's own value, strictly smaller than the cell
// on every glyph of every shipped atlas. Advancing by Width renders text that is
// legible, monospaced, and at the wrong pitch — a defect no test comparing our
// output against our own expectation can see, which is why the pen consults
// Advance and never Width.
//
// A PIXEL'S LEVEL 0 IS A PAINTED PIXEL. The 4-bit value is an intensity into a
// colour the caller supplies, not a palette index, and the format reserves no
// value to mean transparent — transparency is the separate Painted flag. Nothing
// shipped exercises level 0, so treating it as a hole is invisible in data and
// wrong.
//
// Tier: this package imports nothing inside the module and nothing outside the
// standard library; internal/archtest holds it to that. See docs/ARCHITECTURE.md
// and docs/0050-text/spec.md.
package text
