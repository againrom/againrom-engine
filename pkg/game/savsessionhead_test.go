package game

import (
	"os"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func constructedCurrentSessionHead1115() [48]byte {
	return [48]byte{16: 0x80, 17: 0x96, 18: 0x98, 32: 0x10, 33: 0x27}
}

func worldHashWithConstructedCurrentSessionHead1115(t *testing.T, source *sim.World) uint64 {
	t.Helper()
	raw, err := source.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var world sim.World
	if err := world.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	world.SetRawSessionHead(constructedCurrentSessionHead1115())
	return world.Hash()
}

func worldBytesWithConstructedCurrentSessionHead1115(t *testing.T, raw []byte) []byte {
	t.Helper()
	var world sim.World
	if err := world.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	world.SetRawSessionHead(constructedCurrentSessionHead1115())
	encoded, err := world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestCurrentSAVSessionHeadPrecedenceAndAbsence(t *testing.T) {
	var current, retained, partial [48]byte
	for i := range current {
		current[i], retained[i] = byte(i+1), byte(255-i)
	}
	partial[0] = 0x37
	constructed := constructedCurrentSessionHead1115()
	for _, tc := range []struct {
		name               string
		live, source, want [48]byte
	}{
		{"current-before-source", current, retained, current},
		{"current-without-source", current, [48]byte{}, current},
		{"retained-without-current", [48]byte{}, retained, retained},
		{"both-absent", [48]byte{}, [48]byte{}, constructed},
		{"partial-current-is-not-absent", partial, retained, partial},
		{"partial-source-is-not-absent", [48]byte{}, partial, partial},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, _ := documentSnapshotFixture(t)
			ms := f.live.mission.state
			ms.World.SetRawSessionHead(tc.live)
			ms.savedDocument.Document.World.Session.Raw08 = tc.source
			before := ms.World.Hash()
			clock, _ := ms.World.SessionClock()
			for range 2 {
				state, err := snapshotSavedDocument(ms)
				if err != nil {
					t.Fatal(err)
				}
				if state.Document.World.Session.Raw08 != tc.want {
					t.Fatalf("projected session head=%x, want %x", state.Document.World.Session.Raw08, tc.want)
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
				if err != nil || session.RawHead != tc.want {
					t.Fatal("wire head differs from independent expected bytes", err)
				}
				if state.Document.Head.CounterA != clock.SubTick || state.Document.Head.CounterB != clock.FullTick {
					t.Fatal("timer initialization changed simulation counters")
				}
			}
			if ms.World.Hash() != before || ms.World.RawSessionHead() != tc.live || ms.savedDocument.Document.World.Session.Raw08 != tc.source {
				t.Fatal("projection mutated live or retained source state")
			}
		})
	}
}

func TestReleaseMissionSAVConstructsAbsentSessionTimer(t *testing.T) {
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "Timer witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("timer witness")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	want := [48]byte{16: 0x80, 17: 0x96, 18: 0x98, 32: 0x10, 33: 0x27}
	check := func(path string) []byte {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := sav.Open(raw)
		if err != nil || file.World == nil || file.Head.Mission != 20 {
			t.Fatal("ordinary SAVE is not mission20 SAV", err)
		}
		session, err := file.SessionState()
		if err != nil || session.RawHead != want {
			t.Fatalf("ordinary SAVE timer=%x want=%x error=%v", session.RawHead, want, err)
		}
		return raw
	}
	raw := check(generatedMissionSave(t, f, app))
	checkpoint, _, err := f.Snapshot(true)
	if err != nil || f.live.world.RawSessionHead() != ([48]byte{}) || checkpoint.SavedDocument != nil {
		t.Fatal("ordinary SAV projection mutated the live checkpoint or attached retained source state", err)
	}
	native, err := EncodeSave(checkpoint, "retained timer")
	if err != nil {
		t.Fatal(err)
	}
	restored, _, err := DecodeSave(native)
	if err != nil {
		t.Fatal(err)
	}
	g := releaseFront(t)
	opener, town, err := g.Restore(restored)
	if err != nil || town || opener == nil {
		t.Fatal("native checkpoint LOAD", err)
	}
	nativeApp := g.App("native timer")
	if err = nativeApp.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	if g.live.world.RawSessionHead() != ([48]byte{}) {
		t.Fatal("native LOAD must retain the absent World head")
	}
	check(generatedMissionSave(t, g, nativeApp))
	for _, clearHead := range []bool{false, true} {
		input := raw
		if clearHead {
			file, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			if err := file.SetRawHead([48]byte{}); err != nil {
				t.Fatal(err)
			}
			input = file.Marshal()
		}
		g := releaseFront(t)
		opener, town, err := g.RestoreOriginal(input)
		if err != nil || town || opener == nil {
			t.Fatal("cold mission LOAD", err)
		}
		coldApp := g.App("cold timer")
		if err := coldApp.OpenMission(opener); err != nil {
			t.Fatal(err)
		}
		if g.live.world.RawSessionHead() != ([48]byte{}) {
			t.Fatal("LOAD materialized retained timer bytes into current World state")
		}
		for range 16 {
			sim.Step(g.live.world, nil)
		}
		check(generatedMissionSave(t, g, coldApp))
	}
}
