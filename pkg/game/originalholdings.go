package game

import (
	"fmt"
	"reflect"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// originalActorLoad restores Load from the decoded record, not from
// CurrentLoad(OwnWeight, Accumulator). SAV-794 (High) shows the original's own
// load path reads +0x8e/+0x90 back and keeps +0x90 unchanged, even on a
// record where no recompute from the stored container could have produced it
// (own weight 178, stored load 181, empty container, in five preserved
// resaves of the same actor) — so this importer must round-trip the file's
// Load rather than treat it as a cache of OwnWeight it may refresh at import.
func originalActorLoad(source sav.ActorLoadState, nativeSpeed int32) *sim.ActorLoadSnapshot {
	if !source.Present {
		return nil
	}
	return &sim.ActorLoadSnapshot{
		Inventory: sim.ActorLoad{Present: true, OwnWeight: source.OwnWeight,
			ContainerPresent: source.ContainerPresent, InsertIndex: source.InsertIndex, Accumulator: source.Accumulator},
		Load: int32(source.Load), Capacity: int32(source.Capacity), Speed: nativeSpeed,
		Movement:         sim.HumanMovement{Present: true, RawSpeed: source.Speed, NativeSpeed: nativeSpeed, Load: int32(source.Load), Capacity: int32(source.Capacity)},
		HealthHundredths: source.HealthHundredths, ManaHundredths: source.ManaHundredths,
	}
}

func originalActorBasis(b *sav.ActorBasis) sim.SourceActor {
	if b == nil {
		return sim.SourceActor{}
	}
	h := cityHumanState(sav.CityCharacter{Stats: b.Stats, SkillXP: b.SkillXP, Experience: b.Experience}, b.Human)
	class := uint8(2)
	// SourceActor class2 is the existing native humanoid numeric policy.
	// SourceBinding separately retains exact SAV Human versus Humanoid.
	if b.Class == "Unit" {
		class = 1
	}
	s := mapload.HumanSourceActor(h, class)
	repairSourceSkills(&s)
	s.Reach, s.AttackCharge, s.AttackRelax, s.EquipmentRuntimePresent = b.Reach, b.AttackCharge, b.AttackRelax, true
	return s
}

// restoreOriginalActorStock joins unique, nonzero map IDs across the entire
// source and target populations before eligibility filtering. A dead duplicate
// cannot turn an ambiguous identity into a first-wins match. No cell/def fallback.
func restoreOriginalActorStock(ms *Mission, source []sav.ActorHoldings, table *mapload.Table, report *OriginalSaveResume) error {
	if ms == nil || ms.Map == nil || ms.World == nil {
		return fmt.Errorf("original holdings: mission has no map/world")
	}
	counts := make(map[uint16]int)
	for _, actor := range source {
		if actor.MapUnitID != 0 {
			counts[actor.MapUnitID]++
		}
	}
	targets := make(map[uint16][]sim.Entity)
	for _, e := range ms.World.Entities() {
		if e.MapUnitID != 0 {
			targets[e.MapUnitID] = append(targets[e.MapUnitID], e)
		}
	}
	party := make(map[sim.EntityID]bool)
	for _, id := range ms.Start.IDs {
		party[id] = true
	}
	r := *report
	var batch []sim.OriginalActorStock
	var codes []uint16
	var values uint64
	for _, actor := range source {
		var target sim.Entity
		if ms.actorRegistry != nil {
			bound, err := registryTarget(ms, actor.Off)
			if err != nil {
				return err
			}
			if bound == nil {
				r.StockDead++
				continue
			}
			target = *bound
		} else {
			id := actor.MapUnitID
			if id != 0 && (counts[id] != 1 || len(targets[id]) > 1) {
				return fmt.Errorf("original holdings: ambiguous MapUnitID %d at %d", id, actor.Off)
			}
			if id != 0 && originalPartyCarriesMapUnit(ms.Party, id) {
				r.StockParty++
				continue
			}
			if actor.Stage != 0 || int16(actor.HP) <= 0 {
				r.StockDead++
				continue
			}
			if int(actor.Cell&255) >= int(ms.Map.Width) || int(actor.Cell>>8) >= int(ms.Map.Height) {
				r.StockOffMap++
				continue
			}
			if id == 0 {
				r.StockUnbound++
				continue
			}
			if len(targets[id]) == 0 {
				r.StockUnmatched++
				continue
			}
			target = targets[id][0]
			if party[target.ID] {
				r.StockParty++
				continue
			}
			if !target.Alive() {
				r.StockDead++
				continue
			}
			if target.OffMap {
				r.StockOffMap++
				continue
			}
		}
		stock := sim.OriginalActorStock{ID: target.ID, LoadState: originalActorLoad(actor.LoadState, target.Speed)}
		if stock.LoadState != nil {
			stock.LoadState.Inventory.Source = originalActorBasis(actor.Basis)
			if binding, ok := ms.actorRegistry.actor(actor.Off); ok {
				stock.LoadState.Inventory.Source = originalActorBasis(binding.Source.Character.Basis)
			}
			if target.SourceBinding.ActorClass() == 2 {
				stock.LoadState.Inventory.Source.TypeID = reconciledRoodTypeID(stock.LoadState.Inventory.Source.TypeID, target.SourceBinding.ClassFlags, ms.Number, target.MapUnitID)
			}
			if stock.LoadState.Inventory.Source.Class == 1 {
				restoreUnitLoad(stock.LoadState, &stock.LoadState.Inventory.Source)
			}
		}
		add := func(p sav.Piece, slot int) error {
			if p.Code == 0 || p.Stack == 0 {
				if slot < 0 && reflect.DeepEqual(p, sav.Piece{}) {
					stock.Carried = append(stock.Carried, sim.ItemStack{})
					return nil
				}
				return fmt.Errorf("zero item code or count")
			}
			values += uint64(p.Stack) * (1 + uint64(len(p.Effects)))
			if values > sim.MaxOriginalHoldingValues {
				return fmt.Errorf("more than %d expanded item/effect values", sim.MaxOriginalHoldingValues)
			}
			r.UnsupportedItemEffects += len(p.UnsupportedEffectStates)
			item := originalItemInstance(p, nil, table)
			if slot >= 0 {
				codeSlot, ok := data.EquipSlotFor(data.ItemCode(p.Code))
				if !ok || codeSlot != slot+1 || p.Stack != 1 || !stock.Equipped[slot].Empty() {
					return fmt.Errorf("unrepresentable equipment slot %d code %#x count %d", slot+1, p.Code, p.Stack)
				}
				stock.Equipped[slot] = item
			} else {
				stock.Carried = append(stock.Carried, sim.StackItem(item, uint32(p.Stack)))
			}
			codes = append(codes, p.Code)
			return nil
		}
		for slot, piece := range []*sav.Piece{actor.HeldWeapon, actor.HeldShield} {
			if piece != nil {
				if (slot == 0 && piece.Class != "Weapon") || (slot == 1 && piece.Class != "Shield") {
					return fmt.Errorf("original holdings actor %d: wrong held item class %s", actor.Off, piece.Class)
				}
				if err := add(*piece, slot); err != nil {
					return fmt.Errorf("original holdings actor %d: %w", actor.Off, err)
				}
			}
		}
		for slot, piece := range actor.Worn {
			if piece != nil {
				if slot < 2 {
					return fmt.Errorf("original holdings actor %d: armor slot %d overlaps native held role", actor.Off, slot+1)
				}
				if piece.Class != "Armor" {
					return fmt.Errorf("original holdings actor %d: wrong armor class %s", actor.Off, piece.Class)
				}
				if err := add(*piece, slot); err != nil {
					return fmt.Errorf("original holdings actor %d: %w", actor.Off, err)
				}
			}
		}
		for _, piece := range actor.Items {
			if err := add(piece, -1); err != nil {
				return fmt.Errorf("original holdings actor %d: %w", actor.Off, err)
			}
		}
		// Reject the constructor's shield/two-hand normalization rather than
		// silently moving an original equipped object into the pack.
		equipment := stock.Equipped
		var displaced []sim.ItemInstance
		mapload.NormalizeShieldLoadout(&equipment, &displaced, table)
		if len(displaced) != 0 {
			return fmt.Errorf("original holdings actor %d: unrepresentable shield/weapon combination", actor.Off)
		}
		batch = append(batch, stock)
	}
	// Rearm can refuse a derived sheet. Stage the entire handoff in a detached
	// native world so a late refusal changes neither prior stock nor weights.
	encoded, err := ms.World.MarshalBinary()
	if err != nil {
		return err
	}
	var staged sim.World
	mapload.BindSourceDerive(&staged)
	staged.SetRules(ms.World.Rules())
	staged.CopyDiaryUnits(ms.World)
	// STORIES 1130..1135 primed this staging round trip's fresh receiver by
	// hand for every carried-not-wire-form field (two raw session spans, cell-
	// record residue, the SpellEffect graph, the Projectiles store, Diaries):
	// UnmarshalBinary's own composite literal used to carry each one across a
	// decode from the RECEIVER's prior value, so a fresh receiver like staged
	// always decoded to zero regardless of what ms.World already held.
	if err := staged.UnmarshalBinary(encoded); err != nil {
		return err
	}
	if err := staged.ImportOriginalActorStock(batch); err != nil {
		return err
	}
	mapload.DeclareCodeWeights(&staged, table, codes)
	for _, stock := range batch {
		if member, ok := ms.Start.Roster[stock.ID]; ok {
			// Empty saved weapon means unarmed, never fallback starter gear.
			if _, ok := Rearm(&staged, stock.ID, member.Hero, member.Profile, member.Weapon, true, table,
				mapload.RotationSpeedBase(member.Hired(), member.HiredRotationSpeed, member.Class, table)); !ok {
				return fmt.Errorf("original holdings actor %d: cannot rearm saved equipment", stock.ID)
			}
		}
	}
	for _, stock := range batch {
		if stock.LoadState != nil {
			if err := staged.RestoreActorLoad(stock.ID, *stock.LoadState); err != nil {
				return err
			}
		}
	}
	for i, id := range ms.Start.IDs {
		if ms.actorRegistry == nil && i < len(ms.Party) && ms.Party[i].Carry != nil && ms.Party[i].Carry.LiveLoad != nil {
			if err := staged.RestoreActorLoad(id, *ms.Party[i].Carry.LiveLoad); err != nil {
				return err
			}
		}
	}
	*ms.World = staged
	r.Stocked = len(batch)
	*report = r
	return nil
}

// restoreUnitLoad gives a loaded Unit a capacity when its record holds zero,
// and an own weight when its record holds zero against a load the load law
// (ITEM-LOAD-005) does not reach from its container alone. An engine file
// written with those zeros then loads as the state it saves. An original Unit
// record holds both and is unchanged.
func restoreUnitLoad(load *sim.ActorLoadSnapshot, basis *sim.SourceActor) {
	carried := load.Inventory.CurrentLoad()
	held := load.Inventory.OwnWeight != 0 || load.Load == carried
	capacity, _ := unitLoadWords(uint16(load.Capacity), 0, 0, true)
	load.Capacity = int32(int16(capacity))
	if !held && load.Load > carried {
		load.Inventory.OwnWeight = int16(load.Load - carried)
	}
	if load.Movement.Present {
		load.Movement.Capacity = load.Capacity
	}
	basis.Stats[sav.StatOwnWeight], basis.Stats[sav.StatCapacity] = uint16(load.Inventory.OwnWeight), capacity
}
