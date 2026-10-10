package mapload

import (
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// Fresh ROM1 Players use the existing engine construction policy. IDs follow
// sorted initial Slots, not archive identities. This does not select a Player.
func initializeCurrentPlayers(w *sim.World, m *alm.Map, t *Table) error {
	if t != nil && !t.Game.Edition().FreshPlayers {
		return nil
	}
	var slots []uint32
	if m != nil {
		for i := range m.Groups {
			slots = append(slots, uint32(i+1))
		}
	}
	for _, e := range w.Entities() {
		if !slices.Contains(slots, e.Owner) {
			slots = append(slots, e.Owner)
		}
	}
	slices.Sort(slots)
	players := make([]sim.SavedGroupPlayer, len(slots))
	rows := make([]sim.PlayerParticipant, len(slots))
	for i, slot := range slots {
		id, participant := uint32(i+1), uint32(1)
		if m != nil && slot > 0 && uint64(slot) <= uint64(len(m.Groups)) {
			participant = m.Groups[slot-1].Participant
		}
		if slot == sim.SelfSlot {
			participant = 0
		}
		players[i] = sim.SavedGroupPlayer{ID: id, Slot: slot}
		rows[i] = sim.PlayerParticipant{PlayerID: id, Value: participant}
	}
	return w.RestoreCurrentPlayers(players, rows, true)
}
