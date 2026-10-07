package game

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestReleaseMission101FireBallDestroysAndDrawsTheShippedSwitch drives the
// cheapest shipped witness through the mission opener, MapAttack seam, spell
// step, save form and viewer ruin selector. The release gate runs it on EN and
// RU installs.
func TestReleaseMission101FireBallDestroysAndDrawsTheShippedSwitch(t *testing.T) {
	f := releaseFront(t)
	addr, _ := MissionMap(101)
	raw, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	const target = 11
	if len(m.Objects) <= target {
		t.Fatalf("mission 101 has %d structures, want index %d", len(m.Objects), target)
	}
	o := m.Objects[target]
	if col, row := int32(o.X>>8), int32(o.Y>>8); byte(o.Kind) != 29 || col != 54 || row != 54 {
		t.Fatalf("mission 101 structure 11 = class %d at (%d,%d), want class 29 at (54,54)",
			byte(o.Kind), col, row)
	}
	class := f.Structures.Classes[29]
	if class == nil || class.Indestructible || class.GridCells() <= 0 || len(class.Frames) < 2*class.GridCells() {
		t.Fatalf("installed class 29 has no distinct ruin block: %#v", class)
	}

	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	hero.Skill[1] = 100
	party := []mapload.PartyMember{{
		ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero,
		KnownSpells: 1 << 2,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 53, Y: 54}, HP: 100, MaxHP: 100,
			Mana: 1000, MaxMana: 1000, HealthRegenPeriod: 100, ManaRegenPeriod: 50},
	}}
	app := f.App("1052-destructible-release")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(101, party)); err != nil {
		t.Fatalf("open mission 101: %v", err)
	}
	live := f.live
	if live == nil || len(live.mission.ids) != 1 {
		t.Fatal("mission 101 opened without the test mage")
	}
	caster := live.mission.ids[0]
	structures := live.world.Structures()
	if len(structures) <= target || structures[target].Field42 != 1 ||
		structures[target].MaxHealth != 1 || structures[target].Col != 54 || structures[target].Row != 54 ||
		structures[target].Width != 1 || structures[target].Height != 1 || structures[target].Attach&1 == 0 {
		t.Fatalf("production structure 11 = %+v, want live 1/1 single-cell target at (54,54)", structures[target])
	}
	if entries, ruined := live.view.StructureRuinFrames(target); entries == 0 || ruined != 0 {
		t.Fatalf("intact production draw = %d entries, %d ruin; want visible intact entries", entries, ruined)
	}
	entityMana := func(id sim.EntityID) int32 {
		t.Helper()
		for _, entity := range live.world.Entities() {
			if entity.ID == id {
				return entity.Mana
			}
		}
		t.Fatalf("entity %d is absent", id)
		return 0
	}
	fireBall, ok := live.world.Spell(2)
	if !ok {
		t.Fatal("production world has no Fire Ball rule")
	}
	manaBefore := entityMana(caster)

	// This is the exact callback installed as ui.MapAttack by MissionOpener.
	live.attackOrCast(uint32(caster), 0, 2, 54, 54, true)
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindCastAt || live.pending[0].Spell != 2 {
		t.Fatalf("production input queued %+v, want one Fire Ball cell cast", live.pending)
	}
	live.tick()
	if spent := manaBefore - entityMana(caster); spent != fireBall.ManaCost {
		t.Fatalf("Fire Ball admission spent%d mana, want row cost%d", spent, fireBall.ManaCost)
	}
	lastMana := entityMana(caster)
	for tick := 0; tick < 512 && live.world.Structures()[target].Field42 != 0; tick++ {
		live.tick()
		if now := entityMana(caster); now < lastMana {
			t.Fatalf("mana charged twice before impact: %d -> %d", lastMana, now)
		} else {
			lastMana = now
		}
	}
	if got := live.world.Structures()[target].Field42; got != 0 {
		t.Fatalf("Fire Ball left structure 11 at health %d, want destroyed", got)
	}
	manaAfterImpact := entityMana(caster)
	// The explicit command is one application. Let its recovery drain without
	// another input and prove neither a retained cast nor a second payment
	// appears before arming the separate migration fixture below.
	lastMana = manaAfterImpact
	for tick := 0; tick < 512; tick++ {
		var casterState sim.Entity
		for _, entity := range live.world.Entities() {
			if entity.ID == caster {
				casterState = entity
				break
			}
		}
		if casterState.CastWait == 0 {
			break
		}
		live.tick()
		if now := entityMana(caster); now < lastMana {
			t.Fatalf("mana fell again during one command's recovery: %d -> %d", lastMana, now)
		} else {
			lastMana = now
		}
	}
	if spell, _, casting := live.world.CastingSpell(caster); casting {
		t.Fatalf("one Fire Ball command retained spell %d after recovery", spell)
	}
	for tick := 0; tick < 64 && len(live.world.SavedProjectiles().Items) != 0; tick++ {
		live.tick()
	}
	entries, ruined := live.view.StructureRuinFrames(target)
	if entries == 0 || ruined != entries {
		t.Fatalf("destroyed production draw = %d entries, %d ruin; want every visible entry ruined", entries, ruined)
	}

	// The separate historical migration fixture predates delivery columns and
	// paid admission. Start it from the independently peeled idle predecessor;
	// its next command then creates the unpaid lifecycle those forms supported.
	idle, err := live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var predecessor sim.World
	if err := predecessor.UnmarshalBinary(idle); err != nil {
		t.Fatal(err)
	}
	if err := predecessor.ReplaceCurrentObjects(nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	// The blast's burst record and the counter it advanced are state a
	// predecessor form never held.
	predecessor.SetSavedProjectiles(sim.SavedProjectiles{})
	idle, err = predecessor.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	legacy := append(beforeSpellDelivery1183(t, idle), 0, 0, 0, 0)
	legacy[0] = 95
	var legacyWorld sim.World
	if err := legacyWorld.UnmarshalBinary(legacy); err != nil {
		t.Fatal(err)
	}
	*live.world = legacyWorld
	mapload.BindSourceDerive(live.world)
	// A second explicit command is kept in wind-up so migration still carries
	// a genuine production book record, without inventing paid legacy state.
	live.attackOrCast(uint32(caster), 0, 2, 54, 54, true)
	live.tick()
	if spell, _, casting := live.world.CastingSpell(caster); !casting || spell != 2 {
		t.Fatalf("migration fixture did not arm one Fire Ball: spell=%d casting=%v", spell, casting)
	}
	form, err := live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var resumed sim.World
	if err := resumed.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	got := resumed.Structures()[target]
	if len(live.world.ScorchedCells()) == 0 || !slices.Equal(resumed.ScorchedCells(), live.world.ScorchedCells()) {
		t.Fatal("current SAVE lost the impact's fire history")
	}
	if got.Field42 != 0 || got.MaxHealth != 1 || got.Col != 54 || got.Row != 54 || got.Width != 1 || got.Height != 1 {
		t.Fatalf("saved destroyed structure = %+v", got)
	}

	// Save on the impact tick, while the production book-cast lifecycle record
	// is still present. This is the whole persistence boundary the single
	// adversarial return found missing: Snapshot, disk envelope, prior-form
	// widening, mission repair and the committed live viewer.
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatalf("snapshot destroyed structure: %v", err)
	}
	if !bytes.Equal(snapshot.World, form) {
		t.Fatal("Snapshot world differs from the live canonical form")
	}
	bookLifecycle(t, snapshot.World, caster)
	wantHash := live.world.Hash()

	forms := []struct {
		name  string
		form  []byte
		exact bool
	}{
		{"current form", snapshot.World, true},
	}
	for _, tc := range forms {
		t.Run(tc.name, func(t *testing.T) {
			bookLifecycle(t, tc.form, caster)
			disk, err := EncodeSave(withWorld(snapshot, tc.form), label)
			if err != nil {
				t.Fatalf("EncodeSave: %v", err)
			}
			decoded, _, err := DecodeSave(disk)
			if err != nil {
				t.Fatalf("DecodeSave: %v", err)
			}
			restored := releaseFront(t)
			opener, town, err := restored.Restore(decoded)
			if err != nil || town || opener == nil {
				t.Fatalf("Restore = opener %v town %v err %v", opener != nil, town, err)
			}
			if _, _, _, _, _, _, _, _, _, _, err := opener(); err != nil {
				t.Fatalf("open restored mission: %v", err)
			}
			if restored.live == nil {
				t.Fatal("restore committed no live mission")
			}
			if tc.form[0] < 88 && len(restored.live.world.ScorchedCells()) != 0 {
				t.Fatal("historical LOAD invented fire history")
			}
			restoredForm, err := restored.live.world.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			gotHash, expectedHash := restored.live.world.Hash(), wantHash
			gotForm, expectedForm := restoredForm, snapshot.World
			if tc.form[0] < 81 {
				// These historical forms carried the transit pair, but never
				// accepted-stride provenance. Require actual absence on LOAD,
				// then compare every older state byte, including casting/turns.
				for _, e := range restored.live.world.Entities() {
					if e.Stride != (sim.NativeStride{}) {
						t.Fatal("legacy LOAD invented accepted-stride provenance", e.ID)
					}
				}
				gotHash = beforeStrideState1115Hash(t, restored.live.world)
				expectedHash = beforeStrideState1115Hash(t, live.world)
				gotForm = beforeStrideStateForm1115(t, restoredForm)
				expectedForm = beforeStrideStateForm1115(t, snapshot.World)
			}
			if tc.exact && gotHash != expectedHash {
				t.Fatalf("restored hash = %016x, want impact-tick hash %016x", gotHash, expectedHash)
			}
			if tc.exact && !bytes.Equal(gotForm, expectedForm) {
				t.Fatal("restored canonical form differs from the impact-tick world")
			}
			bookLifecycle(t, restoredForm, caster)
			state := restored.live.world.Structures()[target]
			if state.Field42 != 0 || state.MaxHealth != 1 || state.Col != 54 || state.Row != 54 ||
				state.Width != 1 || state.Height != 1 || state.Attach&1 == 0 {
				t.Fatalf("restored destroyed structure = %+v", state)
			}
			entries, ruined := restored.live.view.StructureRuinFrames(target)
			if entries == 0 || ruined != entries {
				t.Fatalf("restored draw = %d entries, %d ruin; want every entry ruined", entries, ruined)
			}
		})
	}
}

func withWorld(snapshot Snapshot, world []byte) Snapshot {
	snapshot.World = append([]byte(nil), world...)
	return snapshot
}

// bookLifecycle locates the production Fire Ball cell cast without using
// the casting decoder under test. Its nonzero lifecycle bytes prove every
// prior-form fixture carries a complete 24-byte book record.
func bookLifecycle(t *testing.T, form []byte, caster sim.EntityID) [4]byte {
	t.Helper()
	var tail [4]byte
	matches := 0
	for at := 0; at+25 <= len(form); at++ {
		if form[at] != 3 || binary.LittleEndian.Uint32(form[at+1:at+5]) != uint32(caster) ||
			binary.LittleEndian.Uint32(form[at+5:at+9]) != 0 ||
			binary.LittleEndian.Uint16(form[at+9:at+11]) != 2 ||
			int32(binary.LittleEndian.Uint32(form[at+11:at+15])) != 54 ||
			int32(binary.LittleEndian.Uint32(form[at+15:at+19])) != 54 || form[at+20] != 1 {
			continue
		}
		matches++
		copy(tail[:], form[at+21:at+25])
	}
	if matches != 1 {
		t.Fatalf("Fire Ball book lifecycle matches = %d, want exactly one", matches)
	}
	if tail == [4]byte{} {
		t.Fatal("Fire Ball record has no lifecycle state; fixture would not kill a duplicate-tail mutation")
	}
	return tail
}
