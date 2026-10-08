package sim

import (
	"fmt"
	"slices"
)

type PlayerParticipant struct {
	PlayerID, Value uint32
}

type currentPlayerState struct {
	Players             []SavedGroupPlayer
	Participants        []PlayerParticipant
	ParticipantsPresent bool
}

func cloneCurrentPlayers(s *currentPlayerState) *currentPlayerState {
	if s == nil {
		return nil
	}
	n := *s
	n.Players, n.Participants = slices.Clone(s.Players), slices.Clone(s.Participants)
	return &n
}

// CurrentPlayers is independent of command/container provenance and selection.
func (w *World) CurrentPlayers() ([]SavedGroupPlayer, bool) {
	if w.currentPlayers != nil {
		return slices.Clone(w.currentPlayers.Players), true
	}
	return w.SavedGroupPlayers()
}

func (w *World) RestoreCurrentPlayers(players []SavedGroupPlayer, rows []PlayerParticipant, present bool) error {
	n := &currentPlayerState{Players: slices.Clone(players), Participants: slices.Clone(rows), ParticipantsPresent: present}
	if err := currentPlayersFault(n); err != nil {
		return err
	}
	if groups, available := w.SavedGroupPlayers(); available && !slices.Equal(groups, players) {
		return fmt.Errorf("sim: current Player identities differ from command containers")
	}
	w.currentPlayers = n
	return nil
}

// RestoreCurrentPlayerRegistryAbsent preserves older continuation absence.
// Independent identities and values disappear; command containers remain.
func (w *World) RestoreCurrentPlayerRegistryAbsent() {
	w.currentPlayers = nil
}

func (w *World) currentPlayersFault() error {
	if err := currentPlayersFault(w.currentPlayers); err != nil {
		return err
	}
	if w.currentPlayers != nil {
		if groups, present := w.SavedGroupPlayers(); present && !slices.Equal(groups, w.currentPlayers.Players) {
			return fmt.Errorf("sim: current Player identities differ from command containers")
		}
	}
	return nil
}

func (w *World) PlayerParticipants() ([]PlayerParticipant, bool) {
	if w.currentPlayers == nil || !w.currentPlayers.ParticipantsPresent {
		return nil, false
	}
	return slices.Clone(w.currentPlayers.Participants), true
}

// RestorePlayerParticipants replaces the complete exact-ID carrier atomically.
// Absence is independent of a present zero word and never selects a Player.
func (w *World) RestorePlayerParticipants(rows []PlayerParticipant, present bool) error {
	players, available := w.CurrentPlayers()
	if !available {
		return fmt.Errorf("sim: Participant requires exact current Player identities")
	}
	n := &currentPlayerState{Players: players}
	n.Participants, n.ParticipantsPresent = slices.Clone(rows), present
	if err := currentPlayersFault(n); err != nil {
		return err
	}
	w.currentPlayers = n
	return nil
}

func (w *World) SetPlayerParticipant(id, value uint32) error {
	if w.currentPlayers != nil && w.currentPlayers.ParticipantsPresent {
		for i, row := range w.currentPlayers.Participants {
			if row.PlayerID == id {
				n := cloneCurrentPlayers(w.currentPlayers)
				n.Participants[i].Value = value
				w.currentPlayers = n
				return nil
			}
		}
	}
	return fmt.Errorf("sim: Participant lacks exact current Player %d", id)
}

func currentPlayersFault(s *currentPlayerState) error {
	if s == nil {
		return nil
	}
	if len(s.Players) > savedGroupPlayerLimit {
		return fmt.Errorf("sim: current Player population exceeds bound")
	}
	for i, player := range s.Players {
		if player.ID == 0 || i > 0 && s.Players[i-1].ID >= player.ID {
			return fmt.Errorf("sim: current Player identities are not strictly ordered")
		}
	}
	if !s.ParticipantsPresent {
		if len(s.Participants) != 0 {
			return fmt.Errorf("sim: absent Participant carrier has values")
		}
		return nil
	}
	if len(s.Participants) != len(s.Players) {
		return fmt.Errorf("sim: Participant carrier does not cover current Players")
	}
	for i, row := range s.Participants {
		if row.PlayerID == 0 || row.PlayerID != s.Players[i].ID {
			return fmt.Errorf("sim: Participant lacks exact ordered Player identity")
		}
	}
	return nil
}
