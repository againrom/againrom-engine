package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func currentEffectFront(t *testing.T, magnitude int32) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	if err := f.App("current effect width").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	spells := make(dbCollection, 21)
	for i := 1; i < len(spells); i++ {
		spells[i] = dbEntry{name: "synthetic effect spell", params: make([]int32, 19)}
	}
	p := spells[20].params
	p[1], p[2], p[4], p[6], p[8], p[14], p[18] = 10, 1, 1, 8, 1, 2, 1
	spells[20].strings = []string{fmt.Sprintf("Absorbtion=%d:duration 64", magnitude)}
	f.Table.Spells = spells
	rules := mapload.SpellRules(f.Table)
	if len(rules) != 20 || rules[19].EffectMagnitude != magnitude {
		t.Fatal("synthetic table did not supply current int32 magnitude", rules)
	}
	actors := emWorld(t, 20, rules[19]).Entities()
	for i := range actors {
		actors[i].TypeID = 1
		actors[i].X, actors[i].Y = int32(3+2*i), 4
		actors[i].HealthRegenPeriod, actors[i].ManaRegenPeriod, actors[i].Capacity = 1, 1, data.UnitCapacity()
	}
	w, err := sim.NewSpelledWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, actors, nil, rules)
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	sim.Step(w, []sim.Command{sim.Cast(1, 2, 20)})
	for tick := 0; tick < 64 && len(w.ActiveEffects()) == 0; tick++ {
		sim.Step(w, nil)
	}
	effects := w.ActiveEffects()
	if len(effects) != 1 || effects[0].Target != 2 || effects[0].Magnitude != magnitude || effects[0].Remaining <= 40 {
		t.Fatal("Cast did not establish the bounded native attachment", effects)
	}
	for _, e := range w.Entities() {
		if e.SourceBinding.Class != 0 || e.ActorLoad.Source.Class != 0 || e.ID == 2 && e.Absorption != magnitude {
			t.Fatal("native current actor differs", e)
		}
	}
	return f
}

func openCurrentEffectSave(t *testing.T, source *FrontEnd, raw []byte) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	f.Table, f.Humans = source.Table, source.Humans
	open, town, err := f.RestoreOriginal(raw)
	if err == nil && !town {
		err = f.App("current effect cold LOAD").OpenMission(open)
	}
	if err != nil || town {
		t.Fatal("current effect cold LOAD", err)
	}
	return f
}

func saveCurrentEffect(t *testing.T, f *FrontEnd) ([]byte, sav.DocumentData, *currentActionData) {
	t.Helper()
	before := f.live.world.Hash()
	native, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var verified sim.World
	if err := verified.UnmarshalBinary(native); err != nil || verified.Hash() != before {
		t.Fatal("native validation", err)
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil || !bytes.Equal(snapshot.World, native) {
		t.Fatal("current snapshot", err)
	}
	raw, err := f.ExportCurrentSave(snapshot, "Current effect width")
	if err != nil || len(raw) == 0 || f.live.world.Hash() != before {
		t.Fatal("current effect SAVE", err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal("current continuation", err)
	}
	return raw, doc, a
}

func TestCurrentEffectMagnitudeSurvivesSAVAndExpiry(t *testing.T) {
	for _, magnitude := range []int32{20, 40000, -40000} {
		t.Run(fmt.Sprint(magnitude), func(t *testing.T) {
			f := currentEffectFront(t, magnitude)
			var previous *sim.World
			for cycle := range 2 {
				raw, doc, a := saveCurrentEffect(t, f)
				wantRows := 0
				if magnitude != int32(int16(magnitude)) {
					wantRows = 1
				}
				if len(a.EffectWidths) != wantRows {
					t.Fatal("width residue is not sparse", a.EffectWidths)
				}
				if wantRows != 0 {
					row := a.EffectWidths[0]
					operand, err := savedStructureValue(&doc.Objects[row.Object-1], "E40")
					if err != nil || uint16(operand) != uint16(magnitude) || uint16(operand>>16) != f.live.world.ActiveEffects()[0].Remaining || int64(int16(row.Wire))+row.Lift != int64(magnitude) {
						t.Fatal("ordinary word and width lift differ", operand, row, err)
					}
				}
				cold := openCurrentEffectSave(t, f, raw)
				assertCurrentWorldEqual(t, f.live.world, cold.live.world, "current effect cold LOAD")
				for tick := range 20 {
					sim.Step(f.live.world, nil)
					sim.Step(cold.live.world, nil)
					if f.live.world.Hash() != cold.live.world.Hash() {
						t.Fatalf("cycle %d tick %d changed full World", cycle, tick)
					}
				}
				previous, f = f.live.world, cold
			}
			for tick := 0; tick < 128 && len(f.live.world.ActiveEffects()) != 0; tick++ {
				sim.Step(previous, nil)
				sim.Step(f.live.world, nil)
				assertCurrentWorldEqual(t, previous, f.live.world, "current effect expiry")
			}
			if len(f.live.world.ActiveEffects()) != 0 {
				t.Fatal("current attachment did not expire")
			}
			for _, e := range f.live.world.Entities() {
				if e.ID == 2 && e.Absorption != 0 {
					t.Fatal("expiry failed to reverse the exact magnitude", e.Absorption)
				}
			}
			raw, _, a := saveCurrentEffect(t, f)
			if len(a.EffectWidths) != 0 {
				t.Fatal("expired attachment retained width residue")
			}
			assertCurrentWorldEqual(t, f.live.world, openCurrentEffectSave(t, f, raw).live.world, "expired effect cold LOAD")
		})
	}
}

func TestCurrentEffectMagnitudeOrdinaryEditsWin(t *testing.T) {
	for _, magnitude := range []int32{0, -7, 32767} {
		t.Run(fmt.Sprint(magnitude), func(t *testing.T) {
			f := currentEffectFront(t, 40000)
			_, doc, a := saveCurrentEffect(t, f)
			row := a.EffectWidths[0]
			beforeLeaf, _, err := sav.NativeActions(doc.State)
			if err != nil {
				t.Fatal(err)
			}
			want := f.live.world.ActiveEffects()
			want[0].Magnitude = magnitude
			savedObjectSetValue(&doc.Objects[row.Object-1], "E40", uint32(uint16(magnitude))|uint32(want[0].Remaining)<<16)
			raw, err := sav.EncodeDocumentData(doc)
			afterLeaf, _, leafErr := sav.NativeActions(doc.State)
			if err != nil || leafErr != nil || !bytes.Equal(beforeLeaf, afterLeaf) {
				t.Fatal("ordinary edit changed width policy", err, leafErr)
			}
			cold := openCurrentEffectSave(t, f, raw)
			if !reflect.DeepEqual(want, cold.live.world.ActiveEffects()) || !reflect.DeepEqual(f.live.world.Entities(), cold.live.world.Entities()) {
				t.Fatal("ordinary effect edit lost precedence or changed actor values", cold.live.world.ActiveEffects())
			}
			for cycle := range 2 {
				raw, _, next := saveCurrentEffect(t, cold)
				if len(next.EffectWidths) != 0 {
					t.Fatal("edited representable magnitude retained width residue")
				}
				second := openCurrentEffectSave(t, cold, raw)
				assertCurrentWorldEqual(t, cold.live.world, second.live.world, "ordinary effect cold LOAD")
				for tick := range 20 {
					sim.Step(cold.live.world, nil)
					sim.Step(second.live.world, nil)
					if cold.live.world.Hash() != second.live.world.Hash() {
						t.Fatalf("edited cycle %d tick %d differs", cycle, tick)
					}
				}
				cold = second
			}
		})
	}
}

func TestCurrentEffectWidthAnchorUsesOrdinaryFieldsAndOwner(t *testing.T) {
	f := currentEffectFront(t, 40000)
	raw, _, _ := saveCurrentEffect(t, f)
	for _, field := range []string{"E0C", "E3C", "E3D", "E40", "owner edge"} {
		t.Run(field, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			row := a.EffectWidths[0]
			if field == "owner edge" {
				savedObjectSetRefs(&doc.Objects[row.Actor-1], "Effects", nil, true)
			} else {
				value, err := savedStructureValue(&doc.Objects[row.Object-1], field)
				if err != nil {
					t.Fatal(err)
				}
				savedObjectSetValue(&doc.Objects[row.Object-1], field, value+1)
			}
			rows, err := matchingCurrentEffectWidths(&doc, a.EffectWidths)
			if err != nil || len(rows) != 0 {
				t.Fatal("ordinary edit retained stale width", rows, err)
			}
		})
	}
}

func TestCurrentEffectWidthMalformedLOADIsAtomic(t *testing.T) {
	f := currentEffectFront(t, 40000)
	raw, _, _ := saveCurrentEffect(t, f)
	for _, name := range []string{"zero lift", "non-width lift", "huge lift", "overflow", "duplicate", "null effect", "null actor", "actor class", "missing values", "zero anchor", "spell range"} {
		t.Run(name, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			row := &a.EffectWidths[0]
			switch name {
			case "zero lift":
				row.Lift = 0
			case "non-width lift":
				row.Lift = 1
			case "huge lift":
				row.Lift = 1<<63 - 1
			case "overflow":
				row.Wire, row.Lift = 1, 1<<31
			case "duplicate":
				a.EffectWidths = append(a.EffectWidths, *row)
			case "null effect":
				row.Object = 0
			case "null actor":
				row.Actor = 0
			case "actor class":
				row.Actor = row.Object
			case "missing values":
				a.Values = nil
			case "zero anchor":
				row.Anchor = [32]byte{}
			case "spell range":
				row.Spell = 29
			}
			leaf, err := json.Marshal(a)
			if err == nil {
				err = sav.SetNativeActions(&doc.State, leaf)
			}
			if err != nil {
				t.Fatal("prepare malformed current leaf", err)
			}
			bad, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal("encode malformed leaf witness", err)
			}
			before, live := f.live.world.Hash(), f.live
			open, town, err := f.RestoreOriginal(bad)
			if err == nil && !town {
				err = f.App("malformed effect width").OpenMission(open)
			}
			if err == nil || f.live != live || f.live.world.Hash() != before {
				t.Fatal("malformed width LOAD changed live state", err)
			}
		})
	}
}
