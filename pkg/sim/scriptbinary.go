package sim

// The byte form's SCRIPT SECTION: the mission script's volatile state, then the
// compiled program itself.
//
// It sits at the very end of the form, after the routes, for the reason every
// field added since version 4 went to a tail: putting it anywhere earlier would
// move every offset after it and buy nothing.
//
// The VOLATILE HALF is fixed-width and always written — the hundred registers,
// the thousand latch bytes, the two counters and the outcome — whether or not
// the world runs a script. That is the same trade the grid makes: a world with
// no script writes the same zeros a world whose script has done nothing writes,
// so there is no absent case below the encoder and no flag recording which way a
// world was built.
//
// The PROGRAM is three counted arrays and a world with no script writes three
// zeros. It is here at all because a world carries no map to re-derive it from;
// the original stores only the volatile half because its own loader re-runs the
// builder first.

import (
	"encoding/binary"
	"fmt"
)

// The script section's fixed widths.
//
//	off                    width      field
//	0                      400        registers, 100 * int32
//	400                    1000       latch bytes, one per trigger position
//	1400                   4 + 4      the win and lose counters, uint32
//	1408                   1          the outcome
//	1409                   4 + 4 + 4  check, instant and trigger counts, uint32
//	1421                   76 each    checks
//	then                   64 each    instants
//	then                   60 each    triggers
//
// A check record is op, register, the ten parameters, the two resolved
// entity references and their two presence bytes, the group a check names
// and its own presence byte, then the two players a check names and their
// own two presence bytes; an instant record is op, the ten parameters, then
// the unit, group and player it names and a presence byte for each, then the
// second unit reference this story adds and its own presence byte; a trigger
// record is three (left, right, code, used) pairs, four instant subscripts,
// the fire-once flag and the latch index.
//
// The eleven reference presence bytes are bytes of their own and not sentinel
// ids, because
// entity id zero is a real entity and group zero is a real group — the same
// reason the entity record carries a target-presence byte rather than a reserved
// coordinate. A ROSTER SLOT is 1-based and zero does name nobody, so a player
// flag alone could have been folded away; it is a byte like its neighbours
// because folding it would make one reference slot read differently from every
// other one, which is a cost paid at every reading of this record to save one
// byte per reference.
//
// The group pair was the check record's tail through version 30, appended at
// +58, so every offset the earlier version fixed is where it put it. The
// instant's three references are its own tail on the same terms, fifteen bytes
// appended at +44: the three values at +44, +48 and +52, then their three flags
// at +56, +57 and +58.
//
// The check's two players are its NEWEST tail, appended at +63 (0122): Player
// and Player2 at +63 and +67, HasPlayer and HasPlayer2 at +71 and +72 — the same
// shape Unit and Unit2 already set inside this record, values together and then
// their flags together, rather than the group pair's interleaved (value, flag)
// beside them. Every offset before +63, the group pair included, is unmoved.
//
// The instant's second unit reference was appended at +59: the value at +59,
// the flag at +63 — the same treatment the group pair and then the player
// pair got inside the check record, applied to the instant record instead.
//
// The instant's ITEM reference is the section's own newest tail now,
// appended at +64: the code as a 16-bit word at +64, its flag at +66, taking
// the record from 64 bytes to 67. Every offset before +64 is unmoved. The
// code is TWO bytes and not four because it is a packed u16 in the value
// space it comes from, and widening it here would store two bytes of zero on
// every instant of every world.
//
// The CHECK's own item reference is the section's newest tail through
// version 57, appended at +73: the code as a 16-bit word at +73, its flag at
// +75, taking the check record from 73 bytes to 76. Every offset before +73
// is unmoved, the player pair at +63..+72 included. It is two bytes for the
// instant's own reason above, and it is a tail rather than a slot beside the
// group pair because moving any earlier offset would rewrite every check
// record for no gain.
//
// THE CHECK's own STRUCTURE reference is the section's newest tail (1033 B3,
// version 58), appended at +76: the id as a uint32 at +76, its flag at +80,
// taking the check record from 76 bytes to 81. Every offset before +76 is
// unmoved, the item reference included, on that reference's own reason.
//
// THE INSTANT's own STRUCTURE reference is appended at +67, on the same
// version 58 move: the id as a uint32 at +67, its flag at +71, taking the
// instant record from 67 bytes to 72. Every offset before +67 is unmoved,
// the item reference at +64..+66 included.
const (
	scriptStateLen   = 4*scriptRegisters + scriptLatches + 4 + 4 + 1
	scriptCountsLen  = 12
	scriptCheckLen   = 81
	scriptInstantLen = 72
	scriptTriggerLen = 60
)

// scriptSectionLen is how many bytes w's script section occupies.
func (w *World) scriptSectionLen() int {
	n := scriptStateLen + scriptCountsLen
	if w.script == nil {
		return n
	}
	return n + scriptCheckLen*len(w.script.checks) +
		scriptInstantLen*len(w.script.instants) +
		scriptTriggerLen*len(w.script.triggers)
}

// encodeScript writes the script section into b at off and returns the offset
// past it.
func (w *World) encodeScript(b []byte, off int) int {
	for i, v := range w.registers {
		binary.LittleEndian.PutUint32(b[off+4*i:off+4*i+4], uint32(v))
	}
	off += 4 * scriptRegisters
	copy(b[off:off+scriptLatches], w.latches[:])
	off += scriptLatches
	binary.LittleEndian.PutUint32(b[off:off+4], w.won)
	binary.LittleEndian.PutUint32(b[off+4:off+8], w.lost)
	b[off+8] = byte(w.outcome)
	off += 9

	var checks []ScriptCheck
	var instants []ScriptInstant
	var triggers []ScriptTrigger
	if w.script != nil {
		checks, instants, triggers = w.script.checks, w.script.instants, w.script.triggers
	}
	binary.LittleEndian.PutUint32(b[off:off+4], uint32(len(checks)))
	binary.LittleEndian.PutUint32(b[off+4:off+8], uint32(len(instants)))
	binary.LittleEndian.PutUint32(b[off+8:off+12], uint32(len(triggers)))
	off += scriptCountsLen

	for _, c := range checks {
		binary.LittleEndian.PutUint32(b[off:off+4], uint32(c.Op))
		binary.LittleEndian.PutUint32(b[off+4:off+8], uint32(c.Register))
		for k, v := range c.Args {
			binary.LittleEndian.PutUint32(b[off+8+4*k:off+12+4*k], uint32(v))
		}
		binary.LittleEndian.PutUint32(b[off+48:off+52], uint32(c.Unit))
		binary.LittleEndian.PutUint32(b[off+52:off+56], uint32(c.Unit2))
		if c.HasUnit {
			b[off+56] = 1
		}
		if c.HasUnit2 {
			b[off+57] = 1
		}
		binary.LittleEndian.PutUint32(b[off+58:off+62], c.Group)
		if c.HasGroup {
			b[off+62] = 1
		}
		binary.LittleEndian.PutUint32(b[off+63:off+67], c.Player)
		binary.LittleEndian.PutUint32(b[off+67:off+71], c.Player2)
		if c.HasPlayer {
			b[off+71] = 1
		}
		if c.HasPlayer2 {
			b[off+72] = 1
		}
		binary.LittleEndian.PutUint16(b[off+73:off+75], c.Item)
		if c.HasItem {
			b[off+75] = 1
		}
		binary.LittleEndian.PutUint32(b[off+76:off+80], uint32(c.Structure))
		if c.HasStructure {
			b[off+80] = 1
		}
		off += scriptCheckLen
	}
	for _, in := range instants {
		binary.LittleEndian.PutUint32(b[off:off+4], uint32(in.Op))
		for k, v := range in.Args {
			binary.LittleEndian.PutUint32(b[off+4+4*k:off+8+4*k], uint32(v))
		}
		binary.LittleEndian.PutUint32(b[off+44:off+48], uint32(in.Unit))
		binary.LittleEndian.PutUint32(b[off+48:off+52], in.Group)
		binary.LittleEndian.PutUint32(b[off+52:off+56], in.Player)
		if in.HasUnit {
			b[off+56] = 1
		}
		if in.HasGroup {
			b[off+57] = 1
		}
		if in.HasPlayer {
			b[off+58] = 1
		}
		binary.LittleEndian.PutUint32(b[off+59:off+63], uint32(in.Unit2))
		if in.HasUnit2 {
			b[off+63] = 1
		}
		binary.LittleEndian.PutUint16(b[off+64:off+66], in.Item)
		if in.HasItem {
			b[off+66] = 1
		}
		binary.LittleEndian.PutUint32(b[off+67:off+71], uint32(in.Structure))
		if in.HasStructure {
			b[off+71] = 1
		}
		off += scriptInstantLen
	}
	for _, t := range triggers {
		for k, p := range t.Pairs {
			o := off + 13*k
			binary.LittleEndian.PutUint32(b[o:o+4], uint32(p.Left))
			binary.LittleEndian.PutUint32(b[o+4:o+8], uint32(p.Right))
			binary.LittleEndian.PutUint32(b[o+8:o+12], uint32(p.Cmp))
			if p.Used {
				b[o+12] = 1
			}
		}
		for k, s := range t.Instants {
			binary.LittleEndian.PutUint32(b[off+39+4*k:off+43+4*k], uint32(s))
		}
		if t.Once {
			b[off+55] = 1
		}
		binary.LittleEndian.PutUint32(b[off+56:off+60], uint32(t.Latch))
		off += scriptTriggerLen
	}
	return off
}

// scriptState is the volatile half as the decoder recovers it, before it is
// assigned over a world. It exists so the decode can fail after reading and
// leave the receiver untouched, which is the rule the whole of UnmarshalBinary
// keeps.
type scriptState struct {
	registers [scriptRegisters]int32
	latches   [scriptLatches]byte
	won       uint32
	lost      uint32
	outcome   Outcome
	script    *Script
}

// decodeScript reads the script section out of the form's tail.
//
// It requires data to be consumed EXACTLY, which is where the whole form's
// "truncated and over-long are the same comparison" now lands: the routes reader
// hands over what it did not use, and anything left after the last trigger
// record is a form this package did not write.
//
// The enumerated bytes — the outcome, the two presence bytes, a pair's used flag,
// the fire-once flag and a latch — are REFUSED outside their value sets rather
// than read as truthy. That is what keeps the encoding injective: were any
// nonzero byte read as true, two different forms would decode to one world.
//
// Everything structural is checked by NewScript rather than here, so a script
// that reaches a world by being decoded is held to exactly what one that reaches
// a world by being compiled is held to.
func decodeScript(data []byte) (scriptState, error) {
	var st scriptState
	if len(data) < scriptStateLen+scriptCountsLen {
		return st, fmt.Errorf("sim: byte form truncated: the script section needs %d byte(s), %d left",
			scriptStateLen+scriptCountsLen, len(data))
	}

	off := 0
	for i := range st.registers {
		st.registers[i] = int32(binary.LittleEndian.Uint32(data[off+4*i : off+4*i+4]))
	}
	off += 4 * scriptRegisters
	for i := 0; i < scriptLatches; i++ {
		switch data[off+i] {
		case 0, 1:
			st.latches[i] = data[off+i]
		default:
			return st, fmt.Errorf("sim: script latch %d is %d, want 0 or 1", i, data[off+i])
		}
	}
	off += scriptLatches
	st.won = binary.LittleEndian.Uint32(data[off : off+4])
	st.lost = binary.LittleEndian.Uint32(data[off+4 : off+8])
	st.outcome = Outcome(data[off+8])
	off += 9
	if !st.outcome.defined() {
		return st, fmt.Errorf("sim: script outcome is %d, which is not defined", uint8(st.outcome))
	}
	// Original Player outcome and session counters are separate persisted fields.
	// In particular primary-hero failure does not increment LOSE (SAV-FLAG-027).
	// Their domains are validated independently; loading must not invent a counter.

	nChecks := binary.LittleEndian.Uint32(data[off : off+4])
	nInstants := binary.LittleEndian.Uint32(data[off+4 : off+8])
	nTriggers := binary.LittleEndian.Uint32(data[off+8 : off+12])
	off += scriptCountsLen

	// Each span is multiplied out in int64 and required to fit what is left
	// before a single record is allocated, so a hostile count cannot ask for
	// memory the form does not carry the bytes for.
	span := int64(nChecks)*scriptCheckLen +
		int64(nInstants)*scriptInstantLen +
		int64(nTriggers)*scriptTriggerLen
	if avail := int64(len(data) - off); span != avail {
		return st, fmt.Errorf("sim: script section declares %d check(s), %d instant(s) and %d trigger(s), "+
			"which are %d byte(s), and carries %d", nChecks, nInstants, nTriggers, span, avail)
	}

	checks := make([]ScriptCheck, nChecks)
	for i := range checks {
		o := off + scriptCheckLen*i
		c := ScriptCheck{
			Op:        int32(binary.LittleEndian.Uint32(data[o : o+4])),
			Register:  int32(binary.LittleEndian.Uint32(data[o+4 : o+8])),
			Unit:      EntityID(binary.LittleEndian.Uint32(data[o+48 : o+52])),
			Unit2:     EntityID(binary.LittleEndian.Uint32(data[o+52 : o+56])),
			Item:      binary.LittleEndian.Uint16(data[o+73 : o+75]),
			Structure: StructureID(binary.LittleEndian.Uint32(data[o+76 : o+80])),
		}
		for k := range c.Args {
			c.Args[k] = int32(binary.LittleEndian.Uint32(data[o+8+4*k : o+12+4*k]))
		}
		var err error
		if c.HasUnit, err = scriptFlag(data[o+56], "check", i, "unit presence"); err != nil {
			return st, err
		}
		if c.HasUnit2, err = scriptFlag(data[o+57], "check", i, "second unit presence"); err != nil {
			return st, err
		}
		c.Group = binary.LittleEndian.Uint32(data[o+58 : o+62])
		if c.HasGroup, err = scriptFlag(data[o+62], "check", i, "group presence"); err != nil {
			return st, err
		}
		c.Player = binary.LittleEndian.Uint32(data[o+63 : o+67])
		c.Player2 = binary.LittleEndian.Uint32(data[o+67 : o+71])
		if c.HasPlayer, err = scriptFlag(data[o+71], "check", i, "player presence"); err != nil {
			return st, err
		}
		if c.HasPlayer2, err = scriptFlag(data[o+72], "check", i, "second player presence"); err != nil {
			return st, err
		}
		if c.HasItem, err = scriptFlag(data[o+75], "check", i, "item presence"); err != nil {
			return st, err
		}
		if c.HasStructure, err = scriptFlag(data[o+80], "check", i, "structure presence"); err != nil {
			return st, err
		}
		checks[i] = c
	}
	off += int(int64(nChecks) * scriptCheckLen)

	instants := make([]ScriptInstant, nInstants)
	for i := range instants {
		o := off + scriptInstantLen*i
		in := ScriptInstant{
			Op:        int32(binary.LittleEndian.Uint32(data[o : o+4])),
			Unit:      EntityID(binary.LittleEndian.Uint32(data[o+44 : o+48])),
			Group:     binary.LittleEndian.Uint32(data[o+48 : o+52]),
			Player:    binary.LittleEndian.Uint32(data[o+52 : o+56]),
			Unit2:     EntityID(binary.LittleEndian.Uint32(data[o+59 : o+63])),
			Item:      binary.LittleEndian.Uint16(data[o+64 : o+66]),
			Structure: StructureID(binary.LittleEndian.Uint32(data[o+67 : o+71])),
		}
		for k := range in.Args {
			in.Args[k] = int32(binary.LittleEndian.Uint32(data[o+4+4*k : o+8+4*k]))
		}
		var err error
		if in.HasUnit, err = scriptFlag(data[o+56], "instant", i, "unit presence"); err != nil {
			return st, err
		}
		if in.HasGroup, err = scriptFlag(data[o+57], "instant", i, "group presence"); err != nil {
			return st, err
		}
		if in.HasPlayer, err = scriptFlag(data[o+58], "instant", i, "player presence"); err != nil {
			return st, err
		}
		if in.HasUnit2, err = scriptFlag(data[o+63], "instant", i, "second unit presence"); err != nil {
			return st, err
		}
		if in.HasItem, err = scriptFlag(data[o+66], "instant", i, "item presence"); err != nil {
			return st, err
		}
		if in.HasStructure, err = scriptFlag(data[o+71], "instant", i, "structure presence"); err != nil {
			return st, err
		}
		instants[i] = in
	}
	off += int(int64(nInstants) * scriptInstantLen)

	triggers := make([]ScriptTrigger, nTriggers)
	for i := range triggers {
		o := off + scriptTriggerLen*i
		var t ScriptTrigger
		for k := range t.Pairs {
			po := o + 13*k
			p := ScriptPair{
				Left:  int32(binary.LittleEndian.Uint32(data[po : po+4])),
				Right: int32(binary.LittleEndian.Uint32(data[po+4 : po+8])),
				Cmp:   int32(binary.LittleEndian.Uint32(data[po+8 : po+12])),
			}
			var err error
			if p.Used, err = scriptFlag(data[po+12], "trigger", i, "pair use"); err != nil {
				return st, err
			}
			t.Pairs[k] = p
		}
		for k := range t.Instants {
			t.Instants[k] = int32(binary.LittleEndian.Uint32(data[o+39+4*k : o+43+4*k]))
		}
		var err error
		if t.Once, err = scriptFlag(data[o+55], "trigger", i, "fire-once"); err != nil {
			return st, err
		}
		t.Latch = int32(binary.LittleEndian.Uint32(data[o+56 : o+60]))
		triggers[i] = t
	}

	if nChecks == 0 && nInstants == 0 && nTriggers == 0 {
		return st, nil
	}
	s, err := NewScript(checks, instants, triggers)
	if err != nil {
		return st, err
	}
	st.script = s
	return st, nil
}

// scriptFlag reads one of the section's boolean bytes, refusing anything but 0
// and 1. It is one function so that the thirteen sites cannot come to differ about
// what a flag byte is.
func scriptFlag(b byte, what string, i int, field string) (bool, error) {
	switch b {
	case 0:
		return false, nil
	case 1:
		return true, nil
	}
	return false, fmt.Errorf("sim: script %s %d: %s byte is %d, want 0 or 1", what, i, field, b)
}
