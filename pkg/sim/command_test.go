package sim

import "testing"

// constructorCase is one constructor beside the command literal it replaced.
// The want side is written out field by field on purpose: it is the frozen
// record of what the tree wrote before there were constructors, so a
// constructor that quietly moves a value into another field, widens a
// conversion or drops one fails here rather than in a corpus comparison weeks
// later.
type constructorCase struct {
	name string
	got  Command
	want Command
}

func constructorCases() []constructorCase {
	// The widest handle there is, held in variables so the conversion the arms
	// undo is the runtime one and not a constant the compiler would refuse.
	var wideEntity EntityID = 4294967295
	var wideStructure StructureID = 4294967295
	var pickup Command
	pickup.Kind, pickup.Entity, pickup.X, pickup.Y = KindPickUp, 7, -3, 9
	return []constructorCase{
		{"PickUp", PickUp(7, CellPoint{X: -3, Y: 9}), pickup},
		{"MoveTo", MoveTo(7, CellPoint{X: -3, Y: 9}),
			Command{Kind: KindMoveTo, Entity: 7, X: -3, Y: 9}},
		{"Kill", Kill(7), Command{Kind: KindKill, Entity: 7}},
		{"TerminalKill", TerminalKill(7), Command{Kind: KindTerminalKill, Entity: 7}},
		{"Damage", Damage(7, 25), Command{Kind: KindDamage, Entity: 7, X: 25}},
		{"Attack", Attack(7, wideEntity),
			Command{Kind: KindAttack, Entity: 7, X: int32(uint32(wideEntity))}},
		{"AttackStructure", AttackStructure(7, wideStructure),
			Command{Kind: KindAttackStructure, Entity: 7, X: int32(uint32(wideStructure))}},
		{"Cast", Cast(7, 9, 24), Command{Kind: KindCast, Entity: 7, X: 9, Y: 24}},
		{"CastAt", CastAt(7, 19, CellPoint{X: 40, Y: 41}),
			Command{Kind: KindCastAt, Entity: 7, X: 40, Y: 41, Spell: 19}},
		{"Autocast", Autocast(7, 3), Command{Kind: KindAutocast, Entity: 7, X: 3}},
		{"Autocast clearing", Autocast(7, 0), Command{Kind: KindAutocast, Entity: 7, X: 0}},
		{"Equip", Equip(7, 5, 12), Command{Kind: KindEquip, Entity: 7, X: 5, Y: 12}},
		{"EquipDisplacing", EquipDisplacing(7, 5, 12, 1),
			Command{Kind: KindEquip, Entity: 7, X: 5, Y: 12, Spell: 1}},
		{"EquipDisplacing naming no second slot", EquipDisplacing(7, 5, 12, 0),
			Command{Kind: KindEquip, Entity: 7, X: 5, Y: 12}},
		{"Unequip", Unequip(7, 1), Command{Kind: KindUnequip, Entity: 7, X: 1}},
		{"DropCarried", DropCarried(7, 6, CellPoint{X: 30, Y: 31}),
			Command{Kind: KindDropCarried, Entity: 7, X: 30, Y: 31, Spell: 6}},
		{"DropWorn", DropWorn(7, 1, CellPoint{X: 30, Y: 31}),
			Command{Kind: KindDropWorn, Entity: 7, X: 30, Y: 31, Spell: 1}},
		{"ReadBook", ReadBook(7, 2), Command{Kind: KindReadBook, Entity: 7, X: 2}},
		{"UsePotion", UsePotion(7, 2), Command{Kind: KindUsePotion, Entity: 7, X: 2}},
		{"UseScroll", UseScroll(7, 2, 9),
			Command{Kind: KindUseScroll, Entity: 7, X: 9, Spell: 2}},
		{"UseScrollAt", UseScrollAt(7, 2, CellPoint{X: 30, Y: 31}),
			Command{Kind: KindUseScrollAt, Entity: 7, X: 30, Y: 31, Spell: 2}},
		{"UseStructure", UseStructure(7, wideStructure),
			Command{Kind: KindUseStructure, Entity: 7, X: int32(uint32(wideStructure))}},
		{"GroupMoveTo", GroupMoveTo(7, CellPoint{X: 12, Y: 13}, 3),
			Command{Kind: KindGroupMoveTo, Entity: 7, X: 12, Y: 13, Group: 3}},
		{"GroupSwarmTo", GroupSwarmTo(7, CellPoint{X: 12, Y: 13}, 3),
			Command{Kind: KindGroupSwarmTo, Entity: 7, X: 12, Y: 13, Group: 3}},
		{"GroupPatrolTo", GroupPatrolTo(7, CellPoint{X: 12, Y: 13}, 3),
			Command{Kind: KindGroupPatrolTo, Entity: 7, X: 12, Y: 13, Group: 3}},
		{"GroupStance guarding", GroupStance(7, OrderGuard, 3),
			Command{Kind: KindGroupStance, Entity: 7, X: OrderGuard, Group: 3}},
		{"GroupStance standing ground", GroupStance(7, OrderStandGround, 3),
			Command{Kind: KindGroupStance, Entity: 7, X: OrderStandGround, Group: 3}},
		{"GroupDefend", GroupDefend(7, 9, 3),
			Command{Kind: KindGroupDefend, Entity: 7, X: 9, Group: 3}},
		{"GroupRetreat", GroupRetreat(7, SelfSlot, 3),
			Command{Kind: KindGroupRetreat, Entity: 7, Player: SelfSlot, Group: 3}},
		{"SetPlayerParameter", SetPlayerParameter(SelfSlot, PlayerParameterRetreat, RetreatModeHigh),
			Command{Kind: KindPlayerParameter, Player: SelfSlot, X: int32(PlayerParameterRetreat), Y: RetreatModeHigh}},
	}
}

// TestConstructorsBuildTheLiteralTheyReplaced compares every constructor
// against the written-out command for the same arguments.
func TestConstructorsBuildTheLiteralTheyReplaced(t *testing.T) {
	for _, c := range constructorCases() {
		if c.got != c.want {
			t.Errorf("%s = %+v, want %+v", c.name, c.got, c.want)
		}
	}
}

// TestEveryCommandKindHasAConstructor holds the set of kinds the constructors
// produce against the set this package defines. A new kind with no constructor
// fails here, because the only way to issue it would be a literal and the tree
// guard refuses those.
func TestEveryCommandKindHasAConstructor(t *testing.T) {
	built := map[uint8]bool{}
	for _, c := range constructorCases() {
		built[c.got.Kind] = true
	}
	kinds := []uint8{
		KindMoveTo, KindKill, KindDamage, KindGroupMoveTo, KindAttack,
		KindEquip, KindCast, KindGroupStance, KindGroupPatrolTo, KindGroupSwarmTo,
		KindUnequip, KindAutocast, KindCastAt, KindDropCarried, KindDropWorn,
		KindReadBook, KindTerminalKill, KindAttackStructure, KindGroupDefend,
		KindGroupRetreat, KindUsePotion, KindUseScroll, KindUseScrollAt,
		KindUseStructure, KindPickUp, KindPlayerParameter,
	}
	for _, k := range kinds {
		if !built[k] {
			t.Errorf("kind %d has no constructor case", k)
		}
	}
	if len(built) != len(kinds) {
		t.Errorf("constructors produce %d distinct kinds, this test names %d", len(built), len(kinds))
	}
}

// TestNarrowingConversionsAreTheArmsOwn sweeps the values where a constructor
// could differ from the literal it replaced: the fields that are narrower than
// the argument. A spell id rides in a uint16 for one kind and an int32 for two
// others, and a slot index rides in both, so each is checked against the
// conversion the tree wrote by hand.
func TestNarrowingConversionsAreTheArmsOwn(t *testing.T) {
	values := []int32{-2147483648, -65537, -65536, -1, 0, 1, 2, 12, 32767, 32768, 65535, 65536, 65537, 2147483647}
	for _, v := range values {
		if got, want := CastAt(1, SpellID(v), CellPoint{}), (Command{Kind: KindCastAt, Entity: 1, Spell: uint16(v)}); got != want {
			t.Errorf("CastAt spell %d = %+v, want %+v", v, got, want)
		}
		if got, want := Cast(1, 2, SpellID(v)), (Command{Kind: KindCast, Entity: 1, X: 2, Y: v}); got != want {
			t.Errorf("Cast spell %d = %+v, want %+v", v, got, want)
		}
		if got, want := Autocast(1, SpellID(v)), (Command{Kind: KindAutocast, Entity: 1, X: v}); got != want {
			t.Errorf("Autocast spell %d = %+v, want %+v", v, got, want)
		}
		if got, want := Equip(1, ItemSlot(v), EquipSlot(v)), (Command{Kind: KindEquip, Entity: 1, X: v, Y: v}); got != want {
			t.Errorf("Equip slots %d = %+v, want %+v", v, got, want)
		}
		if got, want := DropCarried(1, ItemSlot(v), CellPoint{}), (Command{Kind: KindDropCarried, Entity: 1, Spell: uint16(v)}); got != want {
			t.Errorf("DropCarried item %d = %+v, want %+v", v, got, want)
		}
		if got, want := DropWorn(1, EquipSlot(v), CellPoint{}), (Command{Kind: KindDropWorn, Entity: 1, Spell: uint16(v)}); got != want {
			t.Errorf("DropWorn slot %d = %+v, want %+v", v, got, want)
		}
		if got, want := UseScroll(1, ItemSlot(v), 2), (Command{Kind: KindUseScroll, Entity: 1, X: 2, Spell: uint16(v)}); got != want {
			t.Errorf("UseScroll item %d = %+v, want %+v", v, got, want)
		}
		if got, want := SetPlayerParameter(1, PlayerParameter(v), v), (Command{Kind: KindPlayerParameter, Player: 1, X: v, Y: v}); got != want {
			t.Errorf("SetPlayerParameter %d = %+v, want %+v", v, got, want)
		}
	}
}

// TestEntityHandlesCrossWholeInX sweeps the id conversions. An entity or
// structure handle rides in X as the same 32 bits under another name, and the
// arms convert it back with EntityID(uint32(c.X)), so every value has to
// survive the round trip including the half that reads as negative.
func TestEntityHandlesCrossWholeInX(t *testing.T) {
	for _, id := range []uint32{0, 1, 2147483647, 2147483648, 4294967294, 4294967295} {
		if got := EntityID(uint32(Attack(1, EntityID(id)).X)); got != EntityID(id) {
			t.Errorf("Attack victim %d returned %d", id, got)
		}
		if got := StructureID(uint32(AttackStructure(1, StructureID(id)).X)); got != StructureID(id) {
			t.Errorf("AttackStructure target %d returned %d", id, got)
		}
		if got := StructureID(uint32(UseStructure(1, StructureID(id)).X)); got != StructureID(id) {
			t.Errorf("UseStructure target %d returned %d", id, got)
		}
		if got := EntityID(uint32(Cast(1, EntityID(id), 3).X)); got != EntityID(id) {
			t.Errorf("Cast victim %d returned %d", id, got)
		}
		if got := EntityID(uint32(UseScroll(1, 0, EntityID(id)).X)); got != EntityID(id) {
			t.Errorf("UseScroll target %d returned %d", id, got)
		}
		if got := EntityID(uint32(GroupDefend(1, EntityID(id), 0).X)); got != EntityID(id) {
			t.Errorf("GroupDefend subject %d returned %d", id, got)
		}
	}
}
