package ui

// MapRetreat is the seam Ctrl+W leaves this package through. It carries no
// payload: this tier knows that the local participant asked to advance the
// persisted Off/Low/High setting, while the wiring tier owns both that setting
// and the participant's roster slot.
//
// Like MapFormation, it is installed directly on Viewer instead of widening
// MapLoader's aimed-order tuple. The setting still reaches simulation through
// the ordinary queued player-command path on the far side.
type MapRetreat func()

// SetRetreatSink replaces the retreat-command destination. Nil leaves Ctrl+W
// inert, which is the standalone viewer's lawful state: it has no process
// options and no local world to command.
func (v *Viewer) SetRetreatSink(fn MapRetreat) { v.retreatSink = fn }

func (v *Viewer) cycleRetreat() {
	if v.retreatSink != nil {
		v.retreatSink()
	}
}
