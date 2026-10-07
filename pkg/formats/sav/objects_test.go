package sav

import "testing"

func withBuildings(n int) fixture {
	s := standard()
	s.buildings = n
	return s
}

func TestClassRecordsAreLocated(t *testing.T) {
	f := open(t, withBuildings(5))
	var names []string
	for _, r := range f.ClassRecords() {
		names = append(names, r.Name)
	}
	if len(names) != 3 || names[0] != "Player" || names[1] != "Human" || names[2] != "Building" {
		t.Fatalf("class records read %v", names)
	}
	for _, r := range f.ClassRecords() {
		if r.Schema != 1 || r.First <= r.Off {
			t.Fatalf("class record %+v", r)
		}
	}
}

// TestChainWalksAFixedLengthClass. This is a WALK and not a scan: 77 bytes a
// step, the tag learned from the first step and then required, and a 0x0000
// terminator that says the run was consumed exactly.
func TestChainWalksAFixedLengthClass(t *testing.T) {
	f := open(t, withBuildings(6))
	objs, err := f.Chain("Building")
	if err != nil {
		t.Fatalf("Chain: %v", err)
	}
	if len(objs) != 6 {
		t.Fatalf("%d instances, want 6", len(objs))
	}
	for i, o := range objs {
		if o.Fields.End-o.Fields.Start != 77 {
			t.Fatalf("instance %d is %d bytes", i, o.Fields.End-o.Fields.Start)
		}
		if got := o.Fields.Value["RuntimeID"]; got != uint32(i+2) {
			t.Fatalf("instance %d creation order %d", i, got)
		}
		// The map unit id is the LOW u16 of the head's dword at +19 — the
		// same offset the corpus reading called head+0x13.
		if got := o.Fields.Value["T08"] & 0xffff; got != uint32(i+1) {
			t.Fatalf("instance %d map unit id %d", i, got)
		}
		if o.Fields.Off["Identity"] != o.Off+29 {
			t.Fatalf("the identity key sits at +%d", o.Fields.Off["Identity"]-o.Off)
		}
	}
	// Identity keys are the writing process's own addresses, so they are
	// distinct by construction and a decode that read them from the wrong
	// place would collide.
	seen := map[uint32]bool{}
	for _, o := range objs {
		k := o.Fields.Value["Identity"]
		if k == 0 || seen[k] {
			t.Fatalf("identity key %#x is zero or repeated", k)
		}
		seen[k] = true
	}
}

// TestReencodeIsTheCheckAByteCarryingRoundTripCannotMake. Marshal proves nothing
// was disturbed; this proves what was read was understood.
func TestReencodeFromTheDecodedForm(t *testing.T) {
	f := open(t, withBuildings(4))
	objs, err := f.Chain("Building")
	if err != nil {
		t.Fatal(err)
	}
	for i, o := range objs {
		ok, at, err := f.Reencode(o)
		if err != nil {
			t.Fatalf("instance %d: %v", i, err)
		}
		if !ok {
			t.Fatalf("instance %d re-encodes differently at %#x", i, at)
		}
	}
	// TO LEARN WHETHER THIS IS WITNESSED, BREAK THE LAYOUT: a member read at
	// the wrong width survives Marshal, which moves no bytes, and must not
	// survive this.
	broken := Class{Name: "Building", Head: true, Extent: 40, Members: []Member{
		{Name: "B52", Kind: KindRaw, Len: 22},
		{Name: "B40", Kind: KindU8},
		{Name: "B42", Kind: KindU16},
		{Name: "B44", Kind: KindU16},
		{Name: "B46", Kind: KindU8}, // was u16
		{Name: "B48", Kind: KindU8},
		{Name: "B60", Kind: KindU8},
		{Name: "B61", Kind: KindU8},
		{Name: "B64", Kind: KindU32},
		{Name: "B68", Kind: KindU32},
	}}
	fields, err := broken.decode(f.Body, objs[0].Off)
	if err != nil {
		t.Fatal(err)
	}
	got, err := broken.encode(fields)
	if err != nil {
		t.Fatal(err)
	}
	want := f.Body[objs[0].Fields.Start:objs[0].Fields.End]
	if len(got) == len(want) {
		t.Fatal("a member read one byte narrow still produced a record of the right length")
	}
}

func TestChainRefusesWhatIsNotAChain(t *testing.T) {
	// A class with no fixed length, and one the save does not introduce.
	f := open(t, withBuildings(3))
	for _, name := range []string{"Unit", "Human", "Item", "Sack", "Diary"} {
		if _, err := f.Chain(name); err == nil {
			t.Fatalf("%s was chained despite having no fixed record length", name)
		}
	}
	if _, err := open(t, standard()).Chain("Building"); err == nil {
		t.Fatal("a save introducing no Building was chained")
	}
	// A tag that changes mid-run stops the walk rather than being absorbed.
	s := withBuildings(4)
	whole := s.file()
	f2, err := Open(whole)
	if err != nil {
		t.Fatal(err)
	}
	objs, err := f2.Chain("Building")
	if err != nil {
		t.Fatal(err)
	}
	put16(f2.Body, objs[1].Off-2, 0x8099)
	if _, err := f2.Chain("Building"); err == nil {
		t.Fatal("a run whose tag changed mid-way was chained")
	}
	// And a run with no terminator is refused rather than walked off the end.
	f3, err := Open(s.file())
	if err != nil {
		t.Fatal(err)
	}
	f3.Body = f3.Body[:objs[3].Fields.End]
	if _, err := f3.Chain("Building"); err == nil {
		t.Fatal("a run with no terminator was chained")
	}
}

func TestReencodeRefusesAClassWithNoProgramme(t *testing.T) {
	f := open(t, withBuildings(2))
	if _, _, err := f.Reencode(Object{Class: "Unit"}); err == nil {
		t.Fatal("a class with no programme was re-encoded")
	}
	if _, _, err := f.Reencode(Object{Class: "NobodyHasReadThis"}); err == nil {
		t.Fatal("an unknown class was re-encoded")
	}
}
