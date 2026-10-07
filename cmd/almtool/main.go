// Command almtool inspects ROM1 .alm maps (developer tool). It reads a raw
// .alm file from disk, decodes it with pkg/formats/alm, and dumps the result for
// verification against a lawful install. It is developer-run only and is never
// part of the test suite; it writes no asset and embeds no install path.
//
// Usage:
//
//	almtool info    <file.alm>              header, per-record typeId + payloadSize, tiling check
//	almtool meta    <file.alm>              W, H, name, description, content counts
//	almtool grid    <file.alm> <layer>      dump a grid layer: tiles | alt | overlay
//	almtool content <file.alm>              type5 names, type4/type6 counts + sample X/Y, type7 entryCount
//	almtool roster  <file.alm>              the player roster and the diplomacy matrix it authors:
//	                                        per record its 1-based slot, name, owned placements and
//	                                        sixteen raw words, then the effective matrix
//	almtool roundtrip <file.alm>            document round trip: size + md5 on identity, first differing offset on mismatch
//	almtool pass    <file.alm>              census of the derived passability plane: per byte value, per mover, per arm
//	almtool script  <file.alm>              the map's authored mission script: the decoded nodes, the compiled
//	                                        program, and every arm this build cannot evaluate
//	almtool loot    <file.alm>              the map's authored loot (type8): per record its kind, cell,
//	                                        gold and elements, and a closing census
package main

import (
	"crypto/md5"
	"fmt"
	"os"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(2)
	}
	cmd, path := os.Args[1], os.Args[2]

	var err error
	switch cmd {
	case "info":
		err = cmdInfo(path)
	case "meta":
		err = cmdMeta(path)
	case "grid":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "almtool grid: missing layer (tiles|alt|overlay)")
			os.Exit(2)
		}
		err = cmdGrid(path, os.Args[3])
	case "content":
		err = cmdContent(path)
	case "roster":
		err = cmdRoster(path)
	case "roundtrip":
		err = cmdRoundtrip(path)
	case "pass":
		err = cmdPass(path)
	case "script":
		err = cmdScript(path)
	case "loot":
		err = cmdLoot(path)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "almtool: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: almtool <info|meta|grid|content|roster|roundtrip|pass|script|loot> <file.alm> [layer]")
}

func loadMap(path string) (*alm.Map, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	m, err := alm.Open(data)
	if err != nil {
		return nil, nil, err
	}
	return m, data, nil
}

func cmdInfo(path string) error {
	m, data, err := loadMap(path)
	if err != nil {
		return err
	}
	fmt.Printf("magic:   M7R\\0 (valid)\n")
	fmt.Printf("hdrLen:  %d   recordCount: %d   formatVersion: %d\n", m.HdrLen, m.RecordCount, m.FormatVersion)
	fmt.Printf("dataSize: %d (header, read & ignored)\n", m.DataSize)
	fmt.Printf("records: %d (physical order)\n", len(m.Records))
	total := 20 // file header
	for i, r := range m.Records {
		fmt.Printf("  [%d] typeId=%d payloadSize=%d\n", i, r.TypeID, r.PayloadSize)
		total += 20 + int(r.PayloadSize)
	}
	fmt.Printf("tiling: 20 + sum(20+payloadSize) = %d, file size = %d (%s)\n",
		total, len(data), okStr(total == len(data)))
	return nil
}

func cmdMeta(path string) error {
	m, _, err := loadMap(path)
	if err != nil {
		return err
	}
	fmt.Printf("W x H:        %d x %d\n", m.Width, m.Height)
	fmt.Printf("name:         %q\n", m.Name)
	fmt.Printf("description:  %q\n", m.Description)
	fmt.Printf("#type5 (groups): %d\n", m.Meta.Count5)
	fmt.Printf("#type4 (objects): %d\n", m.Meta.Count4)
	fmt.Printf("#type6 (units):  %d\n", m.Meta.Count6)
	fmt.Printf("type0 record word +0x10: %#08x\n", m.MetaRecordWord)
	fmt.Printf("angle (raw, R-1):     %v\n", m.Angle)
	fmt.Printf("bitmask +0x18 (raw, R-1): %#x\n", m.Meta.Bitmask)
	return nil
}

func cmdGrid(path, layer string) error {
	m, _, err := loadMap(path)
	if err != nil {
		return err
	}
	w := m.Width
	if w <= 0 {
		return fmt.Errorf("grid: non-positive width %d", w)
	}
	switch layer {
	case "tiles":
		fmt.Printf("tiles %dx%d (u16: index=cell&0x3ff, impassable=cell&0x2000)\n", m.Width, m.Height)
		for y := 0; y < m.Height; y++ {
			for x := 0; x < w; x++ {
				cell := m.Tiles[y*w+x]
				mark := ' '
				if alm.Impassable(cell) {
					mark = '#'
				}
				fmt.Printf(" %04x/%03x%c", cell, alm.TileIndex(cell), mark)
			}
			fmt.Println()
		}
	case "alt":
		fmt.Printf("altitudes %dx%d (u8 height)\n", m.Width, m.Height)
		dumpU8(m.Altitudes, w, m.Height)
	case "overlay":
		fmt.Printf("object overlay %dx%d (u8; nonzero = occupied)\n", m.Width, m.Height)
		dumpU8(m.Overlay, w, m.Height)
	default:
		return fmt.Errorf("grid: unknown layer %q (want tiles|alt|overlay)", layer)
	}
	return nil
}

func dumpU8(cells []byte, w, h int) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fmt.Printf(" %3d", cells[y*w+x])
		}
		fmt.Println()
	}
}

func cmdContent(path string) error {
	m, _, err := loadMap(path)
	if err != nil {
		return err
	}
	fmt.Printf("groups (type5): %d\n", len(m.Groups))
	for i, g := range m.Groups {
		fmt.Printf("  [%d] %q scalar=%d\n", i, g.Name, g.Scalar)
	}

	fmt.Printf("objects (type4): %d\n", len(m.Objects))
	for i := 0; i < len(m.Objects) && i < 8; i++ {
		o := m.Objects[i]
		fmt.Printf("  [%d] tile=(%d,%d) kind=%#x ext=%v\n", i, o.X>>8, o.Y>>8, o.Kind, o.Ext != nil)
	}

	fmt.Printf("units (type6): %d\n", len(m.Units))
	for i := 0; i < len(m.Units) && i < 8; i++ {
		u := m.Units[i]
		fmt.Printf("  [%d] tile=(%d,%d)\n", i, u.X>>8, u.Y>>8)
	}

	fmt.Printf("triggers (type7): entryCount=%d body=%d bytes\n", m.Triggers.EntryCount, len(m.Triggers.Body))
	fmt.Printf("markers (type8):  body=%d bytes\n", len(m.LootSection.Body))
	fmt.Printf("tile markers (type9): count=%d body=%d bytes\n", m.TileMarkers.Count, len(m.TileMarkers.Body))
	return nil
}

func okStr(ok bool) string {
	if ok {
		return "OK"
	}
	return "MISMATCH"
}

// cmdRoundtrip opens the file as a raw document, writes it back, and compares
// the result against the input measured byte by byte. Identity prints the size
// and MD5 digest and returns nil; a mismatch or a rejection returns an error
// through the shared verb error path (almtool: ..., exit 1). With an honest
// Write the mismatch branch is unreachable in-process; it exists so that a
// corpus run's "identical" is a measurement, not an assumption.
func cmdRoundtrip(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	doc, err := alm.OpenDocument(data)
	if err != nil {
		return err
	}
	out := doc.Write()
	if off := firstDiff(data, out); off >= 0 {
		return fmt.Errorf("roundtrip: first difference at offset %d (input %d bytes, output %d bytes)", off, len(data), len(out))
	}
	fmt.Printf("roundtrip: identical, %d bytes, md5 %x\n", len(data), md5.Sum(data))
	return nil
}

// cmdPass censuses the passability plane the map on the command line describes:
// how many cells hold each byte value, how many block a ground and how many an
// air mover, and how many each of the five arms would block WITH THE OTHER FOUR
// ABSENT.
//
// The five arm counts overlap and are not meant to sum to either blocked count —
// a border cell carrying a water word is counted by both arms and is one blocked
// cell — so they are printed with their total beside them and the overlap stated
// as a number rather than left for a reader to be surprised by.
//
// It counts through pkg/mapload's own census, which walks the same per-cell
// classifier the plane is built from. A second reading of the arms written here
// would agree with the first by having been copied from it, and would measure
// nothing at all.
//
// The path is an argument, no install path is embedded, nothing is written, and
// this verb is not reachable from go test — the whole point of it is a run
// against a lawful install, whose FIGURES ship in a work item's evidence while
// the assets never do.
func cmdPass(path string) error {
	m, _, err := loadMap(path)
	if err != nil {
		return err
	}
	c := mapload.Census(m)
	cells := m.Width * m.Height

	fmt.Printf("passability %dx%d = %d cell(s)\n", m.Width, m.Height, cells)
	fmt.Printf("by value:   0x00 open %d   0x01 ground %d   0x02 air only %d   0x03 ground+air %d\n",
		c.Value[0], c.Value[1], c.Value[2], c.Value[3])
	fmt.Printf("by mover:   ground blocked %d   air blocked %d\n", c.Ground, c.Air)
	fmt.Printf("by arm (each with the other four absent, so these overlap):\n")
	fmt.Printf("  tile bit 13     %d\n", c.Impassable)
	fmt.Printf("  tile index 512..767 %d\n", c.Water)
	fmt.Printf("  strip group 7, blend level 3+ %d\n", c.Mountain)
	fmt.Printf("  nonzero overlay code %d\n", c.Scenery)
	fmt.Printf("  border, 8 cells deep %d\n", c.Border)
	arms := c.Impassable + c.Water + c.Mountain + c.Scenery + c.Border
	fmt.Printf("  arm total %d against %d cells blocking ground: %d cell(s) of overlap\n",
		arms, c.Ground, arms-c.Ground)
	return nil
}

// firstDiff returns the offset of the first byte at which a and b differ, or
// -1 when they are identical. When one is a proper prefix of the other, the
// first difference is the shorter length.
func firstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}
