package ui

// MapFormation is the seam Ctrl+F leaves this package through. It carries no
// payload: this tier knows that the local participant asked to cycle Formation,
// while the wiring tier owns both that participant's roster slot and the
// canonical mode needed to choose the next authored setting.
//
// It is installed directly on Viewer rather than widening MapLoader's order
// tuple. The key is not an aimed map gesture and emits no entity order; this is
// SetAutocastSink's existing shape for a setting that still enters simulation
// through the ordinary queued command path.
type MapFormation func()

// SetFormationSink replaces the formation-command destination. Nil leaves the
// key inert, which is the standalone viewer's lawful state: it owns no world or
// local participant to command.
func (v *Viewer) SetFormationSink(fn MapFormation) { v.formationSink = fn }

// cycleFormation is Ctrl+F's whole UI effect. It keeps no mode and predicts no
// result; the next simulation push is the only state the frame can display.
func (v *Viewer) cycleFormation() {
	if v.formationSink != nil {
		v.formationSink()
	}
}
