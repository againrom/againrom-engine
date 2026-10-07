package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// humanSpeedLaw is SAV-1116's six-step speed over one Human's own fields.
func humanSpeedLaw(body, reaction uint16, mod []byte, typeID uint16, own int16, acc int32) int16 {
	r := min(int32(int16(reaction)), int32(int8(mod[1]))+50)
	speed := r
	if r >= 12 {
		speed = r/5 + 12
	}
	if typeID == 0x13 || typeID == 0x15 {
		speed += 10
	}
	load := int32(own) + acc/2
	if acc >= 0xfa00 {
		load = 0x7d00
	}
	capacity := min(int32(int16(body)), int32(int8(mod[0]))+50)*10 + 1
	if load >= capacity {
		speed = max(speed-load/capacity, 6)
	}
	return int16(uint16(speed) + binary.LittleEndian.Uint16(mod[4:]))
}

func humanSpeedRaw(r *sav.DocumentRecordData, name string) []byte {
	for _, x := range r.Raw {
		if x.Name == name {
			return x.Bytes
		}
	}
	return nil
}

func humanSpeedValue(t *testing.T, r *sav.DocumentRecordData, name string) uint32 {
	t.Helper()
	v, err := savedStructureValue(r, name)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// humanSpeedRecords returns each Human's law, saved word and mover byte.
func humanSpeedRecords(t *testing.T, raw []byte) (law, word []int16, mover []byte) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" {
			continue
		}
		v := func(name string) uint32 { return humanSpeedValue(t, r, name) }
		law = append(law, humanSpeedLaw(uint16(v("Body")), uint16(v("Reaction")), humanSpeedRaw(r, "UD4"), uint16(v("T0E")), int16(v("U8E")), int32(v("Inventory20"))))
		word, mover = append(word, int16(v("Speed"))), append(mover, humanSpeedRaw(r, "U154")[10])
	}
	return law, word, mover
}

func humanSpeedEntity(e sim.Entity) (law, mover int16) {
	s := e.SourceNow()
	law = humanSpeedLaw(s.Stats[0], s.Stats[1], s.Modifier[:], s.TypeID, e.ActorLoad.OwnWeight, e.ActorLoad.Accumulator)
	mover, _ = e.RetainedHumanSpeed()
	return law, mover
}

// An original-written town holding overloaded Humans with a speed modifier:
// LOAD keeps each file speed, a load change across capacity re-derives it,
// and SAVE writes the live word and mover byte.
func TestReleaseOverloadedHumanSpeedAfterOriginalTownLoad(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(os.Getenv("AGAINROM_ASSETS"), "game0002.sav"))
	if err != nil {
		t.Skip("no AGAINROM_ASSETS: game0002.sav speed witness needs a lawful install")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mission.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	_, _, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	open, town, err := load("mission.sav")
	if err != nil || town {
		t.Fatalf("LOAD game0002.sav: town=%v err=%v", town, err)
	}
	if err := f.App("speed 10").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatal(err)
	}
	if err := f.App("speed 20").OpenMission(f.MissionOpenerWith(20, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.LiveCompleteCampaign(); err != nil || !f.Town.Open() {
		t.Fatalf("finish mission 20: town=%v err=%v", f.Town.Open(), err)
	}

	// The original writes no engine-state leaf. Each Human carries a +4
	// speed modifier and a load equal to its capacity, with its speed word
	// and mover byte as the law gives them.
	doc, err := sav.DecodeDocumentData(currentTownSave(t, f))
	if err != nil {
		t.Fatal(err)
	}
	var kept []sav.CityStateRecordData
	for _, r := range doc.State.ValueRecords {
		if r.Path != sav.NativeActionsPath {
			kept = append(kept, r)
		}
	}
	doc.State.ValueRecords = kept
	humans := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" {
			continue
		}
		humans++
		v := func(name string) uint32 { return humanSpeedValue(t, r, name) }
		mod := humanSpeedRaw(r, "UD4")
		binary.LittleEndian.PutUint16(mod[4:], 4)
		acc := int32(v("Inventory20"))
		own := int16(int32(int16(v("Capacity"))) - acc/2)
		speed := humanSpeedLaw(uint16(v("Body")), uint16(v("Reaction")), mod, uint16(v("T0E")), own, acc)
		savedObjectSetValue(r, "U8E", uint32(uint16(own)))
		savedObjectSetValue(r, "U90", v("Capacity"))
		savedObjectSetValue(r, "Speed", uint32(uint16(speed)))
		humanSpeedRaw(r, "U154")[10] = byte(speed)
	}
	source, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	law, word, _ := humanSpeedRecords(t, source)
	if humans == 0 || len(law) != humans {
		t.Fatalf("town holds %d Humans", humans)
	}

	g := currentTownReload(t, source)
	party := g.NextParty()
	if len(party) != humans {
		t.Fatalf("town party %d, Humans %d", len(party), humans)
	}
	for i, m := range party {
		l := m.Carry.LiveLoad
		if l == nil || l.Speed != int32(law[i]) || l.DisplaySpeed() != int32(law[i]) || word[i] != law[i] {
			t.Fatalf("town member %d live speed %d, law %d, word %d", i, l.Speed, law[i], word[i])
		}
	}

	// Loss control: the engine's own town file keeps each speed.
	written := currentTownSave(t, g)
	if l, w, m := humanSpeedRecords(t, written); !equalSpeeds(l, law) || !equalSpeeds(w, law) || !moverMatches(m, law) {
		t.Fatalf("engine town SAVE law %v word %v mover %v, want %v", l, w, m, law)
	}
	back := currentTownReload(t, written)
	for i, m := range back.NextParty() {
		if l := m.Carry.LiveLoad; l == nil || l.Speed != int32(law[i]) || l.DisplaySpeed() != int32(law[i]) {
			t.Fatalf("engine town reload member %d live speed %d, want %d", i, l.Speed, law[i])
		}
	}
	if _, w, m := humanSpeedRecords(t, currentTownSave(t, back)); !equalSpeeds(w, law) || !moverMatches(m, law) {
		t.Fatalf("engine town second SAVE word %v mover %v, want %v", w, m, law)
	}

	// The mission route: every Human spawns at its file speed, and one drop
	// across capacity re-derives it.
	app := g.App("speed mission")
	app.Layout(1024, 768)
	if err := app.OpenMission(g.MissionOpener(g.Town.Chapter())); err != nil {
		t.Fatal(err)
	}
	g.SetDeterministicFrames(true)
	if len(g.live.mission.ids) != humans {
		t.Fatalf("mission holds %d heroes, the town %d", len(g.live.mission.ids), humans)
	}
	for i, id := range g.live.mission.ids {
		e, _ := g.live.entity(id)
		if want, mover := humanSpeedEntity(e); want != law[i] || mover != law[i] || e.Speed != int32(law[i]) {
			t.Fatalf("mission hero %d speed %d mover %d law %d, want %d", i, e.Speed, mover, want, law[i])
		}
	}
	hero := g.live.mission.ids[0]
	e, _ := g.live.entity(hero)
	pack, _ := g.live.world.CarriedStacks(hero)
	drop := sim.DropWorn(hero, sim.EquipSlot(1), sim.CellPoint{X: e.X, Y: e.Y})
	for i, s := range pack {
		if s.Weight >= 2 {
			drop = sim.DropCarried(hero, sim.ItemSlot(i), sim.CellPoint{X: e.X, Y: e.Y})
			break
		}
	}
	g.live.pending = append(g.live.pending, drop)
	g.live.tick()
	after, _ := g.live.entity(hero)
	if after.Load >= after.Capacity {
		t.Fatalf("the drop left load %d at capacity %d", after.Load, after.Capacity)
	}
	want, mover := humanSpeedEntity(after)
	if want != law[0]+1 || mover != want || after.Speed != int32(want) {
		t.Fatalf("after the drop speed %d mover %d, law %d (loaded %d)", after.Speed, mover, want, law[0])
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := g.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	l, w, m := humanSpeedRecords(t, saved)
	if !equalSpeeds(l, w) || !moverMatches(m, w) || !containsSpeed(w, want) {
		t.Fatalf("mission SAVE law %v word %v mover %v, want the dropped hero at %d", l, w, m, want)
	}
}

func equalSpeeds(a, b []int16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func moverMatches(m []byte, w []int16) bool {
	if len(m) != len(w) {
		return false
	}
	for i := range m {
		if m[i] != byte(w[i]) {
			return false
		}
	}
	return true
}

func containsSpeed(w []int16, v int16) bool {
	for _, x := range w {
		if x == v {
			return true
		}
	}
	return false
}
