package sim

import (
	"encoding/binary"
	"fmt"
)

// The form-74 suffix stores only explicit per-instance weights. Ordinals
// enumerate every slot, including empty equipment slots, in this fixed order.
// The in-memory owner remains ItemInstance/ItemStack; no sidecar is retained.
func (w *World) eachItemWeight(visit func(uint32, uint16, *bool, *int16)) {
	var ordinal uint32
	item := func(i *ItemInstance) { visit(ordinal, i.Code, &i.WeightPresent, &i.Weight); ordinal++ }
	for si := range w.sacks {
		for ii := range w.sacks[si].ItemInstances {
			item(&w.sacks[si].ItemInstances[ii])
		}
	}
	for ei := range w.carried {
		for si := range w.carried[ei] {
			s := &w.carried[ei][si]
			visit(ordinal, s.Code, &s.WeightPresent, &s.Weight)
			ordinal++
		}
	}
	for ei := range w.equipment {
		for si := range w.equipment[ei] {
			item(&w.equipment[ei][si])
		}
	}
	for ci := range w.scrollCasts {
		item(&w.scrollCasts[ci].Item)
	}
}

func itemWeightFault(code uint16, present bool, weight int16) error {
	if !present && weight != 0 {
		return fmt.Errorf("absent item weight has residue %d", weight)
	}
	if code == 0 && present {
		return fmt.Errorf("empty item has explicit weight")
	}
	return nil
}

func (w *World) itemWeightsFault() (fault error) {
	w.eachItemWeight(func(o uint32, c uint16, p *bool, v *int16) {
		if err := itemWeightFault(c, *p, *v); fault == nil && err != nil {
			fault = fmt.Errorf("sim: item %d: %w", o, err)
		}
	})
	return
}

func (w *World) instanceWeightSectionLen() int {
	n := 4
	w.eachItemWeight(func(_ uint32, _ uint16, p *bool, _ *int16) {
		if *p {
			n += 6
		}
	})
	return n
}

func (w *World) encodeInstanceWeights(b []byte, off int) int {
	start := off
	w.eachItemWeight(func(o uint32, _ uint16, p *bool, v *int16) {
		if *p {
			binary.LittleEndian.PutUint32(b[off:], o)
			binary.LittleEndian.PutUint16(b[off+4:], uint16(*v))
			off += 6
		}
	})
	binary.LittleEndian.PutUint32(b[off:], uint32(off-start))
	return off + 4
}

func (w *World) decodeInstanceWeights(b []byte) error {
	if len(b)%6 != 0 {
		return fmt.Errorf("sim: invalid instance-weight span")
	}
	for off := 6; off < len(b); off += 6 {
		if binary.LittleEndian.Uint32(b[off:]) <= binary.LittleEndian.Uint32(b[off-6:]) {
			return fmt.Errorf("sim: instance-weight ordinals are not strictly increasing")
		}
	}
	// No allocation depends on a supplied ordinal or record count. Walk the
	// actual decoded population once and reject any unconsumed record.
	off := 0
	var fault error
	w.eachItemWeight(func(o uint32, c uint16, p *bool, v *int16) {
		if off < len(b) && binary.LittleEndian.Uint32(b[off:]) == o {
			if c == 0 {
				fault = fmt.Errorf("sim: instance-weight ordinal %d names an empty item", o)
			}
			*p, *v = true, int16(binary.LittleEndian.Uint16(b[off+4:]))
			off += 6
		}
	})
	if fault != nil {
		return fault
	}
	if off != len(b) {
		return fmt.Errorf("sim: instance-weight ordinal is outside the item population")
	}
	for i, stacks := range w.carried {
		for j, stack := range stacks {
			if (stack.Code == 0 || stack.Count == 0) && !emptyOrderedStack(stack) {
				return fmt.Errorf("sim: entity record %d: carried stack %d has invalid null slot", i, j)
			}
		}
	}
	return nil
}
