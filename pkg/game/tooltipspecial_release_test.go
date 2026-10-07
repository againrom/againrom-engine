package game

// Release witnesses for the spellbook caption, generator attribute-row and
// map-list hover texts. Expectations come from the installed text tables and
// the claim formulas; each has a loss control. PNGs go to
// AGAINROM_TOOLTIP_SHOTS, or a temporary directory.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func tooltipShotDir(t *testing.T) string {
	t.Helper()
	if dir := os.Getenv("AGAINROM_TOOLTIP_SHOTS"); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	return t.TempDir()
}

func writeTooltipShot(t *testing.T, name string, pic *image.RGBA) {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, pic); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tooltipShotDir(t), name+".png"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func samePicture(a, b *image.RGBA) bool {
	return a != nil && b != nil && a.Bounds() == b.Bounds() && bytes.Equal(a.Pix, b.Pix)
}

func cloneRGBA(p *image.RGBA) *image.RGBA {
	return &image.RGBA{Pix: append([]uint8(nil), p.Pix...), Stride: p.Stride, Rect: p.Rect}
}

// captionFormula is the claim's accessor table.
func captionFormula(id uint16, level int) (label int, text string, ok bool) {
	switch id {
	case 24:
		return 182, fmt.Sprintf("%d", level/15+1), true
	case 7:
		return 182, fmt.Sprintf("%d", -(level/15 + 1)), true
	case 5, 16, 10, 22:
		if level/2 == 0 {
			return 0, "", false
		}
		return 183, fmt.Sprintf("+%d", level/2), true
	case 12:
		return 184, fmt.Sprintf("%d", -(level/30 + 1)), true
	case 23:
		return 185, fmt.Sprintf("+%d%%", 4*level/5+20), true
	case 14:
		return 186, fmt.Sprintf("%d", min(level/20+2, 7)), true
	case 18:
		return 217, fmt.Sprintf("%d", level/10+3), true
	}
	return 0, "", false
}

func TestReleaseSpellbookCaptionHoverIsBuiltFromTheClaim(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	mainTable := LoadTextTable(f.Archives.Containers, MainTextPath)
	bookTable := LoadTextTable(f.Archives.Containers, SpellBookNamesTextPath)
	if mainTable == nil || bookTable == nil {
		t.Fatal("install carries no main.txt or spells.txt")
	}
	label := func(i int) string {
		s, ok := mainTable.At(i)
		if !ok {
			t.Fatalf("main.txt has no row %d", i)
		}
		return s
	}
	cellIDs := [...]uint16{1, 2, 3, 4, 5, 23, 24, 16, 15, 14, 13, 12, 6, 7, 8, 9, 10, 25, 26, 22, 21, 20, 19, 18}
	var known uint32
	for _, id := range []uint16{23, 14, 5, 24, 12, 18, 7} {
		known |= 1 << id
	}
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	party := []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true, Class: 0x18,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero, KnownSpells: known,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000}}}
	a := f.App("spellbook caption hover")
	defer a.StopAudio()
	a.Layout(1024, 768)
	a.SetTooltipDelayPreference(0, nil)
	a.SetTooltipFont(f.Font.Value())
	if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	id := f.live.mission.ids[0]
	inspectionCentre(f.live, 29, 50)
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.HeadlessSpellPoint(23); err != nil {
		if err := a.HeadlessKey("book"); err != nil {
			t.Fatal(err)
		}
	}
	entity, ok := f.live.entity(id)
	if !ok {
		t.Fatal("selected caster absent")
	}
	bounds := image.Rect(0, 0, 1024, 768)
	compose := func(lines []string) *image.RGBA {
		pic, _, _ := ui.ComposeTooltipHint(lines, f.Font.Value(), image.Point{}, bounds)
		return pic
	}
	captioned := 0
	for cell, spell := range cellIDs {
		if known&(1<<spell) == 0 {
			continue
		}
		var rule sim.SpellRule
		for _, r := range f.live.world.Spells() {
			if r.ID == spell {
				rule = r
			}
		}
		if rule.ID != spell {
			t.Fatalf("installed rules have no spell %d", spell)
		}
		c := sim.SpellCharacteristicsFor(sim.Rules{}, entity, rule)
		level := min(max(int(entity.Skill[rule.School])+int(entity.Mind)-30, 0), 100)
		if int(c.Power) != level {
			t.Fatalf("spell %d: power %d, claim level %d", spell, c.Power, level)
		}
		row, _ := bookTable.At(cell)
		lines := []string{strings.SplitN(row, "#", 2)[0], fmt.Sprintf("%s: %d", label(117), c.ManaCost)}
		// The record's damage bytes from the installed columns at the record
		// power (TEXT-096), formed here and not by the production helper.
		recordPower := (int(entity.Skill[rule.School]) + int(entity.Mind) - 30) & 0xff
		if recordPower > 100 {
			recordPower = 100
		}
		factor := float64(recordPower)/30.0 + 1.0
		var damageLo, damageSpread int
		if rule.DamageMin > 0 {
			damageLo = int(float64(rule.DamageMin)*factor) & 0xff
		}
		if rule.DamageMax > 0 {
			damageSpread = int(float64(rule.DamageMax)*factor-float64(damageLo)) & 0xff
		}
		if damageSpread+damageLo != 0 {
			if damageSpread == 0 {
				lines = append(lines, fmt.Sprintf("%s: %d", label(118), damageLo))
			} else {
				lines = append(lines, fmt.Sprintf("%s: %d-%d", label(118), damageLo, damageLo+damageSpread))
			}
		}
		if c.Range != 0 {
			lines = append(lines, fmt.Sprintf("%s: %d", label(123), c.Range))
		}
		if c.Duration != 0 {
			lines = append(lines, fmt.Sprintf("%s: %5.1f", label(124), float64(c.Duration)*0.0625))
		}
		withoutCaption := append([]string(nil), lines...)
		n, value, has := captionFormula(spell, level)
		if has {
			lines = append(lines, fmt.Sprintf("%s: %s", label(n), value))
			captioned++
		}

		x, y, err := a.HeadlessSpellPoint(uint32(spell))
		if err != nil {
			t.Fatalf("spell %d: %v", spell, err)
		}
		if err := a.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		state, pic := a.HeadlessTooltip()
		if state.Target != fmt.Sprintf("spell/%d", spell) || !state.Visible || pic == nil {
			t.Fatalf("spell %d hover = %+v", spell, state)
		}
		if !samePicture(pic, compose(lines)) {
			t.Fatalf("spell %d: hover differs from the claim-built lines %q", spell, lines)
		}
		if has {
			// Loss controls: no caption, and another level.
			if samePicture(pic, compose(withoutCaption)) {
				t.Fatalf("spell %d: the popup does not show its caption", spell)
			}
			if n2, shifted, ok := captionFormula(spell, min(level+30, 100)); ok && shifted != value {
				moved := append(append([]string(nil), withoutCaption...), fmt.Sprintf("%s: %s", label(n2), shifted))
				if samePicture(pic, compose(moved)) {
					t.Fatalf("spell %d: the popup does not follow the level", spell)
				}
			}
		}
		writeTooltipShot(t, fmt.Sprintf("spellbook-%d", spell), pic)
	}
	if captioned < 5 {
		t.Fatalf("only %d of the known spells carry a caption; the witness is vacuous", captioned)
	}
}

func costTotal(n int) int { return int(0.349*math.Pow(1.15, float64(n-1)) + 0.5) }

func TestReleaseGeneratorAttributeRowsHover(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	mainTable := LoadTextTable(f.Archives.Containers, MainTextPath)
	if mainTable == nil {
		t.Fatal("install carries no main.txt")
	}
	row := func(i int) string {
		s, ok := mainTable.At(i)
		if !ok {
			t.Fatalf("main.txt has no row %d", i)
		}
		return s
	}
	a := f.App("generator attribute hover")
	defer a.StopAudio()
	a.Layout(640, 480)
	a.SetTooltipDelayPreference(0, nil)
	a.SetTooltipFont(f.Font.Value())
	c := ui.NewChargen(f.ChargenSetup())
	c.CloseTip()
	if err := a.OpenChargen(c, func(res ui.ChargenResult) (ui.MapOpener, error) { return f.NewGameOpener(10, res), nil }); err != nil {
		t.Fatal(err)
	}
	c.Forward()
	if c.Stage() != ui.DetailedStage {
		t.Fatal("generator did not reach the detailed page")
	}
	bounds := image.Rect(0, 0, 640, 480)
	hover := func(p image.Point) *image.RGBA {
		t.Helper()
		if err := a.HeadlessPointer("hover", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		state, pic := a.HeadlessTooltip()
		if !state.Visible || pic == nil {
			t.Fatalf("no popup at %v: %+v", p, state)
		}
		return pic
	}
	check := func(what string, p image.Point, lines []string) *image.RGBA {
		t.Helper()
		pic := hover(p)
		want, _, ok := ui.ComposeTooltipHint(lines, f.Font.Value(), p, bounds)
		if !ok || !samePicture(pic, want) {
			t.Fatalf("%s: hover differs from the claim-built lines %q", what, lines)
		}
		return pic
	}
	centre := func(r image.Rectangle) image.Point { return r.Min.Add(r.Size().Div(2)) }
	for stat := 0; stat < 4; stat++ {
		plate, value, lower, raise, ok := ui.DetailedAttributeBoxes(stat)
		if !ok {
			t.Fatal("no attribute row", stat)
		}
		res, _ := c.Result()
		current := res.Stats[stat]
		check(fmt.Sprintf("row %d plate", stat), plate.Min.Add(image.Pt(12, 4)), strings.Split(row(155+stat), "#"))
		pic := check(fmt.Sprintf("row %d value", stat), centre(value), []string{fmt.Sprintf("%s = %d", row(15+stat), current)})
		writeTooltipShot(t, fmt.Sprintf("attribute-%d-value", stat), pic)
		before := cloneRGBA(pic)
		next := -(costTotal(current+1) - costTotal(current))
		pic = check(fmt.Sprintf("row %d raise", stat), centre(raise), []string{fmt.Sprintf("%+d", next)})
		writeTooltipShot(t, fmt.Sprintf("attribute-%d-cost", stat), pic)
		refund := costTotal(current) - costTotal(current-1)
		check(fmt.Sprintf("row %d lower", stat), centre(lower), []string{fmt.Sprintf("%+d", refund)})

		// Loss control: one more point moves the value text.
		if c.AdjustStat(stat, 1) {
			if samePicture(before, hover(centre(value))) {
				t.Fatalf("row %d: the value text does not follow the value", stat)
			}
			c.AdjustStat(stat, -1)
		}
	}
}

func TestReleaseMapListHoverChoosesByColumn(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	dialogs := LoadTextTable(f.Archives.Containers, DialogsTextPath)
	if dialogs == nil {
		t.Fatal("install carries no dialogs.txt")
	}
	caption := func(i int) string {
		s, ok := dialogs.At(i)
		if !ok {
			t.Fatalf("dialogs.txt has no row %d", i)
		}
		return s
	}
	a := f.App("map list hover")
	defer a.StopAudio()
	a.Layout(640, 480)
	a.SetTooltipDelayPreference(0, nil)
	a.SetTooltipFont(f.Font.Value())
	a.SetNewGameChargen(nil)
	if err := a.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenPicker {
		t.Fatalf("NEW GAME without a generator opened %v, want the map list", a.Screen())
	}
	rows := a.HeadlessRows()
	described := -1
	for i, r := range rows {
		if i < 25 && r.Description != "" && r.HasColumns() {
			described = i
			break
		}
	}
	if described < 0 {
		t.Fatal("no map among the first 25 rows carries a decoded description and size")
	}
	bounds := image.Rect(0, 0, 640, 480)
	hover := func(p image.Point) *image.RGBA {
		t.Helper()
		if err := a.HeadlessPointer("hover", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		state, pic := a.HeadlessTooltip()
		if !state.Visible || pic == nil {
			t.Fatalf("no popup at %v: %+v", p, state)
		}
		return pic
	}
	rowY := func(i int) int { return 40 + 16*i + 1 }
	p := image.Pt(9, rowY(described))
	desc := hover(p)
	wantDesc, _, _ := ui.ComposeTooltipHint(strings.Split(rows[described].Description, "#"), f.Font.Value(), p, bounds)
	if !samePicture(desc, wantDesc) {
		t.Fatal("description column hover differs from the row's own description")
	}
	writeTooltipShot(t, "maplist-description", desc)
	// Each column answers its caption over any row; the captions differ.
	var pics []*image.RGBA
	for column, slot := range map[int]int{1: 134, 2: 135, 3: 136} {
		x := 8 + []int{0, 300, 390, 420}[column] + 1
		want := strings.Split(caption(slot), "#")
		for _, i := range []int{described, 0, 3} {
			at := image.Pt(x, rowY(i))
			expected, _, _ := ui.ComposeTooltipHint(want, f.Font.Value(), at, bounds)
			pic := hover(at)
			if !samePicture(pic, expected) {
				t.Fatalf("column %d over row %d: hover differs from dialogs.txt[%d]", column, i, slot)
			}
			if i == described {
				pics = append(pics, cloneRGBA(pic))
				writeTooltipShot(t, fmt.Sprintf("maplist-column-%d", column), pic)
			}
		}
	}
	for i := range pics {
		for j := i + 1; j < len(pics); j++ {
			if samePicture(pics[i], pics[j]) {
				t.Fatal("two columns paint the same popup; the witness cannot tell them apart")
			}
		}
	}
}

func TestReleaseMapListHoverReadsTheWholeDescriptionBlock(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	src := ArchiveMaps(f.Archives.Containers)
	var base MapEntry
	var raw []byte
	for _, e := range f.Maps {
		if !e.FromArchive || e.Description == "" || !e.Choosable() {
			continue
		}
		data, err := src.Read(e.Source)
		if err != nil {
			t.Fatal(err)
		}
		base, raw = e, data
		break
	}
	if raw == nil {
		t.Fatal("no archive map carries a decoded description")
	}
	const payload, block = 40, 0x78
	lead := raw[payload+block : payload+block+bytes.IndexByte(raw[payload+block:], 0)]
	const copies = 6
	if (len(lead)+1)*copies-1 <= 64 {
		t.Fatalf("description of %d bytes repeated %d times fits the 64-byte field", len(lead), copies)
	}
	patched := append([]byte(nil), raw...)
	parts := make([][]byte, copies)
	for i := range parts {
		parts[i] = lead
	}
	text := bytes.Join(parts, []byte{0x0a})
	for i := 0; i < 512; i++ {
		patched[payload+block+i] = 0
	}
	copy(patched[payload+block:], text)
	list := BuildMapList(oneMap{"probe.alm", patched}, nil)
	if len(list) != 1 || list[0].Err != nil {
		t.Fatalf("patched map did not list: %+v", list)
	}
	f.Maps = list
	selector := 0
	if f.Font.Value() != nil {
		selector = f.Font.Value().Selector
	}
	a := f.App("map list description block")
	defer a.StopAudio()
	a.Layout(640, 480)
	a.SetTooltipDelayPreference(0, nil)
	a.SetTooltipFont(f.Font.Value())
	a.SetNewGameChargen(nil)
	if err := a.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	p := image.Pt(9, 41)
	if err := a.HeadlessPointer("hover", p.X, p.Y); err != nil {
		t.Fatal(err)
	}
	state, pic := a.HeadlessTooltip()
	if !state.Visible || pic == nil {
		t.Fatalf("no popup over the patched row: %+v", state)
	}
	one := EncodeInstallText(base.Description, selector)
	var lines []string
	for i := 0; i < copies; i++ {
		lines = append(lines, one)
	}
	bounds := image.Rect(0, 0, 640, 480)
	want, _, _ := ui.ComposeTooltipHint(lines, f.Font.Value(), p, bounds)
	if !samePicture(pic, want) {
		t.Fatal("hover over a long multi-line description differs from its lines")
	}
	writeTooltipShot(t, "maplist-description-block", pic)
	field := patched[payload+block : payload+block+64]
	if n := bytes.IndexByte(field, 0); n >= 0 {
		field = field[:n]
	}
	old, _, _ := ui.ComposeTooltipHint(strings.Split(EncodeInstallText(string(field), selector), "#"), f.Font.Value(), p, bounds)
	if samePicture(pic, old) {
		t.Fatal("the 64-byte field reading paints the same popup; the witness cannot tell them apart")
	}
}

type oneMap struct {
	name string
	data []byte
}

func (m oneMap) Names() []string { return []string{m.name} }

func (m oneMap) Read(name string) ([]byte, error) {
	if name != m.name {
		return nil, fmt.Errorf("no entry %q", name)
	}
	return append([]byte(nil), m.data...), nil
}

// The map list prints each map's header width and height minus 16 (MENU-067).
// The expectation is read from the archive's own bytes: the first two dwords of
// the map payload, which starts 40 bytes into the file.
func TestReleaseMapListSizeColumnPrintsTheHeaderDwordsMinusSixteen(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	src := ArchiveMaps(f.Archives.Containers)
	a := f.App("map list size column")
	defer a.StopAudio()
	a.Layout(640, 480)
	a.SetNewGameChargen(nil)
	if err := a.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	rows := a.HeadlessRows()
	checked := 0
	for i, e := range f.Maps {
		if !e.FromArchive || i >= len(rows) || !rows[i].HasColumns() {
			continue
		}
		raw, err := src.Read(e.Source)
		if err != nil {
			t.Fatal(err)
		}
		const payload = 40
		w := int(binary.LittleEndian.Uint32(raw[payload:]))
		h := int(binary.LittleEndian.Uint32(raw[payload+4:]))
		want := fmt.Sprintf("%dx%d", w-16, h-16)
		if got := rows[i].ColumnTexts()[0]; got != want {
			t.Errorf("%s: size column %q, want %q from header dwords %d and %d", e.Source, got, want, w, h)
		}
		if rows[i].ColumnTexts()[0] == fmt.Sprintf("%dx%d", w, h) {
			t.Errorf("%s: the size column still prints the raw header dwords", e.Source)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no archive map row carries a decoded size")
	}
}
