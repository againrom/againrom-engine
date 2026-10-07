package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Native slot zero and slots above the ordinary reader's 1..16 domain need
// an explicit transport spelling. Money and all other Player values remain
// ordinary fields; changing either slot field invalidates this narrow policy.
type currentPlayerSlot struct {
	Object  uint16
	Wire    uint32
	Native  uint32
	Trigger *uint32 `json:",omitempty"`
	Shared  bool    `json:",omitempty"`
}

func projectCurrentPlayerSlots(doc *sav.DocumentData, a *currentActionData, world *sim.World) error {
	used, seen := map[uint32]bool{}, map[uint16]bool{}
	counts, money, split := map[uint32]int{}, map[uint32]uint32{}, map[uint32]bool{}
	for _, object := range doc.Players {
		if object == 0 || seen[object] {
			continue
		}
		seen[object] = true
		slot, err := savedStructureValue(&doc.Objects[object-1], "Slot")
		if err != nil {
			return err
		}
		used[slot] = true
		value, err := savedStructureValue(&doc.Objects[object-1], "Money")
		if err != nil {
			return err
		}
		if counts[slot] != 0 && money[slot] != value {
			split[slot] = true
		}
		counts[slot]++
		money[slot] = value
	}
	clear(seen)
	for _, object := range doc.Players {
		if object == 0 || seen[object] {
			continue
		}
		seen[object] = true
		r := &doc.Objects[object-1]
		slot, err := savedStructureValue(r, "Slot")
		if err != nil {
			return err
		}
		again, err := savedStructureValue(r, "SlotAgain")
		if err != nil {
			return err
		}
		shared := counts[slot] > 1 && !split[slot]
		if shared {
			// One native purse is shared by exact Player objects, not owned by
			// a chosen root. Divergent ordinary Money cannot be represented by
			// that scalar: keep those separate values as uncovered import debt.
			mustSetValue(r, "Money", world.Purse(slot))
		}
		if slot >= 1 && slot <= 16 && again == slot && !shared {
			continue
		}
		wire := slot
		if slot < 1 || slot > 16 {
			wire = 1
			for wire < 16 && used[wire] {
				wire++
			}
		}
		// With all sixteen slots occupied, a separate exact Player root shares
		// slot16. Native LOAD still distinguishes its identity. The effect of
		// that transport collision in the original runtime remains engine debt.
		used[wire] = true
		policy := currentPlayerSlot{Object: object, Wire: wire, Native: slot, Shared: shared}
		if again != slot {
			policy.Trigger = &again
		}
		a.PlayerSlots = append(a.PlayerSlots, policy)
		mustSetValue(r, "Slot", wire)
		mustSetValue(r, "SlotAgain", wire)
	}
	return nil
}

func restoreCurrentPlayerSlots(ms *Mission, rows []currentPlayerSlot) error {
	if len(rows) == 0 {
		return nil
	}
	state := ms.savedDocument
	if state == nil || state.Document == nil || state.GroupBindings == nil || len(rows) > len(state.GroupBindings.Players) {
		return fmt.Errorf("current Player slot policy lacks exact roots")
	}
	doc := state.Document
	players := map[uint16]uint32{}
	for _, p := range state.GroupBindings.Players {
		players[p.ObjectIndex] = p.ID
	}
	seen, shared := map[uint16]bool{}, map[uint32]int{}
	for _, p := range rows {
		if p.Shared {
			shared[p.Native]++
		}
	}
	active := map[uint16]currentPlayerSlot{}
	var changes []sim.CurrentPlayerSlot
	for _, p := range rows {
		var domain sim.World
		if players[p.Object] == 0 || seen[p.Object] || p.Wire < 1 || p.Wire > 16 || !domain.SetPurse(p.Native, 0) ||
			p.Trigger != nil && *p.Trigger == p.Native || p.Shared && shared[p.Native] < 2 ||
			p.Native >= 1 && p.Native <= 16 && (p.Native != p.Wire || p.Trigger == nil && !p.Shared) {
			return fmt.Errorf("invalid current Player slot policy")
		}
		seen[p.Object] = true
		r := &doc.Objects[p.Object-1]
		slot, err := savedStructureValue(r, "Slot")
		if err != nil {
			return err
		}
		again, err := savedStructureValue(r, "SlotAgain")
		if err != nil {
			return err
		}
		if slot != p.Wire || again != p.Wire {
			continue
		}
		key, err := savedStructureValue(r, "This")
		if err != nil {
			return err
		}
		active[p.Object] = p
		changes = append(changes, sim.CurrentPlayerSlot{Player: players[p.Object], Key: key, Wire: p.Wire, Native: p.Native, Trigger: p.Trigger, Shared: p.Shared})
	}
	amounts, split := map[uint32]uint32{}, map[uint32]bool{}
	for _, p := range state.GroupBindings.Players {
		r := &doc.Objects[p.ObjectIndex-1]
		slot, err := savedStructureValue(r, "Slot")
		if err != nil {
			return err
		}
		if mapped, ok := active[p.ObjectIndex]; ok {
			slot = mapped.Native
		}
		money, err := savedStructureValue(r, "Money")
		if err != nil {
			return err
		}
		if earlier, exists := amounts[slot]; exists && earlier != money {
			split[slot] = true
		}
		amounts[slot] = money
	}
	for slot := range split {
		delete(amounts, slot)
	}
	if err := ms.World.RestoreCurrentPlayerSlots(changes, amounts); err != nil {
		return err
	}
	for object, p := range active {
		mustSetValue(&doc.Objects[object-1], "Slot", p.Native)
		again := p.Native
		if p.Trigger != nil {
			again = *p.Trigger
		}
		mustSetValue(&doc.Objects[object-1], "SlotAgain", again)
	}
	for i := range state.GroupBindings.Groups {
		g := &state.GroupBindings.Groups[i]
		for _, ref := range []*SnapshotSAVGroupReferenceBinding{&g.Owner, &g.Reference} {
			if p, ok := active[ref.ObjectIndex]; ok && ref.Class == 1 {
				ref.Owner = p.Native
			}
		}
	}
	if state.PlayerPurses != nil {
		purses, err := savedPlayerPurseRows(doc, state.GroupBindings)
		if err != nil {
			return err
		}
		state.PlayerPurses = &SnapshotSAVPlayerPurses{Version: 1, Players: purses}
	}
	return nil
}
