package game

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"testing"
)

// firstFrameBaseDigests are the first frames LOAD restored before the two
// arms went through one restore: the application state, the explored plane,
// the animation clock and the actor IDs the selection binds.
var firstFrameBaseDigests = map[string]string{
	"decoded document":        "1000a34c8fe47699f0927a3f",
	"file store speed 4":      "542e8a294db9f98c0ca2cfef",
	"file store speed 8 book": "43b8ce1c66890b63b7c9fef6",
}

func firstFrameDigest(t *testing.T, f *FrontEnd, ids map[uint32]uint32) string {
	t.Helper()
	mw := f.live
	if mw == nil || mw.applicationState == nil || mw.fog == nil {
		t.Fatal("the LOAD restored no first frame")
	}
	raw, err := json.Marshal(struct {
		State     SnapshotApplicationState
		Explored  []byte
		Visible   []byte
		Animation any
		IDs       map[uint32]uint32
	}{*mw.applicationState, mw.fog.explored, mw.fog.visible, mw.view.SaveAnimation(), ids})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:12])
}

// The source of a first frame is the decoded document of an original save,
// else the mission file's state store; the document restores its current
// view, fog plane and animation, the store none of them.
func TestTheFirstFrameSourceStatesWhatItRestores(t *testing.T) {
	file := &Mission{fileApplication: &fileApplication{}}
	for _, c := range []struct {
		name     string
		ms       *Mission
		original bool
		want     string
	}{
		{"no mission", nil, true, ""},
		{"documentless mission file", file, false, "store"},
		{"original save", file, true, "document"},
		{"native mission", &Mission{}, false, ""},
	} {
		src := firstFrameSourceOf(c.ms, c.original)
		got := ""
		switch {
		case src == nil:
		case src.current != nil && src.fog != nil && src.animation:
			got = "document"
		case src.current == nil && src.fog == nil && !src.animation:
			got = "store"
		default:
			got = "mixed"
		}
		if got != c.want {
			t.Errorf("%s: source %q, want %q", c.name, got, c.want)
		}
	}
}

// A mission file's first frame, read from its state store, and our SAV's
// first frame, read from its decoded document, restore what they restored
// before: view, selection, panels, speed, fog, animation and actor IDs.
func TestFirstFrameSourcesRestoreTheBaseFrames(t *testing.T) {
	got := map[string]string{}
	for _, c := range []struct {
		name                 string
		speed, open, pressed int32
	}{{"file store speed 4", 4, 0, -1}, {"file store speed 8 book", 8, 1, 6}} {
		f, _, _ := secondMissionLoadFile(t, secondMissionFile(secondMissionFixtureHero(), secondMissionStore(c.speed, c.open, c.pressed)))
		file := f.live.mission.state.fileApplication
		if file == nil {
			t.Fatal(c.name, "the mission file kept no state store application")
		}
		got[c.name] = firstFrameDigest(t, f, file.ids)
	}

	f, app := itemMutationOpen(t, 1, false)
	fresh, _ := itemMutationCheckpoint(t, f, app, 1)
	document := fresh.live.mission.state.savedDocument
	if document == nil || document.Document == nil {
		t.Fatal("our SAV LOADed without its document")
	}
	ids, err := applicationActorIDs(document.Document, document.Actors, fresh.live.world)
	if err != nil {
		t.Fatal(err)
	}
	got["decoded document"] = firstFrameDigest(t, fresh, ids)

	names := make([]string, 0, len(got))
	for name := range got {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if want := firstFrameBaseDigests[name]; got[name] != want {
			t.Errorf("%q: %q, want %q", name, got[name], want)
		}
	}
}
