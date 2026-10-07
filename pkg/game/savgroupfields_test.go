package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func groupFieldsLiteral1115() sav.DocumentRecordData {
	ai := make([]byte, 80)
	copy(ai[76:], []byte{0xab, 0xcd, 0xef, 0x91})
	return sav.DocumentRecordData{Class: "Group",
		Values:   []sav.DocumentValueData{{Name: "G1C", Value: 19}, {Name: "G40", Value: 29}, {Name: "G44", Value: 39}},
		Raw:      []sav.DocumentRawData{{Name: "G20", Bytes: []byte{1, 0}}, {Name: "G3C", Bytes: ai}, {Name: "G4C", Bytes: []byte{2, 0, 3, 0}}},
		Counts:   []sav.DocumentCountData{{Name: "Actors", Count: 1}, {Name: "G20", Count: 1}, {Name: "G4C", Count: 2}},
		RefSlots: []sav.DocumentRefsData{{Name: "Actors", Objects: []uint16{7}}}}
}

func currentGroupFieldsLiteral1115() (sim.SavedGroup, []uint16) {
	g := sim.SavedGroup{ID: 919, Selector: 0xfedcba98, Reference: sim.SavedGroupReference{Key: 123}, Owner: sim.SavedGroupReference{Key: 456},
		Words: []uint16{0xff00, 0x1201, 0xff00}, Path: []uint16{0x1002, 0x3040},
		Members: []sim.SavedGroupMember{{Entity: 0, Bound: true, Archive: 313}, {}, {Archive: 414}}}
	for i := range g.AI {
		g.AI[i] = byte(i + 47)
	}
	return g, []uint16{31, 0, 27}
}

func orderFieldsLiteral1115(class string) sav.DocumentRecordData {
	raw := make([]byte, 148)
	copy(raw[144:], []byte{0xde, 0xad, 0xbe, 0xef})
	// Only direct order fields are this helper's validation domain. Unrelated
	// fields remain intact; complete class/graph validation belongs to caller.
	return sav.DocumentRecordData{Class: class,
		Values:   []sav.DocumentValueData{{Name: "Identity", Value: 0x12345678}},
		Raw:      []sav.DocumentRawData{{Name: "Block12", Bytes: []byte{1, 2, 3}}, {Name: "U158", Bytes: raw}, {Name: "U158_90", Bytes: []byte{1, 0}}, {Name: "U50", Bytes: []byte{9, 0, 0, 0}}},
		Counts:   []sav.DocumentCountData{{Name: "U158_90", Count: 1}},
		RefSlots: []sav.DocumentRefsData{{Name: "HeldWeapon", Objects: []uint16{0}}}}
}

func currentOrderFieldsLiteral1115() sim.SavedActorOrder {
	o := sim.SavedActorOrder{Entity: 0, State: 0xf1234567, Patrol: []uint16{0x5678, 0xffff, 0x5678, 0}}
	for i := range o.Raw {
		o.Raw[i] = byte(i + 53)
	}
	return o
}

func TestSavedGroupFields1115CurrentListsKeysNullsAndTransportTail(t *testing.T) {
	record := groupFieldsLiteral1115()
	shared := record
	group, members := currentGroupFieldsLiteral1115()
	if err := projectSavedGroupFields(&record, group, members, 0, 0x88776655); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(shared, groupFieldsLiteral1115()) {
		t.Fatal("projection mutated pre-existing shared slices")
	}
	if !reflect.DeepEqual(record.Values, []sav.DocumentValueData{{Name: "G1C", Value: 0xfedcba98}, {Name: "G40", Value: 0}, {Name: "G44", Value: 0x88776655}}) || !reflect.DeepEqual(record.Counts, []sav.DocumentCountData{{Name: "Actors", Count: 3}, {Name: "G20", Count: 3}, {Name: "G4C", Count: 2}}) {
		t.Fatal("selector/independent caller keys/counts differ", record)
	}
	if !bytes.Equal(record.Raw[0].Bytes, []byte{0, 255, 1, 18, 0, 255}) || !bytes.Equal(record.Raw[2].Bytes, []byte{2, 16, 64, 48}) || !bytes.Equal(record.Raw[1].Bytes[:76], group.AI[:]) || !bytes.Equal(record.Raw[1].Bytes[76:], []byte{0xab, 0xcd, 0xef, 0x91}) || !reflect.DeepEqual(record.RefSlots[0].Objects, []uint16{31, 0, 27}) {
		t.Fatal("lists were sorted/reversed/merged or pointer tail was overwritten", record)
	}
	input, inputMembers := currentGroupFieldsLiteral1115()
	if !reflect.DeepEqual(group, input) || !reflect.DeepEqual(members, inputMembers) {
		t.Fatal("projection changed input Group or mapped members")
	}
	clear(group.Words)
	clear(group.Path)
	clear(group.Members)
	clear(members)
	if !bytes.Equal(record.Raw[0].Bytes, []byte{0, 255, 1, 18, 0, 255}) || !reflect.DeepEqual(record.RefSlots[0].Objects, []uint16{31, 0, 27}) {
		t.Fatal("projected lists share input storage")
	}
	clear(record.Raw[1].Bytes)
	if !reflect.DeepEqual(shared, groupFieldsLiteral1115()) {
		t.Fatal("projected AI shares source storage")
	}
}

func TestSavedOrderFields1115CurrentRawStateAndIndependentRing(t *testing.T) {
	for _, class := range []string{"Unit", "Humanoid", "Human"} {
		t.Run(class, func(t *testing.T) {
			record := orderFieldsLiteral1115(class)
			shared := record
			order := currentOrderFieldsLiteral1115()
			if err := projectSavedOrderFields(&record, order); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(shared, orderFieldsLiteral1115(class)) || !reflect.DeepEqual(order, currentOrderFieldsLiteral1115()) {
				t.Fatal("projection mutated source slices or input order")
			}
			if !bytes.Equal(record.Raw[1].Bytes[:144], order.Raw[:]) || !bytes.Equal(record.Raw[1].Bytes[144:], []byte{0xde, 0xad, 0xbe, 0xef}) || !bytes.Equal(record.Raw[2].Bytes, []byte{0x78, 0x56, 0xff, 0xff, 0x78, 0x56, 0, 0}) || !bytes.Equal(record.Raw[3].Bytes, []byte{0x67, 0x45, 0x23, 0xf1}) || record.Counts[0].Count != 4 {
				t.Fatal("order state/ring/tail differs", record)
			}
			if !reflect.DeepEqual(record.Values, shared.Values) || !reflect.DeepEqual(record.RefSlots, shared.RefSlots) || !bytes.Equal(record.Raw[0].Bytes, []byte{1, 2, 3}) {
				t.Fatal("unowned actor fields changed")
			}
			clear(order.Patrol)
			clear(record.Raw[1].Bytes)
			if !bytes.Equal(record.Raw[2].Bytes, []byte{0x78, 0x56, 0xff, 0xff, 0x78, 0x56, 0, 0}) || !reflect.DeepEqual(shared, orderFieldsLiteral1115(class)) {
				t.Fatal("order projection shares mutable storage")
			}
		})
	}
}

func TestSavedGroupAndOrderFields1115LateRefusalsAreAtomic(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*sav.DocumentRecordData, *sim.SavedGroup, *[]uint16)
	}{
		{"authored", func(_ *sav.DocumentRecordData, g *sim.SavedGroup, _ *[]uint16) { g.Authored = true }},
		{"wrong class", func(r *sav.DocumentRecordData, _ *sim.SavedGroup, _ *[]uint16) { r.Class = "Player" }},
		{"last scalar absent", func(r *sav.DocumentRecordData, _ *sim.SavedGroup, _ *[]uint16) { r.Values = r.Values[:2] }},
		{"duplicate scalar", func(r *sav.DocumentRecordData, _ *sim.SavedGroup, _ *[]uint16) { r.Values[2].Name = "G40" }},
		{"AI short", func(r *sav.DocumentRecordData, _ *sim.SavedGroup, _ *[]uint16) { r.Raw[1].Bytes = r.Raw[1].Bytes[:79] }},
		{"AI long", func(r *sav.DocumentRecordData, _ *sim.SavedGroup, _ *[]uint16) {
			r.Raw[1].Bytes = append(r.Raw[1].Bytes, 0)
		}},
		{"path count", func(r *sav.DocumentRecordData, _ *sim.SavedGroup, _ *[]uint16) { r.Counts[2].Count++ }},
		{"path absent", func(r *sav.DocumentRecordData, _ *sim.SavedGroup, _ *[]uint16) { r.Raw = r.Raw[:2] }},
		{"members count", func(r *sav.DocumentRecordData, _ *sim.SavedGroup, _ *[]uint16) { r.Counts[0].Count++ }},
		{"members absent", func(r *sav.DocumentRecordData, _ *sim.SavedGroup, _ *[]uint16) { r.RefSlots = nil }},
		{"mapped count", func(_ *sav.DocumentRecordData, _ *sim.SavedGroup, m *[]uint16) { *m = (*m)[:2] }},
		{"lost unbound member", func(_ *sav.DocumentRecordData, _ *sim.SavedGroup, m *[]uint16) { (*m)[2] = 0 }},
		{"lost native zero member", func(_ *sav.DocumentRecordData, _ *sim.SavedGroup, m *[]uint16) { (*m)[0] = 0 }},
		{"invented null member", func(_ *sav.DocumentRecordData, _ *sim.SavedGroup, m *[]uint16) { (*m)[1] = 31 }},
		{"mapped index bound", func(_ *sav.DocumentRecordData, _ *sim.SavedGroup, m *[]uint16) { (*m)[2] = 0x8000 }},
		{"list bound", func(_ *sav.DocumentRecordData, g *sim.SavedGroup, _ *[]uint16) { g.Path = make([]uint16, 65537) }},
		{"existing list bound", func(r *sav.DocumentRecordData, _ *sim.SavedGroup, _ *[]uint16) { r.Counts[2].Count = 65537 }},
	} {
		t.Run("group/"+tc.name, func(t *testing.T) {
			r := groupFieldsLiteral1115()
			g, m := currentGroupFieldsLiteral1115()
			tc.mutate(&r, &g, &m)
			before := actorProjection1115Copy(t, sav.DocumentData{Objects: []sav.DocumentRecordData{r}}).Objects[0]
			if err := projectSavedGroupFields(&r, g, m, 1, 2); err == nil || !reflect.DeepEqual(r, before) {
				t.Fatal("invalid Group accepted or partially published", err)
			}
		})
	}
	for _, tc := range []struct {
		name   string
		mutate func(*sav.DocumentRecordData, *sim.SavedActorOrder)
	}{
		{"authored", func(_ *sav.DocumentRecordData, o *sim.SavedActorOrder) { o.Authored = true }},
		{"wrong class", func(r *sav.DocumentRecordData, _ *sim.SavedActorOrder) { r.Class = "Building" }},
		{"short state", func(r *sav.DocumentRecordData, _ *sim.SavedActorOrder) { r.Raw[3].Bytes = r.Raw[3].Bytes[:3] }},
		{"short order", func(r *sav.DocumentRecordData, _ *sim.SavedActorOrder) { r.Raw[1].Bytes = r.Raw[1].Bytes[:147] }},
		{"long order", func(r *sav.DocumentRecordData, _ *sim.SavedActorOrder) { r.Raw[1].Bytes = append(r.Raw[1].Bytes, 0) }},
		{"patrol count", func(r *sav.DocumentRecordData, _ *sim.SavedActorOrder) { r.Counts[0].Count++ }},
		{"patrol count absent", func(r *sav.DocumentRecordData, _ *sim.SavedActorOrder) { r.Counts = nil }},
		{"patrol raw absent", func(r *sav.DocumentRecordData, _ *sim.SavedActorOrder) { r.Raw = append(r.Raw[:2], r.Raw[3:]...) }},
		{"duplicate raw", func(r *sav.DocumentRecordData, _ *sim.SavedActorOrder) { r.Raw[2].Name = "U158" }},
		{"list bound", func(_ *sav.DocumentRecordData, o *sim.SavedActorOrder) { o.Patrol = make([]uint16, 65537) }},
	} {
		t.Run("order/"+tc.name, func(t *testing.T) {
			r, o := orderFieldsLiteral1115("Human"), currentOrderFieldsLiteral1115()
			tc.mutate(&r, &o)
			before := actorProjection1115Copy(t, sav.DocumentData{Objects: []sav.DocumentRecordData{r}}).Objects[0]
			if err := projectSavedOrderFields(&r, o); err == nil || !reflect.DeepEqual(r, before) {
				t.Fatal("invalid order accepted or partially published", err)
			}
		})
	}
}

func TestSavedGroupAndOrderFields1115EmptyAndListBounds(t *testing.T) {
	for _, length := range []int{0, 65535, 65536} {
		r := groupFieldsLiteral1115()
		g := sim.SavedGroup{Words: make([]uint16, length), Path: make([]uint16, length)}
		if length > 0 {
			g.Words[length-1], g.Path[length-1] = 0x1234, 0x5678
		}
		if err := projectSavedGroupFields(&r, g, nil, 0, 0); err != nil || len(r.Raw[0].Bytes) != length*2 || len(r.Raw[2].Bytes) != length*2 || r.Counts[0].Count != 0 || len(r.RefSlots[0].Objects) != 0 {
			t.Fatal("empty or maximum Group list", length, err)
		}
		o := orderFieldsLiteral1115("Unit")
		if err := projectSavedOrderFields(&o, sim.SavedActorOrder{Patrol: slices.Clone(g.Words)}); err != nil || len(o.Raw[2].Bytes) != length*2 {
			t.Fatal("empty or maximum patrol list", length, err)
		}
		if length > 0 && (binary.LittleEndian.Uint16(r.Raw[0].Bytes[length*2-2:]) != 0x1234 || binary.LittleEndian.Uint16(r.Raw[2].Bytes[length*2-2:]) != 0x5678 || binary.LittleEndian.Uint16(o.Raw[2].Bytes[length*2-2:]) != 0x1234) {
			t.Fatal("last list element truncated", length)
		}
		g.Members = make([]sim.SavedGroupMember, length)
		members := make([]uint16, length)
		if length > 0 {
			g.Members[length-1] = sim.SavedGroupMember{Bound: true}
			members[length-1] = 0x7fff
		}
		if err := projectSavedGroupFields(&r, g, members, 0, 0); err != nil || int(r.Counts[0].Count) != length || len(r.RefSlots[0].Objects) != length || (length > 0 && r.RefSlots[0].Objects[length-1] != 0x7fff) {
			t.Fatal("maximum count/nulls/local reference", length, err)
		}
	}
	if err := projectSavedGroupFields(nil, sim.SavedGroup{}, nil, 0, 0); err == nil {
		t.Fatal("nil Group accepted")
	}
	if err := projectSavedOrderFields(nil, sim.SavedActorOrder{}); err == nil {
		t.Fatal("nil actor accepted")
	}
}

func TestSavedGroupAndOrderFields1115FullContainerReadback(t *testing.T) {
	raw := completeDocumentFixture1115(t, &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}})
	document, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	clear(raw)
	player := &document.Objects[document.Players[1]-1]
	if len(player.Groups) != 1 || len(player.Groups[0].RefSlots) != 1 || len(player.Groups[0].RefSlots[0].Objects) != 1 {
		t.Fatal("literal complete fixture changed")
	}
	members := slices.Clone(player.Groups[0].RefSlots[0].Objects)
	actor := &document.Objects[members[0]-1]
	group, _ := currentGroupFieldsLiteral1115()
	group.Members = []sim.SavedGroupMember{{Archive: 0x4321, Entity: 999, Bound: true}}
	// Different literal tails would expose copying the imported list-pointer
	// bits from current native IDs or zero-initializing them.
	for i := range player.Groups[0].Raw {
		if player.Groups[0].Raw[i].Name == "G3C" {
			copy(player.Groups[0].Raw[i].Bytes[76:], []byte{0xa1, 0xb2, 0xc3, 0xd4})
		}
	}
	for i := range actor.Raw {
		if actor.Raw[i].Name == "U158" {
			copy(actor.Raw[i].Bytes[144:], []byte{0x1a, 0x2b, 0x3c, 0x4d})
		}
	}
	if err := projectSavedGroupFields(&player.Groups[0], group, members, 0, 0); err != nil {
		t.Fatal(err)
	}
	order := currentOrderFieldsLiteral1115()
	if err := projectSavedOrderFields(actor, order); err != nil {
		t.Fatal(err)
	}
	encoded, err := sav.EncodeDocumentData(document)
	if err != nil {
		t.Fatal(err)
	}
	f, err := sav.Open(encoded)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := f.ActorGraph()
	if err != nil || len(graph.Groups) != 1 || len(graph.Actors) != 1 {
		t.Fatal("complete SAV actor graph", err)
	}
	g, a := graph.Groups[0], graph.Actors[0]
	if g.Selector != 0xfedcba98 || g.Reference.Key != 0 || g.Owner.Key != 0 || !slices.Equal(g.Words, []uint16{0xff00, 0x1201, 0xff00}) || !slices.Equal(g.AIWords, []uint16{0x1002, 0x3040}) || !bytes.Equal(g.AI[:76], group.AI[:]) || !bytes.Equal(g.AI[76:], []byte{0xa1, 0xb2, 0xc3, 0xd4}) || len(g.Members) != 1 || g.Members[0] != a.ArchiveIndex {
		t.Fatal("complete SAV Group projection differs", g)
	}
	if a.ActorState != 0xf1234567 || !bytes.Equal(a.Order[:144], order.Raw[:]) || !bytes.Equal(a.Order[144:], []byte{0x1a, 0x2b, 0x3c, 0x4d}) || !slices.Equal(a.Patrol, []uint16{0x5678, 0xffff, 0x5678, 0}) {
		t.Fatal("complete SAV actor order projection differs", a.ActorState, a.Patrol)
	}
}
