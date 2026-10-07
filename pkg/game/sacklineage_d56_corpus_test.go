//go:build sessioncorpusaudit

package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestSAVD56OriginalGroundSackLineage(t *testing.T) {
	assets, corpus := os.Getenv("AGAINROM_ASSETS"), os.Getenv("AGAINROM_SAVE_CORPUS")
	if assets == "" || corpus == "" {
		t.Fatal("AGAINROM_ASSETS and AGAINROM_SAVE_CORPUS must name the lawful install and owner save corpus")
	}
	raw, err := os.ReadFile(filepath.Join(corpus, "2026-09-27", "oldsaves7", "game0006.sav"))
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "b652cb4c5745b6dfa1a4cec12ea3be1dbb5991f44bbde97716755d6a5b8197ba" {
		t.Fatalf("D56 source SHA %s", got)
	}
	sourceFile, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	source, err := sackByteWalk(sourceFile)
	if err != nil {
		t.Fatal(err)
	}
	sourceSack := d56Sack(source, 168)
	if sourceSack == nil || sourceSack.values["Identity"] != 49946112 || len(sourceSack.refs["Contents"]) != 1 {
		t.Fatal("D56 source Sack or its one Item is absent")
	}
	sourceItem := source.rows[sourceSack.refs["Contents"][0]]
	if sourceItem == nil || sourceItem.class != "Weapon" || sourceItem.values["F40"] != 33076 || sourceItem.values["Identity"] != 49946256 {
		t.Fatal("D56 source Weapon is absent")
	}
	f, err := NewFrontEnd(assets)
	if err != nil {
		t.Fatal(err)
	}
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("restore D56 source: town=%v err=%v", town, err)
	}
	a := f.App("D56 Sack lineage")
	a.Layout(1024, 768)
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	f.LiveAdvance(1)
	written, stage, err := writerCensusSave(f, true, t.TempDir())
	if err != nil {
		t.Fatalf("%s: %v", stage, err)
	}
	writtenFile, err := sav.Open(written)
	if err != nil {
		t.Fatal(err)
	}
	after, err := sackByteWalk(writtenFile)
	if err != nil {
		t.Fatal(err)
	}
	writtenSack := d56Sack(after, 168)
	if writtenSack == nil || writtenSack.values["Identity"] != sourceSack.values["Identity"] {
		t.Fatal("unchanged D56 Sack runtime 168 at (71,116) is absent after SAVE")
	}
	if len(writtenSack.refs["Contents"]) != 1 {
		t.Fatalf("D56 Sack Contents count %d, want one", len(writtenSack.refs["Contents"]))
	}
	writtenItem := after.rows[writtenSack.refs["Contents"][0]]
	if writtenItem == nil || writtenItem.class != sourceItem.class || writtenItem.values["Identity"] != sourceItem.values["Identity"] {
		t.Fatalf("D56 Sack Item identity changed: source %s %d, written %v", sourceItem.class, sourceItem.values["Identity"], writtenItem)
	}
	cold, err := NewFrontEnd(assets)
	if err != nil {
		t.Fatal(err)
	}
	cold.SetDeterministicFrames(true)
	reopen, town, err := cold.RestoreOriginal(written)
	if err != nil || town {
		t.Fatalf("cold LOAD D56 SAV: town=%v err=%v", town, err)
	}
	coldApp := cold.App("D56 Sack cold LOAD")
	coldApp.Layout(1024, 768)
	if err := coldApp.OpenMission(reopen); err != nil {
		t.Fatal(err)
	}
	if cold.live.world.Hash() != f.live.world.Hash() {
		t.Fatalf("D56 World hash after cold LOAD: %x, want %x", cold.live.world.Hash(), f.live.world.Hash())
	}
	f.LiveAdvance(1)
	cold.LiveAdvance(1)
	if cold.live.world.Hash() != f.live.world.Hash() {
		t.Fatalf("D56 World hash after next tick: %x, want %x", cold.live.world.Hash(), f.live.world.Hash())
	}
}

func d56Sack(source sackByteSource, runtime uint32) *sackByteRecord {
	for _, index := range source.roots {
		row := source.rows[index]
		if row != nil && row.class == "Sack" && writerCensusSackCell(row) == [2]int32{71, 116} && row.values["RuntimeID"] == runtime {
			return row
		}
	}
	return nil
}
