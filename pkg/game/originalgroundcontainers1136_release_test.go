package game

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"againrom/pkg/formats/sav"
)

// SAV-671
func TestReleaseOriginalGroundContainersRestoreOnLoad1136(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	fileSacks, present, err := source.GroundSacks()
	if err != nil || !present || len(fileSacks) != 4 {
		t.Fatalf("GroundSacks: present=%v err=%v n=%d, want 4", present, err, len(fileSacks))
	}
	want := make([]sav.GroundContainerTail, len(fileSacks))
	for i, sack := range fileSacks {
		want[i] = sav.GroundContainerTail{Identity: sack.Identity, InsertIndex: sack.InsertIndex, Accumulator: sack.Accumulator}
	}
	sort.Slice(want, func(i, j int) bool { return want[i].Identity < want[j].Identity })
	if want[0].Accumulator == 0 && want[1].Accumulator == 0 {
		t.Fatal("fixture missing real content: every Accumulator is zero")
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1136 ground containers")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0021.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game0021.sav")

	registry := f.live.world.SavedObjects()
	for _, w := range want {
		tail, ok := liveGroundContainerTail(registry, w.Identity)
		if !ok || tail != w {
			t.Errorf("LOAD live container tail for Sack %#x = %+v present=%v, want %+v", w.Identity, tail, ok, w)
		}
	}

	// Native export: write the live App state into a fresh decode of this
	// file's own bytes and require the decoded container tails to reproduce
	// the source file (the whole-file byte comparison is
	// originalgroundcontainers1136_corpus_test.go's own job, run across the
	// full preserved corpus rather than this one fixture).
	target, err := sav.Open(payload)
	if err != nil {
		t.Fatalf("sav.Open (export target): %v", err)
	}
	if err := exportOriginalGroundContainers(target, f.live.world); err != nil {
		t.Fatalf("exportOriginalGroundContainers: %v", err)
	}
	written, err := sav.Open(target.Marshal())
	if err != nil {
		t.Fatalf("sav.Open (re-decode written): %v", err)
	}
	exportedSacks, exportedPresent, err := written.GroundSacks()
	if err != nil || !exportedPresent || len(exportedSacks) != len(fileSacks) {
		t.Fatalf("GroundSacks (re-decode written): present=%v err=%v n=%d", exportedPresent, err, len(exportedSacks))
	}
	exported := make([]sav.GroundContainerTail, len(exportedSacks))
	for i, sack := range exportedSacks {
		exported[i] = sav.GroundContainerTail{Identity: sack.Identity, InsertIndex: sack.InsertIndex, Accumulator: sack.Accumulator}
	}
	sort.Slice(exported, func(i, j int) bool { return exported[i].Identity < exported[j].Identity })
	for i := range want {
		if exported[i] != want[i] {
			t.Errorf("native export tail %d = %+v, want %+v", i, exported[i], want[i])
		}
	}
	t.Logf("sacks=%d accumulators=%v: LOAD and native export both reproduce the source file's decoded container tails", len(want), func() []int32 {
		v := make([]int32, len(want))
		for i, w := range want {
			v[i] = w.Accumulator
		}
		return v
	}())
}
