package ui

import (
	"image"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/render/menu"
)

// latchSite is one push button behind the kit press latch, opened on its own
// screen through the production step. count reports how many times the
// button has activated; inside is a point on the button and elsewhere a
// point on the same screen off it that activates nothing on a press: another
// button of the same latch where the screen has one.
//
// leave and back are set where a key can leave the screen while a press is
// held: leave takes the key, back returns to the screen, and other is a
// button there whose activation otherDone reports.
type latchSite struct {
	app       *App
	inside    image.Point
	elsewhere image.Point
	count     func() int
	leave     func(t *testing.T)
	back      func(t *testing.T)
	other     image.Point
	otherDone func() bool
}

func (s latchSite) pointer(t *testing.T, action string, p image.Point) {
	t.Helper()
	if err := s.app.HeadlessPointer(action, p.X, p.Y); err != nil {
		t.Fatal(err)
	}
}

func (s latchSite) key(t *testing.T, name string) {
	t.Helper()
	if err := s.app.HeadlessKey(name); err != nil {
		t.Fatal(err)
	}
}

func (s latchSite) action(t *testing.T, name string) {
	t.Helper()
	if err := s.app.HeadlessGameMenuAction(name); err != nil {
		t.Fatal(err)
	}
}

func latchMenuApp(t *testing.T) *App {
	t.Helper()
	a := NewApp("latch", appAssets(t), appRows(1), nil)
	a.Layout(640, 480)
	a.flow.viewer, a.flow.screen = fiViewer(t), ScreenMap
	a.flow.menuFont = gameMenuTestFont()
	volumes := audio.ChannelVolumes{50, 50, 50}
	a.SetGameMenuSettings(nil, nil,
		func() (bool, int, bool) { return true, 100, true },
		func(bool, int) error { return nil })
	a.SetSoundOptionControls(SoundOptionControls{Read: func() audio.ChannelVolumes { return volumes },
		Write: func(channel audio.Channel, value int) error { volumes[channel] = value; return nil },
		Words: DefaultSoundOptionWords()})
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	return a
}

func boolCount(b bool) int {
	if b {
		return 1
	}
	return 0
}

// latchSites is every push button whose screen kept its own press latch
// before the kit latch replaced it.
var latchSites = []struct {
	name string
	open func(t *testing.T) latchSite
}{
	{"main menu", func(t *testing.T) latchSite {
		a := newTestApp(t, appRows(3), okLoader(t))
		x, y := centreOf(menu.NewGameButton)
		ox, oy := centreOf(menu.LoadGameButton)
		s := latchSite{app: a, inside: image.Pt(x, y), elsewhere: image.Pt(ox, oy),
			count: func() int { return boolCount(a.Screen() == ScreenPicker) },
			other: image.Pt(ox, oy), otherDone: func() bool { return a.Screen() == ScreenLoad }}
		s.leave = func(t *testing.T) { s.key(t, "load") }
		s.back = func(t *testing.T) { s.key(t, "escape") }
		return s
	}},
	{"mod main menu entry", func(t *testing.T) latchSite {
		a := modTestApp(t, modTestScreens())
		s := latchSite{app: a, inside: midOf(modMenuEntryRects(2)[1]), elsewhere: midOf(modMenuEntryRects(2)[0]),
			count: func() int { return boolCount(a.Screen() == ScreenMod) },
			other: midOf(modMenuEntryRects(2)[0]), otherDone: func() bool { return a.Screen() == ScreenMod && a.flow.modUI.open == 0 }}
		s.leave = func(t *testing.T) { s.key(t, "load") }
		s.back = func(t *testing.T) { s.key(t, "escape") }
		return s
	}},
	{"mod screen back", func(t *testing.T) latchSite {
		a := modTestApp(t, modTestScreens())
		click(a, midOf(modMenuEntryRects(2)[1]))
		if a.Screen() != ScreenMod {
			t.Fatalf("fixture: the entry opened %v", a.Screen())
		}
		s := latchSite{app: a, inside: midOf(modBackButton), elsewhere: image.Pt(600, 460),
			count: func() int { return boolCount(a.Screen() == ScreenMenu) },
			other: midOf(modBackButton), otherDone: func() bool { return a.Screen() == ScreenMenu }}
		s.leave = func(t *testing.T) { s.key(t, "escape") }
		s.back = func(t *testing.T) { click(a, midOf(modMenuEntryRects(2)[1])) }
		return s
	}},
	{"quest objectives close", func(t *testing.T) latchSite {
		a := latchMenuApp(t)
		if err := a.HeadlessGameMenuAction("objectives"); err != nil {
			t.Fatal(err)
		}
		s := latchSite{app: a, inside: midOf(questButtonRect), elsewhere: image.Pt(questButtonRect.Max.X+4, questButtonRect.Min.Y+4),
			count: func() int { return boolCount(a.flow.menuPage == gameMenuRoot) },
			other: midOf(questButtonRect), otherDone: func() bool { return a.flow.menuPage == gameMenuRoot }}
		s.leave = func(t *testing.T) { s.key(t, "escape") }
		s.back = func(t *testing.T) { s.action(t, "objectives") }
		return s
	}},
	{"sound options OK", func(t *testing.T) latchSite {
		a := latchMenuApp(t)
		if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
			t.Fatal(err)
		}
		ok := soundOptionRect(gameMenuPageReturn)
		s := latchSite{app: a, inside: midOf(ok), elsewhere: image.Pt(ok.Max.X+4, ok.Min.Y+4),
			count: func() int { return boolCount(a.flow.menuPage == gameMenuRoot) },
			other: midOf(ok), otherDone: func() bool { return a.flow.menuPage == gameMenuRoot }}
		s.leave = func(t *testing.T) { s.key(t, "escape") }
		s.back = func(t *testing.T) { s.action(t, "sound-options") }
		return s
	}},
	{"save dialog cancel", func(t *testing.T) latchSite {
		a := newSaveDialogApp(t, &saveDialogSpy{}, ScreenTown)
		s := latchSite{app: a, inside: midOf(saveControlRect(saveCancelControl)), elsewhere: midOf(saveControlRect(saveWriteControl)),
			count: func() int { return boolCount(a.Screen() != ScreenSave) },
			other: midOf(saveControlRect(saveCancelControl)), otherDone: func() bool { return a.Screen() != ScreenSave }}
		s.leave = func(t *testing.T) { s.key(t, "escape") }
		s.back = func(t *testing.T) { s.action(t, "save") }
		return s
	}},
	{"gold editor action", func(t *testing.T) latchSite {
		a, v, g := openPurseEditor(t)
		g.key(appInput{Delete: true})
		typeGold(g, "5")
		ax, ay, err := a.HeadlessGoldControlPoint("action")
		if err != nil {
			t.Fatal(err)
		}
		ex, ey, err := a.HeadlessGoldControlPoint("edit")
		if err != nil {
			t.Fatal(err)
		}
		drops := 0
		s := latchSite{app: a, inside: image.Pt(ax, ay), elsewhere: image.Pt(ex, ey),
			count: func() int {
				if _, ok := v.TakeGoldDrop(); ok {
					drops++
				}
				return drops
			}}
		s.otherDone = func() bool { return s.count() == 1 }
		s.other = s.inside
		s.leave = func(t *testing.T) { s.key(t, "escape") }
		s.back = func(t *testing.T) {
			cx, cy := purseCenter(t, v)
			g.doubleClick(cx, cy)
			if !v.goldModalOpen() {
				t.Fatal("the editor did not reopen")
			}
			g.key(appInput{Delete: true})
			typeGold(g, "5")
		}
		return s
	}},
	{"documents right arrow", func(t *testing.T) latchSite {
		a := newTestApp(t, appRows(1), okLoader(t))
		a.flow.docSrc = &fakeDocumentSource{pages: []DocumentPage{{Text: "one"}, {Text: "two"}, {Text: "three"}}}
		a.flow.docArt = documentsTestArt()
		if !a.flow.openDocuments(ScreenMenu) {
			t.Fatal("fixture: the documents did not open")
		}
		s := latchSite{app: a, inside: midOf(docControlRect(docControlRight)), elsewhere: midOf(docControlRect(docControlLeft)),
			count: func() int { return a.flow.docPanel.doc },
			other: midOf(docControlRect(docControlRight)), otherDone: func() bool { return a.flow.docPanel != nil && a.flow.docPanel.doc == 1 }}
		s.leave = func(t *testing.T) { s.key(t, "escape") }
		s.back = func(t *testing.T) {
			if !a.flow.openDocuments(ScreenMenu) {
				t.Fatal("the documents did not reopen")
			}
		}
		return s
	}},
	{"ending back", func(t *testing.T) latchSite {
		a := endingTestApp(t)
		return latchSite{app: a, inside: midOf(a.flow.endingButtonRect(0)), elsewhere: midOf(a.flow.endingButtonRect(1)),
			count: func() int { return boolCount(a.flow.endingPage != 1) }}
	}},
	{"character generation name", func(t *testing.T) latchSite {
		setup := chargenLegalSetup()
		art := &ChargenPresentation{Layout: testGenerator(), Forward: image.NewRGBA(image.Rect(0, 0, 96, 74))}
		for choice := range art.Choices {
			for state := range art.Choices[choice] {
				art.Choices[choice][state] = image.NewRGBA(image.Rect(0, 0, 160, 240))
			}
		}
		setup.PreCreate = &ChargenPreCreate{Art: art}
		a := newTestApp(t, appRows(1), okLoader(t))
		c := NewChargen(setup)
		if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
			t.Fatal(err)
		}
		c.Move(1)
		if c.Focus() == 0 {
			t.Fatal("fixture: the focus did not leave the name")
		}
		s := latchSite{app: a, inside: midOf(preControlRect(c, chargenName)), elsewhere: image.Pt(10, 10),
			count: func() int { return boolCount(c.Focus() == 0) },
			other: midOf(preControlRect(c, chargenName)), otherDone: func() bool { return c.Focus() == 0 }}
		s.leave = func(t *testing.T) { s.key(t, "escape") }
		s.back = func(t *testing.T) {
			if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
				t.Fatal(err)
			}
			c.Move(1)
		}
		return s
	}},
	{"character generation play", func(t *testing.T) latchSite {
		setup := chargenLegalSetup()
		setup.Name = ""
		setup.PreCreate = &ChargenPreCreate{Art: &ChargenPresentation{Layout: testGenerator()}}
		setup.Detailed = &ChargenDetailed{EmptyName: "EMPTY", ReservedName: "RESERVED", Play: "PLAY"}
		c := NewChargen(setup)
		c.Forward()
		a := newTestApp(t, appRows(1), okLoader(t))
		if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return okOpener(t), nil }); err != nil {
			t.Fatal(err)
		}
		s := latchSite{app: a, inside: midOf(detailedControlRect(c, chargenPlay)), elsewhere: image.Pt(10, 10),
			count: func() int { return boolCount(a.flow.msg == "EMPTY") },
			other: midOf(detailedControlRect(c, chargenPlay)), otherDone: func() bool { return a.flow.msg == "EMPTY" }}
		s.leave = func(t *testing.T) { s.key(t, "escape") }
		s.back = func(t *testing.T) {
			if c.Stage() != PreCreateStage {
				t.Fatalf("fixture: Escape left stage %v", c.Stage())
			}
			c.Forward()
		}
		return s
	}},
	{"town dialogue advance", func(t *testing.T) latchSite {
		f := newDialoguePointerFixture(t, "town")
		return latchSite{app: f.app, inside: midOf(f.button), elsewhere: f.body, count: f.advances}
	}},
	{"mission dialogue advance", func(t *testing.T) latchSite {
		f := newDialoguePointerFixture(t, "mission")
		return latchSite{app: f.app, inside: midOf(f.button), elsewhere: f.body, count: f.advances}
	}},
}

func TestEveryFormerLocalLatchRunsTheKitCases(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T, s latchSite) (int, int)
	}{
		{"press and release inside", func(t *testing.T, s latchSite) (int, int) {
			s.pointer(t, "press", s.inside)
			s.pointer(t, "release", s.inside)
			return s.count(), 1
		}},
		{"press inside, release elsewhere", func(t *testing.T, s latchSite) (int, int) {
			s.pointer(t, "press", s.inside)
			s.pointer(t, "release", s.elsewhere)
			s.pointer(t, "release", s.inside)
			return s.count(), 0
		}},
		{"press inside, focus lost", func(t *testing.T, s latchSite) (int, int) {
			s.pointer(t, "press", s.inside)
			if err := s.app.HeadlessFocus(false); err != nil {
				t.Fatal(err)
			}
			if err := s.app.HeadlessFocus(true); err != nil {
				t.Fatal(err)
			}
			s.pointer(t, "release", s.inside)
			return s.count(), 0
		}},
		{"double press", func(t *testing.T, s latchSite) (int, int) {
			s.pointer(t, "press", s.inside)
			s.pointer(t, "press", s.elsewhere)
			s.pointer(t, "release", s.inside)
			s.pointer(t, "release", s.inside)
			return s.count(), 1
		}},
		{"press held while a key leaves the screen", func(t *testing.T, s latchSite) (int, int) {
			if s.leave == nil {
				// The dialogue takes every key as an advance and cancels the
				// press (DIALOGUE-045); the ending has no key back to the page
				// a key leaves.
				t.Skip("no key leaves this screen and returns to it while a press is held")
			}
			s.pointer(t, "press", s.inside)
			s.leave(t)
			s.pointer(t, "release", image.Pt(1, 1))
			s.back(t)
			s.pointer(t, "press", s.other)
			s.pointer(t, "release", s.other)
			return boolCount(s.otherDone()), 1
		}},
	}
	for _, site := range latchSites {
		t.Run(site.name, func(t *testing.T) {
			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					s := site.open(t)
					if got := s.count(); got != 0 {
						t.Fatalf("fixture activated %d times before any gesture", got)
					}
					if got, want := c.run(t, s); got != want {
						t.Fatalf("activated %d times, want %d", got, want)
					}
				})
			}
		})
	}
}
