package game

// The book's data half: installed fixed-cell identity and selected-population
// availability are resolved here. UI entries and cast orders carry real IDs.

import (
	"fmt"
	"image"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func spellBookRows(t *mapload.Table) []uint16 {
	if t != nil && t.Edition != nil {
		return data.SpellPresentationNamed(t.Edition.Spells).BookRows
	}
	return originalBookIDs
}

// spellGame is the game whose spell rules, book form included, table t was
// read under; no table or no edition is the first game's.
func spellGame(t *mapload.Table) rules.SpellGame {
	if t != nil && t.Edition != nil {
		return data.SpellPresentationNamed(t.Edition.Spells).Game
	}
	return rules.SpellGameFirst
}

// spellNamesFrom resolves the fixed book's installed cell names by spell ID.
// A custom or incomplete spell catalog keeps the Spells collection names.
// Missing or invalid Spells data leaves book entries unnamed.
func spellNamesFrom(t *mapload.Table, bookNames [29]string) map[uint16]string {
	var c data.Collection
	if t != nil {
		c = t.Spells
	}
	spells, err := data.LoadSpells(c)
	if err != nil || len(spells) == 0 {
		return nil
	}
	fixedBookCatalog := true
	for _, id := range spellBookRows(t) {
		if int(id) > len(spells) {
			fixedBookCatalog = false
			break
		}
	}
	names := make(map[uint16]string, len(spells))
	for i, sp := range spells {
		id := uint16(i + 1)
		name := sp.Name
		if fixedBookCatalog && int(id) < len(bookNames) && bookNames[id] != "" {
			name = bookNames[id]
		}
		names[id] = name
	}
	return names
}

// spellSelfOnly reads a self spell off the installed row: its target column is
// not the unit value, its shape is not an area, and its range is zero, so the
// only place it can land is the caster. The installed table has one such row,
// Shield (identical on the English and Russian installs).
func spellSelfOnly(rule sim.SpellRule) bool {
	return !rule.TargetsUnit && !rule.Area && rule.MaxRange == 0
}

// fireSacrificeSpellID is the second self spell: its row is an area, so the
// row alone does not name it, and it keeps its cell order.
const fireSacrificeSpellID = 4

// spellArmOf is the first-game arm of the table row id, or id itself when the
// table carries no such row.
func spellArmOf(table []sim.SpellRule, id uint32) uint32 {
	for _, r := range table {
		if uint32(r.ID) == id {
			return uint32(r.ArmID())
		}
	}
	return id
}

// spellSelfKind is the cast cursor's self kind: Shield by its row, Fire
// Sacrifice by its id. Only the cursor and the click gate read it; the order
// a click makes follows PointTarget.
func spellSelfKind(rule sim.SpellRule) bool {
	return spellSelfOnly(rule) || rule.ArmID() == fireSacrificeSpellID && !rule.TargetsUnit && rule.Area
}

func spellbookOf(r sim.Rules, e sim.Entity, table []sim.SpellRule, names map[uint16]string,
	w ui.Words, icon func(uint16) *image.RGBA) []ui.SpellEntry {
	var book []ui.SpellEntry
	for _, rule := range table {
		var known bool
		rule, known = sim.BookRuleFor(e, rule)
		if !known {
			continue
		}
		// THE AUTOCAST FLAG AND THE POPUP LINES ARE THE ENTRY'S OTHER TWO FIELDS
		// SINCE 0154. Both are resolved here, where the row and the entity are
		// already in hand, so the drawing tier receives a flag and a list of
		// strings and derives neither.
		auto := e.AutoSpell != 0 && uint32(e.AutoSpell) == uint32(rule.ID)
		characteristics := sim.SpellCharacteristicsFor(r, e, rule)
		entry := ui.SpellEntry{ID: uint32(rule.ID), Name: names[rule.ID], PointTarget: !rule.TargetsUnit && (rule.Area || rule.ArmID() == 26), SelfOnly: spellSelfKind(rule),
			Autocast: auto, Info: spellInfoLines(rule, characteristics, names[rule.ID], &w)}
		// THE ICON IS OPTIONAL AND ITS ABSENCE IS ORDINARY (0141;
		// `MAGIC-ICON-024`). Four shipped spells are in no slot of the icon
		// strip and therefore have no picture at all, and an install whose
		// atlas will not read leaves every spell without one; both reach the
		// drawing tier as a nil picture, which is what it drew every spell as
		// before this story.
		//
		// The lookup is a FUNCTION rather than a map so this stays a pure
		// function of its arguments: a test drives it with nil and gets the
		// book this build composed before icons existed.
		if icon != nil {
			entry.Icon = icon(rule.ID)
		}
		book = append(book, entry)
	}
	return book
}

// selectedSpellbook uses the supplied cell order when every row exists.
// Custom tables keep their row order. Availability is the selected books'
// union; information comes from the first knower, or the primary otherwise.
func selectedSpellbook(r sim.Rules, selected []sim.Entity, table []sim.SpellRule, names map[uint16]string, w ui.Words, icon func(uint16) *image.RGBA, rows []uint16) ([]ui.SpellEntry, bool) {
	if len(selected) == 0 {
		return nil, false
	}
	byID := make(map[uint16]sim.SpellRule, len(table))
	for _, rule := range table {
		byID[rule.ID] = rule
	}
	ordered := make([]sim.SpellRule, 0, len(rows))
	for _, id := range rows {
		if rule, ok := byID[id]; ok {
			ordered = append(ordered, rule)
		}
	}
	fixed := len(rows) != 0 && len(ordered) == len(rows)
	if !fixed {
		ordered = table
	}
	book := make([]ui.SpellEntry, 0, len(ordered))
	for _, rule := range ordered {
		// The popup folds over every selected actor that carries the cell;
		// the cell's own row is the first knower's. An unavailable cell
		// states its row for the primary unit and shows no popup.
		var knowers []spellKnower
		base := rule
		for _, e := range selected {
			if resolved, ok := sim.BookRuleFor(e, base); ok {
				if len(knowers) == 0 {
					rule = resolved
				}
				knowers = append(knowers, spellKnower{resolved, sim.SpellCharacteristicsFor(r, e, resolved)})
			}
		}
		known := len(knowers) > 0
		if !known {
			knowers = []spellKnower{{rule, sim.SpellCharacteristicsFor(r, selected[0], rule)}}
		}
		entry := ui.SpellEntry{ID: uint32(rule.ID), Name: names[rule.ID], Unavailable: !known,
			PointTarget: known && !rule.TargetsUnit && (rule.Area || rule.ArmID() == 26), SelfOnly: known && spellSelfKind(rule),
			Info: spellPopupLines(knowers, names[rule.ID], &w)}
		// Autocast remains the existing primary-unit gesture, not a new
		// group policy inferred from the book-command population claim.
		entry.Autocast = known && selected[0].AutoSpell != 0 && uint32(selected[0].AutoSpell) == entry.ID
		if icon != nil {
			entry.Icon = icon(rule.ID)
		}
		book = append(book, entry)
	}
	return book, fixed
}

// Refresh on ordinary frames too: a paused selection changes without a sim
// tick. The accepted UI population is this build's snapshot-presence boundary;
// original object+7c lifetime and upstream mixed selections remain Unknown.
func (mw *mapWorld) pushSpellbook() {
	if mw == nil || mw.view == nil || mw.world == nil {
		return
	}
	// The book of a unit the player does not own shows the no-hero state;
	// the revealed map shows it (DIV-4313).
	revealed := mw.view.FogRevealed()
	var selected []sim.Entity
	for _, id := range mw.view.SelectedUnits() {
		if e, held := mw.entity(sim.EntityID(id)); held && mw.playerMayManage(e.ID, revealed) {
			e.AutoSpell = uint16(mw.autocastShown(e.ID, e.AutoSpell))
			selected = append(selected, e)
		}
	}
	if len(selected) != 0 {
		book, fixed := selectedSpellbook(mw.world.Rules(), selected, mw.world.Spells(), mw.spellNames, mw.view.Words(), mw.spellIcon, mw.spells().BookRows)
		if fixed {
			mw.view.SetSpellbookCatalog(uint32(selected[0].ID), book)
		} else {
			mw.view.SetSpellbook(uint32(selected[0].ID), book)
		}
		return
	}
	mw.view.ClearSpellbook()
}

// spellClientClass reads the drawable's class, not the server persistence-band
// TypeID (HERO-APPEAR-041/042/044). A hero recomputes it from live equipment; a
// map-authored human keeps its Humans-row class. Unresolved records keep their
// explicit class, and hired actors their fixed class.
func (mw *mapWorld) spellClientClass(id sim.EntityID, fallbackClass int32) int32 {
	if e, ok := mw.entity(id); ok && e.ClientClassFixed {
		return e.Class
	}
	var hero mapload.PartyMember
	if member := mw.missionPartyMember(id); member != nil {
		hero = *member
		if member.MercenaryType != 0 {
			return hero.Class
		}
	} else {
		if mw.mission == nil || mw.mission.state == nil {
			return fallbackClass
		}
		roster := mw.mission.state.Start.Roster
		if _, ok := roster[id]; !ok {
			return fallbackClass
		}
		hero = roster[id]
		if hero.MercenaryType != 0 {
			return hero.Class
		}
		// Only a Hero placement derives its class from equipment (ANIM-106).
		if e, held := mw.entity(id); held && !data.FigureIsHero(e.TypeID) {
			if e.TypeID > 0 {
				return e.TypeID
			}
			return hero.Class
		}
	}
	look := heroAppearance(mw.world, id, hero, mw.mission.list)
	// A name no arm matches leaves class 1 (ANIM-107); with no list loaded
	// nothing can be derived.
	if look.Held && (look.Matched || mw.mission.list.Len() != 0) {
		return look.Class
	}
	return hero.Class
}

// spellSchool answers the elemental school of the spell whose id is id, 0
// for an id naming no row of this world's own table and for id 0 itself. It
// is the one place a spell effect mark's own id becomes something pkg/ui can
// draw with: that package may not name a spell row, so the school crosses
// the seam already resolved.
func spellSchool(id uint16, table []sim.SpellRule) int {
	if id == 0 {
		return 0
	}
	for _, rule := range table {
		if rule.ID == id {
			return int(rule.School)
		}
	}
	return 0
}

// The spellbook popup's labels (TEXT-080): main.txt indices for the mana,
// damage, range and duration lines. The caption indices the fold may add are
// in spellCaptions.
const (
	spellLabelManaCost = 117
	spellLabelDamage   = 118
	spellLabelRange    = 123
	spellLabelDuration = 124
)

// spellKnower is one selected actor that knows the spell.
type spellKnower struct {
	rule sim.SpellRule
	c    sim.SpellCharacteristics
}

// spellInfoLines is the popup of a spell known by one actor.
func spellInfoLines(rule sim.SpellRule, c sim.SpellCharacteristics, name string, w *ui.Words) []string {
	return spellPopupLines([]spellKnower{{rule, c}}, name, w)
}

// spellPopupLines composes the book popup (TEXT-080, TEXT-081): name and mana
// line, damage, range and duration lines, then at most one caption, folded over
// every selected knower: damage summed, range, duration and caption minima
// and maxima, mana the last knower's. A zero maximum omits a line and an equal
// pair shows one number. Numbers come from SpellCharacteristicsFor, rebuilt on
// every display push at the power the actor's cast takes: the damage pair the
// cast rolls in, and the record's range and duration (TEXT-096), so a row that
// carries damage columns states them whatever its effect arm (DIV-1882).
func spellPopupLines(knowers []spellKnower, name string, w *ui.Words) []string {
	if name == "" {
		name = "spell"
	}
	label := func(i int) string {
		if i >= 0 && i < len(w.Hover) {
			return w.Hover[i]
		}
		return ""
	}
	add := func(lines []string, i int, value string) []string {
		if l := label(i); l != "" {
			lines = append(lines, l+": "+value)
		}
		return lines
	}
	lines := []string{name}
	if len(knowers) == 0 {
		return lines
	}
	lines = add(lines, spellLabelManaCost, fmt.Sprintf("%d", knowers[len(knowers)-1].c.ManaCost))
	return spellDetailsLines(knowers, lines, w, spellbookDetails)
}

type spellDetailContext uint8

// A Scroll's and a Weapon's description share the item formatter's tag-50 arm
// (TEXT-129): its range line reads the filled Spell for a Scroll and the
// Weapon's own range for any other class, with no zero test (ITEM-211). The
// weaponDetails caller puts that range in the knower's Range.
const (
	spellbookDetails spellDetailContext = iota
	scrollDetails
	weaponDetails
)

func spellDetailsLines(knowers []spellKnower, lines []string, w *ui.Words, context spellDetailContext) []string {
	add := func(lines []string, i int, value string) []string {
		if i >= 0 && i < len(w.Hover) && w.Hover[i] != "" {
			lines = append(lines, w.Hover[i]+": "+value)
		}
		return lines
	}
	var damageMin, damageMax int64
	rangeMin, rangeMax := knowers[0].c.Range, knowers[0].c.Range
	durMin, durMax := knowers[0].c.Duration, knowers[0].c.Duration
	for _, k := range knowers {
		damageMin += k.c.DamageMin
		damageMax += k.c.DamageMax
		rangeMin, rangeMax = min(rangeMin, k.c.Range), max(rangeMax, k.c.Range)
		durMin, durMax = min(durMin, k.c.Duration), max(durMax, k.c.Duration)
	}
	item := context != spellbookDetails
	showDamage := damageMax != 0
	if item {
		showDamage = damageMax != damageMin
	}
	if showDamage {
		value := spellPair(damageMin, damageMax, "%d", "-")
		if item {
			value = fmt.Sprintf("%d-%d", damageMin, damageMax)
		}
		lines = add(lines, spellLabelDamage, value)
	}
	if rangeMax != 0 || context == weaponDetails {
		lines = add(lines, spellLabelRange, spellPair(rangeMin, rangeMax, "%d", "-"))
	}
	if durMax != 0 {
		lines = add(lines, spellLabelDuration, spellDurationPair(durMin, durMax))
	}
	// The blocks run in this order and share one destination, so the last
	// block that holds a value is the only caption shown.
	caption := ""
	order := []int{182, 183, 184, 185, 187, 186, 217}
	if item {
		order = []int{182, 185, 187, 183, 186, 184}
	}
	for _, label := range order {
		var block spellCaption
		for _, candidate := range spellCaptions {
			if candidate.label == label {
				block = candidate
				break
			}
		}
		lo, hi, held := int64(0), int64(0), false
		for _, k := range knowers {
			if context == spellbookDetails && !inOriginalBook(k.rule.ID) {
				continue
			}
			v, ok := block.value(k.rule, k.c)
			if !ok || item && v == 0 {
				continue
			}
			if !held {
				lo, hi, held = v, v, true
			}
			lo, hi = min(lo, v), max(hi, v)
		}
		if !held || (!block.signed && hi == 0) {
			continue
		}
		if l := w.Hover[block.label]; l != "" {
			caption = l + ": " + spellPair(lo, hi, block.format, block.rangeSep)
		}
	}
	if caption != "" {
		lines = append(lines, caption)
	}
	return lines
}

// spellPair formats an equal pair as one number and a differing pair as
// "lo<sep>hi", the percent sign closing the pair.
func spellPair(lo, hi int64, format, sep string) string {
	if lo == hi {
		return fmt.Sprintf(format, lo)
	}
	first := strings.TrimSuffix(format, "%%")
	return fmt.Sprintf(first, lo) + sep + fmt.Sprintf(format, hi)
}

func spellDurationPair(lo, hi uint16) string {
	one := func(v uint16) string { return fmt.Sprintf("%5.1f", float64(v)*0.0625) }
	if lo == hi {
		return one(lo)
	}
	return one(lo) + "-" + one(hi)
}

type spellCaption struct {
	label    int
	signed   bool // runs on any held value; the rest need a nonzero maximum
	format   string
	rangeSep string
	value    func(rule sim.SpellRule, c sim.SpellCharacteristics) (int64, bool)
}

func spellCaptionByID(ids []uint16, f, fromTable func(l int64) int64) func(sim.SpellRule, sim.SpellCharacteristics) (int64, bool) {
	return func(rule sim.SpellRule, c sim.SpellCharacteristics) (int64, bool) {
		for _, want := range ids {
			if want == rule.ID {
				if c.HasMagnitude {
					return fromTable(int64(c.Magnitude)), true
				}
				return f(int64(c.Power)), true
			}
		}
		return 0, false
	}
}

func inOriginalBook(id uint16) bool {
	for _, b := range originalBookIDs {
		if b == id {
			return true
		}
	}
	return false
}

// spellCaptions share the book and item formulas (TEXT-081, TEXT-133).
var spellCaptions = []spellCaption{
	{label: 182, signed: true, format: "%d", rangeSep: "...", value: func(rule sim.SpellRule, c sim.SpellCharacteristics) (int64, bool) {
		l := int64(c.Power)
		switch {
		case rule.ID != 24 && rule.ID != 7 && rule.ID != 28:
			return 0, false
		case c.HasMagnitude:
			return int64(c.Magnitude), true
		case rule.ID == 24:
			return l/15 + 1, true
		}
		return -(l/15 + 1), true
	}},
	{label: 183, format: "+%d", rangeSep: "...", value: spellCaptionByID([]uint16{5, 16, 10, 22}, func(l int64) int64 { return min(l/2, 100) }, func(m int64) int64 { return min(m, 100) })},
	{label: 184, signed: true, format: "%d", rangeSep: "...", value: spellCaptionByID([]uint16{12, 17}, func(l int64) int64 { return -(l/30 + 1) }, func(m int64) int64 { return -m })},
	{label: 185, format: "+%d%%", rangeSep: "...", value: spellCaptionByID([]uint16{23}, func(l int64) int64 { return 4*l/5 + 20 }, func(m int64) int64 { return m })},
	{label: 187, format: "+%d%%", rangeSep: "...", value: spellCaptionByID([]uint16{27}, func(l int64) int64 { return 4*l/5 + 20 }, func(m int64) int64 { return m })},
	{label: 186, format: "%d", rangeSep: "-", value: func(rule sim.SpellRule, c sim.SpellCharacteristics) (int64, bool) {
		return int64(rule.RayLimit(c.Power)), rule.ID == 14
	}},
	{label: 217, format: "%d", rangeSep: "-", value: spellCaptionByID([]uint16{18}, func(l int64) int64 { return l/10 + 3 }, func(m int64) int64 { return m })},
}

// autoCastSetting is the Player's setting with any queued change applied.
func (mw *mapWorld) autoCastSetting(player uint32) sim.AutoCastSetting {
	s := mw.world.AutoCast(player)
	for _, c := range mw.pending {
		if c.Kind == sim.KindPlayerParameter && c.Player == player && sim.PlayerParameter(c.X) == sim.PlayerParameterAutoCast {
			s = sim.AutoCastSetting(c.Y)
		}
	}
	return s
}

// setAutoCastLevel queues the human player's setting at level, keeping the
// target bits; a zero level is Off.
func (mw *mapWorld) setAutoCastLevel(level sim.AutoCastSetting) {
	next := mw.autoCastSetting(sim.SelfSlot).WithLevel(level)
	mw.pending = append(mw.pending, sim.SetPlayerParameter(sim.SelfSlot, sim.PlayerParameterAutoCast, int32(next)))
}

// setAutocast is the ui.MapAutocast the loader hands the front-end: it turns
// the toggle into the simulation's own command and appends it to the SAME
// QUEUE the orders use, and it does NOTHING ELSE.
//
// IT LOOKS NOTHING UP AND STEPS NO WORLD, castAt's own two reasons (world.go):
// whether the named entity is still held, and whether the id names a row this
// world's table loaded, are answered inside the command phase the next advance
// runs — a command naming an absent entity is already a no-op there.
func (mw *mapWorld) setAutocast(entity, spell uint32) {
	id := sim.EntityID(entity)
	cmd := sim.Autocast(id, sim.SpellID(spell))
	// IT DOES NOT MARK THE UNIT COMMANDED, unlike every order beside it.
	// mw.commanded is "the player has taken this unit over", and commands()
	// drops the scripted command track for every entity in it — for good,
	// since nothing ever removes one. Arming an autocast is a SETTING and not
	// an order: a unit whose spell the player put on repeat must keep answering
	// the mission script exactly as it did, and marking it here detached it
	// silently and permanently the first time the key was pressed.
	mw.pending = append(mw.pending, cmd)
	// The book reads the queue through autocastShown; the viewer's entity list
	// is refreshed only by a tick, so Ctrl+A's reading is told here.
	if e, ok := mw.entity(id); ok && mw.view != nil {
		mw.view.SetEntityAutocast(entity, mw.autocastShown(id, e.AutoSpell))
	}
}

// autocastShown is current with every queued autocast command for the entity
// applied in order: what the next tick makes of it. A paused game queues
// without stepping, and the book and the viewer show this value so a change
// is visible at once.
func (mw *mapWorld) autocastShown(id sim.EntityID, current uint16) uint32 {
	spell := uint32(current)
	for _, c := range mw.pending {
		if c.Kind == sim.KindAutocast && c.Entity == id {
			if c.X <= 0 || c.X > 0xffff {
				spell = 0
			} else {
				spell = uint32(c.X)
			}
		}
	}
	return spell
}
