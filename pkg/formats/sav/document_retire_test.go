package sav

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// Format-only literal graph, not a claim about legal gameplay contents:
// A(Sack) -> B(Sack) -> W(Weapon) -> E(Effect), P(Spell)
//
//	-> C(Sack) -> D(Weapon) -> the SAME E and P.
//
// The independently emitted survivor form is A -> C -> D -> E,P.
func retireDocument1115Literal(t *testing.T, survivors bool) DocumentData {
	t.Helper()
	source, _ := documentRootSack1115Literal(t, func(s *stream) {
		var effect, spell uint16
		weapon := func(code uint16, shared bool) {
			s.obj("Weapon")
			s.token(0x1413, 3, 5, 300, 7, 0)
			s.u32(1)
			if shared {
				s.backref(effect)
			} else {
				documentClass1115Literal(s, "Effect")
				effect = s.next - 1
			}
			s.u16(code)
			s.u16(7)
			s.u8(0x44)
			s.u8(0x45)
			s.u8(0x46)
			s.u16(0x4848)
			s.u16(0x4a4a)
			s.u8(0x47)
			s.raw(24, 0x52)
			s.raw(22, 0x6a)
			s.u8(0x50)
			if shared {
				s.backref(spell)
			} else {
				documentClass1115Literal(s, "Spell")
				spell = s.next - 1
			}
		}
		if !survivors {
			s.obj("Sack")
			s.token(0x2221, 17, 19, 501, 0, 0)
			s.u32(21)
			s.u32(2)
			weapon(0x1432, false)
		}
		s.obj("Sack")
		s.token(0x3433, 23, 29, 601, 0, 0)
		s.u32(31)
		s.u32(1)
		weapon(0x2531, !survivors)
		s.u32(32)
		s.u32(33)
		if !survivors {
			s.u32(22)
			s.u32(23)
		}
	})
	d, err := DecodeDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	clear(source)
	wantClasses := []string{"Sack", "Sack", "Weapon", "Effect", "Spell", "Sack", "Weapon"}
	otherSack := uint16(6)
	if survivors {
		wantClasses = []string{"Sack", "Sack", "Weapon", "Effect", "Spell"}
		otherSack = 2
	}
	var classes []string
	for _, r := range d.Objects {
		classes = append(classes, r.Class)
	}
	if !reflect.DeepEqual(classes, wantClasses) {
		t.Fatal("independent archive encounter order changed", classes)
	}
	d.World.Sacks = []uint16{1, otherSack, otherSack}
	// These scalars deliberately equal the retired local IDs. They are NOT
	// archive edges and must remain unchanged even when those objects vanish.
	*documentValue1115(t, &d.Objects[0], "Identity") = 3
	*documentValue1115(t, &d.Objects[0], "Reference") = 2
	d.World.TerrainIdentity = 2
	d.World.Cells = []DocumentCellData{{Cell: 0x100f, Cost: 17, Static: 0x10,
		GroundActor: 2, Layers: [6]uint32{3, 2, 3, 2, 3, 2}, Residue03: 0x35, Residue32: 0xcafe}}
	return d
}

func retireDocument1115Pending(t *testing.T) DocumentData {
	t.Helper()
	d := retireDocument1115Literal(t, false)
	*documentRefsForField(t, &d.Objects[0], "Contents") = []uint16{6}
	return d
}

func assertRetireDocument1115Unchanged(t *testing.T, before []byte, input DocumentData) {
	t.Helper()
	after, _ := documentDataGobCopy(t, input)
	if !bytes.Equal(before, after) {
		t.Fatal("retirement changed caller-owned input")
	}
}

func TestRetireDocument1115ExplicitSackAndItemKeepSharedChildren(t *testing.T) {
	for _, retired := range [][]uint16{{2, 3}, {3, 2}} {
		input := retireDocument1115Pending(t)
		requireCurrentActionRetirement(t, input)
		before, _ := documentDataGobCopy(t, input)
		if _, err := CloneDocumentData(input); err == nil {
			t.Fatal("native adoption silently collected the disconnected source records")
		}
		want := retireDocument1115Literal(t, true)
		got, permutation, err := RetireDocumentData(input, retired)
		if err != nil || !reflect.DeepEqual(permutation, []uint16{0, 1, 0, 0, 4, 5, 2, 3}) || !reflect.DeepEqual(got, want) {
			t.Fatalf("explicit retirement differs from independent survivor literal: permutation %v, error %v", permutation, err)
		}
		assertRetireDocument1115Unchanged(t, before, input)
		raw, err := EncodeDocumentData(got)
		if err != nil {
			t.Fatal(err)
		}
		loaded, err := DecodeDocumentData(raw)
		if err != nil || !reflect.DeepEqual(loaded, want) {
			t.Fatal("retirement output is not a canonical whole-container roundtrip", err)
		}
		// Poison all output allocations, including nested raw values, roots,
		// state and campaign. The input must remain byte-for-byte unchanged.
		retireDocument1115Poison(reflect.ValueOf(&got).Elem())
		assertRetireDocument1115Unchanged(t, before, input)
	}
}

func retireDocument1115Poison(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			retireDocument1115Poison(v.Elem())
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			retireDocument1115Poison(v.Field(i))
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			retireDocument1115Poison(v.Index(i))
		}
	case reflect.String:
		v.SetString("poisoned")
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		v.SetUint(v.Uint() ^ 7)
	case reflect.Int32:
		v.SetInt(v.Int() ^ 7)
	}
}

func TestRetireDocument1115WholeWorldRootsInlineGroupsAndNoop(t *testing.T) {
	for _, remove := range []bool{false, true} {
		want := documentDataFixture(t)
		if len(want.Objects) != 9 || want.Objects[7].Class != "Sack" || want.Objects[8].Class != "Item" {
			t.Fatal("literal complete-world graph changed")
		}
		input := reverseDocumentIndices(t, want)
		var retired []uint16
		if remove {
			input.World.Sacks = nil
			retired = []uint16{2, 1}
			want.World.Sacks = nil
			want.Objects = want.Objects[:7]
		}
		before, _ := documentDataGobCopy(t, input)
		got, permutation, err := RetireDocumentData(input, retired)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatal("root/inline/Group remapping changed retained complete state", remove, err)
		}
		for old := 1; old <= 9; old++ {
			expected := uint16(10 - old)
			if remove && old <= 2 {
				expected = 0
			}
			if len(permutation) != 10 || permutation[0] != 0 || permutation[old] != expected {
				t.Fatal("full old-index permutation is incomplete", permutation)
			}
		}
		retireDocument1115Poison(reflect.ValueOf(&got).Elem())
		assertRetireDocument1115Unchanged(t, before, input)
	}
}

func TestRetireDocument1115ArchiveCyclesAndRetiredOutgoingEdges(t *testing.T) {
	for _, self := range []bool{false, true} {
		want, err := DecodeDocumentData(documentCycleLiteral(t, self))
		if err != nil {
			t.Fatal(err)
		}
		_, input := documentDataGobCopy(t, want)
		source, _ := documentRootSack1115Literal(t, func(s *stream) { documentClass1115Literal(s, "Sack") })
		extra, err := DecodeDocumentData(source)
		if err != nil {
			t.Fatal(err)
		}
		input.Objects = append(input.Objects, extra.Objects[1])
		input = reverseDocumentIndices(t, input)
		before, _ := documentDataGobCopy(t, input)
		got, permutation, err := RetireDocumentData(input, []uint16{1})
		if err != nil || !reflect.DeepEqual(got, want) || permutation[1] != 0 {
			t.Fatal("retirement changed surviving archive cycle", self, permutation, err)
		}
		assertRetireDocument1115Unchanged(t, before, input)
		// The whole explicitly retired cycle may retain dead-to-dead edges.
		input = want
		input.World.Effects = nil
		retired := []uint16{1}
		if !self {
			retired = append(retired, 2)
		}
		got, permutation, err = RetireDocumentData(input, retired)
		if err != nil || len(got.Objects) != 0 || len(got.World.Effects) != 0 || len(permutation) != len(retired)+1 {
			t.Fatal("explicitly retired cycle was treated as live dangling references", self, err)
		}
		for _, index := range permutation {
			if index != 0 {
				t.Fatal("retired cycle retained a live binding", permutation)
			}
		}
	}
}

func TestRetireDocument1115RefusesDanglingMalformedAndImplicitGC(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*DocumentData, *[]uint16)
	}{
		{"zero retirement", func(_ *DocumentData, r *[]uint16) { *r = []uint16{0} }},
		{"unknown retirement", func(_ *DocumentData, r *[]uint16) { *r = []uint16{8} }},
		{"duplicate retirement", func(_ *DocumentData, r *[]uint16) { *r = []uint16{2, 2} }},
		{"retirement count", func(_ *DocumentData, r *[]uint16) { *r = make([]uint16, 8) }},
		{"live root to retired", func(d *DocumentData, _ *[]uint16) { d.World.Sacks[1] = 2 }},
		{"live archive ref to retired", func(d *DocumentData, _ *[]uint16) { *documentRefsForField(t, &d.Objects[0], "Contents") = []uint16{2} }},
		{"shared child still referenced", func(_ *DocumentData, r *[]uint16) { *r = []uint16{2, 3, 4} }},
		{"unlisted orphan", func(_ *DocumentData, r *[]uint16) { *r = []uint16{2} }},
		{"no implicit garbage collection", func(_ *DocumentData, r *[]uint16) { *r = nil }},
		{"unknown root", func(d *DocumentData, _ *[]uint16) { d.World.Sacks[0] = 65535 }},
		{"unknown surviving ref", func(d *DocumentData, _ *[]uint16) {
			*documentRefsForField(t, &d.Objects[0], "Contents") = []uint16{65535}
		}},
		{"unknown retired ref", func(d *DocumentData, _ *[]uint16) { (*documentRefsForField(t, &d.Objects[1], "Contents"))[0] = 65535 }},
		{"retired unknown class", func(d *DocumentData, _ *[]uint16) { d.Objects[2].Class = "Unknown" }},
		{"retired bad count", func(d *DocumentData, _ *[]uint16) { *documentCountForField(t, &d.Objects[1], "Contents") = 3 }},
		{"retired overflow count", func(d *DocumentData, _ *[]uint16) { *documentCountForField(t, &d.Objects[1], "Contents") = 0xffffffff }},
		{"retired bad scalar width", func(d *DocumentData, _ *[]uint16) { *documentValue1115(t, &d.Objects[2], "F40") = 65536 }},
		{"retired missing scalar", func(d *DocumentData, _ *[]uint16) { d.Objects[2].Values = d.Objects[2].Values[1:] }},
		{"retired duplicate scalar", func(d *DocumentData, _ *[]uint16) {
			d.Objects[2].Values = append(d.Objects[2].Values, d.Objects[2].Values[0])
		}},
		{"retired bad raw extent", func(d *DocumentData, _ *[]uint16) { d.Objects[2].Raw[0].Bytes = []byte{1} }},
		{"retired inactive groups", func(d *DocumentData, _ *[]uint16) { d.Objects[2].Groups = []DocumentRecordData{{Class: "Group"}} }},
		{"retired inactive inline", func(d *DocumentData, _ *[]uint16) {
			d.Objects[2].Inline = []DocumentInlineData{{Name: "Diary", Record: DocumentRecordData{Class: "Spell"}}}
		}},
		{"surviving bad count", func(d *DocumentData, _ *[]uint16) { *documentCountForField(t, &d.Objects[0], "Contents") = 0 }},
		{"nil Sack root", func(d *DocumentData, _ *[]uint16) { d.World.Sacks[0] = 0 }},
		{"wrong root class", func(d *DocumentData, _ *[]uint16) { d.World.Buildings = []uint16{4} }},
		{"wrong document version", func(d *DocumentData, _ *[]uint16) { d.Version++ }},
		{"bad envelope label", func(d *DocumentData, _ *[]uint16) { d.Label = []byte{'x', 0} }},
		{"inactive global value", func(d *DocumentData, _ *[]uint16) { d.Marker, d.GlobalDWord = 0, 1 }},
		{"bad campaign", func(d *DocumentData, _ *[]uint16) { d.Campaign.Base.DWords[5] = 3 }},
		{"duplicate state", func(d *DocumentData, _ *[]uint16) {
			d.State.ValueRecords = append(d.State.ValueRecords, d.State.ValueRecords[0])
		}},
		{"unordered blocks", func(d *DocumentData, _ *[]uint16) { d.World.Blocks = []BlockRecord{{Cell: 2}, {Cell: 1}} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := retireDocument1115Pending(t)
			retired := []uint16{2, 3}
			tc.edit(&input, &retired)
			before, _ := documentDataGobCopy(t, input)
			out, permutation, err := RetireDocumentData(input, retired)
			if err == nil || !reflect.DeepEqual(out, DocumentData{}) || permutation != nil {
				t.Fatal("malformed retirement returned an adopted graph or partial bindings", err)
			}
			assertRetireDocument1115Unchanged(t, before, input)
		})
	}
	// A Group member edge is a real archive reference even though the Group is
	// inline. Removing the separate DeadActors root must not hide that edge.
	input := documentDataFixture(t)
	input.DeadActors = nil
	before, _ := documentDataGobCopy(t, input)
	if out, permutation, err := RetireDocumentData(input, []uint16{2}); err == nil || !reflect.DeepEqual(out, DocumentData{}) || permutation != nil {
		t.Fatal("surviving inline Group reference to retired actor was accepted", err)
	}
	assertRetireDocument1115Unchanged(t, before, input)
}

func TestRetireDocument1115BudgetsAndRetiredWireDepth(t *testing.T) {
	for name, edit := range map[string]func(*DocumentData){
		"object count":  func(d *DocumentData) { d.Objects = make([]DocumentRecordData, maxDocumentDataObjects+1) },
		"root count":    func(d *DocumentData) { d.Players = make([]uint16, maxListElements+1) },
		"terrain count": func(d *DocumentData) { d.World.Cells = make([]DocumentCellData, maxListElements+1) },
		"retired shared byte budget": func(d *DocumentData) {
			span := make([]byte, 4<<20)
			for range 17 {
				d.Objects[2].Raw = append(d.Objects[2].Raw, DocumentRawData{Name: "large", Bytes: span})
			}
		},
		"recursive retired inline": func(d *DocumentData) {
			cycle := make([]DocumentInlineData, 1)
			cycle[0] = DocumentInlineData{Name: "Diary", Record: DocumentRecordData{Class: "Diary", Inline: cycle}}
			d.Objects[2].Inline = cycle
		},
	} {
		t.Run(name, func(t *testing.T) {
			input := retireDocument1115Pending(t)
			edit(&input)
			out, permutation, err := RetireDocumentData(input, []uint16{2, 3})
			if err == nil || !reflect.DeepEqual(out, DocumentData{}) || permutation != nil || !strings.Contains(err.Error(), "bound") && !strings.Contains(err.Error(), "budget") {
				t.Fatal("retirement bypassed an input ownership/depth bound", err)
			}
		})
	}
	for _, n := range []int{maxWalkDepth, maxWalkDepth + 1} {
		input, err := DecodeDocumentData(documentCycleLiteral(t, true))
		if err != nil {
			t.Fatal(err)
		}
		base := input.Objects[0]
		input.Objects = make([]DocumentRecordData, n)
		var retired []uint16
		for i := range input.Objects {
			input.Objects[i] = base
			next := uint16(i + 2)
			if i == n-1 {
				next = 0
			}
			input.Objects[i].RefSlots = []DocumentRefsData{{Name: "AE44", Objects: []uint16{next}}}
			retired = append(retired, uint16(i+1))
		}
		input.World.Effects = nil
		before, _ := documentDataGobCopy(t, input)
		out, permutation, err := RetireDocumentData(input, retired)
		if n == maxWalkDepth {
			if err != nil || len(out.Objects) != 0 || len(permutation) != n+1 {
				t.Fatal("valid retired wire-depth boundary refused", err)
			}
		} else if err == nil || !reflect.DeepEqual(out, DocumentData{}) || permutation != nil || !strings.Contains(err.Error(), "archive nesting") {
			t.Fatal("retirement concealed an over-deep source record chain", err)
		}
		assertRetireDocument1115Unchanged(t, before, input)
	}
}

func TestRetireDocument1115DepthUsesActualRootsNotObjectOrder(t *testing.T) {
	input, err := DecodeDocumentData(documentCycleLiteral(t, true))
	if err != nil {
		t.Fatal(err)
	}
	base := input.Objects[0]
	input.Objects = make([]DocumentRecordData, maxWalkDepth+1)
	for i := range input.Objects {
		input.Objects[i] = base
		next := uint16(i + 2)
		if i == len(input.Objects)-1 {
			next = 0
		}
		input.Objects[i].RefSlots = []DocumentRefsData{{Name: "AE44", Objects: []uint16{next}}}
	}
	// Root2 first reaches exactly the legal depth. Later Root1 only points to
	// an already encountered object. Walking the noncanonical table from1 would
	// invent a too-deep traversal that the actual archive never performs.
	input.World.Effects = []uint16{2, 1}
	got, permutation, err := RetireDocumentData(input, nil)
	if err != nil || permutation[1] != uint16(maxWalkDepth+1) || permutation[2] != 1 {
		t.Fatal("retirement used object table order as wire traversal", permutation, err)
	}
	if _, err := EncodeDocumentData(got); err != nil {
		t.Fatal("accepted depth boundary cannot serialize", err)
	}
}
