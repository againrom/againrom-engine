package game

import (
	"encoding/binary"
	"maps"
	"testing"

	"againrom/pkg/formats/sav"
)

// bodyWordsRecord is a Unit record with the given Identity and named words.
func bodyWordsRecord(t *testing.T, identity uint32, words map[string]uint32) sav.DocumentRecordData {
	t.Helper()
	r := mustNewRecord("Unit")
	savedObjectSetValue(&r, "Identity", identity)
	for name, value := range words {
		if name == "U58" {
			mustSetRaw(&r, name, binary.LittleEndian.AppendUint32(nil, value))
			continue
		}
		savedObjectSetValue(&r, name, value)
	}
	return r
}

func bodyWordsOf(t *testing.T, r *sav.DocumentRecordData) map[string]uint32 {
	t.Helper()
	out := map[string]uint32{}
	for _, name := range loadedTerminalWords {
		if name == "U58" {
			raw, err := savedMotionRaw(r, name, 4)
			if err != nil {
				t.Fatal(err)
			}
			out[name] = binary.LittleEndian.Uint32(raw)
			continue
		}
		v, err := savedStructureValue(r, name)
		if err != nil {
			t.Fatal(err)
		}
		out[name] = v
	}
	return out
}

// A body's record takes the words its loaded record held when that record is
// a DeadActors root in no Group, and keeps the words it was built with when
// the loaded root is a Group member.
func TestLoadedBodyWordsComeFromADeadRootInNoGroup(t *testing.T) {
	words := map[string]uint32{"U58": 5, "U136": 1, "U138": 3443, "U134": 8, "UA2": 60, "U48": 1}
	for _, grouped := range []bool{false, true} {
		loaded := &sav.DocumentData{Objects: []sav.DocumentRecordData{bodyWordsRecord(t, 0x2c24500, words)}, DeadActors: []uint16{1}}
		if grouped {
			member := mustNewRecord("Unit")
			member.RefSlots = append(member.RefSlots, sav.DocumentRefsData{Name: "Members", Objects: []uint16{1}})
			loaded.Objects = append(loaded.Objects, member)
		}
		built := map[string]uint32{"U58": 0, "U136": 0, "U138": 0, "U134": 7, "UA2": 0, "U48": 0}
		r := bodyWordsRecord(t, 0x2c24500, built)
		if err := keepLoadedBodyWords(&r, &SnapshotSAVDocument{Document: loaded}, loadedTerminalWords); err != nil {
			t.Fatal(err)
		}
		want := words
		if grouped {
			want = built
		}
		if got := bodyWordsOf(t, &r); !maps.Equal(got, want) {
			t.Fatalf("grouped %v: words %v, want %v", grouped, got, want)
		}
	}
}

// A live actor's loaded reference to a body is written only while that body
// is a DeadActors root of the current document that no World entity holds.
func TestLoadedBodyReferenceNamesABodyNoEntityHolds(t *testing.T) {
	const actor, body = 0x4b8de38, 0x2d15e40
	loaded := &SnapshotSAVDocument{Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
		bodyWordsRecord(t, actor, map[string]uint32{"U5C": body, "U40": body}),
	}}}
	for _, tc := range []struct {
		name   string
		actors []SnapshotSAVActor
		roots  []uint16
		want   uint32
	}{
		{"body outside play", []SnapshotSAVActor{{EntityID: 1, ObjectIndex: 1}, {EntityID: 2, ObjectIndex: 2, Retired: true}}, []uint16{2}, body},
		{"body an entity holds", []SnapshotSAVActor{{EntityID: 1, ObjectIndex: 1}, {EntityID: 2, ObjectIndex: 2}}, []uint16{2}, 0},
		{"body gone", []SnapshotSAVActor{{EntityID: 1, ObjectIndex: 1}}, nil, 0},
	} {
		state := &SnapshotSAVDocument{Actors: tc.actors, Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
			bodyWordsRecord(t, actor, nil), bodyWordsRecord(t, body, nil),
		}, DeadActors: tc.roots}}
		for _, name := range []string{"U5C", "U40"} {
			if got := loadedBodyReference(state, loaded, &state.Document.Objects[0], name); got != tc.want {
				t.Errorf("%s: %s %#x, want %#x", tc.name, name, got, tc.want)
			}
		}
	}
}
