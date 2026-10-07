package game

import (
	"bytes"
	"image"
	"image/draw"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Mission 71's npc59 is one of three placed Lancers on the Humans row that the
// campaign names for a rider of the lance: the row's class is 21, and a scenario
// placed human is drawn with his row's own class (UNIT-APPEAR-030). The dialogue
// figure is the world figure compositor's output on that drawable
// (DLG-FIGURE-020), and it draws the horse layer under a rider of class 0x11 to
// 0x15 (HERO-DOLL-078). The roster template that stands for the placed person
// carries his row's class on a fresh open and a SAV load replaces it with the
// class the entity was placed with; the horse must follow neither.
//
// The mission raises npc59's first message on its first script pass, and the
// state is reached with ordinary frames and the Enter key. SAVE is the ordinary
// producer, written before the first pass; LOAD is the load dialog of a cold
// front end whose own frames raise the message again. The expected picture is
// composed from the installed row and the entity's worn set, and the horse
// sheet is read without the compositor's class gate.
func TestReleasePlacedLancerSpeakerDrawsHisRowClassFreshAndAfterLoad(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	a := f.App("lancer speaker")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(71)); err != nil {
		t.Fatal(err)
	}
	path, _ := writeOrdinarySAV(t, f, "lancer-speaker.sav")
	fresh := f.live

	freshFace, id := lancerSpeaks(t, f, a, "fresh")
	row, rider, mounted := lancerRowFigures(t, f, fresh, id)
	if !imagesEqual(freshFace, mounted) || imagesEqual(freshFace, rider) {
		t.Errorf("fresh: npc59 speaks without his horse: figure is the mounted composition %t, the bare rider %t",
			imagesEqual(freshFace, mounted), imagesEqual(freshFace, rider))
	}

	g, b := lancerLoad(t, filepath.Dir(path), filepath.Base(path))
	cold := g.live
	loadedFace, loaded := lancerSpeaks(t, g, b, "loaded")
	if loaded != id {
		t.Errorf("after LOAD npc59 is entity %d, before SAVE entity %d", loaded, id)
	}
	if !imagesEqual(loadedFace, mounted) || !imagesEqual(loadedFace, freshFace) {
		t.Errorf("after LOAD: npc59's figure is the mounted composition %t and the fresh figure %t",
			imagesEqual(loadedFace, mounted), imagesEqual(loadedFace, freshFace))
	}

	// The template class a fresh mission carries for him is his row's own class:
	// worn equipment does not rewrite it (ANIM-106).
	before, after := fresh.mission.state.Start.Roster[id].Class, cold.mission.state.Start.Roster[id].Class
	if before != row.TypeID || !data.FigureHasHorse(row.TypeID) {
		t.Fatalf("npc59 template class %d before SAVE and %d after LOAD, row class %d",
			before, after, row.TypeID)
	}
	t.Logf("npc59 is entity %d: template class %d before SAVE, %d after LOAD; row %d; %d of the figure's pixels are the horse's",
		id, before, after, row.TypeID, horsePixels(mounted, rider))
}

// lancerSpeaks steps ordinary frames until the mission raises its first
// message, checks that its first part is npc59's, and returns the picture
// standing in the dialogue pane with the entity it shows. The Enter key pages
// the message and closes it.
func lancerSpeaks(t *testing.T, f *FrontEnd, a *ui.App, when string) (*image.RGBA, sim.EntityID) {
	t.Helper()
	live := f.live
	start := live.world.Tick()
	for i := 0; i < 400 && !a.HeadlessNoticeOpen(); i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if !a.HeadlessNoticeOpen() || live.world.Tick() == start {
		t.Fatalf("%s: no message opened while the world ran from tick %d to %d", when, start, live.world.Tick())
	}
	payload, ok := ReadEventText(f.Archives.Containers, 71, 4)
	m := live.mission
	if !ok || !bytes.Equal(m.payload, payload) || m.part != 1 {
		t.Fatalf("%s: the open message is part %d of another text than mission 71's fourth", when, m.part)
	}
	if speaker, named := EventPartSpeaker(payload, 1, m.audience); !named || speaker != 59 {
		t.Fatalf("%s: the first part names speaker %d/%t, want npc59", when, speaker, named)
	}
	portrait, face := live.view.NoticeSpeaker()
	if !portrait || face == nil {
		t.Fatalf("%s: the message window shows no portrait pane", when)
	}
	cast := speakerCast{actors: live.speakerActors, alive: live.entityAlive, worn: live.equipmentOf}
	if dir, ok := live.playerFigureDir(); ok {
		cast.playerDir, cast.hasPlayer = dir, true
	}
	actor, found := cast.resolve(live.npcFaces[59])
	if !found {
		t.Fatalf("%s: no live person answers for npc59", when)
	}
	for n := 0; n < 4 && a.HeadlessNoticeOpen(); n++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if a.HeadlessNoticeOpen() {
		t.Fatalf("%s: Enter left the message open", when)
	}
	return face, actor.id
}

// lancerRowFigures is the picture a person of the installed Lancer row draws
// in a dialogue, built from the row's own columns and the worn set the world
// holds for the entity: the rider alone, and the rider over the horse sheet.
// It states the row's figure directory, sheet and class beside the resolved
// speaker's own and fails on any disagreement.
func lancerRowFigures(t *testing.T, f *FrontEnd, live *mapWorld, id sim.EntityID) (data.HumanDef, *image.RGBA, *image.RGBA) {
	t.Helper()
	ms := live.mission.state
	e, _ := live.entity(id)
	var unit int
	for i := range ms.Map.Units {
		if ms.Map.Units[i].UnitID == e.MapUnitID {
			unit = i
		}
	}
	index := data.FindHumanByName(f.Table.Humans, "NPC08_2")
	if index == data.NotFound || mapload.Resolve(ms.Map.Units[unit], f.Table).Index != index {
		t.Fatalf("npc59's placement %d does not resolve to the row NPC08_2", unit)
	}
	row, err := data.NewHumanDef("NPC08_2", f.Table.Humans.EntryParams(index))
	if err != nil {
		t.Fatal(err)
	}
	dir, face := data.FigureFor(row.TypeID, row.Face, row.Gender)
	for _, a := range live.speakerActors {
		if a.id == id && (a.fig != (figureID{Dir: dir, Face: face}) || a.typeID != row.TypeID) {
			t.Fatalf("npc59's candidate carries figure %+v and class %d; his row gives %s %d and class %d",
				a.fig, a.typeID, dir, face, row.TypeID)
		}
	}
	slots, _ := live.world.Equipped(id)
	composed, _ := composeInventorySubject(f.Archives.Containers, uint32(id), equipmentFromSlots(slots), dir, face)
	if composed.Figure == nil {
		t.Fatalf("figure %s %d is unreadable", dir, face)
	}
	sheet, err := f.Archives.Containers.ReadFile("graphics/infowindow/horse.bmp")
	if err != nil {
		t.Fatal(err)
	}
	mounted := inspectionPortraitBMP(t, sheet)
	draw.Draw(mounted, mounted.Bounds(), composed.Figure, composed.Figure.Bounds().Min, draw.Over)
	return row, composed.Figure, mounted
}

// horsePixels counts the pixels where the mounted composition differs from the
// rider alone, so the comparison above cannot pass on two equal pictures.
func horsePixels(mounted, rider *image.RGBA) int {
	n := 0
	for y := 0; y < mounted.Bounds().Dy(); y++ {
		for x := 0; x < mounted.Bounds().Dx(); x++ {
			if mounted.RGBAAt(x, y) != rider.RGBAAt(x, y) {
				n++
			}
		}
	}
	return n
}

// lancerLoad loads name from dir through the load dialog of a cold front end
// and leaves the mission running.
func lancerLoad(t *testing.T, dir, name string) (*FrontEnd, *ui.App) {
	t.Helper()
	g := releaseFront(t)
	g.Options = OptionsStore{}
	g.SetDeterministicFrames(true)
	b := g.App("lancer speaker load")
	t.Cleanup(b.StopAudio)
	b.Layout(1024, 768)
	_, list, load := g.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	b.SetSaveSeams(nil, list, load)
	groundAppLoad(t, b, list, name)
	return g, b
}
