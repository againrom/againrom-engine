package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func cheatWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 12, Height: 12}, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, Owner: SelfSlot, X: 1, Y: 1, HP: 40, MaxHP: 80, Book: Spellbook{State: BookPresent}},
			{ID: 2, Owner: 2, X: 8, Y: 8, HP: 40, MaxHP: 80}}, nil, Relations{},
		[]Sack{{X: 5, Y: 4, Gold: 7, Items: []uint16{0xe01}}, {X: 10, Y: 10, Gold: 9, Items: []uint16{0xe02}}}, nil,
		[]SpellRule{{ID: 1, MaxRange: 7, ManaCost: 12, Defensive: true}, {ID: 28, MaxRange: 8, ManaCost: 15}})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func cheatCold(t *testing.T, w *World) *World {
	t.Helper()
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if cold.Hash() != w.Hash() {
		t.Fatal("cold load changed canonical state")
	}
	return &cold
}

func TestCheatGodWritesWholeModifierAndUsesSourceDerive(t *testing.T) {
	w := cheatWorld(t)
	w.entities[0].ActorLoad = ActorLoad{Present: true, ContainerPresent: true, Source: SourceActor{Class: 2}}
	w.entities[0].Capacity = 11
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) {
		for j := 0; j < 6; j++ {
			binary.LittleEndian.PutUint16(s.Defence[4+2*j:], binary.LittleEndian.Uint16(s.Modifier[46+2*j:]))
			s.Defence[16+j] = s.Modifier[58+j]
		}
		return s, nil
	})
	if !w.CheatGod(1) {
		t.Fatal("source god command refused")
	}
	for j := 0; j < 6; j++ {
		s := w.entities[0].ActorLoad.Source
		if binary.LittleEndian.Uint16(s.Modifier[46+2*j:]) != 100 || s.Modifier[58+j] != 100 {
			t.Fatalf("modifier slot %d incomplete", j)
		}
	}
	if w.entities[0].Protection != [5]int32{100, 100, 100, 100, 100} || w.entities[0].Resistance != [5]byte{100, 100, 100, 100, 100} || w.entities[0].HP != 40 {
		t.Fatal("derived protection, resistance or current health differs")
	}
	cheatCold(t, w)
}

func TestCheatGodNativeBackingAndSpellInstancesSurviveColdLoad(t *testing.T) {
	w := cheatWorld(t)
	if !w.CheatGod(1) || !w.CheatSpell(1, 1) || !w.CheatSpell(1, 28) {
		t.Fatal("native god or spell command refused")
	}
	hash := w.Hash()
	if w.CheatSpell(1, 0) || w.CheatSpell(1, 2) || w.CheatSpell(1, 32) || w.CheatGod(999) || w.Hash() != hash {
		t.Fatal("invalid command changed state")
	}
	w.entities[1].Book.State = BookAbsent
	if w.CheatSpell(2, 1) || w.entities[1].KnownSpells != 0 {
		t.Fatal("a fighter without a spellbook learned a spell")
	}
	cold := cheatCold(t, w)
	if e := cold.entities[0]; e.KnownSpells != 1<<1|1<<28 || e.Book.Slots[0] != (BookSpell{Range: 7, Defensive: 1, ManaCost: 12}) || e.Book.Slots[27].ManaCost != 15 || e.NativeBasis.ModifierKnown>>46 != (1<<18)-1 {
		t.Fatalf("god or spell instance lost: %+v", e)
	}
}

func TestCheatKillGoldItemAndPickupUseOrdinaryState(t *testing.T) {
	w := cheatWorld(t)
	if w.CheatKillPlayer(2) != 1 || w.entities[1].HP != -50 || w.entities[0].HP != 40 {
		t.Fatal("kill-player did not write its exact owner's health")
	}
	if !w.CheatAddGold(SelfSlot, ^uint32(0)) || !w.CheatAddGold(SelfSlot, 2) || w.Purse(SelfSlot) != 1 || w.CheatAddGold(relationSlots, 1) {
		t.Fatal("gold width or owner guard differs")
	}
	if !w.CheatAddItem(1, StackItem(PlainItem(0xe03), 2)) || w.CheatAddItem(1, ItemStack{}) {
		t.Fatal("item insertion or empty item guard differs")
	}
	if err := w.CheatPickupAll(1); err != nil {
		t.Fatal(err)
	}
	carried, _ := w.Carried(1)
	if len(w.sacks) != 0 || w.Purse(SelfSlot) != 17 || !reflect.DeepEqual(carried, []uint16{0xe03, 0xe03, 0xe01, 0xe02}) {
		t.Fatalf("ordinary pickup transfer differs: purse %d, carry %x", w.Purse(SelfSlot), carried)
	}
	cheatCold(t, w)
}

func TestCheatItemRefusesExpandedValuesAndCumulativeMergeAtomically(t *testing.T) {
	w := cheatWorld(t)
	effects := make([]ItemEffect, 16)
	for i := range effects {
		effects[i] = ItemEffect{Kind: 15, Operand: 1}
	}
	stack := StackItem(ItemInstance{Code: 0xe03, Effects: effects}, 65535)
	hash := w.Hash()
	if allocations := testing.AllocsPerRun(10, func() {
		if w.CheatAddItem(1, stack) {
			t.Fatal("over-capacity effect expansion accepted")
		}
	}); allocations != 0 || w.Hash() != hash {
		t.Fatal("refused quantity allocated or changed state", allocations)
	}
	stack = StackItem(ItemInstance{Code: 0xe03, Effects: effects[:15]}, 65534)
	if !w.CheatAddItem(1, stack) {
		t.Fatal("quantity within the ordinary import bound refused")
	}
	hash = w.Hash()
	stack.Count = 1
	if w.CheatAddItem(1, stack) || w.Hash() != hash {
		t.Fatal("cumulative merge exceeded the ordinary import bound")
	}
	held, _ := w.CarriedStacks(1)
	worn, _ := w.EquippedItems(1)
	if err := w.ImportOriginalActorStock([]OriginalActorStock{{ID: 1, Carried: held, Equipped: worn}}); err != nil {
		t.Fatal("accepted holdings cannot use ordinary import", err)
	}
	cheatCold(t, w)
}

func TestCheatSummonRefusesEffectExpansionBeforeAllocation(t *testing.T) {
	w := cheatWorld(t)
	pack := []ItemStack{StackItem(ItemInstance{Code: 0xe03, Effects: make([]ItemEffect, 4096)}, 65535)}
	hash := w.Hash()
	if allocations := testing.AllocsPerRun(10, func() {
		if _, err := w.CheatSummon(Entity{HP: 20, MaxHP: 20, Reach: 1, TokenSize: 1}, pack, [EquipSlots]ItemInstance{}, 1, 1); err == nil {
			t.Fatal("summon accepted over-capacity effect expansion")
		}
	}); allocations > 2 || w.Hash() != hash {
		t.Fatal("refused summon expanded holdings or changed state", allocations)
	}
	var worn [EquipSlots]ItemInstance
	worn[0] = ItemInstance{Code: 0x101, Effects: make([]ItemEffect, MaxOriginalHoldingValues)}
	if _, err := w.CheatSummon(Entity{HP: 20, MaxHP: 20, Reach: 1, TokenSize: 1}, nil, worn, 1, 1); err == nil || w.Hash() != hash {
		t.Fatal("summon accepted equipment beyond ordinary holding capacity")
	}
}

func TestCheatPickupRefusesAggregateExpansionBeforeAnyTransfer(t *testing.T) {
	w := cheatWorld(t)
	effects := make([]ItemEffect, 15)
	for i := range effects {
		effects[i] = ItemEffect{Kind: 15, Operand: 1}
	}
	item := ItemInstance{Code: 0xe03, Effects: effects}
	w.carried[0] = []ItemStack{StackItem(item, 65534)}
	w.sacks = []Sack{makeSack(5, 4, 7, []ItemInstance{PlainItem(0xe01)}), makeSack(10, 10, 9, []ItemInstance{item})}
	hash := w.Hash()
	if allocations := testing.AllocsPerRun(10, func() {
		if err := w.CheatPickupAll(1); err == nil {
			t.Fatal("over-capacity pickup accepted")
		}
	}); allocations > 2 || w.Hash() != hash || w.Purse(SelfSlot) != 0 || len(w.sacks) != 2 {
		t.Fatal("refused pickup allocated inventory or transferred a sack", allocations)
	}
	w.carried[0][0].Count--
	if err := w.CheatPickupAll(1); err != nil || len(w.sacks) != 0 || w.Purse(SelfSlot) != 16 {
		t.Fatal("pickup within ordinary capacity failed", err)
	}
	cheatCold(t, w)
}

func TestCheatCursePreservesCurrentPoolsAndResetsSourceAttributes(t *testing.T) {
	w := cheatWorld(t)
	w.entities[0].ActorLoad = ActorLoad{Present: true, ContainerPresent: true, Source: SourceActor{Class: 2, Experience: 456, Stats: [14]uint16{30, 40, 50, 60}}}
	w.entities[0].Capacity = 11
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, nil })
	if !w.CheatCurse(1) {
		t.Fatal("curse refused")
	}
	s := w.entities[0].ActorLoad.Source
	if s.Experience != 0 || s.Stats[0] != 10 || s.Stats[1] != 10 || s.Stats[2] != 10 || s.Stats[3] != 1 || w.entities[0].HP != 40 {
		t.Fatalf("curse source state %+v", s)
	}
	cheatCold(t, w)
}

func TestSafeModeOverridesDispatchAndNativeContinuation(t *testing.T) {
	w := cheatWorld(t)
	g := SavedGroup{ID: 1, Members: []SavedGroupMember{{Entity: 2, Bound: true}}}
	g.AI[0x20] = 0xff
	if err := w.ImportSavedGroups([]SavedGroup{g}, nil); err != nil {
		t.Fatal(err)
	}
	w.entities[1].HasTarget = true
	w.entities[1].TargetX, w.entities[1].TargetY = 8, 9
	w.savedGroupPass(nil)
	if !w.entities[1].HasTarget {
		t.Fatal("ordinary dispatch ignored activity gate")
	}
	hash := w.Hash()
	w.SetSafeMode(true)
	if w.Hash() == hash || !cheatCold(t, w).safeMode {
		t.Fatal("runtime toggle absent from deterministic native state")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	form[29] &^= 0x80
	var old World
	if err := old.UnmarshalBinary(form); err != nil || old.safeMode || old.Hash() != hash {
		t.Fatal("ordinary old routing byte did not default safe mode off")
	}
	w.savedGroupPass(nil)
	if w.savedGroups.Groups[0].AI[0x45] != 0 {
		t.Fatal("dispatch override changed activity byte")
	}
	if w.entities[1].HasTarget {
		t.Fatal("idle group was not dispatched")
	}
}

func TestCheatSummonFreshIdentityNearestPlacementAndColdLoad(t *testing.T) {
	w := cheatWorld(t)
	template := Entity{ID: 9, X: 1, Y: 1, Owner: SelfSlot, HP: 20, MaxHP: 20, Reach: 1, TokenSize: 1,
		ActorLoad:     ActorLoad{Present: true, ContainerPresent: true, Source: SourceActor{Class: 1}},
		SourceBinding: SourceBinding{Class: GeneratedUnitBinding, Identity: 1, RuntimeID: 1, TokenRow: 4, TypeID: 5}}
	id, err := w.CheatSummon(template, nil, [EquipSlots]ItemInstance{}, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	e, found := w.Entity(id)
	if !found || id != 3 || e.X != 1 || e.Y != 0 || !e.SourceBinding.Generated() || e.SourceBinding.Identity == 1 || e.SourceBinding.RuntimeID == 1 || e.Owner != SelfSlot {
		t.Fatalf("summoned actor %+v", e)
	}
	cheatCold(t, w)
}

func TestCheatSummonJoinsSavedDispatchAndPreservesExactPlayer(t *testing.T) {
	w := cheatWorld(t)
	if !w.placeAt(1, 3, 1) {
		t.Fatal("hostile actor placement refused")
	}
	w.relations.Set(SelfSlot, 2, relationHostile)
	g := SavedGroup{ID: 1, Owner: SavedGroupReference{Key: 0x12345, Archive: 7, Class: 1, Owner: SelfSlot}, Members: []SavedGroupMember{{Entity: 1, Bound: true}}}
	g.AI[0x45] = 1
	if err := w.ImportSavedGroups([]SavedGroup{g}, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedGroupPlayers([]SavedGroupPlayer{{ID: 10, Slot: SelfSlot}, {ID: 20, Slot: SelfSlot}}, []SavedGroupContainer{{GroupID: 1, PlayerID: 20}}); err != nil {
		t.Fatal(err)
	}
	template := Entity{Owner: SelfSlot, HP: 20, MaxHP: 20, Reach: 1, TokenSize: 1, MaxMana: 100,
		ActorLoad: ActorLoad{Present: true, ContainerPresent: true, Source: SourceActor{Class: 1, ManaFloor: 100, ManaReservePercent: 95}}}
	id, err := w.CheatSummon(template, nil, [EquipSlots]ItemInstance{}, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	spawned := w.savedGroupFor(id)
	order := w.savedOrder(id)
	if spawned == nil || spawned.ContainerID != 20 || spawned.Owner != g.Owner || !spawned.Authored || order == nil || !order.Authored || order.State != uint32(actorStateGuard) {
		t.Fatalf("spawned dispatch provenance: group %+v, order %+v", spawned, order)
	}
	if e, _ := w.Entity(id); !e.ActorLoad.Source.HasOwner || e.ActorLoad.Source.ManaFloor != 95 {
		t.Fatalf("summoned owner reserve %+v", e.ActorLoad.Source)
	}
	w.tick = scriptPassPhase
	cold := cheatCold(t, w)
	Step(w, nil)
	Step(cold, nil)
	if e, _ := w.Entity(id); !e.HasAttackTarget || e.AttackTarget != 2 {
		t.Fatalf("summoned guard did not acquire hostile actor: %+v", e)
	}
	if w.Hash() != cold.Hash() {
		t.Fatal("summoned saved dispatch differs after cold load")
	}
	cheatCold(t, w)
}

func TestCheatSummonRefusesExhaustedSavedGroupsAtomically(t *testing.T) {
	w := cheatWorld(t)
	if err := w.ImportSavedGroups(nil, nil); err != nil {
		t.Fatal(err)
	}
	w.savedGroups.HighWater = ^uint32(0)
	hash := w.Hash()
	if _, err := w.CheatSummon(Entity{HP: 20, MaxHP: 20, Reach: 1, TokenSize: 1}, nil, [EquipSlots]ItemInstance{}, 1, 1); err == nil || w.Hash() != hash {
		t.Fatal("failed group admission changed world")
	}
}

func TestCheatSummonCreatesNativeGuardGroup(t *testing.T) {
	w := cheatWorld(t)
	w.relations.Set(SelfSlot, 2, relationHostile)
	if !w.placeAt(1, 3, 1) {
		t.Fatal("hostile actor placement refused")
	}
	before := w.Entities()
	template := Entity{Owner: SelfSlot, Group: 99, HP: 20, MaxHP: 20, Reach: 1, TokenSize: 1}
	id, err := w.CheatSummon(template, nil, [EquipSlots]ItemInstance{}, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	e, _ := w.Entity(id)
	if order, _, found := w.FrozenGroupAI(e.Owner, e.Group); !found || order != orderNone || e.Group != e.CommandGroup || e.CommandGroup == 0 || e.Group == template.Group {
		t.Fatal("summoned native guard has no independent actor dispatch group")
	}
	if !reflect.DeepEqual(w.Entities()[:len(before)], before) {
		t.Fatal("summon changed existing actors")
	}
	first := e
	second, err := w.CheatSummon(template, nil, [EquipSlots]ItemInstance{}, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	e, _ = w.Entity(second)
	if got, _ := w.Entity(id); !reflect.DeepEqual(got, first) || e.Group == first.Group || e.Group != e.CommandGroup {
		t.Fatal("later summon reused a group or changed the first actor")
	}
	w.tick = scriptPassPhase
	cold := cheatCold(t, w)
	Step(w, nil)
	Step(cold, nil)
	if e, _ := w.Entity(id); !e.HasAttackTarget || e.AttackTarget != 2 {
		t.Fatalf("summoned native guard did not acquire hostile actor: %+v", e)
	}
	if w.Hash() != cold.Hash() {
		t.Fatal("summoned native guard differs after cold load")
	}
}
