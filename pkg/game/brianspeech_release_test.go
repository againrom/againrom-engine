package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/ui"
)

func TestReleaseLoadedBrianKeepsHisClothesInTheTurnDialogue(t *testing.T) {
	path := os.Getenv("AGAINROM_BRIAN_TURN_INPUT")
	if path == "" {
		t.Skip("AGAINROM_BRIAN_TURN_INPUT is not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const sourceHash = "e0d7f9e7df2548a03b5db68aa695519ff240ed775f25e1caeeedc6c0fdae2c94"
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != sourceHash {
		t.Fatal("Brian turn source save changed")
	}
	t.Cleanup(func() {
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(raw, after) {
			t.Error("Brian turn source bytes changed", err)
		}
	})
	f := shopOrderFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "ogrestuck.sav")
	t.Cleanup(app.StopAudio)
	if f.liveMission != 40 {
		t.Fatal("source did not load mission 40")
	}
	want := brianClothedSpeaker(t, f, "original LOAD")
	_, current := ogreF2Save(t, f, app)
	g := shopOrderFront(t)
	cold, _ := openOriginalSAVApp(t, g, current, "brian-current.sav")
	t.Cleanup(cold.StopAudio)
	loaded := brianClothedSpeaker(t, g, "F2 SAVE and cold LOAD")
	if !bytes.Equal(want.Pix, loaded.Pix) || g.live.world.Tick() != f.live.world.Tick() {
		t.Fatal("current SAV changed Brian's speaking picture or clock")
	}
	t.Logf("current SAV %x: tick %d register77=%d native latch11=%v", sha256.Sum256(current), g.live.world.Tick(), g.live.world.ScriptRegister(77), g.live.world.ScriptLatched(11))
	brianTurnDialogue(t, g, cold)
}

func brianClothedSpeaker(t *testing.T, f *FrontEnd, when string) *image.RGBA {
	t.Helper()
	live := f.live
	id := releaseRosterNPC(t, live.mission.state, 25)
	member := live.mission.state.Start.Roster[id]
	e, exists := live.world.Entity(id)
	slots, equipped := live.world.Equipped(id)
	fig := rosterFigureID(member, live.world, id)
	if !exists || !e.Alive() || !equipped || !fig.Hero || !data.FigureIsHero(e.TypeID) || member.CompanionNPC != 25 {
		t.Fatal(when, "Brian is not a living dressed Hero", e, slots, fig)
	}
	n := 0
	for _, code := range slots {
		if code != 0 {
			n++
		}
	}
	if n != 7 {
		t.Fatal(when, "Brian lost his seven equipped pieces", slots)
	}
	want, _ := composeUnitFigure(f.Archives.Containers, equipmentFromSlots(slots), fig)
	bare, _ := composeUnitFigure(f.Archives.Containers, data.Equipment{}, fig)
	if want == nil || bare == nil || differingPixels(want, bare) < 100 {
		t.Fatal(when, "clothing does not distinguish the source actor")
	}
	cast := speakerCast{actors: live.speakerActors, alive: live.entityAlive, worn: live.equipmentOf}
	if dir, ok := live.playerFigureDir(); ok {
		cast.playerDir, cast.hasPlayer = dir, true
	}
	a, present := cast.resolve(live.npcFaces[25])
	for _, pe := range placedEntities(live.mission.state.Map, live.world.Entities(), live.mission.state.Start.Roster, live.mission.state.savedDocument) {
		if pe.id == id {
			t.Logf("%s: Brian id%d TypeID%d ALM pairing%d worn%x", when, id, e.TypeID, pe.index, slots)
		}
	}
	pic, _, got := live.SpeakerFace(25)
	if !got || pic == nil || !present || a.id != id || !a.hero || pic.Bounds() != want.Bounds() || !bytes.Equal(pic.Pix, want.Pix) {
		t.Fatal(when, "loaded Brian's speaking image differs from his own dressed drawable")
	}
	t.Logf("%s: live speaker%d hero=%v, %d pixels distinguish his clothing from the bare figure", when, a.id, a.hero, differingPixels(pic, bare))
	return pic
}

func brianTurnDialogue(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	live := f.live
	payload, exists := ReadEventText(f.Archives.Containers, 40, 7)
	if !exists {
		t.Fatal("mission 40 has no turnpoint text")
	}
	if speaker, named := EventPartSpeaker(payload, 1, live.mission.audience); !named || speaker != 25 {
		t.Fatal("turnpoint message does not name Brian", speaker, named)
	}
	before := live.world.Hash()
	if !live.openDialogue(7) || !bytes.Equal(live.mission.payload, payload) || live.mission.part != 1 {
		t.Fatal("production dialogue handler did not show installed event07")
	}
	if live.world.Hash() != before {
		t.Fatal("opening the speech window changed saved gameplay state")
	}
	want := brianClothedSpeaker(t, f, "installed turnpoint message window")
	portrait, pic := live.view.NoticeSpeaker()
	if !portrait || pic == nil || pic.Bounds() != want.Bounds() || !bytes.Equal(pic.Pix, want.Pix) {
		t.Fatal("turnpoint message window lost Brian's worn set")
	}
	brianPortraitProof(t, f, pic)
	message, _, _ := f.LiveNotice()
	t.Logf("production event07 window at tick%d: %q", live.world.Tick(), message)
	if err := app.HeadlessKey("enter"); err != nil || app.Screen() != ui.ScreenMap || app.HeadlessNoticeOpen() {
		t.Fatal("Enter did not close the real turnpoint window", err, app.Screen())
	}
	start := live.world.Tick()
	for range 8 {
		ogreStep(t, f, app)
	}
	if live.world.Tick() <= start {
		t.Fatal("App did not resume gameplay after the turnpoint window")
	}
	brianClothedSpeaker(t, f, "following ordinary App ticks")
}

func brianPortraitProof(t *testing.T, f *FrontEnd, pic *image.RGBA) {
	t.Helper()
	dir := os.Getenv("AGAINROM_BRIAN_REVIEW_DIR")
	if dir == "" {
		return
	}
	root := reviewDirRoot("hotfix-brian-speech")
	rel, err := filepath.Rel(root, dir)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatal("Brian portrait output is outside its review directory", dir, err)
	}
	locale := filepath.Base(os.Getenv("AGAINROM_ASSETS"))
	if err := os.MkdirAll(filepath.Join(dir, locale), 0700); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(filepath.Join(dir, locale, "turnpoint-brian.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(out, pic); err != nil {
		out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}
