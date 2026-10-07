package sim

import "fmt"

// OriginalSession is the supported canonical state imported from an original
// mid-mission save. The Player outcome is independent of the session counters:
// primary-hero loss can report failure without a LOSE increment (SAV-FLAG-027).
//
// Registers is the file's own hundred trigger-result slots (SAV-SESS-031,
// TRIG-STORE-002); ImportOriginalSession restores it at the point the
// original does, before the constant half of the trigger rebuild presets
// over it (TRIG-SAVE-008, MISSION-SLOT-008 — see presetRegisters and
// docs/1130/story.md for the per-slot-class outcome). RawHead and RawMid are
// the two session spans SAV-SESS-031 locates but promotes no meaning for;
// they are carried opaquely and never interpreted.
type OriginalSession struct {
	HasClock       bool
	Clock          SessionClock
	Registers      [scriptRegisters]int32
	TriggerLatches []byte
	RawHead        [sessionRawHeadLen]byte
	RawMid         [sessionRawMidLen]byte
	Diplomacy      []byte
	Won, Lost      uint32
	Outcome        Outcome
}

// ValidateOriginalSession checks the whole foreign handoff without changing a
// world. It is exported so a game-level load can reject a malformed session
// before it commits any front-end state; ImportOriginalSession repeats the same
// validation at the mutation boundary rather than trusting its caller.
//
// Registers, RawHead and RawMid have no check here BY DESIGN: they are fixed
// arrays, so there is no length to police, and every signed int32 or byte
// pattern is a value the original file can legitimately hold.
func ValidateOriginalSession(s OriginalSession) error {
	if !s.HasClock && s.Clock != (SessionClock{}) {
		return fmt.Errorf("sim: absent original session clock carries state")
	}
	if !s.Outcome.defined() {
		return fmt.Errorf("sim: original Player outcome %d is not defined", s.Outcome)
	}
	if len(s.TriggerLatches) != scriptLatches {
		return fmt.Errorf("sim: original session carries %d trigger latch byte(s), want %d",
			len(s.TriggerLatches), scriptLatches)
	}
	for i, v := range s.TriggerLatches {
		if v != 0 && v != 1 {
			return fmt.Errorf("sim: original session trigger latch %d is %d, want 0 or 1", i, v)
		}
	}
	if len(s.Diplomacy) != relationLen {
		return fmt.Errorf("sim: original session carries %d diplomacy byte(s), want %d for %d slots",
			len(s.Diplomacy), relationLen, relationSlots)
	}
	return nil
}

// ImportOriginalSession replaces registers, latches, diplomacy, counters, the
// two raw spans and the supplied clock pair transactionally. All inputs are
// validated before any assignment, so an error leaves canonical state
// unchanged. An absent clock is a partial handoff from an older caller, not
// an instruction to reset a world's clock.
//
// The trigger programme is not rebuilt: a mission is first compiled from its
// map — which runs presetRegisters once, over the map's own constants — and
// then this method overlays the saved session on top of that, including a
// second presetRegisters call. TRIG-SAVE-008 (corrected) orders the original
// the other way — restore, then rebuild — but the two are equivalent for
// every slot presetRegisters ever touches, since it only ever assigns a
// constant node's own compiled value, never a value read off any other
// slot; running it again after the copy lands each constant slot on the
// value a real rebuild would have given it, which is exactly the value it
// already holds from THIS world's own compile. See presetRegisters and
// docs/1130/story.md for the outcome by slot class.
func (w *World) ImportOriginalSession(s OriginalSession) error {
	if w == nil {
		return fmt.Errorf("sim: import original session into a nil world")
	}
	if err := ValidateOriginalSession(s); err != nil {
		return err
	}
	var latches [scriptLatches]byte
	copy(latches[:], s.TriggerLatches)
	relations, err := NewRelations(s.Diplomacy)
	if err != nil {
		return err
	}
	relations = relations.materialised()

	// The only mutation point: no validation or allocation follows it.
	w.registers = s.Registers
	w.presetRegisters()
	w.latches = latches
	w.relations = relations
	w.won, w.lost, w.outcome = s.Won, s.Lost, s.Outcome
	w.rawSessionHead, w.rawSessionMid = s.RawHead, s.RawMid
	if s.HasClock {
		w.hasSessionClock = true
		w.tick, w.fullTick = uint64(s.Clock.SubTick), s.Clock.FullTick
	}
	return nil
}
