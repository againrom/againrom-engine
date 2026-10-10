package game

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// secondGeneratorClick presses and releases the pointer at p and composes
// the frame after each edge.
func secondGeneratorClick(t *testing.T, app *ui.App, p image.Point) {
	t.Helper()
	for _, action := range []string{"hover", "press", "release"} {
		if err := app.HeadlessPointer(action, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		if _, _, err := app.HeadlessFrame(); err != nil {
			t.Fatalf("%s at %v: %v", action, p, err)
		}
	}
}

// secondGeneratorStage is the generator page showing, or "" off it.
func secondGeneratorStage(app *ui.App) string {
	s, _ := app.HeadlessChargenState()
	return s.Stage
}

// The second game's New Game through the generator on the installed art:
// pre-create opens on the first picture at normal difficulty with the enter
// line's name (R2-ENGINE-285); a picture press names it and a level press
// sets the difficulty; OK opens the detail page on the picture's template
// row (R2-ASSET-076, R2-ENGINE-287) with the template skill lit
// (R2-ENGINE-288); a skill press and two attribute steps change the draft;
// Accept commits the hero (R2-ENGINE-290), slots 776 and 781, the
// difficulty and gold 1000 (R2-ENGINE-283, R2-SESSION-131) and shows town 1
// (R2-SESSION-077). A SAV of that town carries them, and a cold LOAD of it
// restores the same town.
func TestReleaseSecondGameGeneratorStartsTheFirstTown(t *testing.T) {
	f := secondGameFront(t)
	if f.ChargenAssets == nil {
		t.Fatal("the second game's install loaded no generator")
	}
	f.Options = OptionsStore{}
	out := secondMissionSaveDirectory(t)
	app := f.App("second generator")
	app.Layout(640, 480)
	var c *ui.Chargen
	app.SetNewGameChargen(func() *ui.ChargenEntry {
		c = ui.NewChargen(f.ChargenSetup())
		return &ui.ChargenEntry{Model: c, Begin: f.NewGameBegin(f.Base().Profile.Mission())}
	})
	if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenChargen || c == nil {
		t.Fatalf("NEW GAME: screen %s, error %v", app.Screen(), err)
	}
	names := f.ChargenAssets.HeroNames
	if s, _ := app.HeadlessChargenState(); s.Stage != ui.ChargenStagePreCreate || s.Name != f.ChargenAssets.EnterName || s.Difficulty != 2 || c.PreChoice() != 0 {
		t.Fatalf("pre-create opened at %q, difficulty %d, picture %d; want %q, 2, 0", s.Name, s.Difficulty, c.PreChoice(), f.ChargenAssets.EnterName)
	}
	l := f.generator()
	pre := chargenTraceMaskPoints(t, f, l.PreCreate.Mask.Key, image.Point{}, []byte{20, 40, 60, 80, 100, 120, 140, 160, 180})
	// The female mage's picture names npcnames line 26; the third level is
	// hard.
	secondGeneratorClick(t, app, pre[120])
	secondGeneratorClick(t, app, pre[60])
	if s, _ := app.HeadlessChargenState(); s.Name != names[2] || s.Difficulty != 3 || c.PreChoice() != 2 {
		t.Fatalf("after the picture and level: %q, difficulty %d, picture %d; want %q, 3, 2", s.Name, s.Difficulty, c.PreChoice(), names[2])
	}
	secondGeneratorClick(t, app, pre[180])
	if secondGeneratorStage(app) != ui.ChargenStageDetailed {
		t.Fatal("OK did not open the detail page")
	}
	template, _, ok := generatorTemplate(f.Table.Humans, "Start_FM")
	if !ok {
		t.Fatal("no Start_FM row")
	}
	res, ok := c.Result()
	want := []int{int(template.Body), int(template.Reaction), int(template.Mind), int(template.Spirit)}
	if !ok || len(res.Stats) != 4 || res.Stats[0] != want[0] || res.Stats[1] != want[1] || res.Stats[2] != want[2] || res.Stats[3] != want[3] || c.Remaining() != 0 {
		t.Fatalf("detail opened at %v, %d free; want the template %v and 0 free", res.Stats, c.Remaining(), want)
	}
	if res.Choices[chargenChoiceSkill] != 0 {
		t.Fatalf("detail lit skill %d; want the template skill 0", res.Choices[chargenChoiceSkill])
	}
	// Water, then Spirit lowered and Mind raised.
	mage := chargenTraceMaskPoints(t, f, l.Detail.Classes[1].Mask.Key, image.Pt(160, 0), []byte{102})
	_, _, lower, _, _ := ui.DetailedAttributeBoxes(l, 3)
	_, _, _, raise, _ := ui.DetailedAttributeBoxes(l, 2)
	draft := func() {
		secondGeneratorClick(t, app, mage[102])
		secondGeneratorClick(t, app, lower.Min.Add(lower.Size().Div(2)))
		secondGeneratorClick(t, app, raise.Min.Add(raise.Size().Div(2)))
	}
	draft()
	// Reset: four 25s and 100 free with the skill kept on EN; the template
	// and its skill on RU (R2-ENGINE-286).
	reset := l.Detail.Commands[1].Rect.Rectangle()
	secondGeneratorClick(t, app, reset.Min.Add(reset.Size().Div(2)))
	res, _ = c.Result()
	wantReset, wantSkill, wantFree := []int{25, 25, 25, 25}, 1, 100
	if f.Base().Profile.Language == "russian" {
		wantReset, wantSkill, wantFree = want, 0, 0
	}
	if !slices.Equal(res.Stats, wantReset) || res.Choices[chargenChoiceSkill] != wantSkill || c.Remaining() != wantFree {
		t.Fatalf("Reset gave %v, skill %d, %d free; want %v, %d, %d", res.Stats, res.Choices[chargenChoiceSkill], c.Remaining(), wantReset, wantSkill, wantFree)
	}
	if f.Base().Profile.Language == "russian" {
		draft()
	} else {
		// Back to the template through pre-create, then the same draft.
		back := l.Detail.Commands[2].Rect.Rectangle()
		secondGeneratorClick(t, app, back.Min.Add(back.Size().Div(2)))
		secondGeneratorClick(t, app, pre[120])
		secondGeneratorClick(t, app, pre[180])
		if secondGeneratorStage(app) != ui.ChargenStageDetailed || c.Difficulty() != 3 {
			t.Fatalf("Back and OK: stage %s, difficulty %d", secondGeneratorStage(app), c.Difficulty())
		}
		draft()
	}
	res, ok = c.Result()
	if !ok || res.Stats[3] != want[3]-1 || res.Stats[2] != want[2]+1 || res.Choices[chargenChoiceSkill] != 1 {
		t.Fatalf("draft %v, skill %d; want Mind %d, Spirit %d, skill 1", res.Stats, res.Choices[chargenChoiceSkill], want[2]+1, want[3]-1)
	}
	accept := l.Detail.Commands[0].Rect.Rectangle()
	secondGeneratorClick(t, app, accept.Min.Add(accept.Size().Div(2)))
	if app.Screen() != ui.ScreenTown || f.Town == nil || f.Town.second == nil {
		t.Fatalf("Accept showed screen %s", app.Screen())
	}
	if h := f.TownScreen().Header(); h != "ROM2 campaign: town 1" {
		t.Fatalf("Accept showed %q", h)
	}
	g := secondGeneratorCampaign
	bank := f.Town.second.bank
	if bank[g.MageSlot] != 1 || bank[g.FemaleSlot] != 1 || f.Town.Gold() != 1000 || f.Difficulty != mapload.DifficultyHard {
		t.Fatalf("commit: slot %d=%d, slot %d=%d, gold %d, difficulty %d", g.MageSlot, bank[g.MageSlot], g.FemaleSlot, bank[g.FemaleSlot], f.Town.Gold(), f.Difficulty)
	}
	secondGeneratorCheckHero(t, f, res, template)

	before := secondTownSampleNow(t, f, app)
	raw := secondTownNamedSave(t, f, app, out, "Generated hero")
	secondTownShape(t, raw)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	if b := actions.Session.Second.Bank; b[g.MageSlot] != 1 || b[g.FemaleSlot] != 1 {
		t.Fatalf("SAV bank slots %d and %d", b[g.MageSlot], b[g.FemaleSlot])
	}
	if len(doc.Players) == 0 || int(doc.Players[0]) < 1 || int(doc.Players[0]) > len(doc.Objects) {
		t.Fatal("SAV has no first Player")
	}
	if money, err := savedStructureValue(&doc.Objects[doc.Players[0]-1], "Money"); err != nil || money != 1000 {
		t.Fatalf("SAV first Player money %d, %v; want 1000", money, err)
	}
	native, _ := campaignDifficulty(3)
	if doc.Head.Difficulty != uint32(native) {
		t.Fatalf("SAV difficulty %d, want %d", doc.Head.Difficulty, native)
	}
	cold, a := secondTownCold(t, out, "Generated hero.sav")
	secondTownAssertSample(t, before, secondTownSampleNow(t, cold, a))
	secondGeneratorCheckHero(t, cold, res, template)
	if _, err := os.Stat(filepath.Join(out, "Generated hero.sav")); err != nil {
		t.Fatal(err)
	}
}

// secondGeneratorCheckHero checks the committed hero against the result and
// the producer's rules (R2-ENGINE-290): the result's name and statistics,
// the template's spell book, the water skill at 20 and the fifth at 10, and
// the female mage's staff.
func secondGeneratorCheckHero(t *testing.T, f *FrontEnd, res ui.ChargenResult, template data.HumanDef) {
	t.Helper()
	if len(f.Carried) != 1 {
		t.Fatalf("party of %d", len(f.Carried))
	}
	p := f.Carried[0]
	h := p.Hero
	if p.Name != res.Name || int(h.Body) != res.Stats[0] || int(h.Reaction) != res.Stats[1] || int(h.Mind) != res.Stats[2] || int(h.Spirit) != res.Stats[3] {
		t.Fatalf("hero %q %d/%d/%d/%d; want %q %v", p.Name, h.Body, h.Reaction, h.Mind, h.Spirit, res.Name, res.Stats)
	}
	if want := [data.SkillSlots]int32{0, 0, 20, 0, 0, 10}; h.Skill != want {
		t.Fatalf("skills %v, want %v", h.Skill, want)
	}
	if !p.Mage || p.KnownSpells != template.KnownSpells {
		t.Fatalf("mage %v, spells %d; want a mage with %d", p.Mage, p.KnownSpells, template.KnownSpells)
	}
	if p.Weapon == nil || p.Weapon.Name != "Wood Staff" {
		t.Fatalf("weapon %+v, want the Wood Staff", p.Weapon)
	}
	if slot, ok := data.EquipSlotFor(p.Weapon.Code); !ok || p.Worn[slot-1] != uint16(p.Weapon.Code) {
		t.Fatalf("worn %v, want the staff %d worn", p.Worn, p.Weapon.Code)
	}
}
