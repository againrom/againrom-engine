package game

import (
	"bytes"
	"image"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func nativeIdentityEditor(t *testing.T) (*FrontEnd, *mapInspectionEdits, []byte) {
	t.Helper()
	f, s, raw := nativeLookEditor(t)
	p := figureRow(3, 7, 0)
	p[24] = 4321
	f.Table.Humans.(dbCollection)[2] = dbEntry{name: "Courier", params: p}
	npcBytes := synth.NPCReg(map[int32]int32{0: 1234, 7: 1234, 8: 4321, 9: 9999})
	r, err := reg.Parse(npcBytes)
	if err != nil {
		t.Fatal(err)
	}
	catalog := data.LoadNPCCatalog(r)
	f.Table.NPCCatalog, f.Table.NPC = catalog, catalog.Defs
	f.Table.NPCNames = make([]string, 10)
	f.Table.NPCNames[6], f.Table.NPCNames[7] = "Guard native name", "Courier native name"
	f.Archives.Containers.Overlay("scenario/npc.reg", func() ([]byte, error) { return npcBytes, nil })
	s.unitViews, s.inspector.recordOptions = nil, nil
	return f, s, raw
}

func TestEditorNativeNPCIdentityPreviewSaveUndoAndClone(t *testing.T) {
	_, s, raw := nativeIdentityEditor(t)
	fields := s.Fields("Unit", 0)
	f := fields[fieldIndex(t, fields, "npc")]
	if f.Value != -1 {
		t.Fatal("ordinary face key exposed as NPC identity", f.Value)
	}
	seen := map[int64]string{}
	for _, c := range f.Choices {
		seen[c.Value] = c.Label
	}
	if len(seen) != 4 || !strings.Contains(seen[8], "Courier native name") {
		t.Fatal("wrong native identity catalog", seen)
	}
	_, previewFields, pic, err := s.UnitSettings(0, []ui.InspectionFieldValue{{Key: "npc", Value: 8}})
	if err != nil || pic == nil || pic.RGBAAt(0, 0).R != 140 {
		t.Fatal("NPC draft preview", err)
	}
	for _, f := range previewFields {
		if f.Key == "face" || f.Key == "sex" || f.Key == "gear" || f.Key == "npcflag" {
			t.Fatal("NPC arm offers an unread field", f.Key)
		}
	}
	if !bytes.Equal(raw, s.model.Bytes()) || s.CanUndo() {
		t.Fatal("NPC draft changed records")
	}
	if _, err := s.SetField("Unit", 0, "npc", 8); err != nil {
		t.Fatal(err)
	}
	after := s.model.Bytes()
	m, _ := s.model.Map()
	if m.Units[0].Flags != 0x2001 || m.Units[0].ClassSubID != 8 || m.Units[0].ClassID != 3 || mapload.Resolve(m.Units[0], s.inspector.table).Index != 2 {
		t.Fatal("wrong NPC native record", m.Units[0])
	}
	for _, key := range []string{"face", "sex", "gear", "npcflag"} {
		if _, err := s.SetField("Unit", 0, key, 1); err == nil || !bytes.Equal(after, s.model.Bytes()) {
			t.Fatalf("%s bypassed native NPC admission", key)
		}
	}
	if _, err := s.Undo(); err != nil || !bytes.Equal(raw, s.model.Bytes()) {
		t.Fatal("NPC Undo", err)
	}
	if _, err := s.Redo(); err != nil || !bytes.Equal(after, s.model.Bytes()) {
		t.Fatal("NPC Redo", err)
	}
	path, err := s.SaveAs(filepath.Join(t.TempDir(), "npc.alm"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := s.inspector.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened := doc.Edits.(*mapInspectionEdits)
	if got := reopened.Fields("Unit", 0)[fieldIndex(t, reopened.Fields("Unit", 0), "npc")].Value; got != 8 {
		t.Fatal("NPC identity lost on reopen", got)
	}
	if _, err := reopened.Clone("Unit", 0, image.Pt(12, 8)); err != nil {
		t.Fatal(err)
	}
	m, _ = reopened.model.Map()
	if len(m.Units) != 2 || m.Units[1].ClassSubID != 8 || m.Units[1].Flags != 0x2001 || m.Units[1].UnitID == m.Units[0].UnitID {
		t.Fatal("clone did not copy native identity independently", m.Units)
	}
}

func TestEditorNativeNPCIdentityZeroNoneAndBadRecords(t *testing.T) {
	_, s, raw := nativeIdentityEditor(t)
	for _, id := range []int64{9, 65536, -2} {
		if _, err := s.SetField("Unit", 0, "npc", id); err == nil || !bytes.Equal(raw, s.model.Bytes()) {
			t.Fatal("invalid NPC mutated records", id, err)
		}
	}
	if _, err := s.SetField("Unit", 0, "npc", -1); err != nil || !bytes.Equal(raw, s.model.Bytes()) || s.CanUndo() {
		t.Fatal("None changed an ordinary face key", err)
	}
	if _, err := s.SetField("Unit", 0, "npc", 0); err != nil {
		t.Fatal(err)
	}
	m, _ := s.model.Map()
	if m.Units[0].Flags&1 == 0 || m.Units[0].ClassSubID != 0 {
		t.Fatal("NPC zero confused with None")
	}
	if _, err := s.SetField("Unit", 0, "npc", -1); err != nil {
		t.Fatal(err)
	}
	m, _ = s.model.Map()
	if m.Units[0].Flags&1 != 0 || m.Units[0].ClassSubID != 0 || !gearOffered(m.Units[0], s.inspector.table) {
		t.Fatal("None retained NPC mode")
	}
	u := alm.Unit{ClassID: 3, Flags: 0x10, ServerID: 1234}
	table := *s.inspector.table
	table.UnitKeys = mapload.ServerUnitKeys
	if !mapload.Resolve(u, &table).Found() || table.ClassPlacementFields() {
		t.Fatal("server-key Human admitted to class-key fields")
	}
}

func TestEditorNativeNPCIdentitySurvivesMissionSAV(t *testing.T) {
	f, s, _ := nativeIdentityEditor(t)
	if _, err := s.SetField("Unit", 0, "npc", 8); err != nil {
		t.Fatal(err)
	}
	cold := nativeEditorMissionSAV(t, f, s, nativeIdentityEditor)
	if f.live.mission.state.Start.Roster[0].CompanionNPC != 8 {
		t.Fatal("mission lost the NPC identity")
	}
	if cold.live.mission.state.Start.Roster[0].CompanionNPC != 8 || cold.live.figures[0] != f.live.figures[0] || cold.live.world.Hash() != f.live.world.Hash() {
		t.Fatal("cold LOAD changed native identity or state")
	}
}
