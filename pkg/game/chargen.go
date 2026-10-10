package game

// ChargenSetup and ChargenParty are the two expressions 0119-chargen adds to
// this package, and nothing else lives in this file: the ONE place a
// generated character's screen is built out of the definition table and
// pkg/data's decoded arithmetic, and the ONE place a confirmed result is
// turned into a party. Nothing here composes either by hand a second time.

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/textinput"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	chargenChoiceSex = iota
	chargenChoiceClass
	chargenChoiceSkill
	chargenChoiceCount
)

const (
	chargenStatBody = iota
	chargenStatReaction
	chargenStatMind
	chargenStatSpirit
	chargenStatCount
)

// chargenTitle and chargenConfirm are the only two strings this file authors
// outright rather than reads out of pkg/data: generic screen chrome, on
// pkg/ui's own chargenHelp precedent (its doc: "generic UI text, not a game
// label"), never a statistic name, a skill name or an option label.
const (
	chargenTitle   = "Character Generation"
	chargenConfirm = "Confirm begins the mission."
)

// ChargenSetup builds the generation screen's setup off pkg/data's decoded
// arithmetic and this front end's own Humans collection's skill names.
// NOTHING HERE RE-PARSES THE DEFINITION TABLE: f.Humans is taken off the
// same walk StartWeapon and Bodies already ride on (FrontEnd's own doc), and
// nothing here invents a bound, a cost or a label the wiring tier alone
// should carry.
//
// THE FOUR STATISTIC ROWS ARE Body, Reaction, Mind, Spirit, IN THAT ORDER —
// the storage order character generation itself reads and writes (Terms:
// "The spread"), decoded and not a taste, and the order chargenStatBody
// through chargenStatSpirit names above.
func (f *FrontEnd) ChargenSetup() ui.ChargenSetup {
	l := f.generator()
	st := &l.Detail.Stats
	cost := make([]int, st.Ceiling+1)
	for v := range cost {
		cost[v] = generatorCost(st.Cost, v)
	}

	stat := func(name string) ui.ChargenStat {
		return ui.ChargenStat{Name: name, Floor: st.Floor, Ceiling: st.Ceiling, Start: st.Start}
	}

	choices := make([]ui.ChargenChoice, chargenChoiceCount)
	choices[chargenChoiceSex] = ui.ChargenChoice{Name: "Sex", Options: []string{"Male", "Female"}, Parent: -1}
	choices[chargenChoiceClass] = ui.ChargenChoice{Name: "Class", Options: []string{"Fighter", "Mage"}, Parent: -1}
	choices[chargenChoiceSkill] = ui.ChargenChoice{
		Name:       "Skill",
		OptionsFor: [][]string{data.SkillNames(false), data.SkillNames(true)},
		Parent:     chargenChoiceClass,
		// THE ROW'S OWN OPENING INDEX: the description's default skill, else
		// the position -skill's PartySkillSlot holds in SkillNames' own order
		// (SkillBlade is position 0). The screen seeds the row rather than
		// becoming a second writer of that field, and ui.NewChargen clamps it
		// as a dependent row's held index is clamped.
		Start: int(PartySkillSlot() - data.SkillBlade),
	}
	if d := l.Detail.DefaultSkill; d != nil {
		choices[chargenChoiceSkill].Start = *d
	}

	stats := make([]ui.ChargenStat, chargenStatCount)
	stats[chargenStatBody] = stat("Body")
	stats[chargenStatReaction] = stat("Reaction")
	stats[chargenStatMind] = stat("Mind")
	stats[chargenStatSpirit] = stat("Spirit")

	// Every generator tip text is read once here; each page's enter tests
	// TipsMode itself (TOWN-518, TOWN-522).
	var src entrySource
	if f.Archives != nil {
		src = f.Archives.Containers
	}
	tip := func(t ui.GeneratorTipText) string {
		text, _ := ReadShopTip(src, t.File)
		if t.Section == "" || text == "" {
			return text
		}
		return secondGameTextSection(secondGameMissionBytes([]byte(text), LanguageSelector(src)), t.Section)
	}
	var tipSelect [3]string
	for i, t := range l.Tips.Select {
		if i < len(tipSelect) {
			tipSelect[i] = tip(t)
		}
	}
	reset := l.Detail.ResetFor(f.Base().Profile.Language)

	setup := ui.ChargenSetup{
		Title:   chargenTitle,
		Name:    fallbackHeroName, // DIV-1508
		Choices: choices,
		Stats:   stats,
		Cost:    cost,
		Budget:  st.Budget,
		Confirm: chargenConfirm,
		Derive:  f.ChargenDerived,

		// The tip panel (1018 spec behaviours 1, 2, 3, 4): text resolved
		// above, art on shopArt's own cached-once precedent, and the toggle
		// read from and written through the same store every room's own tip
		// reads (tipstore.go).
		TipSelect:     tipSelect,
		TipText:       tip(l.Tips.Fighter),
		TipTextMage:   tip(l.Tips.Mage),
		TipTextDetail: tip(l.Tips.After),
		TipClose:      generatorWord(l, LanguageSelector(src), f.Words.TipClose),
		TipToggle:     generatorWord(l, LanguageSelector(src), f.Words.TipShowNext),
		TipArt:        f.tipArt(),
		TipsOn:        !f.TipsOff(),
		SetTipsOn:     func(on bool) { f.SetTipsOff(!on) },

		ResetStart: reset.Values == "start",
		ResetSkill: reset.Skill == "default",
	}
	if s := l.PreCreate.Sparkle; s != nil && len(s.Within) > 0 {
		setup.Draws = f.randomService().Stream(random.Generator)
	}
	if a := f.ChargenAssets; a != nil {
		// The field opens from the no-name seed, which the page's enter turns
		// into the enter line's installed name (TEXT-073).
		name := l.PreCreate.Name
		setup.Name = name.Unnamed
		setup.PreCreate = &ui.ChargenPreCreate{Prompt: a.Prompt, Back: a.Back, Art: a.Presentation,
			HeroNames: a.HeroNames, Unnamed: name.Unnamed, EnterName: a.EnterName, LastPress: name.LastPress}
		setup.EncodeName = func(r rune) (byte, bool) { return textinput.EncodeRune(r, a.Selector) }
		setup.Detailed = &ui.ChargenDetailed{Back: a.Back, Reset: a.Reset, Play: a.Play,
			EmptyName: a.EmptyName, ReservedName: a.ReservedName, SkillHover: a.SkillHover}
		setup.Preview = f.ChargenPreview
	}
	f.campaign().generatorSetup(f, &setup)
	return setup
}

// generator is the profile's generator description; a front end whose
// profile names none reads the first game's.
func (f *FrontEnd) generator() *ui.GeneratorDescription {
	if l := GeneratorDescription(f.Base().Profile); l != nil {
		return l
	}
	return heroNameGenerator()
}

// generatorCost is the cumulative cost of a statistic standing at v:
// trunc(factor * base^(v-1) + round).
func generatorCost(c ui.GeneratorCost, v int) int {
	return int(math.Trunc(c.Factor*math.Pow(c.Base, float64(v-1)) + c.Round))
}

// firstGeneratorPresets are the four pictures' statistic presets: the
// definition table's base row for each sex and class.
func firstGeneratorPresets(f *FrontEnd, setup *ui.ChargenSetup) {
	if f.Table == nil || f.Table.Humans == nil {
		return
	}
	for i := range setup.Presets {
		d, _, ok := data.ChargenBase(f.Table.Humans, i%2 == 1, i >= 2)
		spread := data.Spread{Body: d.Body, Reaction: d.Reaction, Mind: d.Mind, Spirit: d.Spirit}
		if ok && spread.Legal() {
			setup.Presets[i] = []int{int(d.Body), int(d.Reaction), int(d.Mind), int(d.Spirit)}
		}
	}
}

// chargenTipPath is the tip panel's own class-conditional address (1018 spec
// behaviours 1, 2; TOWN-187, "the class-conditional, not gender-conditional,
// first popup"): 0 (Fighter, the class row's own un-cycled opening index) to
// ChargenFighterTipPath, any other value to ChargenMageTipPath. Pulled out of
// ChargenSetup as its own function because the class row's Start is 0 in
// every setup this tree currently builds (no caller seeds it, unlike the
// skill row's own -skill precedent), so the mage branch has no path to it
// through ChargenSetup alone; this function is what lets a test exercise
// both branches directly.
func chargenTipPath(classStart int) string {
	if classStart != 0 {
		return ChargenMageTipPath
	}
	return ChargenFighterTipPath
}

// chargenDerivedNames are the labels the consequence block shows, in the
// order ChargenDerived returns them. They are DISPLAY LABELS THIS FILE
// AUTHORS, on chargenTitle's own terms — the original's own sheet is not
// decoded as text, and none of these is a statistic name read out of the
// definition table. Two label sets on the block are NOT here and are not this
// file's to author: the five skill names are data.SkillNames' own strings,
// resolved per class at the call, and the five element names on the magic
// resistance row are data.ProtectionNames' — decoded column titles, cited
// there.
//
// WHICH VALUES, AND WHY NOT THE REST. The block answers "what did that point
// buy": the two pools the player watches while he spends (health and mana),
// the two numbers a fight is decided by (to-hit and defence), what a blow
// does, how fast he walks, how far he sees, what magic he shrugs off, and how
// far along he is in each of the five schools his class has. Still left out:
// the absorption, the per-slot experience and the two attack timings.
//
// SIGHT, THE MAGIC RESISTANCES AND THE SKILL LEVELS WERE LEFT OUT UNTIL 0140,
// and the owner asked for all three. The note that excluded them gave two
// reasons and BOTH were wrong.
//
// "Six lines is what fits under the footer" was never measured. It is
// measurable — pkg/ui's own drawChargen puts block line k at
// chargenTop + (rows + chargenDerivedGap + k) * chargenLine, which for this
// screen's seven rows is y = 184 + 16k, and a line occupies [y, y+16) on the
// picker's own stated convention (picker.go: 25 rows "occupy y in [40, 440)").
// Clear of the message line at 452 that is k <= 15, so the block holds SIXTEEN
// lines. Thirteen values and a title is fourteen. pkg/ui measures it now
// (chargenDerivedFit) rather than stating it.
//
// "Constant across every spread the screen can reach" was wrong about the five
// this story adds. A hero's elemental protections are Spirit/2, clamped
// (HERO-RESIST-012) — and Spirit is one of the four statistics the player
// spends his points on. They move with every spread the screen can reach, which
// is exactly what makes them worth a line: a point into Spirit visibly buys
// magic resistance.
//
// THE MAGIC ROW IS d.Protection AND NOT d.Resistance, and the two are named the
// opposite way round from the way they read. UNIT-COMBAT-015's slot → title
// legend binds columns 19…23 `prot Fire..Astral` to Protection — the
// elemental, magic five — and columns 24…28 `res.Blade..res.Shooting` to
// Resistance, the weapon damage-kind five. Only the first is shown. The second
// is NEVER re-derived for a human (HERO-RESIST-012) and no starting loadout
// adds to it, so on a generated character it is five zeros: a row that could
// never move, under a label a reader would take for the one he was looking for.
const (
	chargenDerivedHealth   = "Health"
	chargenDerivedMana     = "Mana"
	chargenDerivedToHit    = "To hit"
	chargenDerivedDefence  = "Defence"
	chargenDerivedDamage   = "Damage"
	chargenDerivedSpeed    = "Speed"
	chargenDerivedSight    = "Sight"
	chargenDerivedMagicRes = "Magic resist"
)

func (f *FrontEnd) ChargenDerived(res ui.ChargenResult) []ui.ChargenDerived {
	party := f.ChargenParty(res)
	if len(party) == 0 {
		return nil
	}
	d, health, mana := mapload.PartySpawnWithTable(party[0], f.Table)

	line := func(name string, v int32) ui.ChargenDerived {
		return ui.ChargenDerived{Name: name, Value: strconv.FormatInt(int64(v), 10)}
	}
	out := []ui.ChargenDerived{
		line(chargenDerivedHealth, health),
		line(chargenDerivedMana, mana),
		line(chargenDerivedToHit, d.Combat.ToHit),
		line(chargenDerivedDefence, d.Combat.Defence),
	}
	spellID, _ := mapload.SpellIDByToken(f.Table, d.Combat.SpellName)
	spellBase, spellSpread, hasWeaponSpell := sim.WeaponSpellDamageFor(mapload.TableRules(f.Table), sim.Entity{
		MaxMana:          mana,
		WeaponSpell:      spellID,
		WeaponSpellLevel: d.Combat.SpellPower,
	}, mapload.SpellRules(f.Table))
	if hasWeaponSpell {
		out = append(out, ui.ChargenDerived{Name: chargenDerivedDamage,
			Value: fmt.Sprintf("%d-%d", spellBase, spellBase+spellSpread)})
	} else {
		out = append(out, ui.ChargenDerived{Name: chargenDerivedDamage, Value: fmt.Sprintf("%d-%d",
			d.Combat.DamageBase, d.Combat.DamageBase+d.Combat.DamageSpread)})
	}
	out = append(out,
		line(chargenDerivedSpeed, d.Speed),
		line(chargenDerivedSight, d.Sight),
		ui.ChargenDerived{Name: chargenDerivedMagicRes, Value: namedRow(data.ProtectionNames(), d.Protection[:])},
	)

	// THE CLASS THE PLAYER PICKED NAMES THE FIVE SCHOOLS, read through the same
	// chargenChoiceIndex ChargenParty itself reads it through, so the labels
	// here and the character built from the same result cannot disagree about
	// which class was chosen. names[i] is the slot data.SkillBlade + i:
	// SkillNames' own order is slot order, starting at the first slot either
	// character sheet shows (slot 0 is shown by neither, so it is not listed
	// and not printed).
	names := data.SkillNames(chargenChoiceIndex(res, chargenChoiceClass) != 0)
	for i, name := range names {
		out = append(out, line(name, d.Skill[data.SkillBlade+int32(i)]))
	}
	return out
}

// ChargenPreview projects the exact party member Play would create without
// creating a world or invoking a mission opener.
func (f *FrontEnd) ChargenPreview(res ui.ChargenResult) ui.ChargenPreview {
	party := f.ChargenParty(res)
	if len(party) == 0 {
		return ui.ChargenPreview{}
	}
	member := party[0]
	preview := ui.ChargenPreview{Subject: partyPanelSubject(member, f.Table, f.Words)}
	// A detailed generator preview is deliberately not on a map yet. The
	// production panel owns the corresponding CELL -, - rendering; prediction
	// of the mission's spawn cell would be a second placement path.
	preview.Subject.Unplaced = true
	if f.Archives == nil {
		return preview
	}
	preview.Doll, _ = composeUnitFigure(f.Archives.Containers, equipmentFromSlots(member.Worn), memberFigureID(member))
	return preview
}

// namedRow prints a fixed array of derived values as ONE row, each value
// carrying its own column's name, in the array's own order. It is the shape a
// five-wide value takes on a screen with one line to give it.
//
// IT PAIRS BY INDEX AND SAYS SO. names[i] must be the title of the column
// values[i] was filled from — which is exactly what data.ProtectionNames
// promises for d.Protection and what no reordering of either may break. A
// shorter name list prints the numbers it has no name for as bare values rather
// than dropping them or panicking: the number is the fact, the name is the
// convenience, and losing a value would be the worse failure.
//
// It is a function rather than a format literal so that the separator, the
// pairing and the order have one home, and so the row cannot silently lose an
// element if either side ever changes width.
func namedRow(names []string, values []int32) string {
	parts := make([]string, len(values))
	for i, n := range values {
		v := strconv.FormatInt(int64(n), 10)
		if i < len(names) && names[i] != "" {
			v = names[i] + " " + v
		}
		parts[i] = v
	}
	return strings.Join(parts, " / ")
}

// chargenChoiceIndex and chargenStatValue are the ONE place ChargenParty
// defends against a malformed ui.ChargenResult — fewer choices or fewer
// stats than the ChargenSetup it was confirmed against declared, which is
// the one shape of "malformed" a slice read can actually panic on. Both
// answer the reading ChargenSetup's own start already stands for: index 0
// (a choice row's first option, NewChargen's own opening state in pkg/ui)
// and data.ChargenStat (a statistic row's own Start), so a short result is
// read exactly as if its missing rows had never been touched rather than as
// an error.
//
// NEITHER FUNCTION DEFENDS AGAINST A VALUE THAT IS MERELY OUT OF AN
// OPTION'S RANGE — a skill choice of 9, a stat of -400 — because that is
// already total everywhere it is read: data.NewHero trains nothing for a
// slot outside 1..5 and StartingWeaponName names no weapon for one either,
// a sex or class choice is read only as "zero or not", and a Spread's four
// ints are legal Go values whatever they hold — GENERATION LEGALITY is
// Spread.Legal's own question, asked by the screen model before it will
// even hand back a Result, never enforced a second time by a constructor
// here. Guarding a range nothing downstream can panic on would be a second
// rule for a case the two fallbacks below already make unreachable.
func chargenChoiceIndex(res ui.ChargenResult, row int) int {
	if row < 0 || row >= len(res.Choices) {
		return 0
	}
	return res.Choices[row]
}

func chargenStatValue(res ui.ChargenResult, row int) int32 {
	if row < 0 || row >= len(res.Stats) {
		return data.ChargenStat
	}
	return int32(res.Stats[row])
}

// tableWeapons is f.Table's three item collections, read defensively: the
// zero triple for a front end whose Table is nil. data.ResolveWeapon is
// already total over a nil ScaleTable or a nil Collection on its own
// account (weapon.go's own findByName/takePrefix nil rule) — this guard
// exists only because a nil *mapload.Table cannot be FIELD-ACCESSED at all,
// which is a narrower problem than the one those two types already solve,
// and one only a hand-assembled front end in a test can reach: NewFrontEnd
// either fills Table or fails outright.
func (f *FrontEnd) tableWeapons() (shapes, materials data.ScaleTable, weapons data.Collection) {
	if f.Table == nil {
		return nil, nil, nil
	}
	return f.Table.Shapes, f.Table.Materials, f.Table.Weapons
}

// ChargenParty turns a confirmed generation result into a one-member party,
// through the SAME assembleParty (hero.go) MissionParty calls — there is
// one construction of a party in this package and this is not a second one.
//
// THE BASE ROW IS SEARCHED BY THE CHOSEN PAIR, through data.ChargenBase over
// f.Humans — the same collection MissionParty now searches too, for its
// own default axes, off the same walk of the same file. Its profile, its
// face and its own ten equipment cell strings are chargenProfile's own
// answer (hero.go), read off f.Humans at the search's own index (0134 D-13):
// the zero profile, face 1 and no cells at all when nothing resolved, so a
// Humans collection this tree cannot read still yields a playable, if less
// accurate and less dressed, character rather than a refusal (R-3).
//
// THE WEAPON IS RESOLVED FRESH, for the CHOSEN CLASS AND SKILL SLOT, off
// f.Table's own three item collections rather than off f.StartWeapon —
// which is always the front end's OWN authored fighter slot's weapon and
// cannot answer for a mage or for a slot the player trained instead.
// resolveWeaponForSlot (hero.go) is the exact resolution f.StartWeapon
// itself came from at construction, so a generated character and this
// build's own authored one resolve their starting weapons through one
// routine and never two that could disagree — class leads over the trained
// slot on class's own arm, exactly as StartingWeaponName states.
func (f *FrontEnd) ChargenParty(res ui.ChargenResult) []mapload.PartyMember {
	return f.campaign().generatorParty(f, res)
}

// NewGameBegin is what an accepted new-game generator result does: the first
// game opens mission n with the generated party; the second game commits its
// new campaign and opens no map, so the generator shows the first town.
func (f *FrontEnd) NewGameBegin(n int) func(ui.ChargenResult) (ui.MapOpener, error) {
	return f.campaign().generatorBegin(f, n)
}

// firstGeneratorParty is the first game's party for a result.
func firstGeneratorParty(f *FrontEnd, res ui.ChargenResult) []mapload.PartyMember {
	female := chargenChoiceIndex(res, chargenChoiceSex) != 0
	class := chargenChoiceIndex(res, chargenChoiceClass) != 0
	slot := data.SkillBlade + int32(chargenChoiceIndex(res, chargenChoiceSkill))

	base, i, ok := data.ChargenBase(f.Humans, class, female)
	profile, face, book, cells := chargenProfile(base, ok, f.Humans, i, class)

	spread := data.Spread{
		Body:     chargenStatValue(res, chargenStatBody),
		Reaction: chargenStatValue(res, chargenStatReaction),
		Mind:     chargenStatValue(res, chargenStatMind),
		Spirit:   chargenStatValue(res, chargenStatSpirit),
	}

	shapes, materials, weapons := f.tableWeapons()
	w, _ := resolveWeaponForSlot(class, shapes, materials, weapons, slot)

	return assembleParty(partyInputs{
		Name:      res.Name,
		Spread:    spread,
		Slot:      slot,
		Mage:      class,
		Profile:   profile,
		Dir:       data.FigureDirFor(class, female),
		Face:      face,
		Book:      book,
		Weapon:    w,
		Cells:     cells,
		List:      f.Bodies,
		Table:     f.Table,
		Documents: true,
	})
}
