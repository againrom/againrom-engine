package alm

// The type-7 leaf grammar: the map's own authored mission script.
//
// alm.go decodes the type-7 record as a count word and a raw body and says the
// leaf grammar is undecoded (R-2). It is decoded now, and this file is the whole
// of it — the raw Body stays exactly as it was, so nothing that round-trips a
// map through Write reads a byte of this.
//
// The payload is THREE COUNTED ARRAYS and not one (ALM-TRIG-044): an action
// array, a condition array and a trigger array, each behind its own count word.
// The first count word is the one alm.go already reads as Triggers.EntryCount,
// which is why Script walks Body from its start rather than skipping anything.
//
// Nothing here resolves an identifier and nothing here judges an opcode. A node
// carries the number the map wrote; what that number names, whether it can be
// evaluated, and which node a trigger's slot points at are the binder's
// questions, one tier up.

import (
	"encoding/binary"
	"fmt"
)

// ScriptNode is one 796-byte type-7 node — an action or a condition, which have
// the same record and are told apart only by which array they were read from
// (ALM-TRIG-045).
//
// Label is the map author's own text, decoded through ScriptLabelText from a
// fixed 64-byte buffer whose tail is uninitialised editor memory, so only the
// bytes before the first NUL are read. It names nothing the engine dispatches on: the
// runtime dispatches on Opcode, and a node whose label disagrees with its opcode
// is still built and still measures what its opcode says.
//
// ID is the node's identity within its own array and is what a trigger's slots
// hold — NOT an array index. The two readings differ on the 16 shipped maps
// whose action ids are not dense, and only the id reading resolves all of them.
//
// Value and Type are stored BY PARAMETER SLOT and are not packed: an unused slot
// ahead of a used one is a shape the shipped corpus carries 804 times, so a
// consumer that compacted them would read a node's parameters at the wrong
// subscripts. Type[i] is the parameter's type code and Value[i] its value; what
// each code names is the binder's business (ALM-TRIG-046).
//
// The ten 64-byte parameter NAMES at +0x09c are read by nothing here. They are
// the editor's own labels for the slots and no consumer of this package needs
// them; leaving them out is what keeps a node 92 bytes in memory rather than
// 796.
type ScriptNode struct {
	Label  string
	Opcode uint32
	ID     uint32
	Value  [ScriptParams]uint32
	Type   [ScriptParams]uint32
}

// ScriptTrigger is one 184-byte type-7 trigger record: the thing that binds
// conditions to actions (ALM-TRIG-047).
//
// Left, Right and Cmp are three PAIRS with one comparison code each, not a flat
// list of six ids: over the shipped corpus 1263 of 1263 pairs are both-set or
// both-clear and none is half-empty. A pair whose two ids are zero is unused.
//
// Left[i] and Right[i] hold a condition node's ID and Acts[j] an action node's
// ID, both by identity and not by index.
//
// Cmp[i] is the pair's comparison code. The alphabet the shipped corpus uses is
// 0..5, and one shipped pair carries 0xffffffff — so this field is carried as
// the word the file holds rather than as a validated enum, and what a code out
// of alphabet does is the runtime's answer, not this decoder's.
//
// Once is the +0xb4 word: nonzero means the trigger fires at most once. It is
// 0 on the single trigger of every standalone map and 1 on 387 of the 421
// shipped triggers.
//
// The 64 bytes at +0x40 are read by nothing here: they are never a printable
// string on any of the 421 shipped triggers and carry editor heap addresses.
//
// The slots are slices so one value serves the record read and the record
// written: Script fills ScriptPairs pairs and ScriptActionRefs actions, every
// slot of the record, and a value built to be written may hold fewer, the
// rest of the record's slots being empty.
//
// Group is pair k's group in the Againrom script record, where a trigger
// holds when any group of pairs all hold. It is nil for a type-7 trigger,
// whose pairs are the one group 0.
type ScriptTrigger struct {
	Name  string
	Left  []uint32
	Right []uint32
	Cmp   []uint32
	Acts  []uint32
	Once  uint32
	Group []uint32
}

// GroupOf is pair k's group, 0 past the groups t spells.
func (t ScriptTrigger) GroupOf(k int) uint32 { return slot(t.Group, k) }

// Pairs is how many condition pairs t spells: the longest of Left, Right and
// Cmp.
func (t ScriptTrigger) Pairs() int { return max(len(t.Left), len(t.Right), len(t.Cmp)) }

// Pair is pair k's left and right condition ids and comparison code, zero
// past the slots t spells.
func (t ScriptTrigger) Pair(k int) (left, right, cmp uint32) {
	return slot(t.Left, k), slot(t.Right, k), slot(t.Cmp, k)
}

// Act is action slot k's id, zero past the slots t spells.
func (t ScriptTrigger) Act(k int) uint32 { return slot(t.Acts, k) }

func slot(s []uint32, k int) uint32 {
	if k < 0 || k >= len(s) {
		return 0
	}
	return s[k]
}

// Script is a map's decoded type-7 payload: the two node arrays and the trigger
// array that binds them.
//
// A map carrying no type-7 record decodes to a Script with three empty arrays
// and no error — the absent record is the skipped arm of the loader's own rule
// and not a malformed map.
//
// Extended is set when the triggers come from the Againrom script record:
// they then run on the extended machine, with Variables variables of their
// own.
type Script struct {
	Actions    []ScriptNode
	Conditions []ScriptNode
	Triggers   []ScriptTrigger
	Extended   bool
	Variables  int
}

// Empty reports whether the script holds nothing at all: no action, no condition
// and no trigger. It exists so a consumer can tell "this map authored no script"
// from "this map's script was decoded" without three length tests that could
// come to disagree.
func (s Script) Empty() bool {
	return len(s.Actions) == 0 && len(s.Conditions) == 0 && len(s.Triggers) == 0
}

// Script decodes the type-7 payload into its three arrays.
//
// It is a METHOD ON THE DECODED MAP and not a second entry point, so a caller
// cannot ask for the leaf grammar of bytes this package never accepted as a map.
// It reads Triggers.EntryCount and Triggers.Body, which alm.go preserved raw, and
// it consumes them EXACTLY: a walk that ends anywhere but at the end of the body
// is an error, because the three arrays tile the whole payload on every shipped
// map and a short walk means the model is wrong rather than that the map has a
// tail.
//
// A map with no type-7 record yields an empty Script and no error. That is not
// the same as an error, and Present(7) is what tells a caller which of the two
// it had — exactly as it does for every other absent record.
//
// A map carrying the Againrom script record takes its triggers from it and is
// Extended. Its type-7 record then holds the nodes alone: a type-7 trigger
// beside the record, or a second record, is a *ScriptRecordError.
func (m *Map) Script() (Script, error) {
	var s Script
	if m.Present(TypeScript) {
		var err error
		if s, err = m.Triggers.Decode(); err != nil {
			return Script{}, err
		}
	}
	if m.scriptRecords == 0 {
		return s, nil
	}
	if m.scriptRecords > 1 {
		return Script{}, recordRefusal(-1, -1, "records", 1, int64(m.scriptRecords))
	}
	if len(s.Triggers) != 0 {
		return Script{}, recordRefusal(-1, -1, "type-7 trigger count", 0, int64(len(s.Triggers)))
	}
	rec, err := DecodeScriptRecord(m.scriptRecord)
	if err != nil {
		return Script{}, err
	}
	s.Triggers, s.Extended, s.Variables = rec.Triggers, true, rec.Variables
	return s, nil
}

// Decode decodes the payload t holds into its three arrays, on Script's
// terms but without asking whether the map carried the record: an absent
// record's zero Triggers does not tile and is an error here.
func (t Triggers) Decode() (Script, error) {
	// The first count word is EntryCount, which alm.go read off the payload's
	// own +0x00; Body is everything after it, so a payload offset is a Body
	// offset plus one count word.
	b := t.Body
	f, err := frameScript(t.EntryCount, b)
	if err != nil {
		return Script{}, err
	}
	acts, _, err := readScriptNodes(b, f.Actions.First-CountWordSize, int64(f.Actions.N), "action")
	if err != nil {
		return Script{}, err
	}
	conds, _, err := readScriptNodes(b, f.Conditions.First-CountWordSize, int64(f.Conditions.N), "condition")
	if err != nil {
		return Script{}, err
	}
	trgs, _, err := readScriptTriggers(b, f.Triggers.First-CountWordSize, int64(f.Triggers.N))
	if err != nil {
		return Script{}, err
	}
	return Script{Actions: acts, Conditions: conds, Triggers: trgs}, nil
}

// ScriptArray is where one counted array of the type-7 payload sits. CountAt
// and First are payload offsets, from the payload's first byte (the action
// count word): its count word's and its first record's. N is the record
// count and Size the record size.
type ScriptArray struct {
	CountAt, First, N, Size int
}

// End is the payload offset just past the array's last record.
func (a ScriptArray) End() int { return a.First + a.N*a.Size }

// ScriptFrame is the type-7 payload's three counted arrays, in file order.
type ScriptFrame struct {
	Actions, Conditions, Triggers ScriptArray
}

// Arrays is the frame's arrays in file order: 0 actions, 1 conditions, 2
// triggers.
func (f ScriptFrame) Arrays() [3]ScriptArray {
	return [3]ScriptArray{f.Actions, f.Conditions, f.Triggers}
}

// ScriptFrameError is a type-7 payload the three counted arrays do not tile.
// Array is the array the walk stopped at (0 actions, 1 conditions, 2
// triggers), or -1 when the arrays end before the payload does. Short means
// the payload ends before the array's count word, with Remain bytes left at
// it; otherwise the array declares Count records of Size bytes and Remain
// bytes follow its count word. End and Len are the payload offset the arrays
// end at and the payload's length.
type ScriptFrameError struct {
	Array        int
	Short        bool
	Count        int64
	Size, Remain int
	End, Len     int
}

// Error words the refusal in Body offsets, the bytes after the first count
// word, as Script has always reported it.
func (e *ScriptFrameError) Error() string {
	what := [3]string{"action", "condition", "trigger"}
	switch {
	case e.Array < 0:
		return fmt.Sprintf("alm: type7 walk consumed %d of %d body bytes", e.End-CountWordSize, e.Len-CountWordSize)
	case e.Short:
		return fmt.Sprintf("alm: type7 body has %d byte(s) left at the %s count word, want %d", e.Remain, what[e.Array], CountWordSize)
	case e.Array == 2:
		return fmt.Sprintf("alm: type7 declares %d trigger(s), which are %d byte(s), and %d remain", e.Count, e.Count*int64(e.Size), e.Remain)
	}
	return fmt.Sprintf("alm: type7 declares %d %s node(s), which are %d byte(s), and %d remain",
		e.Count, what[e.Array], e.Count*int64(e.Size), e.Remain)
}

// FrameScript locates the three counted arrays of a whole type-7 payload,
// its first count word included. It is the one walk of the type-7 framing:
// Script decodes through it and the map editor patches records at its
// offsets. A payload the arrays do not tile exactly is a *ScriptFrameError.
func FrameScript(payload []byte) (ScriptFrame, error) {
	if len(payload) < CountWordSize {
		return ScriptFrame{}, &ScriptFrameError{Array: 0, Short: true, Remain: len(payload), Len: len(payload)}
	}
	return frameScript(binary.LittleEndian.Uint32(payload), payload[CountWordSize:])
}

// frameScript is FrameScript over a payload given as its first count word
// and the body after it, the shape Triggers holds.
//
// Each span is multiplied out in int64 before it is compared, so a declared
// count near the top of its width cannot wrap into a small plausible span
// that the buffer then satisfies. The arrays must end exactly at the end of
// the body: they tile the whole payload on every shipped map, and a short
// walk means the model is wrong rather than that the map has a tail.
func frameScript(first uint32, body []byte) (ScriptFrame, error) {
	var f ScriptFrame
	arrays := [3]*ScriptArray{&f.Actions, &f.Conditions, &f.Triggers}
	sizes := [3]int{ScriptNodeSize, ScriptNodeSize, ScriptTriggerSize}
	total := len(body) + CountWordSize
	at := 0
	for k, a := range arrays {
		n := int64(first)
		if k > 0 {
			if total-at < CountWordSize {
				return f, &ScriptFrameError{Array: k, Short: true, Remain: total - at, Len: total}
			}
			n = int64(binary.LittleEndian.Uint32(body[at-CountWordSize:]))
		}
		start := at + CountWordSize
		if n*int64(sizes[k]) > int64(total-start) {
			return f, &ScriptFrameError{Array: k, Count: n, Size: sizes[k], Remain: total - start, Len: total}
		}
		*a = ScriptArray{CountAt: at, First: start, N: int(n), Size: sizes[k]}
		at = a.End()
	}
	if at != total {
		return f, &ScriptFrameError{Array: -1, End: at, Len: total}
	}
	return f, nil
}

// readScriptNodes reads n fixed-width node records from b at off.
//
// The span is multiplied out in int64 before it is compared, so a declared count
// near the top of its width cannot wrap into a small plausible span that the
// buffer then satisfies. That check is what turns a wrong count into a refusal
// here rather than into a walk that reads a trigger record as a node.
func readScriptNodes(b []byte, off int, n int64, what string) ([]ScriptNode, int, error) {
	span := n * ScriptNodeSize
	if n < 0 || span > int64(len(b)-off) {
		return nil, off, fmt.Errorf("alm: type7 declares %d %s node(s), which are %d byte(s), and %d remain",
			n, what, span, len(b)-off)
	}
	out := make([]ScriptNode, n)
	for i := range out {
		rec := b[off+i*ScriptNodeSize : off+(i+1)*ScriptNodeSize]
		node := ScriptNode{
			Label:  ScriptLabelText.Decode(rec[:ScriptLabelLen]),
			Opcode: binary.LittleEndian.Uint32(rec[ScriptNodeOpcode:]),
			ID:     binary.LittleEndian.Uint32(rec[ScriptNodeID:]),
		}
		for k := 0; k < ScriptParams; k++ {
			node.Value[k] = binary.LittleEndian.Uint32(rec[ScriptNodeValues+4*k:])
			node.Type[k] = binary.LittleEndian.Uint32(rec[ScriptNodeTypes+4*k:])
		}
		out[i] = node
	}
	return out, off + int(span), nil
}

// readScriptTriggers reads n fixed-width trigger records from b at off, on the
// same terms as readScriptNodes.
func readScriptTriggers(b []byte, off int, n int64) ([]ScriptTrigger, int, error) {
	span := n * ScriptTriggerSize
	if n < 0 || span > int64(len(b)-off) {
		return nil, off, fmt.Errorf("alm: type7 declares %d trigger(s), which are %d byte(s), and %d remain",
			n, span, len(b)-off)
	}
	out := make([]ScriptTrigger, n)
	for i := range out {
		rec := b[off+i*ScriptTriggerSize : off+(i+1)*ScriptTriggerSize]
		w := make([]uint32, 3*ScriptPairs+ScriptActionRefs)
		t := ScriptTrigger{
			Name:  ScriptLabelText.Decode(rec[:ScriptLabelLen]),
			Left:  w[0:ScriptPairs:ScriptPairs],
			Right: w[ScriptPairs : 2*ScriptPairs : 2*ScriptPairs],
			Cmp:   w[2*ScriptPairs : 3*ScriptPairs : 3*ScriptPairs],
			Acts:  w[3*ScriptPairs:],
			Once:  binary.LittleEndian.Uint32(rec[ScriptTriggerOnce:]),
		}
		for k := 0; k < ScriptPairs; k++ {
			t.Left[k] = binary.LittleEndian.Uint32(rec[ScriptTriggerPairs+8*k:])
			t.Right[k] = binary.LittleEndian.Uint32(rec[ScriptTriggerPairs+4+8*k:])
			t.Cmp[k] = binary.LittleEndian.Uint32(rec[ScriptTriggerCompare+4*k:])
		}
		for k := 0; k < ScriptActionRefs; k++ {
			t.Acts[k] = binary.LittleEndian.Uint32(rec[ScriptTriggerActions+4*k:])
		}
		out[i] = t
	}
	return out, off + int(span), nil
}
