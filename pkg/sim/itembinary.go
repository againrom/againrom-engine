package sim

import (
	"encoding/binary"
	"fmt"
)

const itemStateCountLen = 4

// secondaryDamageWireLen keeps form 61's landed width. Only its first three
// bytes are state; every remaining byte is required to be zero. Later forms
// preserve that width so the sections behind it do not move unnecessarily.
const secondaryDamageWireLen = 2 * 5 * 4
const itemDerivedStateLen = 3*4 + secondaryDamageWireLen

func (w *World) itemStateSectionLen() int {
	n := itemStateCountLen
	for _, sack := range w.sacks {
		n += itemStateCountLen
		for _, item := range sack.ItemInstances {
			n += itemByteLen(item)
		}
	}
	n += itemStateCountLen
	for _, stacks := range w.carried {
		n += itemStateCountLen
		for _, stack := range stacks {
			n += itemByteLen(stack.Instance()) + stackCountLen
		}
	}
	for _, worn := range w.equipment {
		for _, item := range worn {
			n += itemByteLen(item)
		}
	}
	return n + len(w.entities)*(1+itemDerivedStateLen)
}

func encodeItemAt(dst []byte, off int, item ItemInstance) int {
	encoded := appendItemBytes(nil, item)
	copy(dst[off:], encoded)
	return off + len(encoded)
}

func (w *World) encodeItemState(dst []byte, off int) int {
	binary.LittleEndian.PutUint32(dst[off:off+4], uint32(len(w.sacks)))
	off += itemStateCountLen
	for _, sack := range w.sacks {
		binary.LittleEndian.PutUint32(dst[off:off+4], uint32(len(sack.ItemInstances)))
		off += itemStateCountLen
		for _, item := range sack.ItemInstances {
			off = encodeItemAt(dst, off, item)
		}
	}
	binary.LittleEndian.PutUint32(dst[off:off+4], uint32(len(w.entities)))
	off += itemStateCountLen
	for _, stacks := range w.carried {
		binary.LittleEndian.PutUint32(dst[off:off+4], uint32(len(stacks)))
		off += itemStateCountLen
		for _, stack := range stacks {
			off = encodeItemAt(dst, off, stack.Instance())
			binary.LittleEndian.PutUint32(dst[off:off+4], stack.Count)
			off += stackCountLen
		}
	}
	for _, worn := range w.equipment {
		for _, item := range worn {
			off = encodeItemAt(dst, off, item)
		}
	}
	for _, entity := range w.entities {
		dst[off] = byte(entity.WeaponSpellSource)
		off++
	}
	for _, entity := range w.entities {
		for _, value := range []int32{entity.HealthRegeneration, entity.ManaRegeneration, entity.RotationSpeed} {
			binary.LittleEndian.PutUint32(dst[off:off+4], uint32(value))
			off += 4
		}
		dst[off], dst[off+1], dst[off+2] = entity.SecondaryDamage.Base,
			entity.SecondaryDamage.Spread, entity.SecondaryDamage.Selector
		for k := 3; k < secondaryDamageWireLen; k++ {
			dst[off+k] = 0
		}
		off += secondaryDamageWireLen
	}
	return off
}

func decodeItemState(data []byte, sacks []Sack, carried [][]ItemStack,
	equipment [][EquipSlots]ItemInstance, entities []Entity) (int, error) {
	off := 0
	readCount := func(what string) (uint32, error) {
		if off+itemStateCountLen > len(data) {
			return 0, fmt.Errorf("sim: byte form truncated: %s needs %d byte(s), %d left", what, itemStateCountLen, len(data)-off)
		}
		n := binary.LittleEndian.Uint32(data[off : off+itemStateCountLen])
		off += itemStateCountLen
		return n, nil
	}
	sackCount, err := readCount("the item-state sack count")
	if err != nil {
		return 0, err
	}
	if int(sackCount) != len(sacks) {
		return 0, fmt.Errorf("sim: item-state section names %d sacks, world names %d", sackCount, len(sacks))
	}
	for i := range sacks {
		count, err := readCount(fmt.Sprintf("sack %d's item-state count", i))
		if err != nil {
			return 0, err
		}
		if int(count) != len(sacks[i].Items) {
			return 0, fmt.Errorf("sim: sack %d item-state count is %d, code projection count is %d", i, count, len(sacks[i].Items))
		}
		items := make([]ItemInstance, count)
		for k := range items {
			item, used, err := decodeItemBytes(data[off:], fmt.Sprintf("sack %d item %d", i, k))
			if err != nil {
				return 0, err
			}
			if item.Code != sacks[i].Items[k] {
				return 0, fmt.Errorf("sim: sack %d item %d code %#04x disagrees with projection %#04x", i, k, item.Code, sacks[i].Items[k])
			}
			items[k] = item
			off += used
		}
		sacks[i].ItemInstances = items
	}
	entityCount, err := readCount("the item-state entity count")
	if err != nil {
		return 0, err
	}
	if int(entityCount) != len(entities) {
		return 0, fmt.Errorf("sim: item-state section names %d entities, header names %d", entityCount, len(entities))
	}
	for i := range carried {
		count, err := readCount(fmt.Sprintf("entity %d's item-state stack count", i))
		if err != nil {
			return 0, err
		}
		if uint64(count) > uint64(len(data)-off)/15 {
			return 0, fmt.Errorf("sim: item-state stack count exceeds remaining bytes")
		}
		stacks := make([]ItemStack, count)
		var expected, decoded uint64
		for _, stack := range carried[i] {
			expected += uint64(stack.Count)
		}
		for k := range stacks {
			item, used, err := decodeItemBytes(data[off:], fmt.Sprintf("entity %d carried stack %d", i, k))
			if err != nil {
				return 0, err
			}
			off += used
			if off+stackCountLen > len(data) {
				return 0, fmt.Errorf("sim: byte form truncated: entity %d carried stack %d count needs %d byte(s), %d left", i, k, stackCountLen, len(data)-off)
			}
			quantity := binary.LittleEndian.Uint32(data[off : off+stackCountLen])
			off += stackCountLen
			if (item.Code == 0 || quantity == 0) && !emptyOrderedStack(StackItem(item, quantity)) {
				return 0, fmt.Errorf("sim: entity record %d: carried stack %d has invalid null slot", i, k)
			}
			decoded += uint64(quantity)
			if decoded > expected {
				return 0, fmt.Errorf("sim: entity %d stack quantity exceeds bounded carried projection", i)
			}
			stacks[k] = StackItem(item, quantity)
		}
		// Canonical folding is validated only after the form-74 weight suffix
		// has populated the instances. Equal codes can have distinct weights.
		if got, want := expandContainer(stacks), expandContainer(carried[i]); !equalItemCodes(got, want) {
			return 0, fmt.Errorf("sim: entity record %d: item-state code projection disagrees with carried section", i)
		}
		carried[i] = stacks
		if entities[i].ActorLoad.Present && !entities[i].ActorLoad.ContainerPresent && len(stacks) != 0 {
			return 0, fmt.Errorf("sim: absent actor container carries stacks")
		}
	}
	for i := range equipment {
		for k := 0; k < EquipSlots; k++ {
			item, used, err := decodeItemBytes(data[off:], fmt.Sprintf("entity %d equipment slot %d", i, k+1))
			if err != nil {
				return 0, err
			}
			if item.Code != equipment[i][k].Code {
				return 0, fmt.Errorf("sim: entity %d equipment slot %d item code disagrees with projection", i, k+1)
			}
			equipment[i][k] = item
			off += used
		}
	}
	if len(data)-off < len(entities) {
		return 0, fmt.Errorf("sim: byte form truncated: weapon-spell source tags need %d byte(s), %d left", len(entities), len(data)-off)
	}
	for i := range entities {
		entities[i].WeaponSpellSource = WeaponSpellSource(data[off])
		off++
		if err := validateWeaponSource(entities[i], equipment[i][slotWeapon]); err != nil {
			return 0, fmt.Errorf("sim: entity record %d: %w", i, err)
		}
	}
	if len(data)-off < len(entities)*itemDerivedStateLen {
		return 0, fmt.Errorf("sim: byte form truncated: item-derived state needs %d byte(s), %d left", len(entities)*itemDerivedStateLen, len(data)-off)
	}
	for i := range entities {
		read := func() int32 {
			value := int32(binary.LittleEndian.Uint32(data[off : off+4]))
			off += 4
			return value
		}
		entities[i].HealthRegeneration = read()
		entities[i].ManaRegeneration = read()
		entities[i].RotationSpeed = read()
		// RotationSpeed lands here, not in the fixed entity record decoded
		// earlier in binary.go, so turnFault is checked here too: it is the
		// first point in the decode where an entity's RotationSpeed and its
		// turn-progress bytes are both populated.
		if err := turnFault(entities[i]); err != nil {
			return 0, fmt.Errorf("sim: entity record %d: %w", i, err)
		}
		entities[i].SecondaryDamage = SecondaryDamage{
			Base: data[off], Spread: data[off+1], Selector: data[off+2],
		}
		if err := secondaryDamageFault(entities[i].SecondaryDamage); err != nil {
			return 0, fmt.Errorf("sim: entity record %d: %w", i, err)
		}
		for k, value := range data[off+3 : off+secondaryDamageWireLen] {
			if value != 0 {
				return 0, fmt.Errorf("sim: entity record %d: secondary-damage reserved byte %d is %#02x, want zero", i, k+3, value)
			}
		}
		off += secondaryDamageWireLen
	}
	return off, nil
}

func equalItemCodes(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
