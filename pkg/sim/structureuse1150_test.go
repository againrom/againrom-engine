package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

func useWorld1150(t *testing.T, kind uint16) *World {
	t.Helper()
	a, s := structureCombatActor(), structureCombatTarget()
	a.HP, a.MaxHP, a.Mana, a.MaxMana = 10, 150, 20, 180
	s.Kind, s.Field42, s.MaxHealth, s.UseAmount = kind, 3, 3, 100
	return structureCombatWorld(t, 1150, a, s)
}

func finishUse1150(t *testing.T, w *World) {
	t.Helper()
	for range 400 {
		cold := retreatRoundTrip1089(t, w)
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("cold continuation changed pending interaction")
		}
		if len(w.StructureUses()) == 0 {
			return
		}
	}
	t.Fatal("interaction did not finish")
}

func TestStructureUse1150FountainPoolsAndCharges(t *testing.T) {
	for _, tc := range []struct {
		kind     uint16
		hp, mana int32
	}{{15, 110, 20}, {16, 10, 120}} {
		w := useWorld1150(t, tc.kind)
		Step(w, []Command{{Kind: KindUseStructure, Entity: 0, X: 0}})
		finishUse1150(t, w)
		if e := w.entities[0]; e.HP != tc.hp || e.Mana != tc.mana || w.structures[0].Field42 != 2 {
			t.Fatalf("kind%d pools=%d/%d charge=%d", tc.kind, e.HP, e.Mana, w.structures[0].Field42)
		}
		w.entities[0].HP, w.entities[0].Mana = 150, 180
		Step(w, []Command{{Kind: KindUseStructure, Entity: 0, X: 0}})
		finishUse1150(t, w)
		if w.structures[0].Field42 != 1 || w.entities[0].HP != 150 || w.entities[0].Mana != 180 {
			t.Fatal("full pool must still consume exactly one charge")
		}
		for _, empty := range []uint16{0, 65535} {
			w.structures[0].Field42 = empty
			w.entities[0].HP, w.entities[0].Mana = 10, 20
			w.tick = 20
			Step(w, []Command{{Kind: KindUseStructure, Entity: 0, X: 0}})
			finishUse1150(t, w)
			if w.entities[0].HP != 10 || w.entities[0].Mana != 20 || w.structures[0].Field42 != empty {
				t.Fatal("empty/negative charge applied a potion")
			}
		}
	}
	w := useWorld1150(t, 16)
	w.entities[0].Mana, w.entities[0].MaxMana = 0, 0
	Step(w, []Command{{Kind: KindUseStructure, X: 0}})
	finishUse1150(t, w)
	if w.structures[0].Field42 != 2 || w.entities[0].Mana != 0 {
		t.Fatal("fighter mana use changed class or failed to spend")
	}
}

func TestStructureUse1150ApproachFacingAndCancellation(t *testing.T) {
	w := useWorld1150(t, 28)
	w.structures[0].Col, w.structures[0].Row = 12, 12
	w.structures[0].Field42, w.structures[0].MaxHealth = 0, 1
	Step(w, []Command{{Kind: KindUseStructure, X: 0}})
	if len(w.StructureUses()) != 1 || w.structures[0].Field42 != 0 {
		t.Fatal("distant click did not queue approach")
	}
	finishUse1150(t, w)
	if w.structures[0].Field42 != 1 || cellOf(w.entities[0]).chebyshevTo(cell{12, 12}) > 1 {
		t.Fatal("lever not used at reachable boundary")
	}
	for range 70 {
		Step(w, nil)
	}
	if w.structures[0].Field42 != 1 {
		t.Fatal("one click repeated toggle")
	}
	w.entities[0].X, w.entities[0].Y = 2, 3
	w.entities[0].clearTransit()
	Step(w, []Command{{Kind: KindUseStructure, X: 0}})
	Step(w, []Command{{Kind: KindUseStructure, X: 999}})
	if len(w.StructureUses()) != 1 {
		t.Fatal("invalid command canceled admitted use")
	}
	Step(w, []Command{{Kind: KindMoveTo, X: 1, Y: 1}})
	if len(w.StructureUses()) != 0 {
		t.Fatal("move did not cancel use")
	}
	Step(w, []Command{{Kind: KindUseStructure, X: 0}, {Kind: KindKill}})
	if len(w.StructureUses()) != 0 {
		t.Fatal("death retained use")
	}
	w = useWorld1150(t, 29)
	w.entities[0].Facing = 192
	w.entities[0].RotationSpeed = 10
	w.structures[0].Field42 = 65535
	Step(w, []Command{{Kind: KindUseStructure, X: 0}})
	if len(w.StructureUses()) != 1 || w.structures[0].Field42 != 65535 {
		t.Fatal("used before facing")
	}
	finishUse1150(t, w)
	if w.structures[0].Field42 != 0 {
		t.Fatal("nonzero lever did not toggle to zero")
	}
	Step(w, []Command{{Kind: KindUseStructure, X: 0}})
	finishUse1150(t, w)
	if w.structures[0].Field42 != 1 {
		t.Fatal("zero-health lever was not usable")
	}
}

func TestStructureUse1150RechargeAndRetiredActors(t *testing.T) {
	w := useWorld1150(t, 15)
	w.structures[0].Field42, w.tick = 0, 59
	Step(w, nil)
	if w.structures[0].Field42 != 0 {
		t.Fatal("early recharge")
	}
	Step(w, nil)
	if w.structures[0].Field42 != 1 {
		t.Fatal("counter60 did not recharge")
	}
	for range 240 {
		Step(w, nil)
	}
	if w.structures[0].Field42 != 3 {
		t.Fatal("recharge exceeded maximum")
	}
	w.structures[0].Col = 13
	Step(w, []Command{{Kind: KindUseStructure, X: 0}})
	w.entities[0].OffMap = true
	Step(w, nil)
	if len(w.StructureUses()) != 0 {
		t.Fatal("off-map use survived")
	}
	if _, err := w.MarshalBinary(); err != nil {
		t.Fatal(err)
	}
}

func TestStructureUse1150Footer(t *testing.T) {
	for _, pin := range []struct {
		raw  []byte
		hash uint64
	}{{pinBytes, 0xb8d397ac6da0068b}, {rtfBytes, 0x61ed2031275d899a}} {
		prior := bytes.Clone(pin.raw[:len(pin.raw)-entityIDFloorLen-spellDeliverySpanLen-16])
		prior[0] = 88
		if fnv1a(prior) != pin.hash {
			t.Fatal("frozen form88 hash changed")
		}
	}
	w := useWorld1150(t, 28)
	w.structures[0].Col = 13
	Step(w, []Command{{Kind: KindUseStructure, X: 0}})
	b := mustMarshal(t, w)
	span := int(binary.LittleEndian.Uint32(b[len(b)-entityIDFloorLen-spellDeliverySpanLen-16:]))
	if b[0] != formatVersion || span != 26 {
		t.Fatal("literal metadata and intent length", span)
	}
	start := len(b) - entityIDFloorLen - spellDeliverySpanLen - 16 - span
	for _, mutate := range []func([]byte){
		func(b []byte) {
			binary.LittleEndian.PutUint32(b[len(b)-entityIDFloorLen-spellDeliverySpanLen-16:], 0xffffffff)
		},
		func(b []byte) { b[start+8], b[start+9] = 0, 0 },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+10:], 0xffffffff) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+18:], 999) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+22:], 999) },
	} {
		bad, hash := bytes.Clone(b), w.Hash()
		mutate(bad)
		if w.UnmarshalBinary(bad) == nil || w.Hash() != hash {
			t.Fatal("corrupt footer adopted")
		}
	}
}

func TestStructureUse1150SourcePoolAndFailedCallback(t *testing.T) {
	w := sourceMutationWorld(t, PlainItem(0xe01))
	w.structures = []Structure{{ID: 0, Kind: 15, UseAmount: 100, Width: 1, Height: 1, Field42: 3, MaxHealth: 3}}
	w.applyStructureUse(0, 0)
	if w.entities[0].HP != 100 || w.entities[0].SourceNow().Stats[8] != 100 || w.structures[0].Field42 != 2 {
		t.Fatal("source-derived HP/charge mismatch")
	}
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, fmt.Errorf("fixture refusal") })
	w.entities[0].HP = 20
	hash := w.Hash()
	w.applyStructureUse(0, 0)
	if w.Hash() != hash {
		t.Fatal("failed source callback spent charge or changed actor")
	}
}

func TestStructureUse1150UnreachableRetires(t *testing.T) {
	w := useWorld1150(t, 28)
	w.structures[0].Col = 13
	w.grid = make([]byte, 256)
	for y := 0; y < 16; y++ {
		w.grid[y*16+8] = 1
	}
	Step(w, []Command{{Kind: KindUseStructure, X: 0}})
	if len(w.StructureUses()) != 0 || w.entities[0].HasTarget {
		t.Fatal("unreachable use kept a repeated route query")
	}
	for range 100 {
		Step(w, nil)
	}
	if w.structures[0].Field42 != 3 {
		t.Fatal("unreachable target toggled")
	}
}

func TestStructureUse1150LeverFeedsMissionCheck(t *testing.T) {
	s := mustScript(t, []ScriptCheck{structFieldNode(0, true), constCheck(caConstant, 0)},
		[]ScriptInstant{{Op: ScriptInstantWin}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(caRegister, caConstant, ScriptCmpEQ)}, Instants: acts(0), Once: true}})
	a, lever := structureCombatActor(), structureCombatTarget()
	lever.Kind, lever.Field42, lever.MaxHealth = 28, 1, 1
	w, err := NewStructuredWorld(1150, Bounds{16, 16}, ModeCanonical, Terrain{}, []Entity{a}, s, Relations{}, nil, nil, nil, GhostTemplate{}, []Structure{lever})
	if err != nil {
		t.Fatal(err)
	}
	Step(w, []Command{{Kind: KindUseStructure, X: 0}})
	for range 20 {
		Step(w, nil)
	}
	if won, _ := w.ScriptCounters(); won != 1 {
		t.Fatal("lever state did not reach mission check21")
	}
}

func TestStructureUse1150ManualOrdersAndSavedAI(t *testing.T) {
	for _, saved := range []bool{false, true} {
		newWorld := func() *World {
			w := manualCastWorld(t)
			w.structures = []Structure{{ID: 0, Kind: 28, Width: 1, Height: 1, Col: 12, Row: 12, Field42: 1, MaxHealth: 1}}
			w.entities[0].AutoSpell = 1
			w.entities[0].RotationSpeed = 10
			if saved {
				savedTacticalRegistry(t, w, true)
			}
			return w
		}
		for _, next := range []Command{{Kind: KindAttack, Entity: 1, X: 2}, {Kind: KindCastAt, Entity: 1, Spell: 26, X: 6, Y: 3}} {
			w := newWorld()
			Step(w, []Command{{Kind: KindUseStructure, Entity: 1, X: 0}})
			if len(w.StructureUses()) != 1 {
				t.Fatal("initial use refused", saved)
			}
			Step(w, []Command{next})
			if len(w.StructureUses()) != 0 {
				t.Fatal("manual order failed to cancel use", saved, next.Kind)
			}
		}
		w := newWorld()
		Step(w, []Command{{Kind: KindUseStructure, Entity: 1, X: 0}})
		for range 300 {
			if len(w.StructureUses()) == 0 {
				break
			}
			if len(w.bookCasts) != 0 || w.entities[0].HasAttackTarget {
				t.Fatal("AI displaced admitted use", saved)
			}
			cold := retreatRoundTrip1089(t, w)
			Step(w, nil)
			Step(cold, nil)
			if w.Hash() != cold.Hash() {
				t.Fatal("saved AI continuation changed", saved)
			}
		}
		if len(w.StructureUses()) != 0 || w.structures[0].Field42 != 0 {
			t.Fatal("AI interrupted interaction", saved)
		}
	}
}
