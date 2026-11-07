package game

import (
	"fmt"
	"image"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The unit popup's "identity" field is who the unit is: one choice over
// mapload.Identities, its value the kind in the high word and in the low
// word the Units row of a creature, the Humans row of a person or the npc
// record of a named character. Negative values are the list's headings and
// the entry of a placement whose keys reach nothing.
const identityNone int64 = -1

func identityHeading(kind mapload.UnitKind) int64 { return -2 - int64(kind) }

func identityChoiceValue(id mapload.Identity) int64 {
	n := uint32(id.Row)
	if id.Kind == mapload.UnitNamed {
		n = uint32(id.NPC)
	}
	return int64(id.Kind)<<32 | int64(n)
}

// identityOf is the identity a choice value names among the choices last
// built, false for a heading or any value no kind has.
func (x *MapInspector) identityOf(v int64) (mapload.Identity, bool) {
	x.identityChoices(x.identityMission)
	id, ok := x.identities[v]
	return id, ok
}

// mission is the mission number test play starts this map as, which the
// npc records composed from the starting hero are read for.
func (s *mapInspectionEdits) mission() int { return s.inspector.testPlayNumber(s.workspace) }

// identityOf is the identity a choice value names in this map's mission.
func (s *mapInspectionEdits) identityOf(v int64) (mapload.Identity, bool) {
	s.inspector.identityChoices(s.mission())
	return s.inspector.identityOf(v)
}

// identityKindWords are each kind's words: its name in a unit's label and
// its heading in the identity choice.
var identityKindWords = map[mapload.UnitKind]struct{ kindWord, headingWord string }{
	mapload.UnitCreature: {kindWord: "editor.unit.kind.creature", headingWord: "editor.unit.identity.creatures"},
	mapload.UnitPerson:   {kindWord: "editor.unit.kind.person", headingWord: "editor.unit.identity.persons"},
	mapload.UnitNamed:    {kindWord: "editor.unit.kind.named", headingWord: "editor.unit.identity.named"},
}

var creatureIdentityGroups = []struct {
	word  string
	types []string
}{
	{word: "editor.unit.groups.catapults", types: []string{"Catapult", "Ballista"}},
	{word: "editor.unit.groups.goblins", types: []string{"Goblin_Pike", "Goblin_Sling"}},
	{word: "editor.unit.groups.orcs", types: []string{"Orc_Sword", "Orc_Bow"}},
	{word: "editor.unit.groups.ogres", types: []string{"Ogre", "Troll"}},
	{word: "editor.unit.groups.bats", types: []string{"Bat_Sonic", "Ghost", "Bee"}},
	{word: "editor.unit.groups.squirrels", types: []string{"Squirrel", "Foot_Animated", "Turtle"}},
	{word: "editor.unit.groups.dragons", types: []string{"Dragon"}},
	{word: "editor.unit.groups.daemon", types: []string{"Daemon"}},
}

func (x *MapInspector) identityGroups(choices []ui.InspectionChoice) []ui.InspectionChoiceGroup {
	utf := x.inspectionUTF8
	creatures, people := utf(x.word("editor.unit.groups.creatures")), utf(x.word("editor.unit.groups.people"))
	var groups []ui.InspectionChoiceGroup
	for _, group := range creatureIdentityGroups {
		groups = append(groups, ui.InspectionChoiceGroup{Block: creatures, Head: utf(x.word(group.word))})
	}
	other := len(groups)
	groups = append(groups, ui.InspectionChoiceGroup{Block: creatures, Head: utf(x.word("editor.unit.groups.othercreatures"))})
	for _, word := range personGroupWords {
		groups = append(groups, ui.InspectionChoiceGroup{Block: people, Head: utf(x.word(word))})
	}
	// Each head of the other people holds its men, then its women.
	women := map[int][]int64{}
	for _, c := range choices {
		id, ok := x.identities[c.Value]
		if !ok {
			continue
		}
		group := other
		switch id.Kind {
		case mapload.UnitCreature:
			name, _, _ := strings.Cut(x.table.Units.EntryName(id.Row), ".")
			for i, rule := range creatureIdentityGroups {
				for _, typ := range rule.types {
					if name == typ {
						group = i
					}
				}
			}
		case mapload.UnitPerson, mapload.UnitNamed:
			role, head, female := x.personRole(id)
			group = other + 1 + int(role)
			if role == personOther {
				group = slices.IndexFunc(groups, func(g ui.InspectionChoiceGroup) bool { return g.Block == people && g.Head == utf(head) })
				if group < 0 {
					group = len(groups)
					groups = append(groups, ui.InspectionChoiceGroup{Block: people, Head: utf(head)})
				}
			}
			if female {
				women[group] = append(women[group], c.Value)
				continue
			}
		}
		groups[group].Values = append(groups[group].Values, c.Value)
	}
	var populated []ui.InspectionChoiceGroup
	for i, group := range groups {
		group.Values = append(group.Values, women[i]...)
		if len(group.Values) > 0 {
			populated = append(populated, group)
		}
	}
	return populated
}

// personRole is a group of the people block: story heroes (named
// characters), player characters (the PC_ rows), mercenaries (the tavern's
// NPC<type>_<level> rows) and the other Humans rows, split by the class sheet
// the mission draws them with.
type personRole uint8

const (
	personStory personRole = iota
	personPlayer
	personMercenary
	personFighter
	personMage
	personRider
	personOther
)

// personGroupWords are the heads of the people block's fixed groups, in
// personRole's order; a class that fits none of them heads its own group
// after them.
var personGroupWords = []string{"editor.unit.groups.story", "editor.unit.groups.player", "editor.unit.groups.mercenaries",
	"editor.unit.groups.fighters", "editor.unit.groups.mages", "editor.unit.groups.riders"}

// personRole is the people group of person or named identity id, the head
// of its own group for a class that fits no fixed one, and whether an other
// person is a woman. An other Humans row is grouped by the type id the
// mission draws it with (the client hero type the second game gives it, or
// the row's): a horse type is a rider, the figure's mage class a mage, any
// other figure type a fighter; a type that composes no figure is its class.
func (x *MapInspector) personRole(id mapload.Identity) (personRole, string, bool) {
	if id.Kind == mapload.UnitNamed {
		return personStory, "", false
	}
	name := x.table.Humans.EntryName(id.Row)
	if strings.HasPrefix(name, "PC_") {
		return personPlayer, "", false
	}
	if _, _, ok := mercenaryRowFromName(name); ok {
		return personMercenary, "", false
	}
	u, _ := mapload.Place(alm.Unit{}, id, x.table, x.identityMission)
	typeID, _ := data.HumanTypeID(x.table.Humans.EntryParams(id.Row))
	if hero, ok := mapload.PlacedClientHero(u, x.table); ok {
		typeID = hero
	}
	dir, _, _ := mapload.PlacedFigure(u, x.table)
	switch {
	case !data.ComposesFigure(typeID) || data.FigureIsHero(typeID):
		class := fmt.Sprint(typeID)
		if c := x.personDraws()[identityChoiceValue(id)].class; c != nil && c.Name != "" {
			class = c.Name
		}
		return personOther, fmt.Sprintf(x.word("editor.unit.groups.otherclass"), class), dir.Female()
	case data.FigureHasHorse(typeID):
		return personRider, "", dir.Female()
	case dir.Mage():
		return personMage, "", dir.Female()
	}
	return personFighter, "", dir.Female()
}

// mercenaryCell is the Humans row of mercenary type typ's one choice: its
// lowest level among the identities last built.
func (x *MapInspector) mercenaryCell(typ int) (int, bool) {
	levels := x.mercenaryLevels[typ]
	if len(levels) == 0 {
		return 0, false
	}
	return levels[slices.Min(slices.Collect(maps.Keys(levels)))], true
}

// mercenaryOf is the mercenary type and level of person identity id; false
// for an identity that is no mercenary's row.
func (x *MapInspector) mercenaryOf(id mapload.Identity) (typ, level int, ok bool) {
	if id.Kind != mapload.UnitPerson || x.table == nil {
		return 0, 0, false
	}
	return mercenaryRowFromName(x.table.Humans.EntryName(id.Row))
}

// identityName is who id is, in the font's bytes: a creature's Units row
// name with its variant where its class has several, a person's Humans row
// name, a named character's npc record name and number. A record the start
// composes from the starting hero (Row NotFound) has no name of its own:
// the game names it by the hero too.
func (x *MapInspector) identityName(id mapload.Identity) string {
	enc := x.inspectionUTF8
	switch id.Kind {
	case mapload.UnitCreature:
		name := x.table.Units.EntryName(id.Row)
		u, _ := mapload.Place(alm.Unit{}, id, x.table, x.identityMission)
		if len(x.tiersOf(u)) > 1 {
			return fmt.Sprintf(enc(x.word("editor.unit.identity.variant")), name, u.ClassSubID)
		}
		return name
	case mapload.UnitNamed:
		name := x.table.Humans.EntryName(id.Row)
		if id.Row == data.NotFound {
			name = enc(x.word("editor.unit.identity.byhero"))
		} else if e, ok := x.table.NPCCatalog.Entry(id.NPC); ok {
			name = enc(x.npcName(e, false))
		}
		return fmt.Sprintf(enc(x.word("editor.unit.identity.record")), name, id.NPC)
	}
	return x.table.Humans.EntryName(id.Row)
}

// unitLabel is the one name the editor gives placed unit u, in the font's
// bytes: who it is, in the identity choice's words, its number and its
// cell; a unit whose keys reach nothing in a start of mission is named by
// its class key.
func (x *MapInspector) unitLabel(u alm.Unit, mission int) string {
	enc := x.inspectionUTF8
	cell := unitCell(u)
	id, _, ok := mapload.IdentifyAt(u, x.table, mission)
	if !ok || x.table == nil {
		return fmt.Sprintf(enc(x.word("editor.unit.label.none")), u.ClassID, u.UnitID, cell.X, cell.Y)
	}
	return fmt.Sprintf(enc(x.word("editor.unit.label")), enc(x.word(identityKindWords[id.Kind].kindWord)), x.identityName(id), u.UnitID, cell.X, cell.Y)
}

// identityChoices are the identity field's choices, each kind under its
// heading, and the headings the list shows but does not take, in a start of
// mission.
func (x *MapInspector) identityChoices(mission int) ([]ui.InspectionChoice, []int64) {
	if x.recordOptions == nil {
		x.recordOptions = map[string][]ui.InspectionChoice{}
	}
	choices, ok := x.recordOptions["identity"]
	if !ok || x.identities == nil || x.identityMission != mission {
		choices, x.identities, x.identityMission = nil, map[int64]mapload.Identity{}, mission
		x.mercenaryLevels, x.identityDraws = map[int]map[int]int{}, nil
		ids := mapload.Identities(x.table, mission)
		for _, id := range ids {
			if typ, level, ok := x.mercenaryOf(id); ok {
				if x.mercenaryLevels[typ] == nil {
					x.mercenaryLevels[typ] = map[int]int{}
				}
				x.mercenaryLevels[typ][level] = id.Row
			}
		}
		kind := mapload.UnitKind(math.MaxUint8)
		for _, id := range ids {
			// A mercenary type is one choice, its lowest level; the unit
			// popup's level field picks the others.
			if typ, _, ok := x.mercenaryOf(id); ok {
				if cell, _ := x.mercenaryCell(typ); cell != id.Row {
					continue
				}
			}
			if id.Kind != kind {
				kind = id.Kind
				choices = append(choices, ui.InspectionChoice{Value: identityHeading(kind), Label: x.inspectionUTF8(x.word(identityKindWords[kind].headingWord))})
			}
			v := identityChoiceValue(id)
			choices = append(choices, ui.InspectionChoice{Value: v, Label: x.identityName(id)})
			x.identities[v] = id
		}
		x.recordOptions["identity"] = choices
		delete(x.recordPictures, "identity")
	}
	var shut []int64
	for _, c := range choices {
		if c.Value < 0 {
			shut = append(shut, c.Value)
		}
	}
	return append([]ui.InspectionChoice(nil), choices...), shut
}

// identityPictures are the pictures of the identity choices, by value: the
// sprite the map draws each one with, a creature's at its placed tier and a
// person's or named character's as the mission draws it (personDraws).
func (s *mapInspectionEdits) identityPictures(choices []ui.InspectionChoice) map[int64]*image.RGBA {
	x := s.inspector
	if x.recordPictures == nil {
		x.recordPictures = map[string]map[int64]*image.RGBA{}
	}
	pictures, ok := x.recordPictures["identity"]
	if !ok {
		pictures = map[int64]*image.RGBA{}
		x.recordPictures["identity"] = pictures
	}
	for _, c := range choices {
		if _, done := pictures[c.Value]; done {
			continue
		}
		id, ok := x.identityOf(c.Value)
		if !ok {
			continue
		}
		pictures[c.Value] = s.identityFigure(id)
	}
	out := map[int64]*image.RGBA{}
	for _, c := range choices {
		if pic := pictures[c.Value]; pic != nil {
			out[c.Value] = pic
		}
	}
	return out
}

// identityFigure is the sprite the map draws identity id with, nil for none.
func (s *mapInspectionEdits) identityFigure(id mapload.Identity) *image.RGBA {
	x := s.inspector
	if id.Kind != mapload.UnitCreature {
		draw := x.personDraws()[identityChoiceValue(id)]
		return classPreview(draw.class, draw.tier)
	}
	u, ok := mapload.Place(alm.Unit{}, id, x.table, x.identityMission)
	if !ok {
		return nil
	}
	tier, _ := placementTier(u, x.table)
	return s.unitPreview(u.ClassID, tier)
}

// personDraw is the class sheet the mission draws a person with and the
// tier it draws him at.
type personDraw struct {
	class *terrain.UnitClass
	tier  int
}

// personDraws are the class sheets the mission draws each person and named
// identity of the choices last built with, by value: every one placed on one
// new map, the map started as a mission starts it (on the table a composed
// named character's start resolves him on, mapload.PlaceOn), its look
// settled as a mission's opening settles it and its art taken from
// missionAppearanceArt. An identity with no entry draws nothing.
func (x *MapInspector) personDraws() map[int64]personDraw {
	if x.identityDraws != nil {
		return x.identityDraws
	}
	x.identityDraws = map[int64]personDraw{}
	type start struct {
		table  *mapload.Table
		values []int64
		units  []alm.Unit
	}
	var starts []*start
	values := slices.Sorted(maps.Keys(x.identities))
	for _, v := range values {
		id := x.identities[v]
		if id.Kind == mapload.UnitCreature {
			continue
		}
		u, on, ok := mapload.PlaceOn(alm.Unit{}, id, x.table, x.identityMission)
		if !ok {
			continue
		}
		i := slices.IndexFunc(starts, func(s *start) bool { return s.table == on })
		if i < 0 {
			i = len(starts)
			starts = append(starts, &start{table: on})
		}
		starts[i].values, starts[i].units = append(starts[i].values, v), append(starts[i].units, u)
	}
	src := x.archives.artSource()
	list, _ := ReadBodyList(src)
	for _, st := range starts {
		// A start one placement refuses is started again one placement a
		// time, so the refused one alone draws nothing.
		parts := [][]int{make([]int, len(st.units))}
		for i := range st.units {
			parts[0][i] = i
		}
		for len(parts) > 0 {
			part := parts[0]
			parts = parts[1:]
			units := make([]alm.Unit, len(part))
			for i, n := range part {
				units[i] = st.units[n]
			}
			art, tiers, err := x.peopleArt(units, st.table, src, list)
			if err != nil && len(part) > 1 {
				for _, n := range part {
					parts = append(parts, []int{n})
				}
				continue
			}
			for i, n := range part {
				x.identityDraws[st.values[n]] = personDraw{class: art[sim.EntityID(i)], tier: tiers[sim.EntityID(i)]}
			}
		}
	}
	return x.identityDraws
}

// peopleArt is the class each of units is drawn with when a mission on table
// t starts them on one new map, and the tier each is drawn at, by entity id
// (the placement's index).
func (x *MapInspector) peopleArt(units []alm.Unit, t *mapload.Table, src terrain.EntrySource, list data.BodyList) (map[sim.EntityID]*terrain.UnitClass, map[sim.EntityID]int, error) {
	side := 32
	for (side-4)*(side-4) < len(units) {
		side += 8
	}
	spec := alm.DefaultNewMap()
	spec.Width, spec.Height = side, side
	model, err := mapedit.NewMap(spec)
	if err != nil {
		return nil, nil, err
	}
	if _, err := placeNewUnit(model, image.Pt(2, 2), 0, 0); err != nil {
		return nil, nil, err
	}
	m, err := model.Map()
	if err != nil {
		return nil, nil, err
	}
	blank := m.Units[0]
	m.Units, m.AuthoredUnits = nil, nil
	for i, u := range units {
		keyed := blank
		keyed.ClassID, keyed.ClassSubID, keyed.DefID, keyed.Flags, keyed.ServerID = u.ClassID, u.ClassSubID, u.DefID, u.Flags, u.ServerID
		keyed.X, keyed.Y = cellWords(image.Pt(2+i%(side-4), 2+i/(side-4)), newUnitFraction, newUnitFraction)
		keyed.UnitID = uint16(i + 1)
		m.Units = append(m.Units, keyed)
	}
	w, st, err := mapload.StartCampaignMission(m, t, openDifficulty, nil, x.identityMission)
	if err != nil {
		return nil, nil, err
	}
	ms := &Mission{Number: x.identityMission, Map: m, World: w, Start: st}
	canonicalizePartyAppearance(ms, x.units, src, list)
	return missionAppearanceArt(ms, x.units, list, src, placedClientHeroes(ms, t)), entityTiers(m, t), nil
}

// identityPicture is the picture of the identity choice value v.
func (s *mapInspectionEdits) identityPicture(v int64) *image.RGBA {
	return s.identityPictures([]ui.InspectionChoice{{Value: v}})[v]
}

// identityField is the unit popup's who-is-this choice for placement u: the
// identity mapload.IdentifyAt answers in this map's mission, every identity
// the key rule can write, and what the choice means. A named character the
// start composes from the starting hero says which row each hero gets.
func (s *mapInspectionEdits) identityField(u alm.Unit) ui.InspectionField {
	x := s.inspector
	mission := s.mission()
	choices, shut := x.identityChoices(mission)
	f := ui.InspectionField{Key: "identity", Label: x.word("editor.unit.field.identity"), Choices: choices, Shut: shut, IconTip: true,
		Meaning: x.word("editor.unit.identity.note")}
	if id, rows, ok := mapload.IdentifyAt(u, x.table, mission); ok {
		f.Value = identityChoiceValue(id)
		if typ, _, merc := x.mercenaryOf(id); merc {
			if row, ok := x.mercenaryCell(typ); ok {
				f.Value = identityChoiceValue(mapload.Identity{Kind: mapload.UnitPerson, Row: row})
			}
		}
		if len(rows) > 0 {
			f.Meaning = fmt.Sprintf(x.word("editor.unit.identity.composed"), mission, x.heroRowsText(rows)) + " " + f.Meaning
		}
	} else {
		f.Value, f.Meaning = identityNone, x.word("editor.unit.identity.choose")
		f.Choices = append([]ui.InspectionChoice{{Value: identityNone, Label: x.inspectionUTF8(x.word("editor.unit.identity.none"))}}, f.Choices...)
		f.Shut = append(f.Shut, identityNone)
	}
	f.Icons = s.identityPictures(f.Choices)
	return f
}

// levelField is the unit popup's level choice for a mercenary placement u:
// the levels its type has a Humans row for, its own chosen. False for a
// placement that is no mercenary's row.
func (s *mapInspectionEdits) levelField(u alm.Unit) (ui.InspectionField, bool) {
	x := s.inspector
	mission := s.mission()
	x.identityChoices(mission)
	id, _, ok := mapload.IdentifyAt(u, x.table, mission)
	typ, level, merc := x.mercenaryOf(id)
	if !ok || !merc || len(x.mercenaryLevels[typ]) == 0 {
		return ui.InspectionField{}, false
	}
	f := ui.InspectionField{Key: "level", Label: x.word("editor.unit.field.level"), Value: int64(level), Meaning: x.word("editor.unit.level.note")}
	for _, l := range slices.Sorted(maps.Keys(x.mercenaryLevels[typ])) {
		f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(l), Label: strconv.Itoa(l)})
	}
	return f, true
}

// setMercenaryLevel keys mercenary placement index of m to its type's Humans
// row of level, through the identity field's writer as one edit.
func (s *mapInspectionEdits) setMercenaryLevel(m *mapedit.Editor, index int, level int64) (bool, error) {
	x := s.inspector
	view, err := m.Map()
	if err != nil {
		return false, err
	}
	if index < 0 || index >= len(view.AuthoredUnits) {
		return false, fmt.Errorf("no unit %d", index)
	}
	mission := s.mission()
	x.identityChoices(mission)
	id, _, ok := mapload.IdentifyAt(view.AuthoredUnits[index], x.table, mission)
	typ, _, merc := x.mercenaryOf(id)
	if !ok || !merc {
		return false, fmt.Errorf("only a mercenary has a level")
	}
	row, ok := x.mercenaryLevels[typ][int(min(max(level, 0), math.MaxInt32))]
	if !ok {
		return false, fmt.Errorf("mercenary type %d has no level %d", typ, level)
	}
	return s.keyIdentity(m, index, mapload.Identity{Kind: mapload.UnitPerson, Row: row})
}

// heroRowsText is each kind of starting hero with the Humans row a start
// gives a composed named character for it, in the dialogue editor's hero
// words: "Man fighter: PC_Naira_2; ...".
func (x *MapInspector) heroRowsText(rows []mapload.HeroRow) string {
	parts := make([]string, len(rows))
	for i, r := range rows {
		hero := x.word(ui.HeroWord(ui.HeroOf(r.Hero.Mage(), r.Hero.Female())))
		parts[i] = hero + ": " + x.messageUTF8([]byte(x.table.Humans.EntryName(r.Row)))
	}
	return strings.Join(parts, "; ")
}

// keysField is the unit popup's read-only line of placement u's map keys,
// each with what it does on the rung Resolve takes or that it is not read
// there (MISSION-ARM-006).
func (x *MapInspector) keysField(u alm.Unit) (ui.InspectionField, bool) {
	if !x.table.ClassPlacementFields() {
		return ui.InspectionField{}, false
	}
	enc := x.inspectionUTF8
	arm := mapload.Resolve(u, x.table).Arm
	unread := x.word("editor.unit.key.unread")
	class, sub, flag, def := x.word("editor.unit.key.class.person"), x.word("editor.unit.key.sub.face"), x.word("editor.unit.key.npcflag.off"), unread
	switch arm {
	case mapload.ArmUnits:
		class, sub, flag = x.word("editor.unit.key.class.creature"), x.word("editor.unit.key.sub.variant"), unread
	case mapload.ArmNPC:
		sub, flag = x.word("editor.unit.key.sub.npc"), x.word("editor.unit.key.npcflag.on")
	case mapload.ArmServerID:
		def = x.word("editor.unit.key.definition.id")
	case mapload.ArmHumansByType:
		class, def = x.word("editor.unit.key.class.type"), x.word("editor.unit.key.definition.none")
	}
	line := func(format string, v int64, meaning string) ui.InspectionLine {
		return ui.InspectionLine{Text: enc(fmt.Sprintf(format, v, meaning))}
	}
	return ui.InspectionField{Key: "keys", Label: x.word("editor.unit.keys"), ReadOnly: true, Lines: []ui.InspectionLine{
		{Text: enc(x.word("editor.unit.keys")), Role: "heading"},
		line(x.word("editor.unit.key.class"), int64(u.ClassID), class),
		line(x.word("editor.unit.key.sub"), int64(u.ClassSubID), sub),
		line(x.word("editor.unit.key.npcflag"), int64(u.Flags&1), flag),
		line(x.word("editor.unit.key.definition"), int64(u.DefID), def),
	}}, true
}

// speakerField is the unit popup's read-only line saying whether a dialogue
// at mission start shows placement unit, as which speaker record, or why
// not: UnitSpeaker's answer in the speaker pick's words.
func (s *mapInspectionEdits) speakerField(unit int) ui.InspectionField {
	x := s.inspector
	enc := x.inspectionUTF8
	n, miss := s.UnitSpeaker(unit)
	record := func(id int64) string {
		label, _ := x.speakerLabel(id)
		return fmt.Sprintf("%d: %s", id, enc(label))
	}
	var text string
	if miss.Cause == ui.SpeakerShown {
		text = fmt.Sprintf(enc(x.word("editor.unit.speaker.shown")), record(n))
	} else {
		text = ui.SpeakerMissText(func(id string) string { return enc(x.word(id)) }, &miss, record(miss.Record))
	}
	return ui.InspectionField{Key: "speaker", Label: x.word("editor.unit.speaker"), ReadOnly: true, Lines: []ui.InspectionLine{
		{Text: enc(x.word("editor.unit.speaker")), Role: "heading"}, {Text: text}}}
}

// setIdentity keys placement index of m to the identity choice value: the
// keys mapload.Place writes for it, through the placement's field writer as
// one edit. A placement that already is that identity, or a mercenary of any
// level given his type's one choice, is not edited.
func (s *mapInspectionEdits) setIdentity(m *mapedit.Editor, index int, value int64) (bool, error) {
	t := s.inspector.table
	id, ok := s.identityOf(value)
	if !ok || !t.ClassPlacementFields() {
		return false, fmt.Errorf("choose who this unit is: a creature, a person or a named character")
	}
	if view, err := m.Map(); err == nil && index >= 0 && index < len(view.AuthoredUnits) {
		was, _, _ := mapload.IdentifyAt(view.AuthoredUnits[index], t, s.mission())
		if typ, _, merc := s.inspector.mercenaryOf(was); merc {
			if row, _ := s.inspector.mercenaryCell(typ); id == (mapload.Identity{Kind: mapload.UnitPerson, Row: row}) {
				return false, nil
			}
		}
	}
	return s.keyIdentity(m, index, id)
}

// keyIdentity keys placement index of m to identity id: setIdentity's
// writer, which the level field shares.
func (s *mapInspectionEdits) keyIdentity(m *mapedit.Editor, index int, id mapload.Identity) (bool, error) {
	t := s.inspector.table
	if !t.ClassPlacementFields() {
		return false, fmt.Errorf("choose who this unit is: a creature, a person or a named character")
	}
	view, err := m.Map()
	if err != nil {
		return false, err
	}
	if index < 0 || index >= len(view.AuthoredUnits) {
		return false, fmt.Errorf("no unit %d", index)
	}
	u := view.AuthoredUnits[index]
	want, ok := mapload.Place(u, id, t, s.mission())
	if !ok {
		return false, fmt.Errorf("the installed tables cannot key a unit to that choice")
	}
	var sets []mapedit.FieldSet
	if want.ClassID != u.ClassID {
		sets = append(sets, mapedit.FieldSet{Key: "class", Value: int64(want.ClassID)})
	}
	if want.ClassSubID != u.ClassSubID {
		sets = append(sets, mapedit.FieldSet{Key: "tier", Value: int64(want.ClassSubID)})
	}
	if want.DefID != u.DefID {
		sets = append(sets, mapedit.FieldSet{Key: "definition", Value: int64(want.DefID)})
	}
	if want.Flags&1 != u.Flags&1 {
		sets = append(sets, mapedit.FieldSet{Key: "npcflag", Value: int64(want.Flags & 1)})
	}
	if want.Flags&^1 != u.Flags&^1 || want.ServerID != u.ServerID {
		return false, fmt.Errorf("that choice needs a key this map's unit records do not hold")
	}
	if len(sets) == 0 {
		return false, nil
	}
	was := s.footprintOf(m, mapedit.KindUnit, index)
	if err := m.SetFields(mapedit.KindUnit, index, sets...); err != nil {
		return false, err
	}
	return true, s.checkPlacement(admissionOf(ui.KindUnit, admitField), mapedit.KindUnit, m, index, was)
}
