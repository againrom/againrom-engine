package sim

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// Form95 appends delivery columns, admission payments and prepared effects.
// Each JSON record is a fixed-shape value, length-bounded before decoding.
const spellDeliverySpanLen = 4
const maxSpellDeliveryRecord = 2048

type spellDeliveryState struct {
	rules []SpellRule
	paid  []EntityID
	queue []spellDelivery
}

func (w *World) appendSpellDeliveries(b []byte) []byte {
	var rules []SpellRule
	var paid []EntityID
	for _, r := range w.spells {
		if r.Delivery != 0 || r.EffectSpeed != 0 {
			rules = append(rules, r)
		}
	}
	for _, c := range w.bookCasts {
		if c.Paid {
			paid = append(paid, c.Caster)
		}
	}
	start := len(b)
	if len(rules)+len(paid)+len(w.deliveries) != 0 {
		b = binary.LittleEndian.AppendUint32(b, uint32(len(rules)))
		for _, r := range rules {
			b = binary.LittleEndian.AppendUint16(b, r.ID)
			b = binary.LittleEndian.AppendUint32(b, uint32(r.Delivery))
			b = binary.LittleEndian.AppendUint32(b, uint32(r.EffectSpeed))
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(len(paid)))
		for _, id := range paid {
			b = binary.LittleEndian.AppendUint32(b, uint32(id))
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(len(w.deliveries)))
		for _, d := range w.deliveries {
			raw, _ := json.Marshal(d) // all fields are finite integer/bool values
			b = binary.LittleEndian.AppendUint32(b, uint32(len(raw)))
			b = append(b, raw...)
		}
	}
	return binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
}

func splitSpellDeliveries(data []byte) ([]byte, spellDeliveryState, error) {
	var s spellDeliveryState
	fail := func() ([]byte, spellDeliveryState, error) {
		return nil, s, fmt.Errorf("sim: malformed spell delivery section")
	}
	if len(data) < 4 {
		return fail()
	}
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-4:]))
	if span > uint64(len(data)-4) {
		return fail()
	}
	start := len(data) - 4 - int(span)
	b := data[start : len(data)-4]
	if len(b) == 0 {
		return data[:start], s, nil
	}
	count := func(width int) (int, bool) {
		if len(b) < 4 {
			return 0, false
		}
		n := uint64(binary.LittleEndian.Uint32(b))
		b = b[4:]
		return int(n), n <= uint64(len(b)/width)
	}
	n, ok := count(10)
	if !ok {
		return fail()
	}
	for i := 0; i < n; i++ {
		s.rules = append(s.rules, SpellRule{ID: binary.LittleEndian.Uint16(b), Delivery: int32(binary.LittleEndian.Uint32(b[2:])), EffectSpeed: int32(binary.LittleEndian.Uint32(b[6:]))})
		b = b[10:]
	}
	n, ok = count(4)
	if !ok {
		return fail()
	}
	for i := 0; i < n; i++ {
		s.paid = append(s.paid, EntityID(binary.LittleEndian.Uint32(b)))
		b = b[4:]
	}
	n, ok = count(5)
	if !ok {
		return fail()
	}
	for i := 0; i < n; i++ {
		if len(b) < 4 {
			return fail()
		}
		size := uint64(binary.LittleEndian.Uint32(b))
		b = b[4:]
		if size == 0 || size > maxSpellDeliveryRecord || size > uint64(len(b)) {
			return fail()
		}
		var d spellDelivery
		decoder := json.NewDecoder(bytes.NewReader(b[:size]))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&d); err != nil {
			return fail()
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			return fail()
		}
		canonical, _ := json.Marshal(d)
		if !bytes.Equal(canonical, b[:size]) {
			return fail()
		}
		s.queue = append(s.queue, d)
		b = b[size:]
	}
	if len(b) != 0 || len(s.rules)+len(s.paid)+len(s.queue) == 0 {
		return fail()
	}
	return data[:start], s, nil
}

func (w *World) applySpellDeliveryState(s spellDeliveryState) error {
	last := -1
	for _, r := range s.rules {
		i := -1
		for j := range w.spells {
			if w.spells[j].ID == r.ID {
				i = j
				break
			}
		}
		if i <= last || r.Delivery < 0 || r.EffectSpeed < 0 || r.Delivery == 0 && r.EffectSpeed == 0 {
			return fmt.Errorf("sim: invalid spell delivery columns")
		}
		w.spells[i].Delivery, w.spells[i].EffectSpeed = r.Delivery, r.EffectSpeed
		last = i
	}
	last = -1
	for _, id := range s.paid {
		i, ok := w.bookCastIndex(id)
		if !ok || i <= last || w.bookCasts[i].Phase != bookCharging {
			return fmt.Errorf("sim: invalid paid cast")
		}
		w.bookCasts[i].Paid = true
		last = i
	}
	w.deliveries = s.queue
	return w.spellDeliveryFault()
}

func (w *World) spellDeliveryFault() error {
	for _, r := range w.spells {
		if r.Delivery < 0 || r.EffectSpeed < 0 {
			return fmt.Errorf("sim: invalid spell delivery columns")
		}
	}
	for _, c := range w.bookCasts {
		if c.Paid && c.Phase != bookCharging {
			return fmt.Errorf("sim: invalid paid cast")
		}
	}
	for _, d := range w.deliveries {
		if d.ConstructSacrifice && (d.Rule.arm() != 4 || !d.AtCell || d.Current == nil || SavedAreaPayloadSupported(&d.Current.Payload)) {
			return fmt.Errorf("sim: invalid deferred sacrifice construction")
		}
		if c := d.Current; c != nil {
			if !d.AtCell || c.Key != cellKey(d.X, d.Y) || c.Spell != d.Rule.ID || c.Mode < areaModeBlast || c.Mode > areaModeCloud ||
				c.Direction > 7 || len(c.Cells) != 0 || c.Policy != (CurrentAreaPolicy{}) || d.Payload == nil || c.Payload != *d.Payload {
				return fmt.Errorf("sim: invalid prepared area delivery")
			}
		}
		if d.Payload != nil {
			if err := savedEffectClassFault(*d.Payload); err != nil {
				return err
			}
		}
		// Resolved book cost is a signed word; weapon power keeps its full
		// signed actor domain. Table validation must not narrow either source.
		if d.Rule.Delivery != 1 || d.Rule.bookInstance || d.Rule.bookDefensive != 0 || d.Rule.ID == 0 ||
			d.Rule.DamageMin < 0 || d.Rule.DamageMax < 0 ||
			!d.HasCaster && d.Caster != 0 || d.AtCell != d.Rule.Area ||
			d.BirthTick > w.tick || d.FanHead && (!d.HasCaster || d.Rule.arm() != 14 || d.AtCell) {
			return fmt.Errorf("sim: invalid prepared spell delivery")
		}
		if _, ok := cellIndexIn(w.bounds, d.X, d.Y); !ok {
			return fmt.Errorf("sim: spell delivery destination outside map")
		}
	}
	return nil
}
