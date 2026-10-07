package sim

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"slices"
)

// Form80 appends exact native Player-container provenance and the Group ID
// highwater after the complete form79 payload. A zero span retains absent
// provenance. Nothing is inferred from Group.Owner or retained SAV documents.
const savedGroupPlayerSpanLen = 4

type savedGroupPlayerSection struct {
	present    bool
	highWater  uint32
	players    []SavedGroupPlayer
	containers []SavedGroupContainer
}

func (w *World) appendSavedGroupPlayerSection(b []byte) []byte {
	start := len(b)
	s := w.savedGroups
	if s != nil && (s.PlayersPresent || s.HighWater > maxSavedGroupID(s.Groups)) {
		flag := byte(0)
		if s.PlayersPresent {
			flag = 1
		}
		b = append(b, flag)
		b = binary.LittleEndian.AppendUint32(b, s.HighWater)
		b = binary.LittleEndian.AppendUint32(b, uint32(len(s.Players)))
		for _, p := range s.Players {
			b = binary.LittleEndian.AppendUint32(b, p.ID)
			b = binary.LittleEndian.AppendUint32(b, p.Slot)
		}
		var containers []SavedGroupContainer
		if s.PlayersPresent {
			containers = make([]SavedGroupContainer, len(s.Groups))
			for i, g := range s.Groups {
				containers[i] = SavedGroupContainer{g.ID, g.ContainerID}
			}
			slices.SortFunc(containers, func(a, b SavedGroupContainer) int { return cmp.Compare(a.GroupID, b.GroupID) })
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(len(containers)))
		for _, c := range containers {
			b = binary.LittleEndian.AppendUint32(b, c.GroupID)
			b = binary.LittleEndian.AppendUint32(b, c.PlayerID)
		}
	}
	return binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
}

func splitSavedGroupPlayerSection(data []byte) ([]byte, *savedGroupPlayerSection, error) {
	if len(data) < headerLen+savedGroupPlayerSpanLen {
		return nil, nil, fmt.Errorf("sim: truncated saved Player container footer")
	}
	end := len(data) - savedGroupPlayerSpanLen
	n := uint64(binary.LittleEndian.Uint32(data[end:]))
	if n > uint64(end-headerLen) {
		return nil, nil, fmt.Errorf("sim: saved Player container span exceeds payload")
	}
	if n == 0 {
		return data[:end], nil, nil
	}
	start := end - int(n)
	payload := data[start:end]
	if len(payload) < 13 || payload[0] > 1 {
		return nil, nil, fmt.Errorf("sim: invalid saved Player container presence or header")
	}
	s := &savedGroupPlayerSection{present: payload[0] == 1, highWater: binary.LittleEndian.Uint32(payload[1:])}
	count := binary.LittleEndian.Uint32(payload[5:])
	payload = payload[9:]
	if count > savedGroupPlayerLimit || uint64(count)*8+4 > uint64(len(payload)) {
		return nil, nil, fmt.Errorf("sim: saved Player count exceeds payload or limit")
	}
	if count != 0 {
		s.players = make([]SavedGroupPlayer, count)
	}
	for i := range s.players {
		s.players[i] = SavedGroupPlayer{binary.LittleEndian.Uint32(payload), binary.LittleEndian.Uint32(payload[4:])}
		payload = payload[8:]
	}
	count = binary.LittleEndian.Uint32(payload)
	payload = payload[4:]
	if count > savedGroupPlayerLimit || uint64(count)*8 != uint64(len(payload)) {
		return nil, nil, fmt.Errorf("sim: saved Group container count does not match payload or limit")
	}
	if count != 0 {
		s.containers = make([]SavedGroupContainer, count)
	}
	for i := range s.containers {
		c := SavedGroupContainer{binary.LittleEndian.Uint32(payload), binary.LittleEndian.Uint32(payload[4:])}
		if c.GroupID == 0 || i > 0 && s.containers[i-1].GroupID >= c.GroupID {
			return nil, nil, fmt.Errorf("sim: saved Group container mappings are not strictly ordered")
		}
		s.containers[i] = c
		payload = payload[8:]
	}
	if !s.present && (len(s.players) != 0 || len(s.containers) != 0) {
		return nil, nil, fmt.Errorf("sim: absent saved Player registry carries identities")
	}
	return data[:start], s, nil
}

func applySavedGroupPlayerSection(groups *savedGroupState, section *savedGroupPlayerSection) error {
	if section == nil {
		return nil
	}
	if groups == nil {
		return fmt.Errorf("sim: saved Player containers lack their Group registry")
	}
	if !section.present && section.highWater <= maxSavedGroupID(groups.Groups) {
		return fmt.Errorf("sim: redundant absent Player container section")
	}
	groups.Players, groups.PlayersPresent, groups.HighWater = section.players, section.present, section.highWater
	if section.present {
		if len(section.containers) != len(groups.Groups) {
			return fmt.Errorf("sim: saved Player containers do not cover every Group")
		}
		byID := make(map[uint32]uint32, len(section.containers))
		for _, c := range section.containers {
			byID[c.GroupID] = c.PlayerID
		}
		for i := range groups.Groups {
			id, found := byID[groups.Groups[i].ID]
			if !found {
				return fmt.Errorf("sim: saved Player container mapping names another Group")
			}
			groups.Groups[i].ContainerID = id
		}
	}
	return savedGroupPlayersFault(groups)
}
