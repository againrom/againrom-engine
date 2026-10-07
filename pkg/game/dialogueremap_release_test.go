package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"

	"againrom/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

type fixedClockDraws struct {
	townDraws
	now time.Time
}

func (d fixedClockDraws) animationClock() func() time.Time {
	return func() time.Time { return d.now }
}

func packedBackdropColour(c color.RGBA) color.RGBA {
	return color.RGBA{uint8((int(c.R) / 8 * 13 / 16) * 255 / 31), uint8((int(c.G) / 4 * 13 / 16) * 255 / 63), uint8((int(c.B) / 8 * 13 / 16) * 255 / 31), c.A}
}

func checkInstalledTownRemap(t *testing.T, a *ui.App, s *townScreen, route string) {
	t.Helper()
	now := s.townAnimationNow()
	s.draws = fixedClockDraws{townDraws: s.draws, now: now}
	if s.room != roomTalk || s.TownDialogueShows() != 1 {
		t.Fatal("installed route did not invoke one show", route, s.room, s.TownDialogueShows())
	}
	page := s.said
	checkInstalledDialoguePointer(t, a, route, image.Rect(276, 296, 356, 322), image.Pt(640, 480), func() *image.RGBA {
		t.Helper()
		out, _, err := a.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		return out
	})
	if s.said != page {
		t.Fatal("pointer cancellation advanced installed town page", route)
	}
	if err := a.SetDialogueBackdrop(ui.DialogueBackdrop{Clipped: true}); err != nil {
		t.Fatal(err)
	}
	base, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	a.SetDialogueBackdrop(ui.DialogueBackdrop{})
	first, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	modal, _ := s.TownDialogue()
	box := ui.AuthoredDialogueLayout().Box
	if modal.Bounds().Dx() != box.Dx() {
		t.Fatal("unexpected modal layout")
	}
	checked, changed := 0, 0
	for y := 0; y < 480; y += 7 {
		for x := 0; x < 640; x += 7 {
			if image.Pt(x, y).In(box) {
				continue
			}
			want := packedBackdropColour(base.RGBAAt(x, y))
			if first.RGBAAt(x, y) != want {
				t.Fatalf("%s packed room at %d,%d=%v want %v", route, x, y, first.RGBAAt(x, y), want)
			}
			checked++
			if want != base.RGBAAt(x, y) {
				changed++
			}
		}
	}
	if changed < 1000 {
		t.Fatal("omitted backdrop loss control insensitive", route, changed)
	}
	a.SetTextSmoothing(false)
	a.Draw(ebiten.NewImage(640, 480))
	for _, p := range []image.Point{{8, 8}, {620, 460}, {40, 280}} {
		got, ok := a.DialogueBackdropPixel(p.X, p.Y)
		if !ok || got != first.RGBAAt(p.X, p.Y) {
			t.Fatal("town GPU submission/log disagrees with CPU", route, p, got, ok)
		}
	}
	a.Layout(1920, 1080)
	wide, _, err := a.HeadlessFrame()
	if err != nil || !bytes.Equal(wide.Pix, first.Pix) {
		t.Fatal("wide-window placement changed native town remap", route, err)
	}
	a.Layout(640, 480)
	pages := 0
	for s.room == roomTalk && pages < 64 {
		if err := a.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
		pages++
		if s.room == roomTalk && s.TownDialogueShows() != 1 {
			t.Fatal("page invoked repeated show", route)
		}
	}
	if s.room == roomTalk || s.TownDialogueShows() != 0 {
		t.Fatal("close retained backdrop", route)
	}
	closed, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	a.SetDialogueBackdrop(ui.DialogueBackdrop{Clipped: true})
	clear, _, err := a.HeadlessFrame()
	if err != nil || !bytes.Equal(clear.Pix, closed.Pix) {
		t.Fatal("closed room has stale dim", route, err)
	}
	a.SetDialogueBackdrop(ui.DialogueBackdrop{})
	t.Logf("%s: %d background cells, %d omitted-show differences, %d App page actions; CPU/native-wide/GPU submission/close controls pass", route, checked, changed, pages)
}

func TestReleaseDialogueBackdropTownPlayerRoutes(t *testing.T) {
	f := releaseFront(t)
	a, s := openFirstTownShopDialogue(t, f, "dialogue remap town")
	t.Cleanup(a.StopAudio)
	checkInstalledTownRemap(t, a, s, "shop service")
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	offers := f.Town.Offers(TownTavern)
	if len(offers) == 0 {
		t.Fatal("no installed inn offer")
	}
	for _, target := range []string{"TAVERN", fmt.Sprintf("NPC %d", offers[0].NPC)} {
		if err := a.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	checkInstalledTownRemap(t, a, s, "inn NPC")
	f, _ = hireForOrderTest(t, "Backdrop Hero")
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatal(err)
	}
	a = openLocalTownSAV(t, f, store.Dir, name)
	a.Layout(640, 480)
	s = f.TownScreen().(*townScreen)
	t.Cleanup(a.StopAudio)
	if err := a.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	mercs := s.tavernMercenaries()
	if len(mercs) == 0 {
		t.Fatal("no installed mercenary candidate")
	}
	for _, target := range []string{fmt.Sprintf("Mercenary %d", mercs[0].Type), f.Words.TavernTalk} {
		if err := a.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	checkInstalledTownRemap(t, a, s, "mercenary Talk")
	store2 := SaveStore{Dir: t.TempDir()}
	f.ConfigureSaveSeams(a, store2, OriginalStore{}, nil)
	raw := cityRosterF2Save(t, a, store2, "Backdrop current")
	g, cold, restored := coldOfferApp(t, raw, "")
	t.Cleanup(cold.StopAudio)
	if restored.TownDialogueShows() != 0 {
		t.Fatal("cold SAV load retained presentation depth")
	}
	for _, target := range []string{"TAVERN", fmt.Sprintf("Mercenary %d", g.TownScreen().(*townScreen).tavernMercenaries()[0].Type), g.Words.TavernTalk} {
		if err := cold.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	checkInstalledTownRemap(t, cold, restored, "cold SAV next Talk")
	h := releaseFront(t)
	h.TownAnimationNow = func() time.Time { return time.Unix(100, 0) }
	school, training := roomExitApp(t, h, schoolOfferChapter(t, h))
	t.Cleanup(school.StopAudio)
	if err := school.HeadlessActivate("SCHOOL"); err != nil {
		t.Fatal(err)
	}
	checkInstalledTownRemap(t, school, training, "training service")
}

func TestReleaseDialogueBackdropMissionPlayerRoute(t *testing.T) {
	f := releaseFront(t)
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer = nil, nil, nil, nil
	f.SetDeterministicFrames(true)
	a := f.App("dialogue remap mission")
	t.Cleanup(a.StopAudio)
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	a.Layout(ui.MissionFrameW, ui.MissionFrameH)
	for ticks := 0; ticks < 300 && !f.live.view.NoticeOpen(); ticks++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	v := f.live.view
	_, kind, open := v.NoticeState()
	if !open || kind != ui.NoticeDialogue {
		t.Fatal("installed mission did not open dialogue through App")
	}
	_, depth, clip := v.DialogueBackdropPlan()
	if depth != 1 || clip != image.Rect(0, 0, ui.MissionFrameW, ui.MissionFrameH) {
		t.Fatal("mission omitted full frame show", depth, clip)
	}
	v.SetTextSmoothing(false)
	v.SetNoticeBackdrop(color.RGBA{})
	a.Draw(ebiten.NewImage(ui.MissionFrameW, ui.MissionFrameH))
	base := map[image.Point]color.RGBA{}
	for y := 16; y < ui.MissionFrameH; y += 13 {
		for x := ui.MissionFrameW - ui.MissionPanelW + 8; x < ui.MissionFrameW; x += 11 {
			p := image.Pt(x, y)
			if c, ok := a.DialogueBackdropPixel(x, y); ok {
				base[p] = c
			}
		}
	}
	v.SetDialogueBackdrop(ui.DialogueBackdrop{})
	a.Draw(ebiten.NewImage(ui.MissionFrameW, ui.MissionFrameH))
	changed := 0
	for p, c := range base {
		got, ok := a.DialogueBackdropPixel(p.X, p.Y)
		want := packedBackdropColour(c)
		if !ok || got != want {
			t.Fatal("mission HUD remap submission mismatch", p, got, ok, want)
		}
		if c != want {
			changed++
		}
	}
	if changed < 100 {
		t.Fatal("mission full-frame/right-column omission control insensitive", changed)
	}
	bodyBefore, _, _ := v.NoticeState()
	checkInstalledDialoguePointer(t, a, "mission dialogue", image.Rect(388, 440, 468, 466), image.Pt(ui.MissionFrameW, ui.MissionFrameH), nil)
	bodyAfter, _, stillOpen := v.NoticeState()
	if !stillOpen || bodyBefore != bodyAfter {
		t.Fatal("pointer cancellation advanced installed mission page")
	}
	pages := 0
	for v.NoticeOpen() && pages < 64 {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
		pages++
		_, n, _ := v.DialogueBackdropPlan()
		if v.NoticeOpen() && n != 1 {
			t.Fatal("mission App page invoked show", n)
		}
	}
	if v.NoticeOpen() {
		t.Fatal("mission dialogue did not close")
	}
	_, depth, _ = v.DialogueBackdropPlan()
	if depth != 0 {
		t.Fatal("mission close retained remap")
	}
	t.Logf("mission 10: %d known CPU submission cells, %d right-column omission differences, %d App page actions", len(base), changed, pages)
}
