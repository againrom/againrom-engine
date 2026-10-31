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
type ScriptTrigger struct {
	Name  string
	Left  [ScriptPairs]uint32
	Right [ScriptPairs]uint32
	Cmp   [ScriptPairs]uint32
	Acts  [ScriptActionRefs]uint32
	Once  uint32
}

// Script is a map's decoded type-7 payload: the two node arrays and the trigger
// array that binds them.
//
// A map carrying no type-7 record decodes to a Script with three empty arrays
// and no error — the absent record is the skipped arm of the loader's own rule
// and not a malformed map.
type Script struct {
	Actions    []ScriptNode
	Conditions []ScriptNode
	Triggers   []ScriptTrigger
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
func (m *Map) Script() (Script, error) {
	if !m.Present(TypeScript) {
		return Script{}, nil
	}

	// The first count word is Triggers.EntryCount, which alm.go read off the
	// payload's own +0x00; Body is everything after it. So the action array
	// starts at Body[0] and the other two counts are read as the walk reaches
	// them.
	b := m.Triggers.Body
	off := 0

	acts, off, err := readScriptNodes(b, off, int64(m.Triggers.EntryCount), "action")
	if err != nil {
		return Script{}, err
	}

	nCond, off, err := readCount(b, off, "condition")
	if err != nil {
		return Script{}, err
	}
	conds, off, err := readScriptNodes(b, off, nCond, "condition")
	if err != nil {
		return Script{}, err
	}

	nTrg, off, err := readCount(b, off, "trigger")
	if err != nil {
		return Script{}, err
	}
	trgs, off, err := readScriptTriggers(b, off, nTrg)
	if err != nil {
		return Script{}, err
	}

	if off != len(b) {
		return Script{}, fmt.Errorf("alm: type7 walk consumed %d of %d body bytes", off, len(b))
	}
	return Script{Actions: acts, Conditions: conds, Triggers: trgs}, nil
}

// readCount reads one of the two count words the body carries inline.
func readCount(b []byte, off int, what string) (int64, int, error) {
	if len(b)-off < CountWordSize {
		return 0, off, fmt.Errorf("alm: type7 body has %d byte(s) left at the %s count word, want %d",
			len(b)-off, what, CountWordSize)
	}
	n := int64(binary.LittleEndian.Uint32(b[off : off+CountWordSize]))
	return n, off + CountWordSize, nil
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
		t := ScriptTrigger{
			Name: ScriptLabelText.Decode(rec[:ScriptLabelLen]),
			Once: binary.LittleEndian.Uint32(rec[ScriptTriggerOnce:]),
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
