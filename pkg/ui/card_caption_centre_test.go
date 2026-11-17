package ui

import "testing"

// The SPELLCASTER and ARMOR PIERCING captions stand centred on the card, in
// the authored layout, the compact layout and the compact layout on the card
// face, wherever no left cell shares the row.
func TestCardCaptionsStandCentred(t *testing.T) {
	f := panelFont()
	base := PanelSubject{DetailLevel: 7, DetailSet: true, Name: "MERCENARY MAGE",
		Char:          UnitCharacter{Known: true, ExperienceWithheld: true, Band: CharacterBandPerson},
		OriginalPanel: OriginalPanelActor{Known: true, XPValue: 9, Byte14A: 3}}
	layouts := map[string]struct {
		l      PanelLayout
		centre int
	}{
		"authored":     {AuthoredPanelLayout(), authoredPanelWidth / 2},
		"compact":      {CompactPanelLayout(nil), compactPanelW / 2},
		"compact face": {CompactPanelLayout(cardColumnsBackground()), cardNameCentre},
	}
	for name, c := range layouts {
		for _, field := range []PanelField{PanelFieldSpellcaster, PanelFieldArmorPiercing} {
			found := false
			for _, r := range CharacterPanelReport(c.l, f, base) {
				var label string
				var x int
				switch {
				case r.Field == field && !r.Right:
					label, x = r.Label, r.At.X
				case r.Right && r.RightField == field:
					label, x = r.RightLabel, r.At.X+r.RightX
				default:
					continue
				}
				found = true
				w, _ := f.Measure(label)
				if d := x + w/2 - c.centre; d < -1 || d > 1 {
					t.Errorf("%s field %d %q: text %d..%d, centre %d, off by %d", name, field, label, x, x+w, c.centre, d)
				}
			}
			if !found {
				t.Errorf("%s field %d: caption not stated", name, field)
			}
		}
	}
}

// Beside an XP value the caption keeps its column: the centred place would
// run over the value.
func TestCardCaptionBesideXPKeepsItsColumn(t *testing.T) {
	f := panelFont()
	s := PanelSubject{DetailLevel: 7, DetailSet: true, Name: "GUARD",
		Char:          UnitCharacter{Known: true, Experience: 12345, Band: CharacterBandPerson},
		OriginalPanel: OriginalPanelActor{Known: true, XPValue: 9, Byte14A: 3}}
	for name, l := range map[string]PanelLayout{"authored": AuthoredPanelLayout(), "compact face": CompactPanelLayout(cardColumnsBackground())} {
		seen := false
		for _, r := range CharacterPanelReport(l, f, s) {
			if !r.Right || r.RightField != PanelFieldArmorPiercing {
				continue
			}
			seen = true
			vw, _ := f.Measure(r.Value)
			if left, right := r.At.X+r.ValueX+vw, r.At.X+r.RightX; right < left {
				t.Errorf("%s: caption at %d overlaps the XP value ending at %d", name, right, left)
			}
		}
		if !seen {
			t.Errorf("%s: no XP row shares its line with the caption", name)
		}
	}
}
