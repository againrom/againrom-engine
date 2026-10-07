package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"
)

func TestCurrentProfileKeepsItsTerminalEquipmentPolicy(t *testing.T) {
	for _, basis := range []CurrentProfileBasis{ProfileNative, ProfileOriginalCurrent, ProfileNativeRetired} {
		for _, fail := range []bool{false, true} {
			w := bookWorld(t, Entity{ID: 1, HP: 50, MaxHP: 100}, ItemInstance{Code: 0xe01, WeightPresent: true})
			s := SourceActor{Class: 2, Fighter: true, Stats: [14]uint16{10, 20, 10, 10, 18, 28, 28, 101, 50, 100, 100, 0, 0, 50}, EquipmentRuntimePresent: true}
			if err := w.RestoreActorLoad(1, ActorLoadSnapshot{Inventory: ActorLoad{Present: true, ContainerPresent: true, OwnWeight: 28, Source: s}, Load: 28, Capacity: 101, Speed: 18, Movement: HumanMovement{Present: true, RawSpeed: 18, NativeSpeed: 18, Load: 28, Capacity: 101}}); err != nil {
				t.Fatal(err)
			}
			e := &w.entities[0]
			e.HP, e.Decay, e.Defence, e.CurrentProfileBasis = -19, DecayFallen, 7, basis
			w.equipment[0][0] = ItemInstance{Code: 0x101, WeightPresent: true, Weight: 10, SourceEquipment: SourceEquipment{Class: SourceWeapon, DefinitionRow: 1, OwnKind: 1, Definition: SourceWeaponDefinition{Present: true, Suitable: 1}}}
			w.equipment[0][6] = ItemInstance{Code: 0x701, WeightPresent: true, Weight: 18, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
			w.BindSourceDerive(func(s SourceActor, acc int32, _ Rules) (SourceActor, error) {
				if fail && s.Stats[5] == 0 && acc == 0 {
					return s, fmt.Errorf("late current derive failed")
				}
				s.Stats[6] = s.Stats[5] + uint16(acc/2)
				binary.LittleEndian.PutUint16(s.Defence[:], 9)
				return s, nil
			})
			before := w.Hash()
			ok := w.dropTerminalLoot(0)
			if fail && basis == ProfileNative {
				if ok || w.Hash() != before {
					t.Fatal("late native completion committed a partial death")
				}
				continue
			}
			wantDefence, wantLoad := int32(9), int32(5)
			if basis == ProfileNative {
				wantDefence, wantLoad = 4, 0
			}
			if !ok || w.entities[0].Defence != wantDefence || w.entities[0].Load != wantLoad {
				t.Fatalf("current profile %d changed terminal policy: ok=%v defence=%d load=%d", basis, ok, w.entities[0].Defence, w.entities[0].Load)
			}
		}
	}
}

func TestCurrentGroupIdentityOrderAndModeRestoreAtomically(t *testing.T) {
	w := savedGroupWorld(t)
	w.savedGroups.Groups[0].Owner = SavedGroupReference{Class: 1, Owner: 1, Key: 777, Archive: 2}
	before, _, _ := w.SavedGroups()
	if err := w.RestoreGroupContinuations([]GroupContinuation{{Group: 72, ID: 90}, {Group: 71, ID: 80, Authored: true}}, 100); err != nil {
		t.Fatal(err)
	}
	g, _, _ := w.SavedGroups()
	if g[0].ID != 90 || g[1].ID != 80 || !g[1].Authored || g[1].Owner.Key != 0 || w.GroupHighWater() != 100 || !reflect.DeepEqual(g[1].Members, before[0].Members) || g[1].AI != before[0].AI {
		t.Fatal("current Group order, identity, membership or dispatch mode lost")
	}
	baseline := w.Hash()
	for _, rows := range [][]GroupContinuation{{{Group: 90, ID: 90}}, {{Group: 90, ID: 3}, {Group: 90, ID: 4}}, {{Group: 90, ID: 3}, {Group: 80, ID: 3}}, {{Group: 90, ID: 3, Authored: true}, {Group: 80, ID: 4}}} {
		if err := w.RestoreGroupContinuations(rows, 101); err == nil || baseline != w.Hash() {
			t.Fatal("malformed current Group continuation changed the World", err)
		}
	}
}

func TestCurrentCellCostKeepsDetachBaseline(t *testing.T) {
	w, p := cellPlanesFixture(t, false)
	if err := w.ImportOriginalStructures(nil, nil, []SavedStructureCell{{Cell: 0x100f, BaselineCost: 9}}, bytes.Clone(w.grid)); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	if err := w.RestoreCurrentCellCosts([]CurrentCellCost{{Cell: 0x100f, Cost: 0}}); err != nil {
		t.Fatal(err)
	}
	if w.savedCellPlanes.Cost[0x100f] != 0 || w.motionCell(0x100f).Payload[0] != 9 {
		t.Fatal("current zero cost overwrote the detach baseline")
	}
	before := w.Hash()
	for _, rows := range [][]CurrentCellCost{{{Cell: 0x100f, Cost: 6}, {Cell: 0x100f, Cost: 7}}, {{Cell: 0x100f, Cost: 6}, {Cell: 0xffff, Cost: 8}}} {
		if err := w.RestoreCurrentCellCosts(rows); err == nil || w.Hash() != before {
			t.Fatal("bad cell binding partially applied")
		}
	}
	Step(w, nil)
	if w.savedCellPlanes.Cost[0x100f] != 9 || w.motionCell(0x100f) != nil {
		t.Fatal("detach did not restore the independent original baseline")
	}
}

func TestCurrentActorProfileAndEffectCasterComponents(t *testing.T) {
	w := scrollFixtureWorld(t, 12, 1)
	e := &w.entities[0]
	e.Class, e.TypeID = 12, 33
	e.CurrentProfileBasis, e.RotationSpeed, e.SpellFX, e.SpellFXSpell = ProfileNative, 7, 4, 3
	w.attached = []attachedEffect{{Target: e.ID, Caster: 2, HasCaster: true, Spell: 13, Kind: EffectSpeed, Mode: EffectDuration, Magnitude: 4, Remaining: 17}}
	a := actionCopy(t, w.Actions())
	e.CurrentProfileBasis, e.RotationSpeed, e.SpellFX, e.SpellFXSpell = ProfileOriginalCurrent, 31, 0, 0
	e.Class = 33
	w.attached[0].Caster, w.attached[0].HasCaster = 0, false
	if err := w.RestoreActions(a, nil); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.Actions(), a) {
		t.Fatal("current profile or effect caster reset to import defaults")
	}
	if w.entities[0].Class != 12 || w.entities[0].TypeID != 33 {
		t.Fatal("drawable class replaced the wire type")
	}
	before := w.Hash()
	a.Actors[0].Current.Class = nil
	if err := w.RestoreActions(a, nil); err != nil || w.Hash() != before {
		t.Fatal("legacy current fields reset the drawable class", err)
	}
	for _, invalid := range []string{"basis", "effect"} {
		v := actionCopy(t, a)
		if invalid == "basis" {
			*v.Actors[0].ProfileBasis = 255
		} else {
			v.EffectCasters[0].Spell = 14
		}
		if err := w.RestoreActions(v, nil); err == nil || w.Hash() != before {
			t.Fatal("invalid current component partially applied", invalid, err)
		}
	}
	// Old M6 records lack these additions and keep their imported current basis.
	a.Actors[0].ProfileBasis, a.Actors[0].Current, a.EffectCasters = nil, nil, nil
	if err := w.RestoreActions(a, nil); err != nil || w.Hash() != before {
		t.Fatal("absent supplemental fields reset current state", err)
	}
}

func TestCurrentSavedPointCasterAttributionAndLegacyAbsence(t *testing.T) {
	for _, present := range []bool{false, true} {
		w := transportGraphWorld(t, 1)
		g := w.SavedSpellGraph()
		g.Nodes[1].Value.SE41 = 1
		if present {
			g.Nodes[1].HasCaster, g.Nodes[1].Caster = true, w.entities[0].ID
		}
		if err := w.ImportSavedSpellGraph(g); err != nil {
			t.Fatal(err)
		}
		cold := requireSpellGraphBinary(t, w)
		for range 2 {
			w.stepSavedWorldEffects(nil)
			cold.stepSavedWorldEffects(nil)
		}
		e := cold.entities[1]
		if w.Hash() != cold.Hash() || e.HP != 991 || e.HasKillCredit != present || present && (e.KillCreditSource != w.entities[0].ID || e.KillCreditSpell != 1) {
			t.Fatal("point payload lost current caster or invented original attribution", present, e)
		}
	}
}
