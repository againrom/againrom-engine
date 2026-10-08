package sim

import "fmt"

// oldestReadableVersion is the earliest byte-form version this build reads.
// Every older form is refused by name: no save the game produces carries one.
const oldestReadableVersion byte = 94

// CheckSaveForm reports whether data is a byte form this build reads. The
// forms it reads are formatVersion, its immediate predecessor and the optional
// continuation forms; an optional form is also decoded once so that a malformed
// continuation is refused here rather than at the later restore. Anything else
// is refused with a sentence written for the player: the load window puts the
// error's own text on its message line.
func CheckSaveForm(data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("sim: byte form is empty")
	}
	v := data[0]
	switch v {
	case formatVersion, oldestReadableVersion:
		return nil
	case autoHealingFormVersion, spellGraphFormVersion, currentAreaFormVersion, structureBlockingFormVersion,
		tacticalFormVersion, pendingOrderFormVersion, creatureSpellFormVersion, areaCostFormVersion, nativeTrainingFormVersion, rom2ScriptFormVersion, nativeClassFormVersion, bookSelectionFormVersion, nativeBasisFormVersion, playerParticipantFormVersion, currentPlayerFormVersion, nativeLiveFormVersion, nativeItemFormVersion, nativeScalarFormVersion, pursuitSearchFormVersion:
		var checked World
		return checked.UnmarshalBinary(data)
	}
	switch {
	case v < oldestReadableVersion:
		return fmt.Errorf("this save is too old to open: format %d. This build reads %d and later", v, oldestReadableVersion)
	case v > formatVersion:
		return fmt.Errorf("this save is newer than this build: format %d", v)
	}
	return fmt.Errorf("this save names format %d, which no version of this game ever wrote", v)
}
