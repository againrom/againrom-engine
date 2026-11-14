package game

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/ui"
)

// modMaps is the open project's maps: the one list (mod.MapList) over the
// mod's files with the drafts and modMissions of the rows as they stand.
func (s *mapInspectionEdits) modMaps() []mod.MapEntry {
	d := s.project0()
	if d == nil {
		return nil
	}
	data, _ := s.inspector.modMissions(s.inspector.mod.description(), d.Missions)
	return mod.MapList(s.projectFiles(d), "", data)
}

// ownMap is the path in the mod of the map these edits hold: its row's map,
// or the map no row names; empty for none.
func (s *mapInspectionEdits) ownMap() string {
	if m, ok := s.ownMission(s.project0()); ok {
		return m.Map
	}
	return ""
}

// mapEntryOf is the place in list of the map at rel, -1 for none.
func mapEntryOf(list []mod.MapEntry, rel string) int {
	if rel == "" {
		return -1
	}
	fold := namefold.ModTree(rel)
	return slices.IndexFunc(list, func(e mod.MapEntry) bool {
		return slices.ContainsFunc(append(slices.Clone(e.Files), e.Lacking...), func(f string) bool { return namefold.ModTree(f) == fold })
	})
}

// mapRoles are the roles of map entry e in the editor's language: each row
// that names it, a replacement by its file name, unused or missing.
func (x *MapInspector) mapRoles(e mod.MapEntry) []string {
	switch e.Role() {
	case mod.MapMissing:
		return []string{x.word("editor.mod.map.missing")}
	case mod.MapByName:
		return []string{fmt.Sprintf(x.word("editor.mod.map.byname"), e.Number)}
	case mod.MapUnused:
		return []string{x.word("editor.mod.map.unused")}
	}
	var roles []string
	for _, m := range e.Rows {
		if m.Kind == mod.MissionGuest {
			roles = append(roles, fmt.Sprintf(x.word("editor.mod.map.guest"), m.Number, m.After))
		} else {
			roles = append(roles, fmt.Sprintf(x.word("editor.mod.map.replace"), m.Number))
		}
	}
	return roles
}

// ModMaps is the open project's list of maps; with no project it is empty.
func (s *mapInspectionEdits) ModMaps() ui.InspectionField {
	if s.project0() == nil {
		return ui.InspectionField{}
	}
	return s.modMapField()
}

// modMapField is the list of the mod's maps: one choice per map with its
// role, the map these edits hold chosen.
func (s *mapInspectionEdits) modMapField() ui.InspectionField {
	x := s.inspector
	f := ui.InspectionField{Key: "map", Label: x.word("editor.mod.maps"), ReadOnly: true}
	list := s.modMaps()
	for n, e := range list {
		name := strings.TrimPrefix(e.Path, mod.MapsFolder+"/")
		roles := x.mapRoles(e)
		f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(n + 1), Label: name + ": " + strings.Join(roles, ", ")})
		var lines []string
		for k, role := range roles {
			if k == 0 {
				role = name + ": " + role
			}
			if k < len(roles)-1 {
				role += ","
			}
			lines = append(lines, role)
		}
		f.ChoiceLines = append(f.ChoiceLines, lines)
	}
	f.Value = int64(mapEntryOf(list, s.ownMap()) + 1)
	return f
}

// openMapEntry opens map index of the mod's map list beside the open maps,
// through the switch the mission choice uses: the row that names the map,
// the chosen one when it is among them, or the map alone when none does.
func (s *mapInspectionEdits) openMapEntry(index int) (*ui.InspectionDocument, error) {
	x := s.inspector
	list := s.modMaps()
	if index < 1 || index > len(list) {
		return nil, errors.New(x.word("editor.mod.err.outside"))
	}
	e := list[index-1]
	if len(e.Files) == 0 {
		return nil, errors.New(x.word("editor.mod.err.mapmissing"))
	}
	rel := e.Files[0]
	if at := slices.Index(e.Languages, x.pageLanguage()); at >= 0 {
		rel = e.Files[at]
	}
	d := s.project0()
	row := -1
	for i, m := range d.Missions {
		if m.Off || namefold.ModTree(m.Map) != namefold.ModTree(rel) {
			continue
		}
		if row < 0 || i == d.Selected {
			row = i
		}
	}
	if row >= 0 {
		return s.openModMap(row+1, "")
	}
	return s.openModMap(0, rel)
}
