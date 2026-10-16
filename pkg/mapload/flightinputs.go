package mapload

import "againrom/pkg/sim"

// BindFlightInputs restores installed release inputs after construction or LOAD.
func BindFlightInputs(w *sim.World, t *Table) {
	if w != nil && t != nil && t.FlightInputs != nil {
		w.SetFlightInputs(*t.FlightInputs)
	}
}
