package mapload

import (
	"fmt"
	"reflect"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func partyItemWeight(item sim.ItemInstance, t *Table) int32 {
	if item.WeightPresent {
		return int32(item.Weight)
	}
	if item.Code == 0 || t == nil {
		return 0
	}
	v, ok, err := data.ItemCodeWeight(data.ItemCode(item.Code), t.Shapes, t.Materials, t.Armors, t.Shields, t.Weapons)
	if !ok || err != nil {
		return 0
	}
	return v
}

func partyWeightSums(p PartyMember, t *Table) (worn, carried int32) {
	for _, item := range MemberItemEquipment(p, t) {
		worn += partyItemWeight(item, t)
	}
	for _, item := range MemberCarriedItems(p, t) {
		carried += partyItemWeight(item, t)
	}
	return
}

// UpdatePartyLoad applies a completed city's item operation to its canonical
// load state. Source-to-tray staging uses refresh=false (SAV-CITYMOVE-512).
// A source-backed sale/training supplies its coupled state with ApplyOriginalHuman.
func UpdatePartyLoad(before PartyMember, p *PartyMember, t *Table, refresh, derive bool) error {
	return updatePartyLoad(before, p, t, refresh, derive, true)
}

func UpdatePartyLoadOrdered(before PartyMember, p *PartyMember, t *Table, refresh, derive bool, stacks []sim.ItemStack) error {
	next := clonePartyMember(*p)
	var items []sim.ItemInstance
	var codes []uint16
	ordered := make([]sim.ItemStack, 0, len(stacks))
	var count uint64
	for _, stack := range stacks {
		if sim.StackStateEqual(stack, sim.ItemStack{}) {
			ordered = append(ordered, sim.ItemStack{})
			continue
		}
		count += uint64(stack.Count)
		if stack.Count == 0 || stack.Code == 0 || count > sim.MaxOriginalHoldingValues {
			return fmt.Errorf("party: invalid explicit ordered container")
		}
		ordered = append(ordered, stack.Clone())
		for n := uint32(0); n < stack.Count; n++ {
			items = append(items, stack.Instance().Clone())
			codes = append(codes, stack.Code)
		}
	}
	if next.Carry == nil {
		next.CarriedItems, next.Carried = items, codes
	} else {
		next.Carry.ItemInstances, next.Carry.Items = items, codes
		next.Carry.OrderedStacks = ordered
	}
	if err := updatePartyLoad(before, &next, t, refresh, derive, false); err != nil {
		return err
	}
	*p = next
	return nil
}

func updatePartyLoad(before PartyMember, p *PartyMember, t *Table, refresh, derive, reconcile bool) error {
	if before.Carry == nil || before.Carry.LiveLoad == nil || p.Carry == nil {
		return nil
	}
	s := *before.Carry.LiveLoad
	if s.Inventory.Source.Class != 0 && derive {
		return fmt.Errorf("source equipment or skill mutation requires its ordered producer")
	}
	oldWorn, oldCarry := partyWeightSums(before, t)
	newWorn, newCarry := partyWeightSums(*p, t)
	s.Inventory.OwnWeight += int16(newWorn - oldWorn)
	s.Inventory.Accumulator += newCarry - oldCarry
	if s.Inventory.Source.Class == 2 && !derive {
		h := SourceHumanState(s.Inventory.Source, s.Inventory.Accumulator)
		h.Weight, h.Load, h.Capacity = uint16(s.Inventory.OwnWeight), uint16(s.Load), uint16(s.Capacity)
		if refresh {
			if _, err := h.Derive(); err != nil {
				return err
			}
			n, _, err := h.RefreshInventoryLoad()
			if err != nil {
				return err
			}
			h = n
		}
		s.Inventory.Source = humanSourceWithRuntime(h, s.Inventory.Source)
		s.Load, s.Capacity, s.Speed = int32(int16(h.Load)), int32(int16(h.Capacity)), int32(int16(h.Speed))
		s.Movement = sim.HumanMovement{Present: true, RawSpeed: int16(h.Speed), NativeSpeed: s.Speed, Load: s.Load, Capacity: s.Capacity}
		p.Carry.LiveLoad = &s
		p.Hero = h.Hero()
		if p.Saved != nil {
			p.Saved.HP, p.Saved.MaxHP, p.Saved.Mana, p.Saved.MaxMana = int32(int16(h.Health)), int32(int16(h.HealthMax)), int32(int16(h.Mana)), int32(int16(h.ManaMax))
		}
		if reconcile {
			reconcilePartyStacks(before, p, t)
		}
		return nil
	}
	if refresh {
		if s.Inventory.Present && s.Capacity == 0 {
			return fmt.Errorf("source load refresh has zero capacity")
		}
		s.Load = s.Inventory.CurrentLoad()
	}
	if derive {
		d, _, _ := PartyDisplayWithTable(*p, t)
		s.Capacity, s.Speed, s.Movement = d.Capacity, d.Speed, sim.HumanMovement{}
		s.Load = s.Inventory.CurrentLoad()
	} else if s.Movement.Present {
		if s.Capacity != 0 && before.Carry.LiveLoad.Load/s.Capacity == s.Load/s.Capacity {
			s.Movement.Load = s.Load
		} else {
			s.Movement = sim.HumanMovement{}
		}
	}
	p.Carry.LiveLoad = &s
	if reconcile {
		reconcilePartyStacks(before, p, t)
	}
	return nil
}

// Keep surviving original element boundaries. City commands still expose a
// flat compatibility list; only newly acquired objects take the stored index.
func reconcilePartyStacks(before PartyMember, p *PartyMember, t *Table) {
	remaining := cloneItemInstances(MemberCarriedItems(*p, t))
	var stacks []sim.ItemStack
	for _, stack := range before.Carry.OrderedStacks {
		var count uint32
		for i := 0; i < len(remaining) && count < stack.Count; {
			if reflect.DeepEqual(stack.Instance(), remaining[i]) {
				count++
				remaining = append(remaining[:i], remaining[i+1:]...)
			} else {
				i++
			}
		}
		if count != 0 {
			stacks = append(stacks, sim.StackItem(stack.Instance(), count))
		}
	}
	for _, item := range remaining {
		merged := false
		for i := range stacks {
			if reflect.DeepEqual(stacks[i].Instance(), item) {
				stacks[i].Count++
				merged = true
				break
			}
		}
		if merged {
			continue
		}
		index := int(min(p.Carry.LiveLoad.Inventory.InsertIndex, uint32(len(stacks))))
		stacks = append(stacks, sim.ItemStack{})
		copy(stacks[index+1:], stacks[index:])
		stacks[index] = sim.StackItem(item, 1)
	}
	p.Carry.OrderedStacks = stacks
	p.Carry.Items, p.Carry.ItemInstances = nil, nil
	for _, stack := range stacks {
		for n := uint32(0); n < stack.Count; n++ {
			p.Carry.Items = append(p.Carry.Items, stack.Code)
			p.Carry.ItemInstances = append(p.Carry.ItemInstances, stack.Instance())
		}
	}
}

func ValidatePartyLoad(p PartyMember) error {
	if p.Carry == nil {
		return nil
	}
	c := p.Carry
	if c.LiveLoad == nil {
		if c.OrderedStacks == nil {
			return nil
		}
	} else {
		if err := c.LiveLoad.Validate(); err != nil {
			return err
		}
		if !c.LiveLoad.Inventory.ContainerPresent && len(c.OrderedStacks) != 0 {
			return fmt.Errorf("party: absent container has items")
		}
	}
	var count uint64
	for _, s := range c.OrderedStacks {
		if sim.StackStateEqual(s, sim.ItemStack{}) {
			continue
		}
		if s.Count == 0 || s.Code == 0 {
			return fmt.Errorf("party: invalid ordered stack")
		}
		count += uint64(s.Count)
		if count > sim.MaxOriginalHoldingValues {
			return fmt.Errorf("party: ordered container exceeds bound")
		}
	}
	items := MemberCarriedItems(p, nil)
	if uint64(len(items)) != count {
		return fmt.Errorf("party: ordered container count differs from projection")
	}
	k := 0
	for _, s := range c.OrderedStacks {
		for n := uint32(0); n < s.Count; n++ {
			if !reflect.DeepEqual(s.Instance(), items[k]) {
				return fmt.Errorf("party: ordered container differs from projection")
			}
			k++
		}
	}
	return nil
}

func applyHumanLoad(p *PartyMember, state data.HumanState) {
	if p.Carry == nil || p.Carry.LiveLoad == nil {
		return
	}
	s := *p.Carry.LiveLoad
	s.Inventory.OwnWeight, s.Inventory.Accumulator = int16(state.Weight), state.InventoryWeight
	s.Inventory.Source = humanSourceWithRuntime(state, s.Inventory.Source)
	s.Load, s.Capacity, s.Speed = int32(int16(state.Load)), int32(int16(state.Capacity)), int32(int16(state.Speed))
	s.Movement = sim.HumanMovement{Present: true, RawSpeed: int16(state.Speed), NativeSpeed: s.Speed, Load: s.Load, Capacity: s.Capacity}
	p.Carry.LiveLoad = &s
}
