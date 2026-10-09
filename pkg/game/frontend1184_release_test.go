package game

import (
	"bytes"
	"image"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseFrontendFidelity1184(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.SetDeterministicFrames(true)
	f.Cutscenes = OpenCutscenes(f.Archives.Root, "video4")
	t.Run("generator", func(t *testing.T) {
		setup := f.ChargenSetup()
		if len(setup.PreCreate.Art.Sparkles) != 15 {
			t.Fatal("installed sparkle sheet absent")
		}
		for i := 0; i < 4; i++ {
			c := ui.NewChargen(setup)
			c.SelectPreChoice(i)
			c.Forward()
			r, ok := c.Result()
			d, _, found := data.ChargenBase(f.Table.Humans, i%2 == 1, i >= 2)
			want := []int{int(d.Body), int(d.Reaction), int(d.Mind), int(d.Spirit)}
			if !ok || !found || !reflect.DeepEqual(r.Stats, want) || c.Remaining() != 0 {
				t.Fatal(i, r.Stats, want, c.Remaining())
			}
			c.AdjustStat(0, -1)
			c.Reset()
			r, _ = c.Result()
			if !reflect.DeepEqual(r.Stats, want) {
				t.Fatal("reset lost preset", i)
			}
			t.Logf("portrait %d stats=%v remaining=%d", i, r.Stats, c.Remaining())
		}
		a := f.App("generator fidelity")
		a.SetNewGameChargen(func() *ui.ChargenEntry { return &ui.ChargenEntry{Model: ui.NewChargen(f.ChargenSetup())} })
		a.SetCutscenes(nil)
		a.Layout(640, 480)
		if err := a.HeadlessActivate("new game"); err != nil {
			t.Fatal(err)
		}
		before, _, err := a.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 20; i++ {
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		after, _, err := a.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(before.Pix, after.Pix) {
			t.Fatal("production generator never animates")
		}
		// The Back control lies outside the open tip panel's rect.
		back := image.Pt(-1, -1)
		probe := ui.NewChargen(f.ChargenSetup())
		for p := image.Pt(0, 0); back.X < 0 && p.Y < 480; p.Y++ {
			for p.X = 0; p.X < 640; p.X++ {
				if owner, ok := ui.PreCreateControlAt(probe, p); ok && owner == "back" && !probe.TipPanel().Covers(p) {
					back = p
					break
				}
			}
		}
		if back.X < 0 {
			t.Fatal("no pixel answers for Back")
		}
		if err := a.HeadlessPointer("move", back.X, back.Y); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 40; i++ {
			a.HeadlessStep()
		}
		if state, _ := a.HeadlessTooltip(); !state.Visible {
			t.Fatal("tips suppress generator hover", state)
		}
		// The scenario driver must refund the full preset before buying a
		// different legal spread, using the real keyboard controls.
		state, ok := a.HeadlessChargenState()
		if !ok {
			t.Fatal("no generator")
		}
		if err := headlessChargenPress(a, &state, ui.ChargenControlForward, ""); err != nil {
			t.Fatal(err)
		}
		if err := headlessChargenStats(a, &state, map[string]int{"Body": 25, "Reaction": 25, "Mind": 35, "Spirit": 35}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("initial spell caches and female voice", func(t *testing.T) {
		party := f.ChargenParty(ui.ChargenResult{Name: "Range witness", Choices: []int{1, 1, 4}, Stats: []int{19, 23, 30, 42}})
		party[0].KnownSpells = 0x1ffffffe
		a := f.App("initial spell caches")
		a.SetCutscenes(nil)
		if err := a.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
			t.Fatal(err)
		}
		var hero sim.Entity
		for _, e := range f.live.world.Entities() {
			if e.Owner == sim.SelfSlot && e.KnownSpells == 0x1ffffffe {
				hero = e
				break
			}
		}
		voice := f.live.voiceBank(hero)
		t.Logf("hero ID%d type%#x class%d figure=%v voice %q", hero.ID, hero.TypeID, hero.Class, f.live.figures[hero.ID], voice)
		if voice != "f_mage" {
			t.Fatal("female mage takes another voice bank", voice)
		}
		for _, leaf := range []string{"easy", "hard", "die"} {
			if _, ok := f.SoundBank.VoiceSample(voice + "/" + leaf + ".wav"); !ok {
				t.Fatal("female mage voice does not decode", leaf)
			}
		}
		legacy := hero
		legacy.Book = sim.Spellbook{}
		ranges := heroSpellRanges(f, hero)
		for id, want := range heroSpellRanges(f, legacy) {
			if ranges[id] != want {
				t.Fatalf("initial spell%d range=%d want%d", id+1, ranges[id], want)
			}
		}
		if ranges[24] != 5 {
			t.Fatal("Teleport range", ranges[24])
		}
		store, name, raw := menuSAVE(t, f, a, OriginalStore{})
		file, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		characters, err := file.Party()
		if err != nil {
			t.Fatal(err)
		}
		var saved []sav.SavedSpell
		for _, c := range characters {
			if c.Name == "Range witness" && c.KnownSpells() == 0x1ffffffe {
				saved = c.Spells
			}
		}
		if len(saved) != 28 {
			t.Fatalf("SAV book holds %d spells", len(saved))
		}
		for _, spell := range saved {
			if int(spell.Range) != ranges[spell.ID-1] {
				t.Fatalf("SAV spell%d range=%d live%d", spell.ID, spell.Range, ranges[spell.ID-1])
			}
		}
		cold := releaseFront(t)
		cold.SetDeterministicFrames(true)
		ca := cold.App("cold ranges")
		ca.SetCutscenes(nil)
		ca.Layout(1024, 768)
		save, list, load := cold.SaveSeams(store, OriginalStore{}, nil)
		ca.SetSaveSeams(save, list, load)
		groundAppLoad(t, ca, list, localOriginalSaveToken(name))
		for tick := range 32 {
			loaded := rangeWitnessHero(t, cold)
			if got := heroSpellRanges(cold, loaded); got != ranges {
				t.Fatalf("tick %d: cold ranges %v, live %v", tick, got, ranges)
			}
			if got := cold.live.voiceBank(loaded); got != voice {
				t.Fatal("cold load lost female identity", tick, got, voice)
			}
			live := rangeWitnessHero(t, f)
			if loaded.Mana != live.Mana || loaded.HP != live.HP || loaded.X != live.X || loaded.Y != live.Y {
				t.Fatalf("tick %d: cold hero %d/%d at %d,%d, live %d/%d at %d,%d", tick, loaded.Mana, loaded.HP, loaded.X, loaded.Y, live.Mana, live.HP, live.X, live.Y)
			}
			f.live.tick()
			cold.live.tick()
		}
		var teleport sav.SavedSpell
		for _, spell := range saved {
			if spell.ID == 25 {
				teleport = spell
			}
		}
		lossy := loadAlteredSAV(t, raw, func(doc *sav.DocumentData) bool { return setSavedSpellRange(doc, teleport.Key, 9) })
		if got := heroSpellRanges(lossy, rangeWitnessHero(t, lossy)); got[24] == ranges[24] {
			t.Fatal("altered SAV Teleport range did not load", got[24])
		}
		t.Logf("28 initial ranges; Teleport=%d; female=%v; SAV cold resume through the load window retains both for 32 ticks", ranges[24], voice)
	})
	t.Run("encountered movies and credits", func(t *testing.T) {
		catalog := f.movieCatalog()
		if len(catalog) != 14 || catalog[0].Directory != "intro" {
			t.Fatal("installed media catalog", len(catalog))
		}
		roll := f.creditsView()
		t.Logf("credits lines=%d logos=%v", len(roll.Lines), roll.Logos["nival"] != nil)
		if len(roll.Lines) < 150 || roll.Logos["nival"] == nil {
			t.Fatal("missing installed credits")
		}
		a := f.App("media fidelity")
		a.Layout(640, 480)
		if err := a.HeadlessActivate("cutscenes"); err != nil {
			t.Fatal(err)
		}
		if len(a.HeadlessRows()) != 0 {
			t.Fatal("unseen movies unlocked")
		}
		a.HeadlessKey("escape")
		if !a.PlayCutscene("newgame/01.smk") {
			t.Fatal(a.CutsceneError())
		}
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if err := a.HeadlessCutsceneStep(""); err != nil {
				t.Fatal(err)
			}
			seen, _ := f.Options.EncounteredCutscenes()
			if len(seen) > 0 {
				break
			}
			time.Sleep(time.Millisecond)
		}
		seen, err := f.Options.EncounteredCutscenes()
		if err != nil || !reflect.DeepEqual(seen, []string{"newgame"}) {
			t.Fatal("playback failed to persist encounter", seen, err)
		}
		a.StopCutscene()
		// A new App/profile read must retain exactly that encounter.
		cold := f.App("cold media")
		cold.Layout(640, 480)
		if err := cold.HeadlessActivate("cutscenes"); err != nil {
			t.Fatal(err)
		}
		rows := cold.HeadlessRows()
		if len(rows) != 1 || rows[0].Text != catalog[1].Title {
			t.Fatal(rows)
		}
		if _, _, err := cold.HeadlessFrame(); err != nil {
			t.Fatal(err)
		}
		if err := cold.HeadlessActivate(rows[0].Text); err != nil {
			t.Fatal(err)
		}
		if cold.CutsceneName() != "newgame/01.smk" {
			t.Fatal("replay missing")
		}
		cold.HeadlessCutsceneStep("key")
		cold.HeadlessStep()
		cold.HeadlessKey("escape")
		if err := cold.HeadlessActivate("credits"); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 720; i++ {
			cold.HeadlessStep()
		}
		p, _, err := cold.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		painted := 0
		for i := 0; i < len(p.Pix); i += 4 {
			if p.Pix[i] != 0 || p.Pix[i+1] != 0 || p.Pix[i+2] != 0 {
				painted++
			}
		}
		if painted < 500 {
			t.Fatal("credits stayed black", painted)
		}
		cold.HeadlessKey("escape")
		if cold.Screen() != ui.ScreenMenu {
			t.Fatal("credits cannot close")
		}
		t.Logf("catalog=%d credits=%d rows, Nival %v, pixels=%d", len(catalog), len(roll.Lines), roll.Logos["nival"].Bounds(), painted)
	})
}

func rangeWitnessHero(t *testing.T, f *FrontEnd) sim.Entity {
	t.Helper()
	for _, e := range f.live.world.Entities() {
		if e.Owner == sim.SelfSlot && e.KnownSpells == 0x1ffffffe {
			return e
		}
	}
	t.Fatal("range witness absent")
	return sim.Entity{}
}

func heroSpellRanges(f *FrontEnd, hero sim.Entity) [28]int {
	var out [28]int
	for _, rule := range f.live.world.Spells() {
		if rule.ID >= 1 && rule.ID <= 28 {
			out[rule.ID-1] = int(sim.SpellCharacteristicsFor(sim.Rules{}, hero, rule).Range)
		}
	}
	return out
}
