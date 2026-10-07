package sim

import (
	"reflect"
	"slices"
	"testing"
)

// TestMissionReturnRaisesARestorableBandBody: PARTY-ENDCULL-026 keeps every
// band actor of the participant without a health test and resets its pools
// from their maxima. A band actor the K key's command felled to -1, and one the
// L key's damage command downed at 0, rise through the Heal transition: full
// pools, no corpse stage, the defence the fall halved doubled back, and the
// experience, pack and worn set they fell with. A band actor the same damage
// took to -10 inside its dying window (PARTY-CULL-004), a fallen creature and
// another slot's fallen band actor are not raised.
func TestMissionReturnRaisesARestorableBandBody(t *testing.T) {
	ents := []Entity{
		{ID: 1, X: 1, Y: 1, HP: 60, MaxHP: 100, Mana: 5, MaxMana: 40, Defence: 20, DyingTime: 8, Owner: SelfSlot, TypeID: HumanTypeID},
		{ID: 2, X: 3, Y: 1, HP: 70, MaxHP: 90, Mana: 3, MaxMana: 30, Defence: 53, DyingTime: 8, Owner: SelfSlot, TypeID: HumanTypeID},
		{ID: 3, X: 5, Y: 1, HP: 50, MaxHP: 80, Mana: 0, MaxMana: 20, Defence: 30, DyingTime: 8, Owner: SelfSlot, TypeID: HumanTypeID},
		{ID: 4, X: 7, Y: 1, HP: 40, MaxHP: 40, Defence: 10, DyingTime: 8, Owner: SelfSlot, TypeID: HumanTypeID},
		{ID: 5, X: 1, Y: 5, HP: 40, MaxHP: 40, Defence: 10, DyingTime: 8, Owner: SelfSlot, TypeID: PersistHigh},
		{ID: 6, X: 3, Y: 5, HP: 40, MaxHP: 40, Defence: 10, DyingTime: 8, Owner: SelfSlot + 1, TypeID: HumanTypeID},
	}
	ents[1].SkillXP[2] = 4321
	stock := []Stock{{ID: 2, Items: []uint16{0x101, 0x102}, Equipped: [EquipSlots]uint16{0x201, 0, 0x203}}}
	w := mustStockedWorld(t, 1, ents, stock)
	Step(w, []Command{Kill(2), Damage(3, 50), Damage(4, 50), Kill(5), Kill(6)})
	for id, hp := range map[EntityID]int32{2: -1, 3: 0, 4: -10, 5: -1, 6: -1} {
		if e := cmdEntity(t, w, id); e.HP != hp || e.Decay != DecayFallen {
			t.Fatalf("actor %d at health %d stage %d, want %d fallen", id, e.HP, e.Decay, hp)
		}
	}
	if e := cmdEntity(t, w, 2); e.Defence != 26 || !e.Restorable() {
		t.Fatalf("K left actor 2 at defence %d restorable=%v, want 26 and restorable", e.Defence, e.Restorable())
	}
	carried, _ := w.Carried(2)
	worn, _ := w.Equipped(2)
	left := map[EntityID]Entity{}
	for _, id := range []EntityID{4, 5, 6} {
		left[id] = cmdEntity(t, w, id)
	}

	if err := w.NormalizeMissionSurvivors(SelfSlot); err != nil {
		t.Fatal(err)
	}
	if got := w.BoundarySurvivors(SelfSlot); !reflect.DeepEqual(got, []EntityID{1, 2, 3}) {
		t.Fatalf("the boundary keeps %v, want [1 2 3]", got)
	}
	for id, defence := range map[EntityID]int32{1: 20, 2: 52, 3: 30} {
		e := cmdEntity(t, w, id)
		if !e.Alive() || e.HP != e.MaxHP || e.Mana != e.MaxMana || e.Decay != DecayNone || e.Dwell != 0 || e.Defence != defence {
			t.Fatalf("actor %d health %d/%d mana %d/%d stage %d dwell %d defence %d, want full pools, no stage and defence %d",
				id, e.HP, e.MaxHP, e.Mana, e.MaxMana, e.Decay, e.Dwell, e.Defence, defence)
		}
	}
	if e := cmdEntity(t, w, 2); e.SkillXP[2] != 4321 {
		t.Fatalf("raised actor 2 has experience %v, want 4321 in slot 2", e.SkillXP)
	}
	if got, _ := w.Carried(2); !slices.Equal(got, carried) || len(got) != 2 {
		t.Fatalf("raised actor 2 carries %v, want %v", got, carried)
	}
	if got, _ := w.Equipped(2); got != worn || got[0] != 0x201 || got[2] != 0x203 {
		t.Fatalf("raised actor 2 wears %v, want %v", got, worn)
	}
	for id, before := range left {
		if got := cmdEntity(t, w, id); !reflect.DeepEqual(got, before) {
			t.Fatalf("actor %d changed at the boundary: before=%+v after=%+v", id, before, got)
		}
	}
}
