package game

import (
	"errors"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// SnapshotFame is campaign-owned score history. Known distinguishes a complete
// history from an older AGS that never stored the counters. Unknown histories
// can retain later observations but cannot produce an earned result.
type SnapshotFame struct {
	Known        bool
	Time, Events uint32
	Result       *FameResult
}

// FameResult is captured once on accepted terminal mission completion. Only
// Recorded may later change, after the local hall store accepts the result.
type FameResult struct {
	Name     string
	Score    int32
	Recorded bool
}

func cloneFame(s SnapshotFame) SnapshotFame {
	if s.Result != nil {
		result := *s.Result
		s.Result = &result
	}
	return s
}

func validateSnapshotFame(s *SnapshotFame) error {
	if s != nil && !s.Known && s.Result != nil {
		return errors.New("saved fame result has unknown campaign history")
	}
	return nil
}

func fameFromSnapshot(s Snapshot) SnapshotFame {
	if s.Fame != nil {
		return cloneFame(*s.Fame)
	}
	if s.CampaignState {
		return SnapshotFame{Known: s.Campaign.ScoreEventsKnown, Time: s.Campaign.MissionTime, Events: s.Campaign.ScoreEvents}
	}
	return SnapshotFame{}
}

// earnedFameScore chooses separate binary64 operations, each rounded to 53
// bits, then signed64 truncation and the low 32 bits. This deterministic policy
// does not claim the original runtime's unobserved full FPU control state.
func earnedFameScore(time, events uint32, xp int32) int32 {
	var value float64
	if time != 0 {
		denominator := float64(float64(int32(time)) * 10.0)
		value = float64(float64(xp) / denominator)
	} else {
		value = float64(float64(xp) * float64(2e-6))
	}
	value = float64(value * float64(int32(events)))
	// With three signed32 inputs both branches stay inside signed64 range.
	return int32(int64(value))
}

// completeFame runs only after Town accepted this mission's first completion.
// The simulation's low dword is the original signed subtick counter; division
// truncates toward zero before the wrapping campaign addition.
func (s *CampaignSession) completeFame(campaign Campaign, n int, party []mapload.PartyMember, w *sim.World,
	ids []sim.EntityID, roster map[sim.EntityID]mapload.PartyMember) {
	if w == nil {
		return
	}
	s.fame.Time += uint32(int32(uint32(w.Tick())) / 16)
	if !s.fame.Known || s.fame.Result != nil || !campaign.TerminalMission(n) {
		return
	}
	// Party order and the selected actor are not the primary hero's identity.
	// The roster owns runtime-added identities; the original party/ID pairing
	// remains the fallback for callers without that richer mapping.
	var hero mapload.PartyMember
	var heroID sim.EntityID
	found := false
	for i, member := range party {
		if member.StartingHero && member.PlayerCharacter && !member.Hired() && i < len(ids) {
			hero, heroID, found = member, ids[i], true
			break
		}
	}
	entities := w.Entities()
	for _, e := range entities {
		if member, ok := roster[e.ID]; ok && member.StartingHero && member.PlayerCharacter && !member.Hired() {
			hero, heroID, found = member, e.ID, true
			break
		}
	}
	if !found {
		return
	}
	for _, e := range entities {
		if e.ID != heroID {
			continue
		}
		var xp int32
		for _, skillXP := range e.SkillXP {
			xp += skillXP
		}
		s.fame.Result = &FameResult{Name: hero.Name, Score: earnedFameScore(s.fame.Time, s.fame.Events, xp)}
		return
	}
}

type fameObserver struct {
	state             *SnapshotFame
	previous, scratch []sim.CorpseState
}

func (s *CampaignSession) detachFameObserver() {
	if s.live != nil {
		s.live.fame.state = nil
	}
}

func (s *CampaignSession) bindFameObserver(mw *mapWorld) {
	s.detachFameObserver()
	if mw == nil || mw.world == nil {
		return
	}
	mw.fame.state = &s.fame
	// A cold bind is a baseline, including already dead actors. It reports no
	// received transition and therefore earns no event merely by loading, so a
	// load and an unchanged save keep the counter the source carried.
	mw.fame.previous = mw.world.AppendCorpseStates(mw.fame.previous[:0])
}

func (mw *mapWorld) observeFame() {
	o := &mw.fame
	if o.state == nil {
		return
	}
	next := mw.world.AppendCorpseStates(o.scratch[:0])
	o.observe(next, mw.world.Relations())
}

func (o *fameObserver) observe(next []sim.CorpseState, relations sim.Relations) {
	oldIndex := 0
	for _, current := range next {
		for oldIndex < len(o.previous) && o.previous[oldIndex].ID < current.ID {
			oldIndex++
		}
		old := sim.DecayNone
		if oldIndex < len(o.previous) && o.previous[oldIndex].ID == current.ID {
			old = o.previous[oldIndex].Stage
		}
		if old < sim.DecayBones && current.Stage >= sim.DecayBones && current.Stage <= 4 &&
			relations.Hostile(sim.SelfSlot, current.Owner) {
			o.state.Events++
		}
	}
	// Missing IDs disappear from the baseline without an event. Resurrection
	// replaces a prior dead stage with zero, admitting a later decay transition.
	o.previous, o.scratch = next, o.previous[:0]
}
