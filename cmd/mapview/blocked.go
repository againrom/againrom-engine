package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/databin"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/vfs"
)

// worldArchive and dataBinPath are where the definition table lives: the node
// data/data.bin inside world.res, addressed below the archive's own identity.
//
// The identity comes from the host path the caller gave, so no install path is
// spelled in this file or any other (golden rule 3).
const (
	worldArchive = "world.res"
	dataBinPath  = "data/data.bin"
)

// blockGround is bit 0 of a derived block byte: the bit that stops a GROUND
// mover.
//
// It is spelt here, at the tier that reads one for the tint, exactly as
// pkg/render/terrain spells bit 1 for the margin and pkg/mapload spells both for
// the tier that writes one. The import graph puts a shared constant out of reach
// of all three, and a bare 1 at this read site would leave the bit's meaning
// written down nowhere on this side of the seam.
const blockGround byte = 1 << 0

// blockedCells is which cells of a derived plane are closed to a ground mover, in
// row-major order.
//
// It reads BIT 0 and no other bit. Bit 1 is the air mover's, which the margin
// alone sets and which no structure moves, so a cell on the ring that a structure
// has opened to a ground mover is NOT in this list while still being drawn dim —
// the two instruments say different true things about one cell, and neither is
// wrong.
//
// It is a named function rather than a loop inside the loader because it is the
// whole of what a criterion can be written against: a plane in, cells out, no
// archive, no window and no map.
//
// It is total. A plane shorter than the extent contributes only the cells it
// covers, a non-positive extent yields nothing, and no index is formed before its
// cell is known to be inside both.
func blockedCells(plane []byte, w, h int) []image.Point {
	if w <= 0 || h <= 0 {
		return nil
	}
	var out []image.Point
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if i >= len(plane) {
				return out
			}
			if plane[i]&blockGround != 0 {
				out = append(out, image.Pt(x, y))
			}
		}
	}
	return out
}

// resolveDataArchive is resolveArchive's sibling for the definition table: the
// -databin path when one was given, and <assets>/world.res otherwise.
func resolveDataArchive(assetsFlag, dataBinFlag string) (string, error) {
	if dataBinFlag != "" {
		return dataBinFlag, nil
	}
	root := game.ResolveAssetRoot(assetsFlag, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		return "", fmt.Errorf("-blocked needs a definition table; pass -assets, set AGAINROM_ASSETS, or give -databin")
	}
	return filepath.Join(root, worldArchive), nil
}

// buildingsTable opens an archive, parses the definition table inside it and
// returns the world builder's table carrying that table's Buildings collection.
//
// It carries Buildings ALONE. This tool builds no world and spawns no unit, so
// the two collections a unit placement searches would be read and never
// consulted; handing over only what is used is what keeps the failure surface to
// the one node this feature needs.
func buildingsTable(archivePath string) (*mapload.Table, error) {
	identity, err := vfs.Identity(archivePath)
	if err != nil {
		return nil, err
	}
	fsys, err := vfs.Open([]string{archivePath}, nil)
	if err != nil {
		return nil, err
	}
	address := identity + "/" + dataBinPath
	b, err := fsys.ReadFile(address)
	if err != nil {
		return nil, err
	}
	f, err := databin.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", address, err)
	}
	return &mapload.Table{Buildings: f.Collection(databin.Buildings)}, nil
}

// blockedOverlay derives the tint's cells for one decoded map against the table
// in one archive: the plane the five terrain arms AND that map's placed
// structures describe, reduced to the cells closed to a ground mover.
//
// It is the plane a player would walk, not the plane the viewer holds. The
// viewer's own grid.Block comes from the map-loading tier with no table at all,
// so it cannot tell a bridge deck from the water beneath it; that is the
// difference this overlay exists to show, and deriving the tint from it would
// draw the picture from before the story and call it the picture after.
func blockedOverlay(m *alm.Map, archivePath string) ([]image.Point, error) {
	tbl, err := buildingsTable(archivePath)
	if err != nil {
		return nil, err
	}
	return blockedCells(mapload.PassabilityWith(m, tbl), m.Width, m.Height), nil
}
