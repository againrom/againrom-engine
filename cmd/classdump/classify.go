package main

import (
	"strconv"

	"againrom/pkg/formats/alm"
)

// bucket is the set of override paths one placed type-6 record takes — one bit
// per path, the empty set meaning the record names its class itself.
//
// A record names its unit class by ClassID, a units.reg ID, but two other fields
// of the same record divert that lookup to a table this tool does not hold:
// DefID selects a definition when it is neither 0 nor 0xcdcdcdcd, and Flags bit 0
// selects the scenario/npc.reg path (ALM-CLS-038, ALM-UNIT-040). A diverted
// record is therefore not a failure to resolve — it is a reference to a table
// outside this story — which is why the sweep counts it apart instead of failing
// on it.
//
// The two conditions are evaluated **independently**, and that is the decision
// rather than an accident. Neither claim orders the two paths, so an
// `if npc … else if def …` ladder would invent an ordering research has not
// published, and would hide exactly the records where the two coincide behind
// whichever branch was written first — the figure a future research item needs to
// count. A bit set makes that ordering unwritable: bucketBoth is not a third
// test, it is the two bits at once.
type bucket uint8

const (
	bucketDirect bucket = 0      // neither override: ClassID names the class
	bucketNPC    bucket = 1 << 0 // Flags bit 0: the scenario/npc.reg path
	bucketDef    bucket = 1 << 1 // DefID: the definition collection
	bucketBoth   bucket = bucketNPC | bucketDef
)

// npcFlag is the Flags bit that selects the NPC path. The rest of the word is
// read but not interpreted here: no other bit takes part in this decision.
const npcFlag uint32 = 1 << 0

// The two DefID values that are not a definition id. 0 is the absent override
// and 0xcdcdcdcd is uninitialised-memory filler — both mean "no override", and
// every other value, however odd it looks, is one.
const (
	defIDNone   uint32 = 0
	defIDFiller uint32 = 0xcdcdcdcd
)

// unitRef is what the classifier decides about one record: which paths it takes,
// and its class key at the type a units.reg lookup takes.
type unitRef struct {
	Bucket  bucket
	ClassID int32
}

// classifyUnit is the whole classifier: a pure function of ClassID, Flags and
// DefID that resolves nothing, reads nothing and prints nothing.
//
// The key is widened int16 → int32 and the widening carries the sign, because the
// file word is read sign-extended (ALM-CLS-038). A negative key must reach the
// registry lookup at its signed value and miss *there* — and be reported at that
// value — rather than fold to ~65000 and miss for the wrong reason.
//
// ClassID is carried for every bucket, not only for direct records: a diverted
// record's key still says whether the class it names would have resolved anyway,
// which is what tells whether the divert is load-bearing.
func classifyUnit(u alm.Unit) unitRef {
	var b bucket
	if u.Flags&npcFlag != 0 {
		b |= bucketNPC
	}
	if u.DefID != defIDNone && u.DefID != defIDFiller {
		b |= bucketDef
	}
	return unitRef{Bucket: b, ClassID: int32(u.ClassID)}
}

// String is the fixed token a bucket's count is reported under, so the name and
// the bucket cannot drift apart. The default arm is unreachable for anything
// classifyUnit builds and exists so a third path added later shows up as a
// number instead of silently borrowing another bucket's name.
func (b bucket) String() string {
	switch b {
	case bucketDirect:
		return "direct"
	case bucketNPC:
		return "npc"
	case bucketDef:
		return "def"
	case bucketBoth:
		return "both"
	default:
		return "bucket(0x" + strconv.FormatUint(uint64(b), 16) + ")"
	}
}
