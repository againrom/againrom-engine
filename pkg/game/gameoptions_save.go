package game

import (
	"fmt"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// PendingGameOption is an additive AGS field. Old saves carry none. Only the
// option rows with a queued Player parameter are kept (formation, retreat,
// global healing and the automatic-casting level); display flags ride View.
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
		for o, r := range gameOptionRows {
			if r.queue == nil || sim.PlayerParameter(c.X) != r.queue.parameter {
				continue
			}
			if v, ok := r.queue.value(c.Y); ok {
				out = append(out, PendingGameOption{ui.GameOption(o), v})
			}
		}
	}
	return out
}

func validatePendingGameOptions(orders []PendingGameOption) error {
	if len(orders) > 4096 {
		return fmt.Errorf("pending game options exceed 4096 commands")
	}
	for _, order := range orders {
		if int(order.Option) >= len(gameOptionRows) || gameOptionRows[order.Option].queue == nil || order.Value < 0 || order.Value >= order.Option.Choices() {
			return fmt.Errorf("invalid pending game option %d value %d", order.Option, order.Value)
		}
	}
	return nil
}
