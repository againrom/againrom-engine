package sim

import (
	"encoding/binary"
	"fmt"
)

const sourceEquipmentLen = 77

// Ordinals use the same owner walk as explicit weights. Empty equipment
// slots count too; retained values are sparse, never a parallel live owner.
func (w *World) eachSourceEquipment(visit func(uint32, uint16, *SourceEquipment)) {
	var ordinal uint32
	item := func(i *ItemInstance) { visit(ordinal, i.Code, &i.SourceEquipment); ordinal++ }
	for si := range w.sacks {
		for ii := range w.sacks[si].ItemInstances {
			item(&w.sacks[si].ItemInstances[ii])
		}
	}
	for ei := range w.carried {
		for si := range w.carried[ei] {
			s := &w.carried[ei][si]
			visit(ordinal, s.Code, &s.SourceEquipment)
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
	// Constructor rows follow all live item owners. Their code/weight table
	// has already been decoded; no ordinal controls an allocation.
	for i := range w.itemWeights {
		visit(ordinal, w.itemWeights[i].Code, &w.itemWeights[i].Constructor)
		ordinal++
	}
}

func (w *World) sourceEquipmentFault() (fault error) {
	w.eachSourceEquipment(func(o uint32, code uint16, s *SourceEquipment) {
		if fault != nil {
			return
		}
		if err := s.Validate(); err != nil {
			fault = fmt.Errorf("item %d: %w", o, err)
		}
		if code == 0 && s.Class != 0 {
			fault = fmt.Errorf("empty item %d has source equipment", o)
		}
	})
	if fault == nil {
		_, fault = normaliseItemWeights(w.itemWeights)
	}
	return
}

func (w *World) sourceEquipmentSectionLen() int {
	n := 0
	w.eachSourceEquipment(func(_ uint32, _ uint16, s *SourceEquipment) {
		if s.Class != 0 {
			n += 4 + sourceEquipmentLen
		}
	})
	if n != 0 {
		n += 4
	}
	return n
}

func (w *World) encodeSourceEquipment(b []byte, off int) int {
	w.eachSourceEquipment(func(o uint32, _ uint16, s *SourceEquipment) {
		if s.Class == 0 {
			return
		}
		binary.LittleEndian.PutUint32(b[off:], o)
		_, _ = binary.Encode(b[off+4:off+4+sourceEquipmentLen], binary.LittleEndian, *s)
		off += 4 + sourceEquipmentLen
	})
	return off
}

func (w *World) decodeSourceEquipment(b []byte) error {
	const recordLen = 4 + sourceEquipmentLen
	if len(b)%recordLen != 0 {
		return fmt.Errorf("sim: invalid source equipment span")
	}
	for off := recordLen; off < len(b); off += recordLen {
		if binary.LittleEndian.Uint32(b[off:]) <= binary.LittleEndian.Uint32(b[off-recordLen:]) {
			return fmt.Errorf("sim: source equipment ordinals are not strictly increasing")
		}
	}
	off := 0
	var fault error
	w.eachSourceEquipment(func(o uint32, code uint16, s *SourceEquipment) {
		if off == len(b) || binary.LittleEndian.Uint32(b[off:]) != o {
			return
		}
		_, err := binary.Decode(b[off+4:off+recordLen], binary.LittleEndian, s)
		if err != nil {
			fault = err
		}
		if code == 0 || s.Class == 0 {
			fault = fmt.Errorf("sim: source equipment ordinal %d is empty", o)
		}
		if b[off+4+49] > 1 {
			fault = fmt.Errorf("sim: noncanonical source equipment definition presence")
		}
		if b[off+4+70] > 1 {
			fault = fmt.Errorf("sim: noncanonical source equipment Spell presence")
		}
		if b[off+4+76] > 1 {
			fault = fmt.Errorf("sim: noncanonical source equipment unsupported-effects flag")
		}
		off += recordLen
	})
	if fault != nil {
		return fault
	}
	if off != len(b) {
		return fmt.Errorf("sim: source equipment ordinal is outside item population")
	}
	return w.sourceEquipmentFault()
}
