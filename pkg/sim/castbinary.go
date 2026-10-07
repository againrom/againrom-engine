package sim

import (
	"encoding/binary"
	"fmt"
)

// Version 53 widens the casting section. Area records retain their phase and
// painted cells; attached records retain actor effects. Explicit counts keep
// custom radii out of compiled limits.
const (
	castingCountLen   = 4
	castRecordLen     = 12
	castTargetOffset  = 11
	areaHeadLen       = 26
	attachedRecordLen = 19
	bookRecordLen     = 24
	effectRecordLen   = 27 // compatibility name for the tagged area record
	castingAreaTag    = 1
	castingAttachTag  = 2
	castingBookTag    = 3
)

func (w *World) castingSectionLen() int {
	n := 2*castingCountLen + castRecordLen*len(w.casts) + (bookRecordLen+1)*len(w.bookCasts) +
		(areaHeadLen+1)*len(w.effects) + (attachedRecordLen+1)*len(w.attached)
	for _, e := range w.effects {
		n += 2 * len(e.Cells)
	}
	return n
}

func (w *World) encodeCasting(b []byte, off int) int {
	binary.LittleEndian.PutUint32(b[off:off+4], uint32(len(w.casts)))
	off += castingCountLen
	for _, c := range w.casts {
		b[off], b[off+1], b[off+2], b[off+3], b[off+4] = c.FromX, c.FromY, c.ToX, c.ToY, c.Spell
		binary.LittleEndian.PutUint16(b[off+5:off+7], c.Power)
		binary.LittleEndian.PutUint32(b[off+7:off+11], uint32(c.Target))
		if c.AtUnit {
			b[off+castTargetOffset] = 1
		}
		off += castRecordLen
	}
	binary.LittleEndian.PutUint32(b[off:off+4], uint32(len(w.bookCasts)+len(w.effects)+len(w.attached)))
	off += castingCountLen
	for _, c := range w.bookCasts {
		b[off] = castingBookTag
		off++
		binary.LittleEndian.PutUint32(b[off:off+4], uint32(c.Caster))
		binary.LittleEndian.PutUint32(b[off+4:off+8], uint32(c.Target))
		binary.LittleEndian.PutUint16(b[off+8:off+10], c.Spell)
		binary.LittleEndian.PutUint32(b[off+10:off+14], uint32(c.X))
		binary.LittleEndian.PutUint32(b[off+14:off+18], uint32(c.Y))
		b[off+18] = c.Remaining
		if c.AtCell {
			b[off+19] = 1
		}
		b[off+20] = byte(c.Phase)
		b[off+21] = c.Progress
		if c.Complete {
			b[off+22] = 1
		}
		if c.Retained {
			b[off+23] = 1
		}
		off += bookRecordLen
	}
	for _, e := range w.effects {
		b[off] = castingAreaTag
		off++
		binary.LittleEndian.PutUint16(b[off:off+2], e.Key)
		binary.LittleEndian.PutUint16(b[off+2:off+4], e.Spell)
		binary.LittleEndian.PutUint16(b[off+4:off+6], e.Remaining)
		binary.LittleEndian.PutUint32(b[off+6:off+10], uint32(e.Caster))
		if e.HasCaster {
			b[off+10] = 1
		}
		binary.LittleEndian.PutUint16(b[off+11:off+13], e.Power)
		b[off+13], b[off+14], b[off+15] = e.Mode, e.Phase, e.Direction
		binary.LittleEndian.PutUint32(b[off+16:off+20], uint32(e.DamageMin))
		binary.LittleEndian.PutUint32(b[off+20:off+24], uint32(e.DamageMax))
		binary.LittleEndian.PutUint16(b[off+24:off+26], uint16(len(e.Cells)))
		off += areaHeadLen
		for _, k := range e.Cells {
			binary.LittleEndian.PutUint16(b[off:off+2], k)
			off += 2
		}
	}
	for _, e := range w.attached {
		b[off] = castingAttachTag
		off++
		binary.LittleEndian.PutUint32(b[off:off+4], uint32(e.Target))
		binary.LittleEndian.PutUint32(b[off+4:off+8], uint32(e.Caster))
		if e.HasCaster {
			b[off+8] = 1
		}
		binary.LittleEndian.PutUint16(b[off+9:off+11], e.Spell)
		b[off+11], b[off+12] = byte(e.Kind), byte(e.Mode)
		binary.LittleEndian.PutUint32(b[off+13:off+17], uint32(e.Magnitude))
		binary.LittleEndian.PutUint16(b[off+17:off+19], e.Remaining)
		off += attachedRecordLen
	}
	return off
}

func decodeCasting(data []byte) ([]scriptCast, []bookCast, []cellEffect, []attachedEffect, int, error) {
	need := func(off, n int, what string) error {
		if off < 0 || n < 0 || off > len(data)-n {
			return fmt.Errorf("sim: byte form truncated: %s needs %d byte(s), %d left", what, n, len(data)-off)
		}
		return nil
	}
	if err := need(0, 4, "the pending-cast count"); err != nil {
		return nil, nil, nil, nil, 0, err
	}
	nc, off := int(binary.LittleEndian.Uint32(data[:4])), 4
	if nc > (len(data)-off)/castRecordLen {
		return nil, nil, nil, nil, 0, fmt.Errorf("sim: byte form declares %d pending cast(s) beyond the section", nc)
	}
	casts := make([]scriptCast, nc)
	for i := range casts {
		o := off + i*castRecordLen
		c := scriptCast{FromX: data[o], FromY: data[o+1], ToX: data[o+2], ToY: data[o+3], Spell: data[o+4], Power: binary.LittleEndian.Uint16(data[o+5 : o+7]), Target: EntityID(binary.LittleEndian.Uint32(data[o+7 : o+11]))}
		if c.Spell == 0 {
			return nil, nil, nil, nil, 0, fmt.Errorf("sim: pending cast %d names spell id 0", i)
		}
		// Zero power is valid for cell-entry casts (UNIT-M10CAST-056).
		// Instant 21's default is applied by that producer, not by this format.
		switch data[o+11] {
		case 0:
		case 1:
			c.AtUnit = true
		default:
			return nil, nil, nil, nil, 0, fmt.Errorf("sim: pending cast %d target flag is %d, want 0 or 1", i, data[o+11])
		}
		casts[i] = c
	}
	off += nc * castRecordLen
	if err := need(off, 4, "the area-effect count"); err != nil {
		return nil, nil, nil, nil, 0, err
	}
	nstate := int(binary.LittleEndian.Uint32(data[off : off+4]))
	off += 4
	effects := make([]cellEffect, 0, nstate)
	attached := make([]attachedEffect, 0, nstate)
	book := make([]bookCast, 0, nstate)
	legacyKey, legacyCount, legacySeen := uint16(0), 0, false
	seenArea := false
	seenAttached := false
	for i := 0; i < nstate; i++ {
		if err := need(off, 1, "a casting-state tag"); err != nil {
			return nil, nil, nil, nil, 0, err
		}
		tag := data[off]
		off++
		if tag == castingBookTag && !seenArea && !seenAttached {
			if err := need(off, bookRecordLen, "a book-cast record"); err != nil {
				return nil, nil, nil, nil, 0, err
			}
			c := bookCast{Caster: EntityID(binary.LittleEndian.Uint32(data[off : off+4])),
				Target: EntityID(binary.LittleEndian.Uint32(data[off+4 : off+8])),
				Spell:  binary.LittleEndian.Uint16(data[off+8 : off+10]),
				X:      int32(binary.LittleEndian.Uint32(data[off+10 : off+14])),
				Y:      int32(binary.LittleEndian.Uint32(data[off+14 : off+18])), Remaining: data[off+18],
				Phase: bookPhase(data[off+20]), Progress: data[off+21]}
			switch data[off+19] {
			case 0:
			case 1:
				c.AtCell = true
			default:
				return nil, nil, nil, nil, 0, fmt.Errorf("sim: book cast %d has invalid target flag", len(book))
			}
			switch data[off+22] {
			case 0:
			case 1:
				c.Complete = true
			default:
				return nil, nil, nil, nil, 0, fmt.Errorf("sim: book cast %d has invalid completion flag", len(book))
			}
			switch data[off+23] {
			case 0:
			case 1:
				c.Retained = true
			default:
				return nil, nil, nil, nil, 0, fmt.Errorf("sim: book cast %d has invalid retained flag", len(book))
			}
			if c.Spell == 0 || len(book) > 0 && book[len(book)-1].Caster >= c.Caster {
				return nil, nil, nil, nil, 0, fmt.Errorf("sim: book cast %d has empty or unordered state", len(book))
			}
			if fault := bookCastFault(c); fault != "" {
				return nil, nil, nil, nil, 0, fmt.Errorf("sim: book cast %d: %s", len(book), fault)
			}
			book = append(book, c)
			off += bookRecordLen
			continue
		}
		if tag == castingAttachTag {
			seenAttached = true
			if err := need(off, attachedRecordLen, "an attached-effect record"); err != nil {
				return nil, nil, nil, nil, 0, err
			}
			e := attachedEffect{Target: EntityID(binary.LittleEndian.Uint32(data[off : off+4])), Caster: EntityID(binary.LittleEndian.Uint32(data[off+4 : off+8])), Spell: binary.LittleEndian.Uint16(data[off+9 : off+11]), Kind: EffectKind(data[off+11]), Mode: EffectMode(data[off+12]), Magnitude: int32(binary.LittleEndian.Uint32(data[off+13 : off+17])), Remaining: binary.LittleEndian.Uint16(data[off+17 : off+19])}
			switch data[off+8] {
			case 0:
			case 1:
				e.HasCaster = true
			default:
				return nil, nil, nil, nil, 0, fmt.Errorf("sim: attached effect %d has invalid caster flag", len(attached))
			}
			// Remaining zero is a valid transient state. Instant 30 can author it
			// after the ordinary effect pass, and that state must survive a save
			// until the next pass removes the record.
			if e.Kind == EffectNone || e.Spell == 0 && (e.HasCaster || e.Caster != 0 ||
				(e.Kind != EffectAbsorption && e.Kind != EffectHealthRegeneration && e.Kind != EffectManaRegeneration) ||
				(e.Mode != EffectDuration && e.Mode != EffectContinuous)) {
				return nil, nil, nil, nil, 0, fmt.Errorf("sim: attached effect %d has an empty field", len(attached))
			}
			if len(attached) > 0 {
				p := attached[len(attached)-1]
				if p.Target > e.Target || p.Target == e.Target && p.Spell >= e.Spell {
					return nil, nil, nil, nil, 0, fmt.Errorf("sim: attached effects are not strictly ordered")
				}
			}
			attached = append(attached, e)
			off += attachedRecordLen
			continue
		}
		if tag != castingAreaTag || seenAttached {
			return nil, nil, nil, nil, 0, fmt.Errorf("sim: casting state %d has invalid or out-of-order tag %d", i, tag)
		}
		seenArea = true
		if err := need(off, areaHeadLen, "an area-effect head"); err != nil {
			return nil, nil, nil, nil, 0, err
		}
		e := cellEffect{Key: binary.LittleEndian.Uint16(data[off : off+2]), Spell: binary.LittleEndian.Uint16(data[off+2 : off+4]), Remaining: binary.LittleEndian.Uint16(data[off+4 : off+6]), Caster: EntityID(binary.LittleEndian.Uint32(data[off+6 : off+10])), Power: binary.LittleEndian.Uint16(data[off+11 : off+13]), Mode: data[off+13], Phase: data[off+14], Direction: data[off+15], DamageMin: int32(binary.LittleEndian.Uint32(data[off+16 : off+20])), DamageMax: int32(binary.LittleEndian.Uint32(data[off+20 : off+24]))}
		switch data[off+10] {
		case 0:
		case 1:
			e.HasCaster = true
		default:
			return nil, nil, nil, nil, 0, fmt.Errorf("sim: area effect %d has invalid caster flag", i)
		}
		cells := int(binary.LittleEndian.Uint16(data[off+24 : off+26]))
		off += areaHeadLen
		if err := need(off, cells*2, "an area-effect cell list"); err != nil {
			return nil, nil, nil, nil, 0, err
		}
		e.Cells = make([]uint16, cells)
		for k := range e.Cells {
			e.Cells[k] = binary.LittleEndian.Uint16(data[off+2*k : off+2*k+2])
			if k > 0 && e.Cells[k] <= e.Cells[k-1] {
				return nil, nil, nil, nil, 0, fmt.Errorf("sim: area effect %d cell list is not strictly ascending", i)
			}
		}
		off += cells * 2
		if e.Spell == 0 {
			return nil, nil, nil, nil, 0, fmt.Errorf("sim: area effect %d names spell id 0", i)
		}
		// Unshaped mode0 records retain their explicitly legacy anchor contract.
		if e.Mode == 0 {
			if legacySeen && e.Key < legacyKey {
				return nil, nil, nil, nil, 0, fmt.Errorf("sim: legacy area effects are not in ascending key order")
			}
			if !legacySeen || e.Key != legacyKey {
				legacyCount = 0
			}
			legacyKey, legacySeen = e.Key, true
			legacyCount++
			if legacyCount > cellEffectSlots && e.Spell != 3 {
				return nil, nil, nil, nil, 0, fmt.Errorf("sim: a cell holds more than six legacy effects")
			}
		}

		effects = append(effects, e)
	}
	return casts, book, effects, attached, off, nil
}

// bookCastFault rejects lifecycle shapes no simulation tick can leave behind.
// The target id may be absent after same-tick teardown and entity zero is a
// lawful id, so target existence is deliberately revalidated by the action on
// the next tick rather than guessed here.
func bookCastFault(c bookCast) string {
	if c.Progress > 3 {
		return fmt.Sprintf("retry progress is %d, want 0..3", c.Progress)
	}
	if c.AtCell && c.Target != 0 {
		return "cell-target form carries a unit target"
	}
	switch c.Phase {
	case bookPending:
		if c.Remaining != 0 {
			return "pending phase carries a countdown"
		}
		if !c.Complete && c.Progress != 0 {
			return "incomplete pending phase carries retry progress"
		}
		if !c.Retained {
			return "pending phase is not retained"
		}
	case bookCharging:
		if c.Remaining == 0 {
			return "charging phase has no countdown"
		}
		if c.Progress != 0 || c.Complete {
			return "charging phase carries completion residue"
		}
	case bookRelaxing:
		if c.Remaining == 0 || !c.Complete || !c.Retained || c.Progress != 0 {
			return "relaxing phase has inconsistent countdown, completion or retention"
		}
	case bookBoundaryOne, bookBoundaryTwo:
		if c.Remaining != 0 || !c.Complete || !c.Retained || c.Progress != 0 {
			return "boundary phase has inconsistent countdown, completion or retention"
		}
	case bookApproach:
		if c.Remaining != 0 || c.Complete || !c.Retained || c.Progress != 0 || c.Paid {
			return "approach phase has inconsistent countdown, completion or retention"
		}
	default:
		return fmt.Sprintf("phase is %d, want 0..5", c.Phase)
	}
	return ""
}
