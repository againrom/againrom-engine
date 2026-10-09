package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Sarindar, the placed person npc52, is a man mage in his own robe and amulet.
// Mission 10's Snoot message names him in its parts 2 and 4, and the dialogue
// window draws him in that worn set (DLG-SPEAKER-022). The ordinary SAVE
// writes three departed creatures placed before him as terminal records
// (DIV-2501), so LOAD keeps their placements; the owner save below withdraws
// them. He must speak in his worn set after either load.
//
// The state is reached through ordinary input (the selection, the attack key
// and a click on the target, a click on the ground) and ordinary ticks; each
// mission notice closes with Enter. SAVE is the ordinary producer and LOAD the
// load dialog of a cold front end. Mission 20's Sarindar is the control.
func TestReleasePlacedSpeakerKeepsHisWornSetAfterLoad(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	a := f.App("placed speaker")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live := f.live
	payload, ok := ReadEventText(f.Archives.Containers, 10, 3)
	if !ok {
		t.Fatal("mission 10 ships no Snoot message")
	}
	for _, part := range []int{2, 4} {
		if speaker, named := EventPartSpeaker(payload, part, live.mission.audience); !named || speaker != 52 {
			t.Fatalf("Snoot message part %d names speaker %d/%t, want npc52", part, speaker, named)
		}
	}
	id, before := sarindarSpeaks(t, f, live, "mission 10")

	// The two rogues, a walk to 42,47, and from there three creatures placed
	// before Sarindar, whose bodies leave the world within a few ticks.
	r := placedSpeakerRoute{t: t, a: a, live: live, hero: live.mission.ids[0]}
	r.kill(0)
	r.kill(1)
	r.walk(42, 47)
	gone := []sim.EntityID{15, 14, 12}
	for _, target := range gone {
		r.kill(target)
	}
	for n := 0; ; n++ {
		left := 0
		for _, target := range gone {
			if _, ok := live.entity(target); ok {
				left++
			}
		}
		if left == 0 {
			break
		}
		if n == 2000 {
			t.Fatalf("%d of the three creatures still stand in the world", left)
		}
		r.advance()
	}
	t.Logf("route done at tick %d", live.world.Tick())

	path, _ := writeOrdinarySAV(t, f, "placed-speaker.sav")
	g, b, cold := placedSpeakerLoad(t, filepath.Dir(path), filepath.Base(path))
	if n := len(cold.mission.state.Map.Units); n != len(live.mission.state.Map.Units) {
		t.Fatalf("the load kept %d placements of %d; a departed creature's record keeps its placement", n, len(live.mission.state.Map.Units))
	}
	loaded, after := sarindarSpeaks(t, g, cold, "mission 10 after LOAD")
	if loaded != id || !bytes.Equal(after.Pix, before.Pix) {
		t.Errorf("after LOAD npc52 is entity %d with another face; before SAVE entity %d", loaded, id)
	}
	sarindarPanes(t, g, b, cold, loaded, "mission 10 after LOAD")

	c := f.App("placed speaker mission 20")
	t.Cleanup(c.StopAudio)
	c.Layout(1024, 768)
	if err := c.OpenMission(f.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	sarindarSpeaks(t, f, f.live, "mission 20")
}

// The owner's mission-10 save, whose load withdrew three gone creatures placed
// before Sarindar and stripped him in the dialogue window. It is loaded
// through the load dialog, saved by the ordinary producer and loaded again.
func TestReleaseOwnerSaveSpeakerKeepsHisWornSet(t *testing.T) {
	path := os.Getenv("AGAINROM_SARINDAR_SAV")
	if path == "" {
		t.Skip("no AGAINROM_SARINDAR_SAV: the owner's mission-10 save is an owner input")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "18e126de76e914ae1da420eacaf0da53917d931755f2bd815df3337c13bf1152" {
		t.Fatalf("owner save changed: %s", got)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mission10.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, _, live := placedSpeakerLoad(t, dir, "mission10.sav")
	id, before := sarindarSpeaks(t, f, live, "owner save")
	if n := len(live.mission.state.Map.Units); n > int(id) {
		t.Fatalf("the owner save kept %d placements; Sarindar's id %d still names one", n, id)
	}
	resaved, _ := writeOrdinarySAV(t, f, "owner-resaved.sav")
	g, b, cold := placedSpeakerLoad(t, filepath.Dir(resaved), filepath.Base(resaved))
	loaded, after := sarindarSpeaks(t, g, cold, "owner save after SAVE and LOAD")
	if loaded != id || !bytes.Equal(after.Pix, before.Pix) {
		t.Errorf("after SAVE and LOAD npc52 is entity %d with another face; before entity %d", loaded, id)
	}
	sarindarPanes(t, g, b, cold, loaded, "owner save after SAVE and LOAD")
}

// placedSpeakerLoad loads name from dir through the load dialog of a cold
// front end.
func placedSpeakerLoad(t *testing.T, dir, name string) (*FrontEnd, *ui.App, *mapWorld) {
	t.Helper()
	g := releaseFront(t)
	g.Options = OptionsStore{}
	g.SetDeterministicFrames(true)
	b := g.App("placed speaker load")
	t.Cleanup(b.StopAudio)
	b.Layout(1024, 768)
	_, list, load := g.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	b.SetSaveSeams(nil, list, load)
	groundAppLoad(t, b, list, name)
	releasePauseMission(t, g, b)
	return g, b, g.live
}

// sarindarSpeaks checks the picture the dialogue window draws for npc52
// against his own live worn set and returns his entity and that picture. The
// expected picture is composed from what the world says he wears, on the
// figure his row names; his robe's own sheet, decoded independently of the
// compositor, must show over the body sheet beneath it.
func sarindarSpeaks(t *testing.T, f *FrontEnd, live *mapWorld, when string) (sim.EntityID, *image.RGBA) {
	t.Helper()
	var id sim.EntityID
	found := 0
	for pid, m := range live.mission.state.Start.Roster {
		if m.CompanionNPC == 52 {
			id, found = pid, found+1
		}
	}
	e, alive := live.entity(id)
	slots, _ := live.world.Equipped(id)
	fig := live.figures[id]
	if found != 1 || !alive || e.TypeID != 0x17 || fig.Dir != data.FigureDirManMage || fig.Face != 6 || slots[4] == 0 || slots[6] == 0 {
		t.Fatalf("%s: Sarindar discriminator changed: %d found, %+v slots=%x figure=%+v", when, found, e, slots, fig)
	}
	face, _, ok := live.SpeakerFace(52)
	worn, _ := composeUnitFigure(f.Archives.Containers, equipmentFromSlots(slots), figureID{Dir: fig.Dir, Face: fig.Face})
	if !ok || face == nil || worn == nil {
		t.Fatalf("%s: npc52 face %t, worn composition %t", when, face != nil, worn != nil)
	}
	robe := inspectionSprite(t, f, "graphics/equipment/mmage/primary/1507213.256")
	body := inspectionSprite(t, f, "graphics/equipment/mmage/6.256")
	covered, drawn, bare := 0, 0, 0
	for y := 0; y < robe.Bounds().Dy(); y++ {
		for x := 0; x < robe.Bounds().Dx(); x++ {
			c := robe.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			covered++
			switch got := face.RGBAAt(x, y); {
			case got == c:
				drawn++
			case got == body.RGBAAt(x, y):
				bare++
			}
		}
	}
	t.Logf("%s: npc52 entity %d slots %x face %v worn %v robe %v body %v: %d of %d robe pixels drawn, %d show the body",
		when, id, slots, face.Bounds(), worn.Bounds(), robe.Bounds(), body.Bounds(), drawn, covered, bare)
	if face.Bounds() != worn.Bounds() || !bytes.Equal(face.Pix, worn.Pix) || covered < 1000 || drawn*10 < covered*9 {
		t.Errorf("%s: npc52 speaks without his own worn set: %d of %d robe pixels drawn, %d show the body",
			when, drawn, covered, bare)
	}
	return id, face
}

// sarindarPanes checks the information pane on the hover and the selection
// route against his worn composition. Only the presentation fog is opened,
// and the world hash must not move.
func sarindarPanes(t *testing.T, f *FrontEnd, a *ui.App, live *mapWorld, id sim.EntityID, when string) {
	t.Helper()
	before := live.world.Hash()
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	e, _ := live.entity(id)
	slots, _ := live.world.Equipped(id)
	fig := live.figures[id]
	worn, _ := composeUnitFigure(f.Archives.Containers, equipmentFromSlots(slots), figureID{Dir: fig.Dir, Face: fig.Face})
	inspectionCentre(live, int(e.X), int(e.Y))
	live.push()
	releaseHoverInspection(t, a, live, ui.InspectionSubject{Kind: ui.InspectionUnit, ID: uint32(id)})
	pic, stats, err := a.HeadlessCharacterPane()
	if err != nil || stats {
		t.Fatalf("%s hovered: figure pane unavailable: stats=%t err=%v", when, stats, err)
	}
	checkInspectionFigure(t, pic, worn, when+" hovered")
	if err := a.HeadlessPointer("hover", -1, -1); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(when, err)
	}
	if err := a.HeadlessPointer("hover", -1, -1); err != nil {
		t.Fatal(err)
	}
	if got, ok := live.view.SelectedUnit(); !ok || got != uint32(id) {
		t.Fatalf("%s: selection = %d/%t", when, got, ok)
	}
	pic, stats, err = a.HeadlessCharacterPane()
	if err != nil || stats {
		t.Fatalf("%s selected: figure pane unavailable: stats=%t err=%v", when, stats, err)
	}
	checkInspectionFigure(t, pic, worn, when+" selected")
	if live.world.Hash() != before {
		t.Fatalf("%s: portrait inspection changed the world", when)
	}
}

// placedSpeakerRoute drives mission 10's entering hero through ordinary input.
type placedSpeakerRoute struct {
	t    *testing.T
	a    *ui.App
	live *mapWorld
	hero sim.EntityID
}

// advance is one ordinary tick; a mission notice it opens closes with Enter.
func (r placedSpeakerRoute) advance() {
	r.t.Helper()
	r.live.tick()
	for k := 0; k < 16 && r.live.mission.open; k++ {
		if err := r.a.HeadlessKey("enter"); err != nil {
			r.t.Fatal(err)
		}
	}
	if r.live.mission.open {
		r.t.Fatal("a mission notice did not close")
	}
}

func (r placedSpeakerRoute) selectHero() {
	r.t.Helper()
	h, ok := r.live.entity(r.hero)
	if !ok || !h.Alive() {
		r.t.Fatal("the entering hero is gone")
	}
	inspectionCentre(r.live, int(h.X), int(h.Y))
	r.live.push()
	if err := r.a.HeadlessSelectEntity(uint32(r.hero)); err != nil {
		r.t.Fatal(err)
	}
}

// kill orders the hero to attack target with the attack key and a click on
// it, and advances until the target is dead.
func (r placedSpeakerRoute) kill(target sim.EntityID) {
	r.t.Helper()
	r.selectHero()
	e, ok := r.live.entity(target)
	if !ok || !e.Alive() {
		r.t.Fatalf("target %d is already gone", target)
	}
	inspectionCentre(r.live, int(e.X), int(e.Y))
	r.live.push()
	if r.live.view.AttackArmed() {
		r.t.Fatal("attack mode is armed before the key")
	}
	if err := r.a.HeadlessKey("a"); err != nil {
		r.t.Fatal(err)
	}
	x, y, err := r.a.HeadlessEntityPoint(uint32(target))
	if err != nil {
		r.t.Fatal(err)
	}
	n := len(r.live.pending)
	for _, edge := range []string{"press", "release"} {
		if err := r.a.HeadlessPointer(edge, x, y); err != nil {
			r.t.Fatal(err)
		}
	}
	if len(r.live.pending) != n+1 || r.live.pending[n].Kind != sim.KindAttack || r.live.pending[n].Entity != r.hero ||
		r.live.pending[n].X != int32(target) {
		r.t.Fatalf("the click on %d queued %+v, want the hero's attack", target, r.live.pending[n:])
	}
	for i := 0; i < 1500; i++ {
		r.advance()
		if c, ok := r.live.entity(target); !ok || !c.Alive() {
			r.t.Logf("target %d down at tick %d", target, r.live.world.Tick())
			return
		}
	}
	r.t.Fatalf("target %d still alive at tick %d", target, r.live.world.Tick())
}

// walk orders the hero to the ground cell (col, row) with a click on it and
// advances until he stands there.
func (r placedSpeakerRoute) walk(col, row int) {
	r.t.Helper()
	r.selectHero()
	inspectionCentre(r.live, col, row)
	r.live.push()
	x, y, found := 0, 0, false
	for py := 8; py < 760 && !found; py += 4 {
		for px := 8; px < 1016; px += 4 {
			if cx, cy, err := r.a.HeadlessDropCell(px, py); err == nil && cx == col && cy == row {
				x, y, found = px, py, true
				break
			}
		}
	}
	if !found {
		r.t.Fatalf("no window pixel lands on %d,%d", col, row)
	}
	n := len(r.live.pending)
	for _, edge := range []string{"press", "release"} {
		if err := r.a.HeadlessPointer(edge, x, y); err != nil {
			r.t.Fatal(err)
		}
	}
	if len(r.live.pending) != n+1 || r.live.pending[n].Entity != r.hero || r.live.pending[n].X != int32(col) || r.live.pending[n].Y != int32(row) {
		r.t.Fatalf("the click on %d,%d queued %+v, want the hero's walk", col, row, r.live.pending[n:])
	}
	for i := 0; i < 4000; i++ {
		r.advance()
		if h, _ := r.live.entity(r.hero); int(h.X) == col && int(h.Y) == row {
			r.t.Logf("hero at %d,%d at tick %d", col, row, r.live.world.Tick())
			return
		}
	}
	h, _ := r.live.entity(r.hero)
	r.t.Fatalf("the hero stands at %d,%d, not %d,%d", h.X, h.Y, col, row)
}
