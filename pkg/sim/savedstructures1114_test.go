package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestSavedStructures1114NativeSpatialStateAndNextAction(t *testing.T) {
	a := structureCombatActor()
	a.ID, a.X, a.Y, a.Owner, a.Domain = 1, 10, 10, SelfSlot, DomainGhost
	w := structureAreaWorld(t, nil, a)
	grid := make([]byte, len(w.grid))
	for i := range grid {
		grid[i] = blockAir
	}
	s := []Structure{{ID: 7, Col: 20, Row: 20, Width: 0, Height: 3, Attach: 5, Field42: 0xffff}}
	meta := savedStructureSources1114(s)
	if err := w.ImportOriginalStructures(s, meta, []SavedStructureCell{{Cell: 0x0a0b, ID: 7, HasStructure: true}}, grid); err != nil {
		t.Fatal(err)
	}
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 12, Y: 10}})
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var fresh World
	if err := fresh.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if fresh.Hash() != w.Hash() || !reflect.DeepEqual(fresh.structureSlots, w.structureSlots) {
		t.Fatal("native saved spatial identity lost")
	}
	for range 64 {
		Step(w, nil)
		Step(&fresh, nil)
		if fresh.Hash() != w.Hash() {
			t.Fatal("next movement differs")
		}
	}
	if e := fresh.entities[0]; e.X != 12 || e.Y != 10 {
		t.Fatal("saved Ghost used old Air mask", e)
	}
	bad := append([]byte(nil), form...)
	binary.LittleEndian.PutUint32(bad[len(bad)-9:], 1234)
	before := fresh.Hash()
	if err := fresh.UnmarshalBinary(bad); err == nil || fresh.Hash() != before {
		t.Fatal("late native cell fault partially adopted", err)
	}
	for _, change := range []func(*World){
		func(w *World) { w.savedStructures[0].Base52[0]++ },
		func(w *World) { w.savedStructures[0].RuntimeID++ },
		func(w *World) { w.savedStructures[0].Reference++ },
		func(w *World) { w.savedStructures[0].SourceKey++ },
		func(w *World) { w.savedStructures[0].Kind++ },
		func(w *World) { w.savedStructureCells[0].BaselineCost++ },
		func(w *World) { w.savedStructureCells[0].BaselineStatic++ },
		func(w *World) { w.savedStructureCells[0].Cell++ },
	} {
		var x World
		if err := x.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		h := x.Hash()
		change(&x)
		if h == x.Hash() {
			t.Fatal("saved metadata outside hash")
		}
	}
	empty := structureAreaWorld(t, nil)
	oldHash := empty.Hash()
	if err := empty.ImportOriginalStructures(nil, nil, nil, empty.grid); err != nil {
		t.Fatal(err)
	}
	if oldHash == empty.Hash() {
		t.Fatal("absent and present empty mode hash equally")
	}
}

// Independent current-form extension for fixed older byte pins. An absent
// structure payload is exactly one zero uint32, never a inferred roster.
func widenedSavedStructurePin(old []byte) []byte {
	out := append(append([]byte(nil), old...), 0, 0, 0, 0)
	out[0] = 79
	return widenedSavedGroupPlayerPin(out)
}

func savedStructureSources1114(structures []Structure) []SavedStructure {
	out := make([]SavedStructure, len(structures))
	for i, s := range structures {
		out[i] = SavedStructure{ID: s.ID, Class: SavedBuilding, SourceKey: uint32(900 + i), ArchiveIndex: uint16(3 + i),
			Position: [12]byte{byte(s.Col), byte(s.Row)}, Kind: 1, Blocking: 5}
		out[i].Base52[14], out[i].Base52[15] = s.Width, s.Height
		binary.LittleEndian.PutUint32(out[i].Base52[18:], 5)
	}
	return out
}

func TestSavedStructures1114ExplicitLinksAndRealDamage(t *testing.T) {
	structures := []Structure{{ID: 0, Col: 10, Row: 10, Width: 3, Height: 2, Attach: 63, Field42: 100, MaxHealth: 100},
		{ID: 7, Col: 20, Row: 20, Width: 2, Height: 2, Attach: 15, Field42: 80, MaxHealth: 80}}
	a := structureCombatActor()
	a.X, a.Y = 9, 10
	a.Owner = SelfSlot
	w := structureAreaWorld(t, nil, a)
	cells := []SavedStructureCell{{Cell: 0x0a0a, HasStructure: true, ID: 0}, {Cell: 0x0a0b, HasStructure: true, ID: 0},
		{Cell: 0x0a0b, HasStructure: true, ID: 7}, // alias outside structure7's rectangle
		{Cell: 0x0a0a}, // explicit null wins; no footprint reconstruction
		{Cell: 0xffff, HasStructure: true, ID: 7}}
	if err := w.ImportOriginalStructures(structures, savedStructureSources1114(structures), cells, w.grid); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.structureSlots, map[uint16]int{0x0a0b: 1, 0xffff: 1}) {
		t.Fatalf("explicit slots %v", w.structureSlots)
	}
	w.applyStructureSpellAt(10, 10, SpellRule{DamageMin: 20, DamageMax: 21}, 0)
	if w.structures[0].Field42 != 100 {
		t.Fatal("cleared anchor was hit")
	}
	w.applyStructureSpellAt(11, 10, SpellRule{DamageMin: 20, DamageMax: 21}, 0)
	if w.structures[1].Field42 != 65 {
		t.Fatalf("off-rectangle alias target HP %d", w.structures[1].Field42)
	}
	Step(w, []Command{{Kind: KindAttackStructure, Entity: a.ID, X: 0}})
	for range 64 {
		Step(w, nil)
	}
	if w.structures[0].Field42 == 100 {
		t.Fatal("physical order did not strike current anchor")
	}
}

func TestSavedStructures1114ExactDomainMasksAndLegacyMode(t *testing.T) {
	for raw := byte(0); raw < 8; raw++ {
		w := structureAreaWorld(t, nil)
		grid := make([]byte, len(w.grid))
		grid[10*40+10] = raw&3 | (raw&4)<<1
		if err := w.ImportOriginalStructures(nil, nil, nil, grid); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			d    Domain
			mask byte
		}{{DomainGround, 1}, {DomainGhost, 4}, {DomainAir, 2}} {
			want := raw&tc.mask == 0
			if w.terrainOpen(tc.d, 10, 10) != want || w.terrainOpenFootprint(Entity{Domain: tc.d}, 10, 10) != want {
				t.Fatalf("raw%02x domain%d want open%t", raw, tc.d, want)
			}
		}
	}
	w := structureAreaWorld(t, nil)
	w.grid[410] = blockAir
	if w.terrainOpen(DomainGhost, 10, 10) {
		t.Fatal("legacy Ghost policy changed")
	}
	if _, err := newGrid(Bounds{Width: 1, Height: 1}, []byte{blockStaticObject}); err == nil {
		t.Fatal("legacy reserved grid bit accepted")
	}
}

func TestSavedStructures1114RealAreaCastUsesOnlySavedAliases(t *testing.T) {
	actor := effectMage(1, 6, 10, 1<<2)
	rule := SpellRule{ID: 2, Area: true, Radius: 1, School: 1, MaxRange: 6, ManaCost: 1, Damaging: true, DamageMin: 20, DamageMax: 21}
	w, err := NewSpelledWorld(1114, Bounds{Width: 40, Height: 40}, ModeCanonical, nil, []Entity{actor}, nil, []SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	s := []Structure{{ID: 0, Col: 10, Row: 10, Width: 3, Height: 3, Attach: 511, Field42: 100, MaxHealth: 100},
		{ID: 7, Col: 20, Row: 20, Width: 2, Height: 2, Attach: 15, Field42: 80, MaxHealth: 80}}
	if err := w.ImportOriginalStructures(s, savedStructureSources1114(s), []SavedStructureCell{{Cell: 0x0a0a, ID: 7, HasStructure: true}}, w.grid); err != nil {
		t.Fatal(err)
	}
	Step(w, []Command{{Kind: KindCastAt, Entity: 1, X: 10, Y: 10, Spell: 2}})
	if len(w.bookCasts) != 1 {
		t.Fatal("real area cast not admitted")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var fresh World
	if err := fresh.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	for range 128 {
		Step(w, nil)
		Step(&fresh, nil)
		if w.Hash() != fresh.Hash() {
			t.Fatal("pending area cast changed after native load")
		}
	}
	// Seed1114's first SplitMix draw is odd: spread1 rolls1, so this one
	// saved alias takes 20+1-5=16 damage. The geometric owner takes none.
	if w.structures[0].Field42 != 100 || w.structures[1].Field42 != 64 {
		t.Fatalf("cast used footprint instead of source alias %+v", w.structures)
	}
}

func TestSavedStructures1114ImportRefusalIsAtomic(t *testing.T) {
	s := []Structure{{ID: 7, Col: 10, Row: 10, Width: 1, Height: 1, Attach: 1, Field42: 100, MaxHealth: 100}}
	for _, mutate := range []func([]SavedStructure, []SavedStructureCell){
		func(s []SavedStructure, c []SavedStructureCell) { s[0].SourceKey = 0 },
		func(s []SavedStructure, c []SavedStructureCell) { s[0].Class = 99 },
		func(s []SavedStructure, c []SavedStructureCell) { s[0].Position[0]++ },
		func(s []SavedStructure, c []SavedStructureCell) { c[1].ID = 8 },
	} {
		w := structureAreaWorld(t, s)
		before := w.Hash()
		meta := savedStructureSources1114(s)
		cells := []SavedStructureCell{{Cell: 0x0a0a, ID: 7, HasStructure: true}, {Cell: 0x0a0b, ID: 7, HasStructure: true}}
		mutate(meta, cells)
		if err := w.ImportOriginalStructures(s, meta, cells, w.grid); err == nil {
			t.Fatal("malformed import accepted")
		}
		if w.Hash() != before || w.hasSavedStructures {
			t.Fatal("failed import mutated world")
		}
	}
}

func TestSavedStructures1114FooterChecksEveryTruncationAndFlags(t *testing.T) {
	s := []Structure{{ID: 7, Col: 10, Row: 10, Width: 1, Height: 1, Attach: 1}}
	w := structureAreaWorld(t, s)
	meta := savedStructureSources1114(s)
	meta[0].Class = SavedOutpost
	meta[0].OutpostWords = [4]uint32{10, 20, 30, 40}
	meta[0].OutpostRecords = [][8]byte{{1, 2, 3, 4, 5, 6, 7, 8}}
	if err := w.ImportOriginalStructures(s, meta, []SavedStructureCell{{Cell: 0xffff, ID: 7, HasStructure: true}}, w.grid); err != nil {
		t.Fatal(err)
	}
	b := w.appendSavedStructureSection(make([]byte, headerLen))
	body, present, got, cells, err := splitSavedStructureSection(b)
	if err != nil || !present || len(body) != headerLen || !reflect.DeepEqual(got, meta) || len(cells) != 1 {
		t.Fatalf("footer %+v %v", got, err)
	}
	for end := headerLen + 1; end < len(b)-4; end++ {
		partial := binary.LittleEndian.AppendUint32(append([]byte(nil), b[:end]...), uint32(end-headerLen))
		if _, _, _, _, err := splitSavedStructureSection(partial); err == nil {
			t.Fatalf("accepted payload truncation %d", end)
		}
	}
	for _, at := range []int{headerLen + 4 + 19, len(b) - 5} {
		bad := append([]byte(nil), b...)
		bad[at] = 2
		if _, _, _, _, err := splitSavedStructureSection(bad); err == nil {
			t.Fatalf("noncanonical flag %d accepted", at)
		}
	}
}
