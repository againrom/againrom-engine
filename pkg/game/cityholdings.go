package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// A loaded native mission owns its registry independently of any retained city.
func validateCurrentMissionRegistry(s *Snapshot, w *sim.World) error {
	if s == nil || s.Mission == 0 || w == nil || w.SavedObjects() == nil {
		return fmt.Errorf("current mission item registry is absent")
	}
	return w.SavedObjects().ValidateNoInFlight()
}

func cloneCityItemGraph(g *sav.CityItemGraph) *sav.CityItemGraph {
	if g == nil {
		return nil
	}
	n := *g
	n.Inventory = slices.Clone(g.Inventory)
	n.Objects = slices.Clone(g.Objects)
	for i := range n.Objects {
		r := &n.Objects[i]
		r.Values = slices.Clone(r.Values)
		r.Counts = slices.Clone(r.Counts)
		r.Raw = slices.Clone(r.Raw)
		for k := range r.Raw {
			r.Raw[k].Bytes = slices.Clone(r.Raw[k].Bytes)
		}
		r.RefSlots = slices.Clone(r.RefSlots)
		for k := range r.RefSlots {
			r.RefSlots[k].Objects = slices.Clone(r.RefSlots[k].Objects)
		}
	}
	return &n
}

func cloneCityEquipmentGraph(g *sav.CityEquipmentGraph) *sav.CityEquipmentGraph {
	if g == nil {
		return nil
	}
	n := *g
	n.Objects = slices.Clone(g.Objects)
	for i := range n.Objects {
		r := &n.Objects[i]
		r.Values = slices.Clone(r.Values)
		r.Counts = slices.Clone(r.Counts)
		r.Raw = slices.Clone(r.Raw)
		for k := range r.Raw {
			r.Raw[k].Bytes = slices.Clone(r.Raw[k].Bytes)
		}
		r.RefSlots = slices.Clone(r.RefSlots)
		for k := range r.RefSlots {
			r.RefSlots[k].Objects = slices.Clone(r.RefSlots[k].Objects)
		}
	}
	return &n
}

// checkCityEquipmentGraph cross-validates a captured worn-equipment graph
// against the same current member's live mapload projection, exactly as
// checkCityItemGraph does for the pack. Definition is normalized first
// because it is a load-time rebound field, never a round-tripped archive one.
func checkCityEquipmentGraph(g *sav.CityEquipmentGraph, p mapload.PartyMember) error {
	if err := sav.ValidateCityEquipmentGraph(g); err != nil {
		return err
	}
	doc := sav.DocumentData{Objects: g.Objects}
	for i := range g.Objects {
		if g.Objects[i].Class == "Effect" {
			value, err := savedStructureValue(&g.Objects[i], "E0C")
			if err != nil || value != 0 {
				return fmt.Errorf("city current worn set has unsupported Effect lifetime")
			}
		}
	}
	want := mapload.MemberItemEquipment(p, nil)
	for slot, ref := range g.Slots {
		if ref == 0 {
			if want[slot].Code != 0 {
				return fmt.Errorf("city current worn slot %d omits current item", slot)
			}
			continue
		}
		if want[slot].Code == 0 {
			return fmt.Errorf("city current worn slot %d has no current item", slot)
		}
		item, err := savedItemRecord(&doc, ref)
		if err != nil {
			return err
		}
		item.Value.SourceEquipment.Definition = want[slot].SourceEquipment.Definition
		if !sim.StackStateEqual(item.Value, sim.StackItem(want[slot], 1)) {
			return fmt.Errorf("city current worn slot %d differs from current item", slot)
		}
	}
	return nil
}

// Capture a closed legacy equipment fragment when one exists. The current
// city topology carries the complete graph independently of this fragment.
func captureCityEquipmentGraphs(party []mapload.PartyMember, w *sim.World, ids []sim.EntityID) map[string]*sav.CityEquipmentGraph {
	if w == nil || len(party) != len(ids) || w.SavedObjects() == nil {
		return nil
	}
	if _, err := w.MarshalBinary(); err != nil {
		return nil
	}
	r := w.SavedObjects()
	graphs := make(map[string]*sav.CityEquipmentGraph)
	for i, p := range party {
		g, err := cityEquipmentGraphFromWorld(r, ids[i])
		if err != nil || p.ID == "" || graphs[p.ID] != nil {
			return nil
		}
		graphs[p.ID] = g
	}
	return graphs
}

// Empty worn slots have no ItemRoot; an all-zero fragment is valid.
func cityEquipmentGraphFromWorld(r *sim.SavedObjects, entity sim.EntityID) (*sav.CityEquipmentGraph, error) {
	g := &sav.CityEquipmentGraph{}
	owned := func(row sim.SavedItemObject) bool {
		return slices.ContainsFunc(r.ItemRoots, func(root sim.SavedItemRoot) bool {
			return root.ID == row.ID && root.Owner.Kind == sim.SavedOwnerActorWorn && root.Owner.Entity == entity
		})
	}
	indices := make(map[sim.SavedObjectID]uint16)
	var items []sim.SavedItemObject
	var effects []sim.SavedEffectObject
	var spells []sim.SavedSpellObject
	add := func(id sim.SavedObjectID) error {
		if id == 0 || indices[id] != 0 || len(indices) >= 4096 {
			return fmt.Errorf("city return worn set has aliased/absent item child")
		}
		indices[id] = uint16(len(indices) + 1)
		return nil
	}
	var slots [12]sim.SavedObjectID
	for _, root := range r.ItemRoots {
		if root.Owner.Kind != sim.SavedOwnerActorWorn || root.Owner.Entity != entity {
			continue
		}
		row, exists := r.Item(root.ID)
		if !exists || row.Retired || root.Owner.Slot == 0 || root.Owner.Slot > sim.EquipSlots || row.Coverage != (sim.SavedObjectCoverage{}) {
			return nil, fmt.Errorf("city return has foreign or unsupported worn item")
		}
		position := root.Owner.Slot - 1
		if slots[position] != 0 {
			return nil, fmt.Errorf("city return worn slot %d repeats", position)
		}
		slots[position] = row.ID
		if err := add(row.ID); err != nil {
			return nil, err
		}
		items = append(items, row)
		for _, childID := range row.Effects {
			if err := add(childID); err != nil {
				return nil, err
			}
			found := false
			for _, child := range r.Effects {
				if child.ID == childID && !child.Retired && child.ExternalReferences == 0 && child.E0C == 0 && child.Coverage == (sim.SavedObjectCoverage{}) {
					effects, found = append(effects, child), true
				}
			}
			if !found {
				return nil, fmt.Errorf("city return has unavailable Effect child")
			}
		}
		if row.Spell != 0 {
			if err := add(row.Spell); err != nil {
				return nil, err
			}
			found := false
			for _, child := range r.Spells {
				if child.ID == row.Spell && !child.Retired && child.ExternalReferences == 0 && child.Coverage == (sim.SavedObjectCoverage{}) {
					spells, found = append(spells, child), true
				}
			}
			if !found {
				return nil, fmt.Errorf("city return has unavailable Spell child")
			}
		}
	}
	// A child borrowed by a different current item is not a closed worn graph.
	for _, row := range r.Items {
		if owned(row) || row.Retired {
			continue
		}
		for _, id := range row.Effects {
			if indices[id] != 0 {
				return nil, fmt.Errorf("city return worn Effect has an external owner")
			}
		}
		if row.Spell != 0 && indices[row.Spell] != 0 {
			return nil, fmt.Errorf("city return worn Spell has an external owner")
		}
	}
	for i, id := range slots {
		if id != 0 {
			g.Slots[i] = indices[id]
		}
	}
	g.Objects = make([]sav.DocumentRecordData, len(indices))
	for _, row := range items {
		record, err := savedCurrentItemRecord(row, indices)
		if err != nil {
			return nil, err
		}
		g.Objects[indices[row.ID]-1] = record
	}
	for _, row := range effects {
		g.Objects[indices[row.ID]-1] = savedCurrentEffectRecord(row)
	}
	for _, row := range spells {
		g.Objects[indices[row.ID]-1] = savedCurrentSpellRecord(row)
	}
	if err := sav.ValidateCityEquipmentGraph(g); err != nil {
		return nil, err
	}
	return g, nil
}

func checkCityItemGraph(g *sav.CityItemGraph, p mapload.PartyMember) error {
	if err := sav.ValidateCityItemGraph(g); err != nil {
		return err
	}
	if p.Carry == nil || p.Carry.LiveLoad == nil {
		return fmt.Errorf("city current pack lacks actor load")
	}
	a := p.Carry.LiveLoad.Inventory
	if a.ContainerPresent != g.Present || a.InsertIndex != g.InsertIndex || a.Accumulator != g.Accumulator {
		return fmt.Errorf("city current pack bookkeeping differs from Human")
	}
	stacks := p.Carry.OrderedStacks
	if len(stacks) != len(g.Inventory) {
		return fmt.Errorf("city current pack omits ordered item")
	}
	doc := sav.DocumentData{Objects: g.Objects}
	for i := range g.Objects {
		if g.Objects[i].Class == "Effect" {
			value, err := savedStructureValue(&g.Objects[i], "E0C")
			if err != nil || value != 0 {
				return fmt.Errorf("city current pack has unsupported Effect lifetime")
			}
		}
	}
	for i, ref := range g.Inventory {
		item, err := savedItemRecord(&doc, ref)
		if err != nil {
			return err
		}
		item.Value.SourceEquipment.Definition = stacks[i].SourceEquipment.Definition
		if !sim.StackStateEqual(item.Value, stacks[i]) {
			return fmt.Errorf("city current pack item %d differs from current stack", i)
		}
	}
	var expanded []sim.ItemInstance
	for _, stack := range stacks {
		if stack.Count > 65535 || len(expanded)+int(stack.Count) > 65536 {
			return fmt.Errorf("city pack expansion exceeds bound")
		}
		for range stack.Count {
			expanded = append(expanded, stack.Instance())
		}
	}
	items := mapload.MemberCarriedItems(p, nil)
	if len(items) != len(expanded) {
		return fmt.Errorf("city pack expansion omits current item")
	}
	for i := range items {
		if !sim.StackStateEqual(sim.StackItem(items[i], 1), sim.StackItem(expanded[i], 1)) {
			return fmt.Errorf("city pack expansion differs from current item")
		}
	}
	return nil
}

// Capture an optional closed legacy pack fragment before native handles clear.
func captureCityItemGraphs(party []mapload.PartyMember, w *sim.World, ids []sim.EntityID) map[string]*sav.CityItemGraph {
	if w == nil || len(party) != len(ids) || w.SavedObjects() == nil {
		return nil
	}
	if _, err := w.MarshalBinary(); err != nil {
		return nil
	}
	r := w.SavedObjects()
	graphs := make(map[string]*sav.CityItemGraph)
	for i, p := range party {
		g, err := cityItemGraphFromWorld(r, ids[i])
		if err != nil || p.ID == "" || graphs[p.ID] != nil {
			return nil
		}
		graphs[p.ID] = g
	}
	return graphs
}

func cityItemGraphFromWorld(r *sim.SavedObjects, entity sim.EntityID) (*sav.CityItemGraph, error) {
	owner := sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: entity}
	var container *sim.SavedObjectContainer
	for i := range r.Containers {
		if r.Containers[i].Owner == owner {
			container = &r.Containers[i]
		}
	}
	if container == nil || container.Coverage != (sim.SavedObjectCoverage{}) {
		return nil, fmt.Errorf("city return pack lacks complete registry")
	}
	g := &sav.CityItemGraph{Present: container.Present, InsertIndex: container.InsertIndex, Accumulator: container.Accumulator}
	indices := make(map[sim.SavedObjectID]uint16)
	var items []sim.SavedItemObject
	var effects []sim.SavedEffectObject
	var spells []sim.SavedSpellObject
	add := func(id sim.SavedObjectID) error {
		if id == 0 || indices[id] != 0 || len(indices) >= 4096 {
			return fmt.Errorf("city return has aliased/absent item child")
		}
		indices[id] = uint16(len(indices) + 1)
		return nil
	}
	for position, id := range container.Items {
		row, ok := r.Item(id)
		if !ok || row.Retired || !r.HasLocation(id, sim.SavedItemLocation{Owner: owner, Index: uint32(position)}) || row.Coverage != (sim.SavedObjectCoverage{}) {
			return nil, fmt.Errorf("city return has foreign or unsupported item")
		}
		if err := add(id); err != nil {
			return nil, err
		}
		g.Inventory = append(g.Inventory, indices[id])
		items = append(items, row)
		for _, childID := range row.Effects {
			if err := add(childID); err != nil {
				return nil, err
			}
			found := false
			for _, child := range r.Effects {
				if child.ID == childID && !child.Retired && child.ExternalReferences == 0 && child.E0C == 0 && child.Coverage == (sim.SavedObjectCoverage{}) {
					effects, found = append(effects, child), true
				}
			}
			if !found {
				return nil, fmt.Errorf("city return has unavailable Effect child")
			}
		}
		if row.Spell != 0 {
			if err := add(row.Spell); err != nil {
				return nil, err
			}
			found := false
			for _, child := range r.Spells {
				if child.ID == row.Spell && !child.Retired && child.ExternalReferences == 0 && child.Coverage == (sim.SavedObjectCoverage{}) {
					spells, found = append(spells, child), true
				}
			}
			if !found {
				return nil, fmt.Errorf("city return has unavailable Spell child")
			}
		}
	}
	// A child borrowed by a different current item is not a closed pack graph.
	for _, row := range r.Items {
		if slices.Contains(container.Items, row.ID) || row.Retired {
			continue
		}
		for _, id := range row.Effects {
			if indices[id] != 0 {
				return nil, fmt.Errorf("city return Effect has an external owner")
			}
		}
		if row.Spell != 0 && indices[row.Spell] != 0 {
			return nil, fmt.Errorf("city return Spell has an external owner")
		}
	}
	g.Objects = make([]sav.DocumentRecordData, len(indices))
	for _, row := range items {
		record, err := savedCurrentItemRecord(row, indices)
		if err != nil {
			return nil, err
		}
		g.Objects[indices[row.ID]-1] = record
	}
	for _, row := range effects {
		g.Objects[indices[row.ID]-1] = savedCurrentEffectRecord(row)
	}
	for _, row := range spells {
		g.Objects[indices[row.ID]-1] = savedCurrentSpellRecord(row)
	}
	if err := sav.ValidateCityItemGraph(g); err != nil {
		return nil, err
	}
	return g, nil
}
