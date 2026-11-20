//go:build release

package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// installedSecondMaps is every installed second-game map: its source key and
// bytes, in catalogue order.
func installedSecondMaps(t *testing.T, x *MapInspector) (sources []string, raws [][]byte) {
	t.Helper()
	if x.installGame() != base.GameROM2 {
		t.Skip("requires a second-game install")
	}
	archive, loose := ArchiveMaps(x.archives.Containers), x.archives.Loose
	for _, entry := range x.Maps {
		if entry.Err != nil {
			t.Fatal("catalogue", entry.Err)
		}
		source, read := ui.MapSourceKey(ui.SourceLoose, entry.Source), loose.ReadFile
		if entry.FromArchive {
			source, read = ui.MapSourceKey(ui.SourceScenario, entry.Source), archive.Read
		}
		raw, err := read(entry.Source)
		if err != nil {
			t.Fatal(err)
		}
		sources, raws = append(sources, source), append(raws, raw)
	}
	return sources, raws
}

func TestReleaseSecondMapSavesEveryInstalledMapUnchanged(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)
	dir := t.TempDir()
	equal := 0
	for i, source := range sources {
		d, err := x.Open(source)
		if err != nil {
			t.Fatal(source, err)
		}
		host, ok := d.Edits.(*mapInspectionEdits)
		if !ok {
			t.Fatalf("%s: no second-game edit host", source)
		}
		target, err := host.SaveAs(filepath.Join(dir, fmt.Sprintf("m%03d.alm", i)))
		if err != nil {
			t.Errorf("%s: save refused: %v", source, err)
			continue
		}
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if at := firstDifference(raws[i], got); at >= 0 {
			t.Errorf("%s: first differing offset %d", source, at)
			continue
		}
		equal++
	}
	if equal != len(sources) || equal == 0 {
		t.Fatalf("%d of %d installed maps saved equal", equal, len(sources))
	}
	t.Logf("PASS: %d of %d installed second-game maps save byte for byte equal (%s)", equal, len(sources), installLanguage(x))
}

func firstDifference(a, b []byte) int {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return min(len(a), len(b))
	}
	return -1
}

func TestReleaseSecondMapTerrainEditSavesAndReloads(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)
	tried := 0
	for i, source := range sources {
		was, err := alm.OpenROM2(raws[i])
		if err != nil || was.Width < 16 || was.Height < 16 {
			continue
		}
		d, err := x.Open(source)
		if err != nil {
			t.Fatal(err)
		}
		host := d.Edits.(*mapInspectionEdits)
		cell := image.Pt(int(was.Width)/2, int(was.Height)/2)
		var edited *ui.InspectionDocument
		for _, class := range []uint8{mapedit.ClassMountain, mapedit.ClassWater, mapedit.ClassLand} {
			if edited, err = host.Paint(ui.BrushStroke{Plane: ui.BrushPlaneTerrain, Value: int(class), Cells: []image.Point{cell}}); err != nil || edited != nil {
				break
			}
		}
		if err != nil {
			t.Fatalf("%s: %v", source, err)
		}
		if edited == nil {
			continue
		}
		tried++
		line := x.word("editor.map.secondtool")
		_, moveErr := host.Move(ui.KindEnchantment, 0, cell)
		_, deleteErr := host.Delete(ui.KindScenery, 0)
		_, heightErr := host.Paint(ui.BrushStroke{Plane: ui.BrushPlaneHeight, Value: 1, Cells: []image.Point{cell}})
		for _, err := range []error{moveErr, deleteErr, heightErr} {
			if err == nil || err.Error() != line {
				t.Fatalf("%s: other tool not refused with its line: %v", source, err)
			}
		}
		if !host.Dirty() || !host.CanUndo() {
			t.Fatalf("%s: edit left no history", source)
		}
		target, err := host.SaveAs(filepath.Join(t.TempDir(), "edited.alm"))
		if err != nil {
			t.Fatal(source, err)
		}
		saved, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		reloaded, err := x.Open(target)
		if err != nil {
			t.Fatalf("%s: the saved map does not reload: %v", source, err)
		}
		got, err := alm.OpenROM2(saved)
		if err != nil || !bytes.Equal(reloaded.SourceBytes, saved) {
			t.Fatalf("%s: reload differs: %v", source, err)
		}
		if slices.Equal(got.Tiles, was.Tiles) {
			t.Fatalf("%s: the edit is not in the saved map", source)
		}
		if len(saved) != len(raws[i]) || !onlyTileEditDiffers(t, raws[i], saved) {
			t.Fatalf("%s: bytes other than the tile plane and its group mask changed", source)
		}
		if tried == 3 {
			break
		}
	}
	if tried == 0 {
		t.Fatal("no map took a terrain edit")
	}
	t.Logf("PASS: %d terrain edits saved, reloaded and compared (%s)", tried, installLanguage(x))
}

// onlyTileEditDiffers reports that a and b agree outside the type-1 payload
// and the type-0 group mask word.
func onlyTileEditDiffers(t *testing.T, a, b []byte) bool {
	t.Helper()
	doc, err := alm.OpenDocument(a, alm.SecondFormat)
	if err != nil {
		t.Fatal(err)
	}
	at := alm.FileHeaderSize
	for i, id := range doc.RecordTypeIDs() {
		size := len(doc.RecordPayload(i))
		start := at + alm.RecordHeaderSize
		at = start + size
		for k := start; k < at; k++ {
			skip := id == alm.TypeTiles || id == alm.TypeMeta && k >= start+alm.MetaBitmask && k < start+alm.MetaBitmask+4
			if !skip && a[k] != b[k] {
				return false
			}
		}
	}
	return bytes.Equal(a[:alm.FileHeaderSize], b[:alm.FileHeaderSize])
}
