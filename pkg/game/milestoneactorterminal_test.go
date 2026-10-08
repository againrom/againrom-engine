package game

import (
	"fmt"
	"slices"
)

func (s *milestoneActorSubjects) terminal(archive uint16, off int, class string, identity, runtime uint32, object uint16, current *milestoneActorCurrent, stage byte, hp int16, position []byte) (bool, string) {
	if current == nil || current.Terminal == nil {
		return false, ""
	}
	m := s.mission
	if m == nil || m.World == nil || m.savedDocument != s.state || m.actorRegistry == nil || s.state == nil || s.state.Document == nil {
		return false, "terminal actor subject Mission/registry context unavailable"
	}
	sources := 0
	for _, source := range m.actorRegistry.sources {
		if source.Off != off {
			continue
		}
		sources++
		if source.ArchiveIndex != archive || source.Class != class || source.Identity != identity || source.RuntimeID != runtime || !source.TerminalActor {
			return false, "terminal actor subject source provenance differs"
		}
	}
	if sources > 1 {
		return false, "terminal actor subject source provenance ambiguous"
	}
	if identity == 0 || object == 0 {
		return false, "terminal actor subject ordinary dead root unavailable"
	}
	a, err := readCurrentActions(s.state.Document)
	if err != nil || a == nil {
		return false, "terminal actor subject restored binding unavailable"
	}
	restoredObject, restoredKey, err := currentTerminalActionObject(s.state, a, current.ID)
	if err != nil || restoredKey != identity || restoredObject == 0 {
		return false, "terminal actor subject restored ordinary binding differs"
	}
	r := &s.state.Document.Objects[restoredObject-1]
	key, keyErr := savedStructureValue(r, "Identity")
	rid, runtimeErr := savedStructureValue(r, "RuntimeID")
	if r.Class != class || keyErr != nil || key != identity || runtimeErr != nil || rid != runtime {
		return false, "terminal actor subject ordinary identity differs"
	}
	keys := 0
	for _, record := range s.state.Document.Objects {
		if key, err := savedStructureValue(&record, "Identity"); err == nil && key == identity {
			keys++
		}
	}
	if keys != 1 || !slices.Contains(s.state.Document.DeadActors, restoredObject) {
		return false, "terminal actor subject ordinary identity/root ambiguous"
	}
	if _, live := s.byID[current.ID]; live || s.usedIDs[current.ID] {
		return false, "terminal actor subject collides with live/repeated ID"
	}
	for _, dead := range m.World.OriginalDeadActors() {
		if dead.ID == current.ID {
			return false, "terminal actor subject collides with retained dead ID"
		}
	}
	owners := 0
	var difference string
	for _, terminal := range m.World.CurrentTerminalActors() {
		if terminal.ID != current.ID {
			continue
		}
		owners++
		if terminal != *current.Terminal || terminal.Stage != stage || terminal.HP != int32(hp) || len(position) != 4 || position[0] != byte(terminal.Cell) || position[1] != byte(terminal.Cell>>8) || position[2] != byte(terminal.Cell) || position[3] != byte(terminal.Cell>>8) {
			difference = fmt.Sprintf("terminal actor subject current tuple differs: manager=%+v input=%+v raw Stage=%d HP=%d Cells=%x", terminal, *current.Terminal, stage, hp, position)
		}
	}
	if owners != 1 {
		return false, "terminal actor subject manager owner unavailable"
	}
	s.usedIDs[current.ID] = true
	return true, difference
}
