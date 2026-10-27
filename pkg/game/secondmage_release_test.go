package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func secondGeneratedMage(t *testing.T, female bool) (*FrontEnd, *ui.App) {
	t.Helper()
	f := secondGameFront(t)
	f.Options = OptionsStore{}
	f.SetTipsOff(true)
	app := f.App("new mage")
	app.Layout(640, 480)
	var c *ui.Chargen
	app.SetNewGameChargen(func() *ui.ChargenEntry {
		c = ui.NewChargen(f.ChargenSetup())
		return &ui.ChargenEntry{Model: c, Begin: f.NewGameBegin(f.Base().Profile.Mission())}
	})
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	l := f.generator()
	pre := chargenTraceMaskPoints(t, f, l.PreCreate.Mask.Key, image.Point{}, []byte{120, 140, 180})
	picture := byte(140)
	if female {
		picture = 120
	}
	secondGeneratorClick(t, app, pre[picture])
	want := 3
	if female {
		want = 2
	}
	if c.PreChoice() != want {
		t.Fatalf("picture at %v chose %d, want %d", pre[picture], c.PreChoice(), want)
	}
	secondGeneratorClick(t, app, pre[180])
	accept := l.Detail.Commands[0].Rect.Rectangle()
	secondGeneratorClick(t, app, accept.Min.Add(accept.Max).Div(2))
	if app.Screen() != ui.ScreenTown || len(f.Carried) != 1 || !f.Carried[0].Mage {
		t.Fatalf("generator commit: screen=%v, no mage", app.Screen())
	}
	t.Logf("generated female=%v mask=%#x book=%v", female, f.Carried[0].KnownSpells, f.Carried[0].Book.State)
	return f, app
}

func TestReleaseSecondTownMageTalkPointer(t *testing.T) {
	for _, female := range []bool{true, false} {
		for _, composed := range []bool{true, false} {
			t.Run(fmt.Sprintf("female=%v/composed=%v", female, composed), func(t *testing.T) {
				f, app := secondGeneratedMage(t, female)
				s := f.TownScreen().(*townScreen)
				if !composed {
					s.composed = nil
				}
				for _, target := range []string{"TAVERN", "TALK 517"} {
					if err := app.HeadlessActivate(target); err != nil {
						t.Fatal(err)
					}
				}
				c := f.Town.second
				for page := 0; c.talk.payload != nil; page++ {
					if page >= 64 {
						t.Fatal("talk exceeded 64 pages")
					}
					pic, open := s.TownDialogue()
					button, shown := s.TownDialogueButton()
					if !open || !shown || pic == nil {
						t.Fatal("no talk button")
					}
					p := button.Min.Add(button.Max).Div(2).Add(image.Pt((640-pic.Bounds().Dx())/2, (480-pic.Bounds().Dy())/2))
					before := c.talk.part
					secondGeneratorClick(t, app, p)
					if c.talk.payload != nil && c.talk.part == before {
						t.Fatalf("OK pointer press/release left talk on page %d", before)
					}
				}
			})
		}
	}
}

func TestReleaseSecondGameGeneratedMageBook(t *testing.T) {
	for _, female := range []bool{true, false} {
		t.Run(fmt.Sprintf("female=%v", female), func(t *testing.T) {
			f, app := secondGeneratedMage(t, female)
			for _, target := range []string{"TAVERN", "TALK 517"} {
				if err := app.HeadlessActivate(target); err != nil {
					t.Fatal(err)
				}
			}
			for app.HeadlessActivate("notice") == nil {
			}
			for _, target := range []string{"EXIT", "GATES", "mission 10"} {
				if err := app.HeadlessActivate(target); err != nil {
					t.Fatal(err)
				}
			}
			for range 20 {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			dismissSecondMissionDialogue(t, app, f.live)
			app.Layout(1024, 768)
			if err := app.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			out := secondMissionSaveDirectory(t)
			secondMageBookVisible(t, f, app, "fresh")
			raw := secondMissionNamedSave(t, f, app, out, "Mage book")
			t.Logf("before cast world=%x sav=%x", f.live.world.Hash(), sha256.Sum256(raw))
			cold, a := secondMissionCold(t, out, secondSaveFile(t, out, "Mage book"))
			assertCurrentWorldEqual(t, f.live.world, cold.live.world, "generated mage cold LOAD")
			secondMageBookVisible(t, cold, a, "cold")
			secondMageCastFromBar(t, f, app)
			secondMageCastFromBar(t, cold, a)
			assertCurrentWorldEqual(t, f.live.world, cold.live.world, "generated mage cast after cold LOAD")
		})
	}
}

func secondMageBookVisible(t *testing.T, f *FrontEnd, app *ui.App, stage string) {
	t.Helper()
	id := f.live.mission.ids[0]
	e, _ := f.live.entity(id)
	if e.KnownSpells != 1<<24|1<<27 {
		t.Fatalf("%s hero=%d mask=%#x, want Heal and Shield", stage, id, e.KnownSpells)
	}
	spellbookShow(t, app, f.live, id)
	for _, mode := range []string{"bar", "reopened book"} {
		if mode == "reopened book" {
			for range 2 {
				if err := app.HeadlessKey("book"); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := app.HeadlessPointer("hover", -1, -1); err != nil {
			t.Fatal(err)
		}
		pix, err := app.HeadlessBottomHUD()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s %s frame=%x world=%x mask=%#x", stage, mode, sha256.Sum256(pix.Pix), f.live.world.Hash(), e.KnownSpells)
		for _, known := range []struct {
			id   uint32
			slot int
		}{{24, 4}, {27, 19}} {
			x, y, err := app.HeadlessSpellPoint(known.id)
			if err != nil {
				t.Errorf("%s %s known spell %d missing: %v", stage, mode, known.id, err)
				continue
			}
			icon := cutSpellIcon(f.live.spellAtlas(), known.slot)
			got := f.live.spellIcon(uint16(known.id))
			if icon == nil || got == nil || !bytes.Equal(icon.Pix, got.Pix) {
				t.Errorf("%s spell %d icon does not use second-game slot %d", stage, known.id, known.slot)
			}
			if icon != nil {
				py := y - (768 - pix.Bounds().Dy())
				for dy := -10; dy < 10; dy++ {
					for dx := -10; dx < 10; dx++ {
						if pix.RGBAAt(x+dx, py+dy) != icon.RGBAAt(18+dx, 18+dy) {
							t.Errorf("%s %s spell %d visible pixels differ from slot %d", stage, mode, known.id, known.slot)
							dx, dy = 10, 10
						}
					}
				}
			}
			spellbookClick(t, app, x, y)
			if _, selected, armed := f.live.view.QuickSpellState(); !armed || selected != known.id {
				t.Errorf("%s spell %d bar click armed=%v selected=%d", stage, known.id, armed, selected)
			}
			spellbookClick(t, app, x, y)
		}
	}
}

func secondMageCastFromBar(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	live := f.live
	id := live.mission.ids[0]
	for _, spell := range []uint32{27, 24} {
		secondMageCastSelf(t, live, app, id, spell)
	}
}

func secondMageCastSelf(t *testing.T, live *mapWorld, app *ui.App, id sim.EntityID, spell uint32) {
	t.Helper()
	e, _ := live.entity(id)
	x, y, err := app.HeadlessSpellPoint(spell)
	if err != nil {
		t.Fatal(err)
	}
	spellbookClick(t, app, x, y)
	inspectionCentre(live, int(e.X), int(e.Y))
	live.push()
	x, y, err = app.HeadlessEntityPoint(uint32(id))
	if err != nil {
		t.Fatal(err)
	}
	spellbookClick(t, app, x, y)
	if len(live.pending) != 1 || live.pending[0] != sim.Cast(id, id, sim.SpellID(spell)) {
		t.Fatalf("spell %d bar -> self: pending=%+v", spell, live.pending)
	}
	spent, previous := false, e.Mana
	for range 120 {
		live.tick()
		next, _ := live.entity(id)
		spent = spent || next.Mana < previous
		previous = next.Mana
	}
	after, _ := live.entity(id)
	if !spent {
		t.Fatalf("spell %d never spent mana: %d -> %d; admission=%q", spell, e.Mana, after.Mana, live.world.BookSpellRefusal(id, id, spell))
	}
	t.Logf("spell %d cast mana=%d -> %d world=%x", spell, e.Mana, after.Mana, live.world.Hash())
}
