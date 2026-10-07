package mapload

import (
	"againrom/pkg/sim"
	"fmt"
)

// TrainSourceParty keeps a post-equipment source basis on the established
// Human school path. Unsupported class/slot/domain fails before purse commit.
func TrainSourceParty(p PartyMember, slot int) (PartyMember, int32, error) {
	if !HasSourceActor(p) || p.Carry.LiveLoad.Inventory.Source.Class != 2 {
		return p, 0, fmt.Errorf("no source Human")
	}
	s := p.Carry.LiveLoad
	h := SourceHumanState(s.Inventory.Source, s.Inventory.Accumulator)
	h.Weight, h.Load, h.Capacity = uint16(s.Inventory.OwnWeight), uint16(s.Load), uint16(s.Capacity)
	price, err := h.TrainingPrice(slot)
	if err != nil {
		return p, 0, err
	}
	n, err := h.Train(slot)
	if err != nil {
		return p, 0, err
	}
	out := clonePartyMember(p)
	applyHumanLoad(&out, n)
	out.Hero = n.Hero()
	for i, xp := range n.SkillXP {
		out.Carry.SkillXP[i] = int32(xp)
	}
	if out.Saved != nil {
		out.Saved.HP, out.Saved.MaxHP, out.Saved.Mana, out.Saved.MaxMana = int32(int16(n.Health)), int32(int16(n.HealthMax)), int32(int16(n.Mana)), int32(int16(n.ManaMax))
	}
	out.RetireOriginalHuman()
	return out, price, nil
}

// HasSourceActor distinguishes a retained basis from a native town fold.
func HasSourceActor(p PartyMember) bool {
	return p.Carry != nil && p.Carry.LiveLoad != nil && p.Carry.LiveLoad.Inventory.Source.Class != 0
}

// SourceTownEquipment uses an isolated, clockless world for the same item
// virtuals used in mission commands. index>=0 takes one actual stack unit;
// otherwise a nonempty item comes from the shop. An empty item removes slot.
// No caller state is changed on refusal, including the item source and purse.
func SourceTownEquipment(p PartyMember, table *Table, index, slot int, item sim.ItemInstance, toPack bool, operations ...sim.SourceEquipmentOperation) (PartyMember, sim.ItemInstance, bool) {
	if len(operations) > 1 || !HasSourceActor(p) || ValidatePartyLoad(p) != nil {
		return p, sim.ItemInstance{}, false
	}
	var operation sim.SourceEquipmentOperation
	if len(operations) == 1 {
		operation = operations[0]
	}
	exact := operation.Topology != nil || operation.Receipt != nil || len(operation.PackAliases) != 0
	var receipt sim.SourceEquipmentReceipt
	destination := operation.Receipt
	if destination != nil {
		operation.Receipt = &receipt
	}
	e := sim.Entity{ID: 1, HP: 1, MaxHP: 1, Book: p.Book, KnownSpells: p.KnownSpells}
	stock := sim.Stock{ID: 1, LoadState: p.Carry.LiveLoad, OrderedStacks: p.Carry.OrderedStacks,
		ItemInstances: MemberCarriedItems(p, table), EquippedItems: MemberItemEquipment(p, table)}
	if exact {
		stock.ItemInstances = cloneItemInstances(stock.ItemInstances)
		stock.OrderedStacks = make([]sim.ItemStack, len(p.Carry.OrderedStacks))
		for i, stack := range p.Carry.OrderedStacks {
			stock.OrderedStacks[i] = stack.Clone()
			stock.OrderedStacks[i].ObjectID = 0
		}
		for i := range stock.ItemInstances {
			stock.ItemInstances[i].ObjectID = 0
		}
		for i := range stock.EquippedItems {
			stock.EquippedItems[i] = stock.EquippedItems[i].Clone()
			stock.EquippedItems[i].ObjectID = 0
		}
		item = item.Clone()
		item.ObjectID = 0
	}
	w, err := sim.NewStockedSpelledWorld(0, sim.Bounds{Width: 1, Height: 1}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{e}, nil, sim.Relations{}, nil, []sim.Stock{stock}, SpellRules(table))
	if err != nil {
		return p, sim.ItemInstance{}, false
	}
	BindSourceDerive(w)
	var codes []uint16
	for _, held := range stock.ItemInstances {
		codes = append(codes, held.Code)
	}
	for _, worn := range stock.EquippedItems {
		codes = append(codes, worn.Code)
	}
	codes = append(codes, item.Code)
	DeclareCodeWeights(w, table, codes)
	var removed sim.ItemInstance
	var ok bool
	switch {
	case index >= 0:
		ok = w.EquipSourceCarried(1, index, operation)
	case !item.Empty():
		ok = w.EquipSourceItem(1, item, operation)
	default:
		removed, ok = w.UnequipSource(1, slot, toPack, operation)
	}
	if !ok {
		return p, sim.ItemInstance{}, false
	}
	out := clonePartyMember(p)
	a := w.Entities()[0]
	out.Carry.LiveLoad = a.CurrentActorLoad()
	out.Carry.OrderedStacks, _ = w.CarriedStacks(1)
	out.Carry.ItemInstances, _ = w.CarriedItems(1)
	out.Carry.Items, _ = w.Carried(1)
	out.Carry.EquippedItems, _ = w.EquippedItems(1)
	out.Carry.Equipped, _ = w.Equipped(1)
	out.Carry.SkillXP = a.SkillXP
	out.Hero = SourceHumanState(a.SourceNow(), a.ActorLoad.Accumulator).Hero()
	out.Book, out.KnownSpells = a.Book, a.KnownSpells
	if out.Saved != nil {
		out.Saved.HP, out.Saved.MaxHP, out.Saved.Mana, out.Saved.MaxMana = a.HP, a.MaxHP, a.Mana, a.MaxMana
	}
	out.RetireOriginalHuman()
	if destination != nil {
		*destination = receipt
	}
	return out, removed, true
}
