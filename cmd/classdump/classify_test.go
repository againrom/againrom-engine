package main

import (
	"testing"

	"againrom/pkg/formats/alm"
)

// The classifier alone (SC-9's first half): no archive, no registry, no map —
// the records are alm.Unit values written field by field, which is the whole
// input the decision has.
//
// The two sentinels and the flag bit are written as literals here rather than as
// the constants classify.go defines: a test that reuses the value under test
// agrees with any typo in it.

// classKey is the record's +0x08 word written as the file holds it — a u16 — and
// read the way pkg/formats/alm reads it, sign-extended (ALM-CLS-038). Going
// through the u16 is what makes the 0x8001 case pin the sign: were the decoded
// field ever to stop being signed, this would not compile into it.
func classKey(raw uint16) int16 { return int16(raw) }

// All four buckets, and the two DefID exclusions on both sides of the flag bit.
// The two "both" rows are the ones a precedence chain fails: an
// `if npc … else if def …` ladder reports npc for them, and the reverse ladder
// def, so either ordering is caught here rather than agreeing by luck.
func TestClassifyPutsARecordInOneOfFourIndependentBuckets(t *testing.T) {
	const (
		defID      = uint32(0x0000abcd) // a definition id: neither exclusion
		otherFlags = uint32(0x00000084) // flag bits that are not bit 0
	)
	tests := []struct {
		name string
		unit alm.Unit
		want bucket
	}{
		{"neither override", alm.Unit{ClassID: 7}, bucketDirect},
		{"DefID at the 0 exclusion", alm.Unit{ClassID: 7, DefID: 0}, bucketDirect},
		{"DefID at the 0xcdcdcdcd exclusion", alm.Unit{ClassID: 7, DefID: 0xcdcdcdcd}, bucketDirect},
		{"flag bits other than bit 0", alm.Unit{ClassID: 7, Flags: otherFlags}, bucketDirect},

		{"Flags bit 0", alm.Unit{ClassID: 7, Flags: 0x00000001}, bucketNPC},
		{"Flags bit 0 among other bits", alm.Unit{ClassID: 7, Flags: otherFlags | 1}, bucketNPC},
		{"Flags bit 0, DefID at 0", alm.Unit{ClassID: 7, Flags: 1, DefID: 0}, bucketNPC},
		{"Flags bit 0, DefID at 0xcdcdcdcd", alm.Unit{ClassID: 7, Flags: 1, DefID: 0xcdcdcdcd}, bucketNPC},

		{"DefID a definition id", alm.Unit{ClassID: 7, DefID: defID}, bucketDef},
		{"DefID 1, the smallest override", alm.Unit{ClassID: 7, DefID: 1}, bucketDef},
		{"DefID 0xcdcdcdce, one past the filler", alm.Unit{ClassID: 7, DefID: 0xcdcdcdce}, bucketDef},
		{"DefID an id, flag bits other than bit 0", alm.Unit{ClassID: 7, Flags: otherFlags, DefID: defID}, bucketDef},

		{"both conditions at once", alm.Unit{ClassID: 7, Flags: 1, DefID: defID}, bucketBoth},
		{"both, every other bit set too", alm.Unit{ClassID: 7, Flags: 0xffffffff, DefID: 0xffffffff}, bucketBoth},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyUnit(tc.unit).Bucket; got != tc.want {
				t.Errorf("classifyUnit(%+v).Bucket = %v, want %v", tc.unit, got, tc.want)
			}
		})
	}
}

// SC-9's signed-key case: a class key written 0x8001 must read -32767 and be
// carried at that value, so the registry lookup misses at the key the file gave.
// Read through a u16 it would be 32769 — also a miss, but for a reason the file
// never said, and reported as a value no record holds.
func TestClassifyWidensTheClassKeyWithItsSign(t *testing.T) {
	const raw uint16 = 0x8001
	const want int32 = -32767

	if got := classKey(raw); int32(got) != want {
		t.Fatalf("classKey(%#04x) = %d, want %d: the decoded class key is not signed", raw, got, want)
	}

	// Carried at the signed value on every path, not only the direct one: a
	// diverted record's key is what says whether it would have resolved anyway.
	for _, u := range []alm.Unit{
		{ClassID: classKey(raw)},
		{ClassID: classKey(raw), Flags: 1},
		{ClassID: classKey(raw), DefID: 0xabcd},
		{ClassID: classKey(raw), Flags: 1, DefID: 0xabcd},
	} {
		if got := classifyUnit(u).ClassID; got != want {
			t.Errorf("classifyUnit(%+v).ClassID = %d, want %d (%d would mean the widening went through a uint16)",
				u, got, want, uint32(raw))
		}
	}

	// And every other key widens unchanged, either sign.
	for _, k := range []int16{0, 1, 80, -1, -32768, 32767} {
		if got := classifyUnit(alm.Unit{ClassID: k}).ClassID; got != int32(k) {
			t.Errorf("classifyUnit(ClassID: %d).ClassID = %d, want %d", k, got, k)
		}
	}
}

// The decision is a function of ClassID, Flags and DefID and of nothing else the
// record carries, and it holds no state between calls.
func TestClassifyReadsOnlyTheThreeFields(t *testing.T) {
	base := alm.Unit{ClassID: 7, Flags: 1, DefID: 0xabcd}
	want := classifyUnit(base)

	noisy := base
	noisy.X, noisy.Y, noisy.ClassSubID = 0x0180, 0x0280, 0x0033
	if got := classifyUnit(noisy); got != want {
		t.Errorf("classifyUnit(%+v) = %+v, want %+v: the anchor or the secondary key changed the answer", noisy, got, want)
	}
	if got := classifyUnit(base); got != want {
		t.Errorf("classifyUnit repeated = %+v, want %+v", got, want)
	}
}

// The four tokens the buckets report under, pinned where they are defined.
func TestBucketTokens(t *testing.T) {
	for _, tc := range []struct {
		b    bucket
		want string
	}{
		{bucketDirect, "direct"},
		{bucketNPC, "npc"},
		{bucketDef, "def"},
		{bucketBoth, "both"},
	} {
		if got := tc.b.String(); got != tc.want {
			t.Errorf("bucket(%d).String() = %q, want %q", uint8(tc.b), got, tc.want)
		}
	}
}
