package game

import (
	"encoding/json"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Only identity, admission and availability are read from the supplement.
// Ordinary Body bytes remain the expected combat and holdings values.
type milestoneActorCurrent struct {
	ID                                           sim.EntityID
	SourceBound, LoadPresent, NativeClassPresent bool
	SourceClass                                  uint8
	Basis                                        milestoneActorBasisMask
	Terminal                                     *sim.CurrentTerminalActor
	Legacy                                       bool
	NativeParty                                  bool
}

type milestoneActorBasisMask struct {
	BasePresent                   bool
	BaseKnown                     uint32
	ModifierPresent               bool
	ModifierKnown                 uint64
	BodyPresent, BodyKnown        bool
	AttackPresent, DefencePresent bool
	AttackKnown, DefenceKnown     uint32
	ScalarsPresent                bool
	ScalarKnown                   uint32
	BlockPresent                  bool
	BlockKnown                    uint16
}

type milestoneActorPartyMode struct {
	Entity sim.EntityID
	Base   *struct {
		Native bool
		Lifts  []json.RawMessage
	}
	Template *json.RawMessage
}

type milestoneActorPresence struct {
	Entity  sim.EntityID
	Current *struct{ NativeBasis *milestoneActorBasisMask }
}

func milestoneActorCurrentInputs(doc *sav.DocumentData) (map[uint16]*milestoneActorCurrent, error) {
	transport, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		return nil, err
	}
	var input struct {
		Version              uint32
		NativeHistoryVersion uint8
		Bindings             []struct {
			ID                 sim.EntityID
			Object             uint16
			Missing, Structure bool
		}
		Values map[sim.EntityID]struct {
			SourceBound, LoadPresent, NativeClassPresent bool
			SourceClass                                  uint8
			CurrentTerminal                              *sim.CurrentTerminalActor
		}
		Actions       struct{ Actors []milestoneActorPresence }
		Held          []milestoneActorPresence
		Party, Roster []milestoneActorPartyMode
	}
	if err := json.Unmarshal(transport, &input); err != nil {
		return nil, err
	}
	if input.Version != 1 || input.NativeHistoryVersion > 1 || len(input.Bindings) > 131072 || len(input.Actions.Actors)+len(input.Held) > 65535 {
		return nil, fmt.Errorf("actor subject input version/population invalid")
	}
	objects, ids := map[uint16]bool{}, map[sim.EntityID]uint16{}
	for _, b := range input.Bindings {
		if b.Structure {
			continue
		}
		if _, duplicate := ids[b.ID]; duplicate || b.Missing != (b.Object == 0) || b.Object != 0 && objects[b.Object] {
			return nil, fmt.Errorf("actor subject input binding ambiguous")
		}
		ids[b.ID] = b.Object
		if b.Object != 0 {
			objects[b.Object] = true
		}
	}
	out, seen := map[uint16]*milestoneActorCurrent{}, map[sim.EntityID]bool{}
	for _, rows := range [][]milestoneActorPresence{input.Actions.Actors, input.Held} {
		for _, row := range rows {
			object, bound := ids[row.Entity]
			value, available := input.Values[row.Entity]
			if seen[row.Entity] || !bound || object == 0 || int(object) > len(doc.Objects) || !available {
				return nil, fmt.Errorf("actor subject input presence lacks an exact value/object binding")
			}
			seen[row.Entity] = true
			v := &milestoneActorCurrent{ID: row.Entity, SourceBound: value.SourceBound, LoadPresent: value.LoadPresent,
				SourceClass: value.SourceClass, NativeClassPresent: value.NativeClassPresent}
			if row.Current != nil && row.Current.NativeBasis != nil {
				v.Basis = *row.Current.NativeBasis
				if v.Basis.BaseKnown & ^uint32(0x00ffffff) != 0 || !v.Basis.BasePresent && v.Basis.BaseKnown != 0 || !v.Basis.ModifierPresent && v.Basis.ModifierKnown != 0 || !v.Basis.BodyPresent && v.Basis.BodyKnown {
					return nil, fmt.Errorf("actor subject input basis mask invalid")
				}
				if v.Basis.AttackKnown & ^uint32(0x00ffffff) != 0 || !v.Basis.AttackPresent && v.Basis.AttackKnown != 0 || v.Basis.DefenceKnown & ^uint32((1<<22)-1) != 0 || !v.Basis.DefencePresent && v.Basis.DefenceKnown != 0 || v.Basis.ScalarKnown & ^uint32((1<<sim.ScalarCount)-1) != 0 || !v.Basis.ScalarsPresent && v.Basis.ScalarKnown != 0 || v.Basis.BlockKnown & ^uint16((1<<10)-1) != 0 || !v.Basis.BlockPresent && v.Basis.BlockKnown != 0 {
					return nil, fmt.Errorf("actor subject input scalar/combat mask invalid")
				}
			}
			v.Legacy = input.NativeHistoryVersion == 0 && !v.SourceBound && v.SourceClass == 0
			if v.Legacy {
				if !v.Basis.BasePresent {
					v.Basis.BasePresent, v.Basis.BaseKnown = true, 0x00ffffff
				}
				if !v.Basis.ModifierPresent {
					v.Basis.ModifierPresent, v.Basis.ModifierKnown = true, ^uint64(0)
				}
				if !v.Basis.BodyPresent {
					v.Basis.BodyPresent, v.Basis.BodyKnown = true, true
				}
				if !v.Basis.AttackPresent {
					v.Basis.AttackPresent, v.Basis.AttackKnown = true, 0x00ffffff
				}
				if !v.Basis.DefencePresent {
					v.Basis.DefencePresent, v.Basis.DefenceKnown = true, (1<<22)-1
				}
				if !v.Basis.ScalarsPresent {
					v.Basis.ScalarsPresent, v.Basis.ScalarKnown = true, (1<<sim.ScalarCount)-1
				}
				if !v.Basis.BlockPresent {
					v.Basis.BlockPresent, v.Basis.BlockKnown = true, (1<<10)-1
				}
			}
			out[object] = v
		}
	}
	var rawValues struct {
		Values map[sim.EntityID]json.RawMessage
	}
	if err := json.Unmarshal(transport, &rawValues); err != nil {
		return nil, err
	}
	for id, value := range input.Values {
		if value.CurrentTerminal == nil {
			continue
		}
		var pure sim.ActorValues
		if err := json.Unmarshal(rawValues.Values[id], &pure); err != nil || !pureCurrentTerminalActorValue(pure, id) {
			return nil, fmt.Errorf("actor subject terminal value is not pure")
		}
		object, bound := ids[id]
		if seen[id] || !bound || object != 0 && (int(object) > len(doc.Objects) || !slices.Contains(doc.DeadActors, object)) {
			return nil, fmt.Errorf("actor subject terminal lacks exact dead root")
		}
		if object == 0 {
			continue
		}
		seen[id] = true
		out[object] = &milestoneActorCurrent{ID: id, Terminal: value.CurrentTerminal}
	}
	for _, rows := range [][]milestoneActorPartyMode{input.Party, input.Roster} {
		seen := map[sim.EntityID]bool{}
		for _, row := range rows {
			if seen[row.Entity] {
				return nil, fmt.Errorf("actor subject native party identity ambiguous")
			}
			seen[row.Entity] = true
			if row.Template != nil || row.Base == nil || !row.Base.Native || len(row.Base.Lifts) != 0 {
				continue
			}
			if current := out[ids[row.Entity]]; current != nil && current.ID == row.Entity && current.Terminal == nil && !current.SourceBound && current.SourceClass == 0 {
				current.NativeParty = true
			}
		}
	}
	return out, nil
}

type milestoneActorSubjects struct {
	byArchive    map[uint16]sim.Entity
	byID         map[sim.EntityID]sim.Entity
	duplicateIDs map[sim.EntityID]bool
	state        *SnapshotSAVDocument
	mission      *Mission
	usedIDs      map[sim.EntityID]bool
	nativeSeen   bool
}

func newMilestoneActorSubjects(observed []sim.Entity, state *SnapshotSAVDocument, contexts ...*Mission) (*milestoneActorSubjects, []string) {
	s := &milestoneActorSubjects{byArchive: map[uint16]sim.Entity{}, byID: map[sim.EntityID]sim.Entity{}, duplicateIDs: map[sim.EntityID]bool{}, state: state, usedIDs: map[sim.EntityID]bool{}}
	var differences []string
	if len(contexts) == 1 {
		s.mission = contexts[0]
	} else if len(contexts) > 1 {
		differences = append(differences, "actor subject Mission context ambiguous")
	}
	for _, e := range observed {
		if _, duplicate := s.byID[e.ID]; duplicate {
			s.duplicateIDs[e.ID] = true
			differences = append(differences, "actor subject live ID duplicated")
		}
		s.byID[e.ID] = e
		if e.SourceBinding.Class != 0 {
			if _, duplicate := s.byArchive[e.SourceBinding.ArchiveIndex]; duplicate {
				differences = append(differences, "duplicate live actor SourceBinding archive")
			}
			s.byArchive[e.SourceBinding.ArchiveIndex] = e
		}
	}
	return s, differences
}

func (s *milestoneActorSubjects) actor(archive uint16, off int, class string, identity, runtime uint32, object uint16, current *milestoneActorCurrent) (sim.Entity, bool, bool, string) {
	if current != nil && current.Terminal != nil {
		return sim.Entity{}, false, true, ""
	}
	if current == nil || current.SourceBound || current.SourceClass != 0 {
		e, ok := s.byArchive[archive]
		delete(s.byArchive, archive)
		return e, ok, false, ""
	}
	m := s.mission
	s.nativeSeen = true
	if m == nil || m.World == nil || m.savedDocument != s.state || m.actorRegistry == nil || s.state == nil || s.state.Document == nil {
		return sim.Entity{}, false, true, "native actor subject Mission/registry context unavailable"
	}
	i, indexed := m.actorRegistry.byOff[off]
	if !indexed || i < 0 || i >= len(m.actorRegistry.actors) {
		return sim.Entity{}, false, true, "native actor subject registry offset unavailable"
	}
	b, ok := m.actorRegistry.actor(off)
	if !ok || b.Source.Off != off || b.Source.ArchiveIndex != archive || b.Source.Class != class || b.Source.Identity != identity || b.Source.RuntimeID != runtime || b.ID != current.ID {
		return sim.Entity{}, false, true, "native actor subject registry metadata/ID differs"
	}
	if object == 0 || int(object) > len(s.state.Document.Objects) {
		return sim.Entity{}, false, true, "native actor subject ordinary Object unavailable"
	}
	r := &s.state.Document.Objects[object-1]
	key, keyErr := savedStructureValue(r, "Identity")
	rid, runtimeErr := savedStructureValue(r, "RuntimeID")
	if r.Class != class || keyErr != nil || key != identity || runtimeErr != nil || rid != runtime {
		return sim.Entity{}, false, true, "native actor subject ordinary class/identity differs"
	}
	receipts := 0
	for _, receipt := range s.state.Actors {
		if receipt.EntityID == b.ID || receipt.ObjectIndex == object {
			receipts++
			if receipt.EntityID != b.ID || receipt.ObjectIndex != object || receipt.Retired {
				return sim.Entity{}, false, true, "native actor subject ordinary receipt conflicts"
			}
		}
	}
	e, live := s.byID[b.ID]
	if receipts != 1 || !live || s.duplicateIDs[b.ID] || s.usedIDs[b.ID] {
		return sim.Entity{}, false, true, "native actor subject live/ordinary receipt unavailable or aliased"
	}
	if e.SourceBinding != (sim.SourceBinding{}) || e.ActorLoad.Source != (sim.SourceActor{}) || e.ActorLoad.Present != current.LoadPresent || e.NativeClass.Present != current.NativeClassPresent || e.Humanoid != (class != "Unit") {
		return sim.Entity{}, false, true, "native actor subject arithmetic absence/class differs"
	}
	s.usedIDs[b.ID] = true
	return e, true, true, ""
}

func (s *milestoneActorSubjects) unexpectedNative() int {
	if !s.nativeSeen || s.mission == nil {
		return 0
	}
	n := 0
	for id, e := range s.byID {
		if e.SourceBinding.Class == 0 && !s.usedIDs[id] {
			n++
		}
	}
	return n
}

func milestoneNativeCombatDifferences(prefix string, e sim.Entity, current *milestoneActorCurrent, raw map[string][]byte) []string {
	var differences []string
	b, m := e.NativeBasis, current.Basis
	if err := b.Validate(); err != nil {
		differences = append(differences, prefix+": native basis invalid: "+err.Error())
	}
	if b.BasePresent != m.BasePresent || b.BaseKnown != m.BaseKnown || b.ModifierPresent != m.ModifierPresent || b.ModifierKnown != m.ModifierKnown || b.BodyPresent != m.BodyPresent || b.BodyKnown != m.BodyKnown || b.AttackPresent != m.AttackPresent || b.AttackKnown != m.AttackKnown || b.DefencePresent != m.DefencePresent || b.DefenceKnown != m.DefenceKnown {
		differences = append(differences, prefix+": native basis presence/mask differs")
	}
	for _, block := range []struct {
		name  string
		value []byte
		known uint64
	}{
		{"U114", b.Base[:], uint64(m.BaseKnown)}, {"UD4", b.Modifier[:], m.ModifierKnown},
		{"UA6", b.Attack[:], uint64(m.AttackKnown)}, {"UBE", b.Defence[:], uint64(m.DefenceKnown)},
	} {
		unknown := 0
		for n := range block.value {
			if block.known&(uint64(1)<<n) == 0 {
				unknown++
				continue
			}
			if block.value[n] != raw[block.name][n] {
				differences = append(differences, fmt.Sprintf("%s: native %s byte%d differs", prefix, block.name, n))
			}
		}
		if unknown != 0 {
			differences = append(differences, fmt.Sprintf("%s: native %s basis byte carrier unavailable for %d bytes; retained Document and scalar projections compared separately", prefix, block.name, unknown))
		}
	}
	return differences
}
