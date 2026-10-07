// Package terrain turns a ROM1 map's tile grid into terrain pixels.
//
// It holds the pure pieces of the terrain render path: an 8-bpp Windows BMP
// decoder (the container the terrain tiles ship in), a slicer that cuts a tile
// strip into its 32x32 sub-cells, the research-decoded tile-word -> graphic
// mapping, the relief light, and the compositors that lay a whole map's cells
// out into one image.
//
// A cell's geometry and its light are independent axes, so there are four
// compositors. CompositeProjected and CompositeProjectedLit are the DEFAULT
// geometry (0012): every cell is a quad whose four corners the map's altitudes
// displace vertically, drawn as 32 resampled per-column spans unless its four
// corner altitudes are equal, on a canvas spanning exactly the rows that vertex
// mesh reaches — so the image is not H*32*scale tall and its row 0 stands for
// Render.OriginY, the native row the projection starts at. Composite and
// CompositeLit are the whole-image flat raster, every cell an axis-aligned
// square at (col*32, row*32) and OriginY 0; they keep the pixels, dimensions and
// counts they had before 0012 and are now an explicit diagnostic rather than the
// default. A caller drawing on the un-displaced cell lattice over a projected
// image — the marker overlays — passes -OriginY*scale to the *At entry points.
//
// Two sprite layers stand on that terrain and are placed here, by two builders
// that share no arithmetic. Object and unit art is anchored by a class CANVAS
// and the frame's own ANCHOR PIXEL (statics.go, units.go). A placed STRUCTURE
// has neither: its anchor is a CELL, that cell is the art rectangle's top-left,
// and one frame of its sheet is one map cell, so it is drawn as a vertical strip
// of image rows per rectangle cell rather than as a sprite (structures.go,
// TERR-STRUCT-100/101). The two must not be merged: a structure class declares
// no canvas and no centre, and a builder that invented one would be calibrating
// an invention against shipped art.
//
// Everything here takes plain bytes, decoded images and integers: no archive,
// no map file, no game path. Reading terrain/* out of graphics.res and
// decoding the .alm is the caller's job (see cmd/terraintool), which keeps the
// format tier out of the render tier entirely.
//
// Game facts implemented here are research-derived (claims TERR-LOC-001,
// TERR-LOAD-002, TERR-IDX-003, TERR-SEM-004, TERR-VER-005, and for the
// height displacement TERR-GEOM-031, TERR-GEOM-035, TERR-EDGE-024..026); the
// compositors, the canvas and its origin, scaling, placeholder fill and
// image output are the project's own engineering. See
// docs/0004-terrain-viewer/spec.md and
// docs/0012-height-displaced-terrain/spec.md.
//
// Tier: pkg/render/terrain imports the standard library only.
// See docs/ARCHITECTURE.md.
package terrain
