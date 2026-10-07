package sav

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func documentDataFixture(t *testing.T) DocumentData {
	t.Helper()
	source, _, _, _ := saveDocument1115Literal(t)
	d, err := DecodeDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	clear(source)
	return d
}

func documentDataGobCopy(t *testing.T, d DocumentData) ([]byte, DocumentData) {
	t.Helper()
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(d); err != nil {
		t.Fatal(err)
	}
	var decoded DocumentData
	if err := gob.NewDecoder(bytes.NewReader(b.Bytes())).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	return b.Bytes(), decoded
}

func documentRecordByClass(t *testing.T, d *DocumentData, class string) *DocumentRecordData {
	t.Helper()
	for i := range d.Objects {
		if d.Objects[i].Class == class {
			return &d.Objects[i]
		}
	}
	t.Fatalf("missing class %s", class)
	return nil
}

func documentValue1115(t *testing.T, r *DocumentRecordData, name string) *uint32 {
	t.Helper()
	for i := range r.Values {
		if r.Values[i].Name == name {
			return &r.Values[i].Value
		}
	}
	t.Fatalf("missing %s.%s", r.Class, name)
	return nil
}

func documentRefsForField(t *testing.T, r *DocumentRecordData, name string) *[]uint16 {
	t.Helper()
	for i := range r.RefSlots {
		if r.RefSlots[i].Name == name {
			return &r.RefSlots[i].Objects
		}
	}
	t.Fatalf("missing %s.%s slots", r.Class, name)
	return nil
}

func documentCountForField(t *testing.T, r *DocumentRecordData, name string) *uint32 {
	t.Helper()
	for i := range r.Counts {
		if r.Counts[i].Name == name {
			return &r.Counts[i].Count
		}
	}
	t.Fatalf("missing %s.%s count", r.Class, name)
	return nil
}

func TestDocumentData1115LiteralErasureGobAndWholeContainer(t *testing.T) {
	source, body, state, campaign := saveDocument1115Literal(t)
	wantBody := append([]byte(nil), body...)
	if _, pad := archiveDocument1115Fixture(); pad {
		wantBody[len(wantBody)-1] = 0
	}
	wantState, _ := literalReadState(state)
	wantCampaign := append([]byte(nil), campaign...)
	d, err := DecodeDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	clear(source)
	clear(body)
	clear(state)
	clear(campaign)
	if d.Version != DocumentDataVersion || d.Head.CounterA != 0x87654321 || d.Head.CounterB != 0x76543210 || d.Head.Mission != 23 || d.Head.Difficulty != 3 || d.Head.PlayerListField != 9 || d.Head.Reserved[10] != 0x10a {
		t.Fatal("literal head values lost")
	}
	if !reflect.DeepEqual(d.Players, []uint16{0, 1, 1}) || len(d.DeadActors) != 2 || d.DeadActors[0] != d.DeadActors[1] || d.Objects[d.DeadActors[0]-1].Class != "Human" {
		t.Fatal("local root indices do not retain nulls/repetitions")
	}
	p := documentRecordByClass(t, &d, "Player")
	if len(p.Inline) != 1 || p.Inline[0].Name != "Diary" || p.Inline[0].Record.Class != "Diary" || len(p.Groups) != 1 || p.Groups[0].Class != "Group" || (*documentRefsForField(t, &p.Groups[0], "Actors"))[0] != d.DeadActors[0] {
		t.Fatal("inline Diary/Group ownership or cross-root actor alias lost")
	}
	for _, r := range d.Objects {
		if r.Class == "Group" || r.Class == "Diary" {
			t.Fatal("inline-only record entered the archive object table")
		}
	}
	w := d.World
	if w == nil || len(w.Cells) != 3 || w.Cells[0].Cell != w.Cells[1].Cell || w.Cells[0].GroundActor != 0x07060504 || w.Cells[0].Residue32 != 0x3332 || w.TerrainIdentity != 0x11223344 || w.Session.Diplomacy[0][0] != byte((1856*37+11)%256) {
		t.Fatal("world cells, explicit keys or session value lost")
	}
	if w.Buildings[0] != w.Buildings[1] || w.Sacks[0] != w.Sacks[1] || (*documentRefsForField(t, documentRecordByClass(t, &d, "SpellTransport"), "ST48"))[0] != w.Effects[1] {
		t.Fatal("world/nested aliases lost")
	}
	wire, persisted := documentDataGobCopy(t, d)
	for range 5 {
		again, _ := documentDataGobCopy(t, d)
		if !bytes.Equal(wire, again) {
			t.Fatal("ordinary gob encoding is nondeterministic")
		}
	}
	got, err := EncodeDocumentData(persisted)
	if err != nil {
		t.Fatal(err)
	}
	f, err := Open(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(f.Body, wantBody) || !bytes.Equal(f.TailRest, wantCampaign) {
		t.Fatal("whole body or campaign differs from independent literal after source erasure/gob")
	}
	gotState, _ := literalReadState(f.Store)
	if !reflect.DeepEqual(gotState, wantState) || !bytes.Equal(f.Label, []byte{'s', 0xe0, 'v'}) {
		t.Fatal("state/label values changed")
	}
	reloaded, err := DecodeDocumentData(got)
	if err != nil {
		t.Fatal(err)
	}
	canonicalWire, _ := documentDataGobCopy(t, reloaded)
	if !bytes.Equal(wire, canonicalWire) {
		t.Fatal("full DTO is not canonical across encode/reload")
	}
}

func TestDocumentData1115TypedMutationAndIndependentOwnership(t *testing.T) {
	d := documentDataFixture(t)
	wire, persisted := documentDataGobCopy(t, d)
	clear(wire)
	d.Head.CounterA = 9
	d.World.Cells[0].GroundActor = 7
	d.World.Session.Diplomacy[0][0] = 3
	d.Campaign.Children[0].Age++
	d.State.ValueRecords[0].Value.Bytes[0] ^= 0x12
	if persisted.Head.CounterA != 0x87654321 || persisted.World.Cells[0].GroundActor != 0x07060504 || persisted.World.Session.Diplomacy[0][0] != byte((1856*37+11)%256) {
		t.Fatal("gob retained source ownership")
	}
	d = persisted
	d.Label = []byte("current")
	d.Head.CounterA = 123456
	d.Head.Reserved[7] = 333
	d.Head.Difficulty = 4
	*documentValue1115(t, documentRecordByClass(t, &d, "Outpost"), "O84") = 3456
	p := documentRecordByClass(t, &d, "Player")
	for i := range p.Inline[0].Record.Raw {
		if p.Inline[0].Record.Raw[i].Name == "Journal" {
			p.Inline[0].Record.Raw[i].Bytes = []byte{1, 2, 3, 4, 5, 6, 7, 8}
		}
	}
	*documentCountForField(t, &p.Inline[0].Record, "Journal") = 2
	human := documentRecordByClass(t, &d, "Human")
	human.Texts[0].Value = "changed\xe0"
	for i := range human.Raw {
		if human.Raw[i].Name == "UA6" {
			human.Raw[i].Bytes[3] = 19
		}
	}
	sack := documentRecordByClass(t, &d, "Sack")
	contents := documentRefsForField(t, sack, "Contents")
	(*contents)[0] = (*contents)[1] // Former null becomes the same explicit alias.
	d.World.Cells[1].Layers[5] = 0x98765432
	d.World.Cells[1].Operation = 201
	d.World.Session.Results[92] = -302
	d.World.Session.Raw08[47] = 127
	d.World.Session.RawA828[399] = 19
	d.World.Session.ValueB3B0 = 0x1234
	d.World.Session.Lost = 9
	d.World.TerrainIdentity = 0x33334444
	d.Trailer[99] = 89
	for i := range d.State.ValueRecords {
		v := &d.State.ValueRecords[i]
		switch v.Path {
		case "/Prj266/actionphase":
			v.Value.Int32 = -91
		case "/Fog/Data":
			v.Value.Bytes = literalStateWords(1, 7, 13, 21)
		}
	}
	d.Campaign.Scalars[5] = 112233
	d.Campaign.Children[0].Age = 57
	_, native := documentDataGobCopy(t, d)
	b, err := EncodeDocumentData(native)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeDocumentData(b)
	if err != nil {
		t.Fatal(err)
	}
	wantGob, _ := documentDataGobCopy(t, d)
	gotGob, _ := documentDataGobCopy(t, got)
	if !bytes.Equal(wantGob, gotGob) {
		t.Fatal("a typed mutation was ignored or conflicted with another authority")
	}
	internal, err := saveDocumentFromData(got)
	if err != nil {
		t.Fatal(err)
	}
	actor := internal.archive.dead[0]
	if len(actor.WornSlots) != 12 || len(actor.Refs["Worn"]) != 0 || len(internal.archive.world.sacks[0].Refs["Contents"]) != 3 {
		t.Fatal("nullable authority did not reconstruct compact/worn views")
	}
	// Both conversion directions detach byte arrays even without gob between.
	x := internal.archive.world.buildings[0].Raw["O6C"][0]
	for i := range got.Objects {
		for j := range got.Objects[i].Raw {
			clear(got.Objects[i].Raw[j].Bytes)
		}
	}
	if internal.archive.world.buildings[0].Raw["O6C"][0] != x {
		t.Fatal("DTO-to-document retained byte aliases")
	}
}

// Literal effect graph, including two new objects and backreferences to the
// object currently being read. Numeric old-address keys are unrelated values.
func documentCycleLiteral(t *testing.T, self bool) []byte {
	t.Helper()
	s := newStream()
	s.u32(11)
	s.u32(12)
	s.cstr("10.alm")
	for range 11 {
		s.u32(0)
	}
	s.u32(10)
	s.u32(2)
	s.u32(9)
	s.u32(0) // Players
	s.u32(0) // Dead
	s.u8(1)
	s.u32(0)            // Buildings
	s.u32(2)            // Repeated effect roots as well as a cycle.
	s.obj("AreaEffect") // Archive class1/object2, DTO object1.
	areaToken := len(s.b)
	s.token(0x1413, 1, 2, 300, 4, 0)
	binary.LittleEndian.PutUint32(s.b[areaToken+29:], 0) // Own key is deliberately zero.
	s.raw(6, 0x23)
	s.u16(17)
	if self {
		s.backref(2)
	} else {
		s.obj("PointEffect")
		pointToken := len(s.b)
		s.token(0x1514, 5, 6, 400, 7, 0)
		binary.LittleEndian.PutUint32(s.b[pointToken+29:], 0) // Same key, different object.
		s.raw(2, 0x34)
		s.backref(2)
		s.u32(0x12345678)
	}
	s.backref(2)
	s.u16(0) // Blocks
	s.u16(0) // Cells
	s.u32(0x99887766)
	s.raw(4374, 0)
	s.u32(0) // Sacks
	s.u32(0) // Marker, no global arm
	s.raw(400, 0)
	if len(s.b)&1 != 0 {
		s.u8(0)
	}
	_, campaign := campaignProjectionFixture()
	blob := Compress(s.b)
	out := make([]byte, 16)
	copy(out, "Asg&")
	binary.LittleEndian.PutUint32(out[4:], uint32(16+len(blob)))
	binary.LittleEndian.PutUint32(out[8:], 0x0bad0002)
	binary.LittleEndian.PutUint32(out[12:], uint32(len(blob)))
	out = append(out, blob...)
	out = append(out, make([]byte, 256)...)
	out = append(out, literalStateStore(literalWorldDirectories())...)
	return append(out, campaign...)
}

func TestDocumentData1115CyclesRemainLocalReferences(t *testing.T) {
	for _, self := range []bool{false, true} {
		source := documentCycleLiteral(t, self)
		d, err := DecodeDocumentData(source)
		if err != nil {
			t.Fatal(err)
		}
		clear(source)
		if d.World.Effects[0] != 1 {
			t.Fatal("archive tag used as DTO index")
		}
		area := *documentRefsForField(t, &d.Objects[0], "AE44")
		if (self && area[0] != 1) || (!self && (area[0] != 2 || (*documentRefsForField(t, &d.Objects[1], "PE48"))[0] != 1)) {
			t.Fatal("cycle is not represented by local indices")
		}
		wire, persisted := documentDataGobCopy(t, d)
		encoded, err := EncodeDocumentData(persisted)
		if err != nil {
			t.Fatal(err)
		}
		reloaded, err := DecodeDocumentData(encoded)
		if err != nil {
			t.Fatal(err)
		}
		again, _ := documentDataGobCopy(t, reloaded)
		if !bytes.Equal(wire, again) {
			t.Fatal("cycle did not survive complete gob/format roundtrip")
		}
		internal, err := saveDocumentFromData(reloaded)
		if err != nil {
			t.Fatal(err)
		}
		root := internal.archive.world.effects[0]
		back := root.RefSlots["AE44"][0]
		if !self {
			back = back.RefSlots["PE48"][0]
		}
		if back != root || root.Index != 0 || root.Off != 0 || root.End != 0 {
			t.Fatal("fresh pointers do not recover the cycle without source coordinates")
		}
	}
}

func TestDocumentData1115CityUsesSameDTO(t *testing.T) {
	source := cityTestSource(t)
	d, err := DecodeDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	clear(source)
	if d.World != nil || len(d.State.DirectoryRecords) != 7 || len(d.State.ValueRecords) != 15 {
		t.Fatal("city state incorrectly widened to world")
	}
	wire, stored := documentDataGobCopy(t, d)
	b, err := EncodeDocumentData(stored)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := DecodeDocumentData(b)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := documentDataGobCopy(t, reloaded)
	if !bytes.Equal(wire, again) {
		t.Fatal("city complete DTO changed")
	}
	f, err := Open(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.CityProvenance(); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentData1115RejectsMalformedAndIgnoredFields(t *testing.T) {
	cases := map[string]func(*DocumentData){
		"dto version":      func(d *DocumentData) { d.Version++ },
		"file version":     func(d *DocumentData) { d.FileVersion = 0 },
		"label terminator": func(d *DocumentData) { d.Label = []byte{'x', 0, 'y'} },
		"label size":       func(d *DocumentData) { d.Label = make([]byte, 256) },
		"map size":         func(d *DocumentData) { d.Head.MapName = strings.Repeat("x", 255) },
		"global inactive":  func(d *DocumentData) { d.Marker = 0 },
		"root range":       func(d *DocumentData) { d.Players[0] = uint16(len(d.Objects) + 1) },
		"nested range": func(d *DocumentData) {
			(*documentRefsForField(t, documentRecordByClass(t, d, "Sack"), "Contents"))[0] = uint16(len(d.Objects) + 1)
		},
		"unreachable":       func(d *DocumentData) { d.Objects = append(d.Objects, d.Objects[len(d.Objects)-1]) },
		"index permutation": func(d *DocumentData) { d.Players[1] = 2 },
		"nil dead":          func(d *DocumentData) { d.DeadActors[0] = 0 },
		"nil building":      func(d *DocumentData) { d.World.Buildings[0] = 0 },
		"nil effect":        func(d *DocumentData) { d.World.Effects[0] = 0; d.World.Effects[1] = 0 },
		"wrong root class":  func(d *DocumentData) { d.World.Buildings[0] = d.DeadActors[0] },
		"unknown class":     func(d *DocumentData) { d.Objects[len(d.Objects)-1].Class = "Unproved" },
		"inline class":      func(d *DocumentData) { documentRecordByClass(t, d, "Player").Inline[0].Record.Class = "Spell" },
		"inline missing":    func(d *DocumentData) { documentRecordByClass(t, d, "Player").Inline = nil },
		"inline duplicate": func(d *DocumentData) {
			p := documentRecordByClass(t, d, "Player")
			p.Inline = append(p.Inline, p.Inline[0])
		},
		"inline archive conflict": func(d *DocumentData) {
			p := documentRecordByClass(t, d, "Player")
			p.RefSlots = []DocumentRefsData{{"Diary", []uint16{0}}}
		},
		"group wrong class": func(d *DocumentData) { documentRecordByClass(t, d, "Player").Groups[0].Class = "Diary" },
		"inactive groups": func(d *DocumentData) {
			documentRecordByClass(t, d, "Outpost").Groups = documentRecordByClass(t, d, "Player").Groups
		},
		"missing scalar": func(d *DocumentData) { r := documentRecordByClass(t, d, "Outpost"); r.Values = r.Values[1:] },
		"extra scalar": func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Outpost")
			r.Values = append(r.Values, DocumentValueData{"ZZUnused", 19})
		},
		"duplicate scalar": func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Outpost")
			r.Values = append(r.Values[:1], r.Values...)
		},
		"unsorted scalar": func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Outpost")
			r.Values[0], r.Values[1] = r.Values[1], r.Values[0]
		},
		"scalar width":      func(d *DocumentData) { *documentValue1115(t, documentRecordByClass(t, d, "Outpost"), "T0C") = 256 },
		"scalar saturation": func(d *DocumentData) { *documentValue1115(t, documentRecordByClass(t, d, "Player"), "F54") = 0x8000 },
		"extra text": func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Outpost")
			r.Texts = []DocumentTextData{{"ZZUnused", "text"}}
		},
		"duplicate text": func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Human")
			r.Texts = append(r.Texts, r.Texts[0])
		},
		"long text": func(d *DocumentData) { documentRecordByClass(t, d, "Human").Texts[0].Value = strings.Repeat("x", 255) },
		"extra raw": func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Outpost")
			r.Raw = append(r.Raw, DocumentRawData{"ZZUnused", []byte{0}})
		},
		"duplicate raw": func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Outpost")
			r.Raw = append(r.Raw[:1], r.Raw...)
		},
		"raw wrong extent": func(d *DocumentData) { r := documentRecordByClass(t, d, "Outpost"); r.Raw[0].Bytes = []byte{0} },
		"extra count": func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Outpost")
			r.Counts = append(r.Counts, DocumentCountData{"ZZUnused", 0})
		},
		"duplicate count": func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Outpost")
			r.Counts = append(r.Counts, r.Counts[0])
		},
		"missing count": func(d *DocumentData) { documentRecordByClass(t, d, "Outpost").Counts = nil },
		"count too big": func(d *DocumentData) {
			*documentCountForField(t, documentRecordByClass(t, d, "Outpost"), "O6C") = maxListElements + 1
		},
		"count wrong extent":     func(d *DocumentData) { *documentCountForField(t, documentRecordByClass(t, d, "Outpost"), "O6C") = 1 },
		"derived count conflict": func(d *DocumentData) { *documentCountForField(t, documentRecordByClass(t, d, "Player"), "Actors") = 2 },
		"derived count missing":  func(d *DocumentData) { p := documentRecordByClass(t, d, "Player"); p.Counts = p.Counts[1:] },
		"extra slots": func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Outpost")
			r.RefSlots = []DocumentRefsData{{"ZZUnused", nil}}
		},
		"duplicate slots": func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Sack")
			r.RefSlots = append(r.RefSlots, r.RefSlots[0])
		},
		"slot wrong extent": func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Sack")
			p := documentRefsForField(t, r, "Contents")
			*p = (*p)[1:]
		},
		"empty slot site missing": func(d *DocumentData) { r := documentRecordByClass(t, d, "Item"); r.RefSlots = nil },
		"inactive inventory": func(d *DocumentData) {
			r := documentRecordByClass(t, d, "Human")
			r.Values = append(r.Values, DocumentValueData{"Inventory1C", 0})
			sort.Slice(r.Values, func(i, j int) bool { return r.Values[i].Name < r.Values[j].Name })
		},
		"state duplicate": func(d *DocumentData) {
			d.State.ValueRecords = append(d.State.ValueRecords, d.State.ValueRecords[len(d.State.ValueRecords)-1])
		},
		"state directory duplicate": func(d *DocumentData) {
			d.State.DirectoryRecords = append(d.State.DirectoryRecords, d.State.DirectoryRecords[len(d.State.DirectoryRecords)-1])
		},
		"state inactive scalar": func(d *DocumentData) { d.State.ValueRecords[0].Value.Int32 = 8 },
		"state missing projectile leaf": func(d *DocumentData) {
			for i, v := range d.State.ValueRecords {
				if v.Path == "/Prj266/actionphase" {
					d.State.ValueRecords = append(d.State.ValueRecords[:i], d.State.ValueRecords[i+1:]...)
					break
				}
			}
		},
		"world carries city state": func(d *DocumentData) {
			c, err := DecodeDocumentData(cityTestSource(t))
			if err != nil {
				t.Fatal(err)
			}
			d.State = c.State
		},
		"campaign parallel mismatch": func(d *DocumentData) { d.Campaign.Parallel[0] = append(d.Campaign.Parallel[0], 3) },
		"campaign invalid flag":      func(d *DocumentData) { d.Campaign.Base.DWords[5] = 3 },
		"campaign bad marker":        func(d *DocumentData) { d.Campaign.Markers[0].Text = []byte{'x'} },
		"blocks unsorted":            func(d *DocumentData) { d.World.Blocks[0], d.World.Blocks[1] = d.World.Blocks[1], d.World.Blocks[0] },
		"blocks duplicate":           func(d *DocumentData) { d.World.Blocks[1].Cell = d.World.Blocks[0].Cell },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := documentDataFixture(t)
			mutate(&d)
			if b, err := EncodeDocumentData(d); err == nil || b != nil {
				t.Fatal("malformed DTO produced a save")
			}
			if clone, err := CloneDocumentData(d); err == nil || !reflect.DeepEqual(clone, DocumentData{}) {
				t.Fatal("malformed DTO was accepted for native adoption")
			}
			if out, permutation, err := ReindexDocumentData(d); err == nil || !reflect.DeepEqual(out, DocumentData{}) || permutation != nil {
				t.Fatal("reindex repaired malformed state or returned partial bindings")
			}
		})
	}
	source, _, _, _ := saveDocument1115Literal(t)
	for _, n := range []int{0, 19, len(source) - 1} {
		if d, err := DecodeDocumentData(source[:n]); err == nil || !reflect.DeepEqual(d, DocumentData{}) {
			t.Fatal("truncated source returned a partial DTO")
		}
	}
}

func TestDocumentData1115BoundsBeforeCopies(t *testing.T) {
	for name, mutate := range map[string]func(*DocumentData){
		"objects":       func(d *DocumentData) { d.Objects = make([]DocumentRecordData, maxDocumentDataObjects+1) },
		"root count":    func(d *DocumentData) { d.Players = make([]uint16, maxListElements+1) },
		"terrain count": func(d *DocumentData) { d.World.Cells = make([]DocumentCellData, maxListElements+1) },
		"elements":      func(d *DocumentData) { d.Campaign.DWords = make([]uint32, maxDocumentDataElements+1) },
		"shared large byte aliases": func(d *DocumentData) {
			span := make([]byte, 4<<20)
			for i := range d.State.ValueRecords {
				d.State.ValueRecords[i].Value.Bytes = span
			}
		},
		"recursive inline slices": func(d *DocumentData) {
			cycle := make([]DocumentInlineData, 1)
			cycle[0] = DocumentInlineData{Name: "Diary", Record: DocumentRecordData{Class: "Diary", Inline: cycle}}
			d.Objects[0].Inline = cycle
		},
	} {
		t.Run(name, func(t *testing.T) {
			d := documentDataFixture(t)
			mutate(&d)
			if out, err := EncodeDocumentData(d); err == nil || out != nil || (!strings.Contains(err.Error(), "bound") && !strings.Contains(err.Error(), "budget")) {
				t.Fatalf("unbounded model was not rejected before conversion: %v", err)
			}
			if out, err := CloneDocumentData(d); err == nil || !reflect.DeepEqual(out, DocumentData{}) {
				t.Fatalf("native clone accepted unbounded model: %v", err)
			}
			if out, permutation, err := ReindexDocumentData(d); err == nil || !reflect.DeepEqual(out, DocumentData{}) || permutation != nil {
				t.Fatalf("reindex accepted unbounded model or returned partial bindings: %v", err)
			}
		})
	}
}

func TestDocumentData1115PersistenceTypeHasNoMapsOrCustomGob(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var inspect func(reflect.Type)
	inspect = func(typ reflect.Type) {
		if seen[typ] {
			return
		}
		seen[typ] = true
		if typ.Implements(reflect.TypeFor[gob.GobEncoder]()) || typ.Implements(reflect.TypeFor[gob.GobDecoder]()) || reflect.PointerTo(typ).Implements(reflect.TypeFor[gob.GobEncoder]()) || reflect.PointerTo(typ).Implements(reflect.TypeFor[gob.GobDecoder]()) {
			t.Fatalf("custom gob method hides %s from native preflight", typ)
		}
		switch typ.Kind() {
		case reflect.Map, reflect.Interface:
			t.Fatalf("nondeterministic/untyped persisted field %s", typ)
		case reflect.Pointer, reflect.Slice, reflect.Array:
			inspect(typ.Elem())
		case reflect.Struct:
			for i := 0; i < typ.NumField(); i++ {
				f := typ.Field(i)
				if !f.IsExported() || f.Name == "Index" || f.Name == "Off" || f.Name == "End" || f.Name == "Refs" || f.Name == "SpellSlots" || f.Name == "WornSlots" {
					t.Fatalf("hidden/source/competing authority field %s.%s", typ, f.Name)
				}
				inspect(f.Type)
			}
		}
	}
	inspect(reflect.TypeFor[DocumentData]())
}

// This is a format-only arbitrary-reference fixture, not a claim that a live
// Sack should contain every possible class. The envelope has one admitted root
// and gives each independently spelled class body a reachable archive site.
func documentRootSack1115Literal(t *testing.T, emit func(*stream)) ([]byte, []byte) {
	t.Helper()
	s := newStream()
	s.u32(11)
	s.u32(12)
	s.cstr("10.alm")
	for range 11 {
		s.u32(0)
	}
	s.u32(10)
	s.u32(2)
	s.u32(9)
	s.u32(0)
	s.u32(0)
	s.u8(1)
	s.u32(0) // Buildings
	s.u32(0) // Effects
	s.u16(0) // Blocks
	s.u16(0) // Cells
	s.u32(0x66778899)
	s.raw(4374, 0)
	s.u32(1)
	s.obj("Sack")
	s.token(0x3231, 3, 5, 79, 7, 0)
	s.u32(11)
	s.u32(1)
	emit(s)
	s.u32(12)
	s.u32(13)
	s.u32(0)
	s.raw(400, 0)
	if len(s.b)&1 != 0 {
		s.u8(0)
	}
	_, campaign := campaignProjectionFixture()
	blob := Compress(s.b)
	out := make([]byte, 16)
	copy(out, "Asg&")
	binary.LittleEndian.PutUint32(out[4:], uint32(16+len(blob)))
	binary.LittleEndian.PutUint32(out[8:], 0x0bad0002)
	binary.LittleEndian.PutUint32(out[12:], uint32(len(blob)))
	out = append(out, blob...)
	out = append(out, make([]byte, 256)...)
	out = append(out, literalStateStore(literalWorldDirectories())...)
	out = append(out, campaign...)
	return out, s.b
}

func documentClass1115Literal(s *stream, class string) {
	switch class {
	case "Item", "Shield", "Armor", "Weapon":
		s.item(wantPiece{class: class, code: 0x5432, row: 3, stack: 7})
		return
	case "Human":
		s.human(wantChar{name: "all classes", runtimeID: 300, journal: -1})
		return
	case "Unit", "Humanoid":
		// The independent Human fixture has no nested non-null references.
		// Humanoid adds 24 XP bytes, twelve words and one Diary word to Unit.
		h := newStream()
		h.human(wantChar{name: "all classes", runtimeID: 300, journal: -1})
		body := h.b[6+len("Human"):]
		if class == "Unit" {
			body = body[:len(body)-50]
		}
		s.obj(class)
		s.b = append(s.b, body...)
		return
	}
	s.obj(class)
	switch class {
	case "Player":
		s.cstr("empty groups")
		s.raw(51, 0)
		s.u32(0)
		s.raw(32, 0x45)
		s.u16(2)
		s.raw(8, 0x37)
		s.u16(3)
		s.raw(6, 0x46)
		s.u32(0x456789ab)
	case "Diary":
		s.u16(2)
		s.raw(8, 0x37)
		s.u16(3)
		s.raw(6, 0x46)
		s.u32(0x56789abc)
	case "Spell":
		s.u8(1)
		s.u8(2)
		s.u8(3)
		s.u16(0x3456)
		s.u32(0x456789ab)
	case "Spellbook":
		s.u32(0x456789ab)
		s.u32(4)
		s.null()
		s.null()
		s.null()
	default:
		s.token(0x1413, 3, 5, 300, 7, 0)
		switch class {
		case "Token":
		case "Building", "Shop", "Tavern", "Outpost":
			s.raw(40, 0x31)
			switch class {
			case "Shop", "Tavern":
				s.u32(0x76543210)
			case "Outpost":
				for i := range 4 {
					s.u32(uint32(0x5678 + i))
				}
				s.u16(2)
				s.raw(16, 0x52)
			}
		case "Effect", "Effect_DirectDamage":
			s.raw(7, 0x47)
			if class == "Effect_DirectDamage" {
				s.raw(24, 0x24)
			}
		case "VirtualCaster":
			s.raw(7, 0x64)
		case "Sack":
			s.u32(19)
			s.u32(0)
			s.u32(20)
			s.u32(21)
		case "SpellEffect", "PointEffect", "AreaEffect", "SpellTransport":
			s.raw(2, 0x23)
			switch class {
			case "PointEffect":
				s.null()
				s.u32(0x87654321)
			case "AreaEffect":
				s.raw(4, 0x45)
				s.u16(0x6789)
				s.null()
			case "SpellTransport":
				s.null()
				s.null()
				s.u16(0x6789)
			}
		default:
			panic("missing literal class fixture: " + class)
		}
	}
}

func TestDocumentData1115EveryClassHasIndependentLiteralCoverage(t *testing.T) {
	classes := []string{"Token", "Item", "Shield", "Armor", "Weapon", "Building", "Effect", "Sack", "VirtualCaster", "SpellEffect", "PointEffect", "AreaEffect", "SpellTransport", "Effect_DirectDamage", "Outpost", "Tavern", "Shop", "Diary", "Spell", "Spellbook", "Player", "Unit", "Humanoid", "Human"}
	want := append([]string(nil), classes...)
	sort.Strings(want)
	if !reflect.DeepEqual(want, documentSortedKeys(programmes)) {
		t.Fatal("class programme census differs from the independently spelled fixtures")
	}
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			source, literalBody := documentRootSack1115Literal(t, func(s *stream) { documentClass1115Literal(s, class) })
			wantBody := append([]byte(nil), literalBody...)
			d, err := DecodeDocumentData(source)
			if err != nil {
				t.Fatal(err)
			}
			clear(source)
			clear(literalBody)
			documentRecordByClass(t, &d, class)
			wire, stored := documentDataGobCopy(t, d)
			b, err := EncodeDocumentData(stored)
			if err != nil {
				t.Fatal(err)
			}
			f, err := Open(b)
			if err != nil || !bytes.Equal(f.Body, wantBody) {
				t.Fatalf("literal %s wire changed: %v", class, err)
			}
			reloaded, err := DecodeDocumentData(b)
			if err != nil {
				t.Fatal(err)
			}
			again, _ := documentDataGobCopy(t, reloaded)
			if !bytes.Equal(wire, again) {
				t.Fatal("class DTO changed across encode/reload")
			}
		})
	}
}

func TestDocumentData1115SparseBookAndWornViewsAreDerived(t *testing.T) {
	source, literal := documentRootSack1115Literal(t, func(s *stream) {
		s.human(wantChar{
			name: "sparse views", runtimeID: 300, journal: 3,
			worn:  []wantPiece{{class: "Armor", code: 0x1234}},
			items: []wantPiece{{class: "Shield", code: 0x2345}},
			book: func(s *stream) {
				s.u8(1)
				s.u32(0x12345678)
				s.u32(4)
				s.null()
				s.obj("Spell")
				index := s.next - 1
				s.u8(6)
				s.u8(9)
				s.u8(10)
				s.u16(0x1234)
				s.u32(0xf0000006)
				s.backref(index)
			},
		})
	})
	want := append([]byte(nil), literal...)
	d, err := DecodeDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	clear(source)
	clear(literal)
	_, stored := documentDataGobCopy(t, d)
	internal, err := saveDocumentFromData(stored)
	if err != nil {
		t.Fatal(err)
	}
	h := internal.archive.world.sacks[0].RefSlots["Contents"][0]
	if len(h.SpellSlots) != 3 || h.SpellSlots[0] != nil || h.SpellSlots[1] != h.SpellSlots[2] || len(h.Refs["Spells"]) != 2 || len(h.WornSlots) != 12 || h.WornSlots[0].Class != "Armor" || h.WornSlots[1] != nil || len(h.Refs["Worn"]) != 1 || len(h.Refs["Inventory"]) != 1 || h.Refs["Diary"][0].Class != "Diary" {
		t.Fatal("derived book/worn/inventory views changed nullable slots or aliases")
	}
	// Misleading compact copies cannot override RefSlots in the export path.
	h.SpellSlots = nil
	h.WornSlots = nil
	h.Refs["Spells"] = nil
	h.Refs["Worn"] = nil
	rebuilt, err := saveDocumentToData(internal)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EncodeDocumentData(rebuilt)
	if err != nil {
		t.Fatal(err)
	}
	f, err := Open(b)
	if err != nil || !bytes.Equal(f.Body, want) {
		t.Fatalf("derived views became a competing serialization authority: %v", err)
	}
}

func TestDocumentData1115CloneOwnsEveryNestedAllocation(t *testing.T) {
	d := documentDataFixture(t)
	h := documentRecordByClass(t, &d, "Human")
	shared := bytes.Repeat([]byte{0x47}, 24)
	for i := range h.Raw {
		if len(h.Raw[i].Bytes) == 24 {
			h.Raw[i].Bytes = shared
		}
	}
	d.Campaign.Markers = append(d.Campaign.Markers, d.Campaign.Markers[0])
	clone, err := CloneDocumentData(d)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := documentDataGobCopy(t, clone)
	// Poison all source leaves. Every slice/world/inline subtree must be owned
	// when the native loader adopts the clone, not just one selected raw array.
	var poison func(reflect.Value)
	poison = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Pointer:
			if !v.IsNil() {
				poison(v.Elem())
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				poison(v.Field(i))
			}
		case reflect.Array, reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				poison(v.Index(i))
			}
		case reflect.String:
			v.SetString("poisoned")
		case reflect.Uint8, reflect.Uint16, reflect.Uint32:
			v.SetUint(v.Uint() ^ 7)
		case reflect.Int32:
			v.SetInt(v.Int() ^ 7)
		}
	}
	poison(reflect.ValueOf(&d).Elem())
	again, _ := documentDataGobCopy(t, clone)
	if !bytes.Equal(want, again) {
		t.Fatal("native clone retained a nested source allocation")
	}
	second, err := CloneDocumentData(clone)
	if err != nil {
		t.Fatal(err)
	}
	poison(reflect.ValueOf(&clone).Elem())
	again, _ = documentDataGobCopy(t, second)
	if !bytes.Equal(want, again) {
		t.Fatal("subsequent native clone is not independently owned")
	}
	// Repeated raw byte slices are values, not archive-object aliases.
	r := documentRecordByClass(t, &second, "Human")
	var raw24 [][]byte
	for _, field := range r.Raw {
		if len(field.Bytes) == 24 {
			raw24 = append(raw24, field.Bytes)
		}
	}
	before := raw24[1][0]
	raw24[0][0] ^= 0xff
	if raw24[1][0] != before {
		t.Fatal("clone preserved an incidental shared raw slice alias")
	}
	for _, self := range []bool{false, true} {
		cycle, err := DecodeDocumentData(documentCycleLiteral(t, self))
		if err != nil {
			t.Fatal(err)
		}
		owned, err := CloneDocumentData(cycle)
		if err != nil {
			t.Fatal(err)
		}
		x, _ := documentDataGobCopy(t, cycle)
		y, _ := documentDataGobCopy(t, owned)
		if !bytes.Equal(x, y) {
			t.Fatal("native clone changed repeated/cyclic object indices")
		}
	}
}

func TestDocumentData1115ExactOriginsExcludeInlineAndIgnoreKeys(t *testing.T) {
	source, _, _, _ := saveDocument1115Literal(t)
	d, origins, err := DecodeDocumentDataWithOrigins(source)
	if err != nil {
		t.Fatal(err)
	}
	want := []DocumentObjectOrigin{{2, 1}, {4, 2}, {6, 3}, {8, 4}, {10, 5}, {12, 6}, {14, 7}, {16, 8}, {18, 9}}
	if !reflect.DeepEqual(origins, want) || len(origins) != len(d.Objects) {
		t.Fatalf("literal origin rows differ: %+v", origins)
	}
	clear(source)
	if d.Players[1] != d.Players[2] || d.DeadActors[0] != d.DeadActors[1] || d.World.Sacks[0] != d.World.Sacks[1] {
		t.Fatal("origin collection changed aliases")
	}
	clone, err := CloneDocumentData(d)
	if err != nil {
		t.Fatal(err)
	}
	internal, err := saveDocumentFromData(clone)
	if err != nil || len(internal.archive.origins) != 0 {
		t.Fatalf("source bindings entered native persisted state: %v", err)
	}
	for _, self := range []bool{false, true} {
		source := documentCycleLiteral(t, self)
		data, rows, err := DecodeDocumentDataWithOrigins(source)
		if err != nil {
			t.Fatal(err)
		}
		clear(source)
		want := []DocumentObjectOrigin{{2, 1}, {4, 2}}
		if self {
			want = want[:1]
		}
		if !reflect.DeepEqual(rows, want) || !reflect.DeepEqual(data.World.Effects, []uint16{1, 1}) {
			t.Fatalf("cycle/alias origins differ: %+v", rows)
		}
		for i := range data.Objects {
			if *documentValue1115(t, &data.Objects[i], "Identity") != 0 {
				t.Fatal("zero-key literal was changed to synthesize an origin")
			}
		}
	}
	// Local IDs deliberately differ from tag order: sorted Diary comes first
	// in the DTO; inventory/book/worn preceded it on the wire. The same clone
	// pointer provides each pair, without a numeric-index or key-value join.
	source, _ = documentRootSack1115Literal(t, func(s *stream) {
		s.human(wantChar{
			name: "origin order", runtimeID: 300, journal: 1, spells: 2,
			items: []wantPiece{{class: "Shield", code: 0x2345}},
			worn:  []wantPiece{{class: "Armor", code: 0x1234}},
		})
	})
	data, rows, err := DecodeDocumentDataWithOrigins(source)
	if err != nil {
		t.Fatal(err)
	}
	want = []DocumentObjectOrigin{{2, 1}, {4, 2}, {12, 3}, {6, 4}, {8, 5}, {10, 6}}
	if !reflect.DeepEqual(rows, want) || data.Objects[2].Class != "Diary" || data.Objects[3].Class != "Shield" {
		t.Fatalf("origin binding guessed index order instead of clone identity: %+v", rows)
	}
	for _, n := range []int{0, 19, len(source) - 1} {
		data, rows, err := DecodeDocumentDataWithOrigins(source[:n])
		if err == nil || !reflect.DeepEqual(data, DocumentData{}) || rows != nil {
			t.Fatal("incomplete source returned partial import bindings")
		}
	}
}

func TestDocumentData1115CloneChecksWireNestingAndDecodedSize(t *testing.T) {
	for _, n := range []int{maxWalkDepth, maxWalkDepth + 1} {
		d, err := DecodeDocumentData(documentCycleLiteral(t, true))
		if err != nil {
			t.Fatal(err)
		}
		base := d.Objects[0]
		d.Objects = make([]DocumentRecordData, n)
		for i := range d.Objects {
			d.Objects[i] = base
			next := uint16(i + 2)
			if i == n-1 {
				next = 0
			}
			d.Objects[i].RefSlots = []DocumentRefsData{{"AE44", []uint16{next}}}
		}
		clone, err := CloneDocumentData(d)
		if n == maxWalkDepth {
			if err != nil {
				t.Fatalf("valid archive nesting rejected: %v", err)
			}
			if _, err = EncodeDocumentData(clone); err != nil {
				t.Fatalf("accepted nesting cannot be encoded: %v", err)
			}
		} else if err == nil || !reflect.DeepEqual(clone, DocumentData{}) || !strings.Contains(err.Error(), "archive nesting") {
			t.Fatalf("native adoption did not reject wire nesting: %v", err)
		}
	}

	// The DTO fits its bounded ownership budget, but these separately owned
	// Diary arrays exceed the decoded archive cap. Clone refuses without ever
	// constructing a SAV byte array, even though the source slices alias.
	source, _ := documentRootSack1115Literal(t, func(s *stream) { documentClass1115Literal(s, "Diary") })
	d, err := DecodeDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	diary := d.Objects[1]
	for i := range diary.Raw {
		switch diary.Raw[i].Name {
		case "Journal":
			diary.Raw[i].Bytes = make([]byte, 4*maxListElements)
		case "JournalWords":
			diary.Raw[i].Bytes = nil
		}
	}
	*documentCountForField(t, &diary, "Journal") = maxListElements
	*documentCountForField(t, &diary, "JournalWords") = 0
	d.Objects = d.Objects[:1]
	var ids []uint16
	for range 129 {
		d.Objects = append(d.Objects, diary)
		ids = append(ids, uint16(len(d.Objects)))
	}
	*documentRefsForField(t, &d.Objects[0], "Contents") = ids
	*documentCountForField(t, &d.Objects[0], "Contents") = uint32(len(ids))
	if out, err := CloneDocumentData(d); err == nil || !reflect.DeepEqual(out, DocumentData{}) || !strings.Contains(err.Error(), "decoded archive byte bound") {
		t.Fatalf("native adoption accepted overlarge decoded document: %v", err)
	}
}

func TestDocumentData1115EmptyCampaignIsCanonicalAcrossNativeGob(t *testing.T) {
	// Independent complete campaign with zero children and zero markers. The
	// parser allocates those empty lists; ordinary gob decodes them as nil.
	source, _, _, campaign := saveDocument1115Literal(t)
	source = append(source[:len(source)-len(campaign)], make([]byte, 104)...)
	d, err := DecodeDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	before, err := CloneDocumentData(d)
	if err != nil {
		t.Fatal(err)
	}
	_, persisted := documentDataGobCopy(t, before)
	after, err := CloneDocumentData(persisted)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("native canonical model changed: Campaign.Children nil=%t -> %t, Campaign.Markers nil=%t -> %t", before.Campaign.Children == nil, after.Campaign.Children == nil, before.Campaign.Markers == nil, after.Campaign.Markers == nil)
	}
	if !reflect.DeepEqual(d, before) || before.Campaign.Children != nil || before.Campaign.Markers != nil {
		t.Fatal("decode/clone did not select the same canonical empty-list representation")
	}
}
