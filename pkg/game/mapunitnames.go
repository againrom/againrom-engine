package game

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// unitNamesPage is the mission page naming placed units: one line
// "<unit id>=<name>" per named unit, the unit id the type-6 record's +0x40
// word. It lives with the mission's other pages: the editor holds it in
// UTF-8 in the map's texts folder, a mod serves it in the install's text
// bytes, and the game reads it there by mission number.
const unitNamesPage = "units.txt"

// unitNameField is the unit window's field of the authored name.
const unitNameField = "name"

// maxUnitNameRunes bounds an authored name, so it fits the card's name line.
const maxUnitNameRunes = 32

// unitNames reads a names page, UTF-8 or the font's bytes: each line's unit
// id and name, both cut of ASCII blanks. A line that is not "<id>=<name>"
// with an id of 1..65535 and a name names nothing; the first line for an id
// holds.
func unitNames(text string) map[uint16]string {
	const blanks = " \t\r\n"
	var out map[uint16]string
	for line := range strings.Lines(text) {
		id, name, ok := strings.Cut(line, "=")
		n, err := strconv.ParseUint(strings.Trim(id, blanks), 10, 16)
		name = strings.Trim(name, blanks)
		if !ok || err != nil || n == 0 || name == "" {
			continue
		}
		if out == nil {
			out = map[uint16]string{}
		}
		if _, held := out[uint16(n)]; !held {
			out[uint16(n)] = name
		}
	}
	return out
}

// unitNamesText is the page of names, one CRLF line per id in id order.
func unitNamesText(names map[uint16]string) []byte {
	var b bytes.Buffer
	for _, id := range slices.Sorted(maps.Keys(names)) {
		fmt.Fprintf(&b, "%d=%s\r\n", id, names[id])
	}
	return b.Bytes()
}

// missionUnitNames is the authored names of mission's placed units, in the
// font's bytes: the mission's names page read as its other pages are.
func missionUnitNames(src entrySource, code TextCode, mission int) map[uint16]string {
	return unitNames(installText(src, unitNamesSource.at(mission), code).text())
}

// authoredName is the name the map's names page gives entity e, placed as
// map unit e.MapUnitID; false for an entity the map did not place or did not
// name.
func (mw *mapWorld) authoredName(e *sim.Entity) (string, bool) {
	if e.MapUnitID == 0 {
		return "", false
	}
	name, ok := mw.authoredNames[e.MapUnitID]
	return name, ok
}

// workspaceStage is the workspace an edit in progress writes a unit's name
// or a copied trigger's page into: change adopts it with the edited model as
// one undo step. root is the
// edits value whose workspace the stage starts from.
type workspaceStage struct {
	root *mapInspectionEdits
	next *editorWorkspaceState
}

// workspace is the stage's workspace, a copy of the root's at the first
// write.
func (st *workspaceStage) workspace() (*editorWorkspaceState, error) {
	if st.next == nil {
		if err := st.root.prepareWorkspace(); err != nil {
			return nil, err
		}
		st.next = cloneWorkspace(st.root.workspace)
	}
	return st.next, nil
}

// staged reports whether the stage holds a workspace other than image.
func (st *workspaceStage) staged(image []byte) bool {
	return st != nil && st.next != nil && !bytes.Equal(workspaceBytes(st.next), image)
}

// authoredUnitNames is the names page the workspace holds, UTF-8.
func (s *mapInspectionEdits) authoredUnitNames() map[uint16]string {
	w := s.workspace
	if s.stage != nil && s.stage.next != nil {
		w = s.stage.next
	}
	if w == nil {
		if s.prepareWorkspace() != nil {
			return nil
		}
		w = s.workspace
	}
	return unitNames(string(w.Pages[unitNamesPage]))
}

// unitID is the unit id of unit index on model m.
func unitID(m *mapedit.Editor, index int) (uint16, error) {
	rec, err := m.UnitRecord(index)
	if err != nil {
		return 0, err
	}
	if len(rec) < alm.UnitID+2 {
		return 0, fmt.Errorf("unit %d has no unit id", index)
	}
	return uint16(rec[alm.UnitID]) | uint16(rec[alm.UnitID+1])<<8, nil
}

// unitNameAllowed refuses a name that is not 1..maxUnitNameRunes characters
// without a control character, or holds a character the install's font
// cannot show.
func (s *mapInspectionEdits) unitNameAllowed(name string) error {
	x := s.inspector
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) > maxUnitNameRunes || strings.ContainsFunc(name, unicode.IsControl) {
		return fmt.Errorf(x.word("editor.unit.name.refused"), maxUnitNameRunes)
	}
	selector := 0
	if x.font != nil {
		selector = x.font.Selector
	}
	if r, dropped := ModTextDropped([]byte(name), selector); dropped {
		return fmt.Errorf("this install cannot display %U", r)
	}
	return nil
}

// setUnitName stages unit index's authored name on model m into the names
// page: an empty name clears the unit's line, and a page left with no line
// is removed. It writes nothing to m.
func (s *mapInspectionEdits) setUnitName(m *mapedit.Editor, index int, name string) (bool, error) {
	name = strings.TrimSpace(name)
	if s.stage == nil {
		return false, errors.New("a unit's name is not set here")
	}
	id, err := unitID(m, index)
	if err != nil {
		return false, err
	}
	w, err := s.stage.workspace()
	if err != nil {
		return false, err
	}
	names := unitNames(string(w.Pages[unitNamesPage]))
	if names[id] == name {
		return false, nil
	}
	if id == 0 {
		return false, fmt.Errorf("unit %d has unit id 0, which no name can address", index)
	}
	if err := s.unitNameAllowed(name); err != nil {
		return false, err
	}
	if name == "" {
		delete(names, id)
	} else {
		if names == nil {
			names = map[uint16]string{}
		}
		names[id] = name
	}
	if len(names) == 0 {
		delete(w.Pages, unitNamesPage)
		return false, nil
	}
	w.Pages[unitNamesPage] = unitNamesText(names)
	return false, nil
}

// nameField is the unit window's name: the authored name, or empty with the
// name the game gives the unit's kind shown in its place.
func (s *mapInspectionEdits) nameField(index int) ui.InspectionField {
	x := s.inspector
	f := ui.InspectionField{Key: unitNameField, Label: x.word("editor.unit.field.name"), Text: true, Inherit: true,
		Meaning: x.word("editor.unit.name.note")}
	if id, err := unitID(s.model, index); err == nil {
		f.TextValue = s.authoredUnitNames()[id]
	}
	c := s.unitViewCache()
	if c.kindNames == nil {
		if _, _, err := s.unitWorld(c); err != nil {
			c.kindNames = map[int]string{}
		}
	}
	f.Inherited = c.kindNames[index]
	return f
}
