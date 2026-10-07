package sav

import (
	"bytes"
	"testing"
)

// TestTheLayoutIsTheDecoder: a class's Members slice is what reads its record,
// so changing the slice changes the decode and nothing else has to move. This is
// the property the eleven member layouts still to arrive will plug into.
func TestTheLayoutIsTheDecoder(t *testing.T) {
	c := Class{Name: "T", Members: []Member{
		{Name: "n", Kind: KindCString},
		{Name: "a", Kind: KindU8},
		{Name: "b", Kind: KindU16},
		{Name: "c", Kind: KindU32},
		{Name: "raw", Kind: KindRaw, Len: 3},
		{Name: "x", Kind: KindU32Obfuscated},
		{Name: "s", Kind: KindU16Saturated},
	}}
	var b []byte
	b = append(b, 2, 'h', 'i', 0x7f)
	le16(&b, 0x1234)
	le32(&b, 0xdeadbeef)
	b = append(b, 9, 8, 7)
	le32(&b, 4242^obfuscator)
	le16(&b, 0x0100)
	f, err := c.decode(b, 0)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if f.Text["n"] != "hi" || f.Value["a"] != 0x7f || f.Value["b"] != 0x1234 ||
		f.Value["c"] != 0xdeadbeef || f.Value["x"] != 4242 || f.Value["s"] != 0x0100 {
		t.Fatalf("decoded %+v", f)
	}
	if !bytes.Equal(f.Raw["raw"], []byte{9, 8, 7}) {
		t.Fatalf("raw member reads % x", f.Raw["raw"])
	}
	if f.End != len(b) {
		t.Fatalf("record ends at %d, want %d", f.End, len(b))
	}
	// A CString shifts every offset after it, which is exactly what a tabled
	// offset cannot express and a layout does for free.
	if f.Off["a"] != 3 {
		t.Fatalf("the byte after a two-character name is at %d", f.Off["a"])
	}
}

func TestLayoutWritesThroughTheSameOffsets(t *testing.T) {
	c := Classes["Player"]
	f := open(t, standard())
	p := f.Players[0]
	if err := c.set(f.Body, p.Fields, "Money", 999); err != nil {
		t.Fatalf("set: %v", err)
	}
	// The involution is the member's own, so a plain read of those four bytes
	// must NOT be 999.
	if u32(f.Body, p.Fields.Off["Money"]) == 999 {
		t.Fatal("money was written without its obfuscation")
	}
	back, err := Open(f.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	if back.Players[0].Money != 999 {
		t.Fatalf("money reads %d", back.Players[0].Money)
	}
}

func TestLayoutRefusesWhatItCannotDo(t *testing.T) {
	c := Classes["Player"]
	f := open(t, standard())
	p := f.Players[0]
	if err := c.set(f.Body, p.Fields, "NoSuchMember", 1); err == nil {
		t.Fatal("a member the class does not carry was written")
	}
	if err := c.set(f.Body, p.Fields, "Outcome", 300); err == nil {
		t.Fatal("a value too wide for its member was written")
	}
	if err := c.set(f.Body, p.Fields, "Name", 1); err == nil {
		t.Fatal("a CString was written as an integer")
	}
	// A class with no decoded layout says so rather than answering nothing.
	if _, err := Classes["Unit"].decode(f.Body, 0); err == nil {
		t.Fatal("a class with no member layout decoded a record")
	}
	// And a record that runs off the end is an error, not a partial value.
	short := Class{Name: "T", Members: []Member{{Name: "a", Kind: KindU32}}}
	if _, err := short.decode([]byte{1, 2}, 0); err == nil {
		t.Fatal("a record past the end of the stream decoded")
	}
	if _, err := short.decode([]byte{1, 2, 3, 4}, 3); err == nil {
		t.Fatal("a record starting near the end decoded")
	}
}

// TestTheClassSetIsNotTheCorpusSet. Four shipped saves introduce eleven classes;
// the image carries twenty-eight, and one save already in the corpus introduces a
// twelfth. A reader that knew only eleven would fail on any save carrying a
// spellbook and, by construction, on one from a mission with a shop.
func TestTheClassSetIsNotTheCorpusSet(t *testing.T) {
	observed := []string{"Player", "Human", "Weapon", "Item", "Armor", "Diary",
		"Effect", "Shield", "Unit", "Building", "Sack", "Spell"}
	for _, n := range observed {
		c, ok := Classes[n]
		if !ok {
			t.Fatalf("class %q, which a corpus save introduces, is missing from the table", n)
		}
		if c.Name != n {
			t.Fatalf("class %q names itself %q", n, c.Name)
		}
	}
	// Classes nothing has yet been seen to introduce, named by the image's own
	// descriptors, so that meeting one is a known class with no programme.
	for _, n := range []string{"Shop", "Tavern", "Outpost", "Spellbook", "Humanoid"} {
		if _, ok := Classes[n]; !ok {
			t.Fatalf("class %q is missing from the table", n)
		}
	}
	if len(Classes) < len(observed) {
		t.Fatalf("%d classes in the table, fewer than the %d observed", len(Classes), len(observed))
	}
}

// TestLookupDegradesOnAnUnknownName: seventeen of the twenty-eight classes have
// no programme, so an unread class is the NORMAL case. A name the table does not
// carry must become one of those rather than a second failure mode.
func TestLookupDegradesOnAnUnknownName(t *testing.T) {
	c := Lookup("CMultiShopSomethingNobodyHasRead")
	if c.Name != "CMultiShopSomethingNobodyHasRead" || c.Extent != ExtentUnread {
		t.Fatalf("an unknown name became %+v", c)
	}
	if _, ok := c.Fixed(); ok {
		t.Fatal("an unknown class claimed a fixed record length")
	}
	if _, err := c.decode([]byte{1, 2, 3, 4}, 0); err == nil {
		t.Fatal("an unknown class decoded a record")
	}
	if _, err := f0().Chain("CMultiShopSomethingNobodyHasRead"); err == nil {
		t.Fatal("an unknown class was chained")
	}
}

// TestTheHeadIsThirtySevenBytes, one routine, and the fields the corpus reading
// named are a PREFIX of it rather than the whole.
func TestTheHeadIsThirtySevenBytes(t *testing.T) {
	n, ok := Lookup("Token").Fixed()
	if !ok || n != TokenLen || TokenLen != 37 {
		t.Fatalf("the Token head is %d bytes (fixed=%v)", n, ok)
	}
	if n, ok := Lookup("Building").Fixed(); !ok || n != 77 {
		t.Fatalf("a Building record is %d bytes (fixed=%v), want 77", n, ok)
	}
	if n, ok := Lookup("Effect").Fixed(); !ok || n != 44 {
		t.Fatalf("an Effect record is %d bytes (fixed=%v), want 44", n, ok)
	}
}

// TestHumanIsHumanoidAndHumanoidIsWhereTheBytesAre.
//
// This test exists because the fact it pins was WRONG in this table for one
// round. An earlier research round read eleven Serialize bodies and attributed
// Humanoid's output to Human, giving "a Human record is a Unit record plus 24
// bytes"; this package carried that as Human's row. Humanoid is a twelfth body
// and writes the 24 bytes AND THIRTEEN OBJECT REFERENCES, so the old reading is
// short by thirteen references — a walk built on it desynchronises eight bytes
// into the first Human of a save and never recovers.
//
// The shape of the correction is what the test guards: Human writes nothing of
// its own, Humanoid is a class in its own right, and neither has a length.
func TestHumanIsHumanoidAndHumanoidIsWhereTheBytesAre(t *testing.T) {
	h, hm, u := Lookup("Human"), Lookup("Humanoid"), Lookup("Unit")
	if h.Base != "Humanoid" {
		t.Fatalf("Human's base is %q, want Humanoid", h.Base)
	}
	if h.Extent != 0 {
		t.Fatalf("Human writes %d bytes of its own, want 0", h.Extent)
	}
	if hm.Base != "Unit" {
		t.Fatalf("Humanoid's base is %q, want Unit", hm.Base)
	}
	// NOT 24. The old value is the exact thing this test is here to refuse:
	// Humanoid's own part is 24 bytes and thirteen object references, and an
	// object reference has no constant width.
	if hm.Extent == 24 {
		t.Fatal("Humanoid's extent is 24 — that is the retracted \"Unit + 24\" reading")
	}
	if hm.Extent != ExtentComputable {
		t.Fatalf("Humanoid's extent is %d, want ExtentComputable", hm.Extent)
	}
	for _, c := range []Class{h, hm, u} {
		if _, ok := c.Fixed(); ok {
			t.Fatalf("%s claims a fixed record length", c.Name)
		}
	}
}

// TestUnitHasNoLengthAndThatIsTheANSWER, not a gap. Every embedded class is now
// read and the programme still holds four counted lists, a CString, three object
// references and two presence flags. A row that wanted a constant for Unit would
// be the thing to change, so there is none — and Unit must no longer read as
// "nobody has read this", which is a different and now false statement.
func TestUnitHasNoLengthAndThatIsTheAnswer(t *testing.T) {
	u := Lookup("Unit")
	if u.Extent != ExtentComputable {
		t.Fatalf("Unit's extent is %d, want ExtentComputable", u.Extent)
	}
	for _, n := range []string{"Diary", "Spellbook", "Spell", "CDWordArray", "CWordArray"} {
		if Lookup(n).Extent == ExtentUnread {
			t.Fatalf("%s still reads as unread, but its class was resolved", n)
		}
	}
}

// TestAFixedLengthIsNotEnoughToChain. Effect is exactly 44 bytes and cannot be
// chained, because an Effect is an element of another object's counted list.
// Length and consecutiveness are two facts and the table carries both.
func TestAFixedLengthIsNotEnoughToChain(t *testing.T) {
	e := Lookup("Effect")
	n, ok := e.Fixed()
	if !ok || n != 44 {
		t.Fatalf("an Effect record is %d bytes (fixed=%v), want 44", n, ok)
	}
	if e.Consecutive {
		t.Fatal("Effect is marked consecutive")
	}
	if !Lookup("Building").Consecutive {
		t.Fatal("Building is not marked consecutive")
	}
	f := open(t, withBuildings(3))
	if _, err := f.Chain("Effect"); err == nil {
		t.Fatal("Effect was chained")
	}
}

// TestADerivedClassCarriesItsBaseProgramme: Shield is an Item plus 22 bytes, and
// that is one row of the table rather than a comment somebody keeps true. Item's
// own length depends on a counted list, so neither can be walked yet — which is
// what the test asserts, and it is the row that changes when Item is decoded.
func TestADerivedClassCarriesItsBaseProgramme(t *testing.T) {
	for name, base := range map[string]string{
		"Shield": "Item", "Armor": "Item", "Weapon": "Item",
		"Outpost": "Building", "Tavern": "Building", "Shop": "Building",
	} {
		if got := Lookup(name).Base; got != base {
			t.Fatalf("%s derives from %q, want %q", name, got, base)
		}
	}
	if _, err := Lookup("Shield").decode(make([]byte, 400), 0); err == nil {
		t.Fatal("Shield decoded through an Item whose list is not walked")
	}
	// Building IS walkable, so a class derived from it fails only on its own
	// unread part — the distinction the table has to be able to express.
	if _, ok := Lookup("Building").Fixed(); !ok {
		t.Fatal("Building is not walkable")
	}
	if _, ok := Lookup("Shop").Fixed(); ok {
		t.Fatal("Shop, whose own members are unread, claimed a length")
	}
}

// TestObfuscationIsPlayersAlone. The XOR and the clamp occur at four sites each
// and all eight are inside Player::Serialize; for every other class the bytes in
// the file are the bytes in memory. The census that says so is graded Medium and
// is blind to a field already held obfuscated in memory, so this is a check that
// the machinery has not SPREAD, not a proof that no other field is transformed.
func TestObfuscationIsPlayersAlone(t *testing.T) {
	for name, c := range Classes {
		members, ok := c.programme(0)
		if !ok {
			continue
		}
		for _, m := range members {
			if m.Kind != KindU32Obfuscated && m.Kind != KindU16Saturated {
				continue
			}
			if name != "Player" {
				t.Fatalf("class %s carries %s as an obfuscated or clamped field", name, m.Name)
			}
		}
	}
}

func f0() *File { return &File{Body: make([]byte, 64)} }
