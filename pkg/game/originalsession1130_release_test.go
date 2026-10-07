package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
)

// TestReleaseOriginalSessionRegistersRestoreOnLoad1130 is the registers
// witness. It reads the file's own registers independently of any resume
// path, drives an ordinary App LOAD, and requires the live world's hundred
// registers to equal the file exactly; it then exercises the native-writer
// counterpart (exportOriginalMissionSession) and requires the written span to
// reproduce the source file byte for byte.
func TestReleaseOriginalSessionRegistersRestoreOnLoad1130(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	fileState, err := source.SessionState()
	if err != nil {
		t.Fatalf("SessionState: %v", err)
	}
	nonzero := 0
	for _, v := range fileState.Registers {
		if v != 0 {
			nonzero++
		}
	}
	if nonzero == 0 {
		t.Fatal("fixture carries no non-zero register: not a meaningful witness")
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1130 registers")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game9999.sav")
	if got := f.live.world.ScriptRegisters(); got != fileState.Registers {
		t.Fatalf("LOAD registers = %v, want %v", got, fileState.Registers)
	}

	// Native export: write the live App state into a fresh decode of this
	// file's own bytes, Marshal it, and re-open the marshaled bytes so the
	// comparison reads two decompressed bodies at the same SessionOff.
	target, err := sav.Open(payload)
	if err != nil {
		t.Fatalf("sav.Open (export target): %v", err)
	}
	if err := exportOriginalMissionSession(target, f.live.world); err != nil {
		t.Fatalf("exportOriginalMissionSession: %v", err)
	}
	written, err := sav.Open(target.Marshal())
	if err != nil {
		t.Fatalf("sav.Open (re-decode written): %v", err)
	}
	base := target.World.SessionOff
	if !bytes.Equal(written.Body[base:base+400], source.Body[base:base+400]) {
		t.Fatal("native export of the register span (0..400) does not reproduce the source file")
	}
	t.Logf("registers non-zero=%d/100; LOAD and native export both reproduce the source file exactly", nonzero)
}

// TestReleaseOriginalSessionRawSpansSurviveActorStockStaging1130 is the raw-
// region witness. RawHead carries real ROM1 content in this fixture; RawMid
// is all-zero in every preserved owner save (docs/1130/story.md, Open debt),
// so this test writes one explicitly named diagnostic byte pattern into
// RawMid before LOAD, the same limited role the 1106 witness's own
// "changed-source" variant plays for cell triggers — not a claim about
// original runtime content, only a probe of the carry-through mechanism.
//
// The LOAD path this drives crosses restoreOriginalActorStock's detached
// MarshalBinary/UnmarshalBinary staging round trip, the exact route story
// 1130 found silently dropping both spans one call after
// ImportOriginalSession had restored them correctly.
func TestReleaseOriginalSessionRawSpansSurviveActorStockStaging1130(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	var mid [400]byte
	mid[0], mid[199], mid[399] = 0x11, 0x22, 0x33
	if err := source.SetRawMid(mid); err != nil {
		t.Fatalf("SetRawMid: %v", err)
	}
	changed := source.Marshal()
	reopened, err := sav.Open(changed)
	if err != nil {
		t.Fatal(err)
	}
	fileState, err := reopened.SessionState()
	if err != nil {
		t.Fatalf("SessionState: %v", err)
	}
	if fileState.RawMid != mid {
		t.Fatal("SetRawMid fixture setup did not take")
	}
	headNonzero := 0
	for _, b := range fileState.RawHead {
		if b != 0 {
			headNonzero++
		}
	}
	if headNonzero == 0 {
		t.Fatal("fixture carries no non-zero RawHead: not a meaningful witness for the real span")
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1130 raw spans")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), changed, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game9999.sav")
	if got := f.live.world.RawSessionHead(); got != fileState.RawHead {
		t.Fatalf("LOAD RawSessionHead = %x, want %x", got, fileState.RawHead)
	}
	if got := f.live.world.RawSessionMid(); got != mid {
		t.Fatalf("LOAD RawSessionMid = %x, want %x (restoreOriginalActorStock's staging must carry it across, docs/1130/story.md)", got, mid)
	}

	target, err := sav.Open(changed)
	if err != nil {
		t.Fatalf("sav.Open (export target): %v", err)
	}
	if err := exportOriginalMissionSession(target, f.live.world); err != nil {
		t.Fatalf("exportOriginalMissionSession: %v", err)
	}
	written, err := sav.Open(target.Marshal())
	if err != nil {
		t.Fatalf("sav.Open (re-decode written): %v", err)
	}
	base := target.World.SessionOff
	if !bytes.Equal(written.Body[base+1400:base+1848], reopened.Body[base+1400:base+1848]) {
		t.Fatal("native export of the two raw spans (1400..1848) does not reproduce the fixture")
	}
	t.Logf("RawHead non-zero=%d/48 (real ROM1 content); RawMid diagnostic mutation carried through LOAD, actor-stock staging, and native export", headNonzero)
}
