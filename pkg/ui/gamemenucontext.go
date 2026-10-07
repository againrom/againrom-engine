package ui

// GameMenuRelation is one live roster slot as the Diplomacy page presents it.
// State is already presentation vocabulary. The Client does not duplicate the
// simulation relation bit masks.
type GameMenuRelation struct {
	Slot  uint32
	State string
}

// GameMenuContext is the campaign/session projection read when the in-game menu
// opens. Campaign selects the decoded Load/Diplomacy and Quest gates. Objective
// and Relations supply the two owner-directed information pages.
type GameMenuContext struct {
	Campaign bool
	// CampaignVictory selects the decoded phase-2 End Quest constructor.
	// Hand-built viewers can retain the historical campaign root without
	// claiming they own the live completion seam.
	CampaignVictory  bool
	VictoryAvailable bool
	// LeaveToTown says the running mission was entered from the town and is
	// still undecided, so a mod entry may abandon or restart it.
	LeaveToTown bool
	Objective   string
	Relations   []GameMenuRelation
}

// GameMenuContextSource reads the current live game. A function is used so a
// relation changed by a mission script is read at menu open rather than copied
// when the map was loaded.
type GameMenuContextSource func() GameMenuContext

// GameMenuTipsSource and GameMenuTipsSink expose the existing persistent Tips
// Mode preference without making the Client know where it is stored.
type GameMenuTipsSource func() bool
type GameMenuTipsSink func(bool)

// GameMenuSoundSource and GameMenuSoundSink expose this process's current audio
// settings. available distinguishes a configurable device/archive pair from a
// silent installation. A sink error means the live change could not be saved.
type GameMenuSoundSource func() (enabled bool, volume int, available bool)
type GameMenuSoundSink func(enabled bool, volume int) error

// SetGameMenuContext installs the source associated with this map viewer.
func (v *Viewer) SetGameMenuContext(src GameMenuContextSource) {
	if v != nil {
		v.gameMenuContext = src
	}
}

func (v *Viewer) readGameMenuContext() GameMenuContext {
	if v == nil || v.gameMenuContext == nil {
		// A hand-built viewer has no campaign completion seam. It retains the
		// historical campaign menu root, but cannot claim the phase-2 Victory
		// constructor; production map doors always install an explicit source.
		return GameMenuContext{Campaign: true}
	}
	c := v.gameMenuContext()
	c.Relations = append([]GameMenuRelation(nil), c.Relations...)
	return c
}
