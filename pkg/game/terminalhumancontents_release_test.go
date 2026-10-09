package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestReleaseTerminalHumanOriginalResaveContents(t *testing.T) {
	path := os.Getenv("AGAINROM_TERMINAL_HUMAN_SAV")
	if path == "" {
		t.Skip("set AGAINROM_TERMINAL_HUMAN_SAV to the owner terminal-Human source")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const sourceSHA = "3ad088a444d1b505ba22e0a0decda98c62b9855b5707f360f68a716e3c35abde"
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != sourceSHA {
		t.Fatal("terminal-Human source SHA differs")
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	dead, err := file.DeadActors()
	if err != nil || len(dead) != 193 {
		t.Fatal("source dead actor population", len(dead), err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	type witness struct {
		body  sav.DeadActor
		graph terminalContentGraph
	}
	var humans []witness
	units := 0
	for _, d := range dead {
		if d.Stage != 5 || d.Worn == 0 {
			continue
		}
		if d.Class == "Unit" {
			units++
		}
		if d.Class == "Human" {
			if d.Effects != 0 || d.Carried != 0 {
				t.Fatal("owner source contents population changed", d)
			}
			humans = append(humans, witness{d, terminalHumanContentsGraph(t, doc, d.Identity)})
		}
	}
	if units != 20 || len(humans) != 4 {
		t.Fatal("source terminal equipment population", units, len(humans))
	}
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "terminal-human-source.sav")
	for _, want := range humans {
		got := requireTerminalHumanInactive(t, f.live.world, want.body.Identity)
		if got.Current.FineX != want.body.FineX || got.Current.FineY != want.body.FineY || got.Source.MapUnitID != want.body.MapUnitID {
			t.Fatal("original LOAD changed terminal position or map binding", got, want.body)
		}
	}
	var saved []byte
	for cycle := range 2 {
		_, _, saved = menuSAVE(t, f, app, OriginalStore{})
		written, err := sav.DecodeDocumentData(saved)
		if err != nil {
			t.Fatal(err)
		}
		// The World holds no contents for an original terminal actor, so its
		// root is written with no worn, held, carried or effect object
		// (DIV-2505).
		for _, want := range humans {
			got := terminalHumanContentsGraph(t, written, want.body.Identity)
			if !reflect.DeepEqual(got.Roots, want.graph.Roots) || len(got.Nodes) != 0 {
				t.Fatalf("cycle %d terminal Human %#x roots %v contents %d", cycle, want.body.Identity, got.Roots, len(got.Nodes))
			}
		}
		cold := releaseFront(t)
		coldApp, _ := openOriginalSAVApp(t, cold, saved, "terminal-human-current.sav")
		for tick := range 4 {
			assertCurrentWorldEqual(t, f.live.world, cold.live.world, "owner terminal Human cold continuation")
			if len(cold.live.world.OriginalDeadActors()) != 193 {
				t.Fatal("cold LOAD changed dead population")
			}
			for _, want := range humans {
				got := requireTerminalHumanInactive(t, cold.live.world, want.body.Identity)
				if got.Current.FineX != want.body.FineX || got.Current.FineY != want.body.FineY {
					t.Fatal("cold continuation changed fine position", cycle, tick, got)
				}
			}
			if tick != 3 {
				f.live.tick()
				cold.live.tick()
			}
		}
		f, app = cold, coldApp
	}
	if output := os.Getenv("AGAINROM_TERMINAL_HUMAN_OUT"); output != "" {
		file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, writeErr := file.Write(saved)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			t.Fatal(writeErr, closeErr)
		}
	}
	t.Logf("source=%s dead-roots=193 terminal-equipment=20 Unit/4 Human; two menu SAVE/cold LOAD cycles and 3 ticks each; output-sha=%x", sourceSHA, sha256.Sum256(saved))
}
