package mapload

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

func bindCellTails(w *sim.World, m *alm.Map) error {
	bindings, err := m.CellBindings()
	if err != nil {
		return err
	}
	tails := make([]sim.CellTail, len(bindings))
	for i, b := range bindings {
		tails[i] = sim.CellTail{X: int32(b.X), Y: int32(b.Y),
			Bytes: [6]byte{b.Spell, b.Power, b.SourceX, b.SourceY, b.LastX, b.LastY}}
	}
	return w.DeclareCellTails(tails)
}
