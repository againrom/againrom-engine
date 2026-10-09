package sim

import (
	"encoding/binary"
	"fmt"
)

func (w *World) CheatGod(id EntityID) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return false
	}
	n := w.sourceMutationCopy(i)
	e := &n.entities[i]
	if e.ActorLoad.Source.Class != 0 {
		s := e.SourceNow()
		for j := 0; j < 6; j++ {
			binary.LittleEndian.PutUint16(s.Modifier[46+2*j:], 100)
			s.Modifier[58+j] = 100
		}
		if s.Class == 2 {
			if n.sourceDerive == nil {
				return false
			}
			derived, err := n.sourceDerive(s, e.ActorLoad.Accumulator, n.rules)
			if err != nil {
				return false
			}
			n.finishSourceDerive(i, derived)
		} else {
			e.ActorLoad.Source = s
			if !n.deriveSource(i) {
				return false
			}
		}
	} else {
		e.NativeBasis.ModifierPresent, e.NativeBasis.DefencePresent = true, true
		for j := 0; j < 6; j++ {
			at := 46 + 2*j
			binary.LittleEndian.PutUint16(e.NativeBasis.Modifier[at:], 100)
			e.NativeBasis.ModifierKnown |= uint64(3)<<at | uint64(1)<<(58+j)
			e.NativeBasis.Modifier[58+j] = 100
			binary.LittleEndian.PutUint16(e.NativeBasis.Defence[4+2*j:], 100)
			e.NativeBasis.Defence[16+j] = 100
		}
		e.NativeBasis.DefenceKnown |= uint32((1<<18)-1) << 4
		e.Protection = [5]int32{100, 100, 100, 100, 100}
		e.Resistance = [5]uint8{100, 100, 100, 100, 100}
	}
	*w = n
	return true
}

func (w *World) CheatSpell(id EntityID, spell uint16) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 || spell == 0 || spell >= 32 {
		return false
	}
	if _, ok := w.findSpell(uint32(spell)); !ok {
		return false
	}
	LearnBookSpell(&w.entities[i], spell, w.spells)
	w.refreshSavedBookRoots(i)
	return knowsSpell(w.entities[i], uint32(spell))
}

func (w *World) CheatKillPlayer(owner uint32) int {
	killed := 0
	for i := range w.entities {
		if w.entities[i].Owner == owner {
			w.setUnitProperty(i, propertyHealth, -50)
			killed++
		}
	}
	return killed
}

func (w *World) CheatPickupAll(hero EntityID) error {
	if indexOfEntity(w.entities, hero) < 0 {
		return fmt.Errorf("sim: cheat pickup names absent entity %d", hero)
	}
	if !w.cheatHoldingCapacity(nil, nil, true) {
		return fmt.Errorf("sim: cheat pickup holdings exceed bounded item/effect values")
	}
	for len(w.sacks) != 0 {
		sack := w.sacks[0]
		if err := w.TakeSack(hero, sack.X, sack.Y); err != nil {
			return err
		}
	}
	return nil
}

func (w *World) CheatAddGold(owner, count uint32) bool {
	if owner >= relationSlots {
		return false
	}
	w.purses[owner] += count
	return true
}

func (w *World) CheatAddItem(hero EntityID, stack ItemStack) bool {
	i := indexOfEntity(w.entities, hero)
	if i < 0 || stack.ObjectID != 0 || stack.Code == 0 || stack.Count == 0 || stack.Count > 65535 || len(stack.Effects) > 4096 || !w.hasActorContainer(i) || !w.sourceMutationReady(i) || !w.cheatHoldingCapacity([]ItemStack{stack}, nil, false) || stack.Instance().ValidateWeight() != nil {
		return false
	}
	n := w.sourceMutationCopy(i)
	before := n.beginLoadMutation(i)
	if !n.addCarried(i, stack) || !n.finishLoadMutation(i, before) || !n.savedMutationValid() {
		return false
	}
	n.publishActorItems(i, SelfSlot, false)
	*w = n
	return true
}

func (w *World) CheatCurse(hero EntityID) bool {
	i := indexOfEntity(w.entities, hero)
	if i < 0 {
		return false
	}
	n := w.sourceMutationCopy(i)
	e := &n.entities[i]
	if e.ActorLoad.Source.Class != 0 {
		s := e.SourceNow()
		s.Experience = 0
		s.Stats[0], s.Stats[1], s.Stats[2], s.Stats[3] = 10, 10, 10, 1
		if s.Class != 2 || n.sourceDerive == nil {
			return false
		}
		derived, err := n.sourceDerive(s, e.ActorLoad.Accumulator, n.rules)
		if err != nil {
			return false
		}
		n.finishSourceDerive(i, derived)
	} else {
		e.NativeBasis = e.NativeBasis.WithBody(10)
		e.Reaction, e.Mind, e.Spirit = 10, 10, 1
		RefreshBook(n.rules, e, n.spells)
		n.refreshSavedBookRoots(i)
	}
	*w = n
	return true
}

func (w *World) SetSafeMode(enabled bool) { w.safeMode = enabled }

func (w *World) CheatSummon(template Entity, pack []ItemStack, worn [EquipSlots]ItemInstance, x, y int32) (EntityID, error) {
	if _, ok := w.cellIndex(x, y); !ok {
		return 0, fmt.Errorf("sim: cheat summon anchor is outside map")
	}
	id, available := w.NextEntityID()
	if !available || id >= 1<<28 {
		return 0, fmt.Errorf("sim: cheat summon identity space exhausted")
	}
	template.ID = id
	template.ActorLoad.Source.HasOwner = template.Owner != 0 && template.ActorLoad.Source.Class != 0
	if template.ActorState == 0 {
		template.ActorState = actorStateGuard
	}
	template.OffMap = true
	template.HasTarget = false
	template.clearAttack()
	template.SourceBinding.ArchiveIndex = 0
	template.SourceBinding.GroupIndex, template.SourceBinding.GroupSelector, template.SourceBinding.GroupOwnerKey = 0, 0, 0
	template.SourceBinding.GroupOwnerSlot, template.SourceBinding.GroupOwnerResolved = 0, false
	if template.ActorLoad.Source.Class != 0 {
		template.SourceBinding.Class = GeneratedUnitBinding
		if template.Humanoid {
			template.SourceBinding.Class = GeneratedHumanBinding
		}
		key := uint32(0x50000000) + uint32(id)*16
		for w.cheatActorKeyUsed(key) {
			key += 16
			if key == 0 {
				return 0, fmt.Errorf("sim: cheat summon source key space exhausted")
			}
		}
		template.SourceBinding.Identity, template.SourceBinding.RuntimeID = key, key>>4
	} else {
		template.SourceBinding = SourceBinding{}
	}
	var units uint64
	for _, stack := range pack {
		units += uint64(stack.Count)
		if stack.Count > 65535 || units > MaxOriginalHoldingValues {
			return 0, fmt.Errorf("sim: cheat summon holdings exceed bounded quantity")
		}
	}
	if !w.cheatHoldingCapacity(pack, &worn, false) {
		return 0, fmt.Errorf("sim: cheat summon holdings exceed bounded item/effect values")
	}
	held, equipped, err := normaliseHoldings([]Entity{template}, []Stock{{ID: id, OrderedStacks: pack, ItemInstances: expandItems(pack), EquippedItems: worn}})
	if err != nil {
		return 0, err
	}
	n := w.sourceMutationCopy(0)
	n.actorTraversal = append([]EntityID(nil), w.actorTraversal...)
	n.casts = append([]scriptCast(nil), w.casts...)
	n.entities = append(n.entities, template)
	n.routes = append(n.routes, nil)
	n.carried = append(n.carried, held[0])
	n.equipment = append(n.equipment, equipped[0])
	i := len(n.entities) - 1
	best := int64(1<<63 - 1)
	px, py := int32(0), int32(0)
	for cy := int32(0); cy < n.bounds.Height; cy++ {
		for cx := int32(0); cx < n.bounds.Width; cx++ {
			dx, dy := int64(cx-x), int64(cy-y)
			distance := dx*dx + dy*dy
			if distance < best && n.placeFree(i, cx, cy) {
				best, px, py = distance, cx, cy
			}
		}
	}
	if best == int64(1<<63-1) || !n.placeAt(i, px, py) {
		return 0, fmt.Errorf("sim: cheat summon found no free cell")
	}
	n.entities[i].PostX, n.entities[i].PostY = px, py
	if template.ActorLoad.Source.Class != 0 && !n.deriveSource(i) {
		return 0, fmt.Errorf("sim: cheat summon source actor cannot derive owned state")
	}
	if n.savedGroups != nil {
		container, owner := w.cheatSummonOwner(template.Owner, x, y)
		if !n.newSavedCommandGroup([]int{i}, orderNone, cell{x: px, y: py}, true, container, false) {
			return 0, fmt.Errorf("sim: cheat summon saved group identity space exhausted")
		}
		if owner.Class == 1 {
			n.savedGroupFor(id).Owner = owner
		}
		n.syncSavedActorCommand(i)
	} else {
		n.groups = append([]groupAI(nil), w.groups...)
		n.commandGroup([]int{i}, orderNone, cell{x: px, y: py})
		if template.Owner != 0 && n.entities[i].CommandGroup == 0 {
			return 0, fmt.Errorf("sim: cheat summon native group identity space exhausted")
		}
		n.entities[i].Group = n.entities[i].CommandGroup
	}
	if _, err := n.MarshalBinary(); err != nil {
		return 0, err
	}
	*w = n
	return id, nil
}

func (w *World) cheatHoldingCapacity(pack []ItemStack, worn *[EquipSlots]ItemInstance, ground bool) bool {
	var values uint64
	add := func(count uint32, effects int) bool {
		width := uint64(effects) + 1
		if uint64(count) > (MaxOriginalHoldingValues-values)/width {
			return false
		}
		values += uint64(count) * width
		return true
	}
	for i, e := range w.entities {
		if !e.Alive() && !originalDyingEntity(e) {
			continue
		}
		for _, stack := range w.carried[i] {
			if !add(stack.Count, len(stack.Effects)) {
				return false
			}
		}
		for _, item := range w.equipment[i] {
			if !add(1, len(item.Effects)) {
				return false
			}
		}
	}
	for _, stack := range pack {
		if !add(stack.Count, len(stack.Effects)) {
			return false
		}
	}
	if worn != nil {
		for _, item := range worn {
			if !add(1, len(item.Effects)) {
				return false
			}
		}
	}
	if ground {
		for _, sack := range w.sacks {
			for _, item := range sack.ItemInstances {
				if !add(1, len(item.Effects)) {
					return false
				}
			}
		}
	}
	return true
}

func (w *World) cheatSummonOwner(owner uint32, x, y int32) (uint32, SavedGroupReference) {
	for _, e := range w.entities {
		if e.Owner != owner || e.OffMap || e.X != x || e.Y != y {
			continue
		}
		if g := w.savedGroupFor(e.ID); g != nil {
			if p, found := w.savedPlayerByID(g.ContainerID); found && p.Slot == owner {
				if g.Owner.Class == 1 && g.Owner.Owner == owner {
					return p.ID, g.Owner
				}
				return p.ID, SavedGroupReference{}
			}
		}
	}
	return w.uniqueSavedPlayerSlot(owner), SavedGroupReference{}
}

func (w *World) cheatActorKeyUsed(key uint32) bool {
	for _, e := range w.entities {
		if e.SourceBinding.Identity == key {
			return true
		}
	}
	for _, e := range w.originalDead {
		if e.Source.Identity == key {
			return true
		}
	}
	return false
}
