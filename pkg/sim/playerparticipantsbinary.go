package sim

import (
	"bytes"
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
	if identities && (!groups || !slices.Equal(players, groupPlayers)) {
		return w.appendCurrentPlayers(b, players, rows, present)
	}
	if !present {
		return b
	}
	base, start := b[0], len(b)
	b[0] = playerParticipantFormVersion
	b = binary.LittleEndian.AppendUint32(b, uint32(len(rows)))
	for _, row := range rows {
		b = binary.LittleEndian.AppendUint32(b, row.PlayerID)
		b = binary.LittleEndian.AppendUint32(b, row.Value)
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'P', 'P', 'T', '1')
}

func (w *World) appendCurrentPlayers(b []byte, players []SavedGroupPlayer, rows []PlayerParticipant, present bool) []byte {
	base, start := b[0], len(b)
	b[0] = currentPlayerFormVersion
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
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'C', 'P', 'P', '1')
}

func (w *World) unmarshalCurrentPlayers(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed independent current Player carrier") }
	if len(data) < headerLen+14 || !bytes.Equal(data[len(data)-4:], []byte("CPP1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if baseVersion >= currentPlayerFormVersion || span < 5 || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
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
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
		return err
	}
	if err := next.RestoreCurrentPlayers(players, rows, present); err != nil {
		return fail()
	}
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}

func (w *World) unmarshalPlayerParticipants(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed Player Participant carrier") }
	if len(data) < headerLen+13 || !bytes.Equal(data[len(data)-4:], []byte("PPT1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if baseVersion >= playerParticipantFormVersion || span < 4 || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count > savedGroupPlayerLimit || span != 4+8*count {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
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
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
