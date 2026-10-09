package game

import (
	"bytes"
	"encoding/binary"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func group1155SetRaw(t *testing.T, record *sav.DocumentRecordData, name string, b []byte) {
	t.Helper()
	for i := range record.Raw {
		if record.Raw[i].Name == name {
			record.Raw[i].Bytes = slices.Clone(b)
			return
		}
	}
	t.Fatal("fixture lacks raw field", name)
}

func group1155SetCount(t *testing.T, record *sav.DocumentRecordData, name string, n int) {
	t.Helper()
	for i := range record.Counts {
		if record.Counts[i].Name == name {
			record.Counts[i].Count = uint32(n)
			return
		}
	}
	t.Fatal("fixture lacks count", name)
}

func groups1155Fixture(t *testing.T, duplicateSlots bool) (*FrontEnd, []byte) {
	t.Helper()
	f := newGroupFront(t, -1)
	doc, err := sav.DecodeDocumentData(newGroupLiteral(t, f, duplicateSlots))
	if err != nil {
		t.Fatal(err)
	}
	ordinal := 0
	for _, index := range []uint16{doc.Players[0], doc.Players[3]} {
		player := &doc.Objects[index-1]
		for i := range player.Groups {
			g := &player.Groups[i]
			seed := ordinal + 1
			if ordinal == 1 {
				seed = 1 // Two different empty inline Groups have equal values.
				newGroupSetValue1115(t, g, "G1C", 10)
			}
			ai := make([]byte, 80)
			for j := range ai {
				ai[j] = byte(j + seed)
			}
			ai[0x20] = 0xff
			group1155SetRaw(t, g, "G3C", ai)
			for _, list := range []struct {
				name   string
				values []uint16
			}{
				{"G20", []uint16{uint16(0x8000 + seed), 7, uint16(0x8000 + seed)}},
				{"G4C", []uint16{0x100f, 0x1210}},
			} {
				group1155SetRaw(t, g, list.name, group1155Bytes(list.values))
				group1155SetCount(t, g, list.name, len(list.values))
			}
			ordinal++
		}
	}
	left, right := &doc.Objects[doc.Players[0]-1], &doc.Objects[doc.Players[3]-1]
	g := &left.Groups[2]
	a, b := g.RefSlots[0].Objects[0], g.RefSlots[0].Objects[1]
	g.RefSlots[0].Objects = []uint16{a, 0, b, a}
	group1155SetCount(t, g, "Actors", 4)
	group1155SetCount(t, left, "Actors", 4)
	// B moves across Player containers. Group+44 of the last Group remains
	// LEFT, while the final actor owner is the enclosing RIGHT Player.
	g = &right.Groups[1]
	g.RefSlots[0].Objects = append(g.RefSlots[0].Objects, b)
	group1155SetCount(t, g, "Actors", 2)
	group1155SetCount(t, right, "Actors", 2)
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" && r.Class != "Human" {
			continue
		}
		order := make([]byte, 148)
		order[8], order[10], order[11], order[0x21] = 1, 18, 16, 77
		binary.LittleEndian.PutUint32(order[144:], 0xdeadbeef)
		group1155SetRaw(t, r, "U158", order)
		group1155SetRaw(t, r, "U50", binary.LittleEndian.AppendUint32(nil, 0xb))
		group1155SetRaw(t, r, "U158_90", group1155Bytes([]uint16{0x100f, 0x1210, 0x100f}))
		group1155SetCount(t, r, "U158_90", 3)
	}
	doc, _, err = sav.ReindexDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return f, raw
}

func groups1155Inputs(t *testing.T, raw []byte) (groups1155Source, map[uint16]uint16, sav.DocumentData) {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := groups1155Expected(f)
	if err != nil {
		t.Fatal(err)
	}
	join, err := players1154Origins(raw)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if d := groups1155DocumentDifferences(want, join, &doc); len(d) != 0 {
		t.Fatal("raw Group baseline", d)
	}
	return want, join, doc
}

func TestGroups1155RawDocumentControls(t *testing.T) {
	_, raw := groups1155Fixture(t, false)
	want, join, doc := groups1155Inputs(t, raw)
	g := want.groups[2]
	mutate := func(t *testing.T, change func(*sav.DocumentRecordData)) {
		t.Helper()
		d, err := sav.CloneDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		change(&d.Objects[join[g.player]-1].Groups[g.inline])
		if len(groups1155DocumentDifferences(want, join, &d)) == 0 {
			t.Fatal("raw Group omission/alteration accepted")
		}
	}
	for _, field := range []string{"G1C", "G40", "G44"} {
		t.Run("omit "+field, func(t *testing.T) {
			mutate(t, func(g *sav.DocumentRecordData) {
				g.Values = slices.DeleteFunc(g.Values, func(v sav.DocumentValueData) bool { return v.Name == field })
			})
		})
	}
	for _, field := range []string{"G3C", "G20", "G4C"} {
		t.Run("omit "+field, func(t *testing.T) {
			mutate(t, func(g *sav.DocumentRecordData) {
				g.Raw = slices.DeleteFunc(g.Raw, func(v sav.DocumentRawData) bool { return v.Name == field })
			})
		})
	}
	for _, field := range []string{"Actors", "G20", "G4C"} {
		t.Run("omit count "+field, func(t *testing.T) {
			mutate(t, func(g *sav.DocumentRecordData) {
				g.Counts = slices.DeleteFunc(g.Counts, func(v sav.DocumentCountData) bool { return v.Name == field })
			})
		})
	}
	t.Run("same count reordered members", func(t *testing.T) {
		mutate(t, func(g *sav.DocumentRecordData) {
			g.RefSlots[0].Objects[1], g.RefSlots[0].Objects[2] = g.RefSlots[0].Objects[2], g.RefSlots[0].Objects[1]
		})
	})
	t.Run("normalize raw Document too early", func(t *testing.T) {
		mutate(t, func(record *sav.DocumentRecordData) {
			var members []uint16
			for _, archive := range g.current {
				members = append(members, join[archive])
			}
			record.RefSlots[0].Objects = members
			group1155SetCount(t, record, "Actors", len(members))
		})
	})
	t.Run("replace transport AI pointer bytes", func(t *testing.T) {
		mutate(t, func(g *sav.DocumentRecordData) {
			for i := range g.Raw {
				if g.Raw[i].Name == "G3C" {
					clear(g.Raw[i].Bytes[76:])
				}
			}
		})
	})
	for _, field := range []string{"U158", "U158_90", "U50"} {
		t.Run("omit actor "+field, func(t *testing.T) {
			d, err := sav.CloneDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			for archive := range want.actors {
				r := &d.Objects[join[archive]-1]
				r.Raw = slices.DeleteFunc(r.Raw, func(v sav.DocumentRawData) bool { return v.Name == field })
				break
			}
			if len(groups1155DocumentDifferences(want, join, &d)) == 0 {
				t.Fatal("actor omission accepted")
			}
		})
	}
}

func TestGroups1155LiveNormalizationAndOmissionControls(t *testing.T) {
	f, raw := groups1155Fixture(t, false)
	want, join, _ := groups1155Inputs(t, raw)
	ms, _, err := loadOriginalMission(f, raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(want.groups[2].members) != 4 || len(want.groups[2].current) != 2 || want.groups[2].current[0] != 0 || len(want.groups[4].current) != 2 || want.groups[4].owner.Archive == want.groups[4].player {
		t.Fatal("discriminating alias/null/cross-Player/independent-owner fixture changed")
	}
	if d, p := groups1155LiveDifferences(want, join, ms.savedDocument, ms.World); len(d) != 0 || p.bound != 3 || p.nulls != 1 || p.orders != 3 {
		t.Fatal("normalized live baseline", d, p)
	}
	encoded, err := ms.World.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"AI", "Words", "Path", "swap members", "actor order", "actor Patrol"} {
		t.Run(name, func(t *testing.T) {
			w := &sim.World{}
			if err := w.UnmarshalBinary(encoded); err != nil {
				t.Fatal(err)
			}
			groups, orders, _ := w.SavedGroups()
			switch name {
			case "AI":
				groups[2].AI[5] = 0
			case "Words":
				groups[2].Words = nil
			case "Path":
				groups[2].Path = nil
			case "swap members":
				groups[4].Members[0], groups[4].Members[1] = groups[4].Members[1], groups[4].Members[0]
			case "actor order":
				orders[0].Raw[0x21] = 0
			case "actor Patrol":
				orders[0].Patrol = nil
			}
			if err := w.ImportSavedGroups(groups, orders); err != nil {
				t.Fatal(err)
			}
			bad, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			fresh := &sim.World{}
			if err := fresh.UnmarshalBinary(bad); err != nil {
				t.Fatal(err)
			}
			if w.Hash() != fresh.Hash() {
				t.Fatal("control lacks equal wrong native hashes")
			}
			if d := groups1155DocumentDifferences(want, join, ms.savedDocument.Document); len(d) != 0 {
				t.Fatal("control changed the correct retained Document", d)
			}
			if d, _ := groups1155LiveDifferences(want, join, ms.savedDocument, fresh); len(d) == 0 {
				t.Fatal("source comparator accepted omitted live state despite correct Document/equal native hashes")
			}
		})
	}
	projected, err := snapshotSavedDocument(ms)
	if err != nil {
		t.Fatal(err)
	}
	if d := groups1155CurrentDifferences(want, projected, ms.World); len(d) != 0 {
		t.Fatal("normalized current projection", d)
	}
	if len(groups1155DocumentDifferences(want, join, projected.Document)) == 0 {
		t.Fatal("raw/current distinction lost")
	}
	projected.GroupBindings.Unavailable = "test: missing current Group producer"
	if d := groups1155CurrentDifferences(want, projected, ms.World); len(d) != 1 || !strings.Contains(d[0], "unavailable") {
		t.Fatal("stale unavailable Document accepted as current", d)
	}
}

func TestGroups1155IdentityControls(t *testing.T) {
	f, raw := groups1155Fixture(t, true)
	want, join, doc := groups1155Inputs(t, raw)
	left, right := doc.Players[0], doc.Players[3]
	if !reflect.DeepEqual(doc.Objects[left-1].Groups[0], doc.Objects[left-1].Groups[1]) {
		t.Fatal("equal-valued distinct inline Groups absent")
	}
	ms, _, err := loadOriginalMission(f, raw)
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := groups1155LiveDifferences(want, join, ms.savedDocument, ms.World); len(d) != 0 {
		t.Fatal(d)
	}
	t.Run("equal Group identities cannot collapse", func(t *testing.T) {
		state, err := cloneSavedDocument(ms.savedDocument)
		if err != nil {
			t.Fatal(err)
		}
		state.GroupBindings.Groups[1] = state.GroupBindings.Groups[0]
		if d, _ := groups1155LiveDifferences(want, join, state, ms.World); len(d) == 0 {
			t.Fatal("two raw Groups accepted as one native identity")
		}
	})
	t.Run("different same-Slot container with consistent DTO", func(t *testing.T) {
		state, err := cloneSavedDocument(ms.savedDocument)
		if err != nil {
			t.Fatal(err)
		}
		l, r := &state.Document.Objects[left-1], &state.Document.Objects[right-1]
		moved := l.Groups[0]
		l.Groups = slices.Delete(l.Groups, 0, 1)
		r.Groups = append(r.Groups, moved)
		group1155SetCount(t, l, "Groups", len(l.Groups))
		group1155SetCount(t, r, "Groups", len(r.Groups))
		for i := range state.GroupBindings.Groups {
			b := &state.GroupBindings.Groups[i]
			if b.ID == 1 {
				b.PlayerObject, b.InlineIndex, b.ContainerID = right, uint32(len(r.Groups)-1), 2
			} else if b.PlayerObject == left {
				b.InlineIndex--
			}
		}
		if _, err := cloneSavedGroupBindings(state.GroupBindings, state.Document, state.Actors); err != nil {
			t.Fatal("control DTO is not internally consistent", err)
		}
		encoded, err := ms.World.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		w := &sim.World{}
		if err := w.UnmarshalBinary(encoded); err != nil {
			t.Fatal(err)
		}
		groups, orders, _ := w.SavedGroups()
		groups[0].ContainerID = 2
		groups = append(groups[1:], groups[0])
		if err := w.ImportSavedGroups(groups, orders); err != nil {
			t.Fatal("same-Slot control World is invalid", err)
		}
		if d, _ := groups1155LiveDifferences(want, join, state, w); len(d) == 0 {
			t.Fatal("same Slot substituted for exact raw container identity")
		}
	})
	t.Run("equal-valued distinct actors cannot share a DTO identity", func(t *testing.T) {
		var human []uint16
		for i, r := range doc.Objects {
			if r.Class == "Human" {
				human = append(human, uint16(i+1))
			}
		}
		if len(human) != 2 {
			t.Fatal("two Human subjects absent")
		}
		copy, err := sav.CloneDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		newGroupSetValue1115(t, &copy.Objects[human[0]-1], "Identity", 0)
		copy.Objects[human[1]-1] = copy.Objects[human[0]-1]
		equal, err := sav.EncodeDocumentData(copy)
		if err != nil {
			t.Fatal(err)
		}
		s, mapping, d := groups1155Inputs(t, equal)
		var archive []uint16
		for index, a := range s.actors {
			if a.class == "Human" {
				archive = append(archive, index)
			}
		}
		if len(archive) != 2 || !reflect.DeepEqual(d.Objects[mapping[archive[0]]-1], d.Objects[mapping[archive[1]]-1]) {
			t.Fatal("distinct equal-valued raw actors absent")
		}
		bad := maps.Clone(mapping)
		old := bad[archive[1]]
		bad[archive[1]] = bad[archive[0]]
		for i := range d.Objects {
			for j := range d.Objects[i].Groups {
				for k, v := range d.Objects[i].Groups[j].RefSlots[0].Objects {
					if v == old {
						d.Objects[i].Groups[j].RefSlots[0].Objects[k] = bad[archive[0]]
					}
				}
			}
		}
		if diff := groups1155DocumentDifferences(s, bad, &d); len(diff) == 0 {
			t.Fatal("equal-valued actor collapse accepted")
		}
	})
	// The baseline has repeated raw Player roots and A twice in one Group.
	// A remains one source/native identity; its last raw slot survives LOAD.
	if want.players.roots[0] != want.players.roots[2] || want.groups[2].members[0] != want.groups[2].members[3] || want.groups[2].current[1] != want.groups[2].members[3] {
		t.Fatal("true alias semantics changed")
	}
}

func TestGroups1155RawCountBounds(t *testing.T) {
	for _, n := range []uint32{1, 65535} {
		b := binary.LittleEndian.AppendUint16(nil, 0xffff)
		b = binary.LittleEndian.AppendUint32(b, n)
		b = append(b, bytes.Repeat([]byte{0x34, 0x12}, int(n))...)
		r := player1154Reader{body: b}
		p := 0
		words := group1155Words(r.array(&p, 2))
		if r.err != nil || len(words) != int(n) || p != len(b) || words[len(words)-1] != 0x1234 {
			t.Fatal("extended Group word count", n, r.err)
		}
		for _, cut := range []int{0, 1, 2, 5, len(b) - 1} {
			r := player1154Reader{body: b[:cut]}
			p := 0
			r.array(&p, 2)
			if r.err == nil {
				t.Fatal("truncated Group list accepted", n, cut)
			}
		}
	}
}
