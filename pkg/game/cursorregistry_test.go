package game

// The 28-slot cursor registry's own load, from archive bytes to a resolved
// pkg/ui.CursorRegistry (docs/1030-cursor-lifecycle B1).
//
// This is an internal test because cursorRegistrations is unexported: the
// order/name/hotspot/period table this file compares LoadCursorRegistry's
// output against. openContainers and the .16a/.256 stream builders live in
// package game_test (statics_test.go, cursor_test.go) and are rebuilt here on
// sackGraphicsContainers' own precedent (sacks_test.go).
//
// EVERY FIXTURE IS SYNTHETIC. No game install is read.

import (
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/vfs"
)

func regGraphicsContainers(t *testing.T, files []synth.File) vfsFSCloser {
	t.Helper()
	path := filepath.Join(t.TempDir(), GraphicsArchive)
	if err := os.WriteFile(path, synth.Archive(files), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	f, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatalf("vfs.Open(%s): %v", path, err)
	}
	return f
}

// vfsFSCloser is *vfs.FS, named locally so this file need not import the type
// twice under two names; terrain.EntrySource (LoadCursorRegistry's parameter)
// is satisfied by *vfs.FS the same way every other loader in this package
// takes it.
type vfsFSCloser = *vfs.FS

// --- .16a fixture, rebuilt from cursor_test.go's own grammar spelling ---

const (
	regCurOpLiteral = 0 << 14
)

func regCurPixel(index, level int) uint16 { return uint16(index<<1) | uint16(level<<9) }

func regCurPalette() []byte {
	b := make([]byte, 1024)
	for k := 0; k < 256; k++ {
		b[k*4+0] = byte(3 * k)
		b[k*4+1] = byte(2 * k)
		b[k*4+2] = byte(k)
	}
	return b
}

// regCurSheet16A builds a one-frame, 1x1 .16a stream: a single painted pixel,
// so a decode succeeds with exactly one frame regardless of the row's own
// registered period — this is the fixture the period/frame-count independence
// test (B3's named trap) reads from.
func regCurSheet16A() []byte {
	controls := []uint16{regCurOpLiteral | 1, regCurPixel(5, 15)}
	block := make([]byte, 0, len(controls)*2)
	for _, c := range controls {
		block = binary.LittleEndian.AppendUint16(block, c)
	}
	out := regCurPalette()
	out = binary.LittleEndian.AppendUint32(out, 1) // width
	out = binary.LittleEndian.AppendUint32(out, 1) // height
	out = binary.LittleEndian.AppendUint32(out, uint32(len(block)))
	out = append(out, block...)
	return binary.LittleEndian.AppendUint32(out, 1|0x80000000) // 1 frame, has-palette
}

// --- .256 fixture, rebuilt from spr256's own byte contract (spr256.go doc) ---

func regU32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

// regCurSheet256 builds a one-frame, 1x1 .256 stream: a palette, one frame
// record whose block is a single opaque literal pixel, and the trailer.
func regCurSheet256() []byte {
	block := []byte{0x01, 0x00} // rleLiteral, count=1; one palette-index byte
	frame := append(regU32(1), regU32(1)...)
	frame = append(frame, regU32(uint32(len(block)))...)
	frame = append(frame, block...)
	out := append(regCurPalette(), frame...)
	return append(out, regU32(1|0x80000000)...) // 1 frame, has-palette
}

// regAllSlotsArchive builds one synthetic entry per row of cursorRegistrations,
// each a single-frame sheet in the row's own format, at the row's own path.
func regAllSlotsArchive(t *testing.T) []synth.File {
	t.Helper()
	files := make([]synth.File, 0, len(cursorRegistrations))
	for _, reg := range cursorRegistrations {
		var data []byte
		switch reg.format {
		case cursor16a:
			data = regCurSheet16A()
		case cursor256:
			data = regCurSheet256()
		default:
			t.Fatalf("cursor row %q: unhandled format", reg.name)
		}
		files = append(files, synth.File{Path: reg.path, Data: data})
	}
	return files
}

// TestCursorRegistrationsSlotOrderAndArrowTrap guards B1's own named trap: the
// eight arrow slots are NOT in arrow-number order (SPR16A-CURSOR-067). The
// expected list is authored independently here, not derived from
// cursorRegistrations, so a re-sort of the production table fails this test.
func TestCursorRegistrationsSlotOrderAndArrowTrap(t *testing.T) {
	want := []string{
		"default", "move", "swarm", "attack", "defend", "select", "patrol", "cast", "pickup",
		"arrow0", "arrow4", "arrow6", "arrow2", "arrow7", "arrow5", "arrow1", "arrow3",
		"sdefault", "smove", "sattack", "sdefend", "spatrol", "scast",
		"cantput", "town", "dice", "wait", "backpack",
	}
	if len(cursorRegistrations) != len(want) {
		t.Fatalf("cursorRegistrations has %d rows, want %d", len(cursorRegistrations), len(want))
	}
	for i, name := range want {
		if cursorRegistrations[i].name != name {
			t.Errorf("slot %d = %q, want %q", i, cursorRegistrations[i].name, name)
		}
	}
}

// TestCursorRegistrationsNameOneSheetEach is the half of "every slot has art of
// its own" that needs no install: 28 registrations naming 28 distinct sheets.
// Two names pointing at one path would give the two slots the same picture and
// pass every count, order and hotspot check in this package.
//
// The install-side half is the pixel comparison in
// cursorregistry_release_test.go: a loader that silently fell back to one
// default sheet would give identical PIXELS under distinct paths, which this
// test cannot see.
//
// MUTATION THIS FAILS AGAINST: repointing any one row's path at another row's.
func TestCursorRegistrationsNameOneSheetEach(t *testing.T) {
	byPath := map[string]string{}
	byName := map[string]bool{}
	n16a, n256 := 0, 0
	for _, r := range cursorRegistrations {
		if other, dup := byPath[r.path]; dup {
			t.Errorf("slots %s and %s both name %s, so they resolve to the same picture", other, r.name, r.path)
		}
		byPath[r.path] = r.name
		if byName[r.name] {
			t.Errorf("slot name %s is registered twice", r.name)
		}
		byName[r.name] = true
		switch r.format {
		case cursor16a:
			n16a++
		case cursor256:
			n256++
		}
	}
	if len(byPath) != 28 {
		t.Errorf("%d distinct sheet paths over %d registrations, want 28 (cursor-construction.tsv)", len(byPath), len(cursorRegistrations))
	}
	if n16a != 23 || n256 != 5 {
		t.Errorf("%d `.16a` and %d `.256` registrations, want 23 (SPR16A-CURSOR-046) and 5 (SPR256-CURSOR-046)", n16a, n256)
	}
}

// TestLoadCursorRegistryPreservesOrderHotspotAndPeriod builds one synthetic
// frame per registered row and requires LoadCursorRegistry's own output to
// carry the same order, name, hotspot and period the table names — and a
// FrameCount that comes from the DECODE (1, this fixture's own frame count)
// regardless of how large or small the row's registered period is. "pickup"
// is checked by name for the period/frame-count independence trap: its period
// (66ms, the fastest in the table) has no bearing on the fixture's frame
// count, which is 1 like every other row's fixture here.
func TestLoadCursorRegistryPreservesOrderHotspotAndPeriod(t *testing.T) {
	src := regGraphicsContainers(t, regAllSlotsArchive(t))
	reg, err := LoadCursorRegistry(src)
	if err != nil {
		t.Fatalf("LoadCursorRegistry: %v", err)
	}
	if len(reg.Slots) != len(cursorRegistrations) {
		t.Fatalf("got %d slots, want %d", len(reg.Slots), len(cursorRegistrations))
	}
	for i, row := range cursorRegistrations {
		got := reg.Slots[i]
		if got.Name != row.name {
			t.Errorf("slot %d name = %q, want %q (order not preserved)", i, got.Name, row.name)
		}
		if got.Hotspot != image.Pt(row.hotspotX, row.hotspotY) {
			t.Errorf("slot %d (%s) hotspot = %v, want %v", i, row.name, got.Hotspot, image.Pt(row.hotspotX, row.hotspotY))
		}
		if got.PeriodMillis != row.periodMillis {
			t.Errorf("slot %d (%s) period = %d, want %d", i, row.name, got.PeriodMillis, row.periodMillis)
		}
		if got.FrameCount != 1 || len(got.Frames) != 1 {
			t.Errorf("slot %d (%s) FrameCount/len(Frames) = %d/%d, want 1/1 from this fixture's own decode",
				i, row.name, got.FrameCount, len(got.Frames))
		}
	}
	pickup, ok := reg.Slot("pickup")
	if !ok {
		t.Fatal("registry does not resolve \"pickup\" by name")
	}
	if pickup.PeriodMillis != 66 {
		t.Fatalf("pickup period = %d, want 66 (SPR16A-CURSOR-046)", pickup.PeriodMillis)
	}
	if pickup.FrameCount != 1 {
		t.Fatalf("pickup FrameCount = %d, want 1: a fast registered period does not imply a multi-frame sheet", pickup.FrameCount)
	}
}

// TestLoadCursorRegistryNoSource: LoadAttackPointer's own rule (cursor.go) —
// a nil source is refused rather than read.
func TestLoadCursorRegistryNoSource(t *testing.T) {
	if _, err := LoadCursorRegistry(nil); err == nil {
		t.Error("a nil source loaded a registry")
	}
}

// TestLoadCursorRegistryFailsWholeOnTheFirstMissingSlot: an archive with no
// entries at all fails on the first row ("default") and returns no registry,
// on LoadAttackPointer's own "every failure is an error and none is a
// partial picture" rule, applied to the whole 28-slot table.
func TestLoadCursorRegistryFailsWholeOnTheFirstMissingSlot(t *testing.T) {
	src := regGraphicsContainers(t, nil)
	reg, err := LoadCursorRegistry(src)
	if err == nil {
		t.Fatal("an archive with no cursor entries loaded a registry")
	}
	if reg != nil {
		t.Error("a failed load returned a registry as well as an error")
	}
	if !strings.Contains(err.Error(), "default") {
		t.Errorf("error %v does not name the first slot it failed at", err)
	}
}

// TestLoadCursorRegistryFailsWholeOnOneMalformedSlot: 27 good rows and one
// (mid-table, "attack") whose stream will not decode. The whole registry
// fails, named by the slot that failed, and no partial 27/28 registry is
// returned.
func TestLoadCursorRegistryFailsWholeOnOneMalformedSlot(t *testing.T) {
	files := regAllSlotsArchive(t)
	broken := false
	for i, f := range files {
		if f.Path == "cursors/attack/sprites.16a" {
			// A literal run announcing more words than its block holds:
			// refused at the read, before any frame is produced
			// (spr16.DecodeA's own rejection, exercised in cursor_test.go).
			bad := regCurPalette()
			bad = binary.LittleEndian.AppendUint32(bad, 2)
			bad = binary.LittleEndian.AppendUint32(bad, 2)
			bad = binary.LittleEndian.AppendUint32(bad, 2)
			bad = binary.LittleEndian.AppendUint16(bad, regCurOpLiteral|9)
			bad = binary.LittleEndian.AppendUint32(bad, 1|0x80000000)
			files[i].Data = bad
			broken = true
		}
	}
	if !broken {
		t.Fatal("test setup: did not find the \"attack\" row's fixture entry to corrupt")
	}

	src := regGraphicsContainers(t, files)
	reg, err := LoadCursorRegistry(src)
	if err == nil {
		t.Fatal("an archive with one malformed cursor stream loaded a registry")
	}
	if reg != nil {
		t.Error("a failed load returned a registry as well as an error")
	}
	if !strings.Contains(err.Error(), "attack") {
		t.Errorf("error %v does not name the slot it failed at", err)
	}
}
