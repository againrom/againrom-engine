package sim

import "testing"

func TestAcquiredSavedItem1172TerminalMixedHoldings(t *testing.T) {
	for _, tc := range []struct {
		name     string
		load     bool
		suppress bool
		dwell    int32
		damage   bool
	}{
		{name: "immediate"},
		{name: "fall dwell", dwell: 2},
		{name: "retained load", load: true},
		{name: "damage and decay", dwell: 2, damage: true},
		{name: "damage with retained load", load: true, damage: true},
		{name: "suppressed", load: true, suppress: true, dwell: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := operationsPack1115(t)
			weapon := ItemInstance{Code: 0x0111, Kind: 2, Price: 17, WeightPresent: true, Weight: 1,
				Effects: []ItemEffect{{Kind: 41, Operand: 3<<16 | 1}, {Kind: 41, Operand: 3<<16 | 1}},
				SourceEquipment: SourceEquipment{Class: SourceWeapon, DefinitionRow: 1, OwnKind: 1,
					Spell: SourceItemSpell{Present: true, ID: 1, Range: 7, ManaCost: 3}}}
			shield := ItemInstance{Code: 0x0211, Kind: 1, Price: 19, WeightPresent: true, Weight: 3,
				SourceEquipment: SourceEquipment{Class: SourceShield, DefinitionRow: 1}}
			w.equipment[0][0], w.equipment[0][1] = weapon, shield
			syncWeaponItem(&w.entities[0], weapon)
			w.entities[0].DyingTime, w.entities[0].SuppressCorpseLoot = tc.dwell, tc.suppress
			w.recomputeLoad(0)
			if tc.load {
				w.entities[0].ActorLoad = ActorLoad{Present: true, OwnWeight: 4, ContainerPresent: true, InsertIndex: 1, Accumulator: 6}
				w.syncSavedPack(0)
			}
			w = reloadOperations1115(t, w)
			if tc.damage {
				Step(w, []Command{{Kind: KindDamage, Entity: 7, X: 30}})
				Step(w, nil)
			} else if err := w.HeadlessKill(7); err != nil {
				t.Fatal(err)
			} else {
				for n := int32(0); n < tc.dwell; n++ {
					Step(w, nil)
				}
			}
			e := w.entities[0]
			if e.WeaponSpell != 0 || e.WeaponSpellLevel != 0 || e.WeaponSpellSource != WeaponSpellNone ||
				e.Decay != DecayBones || e.Dwell != 0 || len(w.carried[0]) != 0 || e.Load != 0 {
				t.Fatalf("terminal actor: spell %d/%d/%d, decay %d, dwell %d, pack %d, load %d",
					e.WeaponSpell, e.WeaponSpellLevel, e.WeaponSpellSource, e.Decay, e.Dwell, len(w.carried[0]), e.Load)
			}
			for _, item := range w.equipment[0] {
				if !item.Empty() {
					t.Fatal("terminal actor retained equipment")
				}
			}
			if tc.load && (e.ActorLoad.OwnWeight != 0 || e.ActorLoad.Accumulator != 0 || e.ActorLoad.InsertIndex != 10000 || !e.ActorLoad.ContainerPresent) {
				t.Fatal("terminal actor lost fresh container bookkeeping", e.ActorLoad)
			}
			w = reloadOperations1115(t, w)
			before := w.Hash()
			if err := w.HeadlessKill(7); err != nil || w.Hash() != before {
				t.Fatal("terminal action replayed after native LOAD", err)
			}
			if tc.suppress {
				if len(w.sacks) != 0 {
					t.Fatal("suppressed holdings reached ground")
				}
				for _, row := range w.savedObjects.Items {
					if !row.Retired {
						t.Fatal("suppressed Item remained live", row.ID)
					}
				}
				return
			}
			if len(w.sacks) != 1 || len(w.sacks[0].ItemInstances) != 5 {
				t.Fatal("terminal drop lost mixed holdings", w.sacks)
			}
			sack := w.sacks[0]
			owner := SavedObjectOwner{Kind: SavedOwnerSack, Object: sack.ObjectID}
			c := w.savedObjects.container(owner)
			want := []uint16{0x0204, shield.Code, weapon.Code}
			if tc.load {
				want = []uint16{0x0204, weapon.Code, shield.Code}
			}
			if c == nil || len(c.Items) != len(want) || c.Items[0] != 10 {
				t.Fatal("terminal drop lost ordered bound identity", c)
			}
			wantItems := make([]ItemStack, len(want))
			for k, code := range want {
				row := w.savedObjects.item(c.Items[k])
				count := uint32(1)
				if k == 0 {
					count = 3
				}
				if row == nil || !w.savedObjects.HasLocation(row.ID, SavedItemLocation{Owner: owner, Index: uint32(k)}) || row.Value.Code != code || row.Value.Count != count {
					t.Fatal("terminal drop changed owner, count or order", row)
				}
				if code == weapon.Code && (len(row.Effects) != 2 || row.Effects[0] == row.Effects[1] || row.Spell == 0) {
					t.Fatal("terminal acquisition lost distinct Effect/Spell graph", row)
				}
				wantItems[k] = row.Value.Clone()
			}
			// The one-unit pickup loop retains its first split and merges the
			// other two units into it, retiring the original count-three Item.
			wantItems[0].ObjectID = w.savedObjects.NextID
			if err := w.TakeSack(8, sack.X, sack.Y); err != nil {
				t.Fatal("next pickup could not adopt terminal loot", err)
			}
			if len(w.sacks) != 0 || len(w.carried[1]) != len(wantItems) || !w.savedObjects.item(10).Retired {
				t.Fatal("next pickup lost split/merge retirement")
			}
			for k, item := range wantItems {
				if !StackStateEqual(w.carried[1][k], item) || !w.savedObjects.HasLocation(item.ObjectID, SavedItemLocation{Owner: w.savedPackOwner(1), Index: uint32(k)}) {
					t.Fatal("next pickup changed the retained item graph", k)
				}
			}
			_ = reloadOperations1115(t, w)
		})
	}
}

func TestAcquiredSavedItem1172SourceTerminalMixedHoldings(t *testing.T) {
	for _, suppress := range []bool{false, true} {
		w := sourceMutationWorld(t, ItemInstance{Code: 0x0e01, WeightPresent: true, Weight: 3})
		weapon := sourceEquipmentWeapon(0x0101, 5, 0, 0, 0, 1)
		w.equipment[0][0] = weapon
		e := &w.entities[0]
		e.ActorLoad.OwnWeight, e.ActorLoad.Accumulator, e.ActorLoad.InsertIndex = 5, 777, 1
		e.ActorLoad.Source.EquipmentRuntimePresent = true
		e.Reach, e.SuppressCorpseLoot, e.DyingTime = 4, suppress, 2
		bindOperationsPack1115(t, w, 0)
		w = reloadOperations1115(t, w)
		w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, nil })
		if err := w.HeadlessKill(1); err != nil {
			t.Fatal(err)
		}
		Step(w, nil)
		Step(w, nil)
		e = &w.entities[0]
		if e.Decay != DecayBones || len(w.carried[0]) != 0 || !w.equipment[0][0].Empty() ||
			e.ActorLoad.OwnWeight != 0 || e.ActorLoad.Accumulator != 0 || e.ActorLoad.InsertIndex != 10000 || !e.ActorLoad.ContainerPresent {
			t.Fatalf("source terminal actor (suppress %v): decay %d, pack %d, own weight %d, accumulator %d, index %d, present %v",
				suppress, e.Decay, len(w.carried[0]), e.ActorLoad.OwnWeight, e.ActorLoad.Accumulator, e.ActorLoad.InsertIndex, e.ActorLoad.ContainerPresent)
		}
		if (len(w.sacks) == 0) != suppress || (w.savedObjects.item(10).Retired) != suppress {
			t.Fatal("source terminal drop/disposal changed ownership", suppress)
		}
		_ = reloadOperations1115(t, w)
	}
}
