package ui

import (
	"bytes"
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/render/text"
)

// The panel's composition (AC-6..AC-10). Every fixture here is built in this
// file; nothing reads a game install, and a byte at or above 0x80 is written
// as an escape.

const (
	panelCellW, panelCellH = 5, 6
	// Deliberately SHORTER than the cell, so a record painting in the cell's
	// last column paints past its own advance — which is the only
	// configuration in which placing a value by the label's pen and placing it
	// by the label's measured box are different numbers.
	panelAdvance      = 3
	panelSpaceAdvance = 2
	panelSpacing      = 1
)

// panelFont is a 224-record font in the shipped arrangement — record k is byte
// 32+k — whose records are DISTINGUISHABLE FROM EACH OTHER: record k paints one
// pixel at (row k%6, column (k/6)%5). Record 0 is the space and paints nothing.
//
// Distinguishable is the point. A font whose glyphs all looked alike would let
// a placement that selected the wrong record, or one record for every byte,
// pass every assertion below.
func panelFont() *text.Font {
	f := &text.Font{Spacing: panelSpacing, Glyphs: make([]text.Glyph, 224)}
	for k := range f.Glyphs {
		g := text.Glyph{Width: panelCellW, Height: panelCellH,
			Pixels: make([]text.Pixel, panelCellW*panelCellH), Advance: panelAdvance}
		if k == 0 {
			g.Advance = panelSpaceAdvance
		} else {
			g.Pixels[(k%panelCellH)*panelCellW+(k/panelCellH)%panelCellW] =
				text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		f.Glyphs[k] = g
	}
	return f
}

func TestOriginalPanelDetailLevelsZeroThroughSeven(t *testing.T) {
	s := PanelSubject{
		HP: 10, MaxHP: 20, Mana: 3, MaxMana: 7, DetailSet: true,
		Combat: UnitCombat{Known: true, DamageBase: 4, DamageSpread: 2, ToHit: 5, Defence: 6, Absorption: 7},
		Char:   UnitCharacter{Known: true, Body: 1, Reaction: 2, Mind: 3, Spirit: 4, Sight: 8},
		Speed:  9,
	}
	for level := 0; level <= 7; level++ {
		s.DetailLevel = level
		for _, tc := range []struct {
			field PanelField
			above int
		}{
			{PanelFieldSight, 1}, {PanelFieldMoveSpeed, 1},
			{PanelFieldDamage, 2}, {PanelFieldToHit, 2},
			{PanelFieldDefence, 3}, {PanelFieldAbsorption, 3},
			{PanelFieldBody, 4}, {PanelFieldReaction, 4}, {PanelFieldMind, 4}, {PanelFieldSpirit, 4},
			{PanelFieldSkillsHeading, 6}, {PanelFieldSkillBlade, 6}, {PanelFieldProtFire, 5},
			{PanelFieldResistHeading, 5}, {PanelFieldHealth, 0}, {PanelFieldMana, 0},
		} {
			if _, got := panelText(s, tc.field); got != (level > tc.above) {
				t.Errorf("level %d field %d visible=%v, want level > %d", level, tc.field, got, tc.above)
			}
		}
	}
	s.MaxHP, s.MaxMana = 0, 0
	if _, ok := panelText(s, PanelFieldHealth); ok {
		t.Error("zero health pool remained visible")
	}
	if _, ok := panelText(s, PanelFieldMana); ok {
		t.Error("zero mana pool remained visible")
	}
}

func TestPanelUsesSubjectNameBeforeInstalledClassAndResolvesFixedCaptions(t *testing.T) {
	w := AuthoredWords()
	w.UnitNames[23] = "installed unit"
	w.PanelCaptions[15] = "installed body"
	w.PanelCaptions[37] = "installed water school"
	w.PanelCaptions[190] = "installed spellcaster"
	w.PanelCaptions[191] = "installed armor piercing"
	s := PanelSubject{UnitNameIndex: 23, DetailLevel: 7, DetailSet: true, Words: w,
		Name: "Naira", Char: UnitCharacter{Known: true, Mage: true}}
	if got, ok := panelText(s, PanelFieldName); !ok || got != "Naira" {
		t.Fatalf("named subject panel name = %q,%v, want Naira", got, ok)
	}
	s.Name = ""
	if got, ok := panelText(s, PanelFieldName); !ok || got != "installed unit" {
		t.Fatalf("unnamed subject panel name = %q,%v, want installed class", got, ok)
	}
	if got := panelLabel(s, PanelFieldBody, "BODY"); got != "installed body" {
		t.Fatalf("body caption = %q", got)
	}
	if got := panelLabel(s, PanelFieldSkillAxe, "AXE"); got != "installed water school" {
		t.Fatalf("mage slot two caption = %q", got)
	}
	if got := panelLabel(s, PanelFieldSpellcaster, "SPELLCASTER"); got != "installed spellcaster" {
		t.Fatalf("spellcaster caption = %q", got)
	}
	if got := panelLabel(s, PanelFieldArmorPiercing, "ARMOR PIERCING"); got != "installed armor piercing" {
		t.Fatalf("armor-piercing caption = %q", got)
	}
}

// TestCompactPanelPaintsBothConditionalInstallCaptions closes the complete
// parsed-but-unconsumed class from 1043's first adversarial pass and the
// semantic-substitution defect found by pass 2. It walks the same
// CompactPanelLayout and CharacterPanelReport path RenderCharacterPanel uses,
// then mutation-checks each original actor operand against rendered pixels.
// Slot 190 takes the load row for a non-player unit; slot 191 sits in XP's
// right cell. Neither changes the card's row count or its fixed 160x242
// bounds.
func TestCompactPanelPaintsBothConditionalInstallCaptions(t *testing.T) {
	w := AuthoredWords()
	w.PanelCaptions[190] = "caster190"
	w.PanelCaptions[191] = "pierce191"
	s := PanelSubject{
		Words: w, DetailLevel: 7, DetailSet: true,
		Char:   UnitCharacter{Known: true, Mage: false, Experience: 9},
		Combat: UnitCombat{Known: true, AlwaysHits: false},
		OriginalPanel: OriginalPanelActor{Known: true, Flags: 0,
			XPValue: 7, Byte14A: 2},
		Weight:      35,
		WeightKnown: true,
	}
	l, f := CompactPanelLayout(nil), panelFont()
	report := CharacterPanelReport(l, f, s)
	foundCaster, foundPiercing := false, false
	for _, row := range report {
		if row.Label == "WEIGHT" {
			t.Fatalf("a unit that is not a player character states a load: %+v", row)
		}
		if row.Label == "caster190" && !row.Right {
			foundCaster = true
		}
		if row.RightLabel == "pierce191" {
			foundPiercing = row.Label == "XP"
		}
	}
	if !foundCaster || !foundPiercing {
		t.Fatalf("production report has caster=%v piercing=%v; want slot 190 alone on the load line and 191 on XP: %+v",
			foundCaster, foundPiercing, report)
	}
	full := RenderCharacterPanel(l, f, s)
	withoutCaster := s
	withoutCaster.OriginalPanel.XPValue = 0
	if bytes.Equal(full.Pix, RenderCharacterPanel(l, f, withoutCaster).Pix) {
		t.Fatal("clearing actor +0x1c changed no production pixels; slot 190 was not painted")
	}
	withoutPiercing := s
	withoutPiercing.OriginalPanel.Byte14A = 0
	if bytes.Equal(full.Pix, RenderCharacterPanel(l, f, withoutPiercing).Pix) {
		t.Fatal("clearing actor +0x14a changed no production pixels; slot 191 was not painted")
	}
	player := s
	player.OriginalPanel.Flags = 0x1
	player.Char.Mage = true
	player.Combat.AlwaysHits = true
	for _, row := range CharacterPanelReport(l, f, player) {
		if row.Label == "caster190" || row.RightLabel == "caster190" || row.RightLabel == "pierce191" {
			t.Fatalf("player bit 0 did not suppress both captions despite Mage/AlwaysHits: %+v", row)
		}
	}
	human := s
	human.OriginalPanel.Flags = 0x10
	seenCaster := false
	for _, row := range CharacterPanelReport(l, f, human) {
		if row.Label == "caster190" {
			seenCaster = true
		}
		if row.RightLabel == "pierce191" {
			t.Fatalf("human bit 4 did not suppress slot 191: %+v", row)
		}
	}
	if !seenCaster {
		t.Fatal("human bit 4 incorrectly suppressed slot 190, which tests bit 0 alone")
	}

	levelSix := s
	levelSix.DetailLevel = 6
	for _, row := range CharacterPanelReport(l, f, levelSix) {
		if row.Label == "caster190" || row.RightLabel == "pierce191" {
			t.Fatalf("full-detail-only caption survived at visibility level 6: %+v", row)
		}
	}
	if got := RenderCharacterPanel(l, f, s).Bounds().Size(); got != image.Pt(160, 242) {
		t.Fatalf("conditional captions changed compact geometry to %v", got)
	}
}

func TestMissionPanelCopiesTheOriginalConditionalOperandsWhole(t *testing.T) {
	want := OriginalPanelActor{Known: true, Flags: 0x10, XPValue: 81, Byte14A: 2}
	v := &Viewer{words: AuthoredWords()}
	got := v.panelSubjectFromPresent([]MapEntity{{ID: 7, OriginalPanel: want}})
	if got.OriginalPanel != want {
		t.Fatalf("mission panel actor projection = %+v, want %+v", got.OriginalPanel, want)
	}
}

func TestCompactPanelKeepsTheCompleteValueWhenAnInstalledCaptionIsWide(t *testing.T) {
	w := AuthoredWords()
	w.PanelCaptions[16] = "installed-caption-too-wide-for-one-fixed-column"
	s := PanelSubject{Words: w, DetailLevel: 7, DetailSet: true,
		Char: UnitCharacter{Known: true, Reaction: 34}}
	report := CharacterPanelReport(CompactPanelLayout(nil), panelFont(), s)
	for _, row := range report {
		if row.FullValue != "34" {
			continue
		}
		if row.Value != row.FullValue {
			t.Fatalf("wide installed caption truncated value from %q to %q", row.FullValue, row.Value)
		}
		if row.Label == w.PanelCaptions[16] {
			t.Fatal("fixture did not make the installed caption exceed its fixed column")
		}
		return
	}
	t.Fatal("compact panel report has no Reaction value")
}

// panelInkLayout is the authored layout with the frame blacked out, so any
// pixel carrying a non-zero channel is TEXT and nothing else. It is what makes
// "every painted pixel lies inside the box" an assertion rather than a
// tautology over a filled rectangle.
func panelInkLayout() PanelLayout {
	l := AuthoredPanelLayout()
	l.Fill = color.RGBA{A: 0xff}
	l.Border = color.RGBA{A: 0xff}
	l.LabelColor = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	l.ValueColor = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	return l
}

func panelSubjectFixture() PanelSubject {
	return PanelSubject{ID: 7, Name: "Warrior", HP: 63, MaxHP: 100, Cell: image.Pt(12, 34)}
}

// inkExtent is the furthest right and lowest text pixel in img, and whether any
// was found. Text is any pixel with a non-zero colour channel; the blacked-out
// frame has none.
func inkExtent(img *image.RGBA) (right, bottom int, found bool) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.R == 0 && c.G == 0 && c.B == 0 {
				continue
			}
			found = true
			if x > right {
				right = x
			}
			if y > bottom {
				bottom = y
			}
		}
	}
	return right, bottom, found
}

// AC-6 — composition is a function of its arguments.
func TestComposePanelIsDeterministic(t *testing.T) {
	l, f, s := AuthoredPanelLayout(), panelFont(), panelSubjectFixture()

	a, b := composePanel(l, f, s), composePanel(l, f, s)
	if a == nil || b == nil {
		t.Fatal("composePanel returned nil for a subject with a font")
	}
	if a.Bounds() != b.Bounds() {
		t.Fatalf("two composes of the same inputs differ in size: %v vs %v", a.Bounds(), b.Bounds())
	}
	if len(a.Pix) != len(b.Pix) {
		t.Fatalf("pixel buffers differ in length: %d vs %d", len(a.Pix), len(b.Pix))
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			t.Fatalf("byte %d of the pixel buffer differs: %#02x vs %#02x", i, a.Pix[i], b.Pix[i])
		}
	}
	// A no-font viewer composes nothing rather than an empty frame.
	if got := composePanel(l, nil, s); got != nil {
		t.Errorf("composePanel with no font = %v, want nil", got)
	}
	if got := composePanel(l, &text.Font{}, s); got != nil {
		t.Errorf("composePanel with a font holding no record = %v, want nil", got)
	}
}

// AC-7 — the panel states the layout's rows for the fields the subject has, and
// no other quantity. panelLines is the WHOLE of the text composition draws, so
// asserting over it asserts over everything the picture says.
func TestComposePanelStatesExactlyItsRows(t *testing.T) {
	f, s := panelFont(), panelSubjectFixture()
	lines := panelLines(AuthoredPanelLayout(), f, s)

	// The fixture carries no character and no combat numbers, so the sheet's
	// whole middle is absent and what survives is the name, the health block
	// and the cell.
	want := []struct{ label, value string }{
		{"", "Warrior"},
		{"HEALTH", ""},
		{"", "63/100"},
		{"CELL", "12, 34"},
	}
	if len(lines) != len(want) {
		t.Fatalf("the panel states %d row(s), want %d", len(lines), len(want))
	}
	for i, w := range want {
		if lines[i].label != w.label || lines[i].value != w.value {
			t.Errorf("row %d = (%q, %q), want (%q, %q)",
				i, lines[i].label, lines[i].value, w.label, w.value)
		}
	}

	// The label's own pen, plus the gap — and no gap at all after an empty
	// label, so the title sits flush with the labels beneath it.
	l := AuthoredPanelLayout()
	if lines[0].valueX != 0 {
		t.Errorf("the unlabelled row's value starts at %d, want 0 — an empty label spends no gap",
			lines[0].valueX)
	}
	if wantX := f.Advance("CELL") + l.LabelGap; lines[3].valueX != wantX {
		t.Errorf("the CELL row's value starts at %d, want %d (the label's pen plus the gap)",
			lines[3].valueX, wantX)
	}

	// The PEN and not the measured box, and the two are only distinguishable
	// against a label whose art overhangs its advance. Byte 0x38 selects the
	// record this fixture paints in its cell's last column.
	over := PanelLayout{Flow: true, LabelGap: l.LabelGap,
		Rows: []PanelRow{{Field: PanelFieldHealth, Label: "8"}}}
	if w, _ := f.Measure("8"); w == f.Advance("8") {
		t.Fatal("no glyph of the fixture font overhangs its advance: this case cannot discriminate")
	}
	if got := panelLines(over, f, s)[0].valueX; got != f.Advance("8")+l.LabelGap {
		t.Errorf("an overhanging label put its value at %d, want %d — the box was used for the pen",
			got, f.Advance("8")+l.LabelGap)
	}
}

func TestPanelStatesOnlyWhatTheSubjectCarries(t *testing.T) {
	f := panelFont()
	l := AuthoredPanelLayout()

	t.Run("an unnamed subject drops the name row and leaves no gap", func(t *testing.T) {
		s := panelSubjectFixture()
		s.Name = ""
		lines := panelLines(l, f, s)
		if len(lines) != 3 {
			t.Fatalf("an unnamed subject states %d row(s), want 3", len(lines))
		}
		// The health heading, the pair, the cell — the name row and nothing
		// else has gone.
		if lines[0].label != "HEALTH" || lines[1].value != "63/100" || lines[2].value != "12, 34" {
			t.Fatalf("rows are %q, %q and %q", lines[0].label, lines[1].value, lines[2].value)
		}
		// The dropped row costs no space: the first kept row sits at the pad.
		if lines[0].at.Y != l.Pad.Y {
			t.Errorf("the first kept row is at y=%d, want %d — a skipped row left a gap behind it",
				lines[0].at.Y, l.Pad.Y)
		}
	})

	for _, tc := range []struct {
		name   string
		s      PanelSubject
		health string
		cell   string
	}{
		{"downed", PanelSubject{Name: "n", HP: 0, MaxHP: 100, Cell: image.Pt(1, 2)}, "0/100", "1, 2"},
		{"dead", PanelSubject{Name: "n", HP: -37, MaxHP: 100, Cell: image.Pt(0, 0)}, "-37/100", "0, 0"},
		{"no health system", PanelSubject{Name: "n", HP: 0, MaxHP: 0, Cell: image.Pt(-1, -2)}, "0/0", "-1, -2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lines := panelLines(l, f, tc.s)
			// Name, the health heading, the pair, the cell.
			if len(lines) != 4 {
				t.Fatalf("states %d row(s), want 4 — health and cell are never absent", len(lines))
			}
			if lines[2].value != tc.health {
				t.Errorf("health = %q, want %q — it is stated as it stands, unclamped",
					lines[2].value, tc.health)
			}
			if lines[3].value != tc.cell {
				t.Errorf("cell = %q, want %q", lines[3].value, tc.cell)
			}
		})
	}

	// A field this build does not define omits its row rather than drawing one
	// with nothing in it.
	lines := panelLines(PanelLayout{Flow: true, Rows: []PanelRow{{Field: PanelField(200)}}},
		f, panelSubjectFixture())
	if len(lines) != 0 {
		t.Errorf("an undefined field yielded %d row(s), want none", len(lines))
	}
}

func TestPanelManaRowFollowsTheHealthRowsForm(t *testing.T) {
	l := AuthoredPanelLayout()

	// The block's shape in the LAYOUT ITSELF, not merely in the rows a
	// particular subject happens to leave behind. rightAt reports which row
	// carries a given field as its right cell.
	rightAt := func(f PanelField) int {
		for i, r := range l.Rows {
			if r.Right != nil && r.Right.Field == f {
				return i
			}
		}
		return -1
	}
	for _, pair := range []struct {
		name             string
		heading, value   PanelField
		headingLabelWant string
	}{
		{"health", PanelFieldHealthHeading, PanelFieldHealth, "HEALTH"},
		{"mana", PanelFieldManaHeading, PanelFieldMana, "MANA"},
	} {
		h, v := rightAt(pair.heading), rightAt(pair.value)
		if h < 0 || v < 0 {
			t.Fatalf("the authored layout is missing the %s block: heading at %d, pair at %d",
				pair.name, h, v)
		}
		if v != h+1 {
			t.Errorf("the %s pair is at row %d and its heading at row %d, want the pair "+
				"immediately under the heading", pair.name, v, h)
		}
		if got := l.Rows[h].Right.Label; got != pair.headingLabelWant {
			t.Errorf("the %s heading reads %q, want %q", pair.name, got, pair.headingLabelWant)
		}
		if got := l.Rows[v].Right.Label; got != "" {
			t.Errorf("the %s pair carries the label %q; the heading above it is the label",
				pair.name, got)
		}
	}

	f := panelFont()

	// cellValue finds a drawn cell by its label across BOTH columns. It has to
	// look in both: this fixture carries no character, so the statistics these
	// cells are paired with state nothing and the whole block slides into the
	// left column.
	cellValue := func(lines []panelLine, label string) (string, bool) {
		for _, ln := range lines {
			if ln.label == label {
				return ln.value, true
			}
			if ln.right && ln.rightLabel == label {
				return ln.rightValue, true
			}
		}
		return "", false
	}

	t.Run("a subject with a pool draws the pair, in the health pair's own form", func(t *testing.T) {
		s := panelSubjectFixture()
		s.Mana, s.MaxMana = 12, 40
		lines := panelLines(l, f, s)

		if _, ok := cellValue(lines, "MANA"); !ok {
			t.Fatal("a subject with a pool drew no MANA heading")
		}
		// The pair is the row after the heading, and it is unlabelled, so it
		// is found by position rather than by a label of its own.
		at := -1
		for i, ln := range lines {
			if ln.label == "MANA" || (ln.right && ln.rightLabel == "MANA") {
				at = i
			}
		}
		if at < 0 || at+1 >= len(lines) {
			t.Fatalf("the MANA heading is the last drawn row (%d of %d); nothing states the pair",
				at, len(lines))
		}
		next := lines[at+1]
		value := next.value
		if next.right && next.rightLabel == "" {
			value = next.rightValue
		}
		if value != "12/40" {
			t.Errorf("mana pair = %q, want %q — the health pair's own value form", value, "12/40")
		}
	})

	t.Run("a subject with no pool omits the pair AND its heading", func(t *testing.T) {
		s := panelSubjectFixture() // Mana and MaxMana both zero
		lines := panelLines(l, f, s)
		if v, ok := cellValue(lines, "MANA"); ok {
			t.Errorf("a subject with MaxMana == 0 drew a MANA heading over %q", v)
		}
		for _, ln := range lines {
			if ln.value == "0/0" || (ln.right && ln.rightValue == "0/0") {
				t.Errorf("a subject with MaxMana == 0 drew a mana pair anyway: %+v", ln)
			}
		}
		// The health block is untouched by mana's absence.
		if _, ok := cellValue(lines, "HEALTH"); !ok {
			t.Fatal("a subject with no mana pool drew no HEALTH heading either")
		}
	})
}

func TestPanelSubjectCarriesTheManaPair(t *testing.T) {
	v := &Viewer{
		entities: []MapEntity{{ID: 5, HP: 10, MaxHP: 10, Mana: 7, MaxMana: 30}},
		sel:      selection{5},
	}
	s, ok := v.panelSubject()
	if !ok {
		t.Fatal("no subject over a selection holding the one entity")
	}
	if s.Mana != 7 || s.MaxMana != 30 {
		t.Errorf("subject mana = %d/%d, want 7/30 — the pair off the MapEntity", s.Mana, s.MaxMana)
	}
}

func TestPanelStatesTheRecomputesExperienceAndFamilies(t *testing.T) {
	known := PanelSubject{Char: UnitCharacter{Known: true,
		Experience: 1593, Protection: [5]int{12, 12, 12, 12, 12}, Resistance: [5]int{1, 2, 3, 4, 5}}}

	if got, ok := panelText(known, PanelFieldExperience); !ok || got != "1593" {
		t.Errorf("experience = (%q, %v), want (%q, true)", got, ok, "1593")
	}
	if got, ok := panelText(known, PanelFieldProtection); !ok || got != "12 12 12 12 12" {
		t.Errorf("protection = (%q, %v), want (%q, true)", got, ok, "12 12 12 12 12")
	}
	if got, ok := panelText(known, PanelFieldResistance); !ok || got != "1 2 3 4 5" {
		t.Errorf("resistance = (%q, %v), want (%q, true)", got, ok, "1 2 3 4 5")
	}

	// AN UNKNOWN SUBJECT STATES NONE OF THEM — the existing character rows'
	// own gate, and the values above are still on the struct to prove the gate
	// and not a zero value is what withholds them.
	unknown := PanelSubject{Char: UnitCharacter{
		Experience: 1593, Protection: [5]int{12, 12, 12, 12, 12}, Resistance: [5]int{1, 2, 3, 4, 5}}}
	for _, f := range []PanelField{PanelFieldExperience, PanelFieldProtection, PanelFieldResistance} {
		if _, ok := panelText(unknown, f); ok {
			t.Errorf("field %d stated a value for a subject the load did not know", f)
		}
	}
}

// AC-7 — the skill row states six numbers, five of them zero, for a hero
// trained in exactly one slot, and it states them for a hero trained in none
// too: the row is never suppressed for being all zeros, which is the owner's
// own complaint and the whole of what this story fixes.
func TestPanelStatesTheSkillRowForOneSlotAndForNone(t *testing.T) {
	// Trained only in Pike (slot 4) at level 10 — the owner's own example,
	// typed as "0 0 0 0 10" before this story adds slot 0 to the row.
	oneSlot := PanelSubject{Char: UnitCharacter{Known: true,
		Skills: [PanelSkillSlots]int{0, 0, 0, 0, 10, 0}}}
	if got, ok := panelText(oneSlot, PanelFieldSkill); !ok || got != "0 0 0 0 10 0" {
		t.Errorf("skill row = (%q, %v), want (%q, true)", got, ok, "0 0 0 0 10 0")
	}

	// Trained in nothing at all — every slot zero, slot 0 included — still
	// states the row instead of omitting it.
	noSlots := PanelSubject{Char: UnitCharacter{Known: true}}
	if got, ok := panelText(noSlots, PanelFieldSkill); !ok || got != "0 0 0 0 0 0" {
		t.Errorf("skill row = (%q, %v), want (%q, true)", got, ok, "0 0 0 0 0 0")
	}

	// AN UNKNOWN SUBJECT STILL STATES NEITHER: Char.Known is the row's only
	// gate now, and this proves it still gates something — a row that stated
	// unconditionally, level or no level, would be a different bug from the
	// one this story fixes.
	unknown := PanelSubject{Char: UnitCharacter{Skills: [PanelSkillSlots]int{0, 0, 0, 0, 10, 0}}}
	if _, ok := panelText(unknown, PanelFieldSkill); ok {
		t.Error("the skill row stated a value for a subject the load did not know")
	}
}

func TestPanelSkillRowWidthByBand(t *testing.T) {
	for _, tc := range []struct {
		name string
		band CharacterBand
		want string
	}{
		{"unstated band draws the pre-0137 six-wide row", CharacterBandUnknown, "1 2 3 4 5 6"},
		{"person, six numbers, General included", CharacterBandPerson, "1 2 3 4 5 6"},
		{"creature, five numbers, General (slot 0) dropped", CharacterBandCreature, "2 3 4 5 6"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := PanelSubject{Char: UnitCharacter{Known: true, Band: tc.band,
				Skills: [PanelSkillSlots]int{1, 2, 3, 4, 5, 6}}}
			got, ok := panelText(s, PanelFieldSkill)
			if !ok {
				t.Fatal("a known character stated no skill row")
			}
			if got != tc.want {
				t.Errorf("skill row = %q, want %q", got, tc.want)
			}
		})
	}

	// AC-8 — an unknown CHARACTER (Char.Known == false) states nothing for
	// the skill row, or for any other character row, whatever its band: the
	// row's only gate is still Char.Known, exactly as before this story.
	for _, band := range []CharacterBand{CharacterBandUnknown, CharacterBandPerson, CharacterBandCreature} {
		s := PanelSubject{Char: UnitCharacter{Band: band, Skills: [PanelSkillSlots]int{1, 2, 3, 4, 5, 6}}}
		if _, ok := panelText(s, PanelFieldSkill); ok {
			t.Errorf("band %d: an unknown character stated a skill row", band)
		}
		for _, f := range []PanelField{PanelFieldBody, PanelFieldReaction, PanelFieldMind,
			PanelFieldSpirit, PanelFieldExperience, PanelFieldProtection, PanelFieldResistance} {
			if _, ok := panelText(s, f); ok {
				t.Errorf("band %d, field %d: an unknown character stated a value", band, f)
			}
		}
	}
}

func TestPanelSkillRowNotSuppressedForAllZerosOnEitherBand(t *testing.T) {
	for _, tc := range []struct {
		name string
		band CharacterBand
		want string
	}{
		{"person, six zeroes", CharacterBandPerson, "0 0 0 0 0 0"},
		{"creature, five zeroes", CharacterBandCreature, "0 0 0 0 0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := PanelSubject{Char: UnitCharacter{Known: true, Band: tc.band}}
			got, ok := panelText(s, PanelFieldSkill)
			if !ok {
				t.Fatal("an all-zero known character stated no skill row — the 0125 regression")
			}
			if got != tc.want {
				t.Errorf("skill row = %q, want %q", got, tc.want)
			}
		})
	}
}

// AC-8 — each byte of a name selects its own record.
func TestPanelNameSelectsARecordPerByte(t *testing.T) {
	f, l := panelFont(), panelInkLayout()
	l.Rows = []PanelRow{{Field: PanelFieldName}}

	// Two high bytes, then the same byte twice. Both names are the same length,
	// so a placement that drew one record per string, or one record for every
	// byte, would make these two pictures equal.
	distinct := composePanel(l, f, PanelSubject{Name: "\xc0\xc1"})
	repeated := composePanel(l, f, PanelSubject{Name: "\xc0\xc0"})
	if distinct == nil || repeated == nil {
		t.Fatal("composePanel returned nil for a named subject")
	}
	if distinct.Bounds() != repeated.Bounds() {
		t.Fatalf("the two names measure differently (%v vs %v); the comparison below would be vacuous",
			distinct.Bounds(), repeated.Bounds())
	}
	same := true
	for i := range distinct.Pix {
		if distinct.Pix[i] != repeated.Pix[i] {
			same = false
			break
		}
	}
	if same {
		t.Error(`"\xc0\xc1" and "\xc0\xc0" drew the same pixels: the second byte did not select its own record`)
	}
}

func TestPanelFitsItsContent(t *testing.T) {
	f := panelFont()

	for _, tc := range []struct{ name, subject string }{
		{"narrower than the minimum", "a"},
		{"wider than the minimum", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := panelInkLayout()
			s := PanelSubject{Name: tc.subject, HP: 1, MaxHP: 2, Cell: image.Pt(3, 4)}
			lines := panelLines(l, f, s)
			box := panelBox(l, f, lines)

			if box.X < l.MinWidth {
				t.Errorf("box width %d is under the minimum %d", box.X, l.MinWidth)
			}
			for i, ln := range lines {
				if r := ln.at.X + ln.width; r+l.Pad.X > box.X {
					t.Errorf("row %d reaches x=%d, past a box of %d — the minimum was applied as a ceiling",
						i, r, box.X)
				}
			}

			img := composePanel(l, f, s)
			if img == nil {
				t.Fatal("composePanel returned nil")
			}
			right, bottom, found := inkExtent(img)
			if !found {
				t.Fatal("the picture carries no text pixel at all")
			}
			if right >= box.X || bottom >= box.Y {
				t.Errorf("text reaches (%d, %d) in a box of %v", right, bottom, box)
			}
			// Nothing was clipped: the widest row's own last pixel is present,
			// which a box short of the content would have cut.
			widest := 0
			for _, ln := range lines {
				if r := ln.at.X + ln.width; r > widest {
					widest = r
				}
			}
			if right < widest-panelCellW {
				t.Errorf("the rightmost text pixel is at %d but the widest row runs to %d: content was clipped",
					right, widest)
			}
		})
	}
}

// panelAlignEdgeLayout is a minimal AlignValues layout with two right-only
// rows (1022 round 3, P fix for Defect B): row 0's right value is short
// (PanelFieldHealth) and row 1's is wide (PanelFieldCell, made wide by a
// large Cell). AlignValues gives every right-column value ONE shared edge
// (panel.go, layoutLines), and before this fix that edge was an unclamped
// maximum over every row's own natural width: a wide row 1 inflated the
// shared edge, and row 0's short value was right-aligned to that SAME
// inflated edge, dragging it out of the box too, though its own value alone
// would have fit easily. Fill/Border/label colours are set so inkExtent
// reads only the drawn text, the same discipline panelInkLayout uses for
// AuthoredPanelLayout.
func panelAlignEdgeLayout() PanelLayout {
	return PanelLayout{
		Size: image.Pt(40, 20), Pad: image.Pt(2, 2),
		Gap: 1, LabelGap: 2, ColumnGap: 2,
		Flow: true, AlignValues: true,
		Fill: color.RGBA{A: 0xff}, Border: color.RGBA{A: 0xff},
		LabelColor: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		ValueColor: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		Rows: []PanelRow{
			{Field: PanelFieldBlank, Right: &PanelCell{Field: PanelFieldHealth}},
			{Field: PanelFieldBlank, Right: &PanelCell{Field: PanelFieldCell}},
		},
	}
}

func panelAlignEdgeSubject() PanelSubject {
	return PanelSubject{HP: 1, MaxHP: 1, Cell: image.Pt(9999, 9999)}
}

// TestCompactPanelLayoutRightValuesFitTheBox proves the round-3 P fix for
// Defect B: a fixed-size AlignValues layout must not let a value's own text
// run past its own canvas, and a wide value in one row must not drag a SHORT
// value in another row off the canvas by way of AlignValues' shared edge.
// panelSubjectFixture's near-empty values never exercised this: HP 63/100
// and a two-digit Cell never approached AuthoredPanelLayout's own fit-to-
// content box, and CompactPanelLayout's fixed 160px canvas was never tested
// against a value wide enough to overflow it at all — every existing
// CompactPanelLayout test composes panelFont() glyphs (4px advance) against
// short fixture values, which stayed inside 160px by a wide margin regardless
// of AlignValues' own clamp.
func TestCompactPanelLayoutRightValuesFitTheBox(t *testing.T) {
	l, f, s := panelAlignEdgeLayout(), panelFont(), panelAlignEdgeSubject()

	img := composePanel(l, f, s)
	if img == nil {
		t.Fatal("composePanel returned nil for a font with records")
	}
	if got := img.Bounds().Size(); got != l.Size {
		t.Fatalf("box size = %v, want the layout's own fixed %v", got, l.Size)
	}

	right, _, found := inkExtent(img)
	if !found {
		t.Fatal("no ink at all painted for two rows each carrying a value")
	}
	if limit := l.Size.X - l.Pad.X; right >= limit {
		t.Fatalf("rightmost painted pixel = %d, want < %d (Size.X-Pad.X, the box's own right margin)", right, limit)
	}

	// Row 0's own short value must itself be placed inside the box. This is
	// the defect's own signature: the wide row 1 alone overflowing would
	// already fail the check above, but a fix that merely clipped row 1
	// while leaving AlignValues' shared edge uncapped would still place row
	// 0's short value outside the box, since it aligns to the same edge.
	lines := layoutLines(l, f, panelItems(l, s))
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	row0End := lines[0].at.X + lines[0].rightX + lines[0].rightValueX
	if row0End < 0 || row0End >= l.Size.X {
		t.Fatalf("row 0's own short value starts at x=%d, outside the box (0..%d)", row0End, l.Size.X)
	}
}

// panelWideCellW/H, panelWideAdvance and panelWideSpacing build a font whose
// per-character advance approximates the production font's own measured
// width rather than panelFont's deliberately compact 4px advance:
// townCharacterMode's own comment (townshell.go) records font1 measuring
// "STATS" (5 characters) at 57px and "DOLL" (4) at 52px, roughly 11-13px per
// character. This test uses a narrower 7px estimate, which is enough for a
// realistic value to reach CompactPanelLayout's 160px canvas without every
// right-column LABEL also reaching it — measured (below): with
// panelWideSubjectFixture, the rightmost painted pixel of the WHOLE card
// sits at x=155, one column inside the box's own right margin at this width.
// Records are otherwise built exactly as panelFont's: one painted pixel per
// record, distinguishable from every other.
const (
	panelWideCellW, panelWideCellH = 7, 6
	panelWideAdvance               = 7
	panelWideSpaceAdvance          = 4
	panelWideSpacing               = 1
)

func panelWideFont() *text.Font {
	f := &text.Font{Spacing: panelWideSpacing, Glyphs: make([]text.Glyph, 224)}
	for k := range f.Glyphs {
		g := text.Glyph{Width: panelWideCellW, Height: panelWideCellH,
			Pixels: make([]text.Pixel, panelWideCellW*panelWideCellH), Advance: panelWideAdvance}
		if k == 0 {
			g.Advance = panelWideSpaceAdvance
		} else {
			g.Pixels[(k%panelWideCellH)*panelWideCellW+(k/panelWideCellH)%panelWideCellW] =
				text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		f.Glyphs[k] = g
	}
	return f
}

// panelWideSubjectFixture is a levelled party member, not panelSubjectFixture's
// near-empty preview: three-digit HP, a two-digit combat spread and mid-range
// attributes and skills — the shape of subject that reaches
// CompactPanelLayout by way of a real Viewer or a played-through chargen
// build, rather than the pre-create default preview.
func panelWideSubjectFixture() PanelSubject {
	return PanelSubject{
		ID: 9, Name: "Levelled Fighter", HP: 187, MaxHP: 220,
		Combat: UnitCombat{
			Known:      true,
			DamageBase: 24, DamageSpread: 18,
			ToHit: 58, Defence: 62, Absorption: 47,
		},
		Char: UnitCharacter{
			Known: true, Name: "Levelled Fighter", Band: CharacterBandPerson,
			Body: 9, Reaction: 8, Mind: 5, Spirit: 4,
			Skills:     [PanelSkillSlots]int{2, 5, 4, 4, 3, 1},
			Protection: [5]int{8, 2, 9, 4, 7},
			Resistance: [5]int{2, 6, 1, 9, 8},
			Sight:      8,
		},
		Speed: 9,
	}
}

// TestCompactPanelLayoutTruncatesAWideValueRatherThanOverflow proves the
// round-3 P fix for Defect B through the production card itself, not a
// synthetic minimal layout: CompactPanelLayout(nil), panelWideFont and
// panelWideSubjectFixture's DEFENSE row (right label "DEFENSE", right value
// Combat.Defence = 62) is the row that overflows at this font's width
// without the fix, its own label alone (7 characters) leaving too little
// room for a two-digit value beside it.
func TestCompactPanelLayoutTruncatesAWideValueRatherThanOverflow(t *testing.T) {
	l, f, s := CompactPanelLayout(nil), panelWideFont(), panelWideSubjectFixture()

	card := RenderCharacterPanel(l, f, s)
	if card == nil {
		t.Fatal("RenderCharacterPanel returned nil for a font with records")
	}
	if got := card.Bounds().Size(); got != l.Size {
		t.Fatalf("card size = %v, want the layout's own fixed %v", got, l.Size)
	}

	lines := layoutLines(l, f, panelItems(l, s))
	var defense *panelLine
	for i := range lines {
		if lines[i].right && lines[i].rightLabel == "DEFENSE" {
			defense = &lines[i]
		}
	}
	if defense == nil {
		t.Fatal("no DEFENSE row in CompactPanelLayout's own Rows")
	}
	if defense.rightValue == "" {
		t.Fatal("DEFENSE row's value truncated to nothing; want it shortened, not dropped, for this width")
	}
	if defense.rightValue == "62" {
		t.Fatal("DEFENSE row's value is the untruncated \"62\"; this test's own font/subject choice no longer forces an overflow, so it no longer exercises the fix")
	}
	rvw, _ := f.Measure(defense.rightValue)
	if end := defense.at.X + defense.rightX + defense.rightValueX + rvw; end > l.Size.X-l.Pad.X {
		t.Fatalf("DEFENSE row's own truncated value still ends at x=%d, past the box's own right margin %d", end, l.Size.X-l.Pad.X)
	}

	// The whole card, not only DEFENSE: every LABEL in CompactPanelLayout's
	// own Rows is a fixed, authored string this fix does not shorten, so a
	// wide left column (BLUDGEON/SHOOTING) pushing the right column's own
	// origin far enough right could still run a label past the canvas even
	// with every VALUE correctly bounded. At this font's width it does not:
	// the rightmost painted pixel of the whole card sits inside the box's
	// own right margin.
	ink := l
	ink.Fill, ink.Border = color.RGBA{A: 0xff}, color.RGBA{A: 0xff}
	ink.LabelColor = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	ink.ValueColor = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	inkImg := composePanel(ink, f, s)
	right, _, found := inkExtent(inkImg)
	if !found {
		t.Fatal("no ink at all painted for a subject with real combat and character numbers")
	}
	if limit := l.Size.X - l.Pad.X; right >= limit {
		t.Fatalf("rightmost painted pixel of the whole card = %d, want < %d (Size.X-Pad.X)", right, limit)
	}
}

// AC-10 — the two background forms and the two row modes.
func TestPanelBackgroundAndRowModes(t *testing.T) {
	f, s := panelFont(), panelSubjectFixture()

	t.Run("a supplied background is drawn as given", func(t *testing.T) {
		bg := image.NewRGBA(image.Rect(0, 0, 40, 20))
		mark := color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}
		for y := 0; y < 20; y++ {
			for x := 0; x < 40; x++ {
				bg.SetRGBA(x, y, mark)
			}
		}
		l := AuthoredPanelLayout()
		l.Background, l.Size, l.Rows = bg, image.Pt(40, 20), nil
		img := composePanel(l, f, s)
		if img == nil {
			t.Fatal("composePanel returned nil")
		}
		for _, p := range []image.Point{{X: 0, Y: 0}, {X: 39, Y: 19}, {X: 20, Y: 10}} {
			if got := img.RGBAAt(p.X, p.Y); got != mark {
				t.Fatalf("pixel %v is %v, want the background's own %v", p, got, mark)
			}
		}
	})

	t.Run("no background is the authored fill under a border", func(t *testing.T) {
		l := AuthoredPanelLayout()
		l.Size, l.Rows = image.Pt(30, 20), nil
		img := composePanel(l, f, s)
		if img == nil {
			t.Fatal("composePanel returned nil")
		}
		for _, p := range []image.Point{{X: 0, Y: 0}, {X: 29, Y: 0}, {X: 0, Y: 19}, {X: 29, Y: 19}, {X: 15, Y: 0}} {
			if got := img.RGBAAt(p.X, p.Y); got != l.Border {
				t.Errorf("edge pixel %v is %v, want the border %v", p, got, l.Border)
			}
		}
		for _, p := range []image.Point{{X: 1, Y: 1}, {X: 28, Y: 18}, {X: 15, Y: 10}} {
			if got := img.RGBAAt(p.X, p.Y); got != l.Fill {
				t.Errorf("interior pixel %v is %v, want the fill %v", p, got, l.Fill)
			}
		}
	})

	t.Run("flowing rows stack from the pad", func(t *testing.T) {
		l := AuthoredPanelLayout()
		lines := panelLines(l, f, s)
		for i, ln := range lines {
			want := image.Pt(l.Pad.X, l.Pad.Y+i*(f.Height()+l.Gap))
			if ln.at != want {
				t.Errorf("flowing row %d is at %v, want %v", i, ln.at, want)
			}
		}
	})

	t.Run("placed rows land at their own offsets", func(t *testing.T) {
		l := AuthoredPanelLayout()
		l.Flow = false
		l.Rows = []PanelRow{
			{Field: PanelFieldHealth, Label: "HP", At: image.Pt(7, 41)},
			{Field: PanelFieldCell, Label: "CELL", At: image.Pt(3, 5)},
		}
		lines := panelLines(l, f, s)
		if len(lines) != 2 {
			t.Fatalf("states %d row(s), want 2", len(lines))
		}
		if lines[0].at != image.Pt(7, 41) || lines[1].at != image.Pt(3, 5) {
			t.Fatalf("placed rows are at %v and %v, want (7,41) and (3,5)", lines[0].at, lines[1].at)
		}
	})

	t.Run("a row placed outside a fixed box paints nothing", func(t *testing.T) {
		l := panelInkLayout()
		l.Flow = false
		l.Size = image.Pt(40, 20)
		l.Rows = []PanelRow{{Field: PanelFieldName, At: image.Pt(1000, 1000)}}
		img := composePanel(l, f, s)
		if img == nil {
			t.Fatal("composePanel returned nil")
		}
		if _, _, found := inkExtent(img); found {
			t.Error("a row placed far outside the box still painted into it")
		}
	})
}

// The panel's corner arithmetic, exact and unclamped (part of AC-11's origin).
func TestPanelOrigin(t *testing.T) {
	l := AuthoredPanelLayout()
	area, box := image.Pt(800, 600), image.Pt(200, 90)

	for _, tc := range []struct {
		corner PanelCorner
		want   image.Point
	}{
		{PanelTopLeft, image.Pt(12, 12)},
		{PanelTopRight, image.Pt(800-200-12, 12)},
		{PanelBottomLeft, image.Pt(12, 600-90-12)},
		{PanelBottomRight, image.Pt(800-200-12, 600-90-12)},
	} {
		l.Corner = tc.corner
		if got := panelOrigin(l, area, box); got != tc.want {
			t.Errorf("corner %d: origin = %v, want %v", tc.corner, got, tc.want)
		}
	}

	// A box larger than the area is not repositioned or shrunk.
	l.Corner = PanelBottomLeft
	if got := panelOrigin(l, image.Pt(50, 40), box); got != image.Pt(12, 40-90-12) {
		t.Errorf("an oversized box moved: origin = %v", got)
	}
}

// TestTheUnitPanelSwallowsItsOwnPresses is FR-4b, and it was written at the
// evidence stage: the swallow shipped with no criterion of its own, so the
// defect it closes — a left click over an opaque 400-pixel box walking the
// party to whatever cell lay under it — had nothing watching it.
//
// THE CONTROL IS THE POINT. A press one pixel outside the box still reaches the
// map and still issues an order; without that half, a viewer that issued no
// orders at all would pass.
func TestTheUnitPanelSwallowsItsOwnPresses(t *testing.T) {
	a := inventoryTestApp(t)
	v := a.flow.viewer
	layoutViewport(v, MenuWindowW, MenuWindowH)
	v.SetFont(panelFont())
	v.SetPanelLayout(AuthoredPanelLayout())
	s := panelSubjectFixture()
	v.SetEntities([]MapEntity{{ID: s.ID, Cell: image.Pt(3, 3), Name: s.Name, HP: s.HP, MaxHP: s.MaxHP}})
	v.sel = selection{s.ID}

	box, ok := v.panelRect()
	if !ok {
		t.Fatal("setup: no unit panel with a subject selected")
	}
	cx, cy := (box.Min.X+box.Max.X)/2, (box.Min.Y+box.Max.Y)/2

	if ords, run := tapAt(v, cx, cy); run || len(ords) != 0 {
		t.Errorf("a left click inside the panel %v issued %d order(s) (ok=%v)", box, len(ords), run)
	}

	before := slices.Clone(v.sel)
	v.command(appInput{PrimaryPressed: true, CursorX: cx, CursorY: cy})
	v.command(appInput{PrimaryReleased: true, CursorX: cx, CursorY: cy})
	if !slices.Equal(v.sel, before) {
		t.Errorf("a tap inside the panel moved the selection to %v, want %v", v.sel, before)
	}
	if v.held {
		t.Error("held is still raised after a release the panel swallowed")
	}

	// The control: one pixel to the LEFT of the panel's own edge, at its
	// vertical centre — clear of the two bars along the bottom, which stop at
	// the column this box stands in.
	out := box.Min.X - 1
	if v.panelCaptures(out, cy) {
		t.Fatalf("setup: (%d,%d) is still the panel's own pixel", out, cy)
	}
	if ords, run := tapAt(v, out, cy); !run || len(ords) != 1 {
		t.Errorf("a left click one pixel outside the panel issued %d order(s) (ok=%v), want 1",
			len(ords), run)
	}
}

// TestCompactPanelLayoutCentredNameDoesNotMoveTheAlignedColumn proves the
// accumulation half of 1027 B4: a centred row neither takes AlignValues'
// shared left edge nor sets it. The placement half is witnessed by the name's
// own pen below; the accumulation half needs a centred value WIDER than every
// ordinary value's own natural edge, which the shipped party never supplies
// (measured at the production font: the widest ordinary left cell reaches 58
// pixels and a shipped name's ink is 35 wide), so a synthetic name supplies
// it here. Without the accumulation skip, the name raises leftEdge and every
// ordinary value in the card moves right by the difference.
func TestCompactPanelLayoutCentredNameDoesNotMoveTheAlignedColumn(t *testing.T) {
	l, f := CompactPanelLayout(nil), panelWideFont()

	short := panelWideSubjectFixture()
	short.Name, short.Char.Name = "Ann", "Ann"
	long := panelWideSubjectFixture()
	long.Name, long.Char.Name = "Wide Centred Name", "Wide Centred Name"

	shortLines := layoutLines(l, f, panelItems(l, short))
	longLines := layoutLines(l, f, panelItems(l, long))
	if len(shortLines) != len(longLines) || len(shortLines) < 2 {
		t.Fatalf("line counts = %d and %d, want the same count above 1", len(shortLines), len(longLines))
	}
	if longLines[0].value != long.Name {
		t.Fatalf("the long name was shortened to %q; this fixture no longer measures an untruncated centred value", longLines[0].value)
	}

	// leftEdge itself: with AlignValues on, every non-empty ordinary value
	// ends at the shared edge, so the maximum of their own ends is it.
	edge := 0
	for _, ln := range shortLines[1:] {
		if ln.value == "" || ln.field == PanelFieldWeight || ln.field == PanelFieldExperience {
			continue
		}
		vw, _ := f.Measure(ln.value)
		if e := ln.valueX + vw; e > edge {
			edge = e
		}
	}
	nameW, _ := f.Measure(longLines[0].value)
	if nameW <= edge {
		t.Fatalf("the long name measures %d against a shared edge of %d; this fixture no longer forces the accumulation case", nameW, edge)
	}

	for i := range shortLines[1:] {
		a, b := shortLines[i+1], longLines[i+1]
		if a.valueX != b.valueX {
			t.Fatalf("row %d's value pen moved from x=%d to x=%d when only the centred name changed", i+1, a.valueX, b.valueX)
		}
		if a.rightValueX != b.rightValueX {
			t.Fatalf("row %d's right value pen moved from x=%d to x=%d when only the centred name changed", i+1, a.rightValueX, b.rightValueX)
		}
	}

	// Each name's own pen, computed from the card's width rather than from
	// the production expression that places it. Both widths are checked: a
	// name NARROWER than the shared edge is the one the placement skip
	// protects, since without it the alignment shift moves the short name
	// out to that edge; a name wider than the edge is never shifted and so
	// cannot witness that half.
	shortW, _ := f.Measure(shortLines[0].value)
	for _, c := range []struct {
		what string
		line panelLine
		w    int
	}{
		{"short", shortLines[0], shortW},
		{"long", longLines[0], nameW},
	} {
		got := 2*(c.line.at.X+c.line.valueX) + c.w
		if want := l.Size.X; got < want-2 || got > want+2 {
			t.Fatalf("the centred %s name spans a doubled centre of %d, want the card's own %d (+/-2)", c.what, got, want)
		}
	}
}

// TestCompactPanelHasFixedColumnsAndTheRequestedVerticalCadence pins the
// owner-observed card composition: BODY/HEALTH begins one full row below the
// name, both columns keep one geometry regardless of value widths, and the
// final SIGHT/SPEED rows are centred full-width rows.
func TestCompactPanelHasFixedColumnsAndTheRequestedVerticalCadence(t *testing.T) {
	l, f := CompactPanelLayout(nil), panelWideFont()
	subj := panelWideSubjectFixture()
	lines := layoutLines(l, f, panelItems(l, subj))
	if len(lines) < 3 {
		t.Fatalf("got %d lines, want the compact card population", len(lines))
	}
	stride := f.Height() + l.Gap
	if got, want := lines[1].at.Y-lines[0].at.Y, 2*stride; got != want {
		t.Fatalf("BODY row delta from name = %d, want one blank row (%d)", got, want)
	}
	wantRightX := (l.Size.X-2*l.Pad.X-l.ColumnGap)/2 + l.ColumnGap
	for i, ln := range lines {
		if ln.right && ln.rightX != wantRightX {
			t.Fatalf("row %d rightX = %d, want fixed %d", i, ln.rightX, wantRightX)
		}
	}

	wider := subj
	wider.HP, wider.MaxHP, wider.Char.Experience = 1234567, 7654321, 987654321
	wideLines := layoutLines(l, f, panelItems(l, wider))
	for i := range lines {
		if lines[i].right && wideLines[i].rightX != lines[i].rightX {
			t.Fatalf("row %d moved right column from %d to %d after values widened", i, lines[i].rightX, wideLines[i].rightX)
		}
	}
	for _, ln := range lines[len(lines)-2:] {
		if !strings.Contains(ln.label, "SIGHT") && !strings.Contains(ln.label, "SPEED") {
			t.Fatalf("final centred row label = %q, want SIGHT or SPEED", ln.label)
		}
		if centre := 2*ln.at.X + ln.width; centre < l.Size.X-2 || centre > l.Size.X+2 {
			t.Fatalf("%s row doubled centre = %d, want %d (+/-2)", ln.label, centre, l.Size.X)
		}
	}
}

func TestTheCompactCardDrawsTheWeightRowBetweenTheResistancesAndTheTotals(t *testing.T) {
	l, f := CompactPanelLayout(nil), panelWideFont()
	s := panelWideSubjectFixture()
	s.Weight, s.WeightKnown = 35, true

	stmt := PanelStatement(l, s)
	row := -1
	for i, line := range stmt {
		if strings.HasPrefix(line, "WEIGHT ") {
			row = i
		}
	}
	if row < 1 || row+1 >= len(stmt) {
		t.Fatalf("the card states no WEIGHT row with neighbours on both sides:\n%v", stmt)
	}
	if got := stmt[row]; got != "WEIGHT 3.5" {
		t.Errorf("the weight row reads %q, want \"WEIGHT 3.5\"", got)
	}
	if got := stmt[row-1]; !strings.HasPrefix(got, "SHOOTING ") {
		t.Errorf("the row above WEIGHT is %q, want the last skill/resistance row", got)
	}
	if got := stmt[row+1]; !strings.HasPrefix(got, "XP ") {
		t.Errorf("the row below WEIGHT is %q, want the totals block", got)
	}

	// The two composed cards, differing in one value and nothing else.
	other := s
	other.Weight = 45
	a, b := RenderCharacterPanel(l, f, s), RenderCharacterPanel(l, f, other)
	if a == nil || b == nil {
		t.Fatal("RenderCharacterPanel painted nothing")
	}
	if a.Bounds() != b.Bounds() {
		t.Fatalf("the two cards are %v and %v; only one value changed", a.Bounds(), b.Bounds())
	}
	top, bottom, any := panelDiffRows(a, b)
	if !any {
		t.Fatal("the two cards are identical; the weight value is not drawn at all")
	}

	// Where that ink sits INSIDE a line, measured by drawing the same two
	// strings onto a scratch image at a known origin. It is a property of the
	// font and of the two strings, and of nothing this test is pinning.
	const scratchAt = 40
	inkTop, inkBottom := panelScratchDiffRows(t, f, "3.5", "4.5", scratchAt)

	weightY := -1
	for _, ln := range layoutLines(l, f, panelItems(l, s)) {
		if ln.label == "WEIGHT" {
			weightY = ln.at.Y
			break
		}
	}
	if weightY < 0 {
		t.Fatal("the laid-out compact card has no WEIGHT row")
	}
	pitch := f.Height() + l.Gap
	wantTop := weightY + inkTop
	wantBottom := weightY + inkBottom
	if top != wantTop || bottom != wantBottom {
		t.Errorf("the weight value is drawn on rows %d..%d, want %d..%d "+
			"(pad %d, row %d of %d, pitch %d, ink %d..%d)",
			top, bottom, wantTop, wantBottom, l.Pad.Y, row, len(stmt), pitch, inkTop, inkBottom)
	}
}

// panelDiffRows is the first and last y at which two images of one size differ,
// and whether they differ at all.
func panelDiffRows(a, b *image.RGBA) (top, bottom int, any bool) {
	r := a.Bounds()
	top, bottom = r.Max.Y, r.Min.Y
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				any = true
				if y < top {
					top = y
				}
				if y > bottom {
					bottom = y
				}
				break
			}
		}
	}
	return top, bottom, any
}

// panelScratchDiffRows draws two strings at one origin on two scratch images
// and reports the first and last row they differ on, relative to that origin.
func panelScratchDiffRows(t *testing.T, f *text.Font, first, second string, at int) (top, bottom int) {
	t.Helper()
	r := image.Rect(0, 0, 200, at+4*f.Height()+8)
	a, b := image.NewRGBA(r), image.NewRGBA(r)
	ink := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	f.Draw(a, first, 0, at, ink)
	f.Draw(b, second, 0, at, ink)
	top, bottom, any := panelDiffRows(a, b)
	if !any {
		t.Fatalf("this font draws %q and %q identically; the fixture measures nothing", first, second)
	}
	return top - at, bottom - at
}

// TestTheCompactCardKeepsSightAndSpeedOnAShorterBody pins the foot of the card:
// a body shorter than the reference gives up the difference to the last two
// rows, so both stay on the writing surface.
func TestTheCompactCardKeepsSightAndSpeedOnAShorterBody(t *testing.T) {
	f := panelWideFont()
	rowsFor := func(height int) (sight, speed int) {
		bg := image.NewRGBA(image.Rect(0, 0, 160, height))
		for y := 0; y < height; y++ {
			for x := 0; x < 160; x++ {
				c := color.RGBA{R: 33, G: 60, B: 33, A: 255}
				if x < 4 || x >= 156 || y < 4 || y >= height-4 {
					c = color.RGBA{R: 90, G: 70, B: 20, A: 255}
				}
				bg.SetRGBA(x, y, c)
			}
		}
		l := CompactPanelLayout(bg)
		l.Background = characterCardBackground(bg)
		for _, ln := range layoutLines(l, f, panelItems(l, panelWideSubjectFixture())) {
			switch ln.field {
			case PanelFieldSight:
				sight = ln.at.Y
			case PanelFieldMoveSpeed:
				speed = ln.at.Y
			}
		}
		return sight, speed
	}
	tallSight, tallSpeed := rowsFor(characterCardBodyHeight)
	shortSight, shortSpeed := rowsFor(characterCardBodyHeight - 4)
	if tallSight == 0 || tallSpeed == 0 {
		t.Fatalf("the card lays out no SIGHT/SPEED rows: %d %d", tallSight, tallSpeed)
	}
	if tallSight-shortSight != 4 || tallSpeed-shortSpeed != 4 {
		t.Fatalf("a 4-pixel shorter body moved SIGHT by %d and SPEED by %d, want 4 each",
			tallSight-shortSight, tallSpeed-shortSpeed)
	}
}

// A creature with no known spell states no spellcaster caption even with an
// experience value; one holding a spell does, and a person is unaffected.
func TestPanelSpellcasterCaptionNeedsASpellForACreature(t *testing.T) {
	s := PanelSubject{DetailLevel: 7, DetailSet: true, Char: UnitCharacter{Band: CharacterBandCreature},
		OriginalPanel: OriginalPanelActor{Known: true, XPValue: 9}}
	if _, ok := panelText(s, PanelFieldSpellcaster); ok {
		t.Fatal("a plain creature states the caption")
	}
	s.KnownSpells = 1 << 9
	if _, ok := panelText(s, PanelFieldSpellcaster); !ok {
		t.Fatal("a creature holding a spell omits the caption")
	}
	s.KnownSpells, s.Char.Band = 0, CharacterBandPerson
	if _, ok := panelText(s, PanelFieldSpellcaster); !ok {
		t.Fatal("a person lost the caption")
	}
}

func TestPanelLoadRowIsStatedOnlyForPlayerCharacters(t *testing.T) {
	l := CompactPanelLayout(nil)
	base := PanelSubject{DetailLevel: 7, DetailSet: true, Weight: 35, WeightKnown: true,
		Char: UnitCharacter{Known: true, Band: CharacterBandPerson}}
	rowOf := func(s PanelSubject, f PanelField) (int, bool) {
		for i, it := range panelItems(l, s) {
			if it.field == f || (it.right && it.rightField == f) {
				return i, true
			}
		}
		return 0, false
	}
	hero := base
	hero.OriginalPanel = OriginalPanelActor{Known: true, Flags: 0x1, XPValue: 9}
	if _, ok := rowOf(hero, PanelFieldWeight); !ok {
		t.Fatal("a player character omits the load row")
	}
	other := base
	other.OriginalPanel = OriginalPanelActor{Known: true, XPValue: 9}
	if _, ok := rowOf(other, PanelFieldWeight); ok {
		t.Fatal("a unit that is not a player character states a load")
	}
	spellRow, ok := rowOf(other, PanelFieldSpellcaster)
	if !ok {
		t.Fatal("the spellcaster caption is absent for a person who is not a player character")
	}
	it := panelItems(l, other)[spellRow]
	if it.right || it.field != PanelFieldSpellcaster {
		t.Fatalf("the caption did not take the load row's left position: %+v", it)
	}
	creature := other
	creature.Char.Band = CharacterBandCreature
	creature.KnownSpells = 1 << 9
	if _, ok := rowOf(creature, PanelFieldWeight); ok {
		t.Fatal("a creature states a load")
	}
	if _, ok := rowOf(creature, PanelFieldSpellcaster); !ok {
		t.Fatal("a creature holding a spell omits the caption")
	}
}
