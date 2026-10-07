package game

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseOwnerFidelityHUDAndSavedCursor(t *testing.T) {
	f := releaseFront(t)
	output := t.TempDir()
	f.Options = OptionsStore{Path: filepath.Join(output, "options.txt")}
	var report bytes.Buffer
	if err := f.WitnessOwnerFidelity(os.Getenv("AGAINROM_ASSETS"), output, &report); err != nil {
		t.Fatal(err)
	}
	t.Log(report.String())
	t.Run("bottom-hud", func(t *testing.T) { releaseBottomHUDPixels(t, f, output) })
	t.Run("curved-card", func(t *testing.T) { releaseCurvedCharacterCard(t, f) })
	t.Run("original-selected-cursor", releaseOriginalSelectedCursor)
}

func releaseBottomHUDPixels(t *testing.T, f *FrontEnd, output string) {
	t.Helper()
	file, err := os.Open(filepath.Join(output, "bottom-hud-all-spells.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	pic, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	// Compare the actual mission composite with the installed atlas. Only
	// the final selected cell may differ (its border and quick-key numeral).
	for y := 0; y < 85; y++ {
		for x := 0; x < 464; x++ {
			if image.Pt(x, y).In(image.Rect(424, 44, 460, 80)) {
				continue
			}
			want := f.BottomHUDArt.Value().Book.RGBAAt(x, y)
			if want.A == 255 && color.RGBAModel.Convert(pic.At(192+x, y)) != want {
				t.Fatal("installed atlas was cropped, rescaled or repainted", x, y)
			}
		}
	}
	for i, source := range []int{2, 2, 2, 2, 2, 3, 4} {
		// The controlled run carries one book plus its participant purse;
		// the remaining seven bays must keep their original gold ground.
		for y := 8; y < 80; y++ {
			for x := 2; x < 78; x++ {
				want := f.BottomHUDArt.Value().Pack.RGBAAt(32+80*source+x, 6+y)
				if want.A == 255 && color.RGBAModel.Convert(pic.At(224+80*i+x, 91+y)) != want {
					t.Fatal("empty inventory bay differs from the original frame", i, x, y)
				}
			}
		}
	}
}

func releaseCurvedCharacterCard(t *testing.T, f *FrontEnd) {
	art, err := LoadTownCharacterPaneArt(f.Archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	background := art.Stats.Body.(*image.RGBA)
	untouched := bytes.Clone(background.Pix)
	l := ui.CompactPanelLayout(background)
	if l.LabelColor != (color.RGBA{189, 158, 74, 255}) || l.ValueColor != (color.RGBA{107, 154, 123, 255}) {
		t.Fatal("card palette differs from owner reference")
	}
	s := ui.PanelSubject{Name: "RENIESTA", Words: f.Words, HP: 81, MaxHP: 81, Mana: 309, MaxMana: 309,
		DetailSet: true, DetailLevel: 7, Speed: 16, WeightKnown: true, Weight: 31,
		Char: ui.UnitCharacter{Known: true, Mage: true, Body: 19, Reaction: 23, Mind: 30, Spirit: 42,
			Experience: 1296970, Sight: 6, Sight256: 1561,
			Skills: [6]int{0, 60, 72, 31, 12, 32}, Protection: [5]int{21, 21, 21, 21, 21}},
		Combat: ui.UnitCombat{Known: true, DamageBase: 9, DamageSpread: 18, ToHit: 96, Defence: 24}}
	base := s
	for _, sample := range []string{"mage", "warrior", "creature", "large pools and role"} {
		s = base
		if sample == "warrior" || sample == "creature" {
			s.Name, s.HP, s.MaxHP, s.Mana, s.MaxMana = "DANATH", 245, 245, 0, 0
			s.Speed, s.Weight = 17, 239
			s.Char = ui.UnitCharacter{Known: true, Body: 43, Reaction: 25, Mind: 26, Spirit: 26,
				Experience: 120654, Sight: 6, Skills: [6]int{0, 50, 2, 0, 0, 17},
				Protection: [5]int{13, 13, 13, 13, 13}}
			s.Combat = ui.UnitCombat{Known: true, DamageBase: 27, DamageSpread: 22, ToHit: 166, Defence: 150, Absorption: 7}
			if sample == "creature" {
				s.Char.Band = ui.CharacterBandCreature
			}
		}
		if sample == "large pools and role" {
			s.Name = strings.Repeat("RENIESTA", 10)
			s.Role = f.Words.PanelCaptions[84]
			s.HP, s.MaxHP, s.Mana, s.MaxMana = 32767, 32767, 32767, 32767
			s.Char.Experience = 2147483647
		}
		rows := ui.CharacterPanelReport(l, f.tipFont(), s)
		leftEnd, attributes, manaZero := -1, 0, false
		for _, row := range rows {
			if row.At.Y > 40 && (row.Value != row.FullValue || row.RightValue != row.FullRightValue) {
				t.Fatal("card clipped a number", sample, row)
			}
			if (sample == "warrior" || sample == "creature") && row.At.Y >= 44 && row.At.Y <= 74 {
				width, _ := f.tipFont().Measure(row.Value)
				end := row.At.X + row.ValueX + width
				if (leftEnd >= 0 && end != leftEnd) || end > l.Size.X/2+10 {
					t.Fatal("attribute left its shared left column", sample, row)
				}
				leftEnd, attributes = end, attributes+1
				manaZero = manaZero || row.RightValue == "0/0"
			}
		}
		if (sample == "warrior" || sample == "creature") && (attributes != 4 || manaZero != (sample == "warrior")) {
			t.Fatal("attribute or zero-mana rows missing", sample, attributes, manaZero)
		}
		pic := ui.RenderCharacterPanel(l, f.tipFont(), s)
		ink, headings := 0, 0
		for y := 0; y < pic.Bounds().Dy(); y++ {
			lo, hi := pic.Bounds().Dx(), -1
			for x := 0; x < pic.Bounds().Dx(); x++ {
				if pic.RGBAAt(x, y) == (color.RGBA{148, 89, 0, 255}) && pic.RGBAAt(x, y) != l.Background.RGBAAt(x, y) {
					headings++
					if y < 112 || y >= 122 {
						t.Fatal("section color escaped the heading row", sample, x, y)
					}
				}
				if ui.CharacterCardWritable(l.Background.RGBAAt(x, y)) {
					lo, hi = min(lo, x), max(hi, x)
				}
			}
			for x := 0; x < pic.Bounds().Dx(); x++ {
				if pic.RGBAAt(x, y) != l.Background.RGBAAt(x, y) {
					ink++
					if x < lo || x > hi {
						t.Fatal("text escaped the curved face", x, y)
					}
				}
			}
		}
		if ink < 400 || headings < 30 {
			t.Fatal("card lost its text or orange headings", sample, ink, headings)
		}
	}
	if !bytes.Equal(background.Pix, untouched) {
		t.Fatal("card recolor mutated the shared installed bitmap")
	}
}

// Only the source application's selection and book fields are controlled;
// actor definitions, learned spells and the graph are the authentic corpus.
func releaseOriginalSelectedCursor(t *testing.T) {
	f := releaseFront(t)
	_, source := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	open, town, err := f.RestoreOriginal(source)
	if err != nil || town {
		t.Fatal("source restore", err)
	}
	if err := f.App("select source caster").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	var runtime uint32
	for _, e := range f.live.world.Entities() {
		if e.Owner == sim.SelfSlot && e.HP > 0 && e.KnownSpells&(1<<18) != 0 {
			runtime = e.SourceBinding.RuntimeID
			break
		}
	}
	if runtime == 0 {
		for _, e := range f.live.world.Entities() {
			if e.Owner == sim.SelfSlot {
				t.Logf("source actor%d spells=%x mana=%d", e.ID, e.KnownSpells, e.Mana)
			}
		}
		t.Fatal("source has no owner caster with spell18")
	}
	file, err := sav.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	index := int32(-1)
	for i, id := range originalBookIDs {
		if id == 18 {
			index = int32(i)
		}
	}
	for _, field := range []struct {
		section, key string
		value        int32
	}{{"SpellBook", "Pressed", index}, {"SpellBook", "IsOpen", 0}, {"Inventory", "IsOpen", 0}} {
		if err := file.SetStoreInt(field.section, field.key, field.value); err != nil {
			t.Fatal(err)
		}
	}
	if err := file.SetStoreIntArray("Objects", "Selection", []int32{int32(runtime)}); err != nil {
		t.Fatal(err)
	}
	controlled := file.Marshal()
	var native []byte
	for _, format := range []string{"sav", "ags"} {
		fresh := releaseFront(t)
		fresh.SetDeterministicFrames(true)
		var open ui.MapOpener
		if format == "sav" {
			open, town, err = fresh.RestoreOriginal(controlled)
		} else {
			var s Snapshot
			s, _, err = DecodeSave(native)
			if err == nil {
				open, town, err = fresh.Restore(s)
			}
		}
		if err != nil || town {
			t.Fatal(format, "cold prepare", err)
		}
		a := fresh.App("cold selected spell")
		if err := a.OpenMission(open); err != nil {
			t.Fatal(err)
		}
		a.Layout(1024, 768)
		if _, spell, armed := fresh.live.view.QuickSpellState(); spell != 18 || !armed {
			t.Fatal(format, "lost the loaded cast cursor", spell, armed)
		}
		if format == "sav" {
			s, _, err := fresh.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			native, err = EncodeSave(s, "selected cursor")
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := a.HeadlessKey("0"); err != nil {
			t.Fatal(err)
		}
		e := world1170Entity(t, fresh, runtime)
		inspectionCentre(fresh.live, int(e.X), int(e.Y))
		x, y, err := a.HeadlessEntityPoint(uint32(e.ID))
		if err != nil {
			t.Fatal(err)
		}
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
		orders := fresh.live.pending
		if len(orders) != 1 || orders[0].Kind != sim.KindCast || orders[0].Y != 18 {
			t.Fatal(format, "first loaded click did not cast", orders)
		}
		t.Log(format, "cold LOAD arms spell18; first click queues the spell without a shortcut")
	}
}
