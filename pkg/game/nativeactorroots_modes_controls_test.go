package game

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The fixture's real Cast establishes the effect/caster. The normal SAVE and
// cold LOAD establish mode policies; no expected actor DTO supplies values.
func nativeActorNativeModeFixture(t *testing.T) ([]byte, *FrontEnd, *Mission) {
	t.Helper()
	f := currentEffectFront(t, 10)
	raw, _, _ := saveCurrentEffect(t, f)
	ms, _, err := loadOriginalMission(f, raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, f, ms
}

func nativeActorModeCheck(t *testing.T, raw []byte, ms *Mission, observed []sim.Entity) []string {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	roots, reader, origins, err := readActorRoots(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	differences, _ := actorRootEntityDifferences(roots, reader, origins, ms.savedDocument, ms.World, observed, ms)
	return differences
}

func TestActorRootsNativeBookPackAndCasterModes(t *testing.T) {
	raw, _, ms := nativeActorNativeModeFixture(t)
	if differences := nativeActorModeCheck(t, raw, ms, ms.World.Entities()); len(differences) != 0 {
		t.Fatal("native carrier mode baseline", differences)
	}
	before := ms.World.Hash()
	for _, name := range []string{"membership", "book mode", "cached legacy slot", "missing actor"} {
		t.Run(name, func(t *testing.T) {
			observed := slices.Clone(ms.World.Entities())
			switch name {
			case "membership":
				observed[0].KnownSpells ^= uint32(1) << 1
			case "book mode":
				observed[0].Book.State = sim.BookAbsent
			case "cached legacy slot":
				observed[0].Book.Slots[0].ManaCost = 54321
			case "missing actor":
				observed = observed[1:]
			}
			if len(nativeActorModeCheck(t, raw, ms, observed)) == 0 {
				t.Fatal("accepted lost current book/subject")
			}
		})
	}
	if ms.World.Hash() != before {
		t.Fatal("comparison changed World")
	}
}

func TestActorRootsNativeRawBookAndEmptyHeaderLossControls(t *testing.T) {
	raw, source, ms := nativeActorNativeModeFixture(t)
	for _, name := range []string{"Inventory1C", "Inventory20", "HasInventory", "S09", "S0A", "S0C"} {
		t.Run(name, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			leaf, _, _ := sav.NativeActions(doc.State)
			changed := false
			for i := range doc.Objects {
				r := &doc.Objects[i]
				if strings.HasPrefix(name, "S0") && r.Class == "Spell" {
					mustSetValue(r, name, 73)
					changed = true
					break
				}
				if strings.HasPrefix(name, "Inventory") && (r.Class == "Unit" || r.Class == "Human") {
					mustSetValue(r, name, 17)
					changed = true
					break
				}
				if name == "HasInventory" && (r.Class == "Unit" || r.Class == "Human") {
					savedObjectSetValue(r, name, 0)
					r.Values = slices.DeleteFunc(r.Values, func(v sav.DocumentValueData) bool { return v.Name == "Inventory1C" || v.Name == "Inventory20" })
					r.RefSlots = slices.DeleteFunc(r.RefSlots, func(v sav.DocumentRefsData) bool { return v.Name == "Inventory" })
					r.Counts = slices.DeleteFunc(r.Counts, func(v sav.DocumentCountData) bool { return v.Name == "Inventory" })
					changed = true
					break
				}
			}
			if !changed {
				t.Fatal("independent raw control field absent", name)
			}
			back, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := sav.DecodeDocumentData(back)
			if err != nil {
				t.Fatal(err)
			}
			unchanged, _, _ := sav.NativeActions(decoded.State)
			if !slices.Equal(leaf, unchanged) {
				t.Fatal("ordinary edit changed native leaf")
			}
			if len(nativeActorModeCheck(t, back, ms, ms.World.Entities())) == 0 {
				t.Fatal("accepted stale current holder/book against raw edit", name)
			}
			if strings.HasPrefix(name, "S0") {
				cold, _, err := loadOriginalMission(source, back)
				if err != nil {
					t.Fatal(err)
				}
				if differences := nativeActorModeCheck(t, back, cold, cold.World.Entities()); len(differences) != 0 {
					t.Fatal("edited raw book did not promote exact current slots", differences)
				}
			}
		})
	}
}

func TestActorRootsNativeCasterLossControls(t *testing.T) {
	raw, _, ms := nativeActorNativeModeFixture(t)
	original := ms.World.Actions()
	if len(original.EffectCasters) != 1 || !original.EffectCasters[0].HasCaster {
		t.Fatal("real Cast did not supply exact caster")
	}
	for _, name := range []string{"caster", "presence"} {
		t.Run(name, func(t *testing.T) {
			actions := original
			actions.EffectCasters = slices.Clone(original.EffectCasters)
			if name == "caster" {
				actions.EffectCasters[0].Caster = actions.EffectCasters[0].Target
			} else {
				actions.EffectCasters[0].HasCaster, actions.EffectCasters[0].Caster = false, 0
			}
			if err := ms.World.RestoreActions(actions, nil); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := ms.World.RestoreActions(original, nil); err != nil {
					t.Fatal(err)
				}
			}()
			if len(nativeActorModeCheck(t, raw, ms, ms.World.Entities())) == 0 {
				t.Fatal("accepted lost caster association")
			}
		})
	}
}
