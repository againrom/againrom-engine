package ui

import (
	"image"
	"image/color"
	"testing"
)

func cardColumnsBackground() *image.RGBA {
	bg := image.NewRGBA(image.Rect(0, 0, compactPanelW, compactPanelH))
	for y := 0; y < compactPanelH; y++ {
		for x := 0; x < compactPanelW; x++ {
			bg.SetRGBA(x, y, color.RGBA{50, 90, 50, 255})
		}
	}
	return bg
}

func cardColumnsSubjects() map[string]PanelSubject {
	fighter := PanelSubject{
		Name: "FIGHTER", HP: 245, MaxHP: 245, Mana: 0, MaxMana: 0, Speed: 17, Weight: 239, WeightKnown: true,
		Char: UnitCharacter{Known: true, Body: 43, Reaction: 25, Mind: 26, Spirit: 26, Experience: 120654,
			Sight: 6, Skills: [6]int{0, 50, 2, 0, 11, 17}, Protection: [5]int{13, 14, 15, 14, 100}},
		Combat: UnitCombat{Known: true, DamageBase: 6, DamageSpread: 4, ToHit: 40, Defence: 15, Absorption: 7},
	}
	mage := fighter
	mage.Name, mage.HP, mage.MaxHP, mage.Mana, mage.MaxMana, mage.Speed = "MAGE", 77, 77, 215, 215, 15
	mage.Char.Mage, mage.Char.Sight256 = true, 6<<8|0x80
	return map[string]PanelSubject{"fighter": fighter, "mage": mage}
}

// Labels are flush left and values flush right in every column, so the right
// edge of the digits in a column is one number, whatever the label beside it.
func TestCompactCardLabelsFlushLeftAndValuesFlushRight(t *testing.T) {
	f := panelFont()
	l := CompactPanelLayout(cardColumnsBackground())
	attrs := map[string]bool{"BODY": true, "AGILITY": true, "MIND": true, "SPIRIT": true}
	lower := map[string]bool{"DMG": true, "ATTACK": true, "BLADE": true, "AXE": true, "BLUDGEON": true,
		"PIKE": true, "SHOOTING": true, "FIRE": true, "WATER": true, "AIR": true, "EARTH": true, "ASTRAL": true}
	rightCaptions := map[string]bool{"ABSORB": true, "DEFENSE": true, "FIRE": true, "WATER": true, "AIR": true,
		"EARTH": true, "ASTRAL": true}
	for name, s := range cardColumnsSubjects() {
		seen := map[string]int{}
		for _, r := range CharacterPanelReport(l, f, s) {
			vw, _ := f.Measure(r.Value)
			rvw, _ := f.Measure(r.RightValue)
			leftEnd := r.At.X + r.ValueX + vw
			rightEnd := r.At.X + r.RightX + r.RightValueX + rvw
			check := func(what string, got, want int) {
				t.Helper()
				seen[what]++
				if got != want {
					t.Errorf("%s %s %q/%q: %d, want %d", name, what, r.Label, r.RightLabel, got, want)
				}
			}
			switch {
			case attrs[r.Label]:
				check("attribute label pen", r.At.X, cardLabelPen)
				check("attribute value end", leftEnd, cardAttrValueEnd)
			case lower[r.Label] && r.Value != "":
				check("left label pen", r.At.X, cardLabelPen)
				check("left value end", leftEnd, cardLowerValueEnd)
			case r.Label == "WEIGHT" || r.Label == "XP":
				check("totals label pen", r.At.X, cardTotalsPen)
				check("totals value end", leftEnd, cardTotalsValueEnd)
			case r.Label == "SIGHT" || r.Label == "SPEED":
				check("gauge label pen", r.At.X, cardGaugePen)
				check("gauge value end", leftEnd, cardGaugeValueEnd)
			}
			if r.Right && rightCaptions[r.RightLabel] && r.RightValue != "" {
				check("right label pen", r.At.X+r.RightX, cardRightLabelPen)
				check("right value end", rightEnd, cardRightValueEnd)
			}
			if r.Right && r.RightLabel == "" && r.RightValue != "" {
				w, _ := f.Measure(r.RightValue)
				if c := r.At.X + r.RightX + w/2; c != cardPoolCentre && c != cardPoolCentre-1 {
					t.Errorf("%s pool value %q centred at %d, want %d", name, r.RightValue, c, cardPoolCentre)
				}
				seen["pool value"]++
			}
			if r.Right && (r.RightLabel == "HEALTH" || r.RightLabel == "MANA") {
				w, _ := f.Measure(r.RightLabel)
				if c := r.At.X + r.RightX + w/2; c != cardPoolCentre && c != cardPoolCentre-1 {
					t.Errorf("%s pool heading %q centred at %d, want %d", name, r.RightLabel, c, cardPoolCentre)
				}
				seen["pool heading"]++
			}
		}
		for _, what := range []string{"attribute label pen", "attribute value end", "left label pen", "left value end",
			"totals label pen", "totals value end", "gauge label pen", "gauge value end", "right label pen",
			"right value end", "pool value", "pool heading"} {
			if seen[what] == 0 {
				t.Errorf("%s: no %s was checked", name, what)
			}
		}
	}
}

func TestCardSightStatesOneDecimal(t *testing.T) {
	for _, tc := range []struct {
		sight    int
		sight256 uint16
		want     string
	}{{6, 0, "6.0"}, {9, 0, "9.0"}, {6, 6<<8 | 0x80, "6.5"}, {6, 6 << 8, "6.0"}} {
		s := PanelSubject{Char: UnitCharacter{Known: true, Sight: tc.sight, Sight256: tc.sight256}}
		if got, ok := panelText(s, PanelFieldSight); !ok || got != tc.want {
			t.Errorf("sight %d/%#x: %q, want %q", tc.sight, tc.sight256, got, tc.want)
		}
	}
}

func TestCompactCardNameIsCentredOnTheOriginalsAxis(t *testing.T) {
	f := panelFont()
	l := CompactPanelLayout(cardColumnsBackground())
	for name, s := range cardColumnsSubjects() {
		found := false
		for _, r := range CharacterPanelReport(l, f, s) {
			if r.Value != s.Name {
				continue
			}
			w, _ := f.Measure(r.Value)
			if c := r.At.X + r.ValueX + w/2; c != cardNameCentre {
				t.Errorf("%s: name centred at %d, want %d", name, c, cardNameCentre)
			}
			found = true
		}
		if !found {
			t.Errorf("%s: no name row", name)
		}
	}
	if cardNameCentre != 72 {
		t.Errorf("name axis %d, want 72", cardNameCentre)
	}
}

func TestCompactCardArmorAndDefenceAlwaysStandInTheRightColumn(t *testing.T) {
	f := panelFont()
	l := CompactPanelLayout(cardColumnsBackground())
	staff := cardColumnsSubjects()["mage"]
	staff.Combat.WeaponSpellKnown = true
	subjects := cardColumnsSubjects()
	subjects["staff mage"] = staff
	for name, s := range subjects {
		got := map[string]bool{}
		for _, r := range CharacterPanelReport(l, f, s) {
			if r.RightLabel != "ABSORB" && r.RightLabel != "DEFENSE" {
				continue
			}
			rvw, _ := f.Measure(r.RightValue)
			if x := r.At.X + r.RightX; !r.Right || x != cardRightLabelPen {
				t.Errorf("%s %s: right cell %v at x=%d, want %d", name, r.RightLabel, r.Right, x, cardRightLabelPen)
			}
			if end := r.At.X + r.RightX + r.RightValueX + rvw; end != cardRightValueEnd {
				t.Errorf("%s %s: value ends at %d, want %d", name, r.RightLabel, end, cardRightValueEnd)
			}
			if name == "staff mage" && (r.Label != "" || r.Value != "") {
				t.Errorf("%s %s: left cell %q %q, want empty", name, r.RightLabel, r.Label, r.Value)
			}
			got[r.RightLabel] = true
		}
		if !got["ABSORB"] || !got["DEFENSE"] {
			t.Errorf("%s: rows found %v", name, got)
		}
		for _, r := range CharacterPanelReport(l, f, s) {
			if name == "staff mage" && (r.Label == "ABSORB" || r.Label == "DEFENSE") {
				t.Errorf("%s: %s drawn in the left column", name, r.Label)
			}
		}
	}
}
