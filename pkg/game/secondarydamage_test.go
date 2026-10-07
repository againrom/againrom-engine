package game

import (
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// secondaryBowCollection is one legal fighter Short Bow row. It implements
// the same Collection and MaskTable interfaces the production shop pool and
// equipment resolver consume, so the test below does not hand either path a
// precomposed candidate or a precomputed combat block.
type secondaryBowCollection struct{ attackType int32 }

func (secondaryBowCollection) Len() int { return 2 }
func (secondaryBowCollection) EntryName(i int) string {
	if i == 1 {
		return "Short Bow"
	}
	return ""
}
func (secondaryBowCollection) EntryStrings(int) []string { return nil }
func (c secondaryBowCollection) EntryParams(i int) []int32 {
	if i != 1 {
		return nil
	}
	p := make([]int32, data.SutableForColumn+1)
	for j := range p {
		p[j] = -1
	}
	p[2], p[3] = 100, 1 // price and weight
	attackType := c.attackType
	if attackType == 0 {
		attackType = 10
	}
	p[5], p[6], p[7] = attackType, 10, 20
	p[8], p[9] = 50, 0
	p[11], p[12], p[13] = 1, 0, 50
	p[data.SutableForColumn] = 1 // fighter, not mage
	return p
}

// TestRearmFoldsFlameWeaponDamageAndLiveGeneral is the live equipment route
// for DIV-363. It starts from the same equipped item code Rearm resolves in
// production, first derives a General-0 control and then the trained bearer;
// only General moves ToHit, while the weapon's byte pair appears as Fire and
// never enters the physical pair.
func TestRearmFoldsFlameWeaponDamageAndLiveGeneral(t *testing.T) {
	table := secondaryBowTable(44)
	table.Weapons = secondaryBowCollection{attackType: 11}

	const id = sim.EntityID(7)
	var equipped [sim.EquipSlots]sim.ItemInstance
	equipped[0] = sim.ItemInstance{Code: 0x0101, Kind: 2}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{{ID: id, X: 1, Y: 1, HP: 100, MaxHP: 100}}, nil,
		sim.Relations{}, nil, []sim.Stock{{ID: id, EquippedItems: equipped}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}

	hero := data.Hero{Body: 15, Reaction: 15, Mind: 15, Spirit: 15}
	weapon, wrote := Rearm(w, id, hero, data.Profile{Fighter: true}, nil, true, table, 0)
	if !wrote || weapon == nil || weapon.AttackType != 11 {
		t.Fatalf("General-0 Rearm = weapon %+v wrote=%v, want attack type 11", weapon, wrote)
	}
	control := gaEntity(t, w, id)

	hero.Skill[data.SkillGeneral] = 17
	weapon, wrote = Rearm(w, id, hero, data.Profile{Fighter: true}, nil, true, table, 0)
	if !wrote || weapon == nil || weapon.ToHit != 50 {
		t.Fatalf("General-17 Rearm = weapon %+v wrote=%v, want resolved weapon ToHit 50 as a non-source witness", weapon, wrote)
	}
	got := gaEntity(t, w, id)

	if got.ToHit != control.ToHit+17 {
		t.Errorf("ToHit = %d, want control %d + General 17; weapon ToHit 50 must not be added", got.ToHit, control.ToHit)
	}
	if got.DamageBase != control.DamageBase || got.DamageSpread != control.DamageSpread || got.Defence != control.Defence {
		t.Errorf("ranged Rearm moved physical/defence fields: got %+v, control %+v", got, control)
	}
	if want := (sim.SecondaryDamage{Base: 10, Spread: 10, Selector: 0}); got.SecondaryDamage != want {
		t.Errorf("SecondaryDamage = %+v, want weapon Fire component %+v", got.SecondaryDamage, want)
	}
	if got.XPSlot != 0 || got.Reach != 1 {
		t.Errorf("XPSlot/Reach = %d/%d, want General/1", got.XPSlot, got.Reach)
	}

	// Weapon::Equip walks this effect after writing its own Fire component.
	// The effect therefore replaces the whole triple rather than adding to it.
	worn, _ := w.EquippedItems(id)
	worn[0].Effects = []sim.ItemEffect{{Kind: 45, Operand: 5 | 7<<8}}
	if !w.ReplaceStock(sim.Stock{ID: id, EquippedItems: worn}) {
		t.Fatal("ReplaceStock refused the same actor's enchanted ranged weapon")
	}
	if _, wrote := Rearm(w, id, hero, data.Profile{Fighter: true}, nil, true, table, 0); !wrote {
		t.Fatal("Rearm refused the enchanted ranged weapon")
	}
	if want := (sim.SecondaryDamage{Base: 5, Spread: 7, Selector: 1}); gaEntity(t, w, id).SecondaryDamage != want {
		t.Errorf("enchanted ranged SecondaryDamage = %+v, want later Water effect %+v",
			gaEntity(t, w, id).SecondaryDamage, want)
	}
}
func (secondaryBowCollection) EntryRaw(i int) []byte {
	if i != 1 {
		return nil
	}
	raw := make([]byte, 2*data.ShopShapes)
	binary.LittleEndian.PutUint16(raw, 1) // shape 0 admits material 0
	return raw
}

// secondaryBowScale leaves every resolved weapon and price factor at one. Its
// one stated capacity factor makes the shop's range-effect payload ceiling 100.
type secondaryBowScale struct {
	count  int
	magCap float64
}

func (s secondaryBowScale) Len() int           { return s.count }
func (secondaryBowScale) EntryName(int) string { return "" }
func (s secondaryBowScale) EntryDoubles(int) []float64 {
	d := make([]float64, 9)
	for i := range d {
		d[i] = 1
	}
	d[8] = s.magCap
	return d
}

func secondaryBowTable(kind int) *mapload.Table {
	return &mapload.Table{
		Shapes:    secondaryBowScale{count: data.ShopShapes, magCap: 100},
		Materials: secondaryBowScale{count: data.ShopMaterials, magCap: 1},
		Weapons:   secondaryBowCollection{},
		Armors:    shopCollection{},
		Magic: enchantMagic{kind: {
			weight: 1, cost: 1, minimum: 1, maximum: 100,
		}},
	}
}

// TestGeneratedFighterBowSecondaryDamageSurvivesAMissForEveryProtection is the
// complete production seam behind the pass-2 counterexample. The shop pool
// admits a fighter-slot-1 Short Bow, fixed draws install one of each kind
// 44..48, Rearm derives a legal Body-15 fighter with an empty physical pair,
// and sim.Step makes seed 17's non-auto hit roll miss defence 1000. The selected
// protection is 50 while all four rivals are 100, so only the right selector
// can leave HP 90. Restoring the old unconditional miss return leaves HP 100 in
// every case.
func TestGeneratedFighterBowSecondaryDamageSurvivesAMissForEveryProtection(t *testing.T) {
	for selector := 0; selector < 5; selector++ {
		kind := 44 + selector
		t.Run([]string{"Fire", "Water", "Air", "Earth", "Astral"}[selector], func(t *testing.T) {
			table := secondaryBowTable(kind)
			pool := shopWeaponPool(table, 1_000_000)
			if len(pool) != 1 {
				t.Fatalf("fighter Short Bow pool = %+v, want one candidate", pool)
			}
			candidate := pool[0]
			if candidate.Code != 0x0101 || candidate.ItemKind != 2 || candidate.EffectSlot != 1 ||
				!candidate.Fighter || candidate.ForcedCast || candidate.MagCap != 100 {
				t.Fatalf("Short Bow candidate = %+v, want legal fighter weapon code 0x0101 with capacity 100", candidate)
			}
			if slot, ok := EquipTarget(candidate.Code, table); !ok || slot != 1 {
				t.Fatalf("EquipTarget(%#04x) = %d,%v, want fighter weapon slot 1", candidate.Code, slot, ok)
			}

			draws := &fixedDraws{values: []int{0, 19, 19, 0, 100, 100}}
			generated, ok := shopEnchantedItem(candidate, 1_000_000, table, draws)
			wantEffect := sim.ItemEffect{Kind: uint8(kind), Operand: 20 | 1<<8}
			if !ok || generated.Code != candidate.Code || generated.Kind != 2 || generated.Count != 1 ||
				!reflect.DeepEqual(generated.Effects, []sim.ItemEffect{wantEffect}) {
				t.Fatalf("generated Short Bow = %+v,%v, want one exact effect %+v", generated, ok, wantEffect)
			}
			if wantCalls := []int{1, 100, 100, 50, 101, 101}; !reflect.DeepEqual(draws.calls, wantCalls) || len(draws.values) != 0 {
				t.Fatalf("generator calls = %v remaining=%v, want %v and no remaining draws", draws.calls, draws.values, wantCalls)
			}

			var equipped [sim.EquipSlots]sim.ItemInstance
			equipped[0] = generated.Instance()
			protection := [5]int32{100, 100, 100, 100, 100}
			protection[selector] = 50
			const attackerID, targetID = sim.EntityID(1), sim.EntityID(2)
			w, err := sim.NewStockedWorld(17, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical,
				sim.Terrain{}, []sim.Entity{
					{ID: attackerID, X: 1, Y: 1, HP: 100, MaxHP: 100},
					{ID: targetID, X: 2, Y: 1, HP: 100, MaxHP: 100, Defence: 1000, Protection: protection},
				}, nil, sim.Relations{}, nil, []sim.Stock{{ID: attackerID, EquippedItems: equipped}})
			if err != nil {
				t.Fatalf("NewStockedWorld: %v", err)
			}
			hero := data.Hero{Body: 15, Reaction: 15, Mind: 15, Spirit: 15}
			weapon, wrote := Rearm(w, attackerID, hero, data.Profile{Fighter: true}, nil, true, table, 0)
			if !wrote || weapon == nil || !weapon.Ranged() || weapon.Name != "Short Bow" {
				t.Fatalf("Rearm = weapon %+v wrote=%v, want the generated ranged Short Bow", weapon, wrote)
			}
			attacker := gaEntity(t, w, attackerID)
			wantSecondary := sim.SecondaryDamage{Base: 20, Spread: 1, Selector: uint8(selector)}
			if attacker.DamageBase != 0 || attacker.DamageSpread != 0 || attacker.AlwaysHits ||
				attacker.SecondaryDamage != wantSecondary {
				t.Fatalf("Body-15 ranged combat = physical %d+U[0,%d] always=%v secondary=%+v, want empty/false/%+v",
					attacker.DamageBase, attacker.DamageSpread, attacker.AlwaysHits, attacker.SecondaryDamage, wantSecondary)
			}

			sim.Step(w, []sim.Command{{Kind: sim.KindAttack, Entity: attackerID, X: int32(targetID)}})
			if got := gaEntity(t, w, targetID).HP; got != 90 {
				t.Fatalf("secondary-only ranged miss left HP %d, want 90 through selector %d's 50%% protection", got, selector)
			}
		})
	}
}
