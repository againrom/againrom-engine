package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// castTargetFields are the actor record fields an original SAV holds for an
// order in progress: the completion byte, the countdown, the sub-phase, the
// target reference, the run end, the attack charge, the health hundredths,
// the credit reference and the credit spell (SAV-1321).
var castTargetFields = []string{"U136", "U6C", "U58", "U5C", "U138", "U134", "UA2", "U40", "U48"}

// castTargetActor is one Unit or Human record: its archive index, runtime ID,
// cell and named fields. A reference holds the archive index of the object
// whose Identity it names, or castTargetUnresolved.
type castTargetActor struct {
	Class    string
	Archive  uint16
	Runtime  uint32
	Col, Row int
	Fields   map[string]uint32
}

const castTargetUnresolved = 1 << 20

// castTargetActors reads every Unit and Human record of a SAV, keyed by its
// Identity.
func castTargetActors(raw []byte) (map[uint32]castTargetActor, error) {
	doc, origins, err := sav.DecodeDocumentDataWithOrigins(raw)
	if err != nil {
		return nil, err
	}
	archive := map[uint16]uint16{}
	for _, o := range origins {
		archive[o.ObjectIndex] = o.ArchiveIndex
	}
	byKey := map[uint32]uint16{}
	for i := range doc.Objects {
		if key, err := savedStructureValue(&doc.Objects[i], "Identity"); err == nil && key != 0 {
			byKey[key] = archive[uint16(i+1)]
		}
	}
	out := map[uint32]castTargetActor{}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" && r.Class != "Human" {
			continue
		}
		identity, err := savedStructureValue(r, "Identity")
		if err != nil {
			return nil, err
		}
		runtime, _ := savedStructureValue(r, "RuntimeID")
		position, err := savedMotionRaw(r, "Block12", 12)
		if err != nil {
			return nil, err
		}
		cell := binary.LittleEndian.Uint16(position)
		row := castTargetActor{Class: r.Class, Archive: archive[uint16(i+1)], Runtime: runtime, Col: int(cell & 0xff), Row: int(cell >> 8), Fields: map[string]uint32{}}
		for _, name := range castTargetFields {
			var v uint32
			if name == "U58" {
				b, err := savedMotionRaw(r, name, 4)
				if err != nil {
					return nil, err
				}
				v = binary.LittleEndian.Uint32(b)
			} else if v, err = savedStructureValue(r, name); err != nil {
				return nil, fmt.Errorf("%s of object %d: %v", name, i+1, err)
			}
			if (name == "U5C" || name == "U40") && v != 0 {
				index, ok := byKey[v]
				v = uint32(index)
				if !ok {
					v = castTargetUnresolved
				}
			}
			row.Fields[name] = v
		}
		if _, repeated := out[identity]; repeated || identity == 0 {
			return nil, fmt.Errorf("actor Identity %#x is not unique", identity)
		}
		out[identity] = row
	}
	return out, nil
}

// castTargetCase is one installed original SAV and, when the file holds one,
// the caster in the middle of a cast: its cell, the archive index its target
// word names, its sub-phase and its countdown (SAV-1321).
type castTargetCase struct {
	file, sha            string
	casterCol, casterRow int
	target, phase, count uint32
}

func castTargetCaseFor(t *testing.T) castTargetCase {
	t.Helper()
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the witness needs an installed original SAV")
	}
	for _, c := range []castTargetCase{
		{"game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b", 73, 108, 152, 7, 6},
		{"game0002.sav", "b1ce079cc2b3f1bc101862c1dcf2afd8237421458b3e2474c4447efe5df2e761", -1, -1, 0, 0, 0},
	} {
		if _, err := os.Stat(filepath.Join(root, c.file)); err == nil {
			return c
		}
	}
	t.Skip("the install holds no witness SAV")
	return castTargetCase{}
}

// castTargetCompare reports every named field of an input actor record that
// the SAVE does not hold.
func castTargetCompare(t *testing.T, before, after map[uint32]castTargetActor) {
	t.Helper()
	differ := 0
	for key, a := range before {
		b, ok := after[key]
		if !ok {
			t.Errorf("actor %#x (%s object %d at %d,%d) is absent from the SAVE", key, a.Class, a.Archive, a.Col, a.Row)
			continue
		}
		for _, name := range castTargetFields {
			if a.Fields[name] != b.Fields[name] {
				differ++
				t.Errorf("%s object %d runtime %d at (%d,%d) %s: input %d, SAVE %d", a.Class, a.Archive, a.Runtime, a.Col, a.Row, name, a.Fields[name], b.Fields[name])
			}
		}
	}
	t.Logf("%d actor records, %d named fields differ", len(before), differ)
}

func castTargetAt(actors map[uint32]castTargetActor, col, row int) (castTargetActor, bool) {
	for _, a := range actors {
		if a.Col == col && a.Row == row && a.Runtime != 0 {
			return a, true
		}
	}
	return castTargetActor{}, false
}

// castTargetEntity is the living actor at the cell.
func castTargetEntity(t *testing.T, f *FrontEnd, col, row int) sim.EntityID {
	t.Helper()
	for _, e := range f.live.world.Entities() {
		if e.Alive() && int(e.X) == col && int(e.Y) == row {
			return e.ID
		}
	}
	t.Fatalf("no living actor at (%d,%d)", col, row)
	return 0
}

// TestReleaseSAVKeepsCastingActorFields loads an installed original SAV, saves
// it at once with F2 and compares the named fields of every actor record. On
// game0018 the caster keeps its target, sub-phase and countdown; the SAVE
// loaded cold runs in lockstep with the session that loaded the original, and
// the caster's next cast is the same in both. A SAVE after that cast writes
// the caster's live target.
func TestReleaseSAVKeepsCastingActorFields(t *testing.T) {
	c := castTargetCaseFor(t)
	in, err := os.ReadFile(filepath.Join(os.Getenv("AGAINROM_ASSETS"), c.file))
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(in)); got != c.sha {
		t.Fatalf("%s changed: %s", c.file, got)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, c.file), in, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := castOrderSession(t, dir)
	savedDir, out := castOrderF2Save(t, f, app, "cast target")
	before, err := castTargetActors(in)
	if err != nil {
		t.Fatal(err)
	}
	after, err := castTargetActors(out)
	if err != nil {
		t.Fatal(err)
	}
	castTargetCompare(t, before, after)
	if c.casterCol < 0 {
		return
	}
	caster, ok := castTargetAt(after, c.casterCol, c.casterRow)
	if !ok || caster.Fields["U5C"] != c.target || caster.Fields["U58"] != c.phase || caster.Fields["U6C"] != c.count {
		t.Fatalf("the SAVE's caster at (%d,%d): %+v; want target object %d, sub-phase %d, countdown %d", c.casterCol, c.casterRow, caster, c.target, c.phase, c.count)
	}
	g, gapp := castOrderSession(t, savedDir)
	id := castTargetEntity(t, f, c.casterCol, c.casterRow)
	if cold := castTargetEntity(t, g, c.casterCol, c.casterRow); cold != id {
		t.Fatalf("the cold LOAD binds the caster to entity %d, the original LOAD to %d", cold, id)
	}
	var cast sim.CastEvent
	tick := castOrderLockstep(t, f, g, 256, func(h *FrontEnd, events []sim.CastEvent) bool {
		for _, ev := range events {
			if ev.Caster == id {
				if h == f {
					cast = ev
				} else if ev.Target != cast.Target || ev.Spell != cast.Spell || ev.AtCell != cast.AtCell {
					t.Fatalf("cold LOAD cast %+v, original LOAD cast %+v", ev, cast)
				}
				return true
			}
		}
		return false
	})
	if cast.AtCell {
		t.Fatalf("the caster's next cast is at a cell: %+v", cast)
	}
	_, later := castOrderF2Save(t, g, gapp, "after cast")
	now, err := castTargetActors(later)
	if err != nil {
		t.Fatal(err)
	}
	moved, ok := castTargetAt(now, c.casterCol, c.casterRow)
	target, _ := g.live.world.Entity(cast.Target)
	named, _ := castTargetAt(now, int(target.X), int(target.Y))
	// After the cast the World writes the caster's own state: the target of a
	// cast it holds, else none. The loaded order it held is gone.
	e, _ := g.live.world.Entity(id)
	want := uint32(0)
	if book, casting := castOrderBook(g.live.world, id); casting && !book.AtCell {
		held, _ := g.live.world.Entity(book.Target)
		current, _ := castTargetAt(now, int(held.X), int(held.Y))
		want = uint32(current.Archive)
	}
	if !ok || e.HeldOrder.Kind != sim.HeldOrderNone || moved.Fields["U5C"] != want {
		t.Fatalf("the SAVE after the cast: caster %+v, held order %+v; want target word %d", moved, e.HeldOrder, want)
	}
	t.Logf("caster entity %d, runtime %d: next cast %d ticks after LOAD, spell %d at entity %d (runtime %d, object %d); the SAVE then writes target %d, sub-phase %d, countdown %d",
		id, caster.Runtime, tick, cast.Spell, cast.Target, named.Runtime, named.Archive, moved.Fields["U5C"], moved.Fields["U58"], moved.Fields["U6C"])
}
