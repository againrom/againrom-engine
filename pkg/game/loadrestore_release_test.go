package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func loadRestoreStatus(w *sim.World) map[uint16]string {
	out := map[uint16]string{}
	for _, e := range w.Entities() {
		if e.MapUnitID == 0 {
			continue
		}
		state := "alive"
		if !e.Alive() {
			state = fmt.Sprintf("body%d", e.Decay)
		}
		if e.OffMap {
			state = "hidden-" + state
		}
		out[e.MapUnitID] = state
	}
	for _, d := range w.CurrentTerminalActors() {
		out[d.MapUnitID] = "terminal"
	}
	return out
}

func loadRestoreStructures(w *sim.World) map[sim.StructureID]uint16 {
	out := map[sim.StructureID]uint16{}
	for _, s := range w.Structures() {
		out[s.ID] = s.Field42
	}
	return out
}

func loadRestoreSave(t *testing.T, f *FrontEnd) (SaveStore, string, sav.DocumentData) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	name, doc := deadPatrolSave(t, f, store)
	return store, name, doc
}

func loadRestoreMage() []mapload.PartyMember {
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	hero.Skill[1] = 100
	return []mapload.PartyMember{{
		ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero,
		KnownSpells: 1 << 2,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 20, Y: 20}, HP: 100, MaxHP: 100,
			Mana: 10000, MaxMana: 10000, HealthRegenPeriod: 100, ManaRegenPeriod: 50},
	}}
}

func buildingRecord(t *testing.T, doc *sav.DocumentData, col, row int32) *sav.DocumentRecordData {
	t.Helper()
	var found *sav.DocumentRecordData
	for _, index := range doc.World.Buildings {
		record := &doc.Objects[index-1]
		for _, raw := range record.Raw {
			if raw.Name == "Block12" && len(raw.Bytes) == 12 && int32(raw.Bytes[0]) == col && int32(raw.Bytes[1]) == row {
				if found != nil && found != record {
					t.Fatalf("document holds two Building records at (%d,%d)", col, row)
				}
				found = record
			}
		}
	}
	if found == nil {
		t.Fatalf("document holds no Building record at (%d,%d)", col, row)
	}
	return found
}

func buildingHealth(t *testing.T, doc sav.DocumentData, col, row int32) uint32 {
	t.Helper()
	for _, v := range buildingRecord(t, &doc, col, row).Values {
		if v.Name == "B42" {
			return v.Value
		}
	}
	t.Fatal("Building record has no +0x42 word")
	return 0
}

func TestReleaseDestroyedStructureStaysDestroyedAcrossSaveLoadChain(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("destroyed structure")
	if err := app.OpenMission(f.MissionOpenerWith(20, loadRestoreMage())); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		f.live.tick()
	}
	store0, name0, doc0 := loadRestoreSave(t, f)
	f = deadPatrolLoad(t, store0, name0)
	mw := f.live
	w := mw.world
	var target sim.Structure
	for _, s := range w.Structures() {
		if s.MaxHealth > 0 && s.Kind == 2 {
			target = s
			break
		}
	}
	if target.MaxHealth == 0 {
		t.Fatal("mission 20 has no destructible Orc House")
	}
	if got := buildingHealth(t, doc0, target.Col, target.Row); got != uint32(target.MaxHealth) {
		t.Fatalf("source SAV holds the building at %d, want %d", got, target.MaxHealth)
	}
	caster := mw.mission.ids[0]
	if err := w.HeadlessPlace(caster, target.Col-3, target.Row+1); err != nil {
		t.Fatal(err)
	}
	health := func(w *sim.World) uint16 {
		for _, s := range w.Structures() {
			if s.ID == target.ID {
				return s.Field42
			}
		}
		t.Fatal("structure left the world")
		return 0
	}
	for n := 0; n < 40 && int16(health(w)) > 0; n++ {
		mw.attackOrCast(uint32(caster), 0, 2, int(target.Col), int(target.Row), true)
		for i := 0; i < 80; i++ {
			mw.tick()
		}
	}
	if got := health(w); got != 0 {
		t.Fatalf("Fire Ball left the building at %d, want destroyed", got)
	}
	want := loadRestoreStructures(w)

	store1, name1, doc1 := loadRestoreSave(t, f)
	if got := buildingHealth(t, doc1, target.Col, target.Row); got != 0 {
		t.Fatalf("SAV after the destruction holds +0x42 = %d, want 0", got)
	}
	g := deadPatrolLoad(t, store1, name1)
	if got := loadRestoreStructures(g.live.world); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("cold LOAD structure health differs from the saved world")
	}
	entries, ruined := g.live.view.StructureRuinFrames(uint32(target.ID))
	if entries == 0 || ruined != entries {
		t.Fatalf("restored draw = %d entries, %d ruin", entries, ruined)
	}

	for i := 0; i < 20; i++ {
		g.live.tick()
	}
	if got := health(g.live.world); got != 0 {
		t.Fatalf("20 ticks after LOAD the building has health %d", got)
	}
	store2, name2, doc2 := loadRestoreSave(t, g)
	if got := buildingHealth(t, doc2, target.Col, target.Row); got != 0 {
		t.Fatalf("second SAV holds +0x42 = %d, want 0", got)
	}
	h := deadPatrolLoad(t, store2, name2)
	if got := health(h.live.world); got != 0 {
		t.Fatalf("second cold LOAD restored the building at %d", got)
	}

	t.Run("source word restored", func(t *testing.T) {
		control, err := sav.CloneDocumentData(doc1)
		if err != nil {
			t.Fatal(err)
		}
		record := buildingRecord(t, &control, target.Col, target.Row)
		for i := range record.Values {
			if record.Values[i].Name == "B42" {
				record.Values[i].Value = uint32(target.MaxHealth)
			}
		}
		raw, err := sav.EncodeDocumentData(control)
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "game0000.sav"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		c := deadPatrolLoad(t, SaveStore{Dir: dir}, "game0000.sav")
		if got := health(c.live.world); got != target.MaxHealth {
			t.Fatalf("control LOAD restored health %d, want %d", got, target.MaxHealth)
		}
	})
}

func TestReleaseDeadHiddenActorStaysDeadAcrossSaveLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if err := f.App("dead hidden actor").OpenMission(f.MissionOpener(120)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	w := mw.world
	for w.Tick() < 20 {
		mw.tick()
	}
	var victims []uint16
	for _, e := range w.Entities() {
		if e.OffMap && e.Alive() && e.Class == 69 {
			victims = append(victims, e.MapUnitID)
			if err := w.HeadlessDamage(e.ID, e.HP+3); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(victims) == 0 {
		t.Fatal("mission 120 holds no hidden spirit at tick 20")
	}
	for i := 0; i < 5; i++ {
		mw.tick()
	}
	want := loadRestoreStatus(w)
	for _, mu := range victims {
		if want[mu] == "alive" || want[mu] == "hidden-alive" {
			t.Fatalf("map unit %d is still %s", mu, want[mu])
		}
	}
	store, name, _ := loadRestoreSave(t, f)
	g := deadPatrolLoad(t, store, name)
	got := loadRestoreStatus(g.live.world)
	for mu, state := range want {
		if got[mu] != state {
			t.Fatalf("map unit %d was %s and LOADs as %s", mu, state, got[mu])
		}
	}
	for i := 0; i < 30; i++ {
		g.live.tick()
	}
	for _, mu := range victims {
		if s := loadRestoreStatus(g.live.world)[mu]; s == "alive" || s == "hidden-alive" {
			t.Fatalf("map unit %d is %s after LOAD and 30 ticks", mu, s)
		}
	}
}

func TestReleaseDeadActorWithAttachedEffectStaysDeadAcrossSaveLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if err := f.App("dead effect target").OpenMission(f.MissionOpener(90)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	w := mw.world
	for w.Tick() < 20 {
		mw.tick()
	}
	rows := w.Actions().EffectCasters
	if len(rows) == 0 {
		t.Fatal("mission 90 holds no attached effect at tick 20")
	}
	var mu uint16
	for _, r := range rows {
		e, ok := mw.entity(r.Target)
		if !ok {
			t.Fatal("effect target absent")
		}
		mu = e.MapUnitID
		if err := w.HeadlessDamage(e.ID, e.HP+3); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 60; i++ {
		mw.tick()
	}
	want := loadRestoreStatus(w)
	if want[mu] != "terminal" {
		t.Fatalf("map unit %d is %s, want a removed body", mu, want[mu])
	}
	store, name, _ := loadRestoreSave(t, f)
	g := deadPatrolLoad(t, store, name)
	got := loadRestoreStatus(g.live.world)
	for u, state := range want {
		if got[u] != state {
			t.Fatalf("map unit %d was %s and LOADs as %s", u, state, got[u])
		}
	}
}

func TestReleaseKilledActorsStayDeadThroughRepeatedSaveLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if err := f.App("repeated load").OpenMission(f.MissionOpener(120)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		f.live.tick()
	}
	for gen := 0; gen < 3; gen++ {
		mw := f.live
		w := mw.world
		killed := 0
		for k, e := range w.Entities() {
			if e.Owner == 1 || !e.Alive() || e.OffMap || e.MaxHP <= 0 || (k+gen)%4 != 0 {
				continue
			}
			if err := w.HeadlessDamage(e.ID, e.HP+3); err == nil {
				killed++
			}
		}
		if killed == 0 {
			t.Fatalf("generation %d killed nothing", gen)
		}
		for _, wait := range []int{2, 12, 40} {
			for i := 0; i < wait; i++ {
				mw.tick()
			}
			want := loadRestoreStatus(w)
			store, name, _ := loadRestoreSave(t, f)
			g := deadPatrolLoad(t, store, name)
			got := loadRestoreStatus(g.live.world)
			for mu, state := range want {
				if got[mu] != state {
					t.Fatalf("generation %d, %d ticks after the kills: map unit %d was %s and LOADs as %s", gen, wait, mu, state, got[mu])
				}
			}
		}
		store, name, _ := loadRestoreSave(t, f)
		f = deadPatrolLoad(t, store, name)
	}
}
