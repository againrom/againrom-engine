package ui

import (
	"crypto/sha256"
	"fmt"
	"image"
	"strings"
	"testing"
	"time"
)

func placementOpsApp(t *testing.T, p GameMenuProfile, state string, size image.Point) *App {
	t.Helper()
	a := newTestApp(t, appRows(1), okLoader(t))
	a.flow = openMissionMenu(t)
	if state == "town" {
		a.flow = openTownMenu(t)
	}
	a.flow.menuFont = chargenTestFont()
	a.flow.viewer = backdropViewer(t)
	a.SetGameMenuProfile(p)
	a.Layout(size.X, size.Y)
	a.flow.setMenuUp(true)
	switch state {
	case "load":
		a.flow.setMenuUp(false)
		a.flow.openLoad(ScreenGameMenu)
	case "mod":
		if err := a.SetModScreens([]ModScreen{{Kind: ModScreenInfo, Game: true, MenuLabel: "Details", Title: "Details", Paragraphs: []string{"One page"}}}); err != nil {
			t.Fatal(err)
		}
		a.flow.openModScreen(0, ScreenGameMenu, 0)
	case "map":
		a.flow.closeGameMenu()
	}
	return a
}

func pixHash(p *image.RGBA) string {
	if p == nil {
		return "nil"
	}
	s := sha256.Sum256(p.Pix)
	return fmt.Sprintf("%v:%x", p.Bounds(), s[:6])
}

// placementAnswersDigest is the digest of the same 512 answers taken through
// the base's own branches (the window placed over the mission or not) before
// the operations replaced them; the list is in the hotfix's notes.
const placementAnswersDigest = "0d731e49412f017a08cb0140e4e25faf8ae6a09b6f78d21af1bc0598a8c6423d"

// TestPlacementOperationsAnswerAsTheBaseBranches asks each operation of both
// games' placements, at two window sizes and five menu states, and compares
// every answer with the one the base gave.
func TestPlacementOperationsAnswerAsTheBaseBranches(t *testing.T) {
	var out []string
	for _, size := range []image.Point{{1024, 768}, {1920, 1080}} {
		for _, g := range gameMenuProfiles {
			for _, state := range []string{"menu", "load", "mod", "town", "map"} {
				a := placementOpsApp(t, g.profile, state, size)
				f := a.flow
				add := func(k string, v ...any) {
					out = append(out, fmt.Sprintf("%v %s %s %s %s", size, g.game, state, k, fmt.Sprint(v...)))
				}
				pl := f.menuPlacement()
				add("missionView", pl.menuWindowHost == missionViewHost)
				add("mapShowing", f.mapShowing())
				drawn := false
				if a.place.Valid() {
					pl.underlay.drawOver(f, func() { drawn = true })
				}
				add("drawLayer", drawn)
				add("layer", f.screen != ScreenGameMenu && pl.underlay.layer(a) != nil)
				add("panel", pixHash(a.GameMenuPanel()))
				add("menuUp", f.viewer.menuUp)
				if state == "load" {
					ticks := 0
					f.tick = func() { ticks++ }
					screenHandlers[ScreenLoad].Step(a, appInput{}, time.Unix(0, 0))
					add("hold", ticks)
					pix, err := a.composePage(f.loadView())
					add("backdrop", pixHash(pix), err)
				}
				if state == "mod" {
					pix, err := a.composeModScreen()
					add("backdrop", pixHash(pix), err)
				}
				place, at := pl.transform.place(a)
				if p2, at2 := a.missionMenuPlacement(); p2 != place || at2 != at {
					t.Fatalf("%v %s %s: missionMenuPlacement %v %v, transform %v %v", size, g.game, state, p2, at2, place, at)
				}
				ox, oy := place.Origin()
				add("place", place.Scale(), " ", ox, " ", oy, " ", place.WindowSize(), " ", at)
				for _, w := range []image.Point{{0, 0}, {100, 100}, {512, 384}, {1000, 700}, {1900, 1000}} {
					p, ok := pl.transform.toFrame(a, w.X, w.Y)
					if p2, ok2 := a.windowToNativeFrame(w.X, w.Y); p2 != p || ok2 != ok {
						t.Fatalf("%v %s %s: windowToNativeFrame %v differs from transform", size, g.game, state, w)
					}
					add("toFrame", w, p, " ", ok)
					tp, tok := pl.transform.tip(a, w.X, w.Y)
					add("tip", w, tp, " ", tok)
				}
				for _, p := range []image.Point{{0, 0}, {320, 240}, {639, 479}} {
					x, y, ok := pl.transform.toWindow(a, p)
					if x2, y2, ok2 := a.nativeFrameToWindow(p); x2 != x || y2 != y || ok2 != ok {
						t.Fatalf("%v %s %s: nativeFrameToWindow %v differs from transform", size, g.game, state, p)
					}
					add("toWindow", p, " ", x, " ", y, " ", ok)
				}
				pix, note, err := a.HeadlessPicture(HeadlessRequest{Part: HeadlessPartFrame})
				add("headless", pixHash(pix), " ", note.Note, " ", err != nil)
				add("tipOnPanel", pl.tipOnPanel == tipOnWindowPanel)
				mods := append(append([]gameMenuRow(nil), f.menuRows()...), gameMenuRow{Mod: 1}, gameMenuRow{Mod: 2}, gameMenuRow{Mod: 3})
				for _, s := range []gameMenuSurface{gameMenuMission, gameMenuTown} {
					add("grown", s, " ", gameMenuPanelRectFor(s, mods, f.menuGeometryProfile()))
				}
				add("wash", f.pages.menu.placement().wash == g.profile.MissionBackdrop)
			}
		}
	}
	joined := strings.Join(out, "\n")
	if sum := fmt.Sprintf("%x", sha256.Sum256([]byte(joined))); len(out) != 512 || sum != placementAnswersDigest {
		t.Fatalf("%d answers, digest %s, want 512 and %s:\n%s", len(out), sum, placementAnswersDigest, joined)
	}
}

// TestPlacementRowsCarryTheirOperations is each game's row: the first game's
// wide frame keeps the menu art, its own mapping and the tip on the frame;
// the second game's view holds the map, no art, the centred mapping and the
// tip on the panel, and only while a hosted window stands over the mission.
func TestPlacementRowsCarryTheirOperations(t *testing.T) {
	for _, g := range gameMenuProfiles {
		second := g.profile.NativeMissionWindows
		want := wideFrameHost
		if second {
			want = menuWindowHost{centred: true, underlay: mapUnderlay, backdrop: noBackdrop, transform: missionViewTransform, tipOnPanel: tipOnWindowPanel}
		}
		if got := g.profile.placement().menuWindowHost; got != want {
			t.Fatalf("%s game row %+v, want %+v", g.game, got, want)
		}
		mission := openMissionMenu(t)
		mission.pages.menu = g.profile
		if pl := mission.menuPlacement(); pl.menuWindowHost != want {
			t.Fatalf("%s game mission menu host %+v", g.game, pl.menuWindowHost)
		}
		mission.closeGameMenu()
		if pl := mission.menuPlacement(); pl.menuWindowHost != wideFrameHost {
			t.Fatalf("%s game closed menu host %+v", g.game, pl.menuWindowHost)
		}
		town := openTownMenu(t)
		town.pages.menu = g.profile
		if pl := town.menuPlacement(); pl.menuWindowHost != wideFrameHost {
			t.Fatalf("%s game town host %+v", g.game, pl.menuWindowHost)
		}
	}
	if wideFrameHost != (menuWindowHost{underlay: noUnderlay, backdrop: menuArtBackdrop, transform: wideFrameTransform, tipOnPanel: tipOnFrame}) {
		t.Fatalf("wide frame host %+v", wideFrameHost)
	}
	for s, want := range map[Screen]Screen{ScreenLoad: ScreenMap, ScreenMod: ScreenMap, ScreenGameMenu: ScreenMap} {
		if got := mapUnderlay.shows(s); got != want {
			t.Fatalf("map underlay shows %v for %v", got, s)
		}
		if got := noUnderlay.shows(s); got != s {
			t.Fatalf("no underlay shows %v for %v", got, s)
		}
	}
}
