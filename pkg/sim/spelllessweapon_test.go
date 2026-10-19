package sim

import "testing"

// spelllessRule is a damaging Prismatic Spray stand-in: spell 14, mana 80.
func spelllessRule() SpellRule {
	r := wpnRule(6, 6, 7)
	r.ID, r.ManaCost = 14, 80
	return r
}

// spelllessStaff is a classed Staff whose first castSpell Effect names spell
// 14 at power 20; held selects whether it holds the Spell an equip builds.
func spelllessStaff(held bool) ItemInstance {
	item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
	item.Effects = staffEffects(14, 20)
	if held {
		item.SourceEquipment.Spell = SourceItemSpell{Present: true, ID: 14, Range: 7, ManaCost: 80}
	}
	return item
}

// spelllessWorld puts a party mage of reach 5 wearing staff at (2, 8) and a
// hostile victim at (6, 8), four cells away, with a plain Club in the pack.
func spelllessWorld(t *testing.T, staff ItemInstance, second bool) *World {
	t.Helper()
	caster := wpnCaster(1, 2, 8, 0, 0, 2, 1)
	caster.Reach, caster.Owner = 5, SelfSlot
	caster.TypeID = HeroTypeID(true, false)
	victim := spEnt(2, 6, 8)
	victim.Owner = 2
	rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, relationHostile})
	club := sourceEquipmentWeapon(0x0103, 3, 4, 5, 1, 1)
	w, err := NewStockedSpelledWorld(5, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{caster, victim}, nil, rel, nil,
		[]Stock{{ID: 1, ItemInstances: []ItemInstance{club}, EquippedItems: [EquipSlots]ItemInstance{0: staff}}}, []SpellRule{spelllessRule()})
	if err != nil {
		t.Fatal(err)
	}
	if second {
		w.rom2 = &rom2ScriptState{}
	}
	return w
}

type spelllessAttack struct {
	casting, charging bool
	mana              int32
	hurt              bool
}

// spelllessAttackRun orders the mage to attack and runs 300 steps.
func spelllessAttackRun(w *World) spelllessAttack {
	var got spelllessAttack
	cmds := []Command{cbOrder(1, 2)}
	for range 300 {
		Step(w, cmds)
		cmds = nil
		a := w.entities[indexOfEntity(w.entities, 1)]
		got.casting = got.casting || a.AttackPhase == AttackCasting
		got.charging = got.charging || a.AttackPhase == AttackCharging
	}
	got.mana = w.entities[indexOfEntity(w.entities, 1)].Mana
	got.hurt = w.entities[indexOfEntity(w.entities, 2)].HP < 100
	return got
}

// TestASpelllessSpellWeaponMageAttacksWithoutACast: a first-game mage whose
// worn Staff holds no Spell attacks without a cast: the wind-up is a
// physical one, no mana is spent and no spell reaches the victim, whose
// fixture blow does no damage (MAGIC-379, MAGIC-AUTOCAST-020). Controls: the
// same Staff holding its Spell casts and hurts; in the second game the
// Spell-less Staff casts as before.
func TestASpelllessSpellWeaponMageAttacksWithoutACast(t *testing.T) {
	got := spelllessAttackRun(spelllessWorld(t, spelllessStaff(false), false))
	if got.casting || !got.charging || got.mana != 100 || got.hurt {
		t.Fatalf("Spell-less Staff: %+v, want a physical wind-up, no spell on the victim and no mana spent", got)
	}
	held := spelllessAttackRun(spelllessWorld(t, spelllessStaff(true), false))
	if !held.casting || held.charging || !held.hurt {
		t.Fatalf("control, Staff holding its Spell: %+v, want a cast that hurts", held)
	}
	second := spelllessAttackRun(spelllessWorld(t, spelllessStaff(false), true))
	if !second.casting || !second.hurt {
		t.Fatalf("control, second game: %+v, want a cast that hurts", second)
	}
}

// TestASpelllessSpellWeaponWornAgainCasts: the Staff taken off holds no Spell
// in the pack; worn again it holds a new Spell built from its Effect and the
// mage's attack is the cast (ITEM-187). Control: before the re-equip the
// same mage does not cast.
func TestASpelllessSpellWeaponWornAgainCasts(t *testing.T) {
	w := spelllessWorld(t, spelllessStaff(false), false)
	w.unequip(0, 1)
	stacks := w.carried[0]
	staff := stacks[len(stacks)-1]
	if staff.Code != 0x0102 || staff.SourceEquipment.Spell.Present {
		t.Fatalf("the pack holds %+v after the removal", staff)
	}
	w.equip(0, len(stacks)-1, 1, 0)
	worn := w.equipment[0][0]
	if worn.Code != 0x0102 || worn.SourceEquipment.Spell != (SourceItemSpell{Present: true, ID: 14, Range: 7, ManaCost: 80}) {
		t.Fatalf("worn again the Staff holds %+v", worn.SourceEquipment.Spell)
	}
	if got := spelllessAttackRun(w); !got.casting || !got.hurt {
		t.Fatalf("worn again: %+v, want a cast that hurts", got)
	}
	if control := spelllessAttackRun(spelllessWorld(t, spelllessStaff(false), false)); control.casting {
		t.Fatal("control: the Spell-less Staff cast before the re-equip")
	}
}

// TestAnyOwnerEquipBuildsTheWeaponSpell: a computer mage's equip builds the
// Spell as a party mage's does (ITEM-187 reads no owner).
func TestAnyOwnerEquipBuildsTheWeaponSpell(t *testing.T) {
	w := spelllessWorld(t, ItemInstance{}, false)
	w.entities[0].Owner = 2
	w.carried[0] = append(w.carried[0], StackItem(spelllessStaff(false), 1))
	w.recomputeLoad(0)
	w.equip(0, 1, 1, 0)
	if got := w.equipment[0][0]; got.Code != 0x0102 || !got.SourceEquipment.Spell.Present {
		t.Fatalf("a computer mage's equip left %+v", got)
	}
}

// spelllessRemoval is one removal route of a Staff holding its Spell.
type spelllessRemoval struct {
	name  string
	saved bool
	act   func(w *World)
}

func spelllessRemovals() []spelllessRemoval {
	unequip := func(w *World) { w.unequip(0, 1) }
	swap := func(w *World) { w.equip(0, 0, 1, 0) }
	drop := func(w *World) { w.dropFromEquipment(0, 1, 3, 8) }
	return []spelllessRemoval{
		{"plain unequip", false, unequip},
		{"plain swap", false, swap},
		{"plain drop", false, drop},
		{"saved unequip", true, unequip},
		{"saved swap", true, swap},
		{"saved drop", true, drop},
	}
}

// removedStaff finds the Staff after a removal: in the pack or on the ground.
func removedStaff(t *testing.T, w *World) ItemInstance {
	t.Helper()
	for _, st := range w.carried[0] {
		if st.Code == 0x0102 {
			return st.Instance()
		}
	}
	for _, s := range w.Sacks() {
		for _, item := range s.ItemInstances {
			if item.Code == 0x0102 {
				return item
			}
		}
	}
	t.Fatal("the Staff left the actor and is nowhere")
	return ItemInstance{}
}

// TestEveryRemovalDeletesTheWeaponSpell: on each route an actor without
// source operands has, the Staff taken off holds no Spell, its saved row
// holds none, and the native continuation keeps that (ITEM-187). Control: the
// worn Staff held the Spell before the removal; the second game keeps it.
func TestEveryRemovalDeletesTheWeaponSpell(t *testing.T) {
	for _, route := range spelllessRemovals() {
		for _, second := range []bool{false, true} {
			w := spelllessWorld(t, spelllessStaff(true), second)
			if route.saved {
				w = spelllessWorld(t, ItemInstance{}, second)
				w.carried[0] = append(w.carried[0], StackItem(spelllessStaff(true), 1))
				w.recomputeLoad(0)
				bindOperationsPack(t, w, 0)
				w.equip(0, 1, 1, 0)
				if w.equipment[0][0].ObjectID == 0 {
					t.Fatalf("%s: the native route did not wear the saved Staff", route.name)
				}
			}
			if !w.equipment[0][0].SourceEquipment.Spell.Present {
				t.Fatalf("%s: control: the worn Staff holds no Spell", route.name)
			}
			route.act(w)
			if w.equipment[0][0].Code == 0x0102 {
				t.Fatalf("%s: the Staff is still worn", route.name)
			}
			staff := removedStaff(t, w)
			if staff.SourceEquipment.Spell.Present != second {
				t.Fatalf("%s, second game %v: the removed Staff holds Spell %+v", route.name, second, staff.SourceEquipment.Spell)
			}
			if route.saved {
				if row := w.savedObjects.item(staff.ObjectID); row == nil || (row.Spell != 0) != second {
					t.Fatalf("%s, second game %v: the saved row is %+v", route.name, second, row)
				}
			}
			if second {
				continue
			}
			cold := reloadOperations(t, w)
			if again := removedStaff(t, cold); again.SourceEquipment.Spell.Present != second {
				t.Fatalf("%s, second game %v: the continuation holds Spell %+v", route.name, second, again.SourceEquipment.Spell)
			}
		}
	}
}
