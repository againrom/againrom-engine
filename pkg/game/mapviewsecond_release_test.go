//go:build release

package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/internal/witness"
	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

func TestReleaseSecondMapEditorViewsEveryInstalledMap(t *testing.T) {
	x := installedInspector(t)
	if x.installGame() != base.GameROM2 {
		t.Skip("requires a second-game install")
	}
	archive, loose := ArchiveMaps(x.archives.Containers), x.archives.Loose
	scenarios, roots := 0, 0
	var frameSource string
	for _, entry := range x.Maps {
		t.Run(entry.Source, func(t *testing.T) {
			if entry.Err != nil {
				t.Fatal("catalogue", entry.Err)
			}
			source := ui.MapSourceKey(ui.SourceLoose, entry.Source)
			read := loose.ReadFile
			if entry.FromArchive {
				scenarios++
				source, read = ui.MapSourceKey(ui.SourceScenario, entry.Source), archive.Read
			} else {
				roots++
			}
			raw, err := read(entry.Source)
			if err != nil {
				t.Fatal(err)
			}
			want, err := alm.OpenROM2(raw)
			if err != nil {
				t.Fatal(err)
			}
			d, err := x.Open(source)
			if err != nil {
				t.Fatal("editor open", err)
			}
			counts := map[ui.RecordKind]int{}
			for _, r := range d.Records {
				counts[r.Kind]++
			}
			objects := 0
			for _, value := range want.Overlay {
				if value != 0 {
					objects++
				}
			}
			if d.Width != want.Width || d.Height != want.Height || counts[ui.KindScenery] != objects || counts[ui.KindUnit] != len(want.Units) || counts[ui.KindStructure] != len(want.Objects) || counts[ui.KindSection] != len(want.Records) {
				t.Fatalf("size or counts differ: %dx%d, %v", d.Width, d.Height, counts)
			}
			if _, ok := d.Edits.(*mapInspectionEdits); !ok || d.ReadOnly != "" || !bytes.Equal(raw, d.SourceBytes) {
				t.Fatal("a second-game map lacks its edit host or changes bytes")
			}
			after, err := read(entry.Source)
			if err != nil || !bytes.Equal(raw, after) {
				t.Fatal("installed source changed", err)
			}
			if frameSource == "" && entry.FromArchive {
				frameSource = source
			}
		})
	}
	if scenarios != len(archive.Names()) || scenarios == 0 || roots == 0 || frameSource == "" {
		t.Fatalf("incomplete catalogue: scenarios %d/%d, roots %d", scenarios, len(archive.Names()), roots)
	}
	e := x.Editor()
	e.Layout(1920, 1080)
	if err := e.Open(frameSource); err != nil {
		t.Fatal(err)
	}
	before := bytes.Clone(e.Document().SourceBytes)
	if err := e.Key("move"); err != nil {
		t.Fatal(err)
	}
	if e.Dirty() || !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("a mode key changed the document")
	}
	if pic := e.ChromeImage(); pic.Bounds() != image.Rect(0, 0, 1920, 1080) {
		t.Fatal("wrong chrome size")
	}
	frame, _, err := e.HeadlessFrame()
	if err != nil || frame.Bounds() != image.Rect(0, 0, 1920, 1080) {
		t.Fatal("map frame", err)
	}
	if out := os.Getenv("AGAINROM_EDITOR_FRAMES"); out != "" {
		witness.PNG(t, filepath.Join(out, "second-map-"+installLanguage(x)+".png"), frame)
	}
	t.Logf("PASS: %d scenario maps, %d root maps; counts, edit host, 1920x1080 frame", scenarios, roots)
}

func TestReleaseFirstMapEditorStillOpensAndEdits(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	d, err := x.Open("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	if d.Edits == nil {
		t.Fatal("first-game edits unavailable")
	}
	var unit *ui.InspectionRecord
	for i := range d.Records {
		if d.Records[i].Kind == ui.KindUnit {
			unit = &d.Records[i]
			break
		}
	}
	if unit == nil {
		t.Fatal("no unit in control map")
	}
	m, err := alm.Open(d.SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	health := int64(1)
	if m.Units[unit.Index].CurrentHP == 1 {
		health = 2
	}
	next, err := d.Edits.SetField(ui.KindUnit, unit.Index, "hp", health)
	if err != nil || next == nil || !next.Edits.CanUndo() {
		t.Fatal("first-game edit", err)
	}
	back, err := next.Edits.Undo()
	if err != nil || back == nil || !bytes.Equal(back.SourceBytes, d.SourceBytes) {
		t.Fatal("first-game undo", err)
	}
}
