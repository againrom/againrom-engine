package game

import (
	"bytes"
	"image"
	"strconv"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// armTriggers are the triggers that run an action of opcode, in file order. A
// trigger names its actions by id (ALM-TRIG-047), and an id names the first
// action that carries it, as the script panel reads them.
func armTriggers(sc alm.Script, opcode uint32) []int {
	action := map[uint32]uint32{}
	for _, n := range sc.Actions {
		if _, ok := action[n.ID]; !ok {
			action[n.ID] = n.Opcode
		}
	}
	var out []int
	for i, t := range sc.Triggers {
		for _, id := range t.Acts {
			if id != 0 && action[id] == opcode {
				out = append(out, i)
				break
			}
		}
	}
	return out
}

// victory is the rule for a map nothing wins: a map is won only by a trigger
// that runs instant 4 (TRIG-END-009, MISSION-WIN-003), so a map whose script
// holds none is endless.
func (c *mapCheck) victory() []mapHit {
	if c.scriptErr != nil || len(armTriggers(c.script, scriptOpWin)) > 0 {
		return nil
	}
	return []mapHit{{Text: "nothing wins this map: it is endless; no trigger runs instant 4, the one action that wins a map"}}
}

// mapSides counts the roster's sides: two players are on one side when each
// names the other an ally, and a side holds every player such pairs join.
func mapSides(groups []alm.Group) int {
	side := make([]int, len(groups))
	for i := range side {
		side[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for side[i] != i {
			i = side[i]
		}
		return i
	}
	for i := range groups {
		for j := i + 1; j < len(groups) && j < len(groups[i].Relation); j++ {
			if i < len(groups[j].Relation) && groups[i].Relation[j] == alliedRelation && groups[j].Relation[i] == alliedRelation {
				side[find(j)] = find(i)
			}
		}
	}
	n := 0
	for i := range side {
		if find(i) == i {
			n++
		}
	}
	return n
}

// alliedRelation is the roster relation word of an ally.
const alliedRelation = 2

// MapFacts states the map as the Map panel shows it. The answer is kept while
// the map bytes stay the same; nothing here writes the map.
func (s *mapInspectionEdits) MapFacts() ui.InspectionMapFacts {
	b := s.model.Bytes()
	if s.facts != nil && bytes.Equal(s.factsFor, b) {
		return *s.facts
	}
	f := mapFacts(s.model.Map())
	s.facts, s.factsFor = &f, b
	return f
}

func mapFacts(m *alm.Map, err error) ui.InspectionMapFacts {
	if err != nil {
		return ui.InspectionMapFacts{}
	}
	f := ui.InspectionMapFacts{Name: alm.MapNameText.Decode([]byte(m.Name)), Description: m.Description, Mode: m.Meta.Word70,
		Players: len(m.Groups), Sides: mapSides(m.Groups)}
	for _, d := range mapload.DropCells(m) {
		f.Drops = append(f.Drops, image.Pt(int(d.X), int(d.Y)))
	}
	sc, err := m.Script()
	if err != nil {
		return f
	}
	for _, i := range armTriggers(sc, scriptOpWin) {
		f.Wins = append(f.Wins, sc.Triggers[i].Name)
	}
	for _, i := range armTriggers(sc, scriptOpLose) {
		f.Losses = append(f.Losses, sc.Triggers[i].Name)
	}
	for _, n := range sc.Conditions {
		if n.Opcode == scriptCheckVIP {
			f.Protects++
		}
	}
	return f
}

// modeField is the metadata field of the map's mode, type-0 +0x70: the two
// values a new map is given by name and, when the map holds another, that
// value as itself so it is kept until the author picks another
// (ALM-MODE-070).
func (x *MapInspector) modeField(v uint32) ui.InspectionField {
	f := ui.InspectionField{Key: ui.ModeFieldKey, Label: x.inspectionUTF8(x.word("editor.panel.mode")), Value: int64(v)}
	f.Choices = []ui.InspectionChoice{
		{Value: int64(ui.MapModeSingle), Label: x.inspectionUTF8(x.word("editor.mode.single"))},
		{Value: int64(ui.MapModeMultiplayer), Label: x.inspectionUTF8(x.word("editor.mode.multi"))},
	}
	if v != ui.MapModeSingle && v != ui.MapModeMultiplayer {
		f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(v), Label: strconv.FormatUint(uint64(v), 10)})
	}
	return f
}
