package sim

import (
	"encoding/binary"
	"fmt"
)

const scrollRecordLen = 20

func (w *World) scrollSectionLen() int {
	n := 4
	for _, c := range w.scrollCasts {
		n += scrollRecordLen + itemByteLen(c.Item)
	}
	return n
}

func (w *World) encodeScrolls(b []byte, off int) int {
	binary.LittleEndian.PutUint32(b[off:], uint32(len(w.scrollCasts)))
	off += 4
	for _, c := range w.scrollCasts {
		binary.LittleEndian.PutUint32(b[off:], uint32(c.Caster))
		binary.LittleEndian.PutUint32(b[off+4:], uint32(c.Target))
		binary.LittleEndian.PutUint32(b[off+8:], uint32(c.X))
		binary.LittleEndian.PutUint32(b[off+12:], uint32(c.Y))
		binary.LittleEndian.PutUint16(b[off+16:], c.Index)
		if c.AtCell {
			b[off+18] |= 1
		}
		if c.Started {
			b[off+18] |= 2
		}
		b[off+19] = c.Remaining
		off = encodeItemAt(b, off+scrollRecordLen, c.Item)
	}
	return off
}

func decodeScrolls(b []byte, entities []Entity, bounds Bounds, spells []SpellRule) ([]ScrollCast, int, error) {
	if len(b) < 4 {
		return nil, 0, fmt.Errorf("sim: truncated scroll count")
	}
	n := uint64(binary.LittleEndian.Uint32(b))
	if n > uint64(len(entities)) || n > uint64((len(b)-4)/scrollRecordLen) {
		return nil, 0, fmt.Errorf("sim: invalid scroll count %d", n)
	}
	out, off := make([]ScrollCast, 0, int(n)), 4
	for i := uint64(0); i < n; i++ {
		if off > len(b)-scrollRecordLen {
			return nil, 0, fmt.Errorf("sim: truncated scroll record")
		}
		c := ScrollCast{Caster: EntityID(binary.LittleEndian.Uint32(b[off:])), Target: EntityID(binary.LittleEndian.Uint32(b[off+4:])), X: int32(binary.LittleEndian.Uint32(b[off+8:])), Y: int32(binary.LittleEndian.Uint32(b[off+12:])), Index: binary.LittleEndian.Uint16(b[off+16:]), AtCell: b[off+18]&1 != 0, Started: b[off+18]&2 != 0, Remaining: b[off+19]}
		if b[off+18]&^3 != 0 || c.AtCell && c.Target != 0 || !c.Started && c.Remaining > 1 || c.Started && c.Remaining == 0 {
			return nil, 0, fmt.Errorf("sim: invalid scroll lifecycle")
		}
		if ci := indexOfEntity(entities, c.Caster); ci < 0 || entities[ci].HP <= 0 || entities[ci].OffMap {
			return nil, 0, fmt.Errorf("sim: invalid scroll caster")
		}
		if len(out) != 0 && out[len(out)-1].Caster >= c.Caster {
			return nil, 0, fmt.Errorf("sim: unordered scroll casters")
		}
		if _, inside := cellIndexIn(bounds, c.X, c.Y); !inside {
			return nil, 0, fmt.Errorf("sim: scroll outside map")
		}
		off += scrollRecordLen
		item, used, err := decodeItemBytes(b[off:], "reserved scroll")
		if err != nil {
			return nil, 0, err
		}
		id, _, ok := ScrollSpell(item)
		found := false
		for _, rule := range spells {
			if rule.ID == id && spellApplicable(rule) && (!c.AtCell || !rule.TargetsUnit) {
				found = true
				break
			}
		}
		if !ok || !found {
			return nil, 0, fmt.Errorf("sim: invalid reserved scroll")
		}
		c.Item = item
		off += used
		out = append(out, c)
	}
	return out, off, nil
}
