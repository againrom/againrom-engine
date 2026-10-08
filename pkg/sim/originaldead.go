package sim

import (
	"fmt"
	"slices"
)

// DeadActorState keeps the independent scalars from SAV-DEADLOAD-126.
// A terminal state has Stage 5 and no entity in the gameplay population.
type DeadActorState struct {
	RuntimeID           uint32
	Cell                uint16
	FineX, FineY, Stage uint8
	HP                  int16
	Timer               int8
}

// OriginalDeadSource retains archive bindings. The death clock publishes its
// current HP and Stage here when it mutates them; loading alone does not.
type OriginalDeadSource struct {
	Identity, TerrainKey, OwnerKey uint32
	References                     [5]uint32
	ArchiveIndex, MapUnitID        uint16
	Class                          uint8 // SourceBinding class, including generated provenance
	ContainerPresent               bool
	ContainerTail                  [2]uint32
	State                          DeadActorState
	HeldWeapon                     OriginalDeadWeapon
}

type OriginalDeadActor struct {
	ID     EntityID
	Source OriginalDeadSource
}

// OriginalDeadRecord joins the retained binding to the current actor.
// Current is projected from a corpse entity while one exists; after removal
// (or for a MapUnitID 0 record, which never had one), its held tuple lives
// here and reserves its ID against future reuse.
type OriginalDeadRecord struct {
	OriginalDeadActor
	Current DeadActorState
}

type originalDeadRecord struct {
	OriginalDeadActor
	terminal DeadActorState // Stage 0: current state comes only from the entity
}

// deadStateFault admits the decay ladder's pre-terminal stages 2/3/4 and the
// terminal stage 5 (SAV-DEADLOAD-128's decay ladder and spell-family route);
// stage 0/1 (constructor, mission-end reset, revive, first dying tick) are
// not evidenced in any saved dead record and stay refused. Timer 0 is kept
// unconditionally: SAV-DEADLOAD-131 (High) shows its eight writers all sit in
// the live dying/order tick, never the dead-load or decay routes, and all 101
// archival records carry zero. Stage/HP/RuntimeID pairing and the cell-bounds
// checks are unchanged; SAV-DEADLOAD-126 (High) is why they are read directly
// rather than derived. Source-backed actors keep FineX/FineY through removal;
// virtual records require the canonical center.
func deadStateFault(s DeadActorState, b Bounds) error {
	return deadStateFaultWithPosition(s, b, false)
}

func deadStateFaultWithPosition(s DeadActorState, b Bounds, allowFinePosition bool) error {
	if s.Stage < 2 || s.Stage > 5 || s.Timer != 0 || s.HP > -10 ||
		(s.Stage == 5 && (s.HP >= -10000 || s.RuntimeID != 0)) ||
		(s.Stage < 5 && (s.RuntimeID == 0 || s.HP < -600)) ||
		(!allowFinePosition && (s.FineX != 128 || s.FineY != 128)) ||
		int32(s.Cell&255) >= b.Width || int32(s.Cell>>8) >= b.Height {
		return fmt.Errorf("unsupported late-dead state: stage %d HP %d timer %d runtime %d cell %#x fine %d,%d", s.Stage, s.HP, s.Timer, s.RuntimeID, s.Cell, s.FineX, s.FineY)
	}
	return nil
}

// deadSourceFault no longer rejects MapUnitID 0. SAV-DEADLOAD-125 (promoted,
// High) is that ROM1's own load never joins a dead actor to an authored map
// unit at all; MapUnitID 0 is the shape a saved actor that was never an ALM
// placement (a hired mercenary, observed below) carries, not corruption.
func deadSourceFault(s OriginalDeadSource, b Bounds) error {
	generated := (SourceBinding{Class: s.Class}).Generated()
	if s.Identity == 0 || s.Class < 1 || s.Class > GeneratedHumanBinding ||
		(!generated && s.ArchiveIndex == 0) || (generated && (s.ArchiveIndex != 0 || s.State.Stage != 5)) {
		return fmt.Errorf("invalid original dead identity/class")
	}
	if !s.ContainerPresent && s.ContainerTail != [2]uint32{} {
		return fmt.Errorf("absent original dead container carries tail")
	}
	weapon := s.HeldWeapon
	if !weapon.Present && weapon != (OriginalDeadWeapon{}) {
		return fmt.Errorf("absent original dead weapon carries state")
	}
	if weapon.Present && (s.Class != 1 || s.State.Stage != 5 || weapon.Identity == 0 || weapon.ArchiveIndex == 0 ||
		weapon.Identity == s.Identity || weapon.ArchiveIndex == s.ArchiveIndex) {
		return fmt.Errorf("unsupported original dead held-Weapon binding")
	}
	return deadStateFaultWithPosition(s.State, b, s.MapUnitID != 0)
}

// ImportOriginalDeadActors validates the entire uniquely joined batch before
// touching the candidate. It bypasses death, drops, kill credit and RNG.
//
// A MapUnitID-0 body remains virtual and reserves a fresh native ID. Nonzero
// IDs require one living authored placement. This is a bounded native join;
// ROM1 reconstructs the original graph instead (SAV-DEADLOAD-125).
func (w *World) ImportOriginalDeadActors(batch []OriginalDeadActor, current ...EntityID) error {
	if w == nil || len(w.originalDead) != 0 {
		return fmt.Errorf("original dead import requires a fresh world")
	}
	currentIDs := make(map[EntityID]bool, len(current))
	for _, id := range current {
		if currentIDs[id] {
			return fmt.Errorf("repeated current dead actor binding")
		}
		currentIDs[id] = true
	}
	seenID, seenKey, seenMap, seenIndex := map[EntityID]bool{}, map[uint32]bool{}, map[uint16]bool{}, map[uint16]bool{}
	for _, d := range batch {
		if err := deadSourceFault(d.Source, w.bounds); err != nil {
			return err
		}
		if seenID[d.ID] || seenKey[d.Source.Identity] || d.Source.ArchiveIndex != 0 && seenIndex[d.Source.ArchiveIndex] ||
			d.Source.MapUnitID != 0 && seenMap[d.Source.MapUnitID] {
			return fmt.Errorf("ambiguous original dead actor %d", d.ID)
		}
		seenID[d.ID], seenKey[d.Source.Identity], seenIndex[d.Source.ArchiveIndex] = true, true, true
		if d.Source.MapUnitID != 0 {
			seenMap[d.Source.MapUnitID] = true
		}
		if weapon := d.Source.HeldWeapon; weapon.Present {
			if seenKey[weapon.Identity] || seenIndex[weapon.ArchiveIndex] {
				return fmt.Errorf("ambiguous original dead held-Weapon identity")
			}
			seenKey[weapon.Identity], seenIndex[weapon.ArchiveIndex] = true, true
		}
		if currentIDs[d.ID] {
			i := indexOfEntity(w.entities, d.ID)
			if i < 0 || w.entities[i].Alive() || w.entities[i].OffMap || d.Source.MapUnitID == 0 || d.Source.State.Stage == 5 {
				return fmt.Errorf("current dead actor %d has no exact existing body", d.ID)
			}
			e, s := w.entities[i], d.Source.State
			if e.MapUnitID != d.Source.MapUnitID || int16(e.HP) != s.HP || e.X != int32(s.Cell&255) || e.Y != int32(s.Cell>>8) {
				return fmt.Errorf("current dead actor %d differs from its ordinary body", d.ID)
			}
			continue
		}
		if d.Source.MapUnitID == 0 {
			if indexOfEntity(w.entities, d.ID) >= 0 {
				return fmt.Errorf("original dead actor %d collides with a live entity", d.ID)
			}
			continue
		}
		// The fresh map's own body may already be dead: an ALM placement can
		// author current health 0 (map 131 places five such corpses). The
		// saved tuple replaces that body's lifecycle exactly as a living one.
		i := indexOfEntity(w.entities, d.ID)
		if i < 0 || w.entities[i].OffMap || w.entities[i].MapUnitID != d.Source.MapUnitID {
			return fmt.Errorf("original dead actor %d has no unique map binding", d.ID)
		}
		if w.actorHasSavedItems(i) {
			return fmt.Errorf("original dead import cannot replace bound object ownership")
		}
	}
	for id := range currentIDs {
		if !seenID[id] {
			return fmt.Errorf("current dead actor %d has no retained row", id)
		}
	}
	var gone []EntityID
	for _, d := range batch {
		s := d.Source.State
		r := originalDeadRecord{OriginalDeadActor: d}
		if currentIDs[d.ID] {
			e := &w.entities[indexOfEntity(w.entities, d.ID)]
			e.Decay, e.Dwell, e.SuppressCorpseLoot = DecayStage(s.Stage), 0, true
			w.originalDead = append(w.originalDead, r)
			continue
		}
		if d.Source.MapUnitID == 0 {
			// The dead-list clock owns this tuple without a live entity or
			// occupancy.
			r.terminal = s
			w.originalDead = append(w.originalDead, r)
			continue
		}
		i := indexOfEntity(w.entities, d.ID)
		e := &w.entities[i]
		e.X, e.Y = int32(s.Cell&255), int32(s.Cell>>8)
		e.HP, e.Decay, e.Dwell = int32(s.HP), DecayStage(min(s.Stage, 4)), 0
		// Setting the late stage first makes clearFelled a cleanup only. It
		// cannot halve defence, arm a dwell, or enter the death/drop boundary.
		w.clearFelled(i)
		e.clearKillCredit()
		e.SuppressCorpseLoot = true
		e.GoldChance, e.TreasureMin, e.TreasureMax = 0, 0, 0
		w.carried[i], w.equipment[i] = nil, [EquipSlots]ItemInstance{}
		syncWeaponItem(e, ItemInstance{})
		w.recomputeLoad(i)
		w.clearInvalidTargetReferences(d.ID)
		if s.Stage == 5 {
			r.terminal = s
			e.NativeBasis = NativeActorBasis{}
			gone = append(gone, d.ID)
		}
		w.originalDead = append(w.originalDead, r)
	}
	if len(gone) != 0 {
		w.remove(gone)
	}
	return nil
}

func (w *World) OriginalDeadActors() []OriginalDeadRecord {
	out := make([]OriginalDeadRecord, 0, len(w.originalDead))
	for _, r := range w.originalDead {
		current := r.terminal
		if current.Stage == 0 {
			// A missing nonterminal binding is an internal invariant error,
			// never permission to recreate the original actor on a load.
			i := indexOfEntity(w.entities, r.ID)
			if i >= 0 {
				e := w.entities[i]
				current = r.Source.State
				current.HP, current.Stage = int16(e.HP), uint8(e.Decay)
				current.Cell = uint16(e.Y)<<8 | uint16(e.X)
			}
		}
		out = append(out, OriginalDeadRecord{r.OriginalDeadActor, current})
	}
	return out
}

func (w *World) OriginalDeadActorCount() int {
	return len(w.originalDead)
}

func (w *World) RetainOriginalDeadReferences(id EntityID, identity uint32, references [5]uint32) bool {
	for i := range w.originalDead {
		r := &w.originalDead[i]
		if r.ID == id && r.Source.Identity == identity && r.terminal.Stage == 5 {
			r.Source.References = references
			return true
		}
	}
	return false
}

// decayHeldDead applies the dead-list ladder to records without an authored
// entity. It never replays teardown, drops, rewards or live-actor actions.
func (w *World) decayHeldDead(decrement bool) {
	for i := range w.originalDead {
		r := &w.originalDead[i]
		s := &r.terminal
		if r.Source.MapUnitID != 0 || s.Stage < 2 || s.Stage >= 5 {
			continue
		}
		beforeHP, beforeStage := s.HP, s.Stage
		if decrement {
			s.HP--
		}
		stage := decayStageFor(int32(s.HP))
		if stage > decayLast {
			s.Stage, s.HP, s.RuntimeID = 5, -10001, 0
		} else if uint8(stage) > s.Stage {
			s.Stage = uint8(stage)
		}
		if r.Source.MapUnitID != 0 && (beforeHP != s.HP || beforeStage != s.Stage) {
			r.Source.State.HP, r.Source.State.Stage, r.Source.State.RuntimeID = s.HP, s.Stage, s.RuntimeID
		}
	}
}

func (w *World) syncOriginalDeadState(e Entity, beforeHP int32, beforeStage DecayStage) {
	if beforeHP == e.HP && beforeStage == e.Decay {
		return
	}
	for i := range w.originalDead {
		r := &w.originalDead[i]
		if r.ID != e.ID || r.terminal.Stage != 0 {
			continue
		}
		if r.Source.MapUnitID != 0 {
			r.Source.State.HP, r.Source.State.Stage = int16(e.HP), uint8(e.Decay)
		}
	}
}

// Retain exact source bindings when a source-backed actor reaches the native
// terminal boundary. The current tuple survives entity compaction and a native
// save, so a later SAV producer cannot resurrect the historical living record.
// This is native retirement provenance, not an original heap constructor.
func (w *World) retireOriginalDead(id EntityID) {
	found := false
	for i := range w.originalDead {
		r := &w.originalDead[i]
		if r.ID != id {
			continue
		}
		found = true
		if r.terminal.Stage != 0 {
			continue
		}
		r.terminal = r.Source.State
		if ei := indexOfEntity(w.entities, id); ei >= 0 {
			e := w.entities[ei]
			r.terminal.Cell = uint16(e.Y)<<8 | uint16(e.X)
		}
		r.terminal.Stage, r.terminal.HP, r.terminal.RuntimeID = 5, -10001, 0
		if r.Source.MapUnitID != 0 {
			r.Source.State = r.terminal
		}
	}
	if found {
		return
	}
	ei := indexOfEntity(w.entities, id)
	if ei < 0 {
		return
	}
	e := w.entities[ei]
	if e.SourceBinding.Class == 0 {
		if e.HP < decayGoneHP {
			w.currentTerminalActors = append(w.currentTerminalActors, CurrentTerminalActor{
				ID: id, Cell: uint16(e.Y)<<8 | uint16(e.X), HP: e.HP, Stage: uint8(e.Decay), MapUnitID: e.MapUnitID,
			})
			slices.SortFunc(w.currentTerminalActors, func(a, b CurrentTerminalActor) int {
				if a.ID < b.ID {
					return -1
				}
				if a.ID > b.ID {
					return 1
				}
				return 0
			})
		}
		return
	}
	if e.SourceBinding.Identity == 0 {
		return
	}
	state := DeadActorState{Cell: uint16(e.Y)<<8 | uint16(e.X), FineX: 128, FineY: 128, Stage: 5, HP: -10001}
	source := OriginalDeadSource{Identity: e.SourceBinding.Identity, ArchiveIndex: e.SourceBinding.ArchiveIndex,
		MapUnitID: e.MapUnitID, Class: e.SourceBinding.Class, OwnerKey: e.SourceBinding.GroupOwnerKey,
		ContainerPresent: e.ActorLoad.ContainerPresent,
		ContainerTail:    [2]uint32{e.ActorLoad.InsertIndex, uint32(e.ActorLoad.Accumulator)}, State: state}
	if m := w.motionFor(id); m != nil {
		source.TerrainKey = m.Position.TerrainKey
		if e.Decay < DecayBones || e.HP >= decayGoneHP {
			m.Position.Cell, m.Position.PackedCell = state.Cell, state.Cell
			m.Position.FineX, m.Position.FineY = state.FineX, state.FineY
		}
	}
	w.originalDead = append(w.originalDead, originalDeadRecord{OriginalDeadActor: OriginalDeadActor{ID: id, Source: source}, terminal: state})
}

// NextEntityID is above every extant, imported dead or reserved identity.
// A removed actor can never be minted as a different actor. This query does
// not consume the returned ID; a caller must publish its record before asking
// for another. false means the EntityID space is exhausted.
func (w *World) NextEntityID() (EntityID, bool) {
	next := w.entityIDFloor
	if len(w.entities) > 0 {
		next = max(next, uint64(w.entities[len(w.entities)-1].ID)+1)
	}
	for _, r := range w.originalDead {
		next = max(next, uint64(r.ID)+1)
	}
	for _, r := range w.currentTerminalActors {
		next = max(next, uint64(r.ID)+1)
	}
	if next >= entityIDLimit {
		return 0, false
	}
	return EntityID(next), true
}
