package mapload

import (
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// PlayerPolicy builds a fresh world's Players record from its map.
type PlayerPolicy func(w *sim.World, m *alm.Map) error

func (t *Table) freshPlayers() PlayerPolicy {
	if t == nil || t.FreshPlayers == nil {
		return SlotPlayers
	}
	return t.FreshPlayers
}

func initializeCurrentPlayers(w *sim.World, m *alm.Map, t *Table) error {
	return t.freshPlayers()(w, m)
}

// NoFreshPlayers builds no Players record.
func NoFreshPlayers(*sim.World, *alm.Map) error { return nil }

// SlotPlayers is the engine's construction policy: one Player per initial
// Slot, IDs following the sorted Slots, not archive identities. This does not
// select a Player.
func SlotPlayers(w *sim.World, m *alm.Map) error {
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
