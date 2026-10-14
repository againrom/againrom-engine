package ui

import (
	"bytes"
	"image"
	"reflect"
	"strconv"
	"testing"

	"againrom/pkg/render/text"
)

func resistanceHeadingSubject() PanelSubject {
	w := AuthoredWords()
	w.PanelCaptions[27], w.PanelCaptions[28], w.PanelCaptions[183] = "SLOT27", "RIGHT28", "SLOT183"
	w.Hover[168], w.Hover[188], w.Hover[169] = "SKILL HELP", "WEAPON RESISTANCE HELP", "ELEMENT HELP"
	return PanelSubject{
		ID: 9, Name: "Subject", DetailSet: true, DetailLevel: 7, Words: w,
		Char: UnitCharacter{
			Known: true, Band: CharacterBandCreature,
			Skills:     [PanelSkillSlots]int{91, 12, 23, 34, 45, 56},
			Protection: [5]int{67, 78, 89, 90, 11},
		},
	}
}

type resistanceLayoutFixture struct {
	name string
	make func() PanelLayout
}

func resistanceLayouts() []resistanceLayoutFixture {
	return []resistanceLayoutFixture{
		{"authored", AuthoredPanelLayout},
		{"compact", func() PanelLayout { return CompactPanelLayout(nil) }},
		{"compact face", func() PanelLayout { return CompactPanelLayout(cardColumnsBackground()) }},
	}
}

func TestCreatureResistanceHeadingEditionAndPersonControls(t *testing.T) {
	for _, layout := range resistanceLayouts() {
		for _, rule := range []struct {
			name    string
			enabled bool
		}{
			{"first game", true},
			{"second game", false},
		} {
			for _, subject := range []struct {
				name string
				band CharacterBand
				hero bool
				mage bool
			}{
				{"creature", CharacterBandCreature, false, false},
				{"human", CharacterBandPerson, false, false},
				{"hero fighter", CharacterBandPerson, true, false},
				{"hero mage", CharacterBandPerson, true, true},
			} {
				t.Run(layout.name+"/"+rule.name+"/"+subject.name, func(t *testing.T) {
					l, font, s := layout.make(), panelFont(), resistanceHeadingSubject()
					s.PreserveCreatureSkillsHeading = !rule.enabled
					s.Char.Band, s.Char.Mage = subject.band, subject.mage
					if subject.hero {
						s.OriginalPanel = OriginalPanelActor{Known: true, Flags: 0x1}
					}
					wantLabel, wantSlot, wantHelp := "SLOT27", 168, "SKILL HELP"
					if rule.enabled && subject.band == CharacterBandCreature {
						wantLabel, wantSlot, wantHelp = "RIGHT28", 188, "WEAPON RESISTANCE HELP"
					}
					found := false
					lines := panelLines(l, font, s)
					for _, line := range lines {
						if line.field != PanelFieldSkillsHeading {
							continue
						}
						found = true
						if line.label != wantLabel {
							t.Errorf("drawn left heading = %q, want %q", line.label, wantLabel)
						}
						if !line.right || line.rightLabel != "RIGHT28" || line.value != "" || line.rightValue != "" {
							t.Errorf("heading pair = %+v, want unchanged empty values and RIGHT28", line)
						}
					}
					if !found {
						t.Fatal("left heading was not drawn")
					}
					if got := panelTooltipSlot(PanelFieldSkillsHeading, s); got != wantSlot {
						t.Errorf("left heading hover slot = %d, want %d", got, wantSlot)
					}
					if got := panelTooltipSlot(PanelFieldResistHeading, s); got != 169 {
						t.Errorf("right heading hover slot = %d, want 169", got)
					}
					v := TownCharacterView{Subject: s, HasSubject: true, Statistics: true,
						Font: font, CardLayout: &l, PaneRect: image.Rect(0, 0, 400, 400)}
					for _, line := range panelLines(v.statsLayout(), font, s) {
						if line.field != PanelFieldSkillsHeading {
							continue
						}
						at := line.at.Add(image.Pt(line.leftIndent, 0))
						if got := characterStatsTooltip(v, s.Words, at); !reflect.DeepEqual(got.lines, []string{wantHelp}) {
							t.Errorf("painted left heading hover = %v, want %q", got.lines, wantHelp)
						}
						at = line.at.Add(image.Pt(line.rightX, 0))
						if got := characterStatsTooltip(v, s.Words, at); !reflect.DeepEqual(got.lines, []string{"ELEMENT HELP"}) {
							t.Errorf("painted right heading hover = %v, want ELEMENT HELP", got.lines)
						}
					}
					values := 0
					for _, line := range lines {
						slot, ok := panelSkillSlot(line.field)
						if !ok || slot == 0 {
							continue
						}
						values++
						if line.value != strconv.Itoa(s.Char.Skills[slot]) || !line.right || line.rightValue != strconv.Itoa(s.Char.Protection[slot-1]) {
							t.Errorf("weapon/element row %d changed: %+v", slot, line)
						}
					}
					if values != 5 {
						t.Errorf("drawn weapon/element rows = %d, want 5", values)
					}
				})
			}
		}
	}
}

func TestCreatureResistanceHeadingPreservesKnowledgeGates(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, band := range []CharacterBand{CharacterBandCreature, CharacterBandPerson} {
			for _, known := range []bool{false, true} {
				for level := 0; level <= 7; level++ {
					s := resistanceHeadingSubject()
					s.PreserveCreatureSkillsHeading = !enabled
					s.Char.Band, s.Char.Known, s.DetailLevel = band, known, level
					for _, field := range []PanelField{
						PanelFieldSkillsHeading, PanelFieldSkillBlade, PanelFieldSkillAxe,
						PanelFieldSkillBludgeon, PanelFieldSkillPike, PanelFieldSkillShooting,
					} {
						if _, got := panelText(s, field); got != (known && level == 7) {
							t.Errorf("rule %v band %d known %v level %d field %d visibility = %v", enabled, band, known, level, field, got)
						}
					}
					for _, field := range []PanelField{
						PanelFieldResistHeading, PanelFieldProtFire, PanelFieldProtWater,
						PanelFieldProtAir, PanelFieldProtEarth, PanelFieldProtAstral,
					} {
						if _, got := panelText(s, field); got != (known && level >= 6) {
							t.Errorf("rule %v band %d known %v level %d field %d visibility = %v", enabled, band, known, level, field, got)
						}
					}
				}
			}
		}
	}
}

func TestCreatureResistanceHeadingUsesRightCaptionPath(t *testing.T) {
	for _, caption := range []string{"INSTALLED28", "\x91\x8e\x8f\x90\x8e\x92\x88\x82\x8b\x85\x8d\x88\x85", ""} {
		for _, layout := range resistanceLayouts() {
			t.Run(layout.name+"/"+caption, func(t *testing.T) {
				s := resistanceHeadingSubject()
				s.PreserveCreatureSkillsHeading = false
				s.Words.PanelCaptions[28] = caption
				want := panelLabel(s, PanelFieldResistHeading, "RESISTANCE")
				if got := panelLabel(s, PanelFieldSkillsHeading, "SKILLS"); got != want {
					t.Errorf("left caption path = %q, right caption path = %q", got, want)
				}
				found := false
				for _, line := range panelLines(layout.make(), panelFont(), s) {
					if line.field == PanelFieldSkillsHeading {
						found = true
						if line.label != want || line.rightLabel != want {
							t.Errorf("drawn headings = %q/%q, want %q/%q", line.label, line.rightLabel, want, want)
						}
					}
				}
				if !found {
					t.Fatal("left heading was not drawn")
				}
			})
		}
	}
}

func TestCreatureResistanceHeadingPreservesCardOutsideHeading(t *testing.T) {
	for _, layout := range resistanceLayouts() {
		t.Run(layout.name, func(t *testing.T) {
			l, font, s := layout.make(), panelFont(), resistanceHeadingSubject()
			s.PreserveCreatureSkillsHeading = false
			legacy := s
			legacy.PreserveCreatureSkillsHeading = true
			oldReport := CharacterPanelReport(l, font, legacy)
			newReport := CharacterPanelReport(l, font, s)
			oldLines := panelLines(l, font, legacy)
			if len(newReport) != len(oldReport) {
				t.Fatalf("row count = %d, want %d", len(newReport), len(oldReport))
			}
			heading := -1
			for i, before := range oldReport {
				after := newReport[i]
				if oldLines[i].field == PanelFieldSkillsHeading {
					heading = i
					if after.Label != s.Words.PanelCaptions[28] || after.Value != "" || before.Value != "" {
						t.Errorf("creature heading = %+v, want slot28 with no value", after)
					}
					after.Label, after.ValueX = before.Label, before.ValueX
				}
				if !reflect.DeepEqual(after, before) {
					t.Errorf("row %d geometry or other text changed: got %+v, want %+v", i, after, before)
				}
			}
			if heading < 0 {
				t.Fatal("creature heading missing")
			}
			before := RenderCharacterPanel(l, font, legacy)
			after := RenderCharacterPanel(l, font, s)
			if before == nil || after == nil || before.Bounds() != after.Bounds() {
				t.Fatal("card bounds changed")
			}
			line := oldLines[heading]
			left := line.at.X + line.leftIndent
			changedArea := image.Rect(left, line.at.Y, line.at.X+line.rightX, line.at.Y+font.Height())
			changes := 0
			for y := before.Bounds().Min.Y; y < before.Bounds().Max.Y; y++ {
				for x := before.Bounds().Min.X; x < before.Bounds().Max.X; x++ {
					if before.RGBAAt(x, y) == after.RGBAAt(x, y) {
						continue
					}
					changes++
					if !image.Pt(x, y).In(changedArea) {
						t.Fatalf("pixel %d,%d outside left heading changed", x, y)
					}
				}
			}
			if changes == 0 {
				t.Fatal("the new heading did not change any card pixel")
			}
			blank := legacy
			blank.Words.PanelCaptions[27] = " "
			want := RenderCharacterPanel(l, font, blank)
			ink := l.LabelColor
			if l.CompactCard {
				ink = characterCardHeading
			}
			font.Draw(want, s.Words.PanelCaptions[28], left, line.at.Y, ink)
			if !bytes.Equal(after.Pix, want.Pix) {
				t.Fatal("card differs from an independently painted slot28 caption in the card font")
			}
			for _, control := range []struct {
				name string
				hero bool
				mage bool
			}{
				{"human", false, false},
				{"hero fighter", true, false},
				{"hero mage", true, true},
			} {
				person := s
				person.Char.Band, person.Char.Mage = CharacterBandPerson, control.mage
				if control.hero {
					person.OriginalPanel = OriginalPanelActor{Known: true, Flags: 0x1}
				}
				ordinary := person
				ordinary.PreserveCreatureSkillsHeading = true
				plain, enabled := RenderCharacterPanel(l, font, ordinary), RenderCharacterPanel(l, font, person)
				if !bytes.Equal(plain.Pix, enabled.Pix) || plain.Bounds() != enabled.Bounds() {
					t.Errorf("%s card pixels changed under the creature heading rule", control.name)
				}
			}
		})
	}
}

func TestCreatureResistanceHeadingHoverWithVisibleCombat(t *testing.T) {
	font := panelFont()
	for k := range font.Glyphs {
		g := &font.Glyphs[k]
		g.Height = 10
		g.Pixels = make([]text.Pixel, g.Width*g.Height)
		if k != 0 {
			g.Pixels[(k%g.Height)*g.Width+(k/g.Height)%g.Width] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
	}
	s := resistanceHeadingSubject()
	s.PreserveCreatureSkillsHeading = false
	s.Combat = UnitCombat{Known: true, DamageBase: 4, DamageSpread: 2, ToHit: 5, Defence: 6, Absorption: 7}
	s.Words.Hover[162], s.Words.Hover[164] = "ATTACK HELP", "DEFENSE HELP"
	l := CompactPanelLayout(cardColumnsBackground())
	v := TownCharacterView{Subject: s, HasSubject: true, Statistics: true,
		Font: font, CardLayout: &l, PaneRect: image.Rect(0, 0, 160, 242)}
	var heading, attack panelLine
	foundHeading, foundAttack := false, false
	for _, line := range panelLines(v.statsLayout(), font, s) {
		switch line.field {
		case PanelFieldSkillsHeading:
			heading, foundHeading = line, true
		case PanelFieldToHit:
			attack, foundAttack = line, true
		}
	}
	left := heading.at.X + heading.leftIndent
	if !foundHeading || !foundAttack || left != 6 || attack.at.X+attack.leftIndent != left ||
		heading.at.Y-attack.at.Y != 14 || attack.at.Y+font.Height() > heading.at.Y {
		t.Fatalf("fixture does not separate the combat row from the heading: heading=%+v attack=%+v", heading, attack)
	}
	check := func(name string, view TownCharacterView, at image.Point, slot int) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			if got := characterStatsTooltip(view, s.Words, at); !reflect.DeepEqual(got.lines, []string{s.Words.Hover[slot]}) {
				t.Errorf("point %v hover = %v, want slot %d (%q)", at, got.lines, slot, s.Words.Hover[slot])
			}
		})
	}
	w, _ := font.Measure(heading.label)
	check("creature heading top-left", v, image.Pt(left, heading.at.Y), 188)
	check("creature heading top-right", v, image.Pt(left+w-1, heading.at.Y), 188)
	check("creature heading last-row", v, image.Pt(left, heading.at.Y+font.Height()-1), 188)
	check("attack above heading", v, image.Pt(left, attack.at.Y+font.Height()-1), 162)
	right := heading.at.X + heading.rightX
	check("right heading top unchanged", v, image.Pt(right, heading.at.Y), 169)
	check("right heading last-row unchanged", v, image.Pt(right, heading.at.Y+font.Height()-1), 169)
	human := v
	human.Subject.Char.Band = CharacterBandPerson
	check("human heading unchanged", human, image.Pt(left, heading.at.Y), 168)
	second := v
	second.Subject.PreserveCreatureSkillsHeading = true
	check("second-game heading unchanged", second, image.Pt(left, heading.at.Y), 168)
}

func TestCreatureResistanceHeadingRuleReachesSelectedAndInspectedSubjects(t *testing.T) {
	s := resistanceHeadingSubject()
	e := MapEntity{ID: s.ID, Name: s.Name, Life: LifeAlive, Char: s.Char, Combat: s.Combat}
	v := &Viewer{panelLayout: CompactPanelLayout(nil), entities: []MapEntity{e}, sel: selection{e.ID}, words: s.Words}
	for _, preserve := range []bool{false, true, false} {
		v.SetPreserveCreatureSkillsHeading(preserve)
		serial := v.panelSerial
		layout := AuthoredPanelLayout()
		layout.PreserveCreatureSkillsHeading = !preserve
		v.SetPanelLayout(layout)
		v.SetCharacterPaneArt(TownPane{}, TownPane{}, nil)
		if v.panelLayout.PreserveCreatureSkillsHeading != preserve || v.panelSerial != serial+2 {
			t.Fatalf("rule %v did not survive layout/art changes and invalidate the card", preserve)
		}
		selected, ok := v.panelSubject()
		if !ok || selected.PreserveCreatureSkillsHeading != preserve {
			t.Fatalf("selected subject did not receive rule %v: %+v", preserve, selected)
		}
		inspected, ok := v.inspectionPanel(InspectionSubject{Kind: InspectionUnit, ID: e.ID})
		if !ok || inspected.PreserveCreatureSkillsHeading != preserve {
			t.Fatalf("inspected subject did not receive rule %v: %+v", preserve, inspected)
		}
		inspected.Selected = selected.Selected
		if inspected != selected {
			t.Fatal("selected and inspected subjects differ outside the selection count")
		}
	}
}
