package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
)

const playerParticipantFormVersion byte = 110
const currentPlayerFormVersion byte = 111

func (w *World) appendPlayerParticipants(b []byte) []byte {
	players, identities := w.CurrentPlayers()
	rows, present := w.PlayerParticipants()
	groupPlayers, groups := w.SavedGroupPlayers()
	// A registry the byte form omits carries no Players; the current Players
	// travel on their own and the decoder builds the registry from them.
	groups = groups && w.groupsForm() != nil
	if identities && (!groups || !slices.Equal(players, groupPlayers)) {
		return w.appendCurrentPlayers(b, players, rows, present)
	}
	if !present {
		return b
	}
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(rows)))
	for _, row := range rows {
		b = binary.LittleEndian.AppendUint32(b, row.PlayerID)
		b = binary.LittleEndian.AppendUint32(b, row.Value)
	}
	return endWrapper(b, ws, playerParticipantFormVersion)
}

func (w *World) appendCurrentPlayers(b []byte, players []SavedGroupPlayer, rows []PlayerParticipant, present bool) []byte {
	ws := beginWrapper(b)
	var presence byte
	if present {
		presence = 1
	}
	b = append(b, presence)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(players)))
	for i, player := range players {
		b = binary.LittleEndian.AppendUint32(b, player.ID)
		b = binary.LittleEndian.AppendUint32(b, player.Slot)
		var value uint32
		if present {
			value = rows[i].Value
		}
		b = binary.LittleEndian.AppendUint32(b, value)
	}
	return endWrapper(b, ws, currentPlayerFormVersion)
}

func (w *World) unmarshalCurrentPlayers(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed independent current Player carrier") }
	env, ok := readWrapper(data, currentPlayerFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 5 {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start+1:]))
	if data[start] > 1 || count > savedGroupPlayerLimit || span != 5+12*count {
		return fail()
	}
	present := data[start] == 1
	players := make([]SavedGroupPlayer, int(count))
	var rows []PlayerParticipant
	if present {
		rows = make([]PlayerParticipant, int(count))
	}
	for i := range players {
		at := start + 5 + 12*i
		players[i] = SavedGroupPlayer{binary.LittleEndian.Uint32(data[at:]), binary.LittleEndian.Uint32(data[at+4:])}
		value := binary.LittleEndian.Uint32(data[at+8:])
		if present {
			rows[i] = PlayerParticipant{players[i].ID, value}
		} else if value != 0 {
			return fail()
		}
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	if err := next.RestoreCurrentPlayers(players, rows, present); err != nil {
		return fail()
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}

func (w *World) unmarshalPlayerParticipants(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed Player Participant carrier") }
	env, ok := readWrapper(data, playerParticipantFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4 {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count > savedGroupPlayerLimit || span != 4+8*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	rows := make([]PlayerParticipant, int(count))
	for i := range rows {
		at := start + 4 + 8*i
		rows[i] = PlayerParticipant{binary.LittleEndian.Uint32(data[at:]), binary.LittleEndian.Uint32(data[at+4:])}
	}
	if err := next.RestorePlayerParticipants(rows, true); err != nil {
		return fail()
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
