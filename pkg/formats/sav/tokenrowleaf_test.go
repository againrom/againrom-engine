package sav

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func setDocumentValue(t *testing.T, r *DocumentRecordData, name string, v uint32) {
	t.Helper()
	for i := range r.Values {
		if r.Values[i].Name == name {
			r.Values[i].Value = v
			return
		}
	}
	t.Fatalf("%s record holds no %s", r.Class, name)
}

func documentValue(r DocumentRecordData, name string) uint32 {
	for _, v := range r.Values {
		if v.Name == name {
			return v.Value
		}
	}
	return 0
}

// checkPartyRow: the File reader gives the document's one party Human row
// row, through the party and through the actor graph.
func checkPartyRow(t *testing.T, name string, raw []byte, row int) {
	t.Helper()
	f, err := Open(raw)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	party, err := f.Party()
	if err != nil || len(party) != 1 || int(party[0].DefRow) != row {
		t.Fatalf("%s: party %+v: %v, want one Human of row %d", name, party, err, row)
	}
	graph, err := f.ActorGraph()
	if err != nil || len(graph.Actors) != 1 || int(graph.Actors[0].DefRow) != row {
		t.Fatalf("%s: actor graph %d actors: %v, want one of row %d", name, len(graph.Actors), err, row)
	}
}

// A Human whose Token row is above a byte: the engine's Asg& document keeps
// the low byte in the record and the row in its AgainromRows leaf, and both
// readers give the row back; the second game's Bsg& records hold it whole.
// A document whose rows fit a byte writes no leaf.
func TestTokenRowAboveAByteTravelsInTheEngineDocument(t *testing.T) {
	plain := engineTownDocument(t)
	setDocumentValue(t, &plain.Objects[1], "T0C", 255)
	narrow, err := EncodeDocumentData(plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(narrow, []byte("AgainromRows")) {
		t.Fatal("a document whose rows fit a byte writes the rows leaf")
	}
	checkPartyRow(t, "row 255", narrow, 255)
	stale := plain
	stale.State.ValueRecords = append(slices.Clone(plain.State.ValueRecords), CityStateRecordData{Path: "/CurrentState/AgainromRows",
		Value: CityStateValueData{Kind: 6, Bytes: []byte{1, 0, 0, 0, 2, 0, 0, 0, 8, 1, 0, 0}}})
	slices.SortFunc(stale.State.ValueRecords, func(a, b CityStateRecordData) int { return strings.Compare(a.Path, b.Path) })
	if again, err := EncodeDocumentData(stale); err != nil || !bytes.Equal(again, narrow) {
		t.Fatalf("a stale rows leaf in the state reaches the file: %v", err)
	}

	doc := engineTownDocument(t)
	setDocumentValue(t, &doc.Objects[1], "T0C", 264)
	engine, err := EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(engine, []byte("AgainromRows")) {
		t.Fatal("the engine document carries no rows leaf")
	}
	got, err := DecodeDocumentData(engine)
	if err != nil {
		t.Fatal(err)
	}
	if v := documentValue(got.Objects[1], "T0C"); v != 264 {
		t.Fatalf("the decoded Human row is %d, want 264", v)
	}
	if _, found, _ := got.State.ReadLeaf("/CurrentState/AgainromRows"); found {
		t.Fatal("the decoded document keeps the rows leaf in its state")
	}
	if again, err := EncodeDocumentData(got); err != nil || !bytes.Equal(again, engine) {
		t.Fatalf("the engine document does not re-encode byte for byte: %v", err)
	}
	checkPartyRow(t, "engine document", engine, 264)

	second, _, err := DocumentInProfile(doc, SecondGame, ProfileKeeps{})
	if err != nil {
		t.Fatal(err)
	}
	second.SecondTail = &SecondTail{Pairs: [][2]uint32{{1, 1}}, Locations: []SecondLocation{{ID: 1, Kind: 2}}}
	file := profileFile(t, second, engine)
	records, err := DecodeDocumentData(file)
	if err != nil {
		t.Fatal(err)
	}
	if v := documentValue(records.Objects[1], "T0C"); v != 264 {
		t.Fatalf("the Bsg& Human row is %d, want 264", v)
	}
	if back, err := EngineDocument(file); err != nil || !bytes.Equal(back, engine) {
		t.Fatalf("the Bsg& file does not carry the engine document: %v", err)
	}
	checkPartyRow(t, "Bsg& records", file, 264)
}
