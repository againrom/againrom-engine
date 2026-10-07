package game

// The book's data half: installed fixed-cell identity and selected-population
// availability are resolved here. UI entries and cast orders carry real IDs.

import (
	"fmt"
	"image"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

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
	for _, id := range originalBookIDs {
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

// spellSelfKind is the cast cursor's self kind: Shield by its row, Fire
// Sacrifice by its id. Only the cursor and the click gate read it; the order
// a click makes follows PointTarget.
func spellSelfKind(rule sim.SpellRule) bool {
	return spellSelfOnly(rule) || rule.ID == fireSacrificeSpellID && !rule.TargetsUnit && rule.Area
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
		entry := ui.SpellEntry{ID: uint32(rule.ID), Name: names[rule.ID], PointTarget: !rule.TargetsUnit && (rule.Area || rule.ID == 26), SelfOnly: spellSelfKind(rule),
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

// selectedSpellbook uses AI-SPELLIDENT-286's fixed catalog when the table
// supplies all its IDs. An incomplete/custom table keeps its own row order;
// no compact position is ever translated into a spell identity. Availability
// is the union of accepted selected books (AI-SPELLPOP-287). Information comes
// from the first knower, or the primary when the cell is unavailable.
func selectedSpellbook(r sim.Rules, selected []sim.Entity, table []sim.SpellRule, names map[uint16]string, w ui.Words, icon func(uint16) *image.RGBA) ([]ui.SpellEntry, bool) {
	if len(selected) == 0 {
		return nil, false
	}
	byID := make(map[uint16]sim.SpellRule, len(table))
	for _, rule := range table {
		byID[rule.ID] = rule
	}
	ordered := make([]sim.SpellRule, 0, len(originalBookIDs))
	for _, id := range originalBookIDs {
		if rule, ok := byID[id]; ok {
			ordered = append(ordered, rule)
		}
	}
	fixed := len(ordered) == len(originalBookIDs)
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
			PointTarget: known && !rule.TargetsUnit && (rule.Area || rule.ID == 26), SelfOnly: known && spellSelfKind(rule),
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
	var selected []sim.Entity
	for _, id := range mw.view.SelectedUnits() {
		if e, held := mw.entity(sim.EntityID(id)); held {
			selected = append(selected, e)
		}
	}
	if len(selected) != 0 {
		book, fixed := selectedSpellbook(mw.world.Rules(), selected, mw.world.Spells(), mw.spellNames, mw.view.Words(), mw.spellIcon)
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
	var class int32
	var mage, materialized bool
	var weapon *data.Weapon
	if member := mw.missionPartyMember(id); member != nil {
		class = member.Class
		if member.MercenaryType != 0 {
			return class
		}
		mage, weapon, materialized = member.Mage, member.Weapon, member.WeaponMaterialized
	} else {
		if mw.mission == nil || mw.mission.state == nil {
			return fallbackClass
		}
		roster := mw.mission.state.Start.Roster
		if _, ok := roster[id]; !ok {
			return fallbackClass
		}
		class = roster[id].Class
		if roster[id].MercenaryType != 0 {
			return class
		}
		// Only a Hero placement derives its class from equipment (ANIM-106).
		if e, held := mw.entity(id); held && !data.FigureIsHero(e.TypeID) {
			if e.TypeID > 0 {
				return e.TypeID
			}
			return class
		}
		mage, weapon, materialized = roster[id].Mage, roster[id].Weapon, roster[id].WeaponMaterialized
	}
	if slots, ok := mw.world.Equipped(id); ok {
		eq := equipmentFromSlots(slots)
		if occupied, _ := eq.Occupied(1); !occupied && weapon != nil && !materialized {
			eq.SetCode(1, weapon.Code)
		}
		_, _, resolved, matched := data.HeroAppearance(mw.mission.list, eq, mage, false)
		// A name no arm matches leaves class 1 (ANIM-107); with no list loaded
		// nothing can be derived.
		if matched || len(mw.mission.list) != 0 {
			return resolved
		}
	}
	return class
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
// every display push: the spell record's own values at the actor's power
// (TEXT-096), so a row that carries damage columns states them whatever its
// effect arm (DIV-1882).
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
	var damageMin, damageMax int64
	rangeMin, rangeMax := knowers[0].c.Range, knowers[0].c.Range
	durMin, durMax := knowers[0].c.Duration, knowers[0].c.Duration
	for _, k := range knowers {
		lo, hi := spellRecordDamage(k.rule, k.c)
		damageMin += lo
		damageMax += hi
		rangeMin, rangeMax = min(rangeMin, k.c.Range), max(rangeMax, k.c.Range)
		durMin, durMax = min(durMin, k.c.Duration), max(durMax, k.c.Duration)
	}
	if damageMax != 0 {
		lines = add(lines, spellLabelDamage, spellPair(damageMin, damageMax, "%d", "-"))
	}
	if rangeMax != 0 {
		lines = add(lines, spellLabelRange, spellPair(rangeMin, rangeMax, "%d", "-"))
	}
	if durMax != 0 {
		lines = add(lines, spellLabelDuration, spellDurationPair(durMin, durMax))
	}
	// The blocks run in this order and share one destination, so the last
	// block that holds a value is the only caption shown.
	caption := ""
	for _, block := range spellCaptions {
		lo, hi, held := int64(0), int64(0), false
		for _, k := range knowers {
			v, ok := block.value(k.rule, k.c)
			if !ok {
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
		if l := label(block.label); l != "" {
			caption = l + ": " + spellPair(lo, hi, block.format, block.rangeSep)
		}
	}
	if caption != "" {
		lines = append(lines, caption)
	}
	return lines
}

// spellRecordDamage is the record's bytes +0xe and +0xf at a power: the
// minimum, and the maximum as the minimum plus the spread. Each is the low
// byte of a column times f = power/30 + 1 truncated, formed in double
// arithmetic, the spread from the maximum column's product less the minimum
// byte (TEXT-096). A column at or below zero leaves its byte 0.
func spellRecordDamage(rule sim.SpellRule, c sim.SpellCharacteristics) (minimum, maximum int64) {
	dmin, dmax := rule.DamageMin, rule.DamageMax
	product := func(column int32) int {
		if c.HasDamageFactor {
			return int(int64(column) * int64(c.DamageFactor) / 30)
		}
		return int(float64(float64(column) * (float64(c.RecordPower)/30.0 + 1.0)))
	}
	var lo, spread int
	if dmin > 0 {
		lo = int(uint8(product(dmin)))
	}
	if dmax > 0 {
		spread = int(uint8(product(dmax) - lo))
	}
	return int64(lo), int64(lo + spread)
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
			if want == rule.ID && inOriginalBook(rule.ID) {
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

// spellCaptions are the caption blocks in code order (TEXT-080, TEXT-081).
// Caption 187 and ids 17, 27 and 28 are outside the book's 24 cells.
var spellCaptions = []spellCaption{
	{label: 182, signed: true, format: "%d", rangeSep: "...", value: func(rule sim.SpellRule, c sim.SpellCharacteristics) (int64, bool) {
		l := int64(c.Power)
		switch {
		case rule.ID != 24 && rule.ID != 7 || !inOriginalBook(rule.ID):
			return 0, false
		case c.HasMagnitude:
			return int64(c.Magnitude), true
		case rule.ID == 24:
			return l/15 + 1, true
		}
		return -(l/15 + 1), true
	}},
	{label: 183, format: "+%d", rangeSep: "...", value: spellCaptionByID([]uint16{5, 16, 10, 22}, func(l int64) int64 { return min(l/2, 100) }, func(m int64) int64 { return min(m, 100) })},
	{label: 184, signed: true, format: "%d", rangeSep: "...", value: spellCaptionByID([]uint16{12}, func(l int64) int64 { return -(l/30 + 1) }, func(m int64) int64 { return -m })},
	{label: 185, format: "+%d%%", rangeSep: "...", value: spellCaptionByID([]uint16{23}, func(l int64) int64 { return 4*l/5 + 20 }, func(m int64) int64 { return m })},
	{label: 186, format: "%d", rangeSep: "-", value: func(rule sim.SpellRule, c sim.SpellCharacteristics) (int64, bool) {
		return int64(rule.RayLimit(c.Power)), rule.ID == 14 && inOriginalBook(rule.ID)
	}},
	{label: 217, format: "%d", rangeSep: "-", value: spellCaptionByID([]uint16{18}, func(l int64) int64 { return l/10 + 3 }, func(m int64) int64 { return m })},
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
}
