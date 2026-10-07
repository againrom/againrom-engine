package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentProfileFront(t *testing.T, basis sim.CurrentProfileBasis) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	if err := f.App("current profile").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	actors := []sim.Entity{
		{ID: 0, MapUnitID: 91, TypeID: 1, Owner: 1, X: 3, Y: 4, HP: 40, MaxHP: 80, Mana: 10, MaxMana: 900},
		{ID: 7, MapUnitID: 92, TypeID: 1, Owner: 1, X: 5, Y: 4, HP: 40, MaxHP: 80, Mana: 20, MaxMana: 900},
	}
	for i := range actors {
		actors[i].CurrentProfileBasis = basis
		actors[i].HealthRegenPeriod, actors[i].Capacity = 100, data.UnitCapacity()
		actors[i].HealthHundredths, actors[i].ManaHundredths = 17, 19
	}
	w, err := sim.NewWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, actors)
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	return f
}

func saveCurrentProfile(t *testing.T, f *FrontEnd) ([]byte, sav.DocumentData) {
	t.Helper()
	w := f.live.world
	native, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var verified sim.World
	if err := verified.UnmarshalBinary(native); err != nil || verified.Hash() != w.Hash() {
		t.Fatal("native profile validation", err)
	}
	s, _, err := f.Snapshot(true)
	if err != nil || !bytes.Equal(s.World, native) {
		t.Fatal("current profile snapshot", err)
	}
	raw, err := f.ExportCurrentSave(s, "Current profile")
	if err != nil || len(raw) == 0 || w.Hash() != verified.Hash() {
		t.Fatal("current profile SAVE", err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, doc
}

func TestCurrentProfileZeroPeriodSurvivesSAVAndRegen(t *testing.T) {
	for _, basis := range []sim.CurrentProfileBasis{sim.ProfileNative, sim.ProfileNativeRetired} {
		t.Run(fmt.Sprint(basis), func(t *testing.T) {
			f := currentProfileFront(t, basis)
			for cycle := range 2 {
				raw, _ := saveCurrentProfile(t, f)
				cold := generatedCurrentCold(t, raw)
				// The current continuation explicitly restores the native absence
				// policy after the ordinary SAV constructor head is loaded.
				want := currentProfileExpectedWorld(t, f.live.world)
				assertCurrentWorldEqual(t, want, cold.live.world, "zero-period cold LOAD")
				for tick := range 96 {
					sim.Step(want, nil)
					sim.Step(cold.live.world, nil)
					assertCurrentWorldEqual(t, want, cold.live.world, fmt.Sprintf("cycle %d tick %d", cycle, tick))
				}
				for _, e := range cold.live.world.Entities() {
					wantMana := int32(10)
					if e.ID == 7 {
						wantMana = 20
					}
					if e.CurrentProfileBasis != basis || e.Mana != wantMana || e.ManaHundredths != 19 || e.ManaRegenPeriod != 0 || e.HP <= 40 {
						t.Fatal("native zero-period or real health regeneration changed", e)
					}
				}
				f = cold
			}
		})
	}
}

func currentProfileExpectedWorld(t *testing.T, source *sim.World) *sim.World {
	t.Helper()
	raw, err := source.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var want sim.World
	if err := want.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	return &want
}

func TestCurrentProfileOrdinaryPeriodWinsWithUnchangedPolicy(t *testing.T) {
	f := currentProfileFront(t, sim.ProfileNative)
	_, doc := saveCurrentProfile(t, f)
	leaf, _, err := sav.NativeActions(doc.State)
	if err != nil {
		t.Fatal(err)
	}
	actor := generatedActorRecord(t, &doc, 0)
	for name, value := range map[string]uint32{"ManaRegen": 100, "ManaMax": 700, "Mana": 23, "UA3": 31} {
		if err := savedActorSetValue(actor, name, value); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := sav.EncodeDocumentData(doc)
	unchanged, _, leafErr := sav.NativeActions(doc.State)
	if err != nil || leafErr != nil || !bytes.Equal(leaf, unchanged) {
		t.Fatal("ordinary profile edit changed policy", err, leafErr)
	}
	f = generatedCurrentCold(t, raw)
	e := generatedCampaignEntity(t, f, 0)
	if e.CurrentProfileBasis != sim.ProfileNative || e.ManaRegenPeriod != 100 || e.Mana != 23 || e.MaxMana != 700 || e.ManaHundredths != 31 {
		t.Fatal("ordinary profile values were overwritten", e)
	}
	for cycle := range 2 {
		raw, _ := saveCurrentProfile(t, f)
		cold := generatedCurrentCold(t, raw)
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, "edited profile cold LOAD")
		for tick := range 32 {
			sim.Step(f.live.world, nil)
			sim.Step(cold.live.world, nil)
			assertCurrentWorldEqual(t, f.live.world, cold.live.world, fmt.Sprintf("edited cycle %d tick %d", cycle, tick))
		}
		if generatedCampaignEntity(t, cold, 0).Mana <= 23 || generatedCampaignEntity(t, cold, 7).Mana != 20 {
			t.Fatal("ordinary positive period did not select the exact actor's next regeneration")
		}
		f = cold
	}
}

func TestCurrentProfileAdmissionRejectsMissingOrInvalidPolicyAtomically(t *testing.T) {
	for _, kind := range []string{"missing policy", "original arithmetic", "invalid arithmetic", "missing binding", "missing endpoint", "structure binding", "duplicate action", "duplicate object"} {
		t.Run(kind, func(t *testing.T) {
			f := currentProfileFront(t, sim.ProfileNative)
			_, doc := saveCurrentProfile(t, f)
			a, err := readCurrentActions(&doc)
			if err != nil || a == nil || len(a.Actions.Actors) != 2 {
				t.Fatal("current action fixture", err)
			}
			row := &a.Actions.Actors[1]
			var at int
			for i, b := range a.Bindings {
				if !b.Structure && b.ID == row.Entity {
					at = i
				}
			}
			switch kind {
			case "missing policy":
				row.ProfileBasis = nil
			case "original arithmetic":
				*row.ProfileBasis = sim.ProfileOriginalCurrent
			case "invalid arithmetic":
				*row.ProfileBasis = 255
			case "missing binding":
				a.Bindings = append(a.Bindings[:at], a.Bindings[at+1:]...)
			case "missing endpoint":
				a.Bindings[at].Missing, a.Bindings[at].Object = true, 0
			case "structure binding":
				a.Bindings[at].Structure = true
			case "duplicate action":
				a.Actions.Actors = append(a.Actions.Actors, *row)
			case "duplicate object":
				for _, b := range a.Bindings {
					if !b.Structure && b.ID != row.Entity && !b.Missing {
						a.Bindings[at].Object = b.Object
						break
					}
				}
			}
			leaf, err := json.Marshal(a)
			if err == nil {
				err = sav.SetNativeActions(&doc.State, leaf)
			}
			if err != nil {
				t.Fatal(err)
			}
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal("encode policy control", err)
			}
			live, before := f.live, f.live.world.Hash()
			open, town, err := f.RestoreOriginal(raw)
			if err == nil && !town {
				err = f.App("invalid current profile").OpenMission(open)
			}
			if err == nil || f.live != live || f.live.world.Hash() != before {
				t.Fatal("invalid profile changed active state", err)
			}
		})
	}
}
