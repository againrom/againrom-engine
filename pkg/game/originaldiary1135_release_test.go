package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseOriginalDiariesRestoreOnLoad1135(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	chars, playerRec, err := source.PartyWalk()
	if err != nil {
		t.Fatal(err)
	}
	wantPlayer, wantHasPlayer, err := sav.PlayerDiary(playerRec)
	if err != nil || !wantHasPlayer {
		t.Fatalf("PlayerDiary: has=%v err=%v", wantHasPlayer, err)
	}
	if wantPlayer.Length != 119 {
		t.Fatalf("fixture missing real content: player diary length=%d, want 119", wantPlayer.Length)
	}
	var wantHero sav.Character
	heroes := 0
	for _, c := range chars {
		if c.Hero {
			wantHero, heroes = c, heroes+1
		}
	}
	if heroes != 1 || !wantHero.HasDiary || wantHero.MapUnitID != 0 {
		t.Fatalf("fixture is not the expected case: heroes=%d hasDiary=%v mapUnitID=%d", heroes, wantHero.HasDiary, wantHero.MapUnitID)
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1135 diaries")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0021.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game0021.sav")

	got := f.live.world.SavedDiaries()
	var gotPlayer *sim.SavedDiary
	var gotHero *sim.SavedDiary
	for i, d := range got {
		if d.Owner.Player {
			gotPlayer = &got[i]
		} else {
			gotHero = &got[i] // The fixture carries exactly one admitted actor diary: the hero's own.
		}
	}
	if gotPlayer == nil || gotPlayer.Length != wantPlayer.Length || len(gotPlayer.Entries) != len(wantPlayer.Entries) {
		t.Fatalf("LOAD player diary = %+v, want length=%d entries=%d", gotPlayer, wantPlayer.Length, len(wantPlayer.Entries))
	}
	if gotHero == nil || gotHero.Length != wantHero.Diary.Length || len(gotHero.Entries) != len(wantHero.Diary.Entries) {
		t.Fatalf("LOAD hero diary = %+v, want length=%d entries=%d", gotHero, wantHero.Diary.Length, len(wantHero.Diary.Entries))
	}

	// Native export: write the live App state into a fresh decode of this
	// file's own bytes and require the re-decoded Diary content to reproduce
	// the source file, going through exactly the ArchiveIndex join key this
	// fixture's own MapUnitID-less hero needs.
	target, err := sav.Open(payload)
	if err != nil {
		t.Fatalf("sav.Open (export target): %v", err)
	}
	if err := exportOriginalDiaries(target, f.live.world); err != nil {
		t.Fatalf("exportOriginalDiaries: %v", err)
	}
	written, err := sav.Open(target.Marshal())
	if err != nil {
		t.Fatalf("sav.Open (re-decode written): %v", err)
	}
	writtenChars, writtenPlayerRec, err := written.PartyWalk()
	if err != nil {
		t.Fatal(err)
	}
	exportedPlayer, exportedHasPlayer, err := sav.PlayerDiary(writtenPlayerRec)
	if err != nil || !exportedHasPlayer {
		t.Fatalf("PlayerDiary (re-decode written): has=%v err=%v", exportedHasPlayer, err)
	}
	if exportedPlayer.Length != wantPlayer.Length || len(exportedPlayer.Entries) != len(wantPlayer.Entries) {
		t.Fatalf("native export player diary = %+v, want length=%d entries=%d", exportedPlayer, wantPlayer.Length, len(wantPlayer.Entries))
	}
	exportedHeroes := 0
	for _, c := range writtenChars {
		if c.Hero {
			exportedHeroes++
			if c.Diary.Length != wantHero.Diary.Length || len(c.Diary.Entries) != len(wantHero.Diary.Entries) {
				t.Fatalf("native export hero diary = %+v, want length=%d entries=%d", c.Diary, wantHero.Diary.Length, len(wantHero.Diary.Entries))
			}
		}
	}
	if exportedHeroes != 1 {
		t.Fatalf("native export hero count = %d, want 1", exportedHeroes)
	}
	t.Logf("player diary length=%d entries=%d; hero diary length=%d entries=%d (MapUnitID 0): LOAD and native export both reproduce the source file",
		wantPlayer.Length, len(wantPlayer.Entries), wantHero.Diary.Length, len(wantHero.Diary.Entries))
}

// sameSavDiaryEntries1135 compares two sav.DiaryEntry lists by value
// (review F-1/F-2: a witness must assert entry values, not lengths alone).
// diaryFromRecord and encodeDiaryArrays' own inverse both build/consume
// Entries in ascending Index order, so a position-for-position compare is
// exact, not merely a length check.
func sameSavDiaryEntries1135(got, want []sav.DiaryEntry) bool {
	if len(got) != len(want) {
		return false
	}
	for i, g := range got {
		if g != want[i] {
			return false
		}
	}
	return true
}

// sameLiveDiaryEntries1135 is sameSavDiaryEntries1135's own counterpart for
// the sim-layer mirror type, comparing a live SavedDiary's own Entries
// against the file's own decoded sav.DiaryEntry list.
func sameLiveDiaryEntries1135(got []sim.SavedDiaryEntry, want []sav.DiaryEntry) bool {
	if len(got) != len(want) {
		return false
	}
	for i, g := range got {
		w := want[i]
		if g.Index != w.Index || g.Count != w.Count || g.Remaining != w.Remaining {
			return false
		}
	}
	return true
}

func gameNineNineNineNineDiaryEntries1135() []sav.DiaryEntry {
	return []sav.DiaryEntry{
		{Index: 64, Count: 2, Remaining: 1022},
		{Index: 88, Count: 2, Remaining: 1022},
		{Index: 96, Count: 6, Remaining: 1018},
		{Index: 100, Count: 5, Remaining: 1019},
	}
}

// TestReleaseOriginalDiaryRealPlayerEntriesRestoreOnLoad1135 is F-1's own
// witness: TestReleaseOriginalDiariesRestoreOnLoad1135 drives a fixture
// whose Player and hero Diaries are both all-default, so it proves presence,
// not content, and every assertion it makes is a comparison of 0 against 0.
func TestReleaseOriginalDiaryRealPlayerEntriesRestoreOnLoad1135(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-24/game9999.sav", "d954bd394473d0311e934b5358519eb73ca45773732b8070e6948346f6475b27")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	_, playerRec, err := source.PartyWalk()
	if err != nil {
		t.Fatal(err)
	}
	wantPlayer, wantHasPlayer, err := sav.PlayerDiary(playerRec)
	if err != nil || !wantHasPlayer {
		t.Fatalf("PlayerDiary: has=%v err=%v", wantHasPlayer, err)
	}
	wantEntries := gameNineNineNineNineDiaryEntries1135()
	if len(wantPlayer.Entries) == 0 {
		t.Fatal("fixture missing real content: player diary carries no non-default entry")
	}
	if !sameSavDiaryEntries1135(wantPlayer.Entries, wantEntries) {
		t.Fatalf("fixture entries changed: %+v, want %+v", wantPlayer.Entries, wantEntries)
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1135 real diary entries")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game9999.sav")

	got := f.live.world.SavedDiaries()
	var gotPlayer *sim.SavedDiary
	for i, d := range got {
		if d.Owner.Player {
			gotPlayer = &got[i]
		}
	}
	if gotPlayer == nil || gotPlayer.Length != wantPlayer.Length || !sameLiveDiaryEntries1135(gotPlayer.Entries, wantEntries) {
		t.Fatalf("LOAD player diary = %+v, want length=%d entries=%+v", gotPlayer, wantPlayer.Length, wantEntries)
	}
	t.Logf("player diary length=%d entries=%+v: LOAD restored the fixture's own four real entries, not merely their count",
		gotPlayer.Length, gotPlayer.Entries)
}

// TestReleaseOriginalDiaryRealPlayerEntriesSurviveNativeExport1135 is F-1's
// export-side counterpart: every release-gated export check so far only ever
// patched an all-default Diary
// (TestReleaseOriginalDiariesRestoreOnLoad1135), which reproduces trivially
// since every write is the construction default.
func TestReleaseOriginalDiaryRealPlayerEntriesSurviveNativeExport1135(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-24/game9999.sav", "d954bd394473d0311e934b5358519eb73ca45773732b8070e6948346f6475b27")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	_, playerRec, err := source.PartyWalk()
	if err != nil {
		t.Fatal(err)
	}
	wantPlayer, wantHasPlayer, err := sav.PlayerDiary(playerRec)
	if err != nil || !wantHasPlayer || len(wantPlayer.Entries) == 0 {
		t.Fatalf("fixture missing real content: has=%v err=%v entries=%d", wantHasPlayer, err, len(wantPlayer.Entries))
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1135 real diary export")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game9999.sav")

	target, err := sav.Open(payload)
	if err != nil {
		t.Fatalf("sav.Open (export target): %v", err)
	}
	if err := exportOriginalDiaries(target, f.live.world); err != nil {
		t.Fatalf("exportOriginalDiaries: %v", err)
	}
	written, err := sav.Open(target.Marshal())
	if err != nil {
		t.Fatalf("sav.Open (re-decode written): %v", err)
	}
	if len(written.Body) != len(source.Body) {
		t.Fatalf("native export changed the body length: %d -> %d", len(source.Body), len(written.Body))
	}
	diff := 0
	for i := range source.Body {
		if source.Body[i] != written.Body[i] {
			diff++
		}
	}
	if diff > 0 {
		t.Fatalf("native export of the diary state does not reproduce the source file: %d byte(s) differ", diff)
	}
	_, writtenPlayerRec, err := written.PartyWalk()
	if err != nil {
		t.Fatal(err)
	}
	exportedPlayer, exportedHasPlayer, err := sav.PlayerDiary(writtenPlayerRec)
	if err != nil || !exportedHasPlayer {
		t.Fatalf("PlayerDiary (re-decode written): has=%v err=%v", exportedHasPlayer, err)
	}
	if exportedPlayer.Length != wantPlayer.Length || !sameSavDiaryEntries1135(exportedPlayer.Entries, wantPlayer.Entries) {
		t.Fatalf("native export player diary = %+v, want length=%d entries=%+v", exportedPlayer, wantPlayer.Length, wantPlayer.Entries)
	}
	t.Logf("player diary length=%d entries=%+v: native export reproduces the source file's own four real entries and all %d body bytes exactly",
		exportedPlayer.Length, exportedPlayer.Entries, len(source.Body))
}
