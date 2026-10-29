package game

import (
	"bytes"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func nativeLookEditor(t *testing.T) (*FrontEnd, *mapInspectionEdits, []byte) {
	t.Helper()
	f := structureFront(t, true)
	p := figureRow(3, 2, 1)
	p[24] = 1234
	f.Table.Humans.(dbCollection)[1] = dbEntry{name: "Guard", params: p}
	f.Units.Classes[3] = worldFixtureArt(16, 16, 8, 16, 3, 3, 1)
	raw, err := f.Archives.Containers.ReadFile("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	m, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		if err := m.SetField(mapedit.KindStructure, i, "owner", 1); err != nil {
			t.Fatal(err)
		}
	}
	unit, err := mapedit.New(synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: []synth.ALMUnit{{X: 8 << 8, Y: 8 << 8, ClassID: 3, ClassSubID: 0xab80, Flags: 0x2000, UnitID: 7}}}))
	if err != nil {
		t.Fatal(err)
	}
	rec, _ := unit.UnitRecord(0)
	if _, err := m.PlaceUnit(rec); err != nil {
		t.Fatal(err)
	}
	if err := m.SetUnitField(0, "owner", 1); err != nil {
		t.Fatal(err)
	}
	raw = m.Bytes()
	for _, dir := range []data.FigureDir{data.FigureDirManFighter, data.FigureDirWomanFighter} {
		for _, face := range []int{2, 7} {
			pic := invSparseSheet(color.RGBA{R: byte(face * 20), G: 0x40, A: 255}, 0)
			f.Archives.Containers.Overlay(graphicsPrefix+data.ItemFigureBasePath(dir, face), func() ([]byte, error) { return pic, nil })
		}
	}
	x := &MapInspector{archives: f.Archives, table: f.Table, tiles: f.Tiles, units: f.Units, structures: f.Structures}
	doc, err := x.inspect(raw, "look.alm")
	if err != nil {
		t.Fatal(err)
	}
	return f, doc.Edits.(*mapInspectionEdits), raw
}

func TestEditorNativeFaceSexPreviewApplyUndoAndReopen(t *testing.T) {
	_, s, raw := nativeLookEditor(t)
	fields := s.Fields("Unit", 0)
	if got := fields[fieldIndex(t, fields, "sex")].Value; got != 1 {
		t.Fatalf("inherited sex = %d, want female", got)
	}
	_, _, before, err := s.UnitSettings(0, nil)
	if err != nil || before == nil || before.RGBAAt(0, 0).R != 40 {
		t.Fatal("missing inherited face preview", err)
	}
	changes := []ui.InspectionFieldValue{{Key: "face", Value: 7}, {Key: "sex", Value: 0}}
	_, _, preview, err := s.UnitSettings(0, changes)
	if err != nil || preview == nil || preview.RGBAAt(0, 0).R != 140 {
		t.Fatal("draft did not draw the selected face", err)
	}
	if !bytes.Equal(raw, s.model.Bytes()) || s.CanUndo() {
		t.Fatal("preview wrote map/history")
	}
	if _, err := s.SetFields("Unit", 0, changes); err != nil {
		t.Fatal(err)
	}
	m, _ := s.model.Map()
	if m.Units[0].ClassSubID != 0xab87 || m.Units[0].Flags != 0x2000 {
		t.Fatal("wrong native face/sex bits", m.Units[0])
	}
	changed := s.model.Bytes()
	if _, err := s.Undo(); err != nil || !bytes.Equal(raw, s.model.Bytes()) {
		t.Fatal("one Undo did not restore exact map", err)
	}
	if _, err := s.Redo(); err != nil || !bytes.Equal(changed, s.model.Bytes()) {
		t.Fatal("Redo changed bytes", err)
	}
	path, err := s.SaveAs(filepath.Join(t.TempDir(), "look.alm"))
	if err != nil {
		t.Fatal(err, s.Report())
	}
	doc, err := s.inspector.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, _, reloaded, err := doc.Edits.(*mapInspectionEdits).UnitSettings(0, nil)
	if err != nil || reloaded == nil || !bytes.Equal(reloaded.Pix, preview.Pix) {
		t.Fatal("Save/reopen changed the preview", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestEditorNativeSexMaterializesInheritedFace(t *testing.T) {
	_, s, _ := nativeLookEditor(t)
	if _, err := s.SetFields("Unit", 0, []ui.InspectionFieldValue{{Key: "sex", Value: 0}}); err != nil {
		t.Fatal(err)
	}
	m, _ := s.model.Map()
	if m.Units[0].ClassSubID != 0xab82 || m.Units[0].Flags != 0x2000 {
		t.Fatal("sex did not fix inherited face", m.Units[0])
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetField("Unit", 0, "face", 7); err != nil {
		t.Fatal(err)
	}
	m, _ = s.model.Map()
	if m.Units[0].Flags != 0x2004 {
		t.Fatal("face change lost inherited sex", m.Units[0])
	}
}

func TestEditorNativeLookSurvivesMissionSAVAndColdLoad(t *testing.T) {
	f, s, _ := nativeLookEditor(t)
	if _, err := s.SetFields("Unit", 0, []ui.InspectionFieldValue{{Key: "face", Value: 7}, {Key: "sex", Value: 0}}); err != nil {
		t.Fatal(err)
	}
	cold := nativeEditorMissionSAV(t, f, s, nativeLookEditor)
	if want := f.live.figures[0]; want.Face != 7 || want.Dir != data.FigureDirManFighter || cold.live.figures[0] != want {
		t.Fatal("mission/SAV ignored placement look", want, cold.live.figures[0])
	}
}

func nativeEditorMissionSAV(t *testing.T, f *FrontEnd, s *mapInspectionEdits, setup func(*testing.T) (*FrontEnd, *mapInspectionEdits, []byte)) *FrontEnd {
	t.Helper()
	raw, err := s.model.SaveBytes()
	if err != nil {
		t.Fatal(err)
	}
	f.Archives.Containers.Overlay("scenario/10.alm", func() ([]byte, error) { return raw, nil })
	if err := f.App("native editor").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	cold, _, _ := setup(t)
	cold.Archives.Containers.Overlay("scenario/10.alm", func() ([]byte, error) { return raw, nil })
	open, town, err := cold.RestoreOriginal(saved)
	if err != nil || town {
		t.Fatal("cold LOAD", town, err)
	}
	if err := cold.App("native editor loaded").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if cold.live.world.Hash() != f.live.world.Hash() {
		t.Fatal("cold LOAD changed world")
	}
	return cold
}
