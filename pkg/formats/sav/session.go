package sav

import "fmt"

// SessionState is the supported continuity subset of the original mission
// session block. TRIG-SAVE-008 (corrected) states the world-half session
// block is restored before the map trigger programme is rebuilt, and
// MISSION-SLOT-008 states the rebuild presets only the 0x10002 constant arm;
// a check-owned slot is not overwritten there, and is instead recomputed on
// that check's own next pass — so Registers rides beside the rest here, and
// a consumer that restores it before that rebuild reproduces the original
// for every slot class (docs/1130/story.md). RawHead and RawMid have no
// promoted meaning (SAV-SESS-031) and are carried opaquely, never
// interpreted.
//
// Every slice and array is detached from File.Body. A caller may validate,
// retain or mutate them without changing the decoded save.
type SessionState struct {
	// The world-head pair precedes the session block. Preserve both words;
	// signed phase interpretation belongs to the simulation, not this reader.
	SubTick, FullTick uint32
	Registers         [triggerResultCount]int32
	TriggerLatches    []byte
	RawHead           [rawHeadLen]byte
	RawMid            [rawMidLen]byte
	Diplomacy         []byte
	Won, Lost         uint32
}

// SessionState returns the hundred trigger result registers, the thousand
// trigger latches, the two opaque raw regions and the complete 50x50
// diplomacy matrix from a mid-mission save.
//
// The values are read through TriggerResult, TriggerLatch, RawHead, RawMid
// and Diplomacy rather than by taking a second raw view over the session
// block. Those accessors are the package's already witnessed format
// boundary; this method only groups their complete populations into one
// transactional handoff.
func (f *File) SessionState() (SessionState, error) {
	if f == nil || f.World == nil {
		return SessionState{}, fmt.Errorf("sav: this save has no world session")
	}
	if off := f.World.SessionOff; off < 0 || off > len(f.Body) || len(f.Body)-off < sessionLen {
		return SessionState{}, fmt.Errorf("sav: truncated world session")
	}
	out := SessionState{
		SubTick:        f.Head.CounterA,
		FullTick:       f.Head.CounterB,
		TriggerLatches: make([]byte, triggerLatchCount),
		Diplomacy:      make([]byte, diplomacySide*diplomacySide),
	}
	var err error
	if out.Won, out.Lost, err = f.Counters(); err != nil {
		return SessionState{}, err
	}
	for i := range out.Registers {
		v, err := f.TriggerResult(i)
		if err != nil {
			return SessionState{}, err
		}
		out.Registers[i] = v
	}
	if out.RawHead, err = f.RawHead(); err != nil {
		return SessionState{}, err
	}
	if out.RawMid, err = f.RawMid(); err != nil {
		return SessionState{}, err
	}
	for i := range out.TriggerLatches {
		v, err := f.TriggerLatch(i)
		if err != nil {
			return SessionState{}, err
		}
		out.TriggerLatches[i] = v
	}
	for from := 0; from < diplomacySide; from++ {
		for to := 0; to < diplomacySide; to++ {
			v, err := f.Diplomacy(from, to)
			if err != nil {
				return SessionState{}, err
			}
			out.Diplomacy[from*diplomacySide+to] = v
		}
	}
	return out, nil
}
