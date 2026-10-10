package game

import (
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// overloadOrderParty is the Haste party with the companion carrying the
// installed table's heaviest item until the load is at least sixteen times
// the largest capacity his body admits, so the overload floor binds.
func overloadOrderParty(t *testing.T, f *FrontEnd) []mapload.PartyMember {
	t.Helper()
	party := hasteParty()
	code, heaviest := uint16(0), int32(0)
	for c := range 0x10000 {
		if l := mapload.PartyLoad(mapload.PartyMember{Carried: []uint16{uint16(c)}}, f.Table); l > heaviest {
			code, heaviest = uint16(c), l
		}
	}
	capacity := int32(party[1].Hero.Body)*10 + 1
	n := int(16*capacity/heaviest) + 1
	if heaviest == 0 || n > 64 {
		t.Fatalf("heaviest carried item loads %d; %d copies needed", heaviest, n)
	}
	for range n {
		party[1].Carried = append(party[1].Carried, code)
	}
	return party
}

func overloadOrderHero(t *testing.T, f *FrontEnd) sim.Entity {
	t.Helper()
	var found []sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.Humanoid && e.Reaction == 15 {
			found = append(found, e)
		}
	}
	if len(found) != 1 {
		t.Fatalf("Humans with Reaction 15: %d, want one", len(found))
	}
	return found[0]
}

// An overloaded native hero under Haste moves and turns at the claimed
// derive: base less load/capacity floored at six, then the modifier. SAVE
// writes that word, its modifier and mover byte; cold LOAD walks the same
// cadence; the same file read as an original SAV gives the same derive, and
// a load change on that original-loaded hero re-derives it alike.
func TestReleaseOverloadedHastedHeroMovesInTheClaimedOrder(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if err := f.App("overload order").OpenMission(f.MissionOpenerWith(101, overloadOrderParty(t, f))); err != nil {
		t.Fatal(err)
	}
	caster := f.live.mission.ids[0]
	hero := overloadOrderHero(t, f)
	base := hero.Speed
	if hero.SpeedModifier != 0 || base-hero.Load/hero.Capacity >= 6 || hero.SpeedWord() != 6 || hero.RotationSpeed != 6 {
		t.Fatalf("unhasted hero speed %d load %d/%d word %d turn %d; want the floor 6", base, hero.Load, hero.Capacity, hero.SpeedWord(), hero.RotationSpeed)
	}
	f.live.attackOrCast(uint32(caster), uint32(hero.ID), 24, int(hero.X), int(hero.Y), false)
	var mag int32
	for tick := 0; mag == 0; tick++ {
		if tick > 256 {
			t.Fatal("Haste never attached", f.live.world.BookSpellRefusal(caster, hero.ID, 24))
		}
		f.live.tick()
		if e, ok := hasteActive(f); ok && e.Target == hero.ID && e.Kind == sim.EffectSpeed {
			mag = e.Magnitude
		}
	}
	hero = overloadOrderHero(t, f)
	word := 6 + mag
	formerOrder := max(base+mag-hero.Load/hero.Capacity, 6)
	t.Logf("hero base %d, load %d, capacity %d, Haste %d: word %d, former order %d", base, hero.Load, hero.Capacity, mag, word, formerOrder)
	if formerOrder == word {
		t.Fatal("the witness does not separate the two orders")
	}
	if hero.Speed != base+mag || hero.SpeedModifier != mag || hero.SpeedWord() != word || hero.RotationSpeed != word {
		t.Fatalf("hasted hero speed %d modifier %d word %d turn %d; want %d, %d, %d, %d",
			hero.Speed, hero.SpeedModifier, hero.SpeedWord(), hero.RotationSpeed, base+mag, mag, word, word)
	}
	rate, _, _, ok := f.live.world.StepRate(hero.ID, hero.X+1, hero.Y)
	if !ok {
		t.Fatal("no step rate")
	}

	store := SaveStore{Dir: t.TempDir()}
	name, doc := deadPatrolSave(t, f, store)
	saved := hasteSaveRead(t, doc, 15)
	if int32(saved.Speed) != word || int32(saved.Modifier) != mag || int32(saved.Mover) != word {
		t.Fatalf("SAV speed word %d modifier %d mover %d; want %d, %d, %d", saved.Speed, saved.Modifier, saved.Mover, word, mag, word)
	}

	cold := deadPatrolLoad(t, store, name)
	loaded := overloadOrderHero(t, cold)
	if loaded.Speed != hero.Speed || loaded.SpeedModifier != mag || loaded.SpeedWord() != word || loaded.RotationSpeed != word {
		t.Fatalf("cold hero speed %d modifier %d word %d turn %d", loaded.Speed, loaded.SpeedModifier, loaded.SpeedWord(), loaded.RotationSpeed)
	}
	if r, _, _, ok := cold.live.world.StepRate(loaded.ID, loaded.X+1, loaded.Y); !ok || r != rate {
		t.Fatalf("cold step rate %d, live %d", r, rate)
	}
	to := sim.CellPoint{X: hero.X + 6, Y: hero.Y + 3}
	f.live.pending = append(f.live.pending, sim.MoveTo(hero.ID, to))
	cold.live.pending = append(cold.live.pending, sim.MoveTo(loaded.ID, to))
	moved := false
	for tick := range 48 {
		f.live.tick()
		cold.live.tick()
		a, b := overloadOrderHero(t, f), overloadOrderHero(t, cold)
		if a.X != b.X || a.Y != b.Y || a.Facing != b.Facing || a.SpeedWord() != b.SpeedWord() || a.RotationSpeed != b.RotationSpeed {
			t.Fatalf("tick %d: live (%d,%d) facing %d word %d, cold (%d,%d) facing %d word %d",
				tick, a.X, a.Y, a.Facing, a.SpeedWord(), b.X, b.Y, b.Facing, b.SpeedWord())
		}
		moved = moved || a.X != hero.X || a.Y != hero.Y
	}
	if !moved {
		t.Fatal("the hero did not move in 48 ticks")
	}

	// The same file without its engine-state leaf is an original SAV: the hero
	// loads as a source Human holding the saved word.
	raw := mustReadSave(t, store, name)
	odoc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	odoc.State.ValueRecords = slices.DeleteFunc(odoc.State.ValueRecords, func(row sav.CityStateRecordData) bool {
		return row.Path == sav.NativeActionsPath
	})
	// The native writer takes the pack's running weight from instance
	// weights, and an item weighed by the table writes zero (DIV-2761). The
	// original keeps that sum on the container; the control writes it.
	hero = overloadOrderHero(t, f)
	pack, _ := f.live.world.CarriedStacks(hero.ID)
	var carried int32
	for _, st := range pack {
		for _, iw := range f.live.world.ItemWeights() {
			if iw.Code == st.Code {
				carried += iw.Weight * int32(st.Count)
			}
		}
	}
	if carried/2 != hero.Load {
		t.Fatalf("pack weight %d does not give load %d", carried, hero.Load)
	}
	overloadOrderSetAccumulator(t, &odoc, uint32(carried))
	// The fixture caster's restored mana pool has no regeneration period, which
	// an original profile refuses; give it one so the file loads.
	for i := range odoc.Objects {
		r := &odoc.Objects[i]
		if r.Class == "Human" && humanSpeedValue(t, r, "ManaMax") != 0 && humanSpeedValue(t, r, "ManaRegen") == 0 {
			savedObjectSetValue(r, "ManaRegen", 1)
		}
	}
	if law := overloadOrderRecordLaw(t, odoc); int32(law) != word {
		t.Fatalf("the saved record's derive law %d, want %d", law, word)
	}
	if raw, err = sav.EncodeDocumentData(odoc); err != nil {
		t.Fatal(err)
	}
	g := releaseFront(t)
	g.SetDeterministicFrames(true)
	mission, town, err := g.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("original SAV LOAD: town=%t err=%v", town, err)
	}
	if err := g.App("overload order original").OpenMission(mission); err != nil {
		t.Fatal(err)
	}
	src := overloadOrderHero(t, g)
	if law, mover := humanSpeedEntity(src); src.ActorLoad.Source.Class != 2 || int32(law) != word || int32(mover) != word || src.RotationSpeed != word {
		t.Fatalf("original-loaded hero source %d law %d word %d turn %d; want %d", src.ActorLoad.Source.Class, law, mover, src.RotationSpeed, word)
	}
	pack, _ = g.live.world.CarriedStacks(src.ID)
	if len(pack) == 0 {
		t.Fatal("original-loaded hero carries nothing")
	}
	g.live.pending = append(g.live.pending, sim.DropCarried(src.ID, sim.ItemSlot(0), sim.CellPoint{X: src.X, Y: src.Y}))
	g.live.tick()
	after := overloadOrderHero(t, g)
	if after.Load >= src.Load {
		t.Fatalf("the drop left load %d from %d", after.Load, src.Load)
	}
	if law, mover := humanSpeedEntity(after); int32(law) != word || int32(mover) != word || after.RotationSpeed != word {
		t.Fatalf("original-loaded hero after the drop: law %d word %d turn %d; want %d", law, mover, after.RotationSpeed, word)
	}
}

// overloadOrderRecordLaw is SAV-1116's derive over the saved hero record.
func overloadOrderRecordLaw(t *testing.T, doc sav.DocumentData) int16 {
	t.Helper()
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" || humanSpeedValue(t, r, "Reaction") != 15 {
			continue
		}
		v := func(name string) uint32 { return humanSpeedValue(t, r, name) }
		return humanSpeedLaw(uint16(v("Body")), uint16(v("Reaction")), humanSpeedRaw(r, "UD4"), uint16(v("T0E")), int16(v("U8E")), int32(v("Inventory20")))
	}
	t.Fatal("no saved hero record")
	return 0
}

func overloadOrderSetAccumulator(t *testing.T, doc *sav.DocumentData, carried uint32) {
	t.Helper()
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class == "Human" && humanSpeedValue(t, r, "Reaction") == 15 {
			savedObjectSetValue(r, "Inventory20", carried)
			return
		}
	}
	t.Fatal("no saved hero record")
}

func mustReadSave(t *testing.T, store SaveStore, name string) []byte {
	t.Helper()
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
