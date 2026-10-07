package game

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"againrom/pkg/formats/sav"
)

type currentAbsentPlayer struct {
	Object uint16
	Anchor [32]byte
}

func currentPlayerPresenceAnchor(record sav.DocumentRecordData) ([32]byte, error) {
	raw, err := json.Marshal(record)
	if err != nil {
		return [32]byte{}, err
	}
	// JSON replaces invalid UTF-8. Include each original byte string as a
	// length-delimited suffix so distinct installed names remain distinct.
	var text func(sav.DocumentRecordData)
	text = func(r sav.DocumentRecordData) {
		for _, value := range r.Texts {
			raw = binary.LittleEndian.AppendUint32(raw, uint32(len(value.Value)))
			raw = append(raw, value.Value...)
		}
		for _, child := range r.Inline {
			text(child.Record)
		}
		for _, child := range r.Groups {
			text(child)
		}
	}
	text(record)
	return sha256.Sum256(raw), nil
}

func captureCurrentAbsentPlayers(doc *sav.DocumentData, state *SnapshotSAVDocument, a *currentActionData) error {
	if state.GroupBindings == nil {
		return nil
	}
	for _, p := range state.GroupBindings.Players {
		if !p.Constructed {
			continue
		}
		if p.ObjectIndex == 0 || int(p.ObjectIndex) > len(doc.Objects) {
			return fmt.Errorf("constructed Player has no ordinary binding")
		}
		anchor, err := currentPlayerPresenceAnchor(doc.Objects[p.ObjectIndex-1])
		if err != nil {
			return err
		}
		a.AbsentPlayers = append(a.AbsentPlayers, currentAbsentPlayer{Object: p.ObjectIndex, Anchor: anchor})
	}
	return validateCurrentAbsentPlayers(doc, a)
}

func validateCurrentAbsentPlayers(doc *sav.DocumentData, a *currentActionData) error {
	if len(a.AbsentPlayers) == 0 {
		return nil
	}
	if doc == nil || len(a.AbsentPlayers) > 65535 || a.GroupPlayers == nil || !*a.GroupPlayers {
		return fmt.Errorf("current Player absence lacks its native carrier")
	}
	roots, seen, native := map[uint16]bool{}, map[uint16]bool{}, map[uint16]bool{}
	for _, object := range doc.Players {
		roots[object] = object != 0
	}
	for _, group := range a.Groups {
		if !group.RootOnly {
			native[group.Object] = true
		}
	}
	for _, row := range a.AbsentPlayers {
		if !roots[row.Object] || int(row.Object) > len(doc.Objects) || doc.Objects[row.Object-1].Class != "Player" || seen[row.Object] || row.Anchor == ([32]byte{}) {
			return fmt.Errorf("current Player absence has an invalid or repeated exact root")
		}
		if native[row.Object] {
			return fmt.Errorf("current Player absence conflicts with a native Group container")
		}
		seen[row.Object] = true
	}
	return nil
}

func matchCurrentAbsentPlayers(doc *sav.DocumentData, a *currentActionData) (map[uint16]bool, error) {
	if err := validateCurrentAbsentPlayers(doc, a); err != nil {
		return nil, err
	}
	matching := map[uint16]bool{}
	for _, row := range a.AbsentPlayers {
		anchor, err := currentPlayerPresenceAnchor(doc.Objects[row.Object-1])
		if err != nil {
			return nil, err
		}
		matching[row.Object] = anchor == row.Anchor
	}
	return matching, nil
}

// A changed Player, or a now-real Group using it, promotes this ordinary root
// to a native container. Absence never replaces an ordinary scalar or edge.
func absentCurrentPlayerIDs(bindings *SnapshotSAVGroupBindings, groups []SnapshotSAVGroupBinding, matching map[uint16]bool) []uint32 {
	var absent []uint32
	used := map[uint16]bool{}
	for _, g := range groups {
		if !g.RootOnly {
			used[g.PlayerObject], used[g.Owner.ObjectIndex], used[g.Reference.ObjectIndex] = true, true, true
		}
	}
	for i := range bindings.Players {
		p := &bindings.Players[i]
		p.Constructed = matching[p.ObjectIndex] && !used[p.ObjectIndex]
		if p.Constructed {
			absent = append(absent, p.ID)
		}
	}
	return absent
}

func finalizeCurrentAbsentPlayers(doc *sav.DocumentData, a *currentActionData) error {
	for i := range a.AbsentPlayers {
		row := &a.AbsentPlayers[i]
		anchor, err := currentPlayerPresenceAnchor(doc.Objects[row.Object-1])
		if err != nil {
			return err
		}
		row.Anchor = anchor
	}
	return validateCurrentAbsentPlayers(doc, a)
}
