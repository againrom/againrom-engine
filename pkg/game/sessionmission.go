package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
)

// missionIdentity is the session's one answer to which mission it plays as
// mission n. played is n. record is the session's guest record, nil for
// none; guest is that n is that record's link under way, a mod's own mission
// and no mission of the waiting campaign. own is the mod mission an active
// mod numbers n, replaced the mod mission whose map replaces n's, each nil
// for none. back is the mission whose return the world map shows after n:
// for a link of the guest's chain, the win that started the chain.
type missionIdentity struct {
	played   int
	record   *guestMission
	guest    bool
	own      *mod.Mission
	replaced *mod.Mission
	back     int
}

// newMissionIdentity is the one builder of the identity of mission n over
// guest record g and the active mods' missions.
func newMissionIdentity(n int, g *guestMission, missions mod.MissionData) missionIdentity {
	id := missionIdentity{played: n, record: g, guest: g.playing(n), back: g.returnMission(n)}
	if m, ok := missions.ByNumber(n); ok {
		id.own = &m
	}
	if m, ok := missions.Replacing(n); ok {
		id.replaced = &m
	}
	return id
}

// tableMissions are the missions the active mods of table add.
func tableMissions(table *mapload.Table) mod.MissionData {
	if table == nil {
		return mod.MissionData{}
	}
	return table.Mods.Missions
}

// missionIdentity is the identity of mission n in this town's session.
func (t *Town) missionIdentity(n int, missions mod.MissionData) missionIdentity {
	return newMissionIdentity(n, t.guestMission(), missions)
}

// missionIdentity is the identity of mission n in this front end's session.
func (f *FrontEnd) missionIdentity(n int) missionIdentity {
	return f.Town.missionIdentity(n, tableMissions(f.Table))
}

// modMap reports that the mission plays a mod's map: a mod's own mission or
// one whose map a mod replaces.
func (id missionIdentity) modMap() bool { return id.own != nil || id.replaced != nil }

// entry is the party and purse the guest's mission opens with.
func (id missionIdentity) entry() ([]mapload.PartyMember, uint32, bool) {
	if !id.guest {
		return nil, 0, false
	}
	return id.record.guestEntry(id.played)
}

// restoreMissionIdentity is the identity a save of mission n states: the
// guest record its session carries, checked against the missions the active
// mods declare. A save inside a mod's own mission loads only with its guest
// record; a save inside a mission whose map a mod replaces loads only with a
// mod mark.
func restoreMissionIdentity(saved []byte, n int, s *currentSessionData, mods mapload.ModContext) (missionIdentity, error) {
	guest, err := restoreGuest(s, mods)
	if err != nil {
		return missionIdentity{}, err
	}
	id := newMissionIdentity(n, guest, mods.Missions)
	if id.own != nil && guest == nil {
		return missionIdentity{}, fmt.Errorf("this save is mission %d of mod %q and carries no guest record; it cannot be loaded", n, id.own.Mod)
	}
	if err := id.refuseUnmarked(saved); err != nil {
		return missionIdentity{}, err
	}
	return id, nil
}

// refuseUnmarkedReplaced refuses a save inside mission n, written without a
// mod mark, while an active mod replaces that mission's map: its world was
// built from another map. A marked save passed the mod mark, so the mod that
// replaces the map now replaced it when the save was written.
func refuseUnmarkedReplaced(saved []byte, n int, mods mapload.ModContext) error {
	return newMissionIdentity(n, nil, mods.Missions).refuseUnmarked(saved)
}

func (id missionIdentity) refuseUnmarked(saved []byte) error {
	if id.replaced == nil || carriesModMark(saved) {
		return nil
	}
	return modMarkRefusal("this save plays mission %d without a mod mark, and mod %q replaces that mission's map; start without that mod to load it", id.played, id.replaced.Mod)
}

// carriesModMark reports that saved holds a readable mod mark.
func carriesModMark(saved []byte) bool {
	if !sav.MayCarryNativeMods(saved) {
		return false
	}
	doc, err := sav.DecodeDocumentData(saved)
	if err != nil {
		return false
	}
	_, ok, err := readModMark(doc)
	return ok && err == nil
}

// guestMission returns the town's guest; nil for a nil town.
func (t *Town) guestMission() *guestMission {
	if t == nil {
		return nil
	}
	return t.guest
}

// snapshotGuest is a copy of the town's guest record for a save.
func (t *Town) snapshotGuest() *guestMission { return t.guestMission().clone() }

// takeGuestJoining is the guest the town of chapter grants as companion
// npc, as she ended her mission, when one is waiting; taking her ends the
// guest record.
func (t *Town) takeGuestJoining(chapter, npc int) (mapload.PartyMember, bool) {
	member, ok := t.guestMission().joining(chapter, npc)
	if ok {
		t.guest = nil
	}
	return member, ok
}

// playing reports that mission n is the guest's mission, still under way.
func (g *guestMission) playing(n int) bool {
	return g != nil && g.stash != nil && g.mission.Number == n
}

// returnMission is the mission whose return the world map shows after mission
// n: for a link of the guest's chain, the win that started the chain.
func (g *guestMission) returnMission(n int) int {
	if g != nil && slices.ContainsFunc(g.links(), func(m mod.Mission) bool { return m.Number == n }) {
		return g.links()[0].After
	}
	return n
}

// guestEntry is the party and purse the guest's mission n opens with: the
// guest party alone with the declared purse.
func (g *guestMission) guestEntry(n int) ([]mapload.PartyMember, uint32, bool) {
	if !g.playing(n) {
		return nil, 0, false
	}
	return g.party(), uint32(g.mission.Gold), true
}
