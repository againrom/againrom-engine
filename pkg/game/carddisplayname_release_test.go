package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func cardCaptionMission(t *testing.T, f *FrontEnd, mission ui.MapOpener) *ui.App {
	t.Helper()
	f.SetDeterministicFrames(true)
	app := f.App("installed card captions")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(mission); err != nil {
		t.Fatal(err)
	}
	for range 16 {
		if !f.live.mission.open {
			break
		}
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if !f.live.stopped {
		if err := app.HeadlessKey("0"); err != nil {
			t.Fatal(err)
		}
	}
	return app
}

func cardCaptionRender(t *testing.T, f *FrontEnd, app *ui.App, typeID int32, stage string) {
	t.Helper()
	pic, err := app.HeadlessMissionCard()
	if err != nil || pic == nil || pic.Bounds().Empty() {
		t.Fatal("installed mission card did not render", stage, typeID, err)
	}
	out := os.Getenv("AGAINROM_WIDGET_KIT_WITNESS_DIR")
	if out == "" {
		return
	}
	root, err := editorPhysicalDirectory(f.Archives.Root)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(root, filepath.Clean(out))
	if !filepath.IsAbs(out) || err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)) {
		t.Fatal("card output must be absolute and outside the installed root")
	}
	lang := strings.ToLower(filepath.Base(root))
	writeWidgetKitPNG(t, filepath.Join(out, fmt.Sprintf("%s-card-type-%d-%s.png", lang, typeID, stage)), pic)
}

func cardCaptionCheck(t *testing.T, f *FrontEnd, app *ui.App, id sim.EntityID, want, stage string) {
	t.Helper()
	if want == "" {
		t.Fatal("installed card caption is empty")
	}
	subject, statement := enemyCardSubject(t, app, f.live, id)
	cardCaptionRender(t, f, app, int32(subject.UnitNameIndex), stage)
	if name, present := ui.PanelSubjectName(subject); !present || name != want {
		t.Fatalf("type %d card caption=%q, want installed caption %q", subject.UnitNameIndex, name, want)
	}
	if !slices.Contains(statement, want) {
		t.Fatalf("type %d drawn card does not contain its complete installed caption", subject.UnitNameIndex)
	}
}

func TestReleaseSavedCreatureCardsUseInstalledClassCaptions(t *testing.T) {
	f := releaseFront(t)
	app := cardCaptionMission(t, f, f.DirectNewGame(120))
	unitNames := LoadTextTable(f.Archives.Containers, UnitNameTextPath)
	if unitNames == nil {
		t.Fatal("install has no unit-name table")
	}
	chosen := map[int32]sim.EntityID{}
	for _, e := range f.live.world.Entities() {
		if e.Alive() && e.TypeID >= 0x40 && e.TypeID <= 0x50 {
			if _, seen := chosen[e.TypeID]; !seen {
				chosen[e.TypeID] = e.ID
			}
		}
	}
	bat, found := chosen[70]
	if !found || len(chosen) < 2 {
		t.Fatal("mission 120 does not supply a bat and another creature")
	}
	otherType := int32(0)
	for typ := range chosen {
		if typ != 70 && (otherType == 0 || typ < otherType) {
			otherType = typ
		}
	}
	for _, id := range []sim.EntityID{bat, chosen[otherType]} {
		e, _ := f.live.entity(id)
		name, ok := unitNames.At(int(e.TypeID))
		if !ok {
			t.Fatal("installed unit-name index is absent", e.TypeID)
		}
		cardCaptionCheck(t, f, app, id, name, "fresh")
	}

	raw, _, _ := saveCurrentEffect(t, f)
	doc, origins, err := sav.DecodeDocumentDataWithOrigins(raw)
	if err != nil {
		t.Fatal(err)
	}
	objects := map[uint16]uint16{}
	for _, origin := range origins {
		objects[origin.ArchiveIndex] = origin.ObjectIndex
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := file.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	internalNames := map[uint16]string{}
	for _, actor := range graph.Actors {
		if actor.Class != "Unit" || int32(actor.TypeID) != 70 && int32(actor.TypeID) != otherType {
			continue
		}
		if int(actor.DefRow) >= f.Table.Units.Len() || actor.MapUnitID == 0 {
			t.Fatal("installed creature lacks its definition row or map id")
		}
		name := f.Table.Units.EntryName(int(actor.DefRow))
		if name == "" {
			t.Fatal("installed creature definition key is empty")
		}
		index := objects[actor.ArchiveIndex]
		if index == 0 || int(index) > len(doc.Objects) || doc.Objects[index-1].Class != actor.Class {
			t.Fatal("creature archive record has no matching document object")
		}
		mustSetText(&doc.Objects[index-1], "Name", name)
		internalNames[actor.MapUnitID] = name
	}
	if len(internalNames) < 2 {
		t.Fatal("SAV supplies fewer than two creature definition keys")
	}
	doc.State.ValueRecords = slices.DeleteFunc(doc.State.ValueRecords, func(row sav.CityStateRecordData) bool {
		return row.Path == sav.NativeActionsPath
	})
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := releaseFront(t)
	mission, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("ordinary SAV LOAD: town=%t err=%v", town, err)
	}
	coldApp := cardCaptionMission(t, cold, mission)
	for cycle := range 2 {
		stage := "original-load"
		if cycle != 0 {
			stage = "current-load"
		}
		manifest := cloneActorManifest(cold.live.mission.state.ActorManifest)
		hash := cold.live.world.Hash()
		checked := map[int32]bool{}
		for _, e := range cold.live.world.Entities() {
			internal, found := internalNames[e.MapUnitID]
			if !found || !e.Alive() {
				continue
			}
			if cold.live.actorNames[e.ID] != internal {
				t.Fatal("LOAD changed ordinary UnitState Name", e.MapUnitID)
			}
			if checked[e.TypeID] {
				continue
			}
			name, ok := unitNames.At(int(e.TypeID))
			if !ok {
				t.Fatal("installed unit-name index is absent", e.TypeID)
			}
			cardCaptionCheck(t, cold, coldApp, e.ID, name, stage)
			checked[e.TypeID] = true
		}
		if len(checked) != 2 || cold.live.world.Hash() != hash || !reflect.DeepEqual(cold.live.mission.state.ActorManifest, manifest) {
			t.Fatal("card witness changed World/ordinary Name or missed a creature kind", len(checked))
		}
		if cycle == 0 {
			raw, _, _ = saveCurrentEffect(t, cold)
			cold = releaseFront(t)
			mission, town, err = cold.RestoreOriginal(raw)
			if err != nil || town {
				t.Fatalf("current SAV cold LOAD: town=%t err=%v", town, err)
			}
			coldApp = cardCaptionMission(t, cold, mission)
		}
	}

	controls := releaseFront(t)
	controlsApp := cardCaptionMission(t, controls, controls.MissionOpener(20))
	checkedPerson := false
	for _, e := range controls.live.world.Entities() {
		if !e.Alive() || !e.Humanoid || sim.InPersistBand(e.TypeID) || controls.live.missionPartyMember(e.ID) != nil {
			continue
		}
		name, ok := LoadTextTable(controls.Archives.Containers, UnitNameTextPath).At(int(e.TypeID))
		if ok && name != "" {
			cardCaptionCheck(t, controls, controlsApp, e.ID, name, "human")
			checkedPerson = true
			break
		}
	}
	if !checkedPerson {
		t.Fatal("mission 20 provides no ordinary Human name control")
	}
	buildings := LoadTextTable(controls.Archives.Containers, BuildingTextPath)
	checkedStructure := false
	for _, structure := range controls.live.world.Structures() {
		if structure.MaxHealth == 0 {
			continue
		}
		kind := int(controls.live.mission.state.Map.Objects[structure.ID].Kind)
		name, ok := buildings.At(kind - 1)
		if !ok || name == "" {
			continue
		}
		inspectionCentre(controls.live, int(structure.Col), int(structure.Row))
		releaseHoverInspection(t, controlsApp, controls.live, ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(structure.ID)})
		subject, ok := controls.live.view.InspectionPanel()
		if got, present := ui.PanelSubjectName(subject); !ok || !present || got != name {
			t.Fatal("structure card differs from the installed building name", kind)
		}
		checkedStructure = true
		break
	}
	if !checkedStructure {
		t.Fatal("mission 20 provides no structure name control")
	}
}
