package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// flightStateNodes lists the Projectiles and Prj leaves of a SAV in YA1
// record order with the Store offset of each value word. It reads the record
// table itself; no production reader supplies it.
func flightStateNodes(t *testing.T, raw []byte) ([]string, map[string]int) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	b := file.Store
	u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }
	n := int(u32(16))
	pool := b[24+32*n+4:]
	var lines []string
	offsets := map[string]int{}
	var walk func(first, count uint32, parent string)
	walk = func(first, count uint32, parent string) {
		if int(first)+int(count) > n {
			t.Fatal("YA1 child range outside the table")
		}
		for i := first; i < first+count; i++ {
			off := 24 + 32*int(i)
			name := b[off+16 : off+32]
			if end := bytes.IndexByte(name, 0); end >= 0 {
				name = name[:end]
			}
			path := parent + "/" + string(name)
			value, size, kind := u32(off+4), u32(off+8), u32(off+12)
			if kind&1 != 0 {
				walk(value, size, path)
				continue
			}
			if !strings.HasPrefix(path, "/Prj") && !strings.HasPrefix(path, "/Projectiles/") {
				continue
			}
			text := fmt.Sprint(int32(value))
			if kind&14 == 6 {
				text = fmt.Sprintf("%x", pool[value:value+size])
			}
			lines = append(lines, fmt.Sprintf("%s kind%d %s", path, kind, text))
			offsets[path] = off + 4
		}
	}
	walk(u32(4), u32(8), "")
	return lines, offsets
}

// flightEffectLines lists every class, value and raw field of the effect
// records reachable from the world's effect list.
func flightEffectLines(t *testing.T, raw []byte) []string {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.World == nil {
		t.Fatal("effect Document", err)
	}
	var lines []string
	seen := map[uint16]bool{}
	var visit func(index uint16)
	visit = func(index uint16) {
		if index == 0 || seen[index] {
			return
		}
		seen[index] = true
		o := doc.Objects[index-1]
		lines = append(lines, fmt.Sprintf("%s %v", o.Class, o.Values))
		for _, r := range o.Raw {
			lines = append(lines, fmt.Sprintf("%s %s %x", o.Class, r.Name, r.Bytes))
		}
		for _, slot := range o.RefSlots {
			for _, child := range slot.Objects {
				if child != 0 && (strings.Contains(doc.Objects[child-1].Class, "Effect") || doc.Objects[child-1].Class == "SpellTransport") {
					visit(child)
				}
			}
		}
	}
	for _, index := range doc.World.Effects {
		visit(index)
	}
	return lines
}

func flightResave(t *testing.T, raw []byte) (sim.SavedProjectiles, []byte) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0018.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	app := f.App("flight records")
	app.Layout(1024, 768)
	save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game0018.sav")
	loaded := f.live.world.SavedProjectiles()
	drivers := f.live.world.SavedWorldEffectDrivers()
	if drivers == nil || len(drivers.Projectiles) != 1 || !drivers.Projectiles[0].HasTarget || drivers.Projectiles[0].Retired {
		t.Fatal("loaded record lost its target binding", drivers, loaded)
	}
	_, _, written := menuSAVE(t, f, app, OriginalStore{Dir: dir})
	return loaded, written
}

// The original's own mid-cast SAVE: our LOAD then SAVE with no tick between
// writes the projectile record, allocator, IDs and the pending cast's effect
// records as the original wrote them (SAV-1202, SAV-1322, MAGIC-413). A
// record whose action bytes and target word carry high bits keeps their low
// byte or word at LOAD and writes them back sign- or zero-extended (SAV-1322).
func TestReleaseOriginalFlightRecordsSurviveLoadAndSave(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	_, raw := groundCorpusFile(t, "2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b")
	source, offsets := flightStateNodes(t, raw)
	effects := flightEffectLines(t, raw)
	if len(source) != 18 || len(effects) < 3 || !strings.HasPrefix(effects[0], "SpellTransport ") {
		t.Fatalf("source records changed: %v %v", source, effects)
	}
	loaded, written := flightResave(t, raw)
	if got, _ := flightStateNodes(t, written); !slices.Equal(got, source) {
		t.Fatalf("SAVE wrote projectile records\n%v\nwant\n%v", got, source)
	}
	if got := flightEffectLines(t, written); !slices.Equal(got, effects) {
		t.Fatalf("SAVE wrote effect records\n%v\nwant\n%v", got, effects)
	}

	wide := map[string]uint32{"/Prj266/action": 0x101, "/Prj266/actiondir": 0x305, "/Prj266/actiontarget": 0x2009d, "/Projectiles/FreeIndex": 0x3010b}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	file.Store = bytes.Clone(file.Store)
	for path, value := range wide {
		binary.LittleEndian.PutUint32(file.Store[offsets[path]:], value)
	}
	wideRaw := file.Marshal()
	if got, _ := flightStateNodes(t, wideRaw); slices.Equal(got, source) {
		t.Fatal("loss control: the wide input equals the source")
	}
	// Without the low byte and word the target 0x2009d binds no actor and the
	// record would not continue.
	wideLoaded, wideWritten := flightResave(t, wideRaw)
	if p := wideLoaded.Items[0]; wideLoaded.FreeIndex != 267 || p != loaded.Items[0] {
		t.Fatalf("LOAD kept %+v free %d, want %+v", p, wideLoaded.FreeIndex, loaded.Items[0])
	}
	if got, _ := flightStateNodes(t, wideWritten); !slices.Equal(got, source) {
		t.Fatalf("SAVE of the wide record wrote %v, want %v", got, source)
	}

	carried := alterSAV(t, raw, func(doc *sav.DocumentData) bool {
		for i := range doc.Objects {
			if doc.Objects[i].Class == "SpellTransport" {
				savedObjectSetValue(&doc.Objects[i], "T0C", 0x55)
				return true
			}
		}
		return false
	})
	_, carriedWritten := flightResave(t, carried)
	if got := flightEffectLines(t, carriedWritten); slices.Equal(got, effects) || !strings.Contains(got[0], "{T0C 85}") {
		t.Fatalf("loss control: a changed transport Token +0c was not carried: %v", got)
	}
	t.Logf("game0018 Prj266, FreeIndex, IDs and %d effect lines equal after LOAD and SAVE; wide action/actiondir/actiontarget/FreeIndex keep their low byte or word", len(effects))
}
