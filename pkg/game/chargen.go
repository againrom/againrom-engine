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

	"againrom/pkg/base"
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

	// Every generator tip row's text is read once here; each page's enter
	// tests TipsMode itself (TOWN-518, TOWN-522).
	var src entrySource
	if f.Archives != nil {
		src = f.Archives.Containers
	}
	table := TipTable(f.Base().Profile.Edition())
	tip := func(event, class string) ui.ChargenTip {
		row := table.Row(event, "", class)
		if row == nil {
			return ui.ChargenTip{}
		}
		return ui.NewChargenTip(table, row, readTipText(src, row.Text, f.textCode()))
	}
	tips := ui.ChargenTips{Fighter: tip("detail-enter", "fighter"), Mage: tip("detail-enter", "mage"), After: tip("first-skill-click", "")}
	for _, event := range ui.PreCreateTipEvents {
		tips.Select = append(tips.Select, tip(event, ""))
	}
	reset := l.Detail.ResetFor(f.Base().Profile.Language)
	restore := l.Detail.RestoreFor(f.Base().Profile.Language)

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
		Tips:      tips,
		TipClose:  f.Words.TipClose,
		TipToggle: f.Words.TipShowNext,
		TipArt:    f.tipArt(),
		TipsOn:    !f.TipsOff(),
		SetTipsOn: func(on bool) { f.SetTipsOff(!on) },

		ResetStart: reset.Values == "start",
		ResetSkill: reset.Skill,

		RestoreStart: restore.Values == "start",
		RestoreSkill: restore.Skill,
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
		if len(l.Detail.Commands) == 4 {
			setup.Detailed.Restore = encodeEngineWord(f.Words.Engine.Text("chargen.restore"), a.Selector)
		}
		setup.Preview = f.ChargenPreview
	}
	f.generatorSetup(&setup)
	return setup
}

// generator is the generator description the profile's edition names. Every
// edition names one.
func (f *FrontEnd) generator() *ui.GeneratorDescription {
	return GeneratorDescription(f.Base().Profile)
}

// generator is the generator description the install's profile names; an
// install assembled without archives reads the first game's edition.
func (in *InstallResources) generator() *ui.GeneratorDescription {
	var m base.Match
	if in != nil && in.Archives != nil {
		m = in.Archives.Base
	}
	return GeneratorDescription(m.Profile)
}

// generatorCost is the cumulative cost of a statistic standing at v:
// trunc(factor * base^(v-1) + round).
func generatorCost(c ui.GeneratorCost, v int) int {
	return int(math.Trunc(c.Factor*math.Pow(c.Base, float64(v-1)) + c.Round))
}

// generatorProfile is a game's hero generator row: where each picture's
// preset comes from, what the party producer reads and what an accepted
// result begins. Each game's gameFiles holds one; generatorSetup,
// generatorParty and generatorBegin read it for both games.
type generatorProfile struct {
	// templates makes each picture's Humans template row its statistic preset
	// and the hero's base row (R2-ENGINE-285, R2-ASSET-076). Without it the
	// preset is npc.reg's archetype section for the picture's sex and class,
	// which the original's Forward loads (HERO-STAT-001, HERO-CHARGEN-083;
	// DIV-1277), and the base row is data.ChargenBase's over f.Humans.
	templates bool
	// classProfile gives a hero without a base row his class's plain profile;
	// without it he keeps the constructor default (HERO-163, HERO-164).
	classProfile bool
	// campaignOutfit takes the hero's weapon and school skill levels from the
	// description's campaign part (R2-ENGINE-290; DIV-2772), and a description
	// without one makes no party. Without it the hero holds the trained slot's
	// weapon and keeps his row's skills.
	campaignOutfit bool
	// documents gives the hero the campaign's documents (DIV-305).
	documents bool
	// start is what an accepted result begins.
	start generatorStart
}

// generatorStart is what an accepted generator result begins.
type generatorStart int

const (
	// generatorStartMission opens the chosen mission with the generated party.
	generatorStartMission generatorStart = iota
	// generatorStartTown commits a new campaign at its first town and opens no
	// map, so the generator shows the town (R2-ENGINE-283).
	generatorStartTown
)

// generatorRow is the generator row of the front end's game.
func (f *FrontEnd) generatorRow() generatorProfile { return f.campaign().files().generator }

// generatorSetup gives each picture the preset its game's generator row
// names, and marks a start in town on the setup. A picture whose source row
// is missing, or whose archetype holds an illegal spread, keeps the generic
// start.
func (f *FrontEnd) generatorSetup(setup *ui.ChargenSetup) {
	p := f.generatorRow()
	setup.AcceptShowsTown = p.start == generatorStartTown
	for i, h := range f.generator().PreCreate.Heroes {
		if i >= len(setup.Presets) {
			break
		}
		if stats, skill, ok := f.generatorPreset(p, h); ok {
			setup.Presets[i] = stats
			setup.PresetSkills[i] = skill
		}
	}
}

// generatorPreset is picture h's four statistics and one-based skill under
// row p, or false when the source holds none. A template row states no skill.
func (f *FrontEnd) generatorPreset(p generatorProfile, h ui.GeneratorHero) ([]int, int, bool) {
	if p.templates {
		d, _, ok := generatorTemplate(missionHumans(f.Table), h.Template)
		return []int{int(d.Body), int(d.Reaction), int(d.Mind), int(d.Spirit)}, 0, ok
	}
	if f.Table == nil {
		return nil, 0, false
	}
	a, ok := f.Table.NPC.HeroArchetype(h.Sex != 0, h.Class != 0)
	spread := data.Spread{Body: a.Body, Reaction: a.Reaction, Mind: a.Mind, Spirit: a.Spirit}
	return []int{int(a.Body), int(a.Reaction), int(a.Mind), int(a.Spirit)}, int(a.Skill), ok && spread.Legal()
}

// generatorTemplate is the Humans row a hero picture names, by name.
func generatorTemplate(humans data.Collection, name string) (data.HumanDef, int, bool) {
	i := data.FindHumanByName(humans, name)
	if name == "" || i == data.NotFound {
		return data.HumanDef{}, data.NotFound, false
	}
	d, err := data.NewHumanDef(name, humans.EntryParams(i))
	if err != nil {
		return data.HumanDef{}, data.NotFound, false
	}
	return d, i, true
}

// heroTemplate is the template name of the picture standing for a sex and a
// class.
func heroTemplate(l *ui.GeneratorDescription, female, mage bool) string {
	for _, h := range l.PreCreate.Heroes {
		if (h.Sex != 0) == female && (h.Class != 0) == mage {
			return h.Template
		}
	}
	return ""
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
	if member.Mage && !tableFiles(f.Table).releasedSpellReadout {
		worn := mapload.MemberItemEquipment(member, f.Table)
		base, spread, ok := generatorCardDamage(worn[0], mapload.TableRules(f.Table), mapload.SpellRules(f.Table))
		preview.Subject.Combat.WeaponSpellDamageKnown = ok
		preview.Subject.Combat.SpellDamageBase, preview.Subject.Combat.SpellDamageSpread = base, spread
	}
	if f.Archives == nil {
		return preview
	}
	var mask *ui.SlotMask
	preview.Doll, mask = composeUnitFigure(f.Archives.Containers, equipmentFromSlots(member.Worn), memberFigureID(member))
	preview.Worn = &ui.ChargenDollHover{Mask: mask,
		Info: slotItemInfoLinesWithWeaponDamage(mapload.MemberItemEquipment(member, f.Table), f.Table, nil, f.Words)}
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
// the constructor-default profile, face 1 and no cells at all when the
// chosen archetype's row is absent, the hero the original builds there
// (HERO-163, HERO-164).
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
	return f.generatorParty(res)
}

// NewGameBegin is what an accepted new-game generator result does: the start
// its game's generator row names.
func (f *FrontEnd) NewGameBegin(n int) func(ui.ChargenResult) (ui.MapOpener, error) {
	return f.generatorBegin(f.generatorRow().start, n)
}

// generatorBegin is what an accepted result begins under start: mission n
// opened with the generated party, or a new campaign committed at its first
// town with no map opened.
func (f *FrontEnd) generatorBegin(start generatorStart, n int) func(ui.ChargenResult) (ui.MapOpener, error) {
	if start == generatorStartTown {
		return func(res ui.ChargenResult) (ui.MapOpener, error) { return nil, f.startGeneratedTown(res) }
	}
	return func(res ui.ChargenResult) (ui.MapOpener, error) { return f.NewGameOpener(n, res), nil }
}

// generatorParty is the party a result makes under its game's generator row.
// The chosen sex and class pick the base row; with campaignOutfit the weapon
// is the campaign part's for the chosen skill, the school skills are cleared
// and the fifth and the chosen skill stand at the part's levels
// (R2-ENGINE-290). Item modifiers and the experience the original computes
// are not carried (DIV-2772).
func (f *FrontEnd) generatorParty(res ui.ChargenResult) []mapload.PartyMember {
	p := f.generatorRow()
	var outfit *secondGenerator
	if p.campaignOutfit {
		g, err := f.secondGeneratorCampaign()
		if err != nil {
			return nil
		}
		outfit = g
	}
	female := chargenChoiceIndex(res, chargenChoiceSex) != 0
	mage := chargenChoiceIndex(res, chargenChoiceClass) != 0
	skill := chargenChoiceIndex(res, chargenChoiceSkill)
	slot := data.SkillBlade + int32(skill)

	humans := f.Humans
	var row data.HumanDef
	var i int
	var ok bool
	if p.templates {
		humans = missionHumans(f.Table)
		row, i, ok = generatorTemplate(humans, heroTemplate(f.generator(), female, mage))
	} else {
		row, i, ok = data.ChargenBase(humans, mage, female)
	}
	missing := constructorDefaultProfile()
	if p.classProfile {
		missing = data.Profile{Fighter: !mage, HealthColumn: true}
	}
	profile, face, book, cells := chargenProfile(row, ok, humans, i, missing)

	spread := data.Spread{
		Body:     chargenStatValue(res, chargenStatBody),
		Reaction: chargenStatValue(res, chargenStatReaction),
		Mind:     chargenStatValue(res, chargenStatMind),
		Spirit:   chargenStatValue(res, chargenStatSpirit),
	}

	var w *data.Weapon
	var weaponCell string
	if outfit != nil {
		w, weaponCell = generatorCampaignWeapon(outfit, f.Table, mage, female, skill)
	} else {
		shapes, materials, weapons := f.tableWeapons()
		w, _ = resolveWeaponForSlot(mage, female, shapes, materials, weapons, slot)
	}

	party := assembleParty(partyInputs{
		Name:       res.Name,
		Spread:     spread,
		Slot:       slot,
		Mage:       mage,
		Profile:    profile,
		Dir:        data.FigureDirFor(mage, female),
		Face:       face,
		Book:       book,
		Weapon:     w,
		WeaponCell: weaponCell,
		Cells:      cells,
		List:       f.Bodies,
		Table:      f.Table,
		Documents:  p.documents,
	})
	if outfit != nil {
		h := &party[0].Hero
		for s := data.SkillBlade; s < data.SkillSlots; s++ {
			h.Skill[s] = 0
		}
		h.Skill[data.SkillSlots-1] = outfit.FifthSkill
		if slot > data.SkillGeneral && slot < data.SkillSlots {
			h.Skill[slot] = outfit.ChosenSkill
		}
	}
	return party
}

// encodeEngineWord is an engine word in the install's code page, the form the
// fonts draw.
func encodeEngineWord(word string, selector int) string {
	return EncodeInstallText(word, selector)
}
