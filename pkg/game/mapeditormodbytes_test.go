package game

import (
	"bytes"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

// recordHolds reports the page count and whether b holds the texts record.
func recordHolds(t *testing.T, b []byte) (int, bool) {
	t.Helper()
	pages, held, err := alm.TextsOf(b)
	if err != nil {
		t.Fatal(err)
	}
	return len(pages), held
}

// mapBytesCases are a map whose pages are inside and one without pages.
func mapBytesCases(t *testing.T) map[string][]byte {
	return map[string][]byte{
		"with pages":    withInside(t, insideMap(t, true), insidePages()),
		"without pages": insideMap(t, true),
	}
}

// Each route that writes a map's bytes writes the bytes it wrote before the
// one builder: a mod's map file has no record, a mod draft and the save routes
// have it as they had. The expected bytes are the old expressions.
func TestMapBytesKeepsEachRoutesBytes(t *testing.T) {
	x := insideInspector(t)
	for name, src := range mapBytesCases(t) {
		path := filepath.Join(t.TempDir(), "map.almx")
		writeTestPlayInput(t, path, src)
		t.Run(name, func(t *testing.T) {
			s := insideHost(t, x, path)
			raw := s.model.Bytes()
			save, err := s.model.SaveBytes()
			if err != nil {
				t.Fatal(err)
			}
			withRecord, err := s.withTexts(raw)
			if err != nil {
				t.Fatal(err)
			}
			saveRecord, err := s.withTexts(save)
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range []struct {
				route string
				p     mapBytesPurpose
				want  []byte
			}{
				{"editor save and package", mapSave, saveRecord},
				{"test play and new map", mapSaveBare, save},
				{"mod draft", mapDraft, withRecord},
				{"mod map file", mapModFile, raw},
			} {
				got, err := s.mapBytes(c.p)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, c.want) {
					t.Errorf("%s: bytes differ from the route's own (%d, want %d)", c.route, len(got), len(c.want))
				}
			}

			w := &editorWorkspaceState{Package: editorPackageSpec{ID: "probe", Title: "probe"}}
			if err := s.newModDocument(w); err != nil {
				t.Fatal(err)
			}
			if got := w.Project.Missions[0].MapBytes; !bytes.Equal(got, raw) {
				t.Errorf("new mod document: bytes differ from the model's (%d, want %d)", len(got), len(raw))
			}

			d := &editorModDocument{Files: map[string][]byte{}}
			place, err := s.draftOpenMap(d)
			if err != nil {
				t.Fatal(err)
			}
			if got := d.Files[place.Map]; !bytes.Equal(got, raw) {
				t.Errorf("add open map: bytes differ from the model's (%d, want %d)", len(got), len(raw))
			}
			if _, held := recordHolds(t, d.Files[place.Map]); held {
				t.Error("add open map writes a record: its pages are folder files")
			}
		})
	}
}

// The open map a mod draft carries keeps the pages the map holds inside; a
// map without pages is carried as it was.
func TestMapBytesOpenMapDraftKeepsPages(t *testing.T) {
	for name, src := range mapBytesCases(t) {
		t.Run(name, func(t *testing.T) {
			from, dir := keptMod(t, src)
			to, err := from.inspector.Open(filepath.Join(dir, "maps", "second.almx"))
			if err != nil {
				t.Fatal(err)
			}
			next := to.Edits.(*mapInspectionEdits)
			from.saved = nil
			if err := next.Shown(from, []ui.InspectionEdits{from, next}); err != nil {
				t.Fatal(err)
			}
			m, ok := from.ownMission(from.project0())
			if !ok {
				t.Fatal("the open map is not a mission of the mod")
			}
			got := next.workspace.Project.Files[m.Map]
			n, held := recordHolds(t, got)
			if name == "with pages" {
				if !held || n != len(insidePages()) {
					t.Errorf("open map draft drops the pages: held %v, pages %d", held, n)
				}
				return
			}
			if held || !bytes.Equal(got, from.model.Bytes()) {
				t.Errorf("open map draft of a map without pages changed: held %v", held)
			}
		})
	}
}
