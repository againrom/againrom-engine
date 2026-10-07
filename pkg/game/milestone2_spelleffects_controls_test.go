package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// This writer spells synthetic bytes directly, not by converting the reader's
// expected values or through a production SAV graph encoder. Shared and distinct
// equal-valued children exercise archive identity independently of field values.
type spell1152Literal struct {
	b       []byte
	next    uint16
	classes map[string]uint16
}

func (s *spell1152Literal) u16(v uint16) { s.b = binary.LittleEndian.AppendUint16(s.b, v) }
func (s *spell1152Literal) u32(v uint32) { s.b = binary.LittleEndian.AppendUint32(s.b, v) }
func (s *spell1152Literal) object(class string) uint16 {
	if id := s.classes[class]; id != 0 {
		s.u16(0x8000 | id)
	} else {
		s.u16(0xffff)
		s.u16(1)
		s.u16(uint16(len(class)))
		s.b = append(s.b, class...)
		s.classes[class] = s.next
		s.next++
	}
	id := s.next
	s.next++
	return id
}
func (s *spell1152Literal) token(seed byte) {
	s.b = append(s.b, 5, 6, 5, 6, 0x31, 0x42, 0x53, 0x64, 0x75, 0x86, 0x97, 0xa8)
	s.u32(0x11223300 | uint32(seed))
	s.b = append(s.b, 0x19)
	s.u16(0x2a3b)
	s.u32(0x4c5d6e7f)
	s.u16(0x8091)
	s.u32(0xa2b3c4d5)
	s.u32(0xd1000000 | uint32(seed))
	s.u32(0xe6f70819)
}
func (s *spell1152Literal) damage() uint16 {
	id := s.object("Effect_DirectDamage")
	s.token(3)
	s.b = append(s.b, 0x11, 0x22)
	s.u32(0x33445566)
	s.b = append(s.b, 0x77)
	s.b = append(s.b, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24)
	return id
}

func spell1152LiteralList(loc sav.SpellEffectArchiveLocation, shared bool) []byte {
	s := &spell1152Literal{next: uint16(loc.NextIndex), classes: map[string]uint16{}}
	for id, class := range loc.Classes {
		s.classes[class] = id
	}
	s.u32(4)
	s.object("SpellTransport")
	s.token(1)
	s.b = append(s.b, 0x11, 0x22)
	point := s.object("PointEffect")
	s.token(2)
	s.b = append(s.b, 0x33, 0x44)
	damage := s.damage()
	s.u32(0x91827364)
	s.object("AreaEffect")
	s.token(4)
	s.b = append(s.b, 0x55, 0x66, 0x10, 0x20, 0x30, 0x40)
	s.u16(0xa55a)
	if shared {
		s.u16(damage)
	} else {
		s.damage()
	}
	s.u16(0xbeef)
	s.u16(point) // A root aliases the already introduced nested PointEffect.
	s.object("SpellEffect")
	s.token(5)
	s.b = append(s.b, 0x88, 0x99)
	s.object("PointEffect")
	s.token(6)
	s.b = append(s.b, 0xaa, 0xbb)
	s.object("Effect")
	s.token(7)
	s.b = append(s.b, 0xcc, 0xdd)
	s.u32(0x1234abcd)
	s.b = append(s.b, 0xee)
	s.u32(0)
	return s.b
}

func spell1152FixtureFront(t *testing.T) *FrontEnd {
	f := poolFixtureFront(t, 91)
	f.Campaign = resolved(saveCampaign(), nil)
	f.SetDeterministicFrames(true)
	return f
}

func spell1152Fixture(t *testing.T, shared bool) (*FrontEnd, []byte, spell1152Graph) {
	t.Helper()
	f := spell1152FixtureFront(t)
	source, err := sav.Open(completeDocumentFixture1115(t, f))
	if err != nil {
		t.Fatal(err)
	}
	loc, present, err := source.SpellEffectArchiveLocation()
	if err != nil || !present {
		t.Fatal(present, err)
	}
	if source.World.BlocksOff != loc.Off+4 || binary.LittleEndian.Uint32(source.Body[loc.Off:]) != 0 {
		t.Fatal("synthetic base no longer has an empty effect list")
	}
	// The fixture spells an empty Sack list, marker/global and 400-byte
	// trailer. Exclude its optional codec alignment byte before recompressing.
	logicalEnd := source.World.SessionOff + 4374 + 4 + 8 + 400
	if logicalEnd+(logicalEnd&1) != len(source.Body) {
		t.Fatal("synthetic trailer changed")
	}
	body := append([]byte(nil), source.Body[:loc.Off]...)
	body = append(body, spell1152LiteralList(loc, shared)...)
	body = append(body, source.Body[source.World.BlocksOff:logicalEnd]...)
	source.Body = body
	raw := source.Marshal()
	source, err = sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := spell1152Expected(source)
	if err != nil {
		t.Fatal(err)
	}
	return f, raw, want
}

func spell1152DocRef(t *testing.T, doc *sav.DocumentData, id uint16, name string) *uint16 {
	t.Helper()
	for i := range doc.Objects[id-1].RefSlots {
		r := &doc.Objects[id-1].RefSlots[i]
		if r.Name == name && len(r.Objects) == 1 {
			return &r.Objects[0]
		}
	}
	t.Fatalf("missing retained reference %d.%s", id, name)
	return nil
}

func TestSpellEffect1152IndependentDAGAndNativeCycle(t *testing.T) {
	for _, shared := range []bool{true, false} {
		name := "distinct equal children"
		if shared {
			name = "shared child"
		}
		t.Run(name, func(t *testing.T) {
			f, raw, want := spell1152Fixture(t, shared)
			fields, typed, refs, aliases, classes := want.population()
			if len(want.roots) != 4 || refs != 5 || classes["SpellEffect"] != 1 || classes["PointEffect"] != 2 || classes["AreaEffect"] != 1 || classes["SpellTransport"] != 1 || classes["Effect"] != 1 {
				t.Fatalf("literal independent population changed: %+v, refs=%d", classes, refs)
			}
			nodes, expectedAliases := 8, 1
			if shared {
				nodes, expectedAliases = 7, 2
			}
			if len(want.nodes) != nodes || aliases != expectedAliases || fields-typed != nodes*37 {
				t.Fatal("literal graph identity/Token accounting changed")
			}
			root := want.nodes[want.roots[0]]
			point, area := want.nodes[root.refs["ST44"]], want.nodes[root.refs["ST48"]]
			if point.refs["PE48"] == area.refs["AE44"] != shared {
				t.Fatal("literal DAG alias changed")
			}
			left, right := want.nodes[point.refs["PE48"]], want.nodes[area.refs["AE44"]]
			if !reflect.DeepEqual(left.values, right.values) || !reflect.DeepEqual(left.raw, right.raw) {
				t.Fatal("literal damage children must have equal values")
			}
			ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			spell1152Check(t, want, ms.World, ms.savedDocument)
			live := ms.World.SavedSpellEffects()
			if live[0].ST44.PE48 != live[1].PE48 {
				t.Fatal("shared Point root identity lost")
			}
			graph := ms.World.SavedSpellGraph()
			if graph == nil || graph.Nodes[graph.Roots[0]-1].Primary != graph.Roots[1] {
				t.Fatal("archive node alias lost")
			}
			app, store := spell1152AppLoad(t, f, raw)
			spell1152CheckFront(t, want, f)
			driver := f.live
			fresh := spell1152MenuFresh(t, f, app, store, spell1152FixtureFront)
			spell1152CheckFront(t, want, fresh)
			spell1152Continue(t, want, driver, f, fresh)
		})
	}
}

func TestSpellEffect1152WitnessRejectsLostState(t *testing.T) {
	f, raw, want := spell1152Fixture(t, true)
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	spell1152Check(t, want, ms.World, ms.savedDocument)
	for _, tc := range []struct {
		name   string
		mutate func(*[]sim.SavedSpellEffect)
	}{
		{"dropped typed root", func(v *[]sim.SavedSpellEffect) { *v = (*v)[:3] }},
		{"typed class", func(v *[]sim.SavedSpellEffect) { (*v)[0].Class = "SpellEffect" }},
		{"SE40", func(v *[]sim.SavedSpellEffect) { (*v)[0].SE40++ }},
		{"SE41", func(v *[]sim.SavedSpellEffect) { (*v)[0].SE41++ }},
		{"ST4C", func(v *[]sim.SavedSpellEffect) { (*v)[0].ST4C++ }},
		{"ST44", func(v *[]sim.SavedSpellEffect) { (*v)[0].ST44 = nil }},
		{"ST48", func(v *[]sim.SavedSpellEffect) { (*v)[0].ST48 = nil }},
		{"PE44", func(v *[]sim.SavedSpellEffect) { (*v)[0].ST44.PE44++ }},
		{"PE48", func(v *[]sim.SavedSpellEffect) { (*v)[0].ST44.PE48 = nil }},
		{"AE48", func(v *[]sim.SavedSpellEffect) { (*v)[0].ST48.AE48[3]++ }},
		{"AE4C", func(v *[]sim.SavedSpellEffect) { (*v)[0].ST48.AE4C++ }},
		{"AE44", func(v *[]sim.SavedSpellEffect) { (*v)[0].ST48.AE44 = nil }},
		{"E3C", func(v *[]sim.SavedSpellEffect) { (*v)[0].ST44.PE48.E3C++ }},
		{"E3D", func(v *[]sim.SavedSpellEffect) { (*v)[0].ST44.PE48.E3D++ }},
		{"E40", func(v *[]sim.SavedSpellEffect) { (*v)[0].ST44.PE48.E40++ }},
		{"E0C", func(v *[]sim.SavedSpellEffect) { (*v)[0].ST44.PE48.E0C++ }},
		{"EDD48", func(v *[]sim.SavedSpellEffect) { (*v)[0].ST44.PE48.DirectDamage[23]++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := ms.World.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var copy sim.World
			if err := copy.UnmarshalBinary(encoded); err != nil {
				t.Fatal(err)
			}
			live := copy.SavedSpellEffects()
			tc.mutate(&live)
			if len(want.typedDifferences(live)) == 0 {
				t.Fatal("lost typed state escaped independent witness")
			}
		})
	}
	for _, tc := range []struct {
		name   string
		mutate func(*sav.DocumentData)
	}{
		{"dropped document root", func(d *sav.DocumentData) { d.World.Effects = d.World.Effects[:3] }},
		{"changed root alias", func(d *sav.DocumentData) { d.World.Effects[1] = d.World.Effects[2] }},
		{"dropped scalar", func(d *sav.DocumentData) { i := d.World.Effects[0] - 1; d.Objects[i].Values = d.Objects[i].Values[1:] }},
		{"dropped Token Position", func(d *sav.DocumentData) { d.Objects[d.World.Effects[0]-1].Raw = nil }},
		{"changed Token byte", func(d *sav.DocumentData) { d.Objects[d.World.Effects[0]-1].Raw[0].Bytes[7]++ }},
		{"split equal shared child", func(d *sav.DocumentData) {
			area := *spell1152DocRef(t, d, d.World.Effects[0], "ST48")
			ref := spell1152DocRef(t, d, area, "AE44")
			d.Objects = append(d.Objects, d.Objects[*ref-1])
			*ref = uint16(len(d.Objects))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, err := cloneSavedDocument(ms.savedDocument)
			if err != nil {
				t.Fatal(err)
			}
			tc.mutate(state.Document)
			if len(want.documentDifferences(state)) == 0 {
				t.Fatal("lost document state escaped independent witness")
			}
		})
	}
	// Alter every retained scalar, including all eight Token scalar fields.
	for _, root := range ms.savedDocument.Document.World.Effects {
		for _, v := range ms.savedDocument.Document.Objects[root-1].Values {
			state, err := cloneSavedDocument(ms.savedDocument)
			if err != nil {
				t.Fatal(err)
			}
			for i := range state.Document.Objects[root-1].Values {
				field := &state.Document.Objects[root-1].Values[i]
				if field.Name == v.Name {
					field.Value ^= 1
				}
			}
			if len(want.documentDifferences(state)) == 0 {
				t.Fatalf("retained scalar %d.%s escaped", root, v.Name)
			}
		}
	}
	_, distinctRaw, distinct := spell1152Fixture(t, false)
	distinctMission, _, err := ResumeOriginalSave(f.Archives.Containers, distinctRaw, f.Table, f.Difficulty, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	state, err := cloneSavedDocument(distinctMission.savedDocument)
	if err != nil {
		t.Fatal(err)
	}
	d := state.Document
	point := *spell1152DocRef(t, d, d.World.Effects[0], "ST44")
	area := *spell1152DocRef(t, d, d.World.Effects[0], "ST48")
	*spell1152DocRef(t, d, area, "AE44") = *spell1152DocRef(t, d, point, "PE48")
	if len(distinct.documentDifferences(state)) == 0 {
		t.Fatal("distinct equal-valued children collapsed undetected")
	}
}

func TestSpellEffect1152RawCountsBoundsAndTags(t *testing.T) {
	loc := sav.SpellEffectArchiveLocation{NextIndex: 1}
	b := spell1152LiteralList(loc, true)
	if _, err := spell1152Read(b, loc, len(b)); err != nil {
		t.Fatal(err)
	}
	for end := 0; end < len(b); end++ {
		if _, err := spell1152Read(b[:end], loc, end); err == nil {
			t.Fatalf("accepted truncation at %d/%d", end, len(b))
		}
	}
	for _, count := range []uint32{0, 3, 5, 0xffffffff} {
		changed := bytes.Clone(b)
		binary.LittleEndian.PutUint32(changed, count)
		if _, err := spell1152Read(changed, loc, len(changed)); err == nil {
			t.Fatalf("accepted root count %d", count)
		}
	}
	for _, tag := range []uint16{0, 0x7777, 0x8001} {
		changed := bytes.Clone(b)
		binary.LittleEndian.PutUint16(changed[4:], tag)
		if _, err := spell1152Read(changed, loc, len(changed)); err == nil {
			t.Fatalf("accepted malformed first tag %#x", tag)
		}
	}
	// A complete empty list remains distinguishable from an omitted root.
	if empty, err := spell1152Read(make([]byte, 4), loc, 4); err != nil || len(empty.roots) != 0 || len(empty.nodes) != 0 {
		t.Fatal(empty, err)
	}
}

func TestSpellEffect1152LiteralFieldsAndPriorReference(t *testing.T) {
	loc := sav.SpellEffectArchiveLocation{NextIndex: 1}
	b := spell1152LiteralList(loc, true)
	g, err := spell1152Read(b, loc, len(b))
	if err != nil {
		t.Fatal(err)
	}
	root := g.nodes[g.roots[0]]
	if !reflect.DeepEqual(root.values, map[string]uint32{
		"RuntimeID": 0x11223301, "T0C": 0x19, "T0E": 0x2a3b, "T08": 0x4c5d6e7f,
		"T18": 0x8091, "T1C": 0xa2b3c4d5, "Identity": 0xd1000001, "Reference": 0xe6f70819,
		"SE40": 0x11, "SE41": 0x22, "ST4C": 0xbeef,
	}) || !bytes.Equal(root.raw["Block12"], []byte{5, 6, 5, 6, 0x31, 0x42, 0x53, 0x64, 0x75, 0x86, 0x97, 0xa8}) {
		t.Fatal("literal Token/SpellTransport field layout changed", root)
	}
	point, area := g.nodes[root.refs["ST44"]], g.nodes[root.refs["ST48"]]
	damage := g.nodes[point.refs["PE48"]]
	if point.values["SE40"] != 0x33 || point.values["SE41"] != 0x44 || point.values["PE44"] != 0x91827364 ||
		area.values["SE40"] != 0x55 || area.values["SE41"] != 0x66 || area.values["AE4C"] != 0xa55a ||
		!bytes.Equal(area.raw["AE48"], []byte{0x10, 0x20, 0x30, 0x40}) ||
		damage.values["E3C"] != 0x11 || damage.values["E3D"] != 0x22 || damage.values["E40"] != 0x33445566 || damage.values["E0C"] != 0x77 ||
		!bytes.Equal(damage.raw["EDD48"], []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}) {
		t.Fatal("literal PointEffect/AreaEffect/DirectDamage fields changed")
	}
	bare, secondPoint := g.nodes[g.roots[2]], g.nodes[g.roots[3]]
	effect := g.nodes[secondPoint.refs["PE48"]]
	if bare.values["SE40"] != 0x88 || bare.values["SE41"] != 0x99 || secondPoint.values["SE40"] != 0xaa || secondPoint.values["SE41"] != 0xbb || secondPoint.values["PE44"] != 0 ||
		effect.values["E3C"] != 0xcc || effect.values["E3D"] != 0xdd || effect.values["E40"] != 0x1234abcd || effect.values["E0C"] != 0xee || len(effect.raw) != 1 {
		t.Fatal("literal base SpellEffect/Effect fields changed")
	}
	// A typed reference may name an Effect introduced in an earlier Item.
	// The prefix supplies its body locator only; its seven fields still come
	// from this reader's raw bytes. No source value is carried in the locator.
	s := &spell1152Literal{next: 1, classes: map[string]uint16{}}
	prior := s.object("Effect")
	off := len(s.b)
	s.token(8)
	s.b = append(s.b, 9, 10)
	s.u32(0xabcdef12)
	s.b = append(s.b, 11)
	loc = sav.SpellEffectArchiveLocation{Off: len(s.b), NextIndex: int(s.next), Classes: map[uint16]string{1: "Effect"}, Objects: []sav.SpellEffectPriorObject{{Index: prior, Off: off, Class: "Effect"}}}
	s.u32(1)
	s.object("PointEffect")
	s.token(9)
	s.b = append(s.b, 12, 13)
	s.u16(prior)
	s.u32(0)
	g, err = spell1152Read(s.b, loc, len(s.b))
	if err != nil || len(g.nodes) != 2 || g.nodes[prior].values["E40"] != 0xabcdef12 || g.nodes[g.roots[0]].refs["PE48"] != prior {
		t.Fatal("prior Effect raw-field/reference read", g, err)
	}
}
