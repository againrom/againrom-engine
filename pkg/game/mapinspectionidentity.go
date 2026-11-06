package game

import (
	"fmt"
	"image"
	"math"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
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
	groups = append(groups,
		ui.InspectionChoiceGroup{Block: creatures, Head: utf(x.word("editor.unit.groups.othercreatures"))},
		ui.InspectionChoiceGroup{Block: people, Head: utf(x.word("editor.unit.groups.npc"))},
		ui.InspectionChoiceGroup{Block: people, Head: utf(x.word("editor.unit.groups.others"))},
		ui.InspectionChoiceGroup{Block: people, Head: utf(x.word("editor.unit.groups.heroes"))},
		ui.InspectionChoiceGroup{Block: people, Head: utf(x.word("editor.unit.groups.pc"))})
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
		case mapload.UnitPerson:
			name := x.table.Humans.EntryName(id.Row)
			group = other + 2
			if strings.HasPrefix(name, "NPC") {
				group = other + 1
			} else if strings.HasPrefix(name, "PC_") {
				group = other + 4
			}
		case mapload.UnitNamed:
			group = other + 3
		}
		groups[group].Values = append(groups[group].Values, c.Value)
	}
	var populated []ui.InspectionChoiceGroup
	for _, group := range groups {
		if len(group.Values) > 0 {
			populated = append(populated, group)
		}
	}
	return populated
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
		kind := mapload.UnitKind(math.MaxUint8)
		for _, id := range mapload.Identities(x.table, mission) {
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

// identityPictures are the pictures of the identity choices, by value: a
// creature's placed tier sprite, a person's
// figure as the face choice draws it, and a named character's portrait as a
// dialogue on this map shows it at mission start.
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
		if !ok || id.Kind == mapload.UnitNamed {
			continue
		}
		pictures[c.Value] = s.identityFigure(id)
	}
	out := map[int64]*image.RGBA{}
	var named []ui.InspectionChoice
	for _, c := range choices {
		if pic := pictures[c.Value]; pic != nil {
			out[c.Value] = pic
		}
		if c.Value >= 0 && mapload.UnitKind(c.Value>>32) == mapload.UnitNamed {
			named = append(named, ui.InspectionChoice{Value: int64(uint32(c.Value))})
		}
	}
	for npc, pic := range s.npcPortraits(named) {
		out[int64(mapload.UnitNamed)<<32|int64(uint32(npc))] = pic
	}
	return out
}

// identityFigure is a creature's or a person's picture, nil for none.
func (s *mapInspectionEdits) identityFigure(id mapload.Identity) *image.RGBA {
	x := s.inspector
	u, ok := mapload.Place(alm.Unit{}, id, x.table, x.identityMission)
	if !ok {
		return nil
	}
	if id.Kind == mapload.UnitCreature {
		tier, _ := placementTier(u, x.table)
		return s.unitPreview(u.ClassID, tier)
	}
	dir, face, ok := mapload.PlacedFigure(u, x.table)
	if !ok || x.archives == nil {
		return nil
	}
	doll, _ := composeDoll(x.archives.artSource(), 0, data.Equipment{}, nil, dir, face, false)
	return doll.Figure
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
// one edit. A placement that already is that identity is not edited.
func (s *mapInspectionEdits) setIdentity(m *mapedit.Editor, index int, value int64) (bool, error) {
	t := s.inspector.table
	id, ok := s.identityOf(value)
	if !ok || !t.ClassPlacementFields() {
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
