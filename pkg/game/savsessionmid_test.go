package game

import (
	"againrom/pkg/formats/sav"
	"testing"
)

func TestCurrentSessionMidReplacesRetainedBytes(t *testing.T) {
	f, _ := documentSnapshotFixture(t)
	ms := f.live.mission.state
	var current, retained [400]byte
	for i := range current {
		current[i], retained[i] = byte(i*17+3), byte(i*29+5)
	}
	ms.World.SetRawSessionMid(current)
	ms.savedDocument.Document.World.Session.RawA828 = retained
	before := ms.World.Hash()
	for range 2 {
		state, err := snapshotSavedDocument(ms)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := sav.EncodeDocumentData(*state.Document)
		if err != nil {
			t.Fatal(err)
		}
		file, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		session, err := file.SessionState()
		if err != nil || session.RawMid != current {
			t.Fatal("current400-byte session carrier was not written", err)
		}
	}
	if ms.World.Hash() != before || ms.savedDocument.Document.World.Session.RawA828 != retained {
		t.Fatal("projection mutated a source")
	}
}
