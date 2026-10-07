package game

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// widePoolFixtures are current pool tuples (HP, MaxHP, Mana, MaxMana) beyond
// the ordinary signed and unsigned 16-bit words, including the 16-bit edges.
var widePoolFixtures = [][4]int32{
	{32767, 65535, 65535, 65535},
	{32768, 65536, 65536, 65536},
	{65536, math.MaxInt32, math.MaxInt32, math.MaxInt32},
	{math.MaxInt32, math.MaxInt32, -1, 65535},
	{65535, 65536, -65537, 65536},
}

func widePoolWidth(field int, value int32) *sim.ActorNumericResidue {
	wire := uint16(value)
	decoded := int32(wire)
	if field == 0 {
		decoded = int32(int16(wire))
	}
	lift := int64(value) - int64(decoded)
	if lift == 0 {
		return nil
	}
	return &sim.ActorNumericResidue{Field: uint8(field), Wire: uint32(wire), Lift: lift}
}

// applyWidePools gives the first living actors the fixture tuples through the
// typed current pool and width APIs and returns their IDs.
func applyWidePools(t *testing.T, f *FrontEnd) []sim.EntityID {
	t.Helper()
	w := f.live.world
	before := w.Entities()
	values := map[sim.EntityID]sim.ActorValues{}
	for _, e := range before {
		v := e.Values()
		v.Widths = nil
		group := e.Group
		v.NativeGroup = &group
		values[e.ID] = v
	}
	var pools []sim.OriginalActorPools
	var ids []sim.EntityID
	for _, e := range before {
		if len(ids) == len(widePoolFixtures) {
			break
		}
		if e.Decay != 0 || e.HP <= 0 {
			continue
		}
		want := widePoolFixtures[len(ids)]
		v := values[e.ID]
		for field, value := range want {
			if x := widePoolWidth(field, value); x != nil {
				v.Widths = append(v.Widths, *x)
			}
		}
		values[e.ID] = v
		pools = append(pools, sim.OriginalActorPools{ID: e.ID, HP: int32(int16(uint16(want[0]))), MaxHP: int32(uint16(want[1])), Mana: int32(uint16(want[2])), MaxMana: int32(uint16(want[3]))})
		ids = append(ids, e.ID)
	}
	if len(ids) != len(widePoolFixtures) {
		t.Fatalf("mission has %d living actors for %d fixtures", len(ids), len(widePoolFixtures))
	}
	if err := w.ImportOriginalActorPools(pools, ids...); err != nil {
		t.Fatal("pool admission:", err)
	}
	if err := w.RestoreCurrentContinuation(nil, values, w.Actions(), nil); err != nil {
		t.Fatal("width admission:", err)
	}
	after := w.Entities()
	for i := range before {
		for n, id := range ids {
			if before[i].ID == id {
				p := widePoolFixtures[n]
				before[i].HP, before[i].MaxHP, before[i].Mana, before[i].MaxMana = p[0], p[1], p[2], p[3]
			}
		}
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("fixture changed fields beyond the declared pools")
	}
	return ids
}

func TestReleaseGeneratedWidePoolsSAV(t *testing.T) {
	if path := os.Getenv("AGAINROM_GENERATED_MISSION_INPUT"); path != "" {
		generatedMissionCold(t, path)
		return
	}
	c := releaseFront(t).Campaign.Value()
	mission := slices.Min(append(slices.Clone(c.Main), c.Side...))
	generatedMissionRoute(t, mission, func(t *testing.T, f *FrontEnd) { applyWidePools(t, f) })
}

// applyWideNumerics lifts every other numeric actor field except the load,
// capacity and type fields (which have their own admission rules) by a
// multiple of its word width, on two living actors: one by a positive and one
// by a negative multiple.
func applyWideNumerics(t *testing.T, f *FrontEnd, skills bool, extra ...int) {
	t.Helper()
	w := f.live.world
	before := w.Entities()
	values := map[sim.EntityID]sim.ActorValues{}
	for _, e := range before {
		v := e.Values()
		v.Widths = nil
		group := e.Group
		v.NativeGroup = &group
		values[e.ID] = v
	}
	var picked []sim.EntityID
	for _, e := range before {
		if len(picked) < 2 && e.Decay == 0 && e.HP > 0 {
			picked = append(picked, e.ID)
		}
	}
	if len(picked) != 2 {
		t.Fatal("mission has fewer than two living actors")
	}
	for n, id := range picked {
		e, _ := w.Entity(id)
		v := values[id]
		for field, probe := range wideNumericFields(&e) {
			if field < 4 || (field == 19 || field == 20 || field == 21) && !slices.Contains(extra, field) || !skills && field >= 22 && field <= 27 {
				continue
			}
			modulus := int64(1) << probe.width
			wire := uint32(*probe.value) & uint32(modulus-1)
			lift := modulus * int64(field%3+1)
			if n == 1 {
				lift = -lift
			}
			v.Widths = append(v.Widths, sim.ActorNumericResidue{Field: uint8(field), Wire: wire, Lift: lift})
		}
		values[id] = v
	}
	if err := w.RestoreCurrentContinuation(nil, values, w.Actions(), nil); err != nil {
		t.Fatal("width admission:", err)
	}
}

type wideNumericField struct {
	value *int32
	width uint8
}

// wideNumericFields lists the ordinal numeric fields in their persisted order,
// declared independently of the producer's own table.
func wideNumericFields(e *sim.Entity) []wideNumericField {
	r := []wideNumericField{{&e.HP, 16}, {&e.MaxHP, 16}, {&e.Mana, 16}, {&e.MaxMana, 16},
		{&e.Speed, 16}, {&e.Reaction, 16}, {&e.Mind, 16}, {&e.Spirit, 16},
		{&e.ToHit, 16}, {&e.Defence, 16}, {&e.Absorption, 16}, {&e.DamageBase, 8}, {&e.DamageSpread, 8},
		{&e.HealthRegenPeriod, 16}, {&e.ManaRegenPeriod, 16}, {&e.HealthRegeneration, 16}, {&e.ManaRegeneration, 16},
		{&e.AttackCharge, 8}, {&e.AttackRelax, 8}, {&e.Load, 16}, {&e.Capacity, 16}, {&e.TypeID, 16}}
	for i := range e.Skill {
		r = append(r, wideNumericField{&e.Skill[i], 16})
	}
	for i := range e.Protection {
		r = append(r, wideNumericField{&e.Protection[i], 16})
	}
	return r
}

// TestReleaseGeneratedWideNumericSAV runs every wide numeric field through
// ordinary SAVE, a cold LOAD, a second SAVE and a second cold LOAD without a
// tick between. A wide skill level makes the live mission derive the hero
// sheet again at the next tick, so the continuation test leaves skills out.
func TestReleaseGeneratedWideNumericSAV(t *testing.T) {
	c := releaseFront(t).Campaign.Value()
	mission := slices.Min(append(slices.Clone(c.Main), c.Side...))
	f, app := generatedMissionStart(t, mission)
	applyWidePools(t, f)
	applyWideNumerics(t, f, true, 19, 20, 21)
	want := generatedMissionSample(t, f)
	wantEntities := f.live.world.Entities()
	first := generatedMissionSave(t, f, app)
	firstRaw, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	cold, coldApp := loadSAVWindow(t, SaveStore{Dir: filepath.Dir(first)}, filepath.Base(first))
	requireGeneratedSample(t, generatedMissionSample(t, cold), want, 0)
	if !reflect.DeepEqual(wantEntities, cold.live.world.Entities()) {
		t.Fatal("first cold LOAD changed an actor field")
	}
	second := generatedMissionSave(t, cold, coldApp)
	secondRaw, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	// Item identities and runtime numbers are assigned again by each SAVE, so
	// the comparison covers every actor record and the actor operands.
	da, err := sav.DecodeDocumentData(firstRaw)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sav.DecodeDocumentData(secondRaw)
	if err != nil {
		t.Fatal(err)
	}
	if len(da.Objects) != len(db.Objects) {
		t.Fatal("second SAVE changed the object count", len(da.Objects), len(db.Objects))
	}
	actors := 0
	for i := range da.Objects {
		switch da.Objects[i].Class {
		case "Unit", "Human", "Humanoid":
			actors++
			if !reflect.DeepEqual(da.Objects[i], db.Objects[i]) {
				t.Fatalf("second SAVE changed actor record %d", i+1)
			}
		}
	}
	a1, err := readCurrentActions(&da)
	if err != nil || a1 == nil {
		t.Fatal(err)
	}
	a2, err := readCurrentActions(&db)
	if err != nil || a2 == nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a1.Values, a2.Values) {
		t.Fatal("second SAVE changed the current actor operands")
	}
	if actors == 0 {
		t.Fatal("no actor record was compared")
	}
	again, _ := loadSAVWindow(t, SaveStore{Dir: filepath.Dir(second)}, filepath.Base(second))
	requireGeneratedSample(t, generatedMissionSample(t, again), want, 0)
	if !reflect.DeepEqual(wantEntities, again.live.world.Entities()) {
		t.Fatal("second cold LOAD changed an actor field")
	}
}

// Skill levels are the one wide field group whose change makes the live mission
// derive the hero sheet again at the next tick, so a continuation over ticks
// leaves them out; every other numeric field keeps its wide value.
func TestReleaseGeneratedWideNumericContinuationSAV(t *testing.T) {
	if path := os.Getenv("AGAINROM_GENERATED_MISSION_INPUT"); path != "" {
		generatedMissionCold(t, path)
		return
	}
	c := releaseFront(t).Campaign.Value()
	mission := slices.Min(append(slices.Clone(c.Main), c.Side...))
	generatedMissionRoute(t, mission, func(t *testing.T, f *FrontEnd) {
		applyWidePools(t, f)
		applyWideNumerics(t, f, false)
	})
}
