package game

import (
	"fmt"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// PendingGameOption is an additive AGS field. Old saves carry none. Only
// formation, retreat and global healing are queued; display flags ride View.
type PendingGameOption struct {
	Option ui.GameOption
	Value  int
}

func pendingGameOptions(commands []sim.Command) []PendingGameOption {
	var out []PendingGameOption
	for _, c := range commands {
		if c.Kind != sim.KindPlayerParameter || c.Player != sim.SelfSlot {
			continue
		}
		switch sim.PlayerParameter(c.X) {
		case sim.PlayerParameterFormation:
			out = append(out, PendingGameOption{ui.GameOptionFormation, int(c.Y)})
		case sim.PlayerParameterRetreat:
			out = append(out, PendingGameOption{ui.GameOptionRetreat, int(c.Y)})
		case sim.PlayerParameterAutoHealing:
			out = append(out, PendingGameOption{ui.GameOptionAutoHealing, int(c.Y)})
		}
	}
	return out
}

func validatePendingGameOptions(orders []PendingGameOption) error {
	if len(orders) > 4096 {
		return fmt.Errorf("pending game options exceed 4096 commands")
	}
	for _, order := range orders {
		if (order.Option != ui.GameOptionFormation && order.Option != ui.GameOptionRetreat && order.Option != ui.GameOptionAutoHealing) || order.Value < 0 || order.Value > 2 {
			return fmt.Errorf("invalid pending game option %d value %d", order.Option, order.Value)
		}
	}
	return nil
}
