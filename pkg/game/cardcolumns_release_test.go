package game

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// The statistics card on the installed font and installed words: every label
// flush left and every value flush right per column, health and mana centred in
// their half, sight with one decimal, and nothing shortened.
func TestReleaseStatisticsCardColumnsAreFlush(t *testing.T) {
	f := releaseFront(t)
	art, err := LoadTownCharacterPaneArt(f.Archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	background := art.Stats.Body.(*image.RGBA)
	l := ui.CompactPanelLayout(background)
	font := f.tipFont()

	mage := ui.PanelSubject{Name: "FERION", Words: f.Words, HP: 77, MaxHP: 77, Mana: 215, MaxMana: 215,
		DetailSet: true, DetailLevel: 7, Speed: 15, WeightKnown: true, Weight: 6,
		Char: ui.UnitCharacter{Known: true, Mage: true, Body: 26, Reaction: 15, Mind: 40, Spirit: 37,
			Experience: 302022, Sight: 6, Skills: [6]int{0, 28, 58, 22, 17, 38}, Protection: [5]int{23, 18, 18, 18, 18}},
		Combat: ui.UnitCombat{Known: true, Defence: 35}}
	fighter := mage
	fighter.Name, fighter.HP, fighter.MaxHP, fighter.Mana, fighter.MaxMana = "DANATH", 245, 245, 0, 0
	fighter.Speed, fighter.Weight = 17, 239
	fighter.Char = ui.UnitCharacter{Known: true, Body: 43, Reaction: 25, Mind: 26, Spirit: 26,
		Experience: 120654, Sight: 6, Skills: [6]int{0, 50, 2, 0, 11, 17}, Protection: [5]int{13, 14, 15, 14, 100}}
	fighter.Combat = ui.UnitCombat{Known: true, DamageBase: 6, DamageSpread: 4, ToHit: 40, Defence: 15, Absorption: 7}

	staff := mage
	staff.Name = "STAFFER"
	staff.Combat.WeaponSpellKnown = true
	for name, s := range map[string]ui.PanelSubject{"mage": mage, "fighter": fighter, "staff-mage": staff} {
		rows := ui.CharacterPanelReport(l, font, s)
		statement := ui.PanelStatement(l, s)
		if len(rows) != len(statement) {
			t.Fatalf("%s: %d rows drawn, %d stated", name, len(rows), len(statement))
		}
		pic := ui.RenderCharacterPanel(l, font, s)
		ink := func(x0, x1, y0, y1 int) (lo, hi int) {
			lo, hi = 1<<30, -1
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					if pic.RGBAAt(x, y) == l.ValueColor {
						lo, hi = min(lo, x), max(hi, x)
					}
				}
			}
			return lo, hi
		}
		for _, r := range rows {
			if r.Value != s.Name || r.Label != "" {
				continue
			}
			lo, hi := 1<<30, -1
			for y := r.At.Y; y < r.At.Y+font.Height(); y++ {
				for x := 0; x < pic.Bounds().Dx(); x++ {
					if pic.RGBAAt(x, y) == l.LabelColor {
						lo, hi = min(lo, x), max(hi, x)
					}
				}
			}
			if c := (lo + hi + 1) / 2; c < 71 || c > 73 {
				t.Errorf("%s: name ink centred at %d (%d..%d), want 72", name, c, lo, hi)
			}
		}
		for _, r := range rows {
			armor := r.RightLabel == f.Words.PanelCaptions[24] || r.RightLabel == f.Words.PanelCaptions[26]
			if r.Label == f.Words.PanelCaptions[24] || r.Label == f.Words.PanelCaptions[26] {
				t.Errorf("%s: %q is drawn in the left column", name, r.Label)
			}
			if armor {
				rvw, _ := font.Measure(r.RightValue)
				if r.At.X+r.RightX != 74 || r.At.X+r.RightX+r.RightValueX+rvw != 140 {
					t.Errorf("%s: %q at x=%d ending %d, want 74 and 140", name, r.RightLabel, r.At.X+r.RightX, r.At.X+r.RightX+r.RightValueX+rvw)
				}
				if name == "staff-mage" && (r.Label != "" || r.Value != "") {
					t.Errorf("%s: %q has a left cell %q %q", name, r.RightLabel, r.Label, r.Value)
				}
			}
		}
		columns := map[string][]int{}
		for i, r := range rows {
			joined := strings.TrimSpace(r.Label + " " + r.Value)
			if r.Right {
				joined += "  " + strings.TrimSpace(r.RightLabel+" "+r.RightValue)
			}
			if r.Label == "" && r.Value == "" && r.Right {
				joined = "  " + strings.TrimSpace(r.RightLabel+" "+r.RightValue)
			}
			if strings.Join(strings.Fields(joined), " ") != strings.Join(strings.Fields(statement[i]), " ") {
				t.Errorf("%s row %d drawn %q, stated %q", name, i, joined, statement[i])
			}
			h := font.Height()
			if r.Value != "" && r.At.Y >= 44 {
				vw, _ := font.Measure(r.Value)
				end := r.At.X + r.ValueX + vw
				lo, hi := ink(r.At.X+r.ValueX, end+2, r.At.Y, r.At.Y+h)
				if hi < 0 || hi < end-2 || hi > end-1 {
					t.Errorf("%s %q %q: value ink ends at %d, measured end %d", name, r.Label, r.Value, hi, end)
				}
				columns[fmt.Sprintf("left-%d", end)] = append(columns[fmt.Sprintf("left-%d", end)], lo)
			}
			if r.Right && r.RightValue != "" && r.RightLabel != "" {
				rvw, _ := font.Measure(r.RightValue)
				end := r.At.X + r.RightX + r.RightValueX + rvw
				_, hi := ink(r.At.X+r.RightX+r.RightValueX, end+2, r.At.Y, r.At.Y+h)
				if hi < 0 || hi < end-2 || hi > end-1 {
					t.Errorf("%s %q %q: right value ink ends at %d, measured end %d", name, r.RightLabel, r.RightValue, hi, end)
				}
				columns[fmt.Sprintf("right-%d", end)] = append(columns[fmt.Sprintf("right-%d", end)], hi)
			}
			if r.Label != "" && r.Value != "" {
				labelEnd := r.At.X + font.Advance(r.Label)
				if r.At.X+r.ValueX < labelEnd+1 {
					t.Errorf("%s %q %q: value starts at %d inside the label ending at %d", name, r.Label, r.Value, r.At.X+r.ValueX, labelEnd)
				}
			}
		}
		if len(columns) < 4 {
			t.Errorf("%s: only %d value columns found (%v)", name, len(columns), columns)
		}
		var sight, speed bool
		for _, r := range rows {
			switch r.Label {
			case f.Words.PanelCaptions[21]:
				sight = r.Value == "6.0"
				if !sight {
					t.Errorf("%s: sight %q, want 6.0", name, r.Value)
				}
			case f.Words.PanelCaptions[22]:
				speed = r.Value == fmt.Sprint(s.Speed)
			}
		}
		if !sight || !speed {
			t.Errorf("%s: sight row %v, speed row %v", name, sight, speed)
		}
		if dir := os.Getenv("AGAINROM_CARD_LAYOUT_OUT"); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			out, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s-%s.png", filepath.Base(os.Getenv("AGAINROM_ASSETS")), name)))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(out, pic); err != nil {
				t.Fatal(err)
			}
			out.Close()
		}
	}
}
