package sim

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"slices"
)

func (w *World) appendSavedFormations(b []byte) []byte {
	start := len(b)
	if w.hasSavedFormations() {
		s := w.savedGroups
		b = binary.LittleEndian.AppendUint32(b, uint32(len(s.Formations)))
		for _, p := range s.Formations {
			b = binary.LittleEndian.AppendUint32(b, p.PlayerID)
			b = binary.LittleEndian.AppendUint16(b, uint16(p.CommandID))
			b = binary.LittleEndian.AppendUint32(b, p.TriggerID)
			b = append(b, p.Mode)
		}
		groups := slices.Clone(s.Groups)
		slices.SortFunc(groups, func(a, b SavedGroup) int { return cmp.Compare(a.ID, b.ID) })
		b = binary.LittleEndian.AppendUint32(b, uint32(len(groups)))
		for _, g := range groups {
			b = binary.LittleEndian.AppendUint32(b, g.ID)
			b = binary.LittleEndian.AppendUint32(b, g.OwnerID)
		}
	}
	return binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
}

func splitSavedFormations(data []byte) ([]byte, []SavedPlayerFormation, []SavedGroupOwner, bool, error) {
	fail := func() ([]byte, []SavedPlayerFormation, []SavedGroupOwner, bool, error) {
		return nil, nil, nil, false, fmt.Errorf("sim: invalid Player formation footer")
	}
	if len(data) < headerLen+4 {
		return fail()
	}
	end := len(data) - 4
	span := uint64(binary.LittleEndian.Uint32(data[end:]))
	if span > uint64(end-headerLen) {
		return fail()
	}
	if span == 0 {
		return data[:end], nil, nil, false, nil
	}
	if span < 8 {
		return fail()
	}
	start := end - int(span)
	payload := data[start:end]
	n := binary.LittleEndian.Uint32(payload)
	payload = payload[4:]
	if n > savedGroupPlayerLimit || uint64(n)*11+4 > uint64(len(payload)) {
		return fail()
	}
	var players []SavedPlayerFormation
	for range n {
		players = append(players, SavedPlayerFormation{
			PlayerID:  binary.LittleEndian.Uint32(payload),
			CommandID: int16(binary.LittleEndian.Uint16(payload[4:])),
			TriggerID: binary.LittleEndian.Uint32(payload[6:]), Mode: payload[10],
		})
		payload = payload[11:]
	}
	n = binary.LittleEndian.Uint32(payload)
	payload = payload[4:]
	if n > savedGroupPlayerLimit || uint64(n)*8 != uint64(len(payload)) {
		return fail()
	}
	var owners []SavedGroupOwner
	for range n {
		owners = append(owners, SavedGroupOwner{binary.LittleEndian.Uint32(payload), binary.LittleEndian.Uint32(payload[4:])})
		payload = payload[8:]
	}
	return data[:start], players, owners, true, nil
}
