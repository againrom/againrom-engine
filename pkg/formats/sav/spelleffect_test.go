package sav

import "testing"

// ---------------------------------------------------------------------------
// Every byte below is written from the serializer programmes in
// knowledge/formats/sav/format.md (SAV-CLASSSER-172..177, SAV-EFFECTGRAPH-366's
// own witnessed shape); nothing here is copied out of a game file, and no
// test in this package reads one (golden rule 2).
// ---------------------------------------------------------------------------

// spellEffectHead is one arbitrary, distinguishable Token head: every
// SpellEffect-family and Effect class's programme opens with Token's own 37
// bytes (stpClass "Token", directly or through Effect/SpellEffect), and
// nothing this package projects out of a SpellEffect graph reads a Token
// field, so the fixture only needs the head to be present and self-consistent.
func (s *stream) spellEffectHead(tag uint32) {
	s.token(0, 0, 0, tag, 0, 0)
}

func (s *stream) bareSpellEffect(se40, se41 uint8) {
	s.obj("SpellEffect")
	s.spellEffectHead(uint32(se40)<<8 | uint32(se41))
	s.u8(se40)
	s.u8(se41)
}

func (s *stream) pointEffect(se40, se41 uint8, effect func(*stream), pe44 uint32) {
	s.obj("PointEffect")
	s.spellEffectHead(pe44)
	s.u8(se40)
	s.u8(se41)
	if effect == nil {
		s.null()
	} else {
		effect(s)
	}
	s.u32(pe44)
}

func (s *stream) areaEffect(se40, se41 uint8, ae48 [4]byte, ae4c uint16, effect func(*stream)) {
	s.obj("AreaEffect")
	s.spellEffectHead(uint32(ae4c))
	s.u8(se40)
	s.u8(se41)
	s.b = append(s.b, ae48[:]...)
	s.u16(ae4c)
	if effect == nil {
		s.null()
	} else {
		effect(s)
	}
}

func (s *stream) spellTransport(se40, se41 uint8, target, area func(*stream), st4c uint16) {
	s.obj("SpellTransport")
	s.spellEffectHead(uint32(st4c))
	s.u8(se40)
	s.u8(se41)
	if target == nil {
		s.null()
	} else {
		target(s)
	}
	if area == nil {
		s.null()
	} else {
		area(s)
	}
	s.u16(st4c)
}

func (s *stream) effect(e3c, e3d uint8, e40 uint32, e0c uint8) {
	s.obj("Effect")
	s.spellEffectHead(e40)
	s.u8(e3c)
	s.u8(e3d)
	s.u32(e40)
	s.u8(e0c)
}

func (s *stream) effectDirectDamage(e3c, e3d uint8, e40 uint32, e0c uint8, raw [24]byte) {
	s.obj("Effect_DirectDamage")
	s.spellEffectHead(e40)
	s.u8(e3c)
	s.u8(e3d)
	s.u32(e40)
	s.u8(e0c)
	s.b = append(s.b, raw[:]...)
}

// sameSpellEffect deep-compares two SpellEffect values field by field. PE48,
// AE44, ST44 and ST48 are pointers a fresh decode always allocates anew, so
// comparing a SpellEffect with == or != would compare those addresses
// instead of what they point to; every test that checks a round trip needs
// this instead.
func sameSpellEffect(a, b SpellEffect) bool {
	return a.Off == b.Off && a.Class == b.Class && a.SE40 == b.SE40 && a.SE41 == b.SE41 &&
		a.PE44 == b.PE44 && a.AE48 == b.AE48 && a.AE4C == b.AE4C && a.ST4C == b.ST4C &&
		sameEffect(a.PE48, b.PE48) && sameEffect(a.AE44, b.AE44) &&
		sameSpellEffectPtr(a.ST44, b.ST44) && sameSpellEffectPtr(a.ST48, b.ST48)
}

func sameEffect(a, b *Effect) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	return a == nil || *a == *b
}

func sameSpellEffectPtr(a, b *SpellEffect) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	return a == nil || sameSpellEffect(*a, *b)
}

func directDamageBytes(fill byte) [24]byte {
	var b [24]byte
	for i := range b {
		b[i] = fill + byte(i)
	}
	return b
}

// TestSpellEffectsIsEmptyByDefault is the shape every fixture predating this
// story used (document.go, ground.go): a present world with a zero-count
// SpellEffect list is an empty typed list, not a missing one.
func TestSpellEffectsIsEmptyByDefault(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()}})
	effects, present, err := f.SpellEffects()
	if err != nil || !present {
		t.Fatalf("SpellEffects: present=%v err=%v", present, err)
	}
	if len(effects) != 0 {
		t.Fatalf("got %d effects, want 0", len(effects))
	}
}

// TestSpellEffectsRefusesAMissingWorld matches Buildings/GroundSacks/Cells'
// own no-world-half rule. The shared walk fixture always carries a world
// half, so this uses the other package fixture, which offers noWorld.
func TestSpellEffectsRefusesAMissingWorld(t *testing.T) {
	s := standard()
	s.noWorld = true
	s.mission = 0
	f := open(t, s)
	if _, present, err := f.SpellEffects(); err != nil || present {
		t.Fatalf("SpellEffects on a no-world save: present=%v err=%v", present, err)
	}
}

// TestSpellEffectGraphReadsEveryClassAndField constructs one top-level list
// exercising all four SpellEffect-family classes, both Effect classes, a
// null typed reference (SAV-EFFECTGRAPH-366's own witnessed AreaEffect arm
// under SpellTransport) and a nested nested reference (PointEffect under
// SpellTransport's own ST44), and checks every field SAV-CLASSSER-172..177
// name. Four consecutive records with nested content also locks each
// class's width: a wrong width desynchronises every record after the first,
// so reading record 3's known sentinel fields correctly proves records 0-2
// consumed exactly the bytes their own programmes claim.
func TestSpellEffectGraphReadsEveryClassAndField(t *testing.T) {
	damage := directDamageBytes(0x60)
	area48 := [4]byte{0x11, 0x22, 0x33, 0x44}
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()},
		spellEffects: func(s *stream) {
			s.u32(4)
			s.pointEffect(1, 2, func(s *stream) {
				s.effectDirectDamage(10, 11, 0xaabbccdd, 12, damage)
			}, 0xdeadbeef)
			s.spellTransport(3, 4,
				func(s *stream) { s.pointEffect(5, 6, nil, 0x11112222) },
				nil, // the null typed AreaEffect reference SAV-EFFECTGRAPH-366 witnesses
				0x3333)
			s.areaEffect(7, 8, area48, 0x4444, func(s *stream) {
				s.effect(20, 21, 0x99aabbcc, 22)
			})
			s.bareSpellEffect(9, 250)
		}})
	effects, present, err := f.SpellEffects()
	if err != nil || !present {
		t.Fatalf("SpellEffects: present=%v err=%v", present, err)
	}
	if len(effects) != 4 {
		t.Fatalf("got %d top-level effects, want 4", len(effects))
	}

	pe := effects[0]
	if pe.Class != "PointEffect" || pe.SE40 != 1 || pe.SE41 != 2 || pe.PE44 != 0xdeadbeef {
		t.Fatalf("record 0 (PointEffect) reads %+v", pe)
	}
	if pe.PE48 == nil || pe.PE48.Class != "Effect_DirectDamage" {
		t.Fatalf("record 0's PE48 is %+v, want a typed Effect_DirectDamage", pe.PE48)
	}
	if pe.PE48.E3C != 10 || pe.PE48.E3D != 11 || pe.PE48.E40 != 0xaabbccdd || pe.PE48.E0C != 12 {
		t.Fatalf("record 0's PE48 base fields read %+v", pe.PE48)
	}
	if pe.PE48.DirectDamage != damage {
		t.Fatalf("record 0's PE48 direct-damage bytes read %v, want %v", pe.PE48.DirectDamage, damage)
	}

	st := effects[1]
	if st.Class != "SpellTransport" || st.SE40 != 3 || st.SE41 != 4 || st.ST4C != 0x3333 {
		t.Fatalf("record 1 (SpellTransport) reads %+v", st)
	}
	if st.ST48 != nil {
		t.Fatalf("record 1's ST48 is %+v, want nil (the witnessed null AreaEffect arm)", st.ST48)
	}
	if st.ST44 == nil || st.ST44.Class != "PointEffect" || st.ST44.SE40 != 5 || st.ST44.SE41 != 6 ||
		st.ST44.PE44 != 0x11112222 || st.ST44.PE48 != nil {
		t.Fatalf("record 1's ST44 (nested PointEffect) reads %+v", st.ST44)
	}

	ae := effects[2]
	if ae.Class != "AreaEffect" || ae.SE40 != 7 || ae.SE41 != 8 || ae.AE48 != area48 || ae.AE4C != 0x4444 {
		t.Fatalf("record 2 (AreaEffect) reads %+v", ae)
	}
	if ae.AE44 == nil || ae.AE44.Class != "Effect" || ae.AE44.E3C != 20 || ae.AE44.E3D != 21 ||
		ae.AE44.E40 != 0x99aabbcc || ae.AE44.E0C != 22 {
		t.Fatalf("record 2's AE44 reads %+v", ae.AE44)
	}

	bare := effects[3]
	if bare.Class != "SpellEffect" || bare.SE40 != 9 || bare.SE41 != 250 {
		t.Fatalf("record 3 (bare SpellEffect) reads %+v, want the sentinel 9/250", bare)
	}
}

// TestSpellEffectSharedReferenceResolvesToTheSameContent covers a back
// reference: two PointEffect entries whose PE48 names the SAME archive
// object. SAV-CLASSSER-177 finds no corpus witness for this shape, but the
// wire grammar (SAV-STREAM-013) permits it and SetSpellEffects' own refSpan
// guard exists specifically to leave a shared child's bytes untouched on the
// second occurrence rather than misplace a write; this is that guard's only
// exercise.
func TestSpellEffectSharedReferenceResolvesToTheSameContent(t *testing.T) {
	var effIdx uint16
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()},
		spellEffects: func(s *stream) {
			s.u32(2)
			s.pointEffect(1, 1, func(s *stream) {
				// "Effect" is a fresh class here: obj() spends one index on
				// the class descriptor and the next on this instance, the
				// same two-slot cost PointEffect itself just paid.
				effIdx = s.next + 1
				s.effect(1, 2, 3, 4)
			}, 0x1000)
			s.obj("PointEffect")
			s.spellEffectHead(0x2000)
			s.u8(2)
			s.u8(2)
			s.backref(effIdx) // the Effect object PointEffect 0 just introduced
			s.u32(0x2000)
		}})
	effects, present, err := f.SpellEffects()
	if err != nil || !present {
		t.Fatalf("SpellEffects: present=%v err=%v", present, err)
	}
	if len(effects) != 2 {
		t.Fatalf("got %d effects, want 2", len(effects))
	}
	if effects[0].PE48 == nil || effects[1].PE48 == nil {
		t.Fatalf("a back-referenced PE48 decoded null: %+v / %+v", effects[0].PE48, effects[1].PE48)
	}
	if *effects[0].PE48 != *effects[1].PE48 {
		t.Fatalf("the shared Effect decoded to two different values: %+v vs %+v",
			effects[0].PE48, effects[1].PE48)
	}
	if effects[0].PE48.E3C != 1 || effects[0].PE48.E0C != 4 {
		t.Fatalf("the shared Effect reads %+v", effects[0].PE48)
	}

	// A round trip through SetSpellEffects must not corrupt the back-referenced
	// occurrence: it carries no bytes of its own to patch.
	if err := f.SetSpellEffects(effects); err != nil {
		t.Fatalf("SetSpellEffects: %v", err)
	}
	again, present, err := f.SpellEffects()
	if err != nil || !present {
		t.Fatalf("SpellEffects (re-decoded): present=%v err=%v", present, err)
	}
	if !sameSpellEffect(again[0], effects[0]) || !sameSpellEffect(again[1], effects[1]) {
		t.Fatalf("an unchanged SetSpellEffects round trip moved a field: got %+v, want %+v", again, effects)
	}
}

// TestSpellEffectRefusesASelfReferencingSpellTransport builds a SpellTransport
// whose own ST44 back-references itself. The wire grammar's back-reference
// arm (SAV-STREAM-013) permits naming any object already read, and the
// walker registers a record before running its own programme, so this is a
// valid *Record graph: exactDocument returns it without error, ST44 equal to
// the record itself. Before this fix, spellEffectFromRecord had no depth
// bound and this exact shape recursed the process into a stack overflow on
// both original LOAD doors (review finding F-2); this test requires a
// returned error instead.
func TestSpellEffectRefusesASelfReferencingSpellTransport(t *testing.T) {
	var stIdx uint16
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()},
		spellEffects: func(s *stream) {
			s.u32(1)
			// "SpellTransport" is a fresh class here: obj() spends one index
			// on the class descriptor and the next on this instance, the
			// same +1 TestSpellEffectSharedReferenceResolvesToTheSameContent
			// uses for its own back-referenced Effect.
			stIdx = s.next + 1
			s.obj("SpellTransport")
			s.spellEffectHead(0x9999)
			s.u8(1)
			s.u8(2)
			s.backref(stIdx) // ST44 names this same record
			s.null()         // ST48
			s.u16(0x9999)
		}})
	if _, _, err := f.SpellEffects(); err == nil {
		t.Fatal("a SpellTransport whose ST44 back-references itself was accepted instead of refused")
	}
}

// TestSetSpellEffectsRoundTripsEveryField writes new scalar values (never a
// new shape) over the graph TestSpellEffectGraphReadsEveryClassAndField
// builds, re-decodes, and confirms every field including the two nested
// Effect classes and the null AreaEffect arm.
func TestSetSpellEffectsRoundTripsEveryField(t *testing.T) {
	damage := directDamageBytes(0x60)
	area48 := [4]byte{0x11, 0x22, 0x33, 0x44}
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()},
		spellEffects: func(s *stream) {
			s.u32(3)
			s.pointEffect(1, 2, func(s *stream) {
				s.effectDirectDamage(10, 11, 0xaabbccdd, 12, damage)
			}, 0xdeadbeef)
			s.spellTransport(3, 4,
				func(s *stream) { s.pointEffect(5, 6, nil, 0x11112222) }, nil, 0x3333)
			s.areaEffect(7, 8, area48, 0x4444, func(s *stream) { s.effect(20, 21, 0x99aabbcc, 22) })
		}})
	before, present, err := f.SpellEffects()
	if err != nil || !present {
		t.Fatalf("SpellEffects: present=%v err=%v", present, err)
	}
	newDamage := directDamageBytes(0x90)
	want := append([]SpellEffect(nil), before...)
	want[0].SE40, want[0].SE41, want[0].PE44 = 111, 112, 0xcafef00d
	want[0].PE48.E3C, want[0].PE48.E0C, want[0].PE48.DirectDamage = 200, 201, newDamage
	want[1].ST4C = 0x7777
	want[1].ST44.PE44 = 0x99998888
	want[2].AE4C = 0x1212
	want[2].AE48 = [4]byte{9, 8, 7, 6}
	want[2].AE44.E40 = 0x13131313

	if err := f.SetSpellEffects(want); err != nil {
		t.Fatalf("SetSpellEffects: %v", err)
	}
	got, present, err := f.SpellEffects()
	if err != nil || !present {
		t.Fatalf("SpellEffects (re-decoded): present=%v err=%v", present, err)
	}
	for i := range want {
		if !sameSpellEffect(got[i], want[i]) {
			t.Errorf("record %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	// A round trip through Marshal/Open must reproduce the written bytes.
	reopened, err := Open(f.Marshal())
	if err != nil {
		t.Fatalf("re-open written file: %v", err)
	}
	again, present, err := reopened.SpellEffects()
	if err != nil || !present {
		t.Fatalf("SpellEffects (re-opened): present=%v err=%v", present, err)
	}
	for i := range want {
		if !sameSpellEffect(again[i], want[i]) {
			t.Errorf("re-opened record %d = %+v, want %+v", i, again[i], want[i])
		}
	}
}

// TestSetSpellEffectsLeavesUnrelatedBytesAlone: patching the SpellEffect
// list must not disturb the world state that follows it (blocks, cell
// records, session) or the actor graph that precedes it, the same isolation
// TestAnEditChangesTheFieldAndNothingElse checks for the fixed-offset
// setters.
func TestSetSpellEffectsLeavesUnrelatedBytesAlone(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero(), merc()},
		spellEffects: func(s *stream) {
			s.u32(1)
			s.pointEffect(1, 2, func(s *stream) { s.effect(1, 2, 3, 4) }, 5)
		}})
	before := append([]byte(nil), f.Body...)
	effects, _, err := f.SpellEffects()
	if err != nil {
		t.Fatalf("SpellEffects: %v", err)
	}
	effects[0].SE40, effects[0].PE44 = 250, 0xfeedface
	effects[0].PE48.E0C = 99
	if err := f.SetSpellEffects(effects); err != nil {
		t.Fatalf("SetSpellEffects: %v", err)
	}
	if len(f.Body) != len(before) {
		t.Fatalf("SetSpellEffects changed the body length: %d -> %d", len(before), len(f.Body))
	}
	n := 0
	for i := range before {
		if before[i] != f.Body[i] {
			n++
		}
	}
	// Exactly SE40 (1 byte), PE44 (4 bytes) and the nested Effect's E0C (1
	// byte) moved.
	if n != 6 {
		t.Fatalf("%d bytes moved, want exactly 6 (SE40, PE44, PE48.E0C)", n)
	}
	if f.World == nil || len(f.World.Blocks) != 0 {
		t.Fatalf("the world half after the SpellEffect list was disturbed: %+v", f.World)
	}
	chars, err := f.Party()
	if err != nil || len(chars) != 2 {
		t.Fatalf("the actor graph before the SpellEffect list was disturbed: %v, %d chars", err, len(chars))
	}
}

func TestSetSpellEffectsRefusesAShapeMismatch(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()},
		spellEffects: func(s *stream) {
			s.u32(1)
			s.pointEffect(1, 2, func(s *stream) { s.effect(1, 2, 3, 4) }, 5)
		}})
	effects, _, err := f.SpellEffects()
	if err != nil {
		t.Fatalf("SpellEffects: %v", err)
	}
	if err := f.SetSpellEffects(nil); err == nil {
		t.Fatal("a record-count mismatch was accepted")
	}
	if err := f.SetSpellEffects(append(effects, effects[0])); err == nil {
		t.Fatal("an added record was accepted")
	}
	withoutChild := append([]SpellEffect(nil), effects...)
	withoutChild[0].PE48 = nil
	if err := f.SetSpellEffects(withoutChild); err == nil {
		t.Fatal("dropping a reference the record already carries was accepted")
	}
	wrongClass := append([]SpellEffect(nil), effects...)
	wrongClass[0].Class = "AreaEffect"
	if err := f.SetSpellEffects(wrongClass); err == nil {
		t.Fatal("a class that does not match the record was accepted")
	}
}
