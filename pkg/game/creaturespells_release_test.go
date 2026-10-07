package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const creatureWitnessMission = 130

// creatureWitnessParty is one mage hero, the party every witness below starts
// the mission with.
func creatureWitnessParty() []mapload.PartyMember {
	return []mapload.PartyMember{{
		ID: "witness-mage", Class: 100, Mage: true, StartingHero: true, PlayerCharacter: true,
		Hero:    data.Hero{Body: 50, Reaction: 50, Mind: 50, Spirit: 50},
		Profile: data.Profile{HealthColumn: true, ManaColumn: true},
	}}
}

// creatureWitnessWorld rebuilds source with the hero moved to (x, y) and given
// a health pool large enough to outlast the fight, so the witness measures the
// creature's decision and not the hero's death.
func creatureWitnessWorld(t *testing.T, source *sim.World, m *alm.Map, table *mapload.Table, hero sim.EntityID, x, y int32, edit ...func(*sim.Entity)) *sim.World {
	t.Helper()
	ents := source.Entities()
	for i := range ents {
		if ents[i].ID == hero {
			ents[i].X, ents[i].Y = x, y
			ents[i].HP, ents[i].MaxHP = 30000, 30000
		}
		for _, f := range edit {
			f(&ents[i])
		}
	}
	out, err := sim.NewStructuredWorld(mapload.Seed, source.Bounds(), sim.ModeCanonical,
		mapload.Planes(m, table), ents, source.Script(), source.Relations(), source.Sacks(),
		source.Stock(), source.Spells(), source.Ghost(), source.Structures())
	if err != nil {
		t.Fatalf("rebuild world with the hero at (%d,%d): %v", x, y, err)
	}
	if err := out.DeclareItemWeights(source.ItemWeights()); err != nil {
		t.Fatal(err)
	}
	return out
}

// creatureFreeCellNear is a passable, unoccupied cell two or three away from
// (x, y), or false.
func creatureFreeCellNear(w *sim.World, planes sim.Terrain, width, height int, x, y int32) (int32, int32, bool) {
	occupied := map[[2]int32]bool{}
	for _, e := range w.Entities() {
		occupied[[2]int32{e.X, e.Y}] = true
	}
	for r := int32(2); r <= 3; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				cx, cy := x+dx, y+dy
				if cx < 1 || cy < 1 || cx >= int32(width)-1 || cy >= int32(height)-1 {
					continue
				}
				if planes.Block[int(cy)*width+int(cx)] == 0 && !occupied[[2]int32{cx, cy}] {
					return cx, cy, true
				}
			}
		}
	}
	return 0, 0, false
}

// creatureCastWithin runs ticks and reports the first tick on which caster
// casts at target, or -1.
func creatureCastWithin(w *sim.World, caster, target sim.EntityID, ticks int) (int, uint16) {
	for tick := 0; tick < ticks; tick++ {
		for _, ev := range sim.StepObserved(w, nil) {
			if ev.Caster == caster && ev.Target == target {
				return tick, ev.Spell
			}
		}
	}
	return -1, 0
}

// TestReleaseCreatureSpellCensus counts the placed creatures that hold class
// spell slots: 602 per install, 598 of them in missions 121..151.
func TestReleaseCreatureSpellCensus(t *testing.T) {
	f := releaseFront(t)
	total, late := 0, 0
	for n := 1; n <= 160; n++ {
		addr, ok := MissionMap(n)
		if !ok {
			continue
		}
		raw, err := f.Archives.Containers.ReadFile(addr)
		if err != nil {
			continue
		}
		m, err := alm.Open(raw)
		if err != nil {
			t.Fatalf("decode %s: %v", addr, err)
		}
		ms, err := StartMissionFrom(m, addr, n, f.Table, mapload.DifficultyNormal, creatureWitnessParty())
		if err != nil {
			t.Fatalf("mission %d: %v", n, err)
		}
		for _, e := range ms.World.Entities() {
			if e.CreatureSpells == ([sim.CreatureSpellSlots]sim.CreatureSpell{}) {
				continue
			}
			total++
			if n >= 121 && n <= 151 {
				late++
			}
			if !e.Book.HasInstances() {
				t.Fatalf("mission %d entity %d holds slots but no class book", n, e.ID)
			}
			if e.MaxMana != 0 {
				t.Fatalf("mission %d entity %d (class %d) holds slots and a mana pool %d: it would be a mage as well", n, e.ID, e.Class, e.MaxMana)
			}
		}
	}
	if total != 602 || late != 598 {
		t.Fatalf("creatures with spell slots: %d total, %d in missions 121..151; want 602 and 598", total, late)
	}
}

// TestReleaseCreatureCastsAtThePartyInMission130 puts the party next to
// placed spell-class creatures in mission 130. Some creature casts its class
// spell at the hero within the window; with the slots cleared the same
// creature never casts.
func TestReleaseCreatureCastsAtThePartyInMission130(t *testing.T) {
	f := releaseFront(t)
	m := releaseMissionMap(t, f, creatureWitnessMission)
	ms, err := StartMissionFrom(m, "witness.alm", creatureWitnessMission, f.Table, mapload.DifficultyNormal, creatureWitnessParty())
	if err != nil {
		t.Fatal(err)
	}
	hero := ms.Start.IDs[0]
	planes := mapload.Planes(m, f.Table)
	const window = 1500
	casts, tried := 0, 0
	for _, c := range ms.World.Entities() {
		if c.CreatureSpells[0].ID == 0 || !c.Alive() || c.Owner == sim.SelfSlot {
			continue
		}
		x, y, ok := creatureFreeCellNear(ms.World, planes, m.Width, m.Height, c.X, c.Y)
		if !ok {
			continue
		}
		if tried == 12 {
			break
		}
		tried++
		w := creatureWitnessWorld(t, ms.World, m, f.Table, hero, x, y)
		at, spell := creatureCastWithin(w, c.ID, hero, window)
		if at < 0 {
			continue
		}
		casts++
		if uint32(spell) != c.CreatureSpells[0].ID {
			t.Errorf("creature %d cast spell %d, its class slot holds %d", c.ID, spell, c.CreatureSpells[0].ID)
		}
		if casts > 1 {
			continue
		}
		// Loss controls on the first caster: no slots, and no book.
		for name, strip := range map[string]func(*sim.Entity){
			"no slots": func(e *sim.Entity) {
				if e.ID == c.ID {
					e.CreatureSpells = [sim.CreatureSpellSlots]sim.CreatureSpell{}
				}
			},
			"no book": func(e *sim.Entity) {
				if e.ID == c.ID {
					e.Book, e.KnownSpells = sim.Spellbook{State: sim.BookAbsent}, 0
				}
			},
		} {
			cw := creatureWitnessWorld(t, ms.World, m, f.Table, hero, x, y, strip)
			if got, _ := creatureCastWithin(cw, c.ID, hero, window); got >= 0 {
				t.Errorf("creature %d cast with %s at tick %d", c.ID, name, got)
			}
		}
	}
	if casts == 0 {
		t.Fatalf("none of %d creatures cast at the party within %d ticks", tried, window)
	}
	t.Logf("%d of %d creatures cast at the party within %d ticks", casts, tried, window)
}

// TestReleaseOriginalSAVLoadsCreatureSpellSlots loads an original SAV from
// mission 131 and checks that the creature slots its order blocks carry reach
// the world, unchanged.
func TestReleaseOriginalSAVLoadsCreatureSpellSlots(t *testing.T) {
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-09-24/game0002-bigsack.sav", "")
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	books, err := file.ActorSpellbooks()
	if err != nil {
		t.Fatal(err)
	}
	savedSlots := map[sav.SavedCreatureSpell]int{}
	for _, b := range books {
		for _, slot := range b.CreatureSpells {
			if slot != (sav.SavedCreatureSpell{}) {
				savedSlots[slot]++
			}
		}
	}
	if len(savedSlots) == 0 {
		t.Fatal("the original SAV carries no creature spell slots")
	}
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal(err)
	}
	app := f.App("creature slots")
	if err = app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	savedBlocks := 0
	for _, b := range books {
		if b.CreatureSpells != ([sav.CreatureSpellSlots]sav.SavedCreatureSpell{}) {
			savedBlocks++
		}
	}
	alive := 0
	for _, e := range f.live.world.Entities() {
		if e.CreatureSpells != ([sim.CreatureSpellSlots]sim.CreatureSpell{}) && e.Alive() {
			alive++
		}
	}
	t.Logf("loaded entities with slots and alive: %d", alive)
	held := 0
	for _, e := range f.live.world.Entities() {
		if e.CreatureSpells != ([sim.CreatureSpellSlots]sim.CreatureSpell{}) && e.Alive() {
			held++
		}
	}
	if held > savedBlocks {
		t.Errorf("the loaded world holds slots for %d living entities, the SAV has %d nonzero blocks: a class value filled a zero block", held, savedBlocks)
	}
	loadedSlots := map[sav.SavedCreatureSpell]int{}
	for _, e := range f.live.world.Entities() {
		for _, slot := range e.CreatureSpells {
			if slot != (sim.CreatureSpell{}) {
				loadedSlots[sav.SavedCreatureSpell{ID: slot.ID, Threshold: slot.Threshold}]++
			}
		}
	}
	for slot, n := range savedSlots {
		if loadedSlots[slot] < n {
			t.Errorf("slot %+v: the SAV holds %d, the loaded world %d", slot, n, loadedSlots[slot])
		}
	}
	t.Logf("SAV slots %v; loaded world slots %v", savedSlots, loadedSlots)
}

// TestReleaseCreatureSpellSaveColdLoadCastsNext starts mission 130 with a
// spell-class creature moved beside the party, saves while that creature's
// cast charges, loads the file cold and runs both sessions to the cast landing.
func TestReleaseCreatureSpellSaveColdLoadCastsNext(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "SlotProbe", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	for i := range party {
		party[i].Hero.Body = 400
	}
	// The party's start cell comes from an ordinary start of the same map; the
	// first spell-class unit is then moved to a free cell beside it before the
	// world is built.
	probe := releaseMissionMap(t, f, creatureWitnessMission)
	started, err := StartMissionFrom(probe, "probe.alm", creatureWitnessMission, f.Table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatal(err)
	}
	planes := mapload.Planes(probe, f.Table)
	var heroAt sim.Entity
	for _, e := range started.World.Entities() {
		if e.ID == started.Start.IDs[0] {
			heroAt = e
		}
	}
	type move struct {
		unit   int
		cx, cy int32
	}
	var moves []move
	taken := map[[2]int32]bool{}
	for _, e := range started.World.Entities() {
		if len(moves) == 8 {
			break
		}
		if e.CreatureSpells[0].ID == 0 || !e.Alive() || e.Owner == sim.SelfSlot {
			continue
		}
		for r := int32(2); r <= 4; r++ {
			placed := false
			for dy := -r; dy <= r && !placed; dy++ {
				for dx := -r; dx <= r && !placed; dx++ {
					x, y := heroAt.X+dx, heroAt.Y+dy
					if x < 1 || y < 1 || taken[[2]int32{x, y}] || planes.Block[int(y)*probe.Width+int(x)] != 0 {
						continue
					}
					occupied := false
					for _, o := range started.World.Entities() {
						occupied = occupied || (o.X == x && o.Y == y)
					}
					if !occupied {
						taken[[2]int32{x, y}] = true
						moves = append(moves, move{int(e.ID), x, y})
						placed = true
					}
				}
			}
			if placed {
				break
			}
		}
	}
	if len(moves) == 0 {
		t.Fatal("no spell-class creature or no free cell beside the party")
	}
	app := f.App("creature slots SAVE")
	t.Cleanup(app.StopAudio)
	opener := f.missionOpener(creatureWitnessMission, party, nil, nil, func(m *alm.Map) (*originalFog, error) {
		for _, mv := range moves {
			m.Units[mv.unit].X, m.Units[mv.unit].Y = uint32(mv.cx*256+128), uint32(mv.cy*256+128)
		}
		return nil, nil
	})
	if err := app.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	live := f.live
	hero := live.mission.ids[0]
	var caster sim.EntityID
	var charging sim.BookContinuation
	for tick := 0; tick < 1500 && caster == 0; tick++ {
		live.tick()
		for _, c := range live.world.Actions().Books {
			if e, ok := live.entity(c.Caster); ok && c.Target == hero && c.Phase == 1 && c.Remaining >= 2 && e.CreatureSpells[0].ID != 0 {
				caster, charging = c.Caster, c
			}
		}
	}
	if caster == 0 {
		t.Fatal("no creature began a cast at the party within the window")
	}
	slots := map[sim.EntityID][sim.CreatureSpellSlots]sim.CreatureSpell{}
	want := 0
	for _, e := range live.world.Entities() {
		slots[e.ID] = e.CreatureSpells
		if e.CreatureSpells != ([sim.CreatureSpellSlots]sim.CreatureSpell{}) {
			want++
		}
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := f.ExportCurrentWorldSave(snapshot, "creature cast")
	if err != nil {
		t.Fatal(err)
	}
	if ce, ok := live.entity(caster); ok && charging.Retained {
		if order := creatureWireOrder(t, saved, ce.MapUnitID); order[0x60] != 1 {
			t.Fatalf("the retained creature cast saved with retention flag %d, want 1", order[0x60])
		}
	}
	savedDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(savedDir, "creature cast.sav"), saved, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	books, err := file.ActorSpellbooks()
	if err != nil {
		t.Fatal(err)
	}
	inFile := 0
	for _, b := range books {
		if b.CreatureSpells != ([sim.CreatureSpellSlots]sav.SavedCreatureSpell{}) {
			inFile++
		}
	}
	if inFile != want || want == 0 {
		t.Fatalf("SAVE wrote slots for %d actors, the world holds %d", inFile, want)
	}
	g, _ := castOrderSession(t, savedDir)
	for _, e := range g.live.world.Entities() {
		if e.CreatureSpells != slots[e.ID] {
			t.Fatalf("cold LOAD changed entity %d slots %v to %v", e.ID, slots[e.ID], e.CreatureSpells)
		}
	}
	if cold, ok := castOrderBook(g.live.world, caster); !ok || cold != charging {
		t.Fatalf("cold LOAD cast %+v, saved %+v", cold, charging)
	}
	castOrderLockstep(t, f, g, 256, func(h *FrontEnd, events []sim.CastEvent) bool {
		for _, ev := range events {
			if ev.Caster == caster && ev.Target == hero {
				return true
			}
		}
		return false
	})
}
