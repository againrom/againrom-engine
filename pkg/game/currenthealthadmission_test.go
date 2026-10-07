package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentHealthAdmissionFixture(t *testing.T, hp, maximum uint32, widths []sim.ActorNumericResidue) *SnapshotSAVDocument {
	t.Helper()
	doc := &sav.DocumentData{Objects: []sav.DocumentRecordData{{Class: "Human", Values: []sav.DocumentValueData{
		{Name: "Identity", Value: 308064264}, {Name: "Health", Value: hp}, {Name: "HealthMax", Value: maximum},
	}}}}
	a := currentActionData{Version: 1, Bindings: []currentActionBinding{{ID: 85, Object: 1}},
		Actions: sim.ActionContinuations{Actors: []sim.ActorContinuation{{Entity: 85}}},
		Values:  map[sim.EntityID]sim.ActorValues{85: {Widths: widths}}}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, b); err != nil {
		t.Fatal(err)
	}
	return &SnapshotSAVDocument{Document: doc}
}

func TestCurrentHealthAdmissionAnchorsAndBounds(t *testing.T) {
	for _, tc := range []struct {
		name        string
		hp, maximum uint32
		widths      []sim.ActorNumericResidue
		want        currentAdmissionHealth
	}{
		{"current85", 65535, 0, []sim.ActorNumericResidue{{Field: 0, Wire: 65535, Lift: 65536}, {Field: 1, Wire: 0, Lift: 65536}}, currentAdmissionHealth{65535, 65536, 65535, 0}},
		{"current82", 32768, 0, []sim.ActorNumericResidue{{Field: 0, Wire: 32768, Lift: 65536}, {Field: 1, Wire: 0, Lift: 65536}}, currentAdmissionHealth{32768, 65536, 32768, 0}},
		{"signedmax", 65535, 65535, []sim.ActorNumericResidue{{Field: 0, Wire: 65535, Lift: 2147483648}, {Field: 1, Wire: 65535, Lift: 2147418112}}, currentAdmissionHealth{2147483647, 2147483647, 65535, 65535}},
		{"unsignedmaximum", 32767, 65535, nil, currentAdmissionHealth{32767, 65535, 32767, 65535}},
		{"ordinaryhealthwins", 65534, 0, []sim.ActorNumericResidue{{Field: 0, Wire: 65535, Lift: 65536}, {Field: 1, Wire: 0, Lift: 65536}}, currentAdmissionHealth{-2, 65536, 65534, 0}},
		{"ordinarymaximumwins", 65535, 300, []sim.ActorNumericResidue{{Field: 0, Wire: 65535, Lift: 65536}, {Field: 1, Wire: 0, Lift: 65536}}, currentAdmissionHealth{65535, 300, 65535, 300}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := currentHealthAdmissionFixture(t, tc.hp, tc.maximum, tc.widths)
			before, _ := json.Marshal(state)
			got, err := currentAdmissionHealthByIdentity(state)
			if err != nil || got[308064264] != tc.want || len(got) != 1 {
				t.Fatal(got, err, tc.want)
			}
			after, _ := json.Marshal(state)
			if string(before) != string(after) {
				t.Fatal("admission projection mutated input")
			}
		})
	}
	for _, widths := range [][]sim.ActorNumericResidue{
		{{Field: 0, Wire: 65535, Lift: 0}}, {{Field: 0, Wire: 65536, Lift: 65536}}, {{Field: 0, Wire: 65535, Lift: 1}}, {{Field: 0, Wire: 65535, Lift: 1 << 32}},
		{{Field: 0, Wire: 65535, Lift: 65536}, {Field: 0, Wire: 65535, Lift: 65536}}, {{Field: 1, Wire: 0, Lift: -1 << 32}},
	} {
		state := currentHealthAdmissionFixture(t, 65535, 0, widths)
		before, _ := json.Marshal(state)
		if _, err := currentAdmissionHealthByIdentity(state); err == nil {
			t.Fatal("malformed admission residue accepted", widths)
		}
		after, _ := json.Marshal(state)
		if string(before) != string(after) {
			t.Fatal("rejected projection mutated input")
		}
	}
	for _, state := range []*SnapshotSAVDocument{nil, {}, {Document: &sav.DocumentData{}}} {
		got, err := currentAdmissionHealthByIdentity(state)
		if err != nil || got != nil {
			t.Fatal("original save acquired current override", got, err)
		}
	}
}

func TestCurrentHealthAdmissionTwoSAVCycles(t *testing.T) {
	f := currentProfileFront(t, sim.ProfileNativeRetired)
	entities := f.live.world.Entities()
	for i := range entities {
		entities[i].HP, entities[i].MaxHP = 65535, 65536
	}
	w, err := sim.NewWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	for cycle := 0; cycle < 2; cycle++ {
		raw, _ := saveCurrentProfile(t, f)
		cold := generatedCurrentCold(t, raw)
		if !reflect.DeepEqual(f.live.world.Entities(), cold.live.world.Entities()) {
			t.Fatal("current health admission changed entities", cycle)
		}
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, "wide health current LOAD")
		f.LiveAdvanceCasts(1)
		cold.LiveAdvanceCasts(1)
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, "wide health next tick")
		f = cold
	}
}

func TestCurrentHealthAdmissionRejectsMalformedSaveAtomically(t *testing.T) {
	for _, kind := range []string{"missing binding", "missing endpoint", "structure binding", "duplicate action", "duplicate binding", "duplicate object", "bad wire", "zero lift", "fractional lift", "overflow", "duplicate field"} {
		t.Run(kind, func(t *testing.T) {
			f := currentProfileFront(t, sim.ProfileNativeRetired)
			entities := f.live.world.Entities()
			for i := range entities {
				entities[i].HP, entities[i].MaxHP = 65535, 65536
			}
			w, err := sim.NewWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, entities)
			if err != nil {
				t.Fatal(err)
			}
			f.live.world, f.live.mission.state.World = w, w
			_, doc := saveCurrentProfile(t, f)
			a, err := readCurrentActions(&doc)
			if err != nil || a == nil || len(a.Actions.Actors) != 2 {
				t.Fatal("wide action fixture", err)
			}
			row := a.Actions.Actors[1]
			at := -1
			for i, b := range a.Bindings {
				if !b.Structure && b.ID == row.Entity {
					at = i
				}
			}
			if at < 0 {
				t.Fatal("wide binding fixture")
			}
			v := a.Values[row.Entity]
			field := -1
			for i, x := range v.Widths {
				if x.Field == 0 {
					field = i
				}
			}
			if field < 0 || v.Widths[field].Wire != 65535 || v.Widths[field].Lift != 65536 {
				t.Fatal("literal HP operand missing", v.Widths)
			}
			switch kind {
			case "missing binding":
				a.Bindings = append(a.Bindings[:at], a.Bindings[at+1:]...)
			case "missing endpoint":
				a.Bindings[at].Missing, a.Bindings[at].Object = true, 0
			case "structure binding":
				a.Bindings[at].Structure = true
			case "duplicate action":
				a.Actions.Actors = append(a.Actions.Actors, row)
			case "duplicate binding":
				a.Bindings = append(a.Bindings, a.Bindings[at])
			case "duplicate object":
				for _, b := range a.Bindings {
					if !b.Structure && b.ID != row.Entity && !b.Missing {
						a.Bindings[at].Object = b.Object
						break
					}
				}
			case "bad wire":
				v.Widths[field].Wire = 65536
			case "zero lift":
				v.Widths[field].Lift = 0
			case "fractional lift":
				v.Widths[field].Lift = 1
			case "overflow":
				v.Widths[field].Lift = 1 << 32
			case "duplicate field":
				v.Widths = append(v.Widths, v.Widths[field])
			}
			a.Values[row.Entity] = v
			leaf, err := json.Marshal(a)
			if err == nil {
				err = sav.SetNativeActions(&doc.State, leaf)
			}
			if err != nil {
				t.Fatal(err)
			}
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal("encode admission control", err)
			}
			live, before := f.live, f.live.world.Hash()
			open, town, err := f.RestoreOriginal(raw)
			if err == nil && !town {
				err = f.App("invalid current health").OpenMission(open)
			}
			if err == nil || f.live != live || f.live.world.Hash() != before {
				t.Fatal("invalid admission changed active state", err)
			}
		})
	}
}

// This exercises the public continuation endpoint on an already admitted full
// health value. A second pass must not add the same 65536 lift again.
func TestCurrentHealthAdmissionDecodeIsIdempotent(t *testing.T) {
	f := currentProfileFront(t, sim.ProfileNativeRetired)
	entities := f.live.world.Entities()
	values := make(map[sim.EntityID]sim.ActorValues)
	for i := range entities {
		entities[i].HP, entities[i].MaxHP = 65535, 65536
		v := entities[i].Values()
		v.Widths = []sim.ActorNumericResidue{{Field: 0, Wire: 65535, Lift: 65536}, {Field: 1, Wire: 0, Lift: 65536}}
		values[entities[i].ID] = v
	}
	w, err := sim.NewWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	for pass := 0; pass < 2; pass++ {
		if err := w.RestoreCurrentContinuation(nil, values, w.Actions(), nil); err != nil {
			t.Fatal(err)
		}
		for _, e := range w.Entities() {
			if e.HP != 65535 || e.MaxHP != 65536 {
				t.Fatal("health lift reapplied", pass, e)
			}
		}
		if w.Hash() != before {
			t.Fatal("health reapplication changed full World", pass)
		}
	}
	for i := range entities {
		entities[i].HP, entities[i].MaxHP = 123, 300
	}
	w, err = sim.NewWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	before = w.Hash()
	if err := w.RestoreCurrentContinuation(nil, values, w.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	if w.Hash() != before {
		t.Fatal("stale health anchor overrode ordinary words")
	}
}

func currentHealthWideFront(t *testing.T, hp, maximum int32, decay sim.DecayStage) *FrontEnd {
	t.Helper()
	f := currentProfileFront(t, sim.ProfileNativeRetired)
	entities := f.live.world.Entities()
	for i := range entities {
		entities[i].HP, entities[i].MaxHP, entities[i].Decay = hp, maximum, decay
	}
	w, err := sim.NewWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	return f
}

func TestCurrentHealthAdmissionStageAndPosition(t *testing.T) {
	for _, change := range []string{"stale stage one", "ordinary position", "ordinary health"} {
		t.Run(change, func(t *testing.T) {
			f := currentHealthWideFront(t, 65535, 65536, sim.DecayNone)
			_, doc := saveCurrentProfile(t, f)
			record := generatedActorRecord(t, &doc, 0)
			leaf, _, err := sav.NativeActions(doc.State)
			if err != nil {
				t.Fatal(err)
			}
			want := currentProfileExpectedWorld(t, f.live.world)
			switch change {
			case "stale stage one":
				mustSetValue(record, "Stage", 1)
			case "ordinary position":
				position, err := savedActorRaw(record, "Block12", 12)
				if err != nil {
					t.Fatal(err)
				}
				binary.LittleEndian.PutUint16(position[:2], 0x0605)
				binary.LittleEndian.PutUint16(position[2:4], 0x0605)
				mustSetRaw(record, "Block12", position)
				actions := want.Actions()
				found := false
				for i := range actions.Actors {
					if actions.Actors[i].Entity == 0 {
						actions.Actors[i].X, actions.Actors[i].Y = 5, 6
						found = true
					}
				}
				if !found {
					t.Fatal("missing literal actor zero")
				}
				if err := want.RestoreActions(actions, nil); err != nil {
					t.Fatal(err)
				}
			case "ordinary health":
				mustSetValue(record, "Health", 123)
				mustSetValue(record, "HealthMax", 300)
				entities := f.live.world.Entities()
				for i := range entities {
					if entities[i].ID == 0 {
						entities[i].HP, entities[i].MaxHP = 123, 300
					}
				}
				want, err = sim.NewWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, entities)
				if err != nil {
					t.Fatal(err)
				}
			}
			after, _, err := sav.NativeActions(doc.State)
			if err != nil || !bytes.Equal(leaf, after) {
				t.Fatal("ordinary edit changed current policy", err)
			}
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			for cycle := 0; cycle < 2; cycle++ {
				cold := generatedCurrentCold(t, raw)
				assertCurrentWorldEqual(t, want, cold.live.world, "health ordering ordinary edit LOAD")
				e := generatedCampaignEntity(t, cold, 0)
				if !e.Alive() || e.Decay != sim.DecayNone {
					t.Fatal("full positive HP became dying", change, cycle, e)
				}
				if change == "ordinary position" && (e.X != 5 || e.Y != 6) {
					t.Fatal("Block12 lost to continuation", e)
				}
				if change == "ordinary health" && (e.HP != 123 || e.MaxHP != 300) {
					t.Fatal("stale width defeated ordinary health", e)
				}
				sim.Step(want, nil)
				sim.Step(cold.live.world, nil)
				assertCurrentWorldEqual(t, want, cold.live.world, "health ordering ordinary edit next tick")
				raw, _ = saveCurrentProfile(t, cold)
			}
		})
	}
	f := currentHealthWideFront(t, 65535, 65536, sim.DecayNone)
	_, doc := saveCurrentProfile(t, f)
	mustSetValue(generatedActorRecord(t, &doc, 0), "Stage", 5)
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	before, live := f.live.world.Hash(), f.live
	open, town, err := f.RestoreOriginal(raw)
	if err == nil && !town {
		err = f.App("invalid wide terminal").OpenMission(open)
	}
	if err == nil || f.live != live || f.live.world.Hash() != before {
		t.Fatal("wide living actor admitted terminal wire stage", err)
	}
}

func TestCurrentHealthAdmissionDyingAndRetained(t *testing.T) {
	for _, hp := range []int32{0, -1} {
		f := currentHealthWideFront(t, hp, 65536, sim.DecayFallen)
		for cycle := 0; cycle < 2; cycle++ {
			raw, _ := saveCurrentProfile(t, f)
			cold := generatedCurrentCold(t, raw)
			assertCurrentWorldEqual(t, f.live.world, cold.live.world, "dying wide maximum LOAD")
			sim.Step(f.live.world, nil)
			sim.Step(cold.live.world, nil)
			assertCurrentWorldEqual(t, f.live.world, cold.live.world, "dying wide maximum next tick")
			f = cold
		}
	}
	f := currentBoundCorpseFixture(t, 0xabcdef77, -14)
	dead := f.live.world.OriginalDeadActors()
	if len(dead) != 1 {
		t.Fatal("retained fixture", dead)
	}
	before := f.live.world.Entities()
	prior, present := f.live.world.Entity(dead[0].ID)
	if !present || prior.MapUnitID != 92 || prior.HP != -14 || prior.MaxHP != 100 || prior.Decay != sim.DecayStage(2) || prior.Alive() || dead[0].Source.State.HP != -14 || dead[0].Current.HP != -14 {
		t.Fatal("retained current fixture lacks literal ALM health/lifecycle", prior.ID, prior.HP, prior.MaxHP, prior.Decay, dead[0])
	}
	// The ordinary original corpse held MaxHP 30, but that import only owns
	// its death tuple. The current ALM Entity retains SpawnHP 100. A stale
	// operand anchored to 30 must lose to the actual current maximum 100.
	_, staleMaximum, err := sim.CurrentActorHealthFromWords(65522, 100, []sim.ActorNumericResidue{{Field: 1, Wire: 30, Lift: 65536}})
	if err != nil || staleMaximum != 100 {
		t.Fatal("stale corpse backing overrode current maximum", staleMaximum, err)
	}
	doc, err := sav.DecodeDocumentData(currentRuntimeSave(t, f))
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, entity := range f.live.world.Entities() {
		value := a.Values[entity.ID]
		value.Decay, value.Dwell = entity.Decay, entity.Dwell
		a.Values[entity.ID] = value
	}
	v := a.Values[dead[0].ID]
	for _, x := range v.Widths {
		if x.Field < 2 {
			t.Fatal("retained fixture already has health width", x)
		}
	}
	v.Widths = append(v.Widths, sim.ActorNumericResidue{Field: 0, Wire: 65522, Lift: -65536}, sim.ActorNumericResidue{Field: 1, Wire: 100, Lift: 65536})
	a.Values[dead[0].ID] = v
	if err := f.live.world.RestoreCurrentContinuation(nil, a.Values, f.live.world.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	e, ok := f.live.world.Entity(dead[0].ID)
	if !ok || e.HP != -65550 || e.MaxHP != 65636 || e.Alive() {
		t.Fatal("literal retained current health", e)
	}
	for i := range before {
		if before[i].ID == dead[0].ID {
			before[i].HP, before[i].MaxHP = -65550, 65636
		}
	}
	if !reflect.DeepEqual(before, f.live.world.Entities()) {
		t.Fatal("retained fixture changed fields beyond current HP/MaxHP")
	}
	for cycle := 0; cycle < 2; cycle++ {
		raw := currentRuntimeSave(t, f)
		cold := openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, "wide retained body LOAD")
		sim.Step(f.live.world, nil)
		sim.Step(cold.live.world, nil)
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, "wide retained body next tick")
		f = cold
	}
}

func TestCurrentHealthAdmissionPoolsKeepWideValueOnlyForMatchingWords(t *testing.T) {
	for _, tc := range []struct {
		name         string
		hp, maxHP    int32
		current      bool
		wantHP, want int32
	}{
		{"matching words", -1, 0, true, 65535, 65536},
		{"ordinary HP wins", -2, 0, true, -2, 65536},
		{"ordinary maximum wins", -1, 300, true, 65535, 300},
		{"not a current actor", 100, 300, false, 100, 300},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := currentHealthWideFront(t, 65535, 65536, sim.DecayNone)
			w := f.live.world
			var current []sim.EntityID
			if tc.current {
				current = []sim.EntityID{0}
			}
			if err := w.ImportOriginalActorPools([]sim.OriginalActorPools{{ID: 0, HP: tc.hp, MaxHP: tc.maxHP, Mana: 10, MaxMana: 900}}, current...); err != nil {
				t.Fatal(err)
			}
			e, ok := w.Entity(0)
			if !ok || e.HP != tc.wantHP || e.MaxHP != tc.want {
				t.Fatalf("HP %d/%d, want %d/%d", e.HP, e.MaxHP, tc.wantHP, tc.want)
			}
		})
	}
}

// DIV-1675: a Unit capacity whose low word is 0 is written as the default word,
// so cold LOAD restores the default; every other low word survives.
func TestCurrentUnitCapacityWithLowWordZeroRestoresTheDefaultWord(t *testing.T) {
	for _, tc := range []struct{ capacity, want int32 }{{65536, 300}, {131072, 300}, {-65536, 300}, {65836, 65836}, {300, 300}} {
		f := currentProfileFront(t, sim.ProfileNativeRetired)
		entities := f.live.world.Entities()
		for i := range entities {
			entities[i].Capacity = tc.capacity
		}
		w, err := sim.NewWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, entities)
		if err != nil {
			t.Fatal(err)
		}
		f.live.world, f.live.mission.state.World = w, w
		raw, _ := saveCurrentProfile(t, f)
		for _, e := range generatedCurrentCold(t, raw).live.world.Entities() {
			if e.Capacity != tc.want {
				t.Fatalf("capacity %d restored as %d, want %d", tc.capacity, e.Capacity, tc.want)
			}
		}
	}
}
