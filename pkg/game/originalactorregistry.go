package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// originalActorBinding separates source identity from native entity identity.
// New names an actor absent from the surviving ALM and initial party. No native
// ID is copied into MapUnitID, even when the latter is zero or duplicated.
type originalActorBinding struct {
	Source sav.ActorRecord
	ID     sim.EntityID
	Party  bool
	New    bool
}

type originalActorRegistry struct {
	actors   []originalActorBinding
	byOff    map[int]int
	groups   []sav.ActorGroup
	sources  []sav.ActorRecord
	profiles map[uint32]sim.CurrentProfileBasis
	health   map[sim.EntityID]currentAdmissionHealth
}

func (r *originalActorRegistry) actor(off int) (originalActorBinding, bool) {
	if r == nil {
		return originalActorBinding{}, false
	}
	i, ok := r.byOff[off]
	if !ok {
		return originalActorBinding{}, false
	}
	return r.actors[i], true
}

// ownsOriginalEntity reports whether a source actor is constructed as an
// Entity: living, dying, or explicitly current, and never a current terminal.
func ownsOriginalEntity(actor sav.ActorRecord) bool {
	return !actor.TerminalActor && (!actor.Dead() || actor.Dying() || actor.CurrentEntity)
}

// planOriginalActors is a detached admission plan. Existing ALM IDs retain the
// old party-withdrawal convention; source-only IDs follow that whole namespace.
// Duplicate source map IDs with no ALM target remain distinct source actors.
// Several claimants of one ALM unit resolve by a fixed rule (DIV-1813); two ALM
// units with one ID are refused.
func planOriginalActors(m *alm.Map, graph sav.SavedActorGraph, party []mapload.PartyMember,
	sourceOffsets []int, reserved ...sim.EntityID) (*originalActorRegistry, []alm.Unit, error) {
	if m == nil || len(sourceOffsets) != 0 && len(sourceOffsets) != len(party) {
		return nil, nil, fmt.Errorf("original actor registry: missing map or source party binding")
	}
	actors := append([]sav.ActorRecord(nil), graph.Actors...)
	slices.SortFunc(actors, func(a, b sav.ActorRecord) int { return int(a.ArchiveIndex) - int(b.ArchiveIndex) })
	byOff, byIndex, byKey := map[int]int{}, map[uint16]bool{}, map[uint32]bool{}
	sourceCounts := map[uint16]int{}
	byIndexPos := map[uint16]int{}
	for i, actor := range actors {
		if _, ok := byOff[actor.Off]; ok || actor.ArchiveIndex == 0 || byIndex[actor.ArchiveIndex] || actor.Identity != 0 && byKey[actor.Identity] {
			return nil, nil, fmt.Errorf("original actor registry: ambiguous source actor at %d", actor.Off)
		}
		byOff[actor.Off], byIndex[actor.ArchiveIndex], byKey[actor.Identity] = i, true, true
		byIndexPos[actor.ArchiveIndex] = i
		if ownsOriginalEntity(actor) && actor.MapUnitID != 0 {
			sourceCounts[actor.MapUnitID]++
		}
	}
	placements := map[uint16][]int{}
	for i, unit := range m.Units {
		if unit.UnitID != 0 {
			placements[unit.UnitID] = append(placements[unit.UnitID], i)
		}
	}
	// A saved MapUnitID may be claimed by several actors: the original re-creates
	// hired units while the loaded ones survive, so the copies keep one stale ID.
	// The original's own rule is Unknown (DIV-1813). Exactly one claimant binds
	// to the single ALM unit carrying the ID: the one whose owner slot matches
	// the ALM owner and whose type matches the ALM class, then the one whose
	// owner matches, then the lowest ArchiveIndex. Every other claimant stays a
	// distinct source-only actor, so no Unit is dropped or invented.
	winner := map[uint16]uint16{}
	for _, actor := range actors {
		if !ownsOriginalEntity(actor) || actor.MapUnitID == 0 || len(placements[actor.MapUnitID]) == 0 {
			continue
		}
		if len(placements[actor.MapUnitID]) != 1 {
			return nil, nil, fmt.Errorf("original actor registry: ambiguous ALM binding for source actor %d MapUnitID %d", actor.ArchiveIndex, actor.MapUnitID)
		}
		if sourceCounts[actor.MapUnitID] == 1 {
			winner[actor.MapUnitID] = actor.ArchiveIndex
			continue
		}
		unit := m.Units[placements[actor.MapUnitID][0]]
		score := func(a sav.ActorRecord) int {
			n := 0
			if uint32(a.OwnerSlot) == unit.Owner {
				n += 2
				if int32(a.TypeID) == int32(unit.ClassID) {
					n++
				}
			}
			return n
		}
		if best, ok := winner[actor.MapUnitID]; !ok || score(actor) > score(actors[byIndexPos[best]]) {
			winner[actor.MapUnitID] = actor.ArchiveIndex
		}
	}
	binds := func(a sav.ActorRecord) bool {
		w, ok := winner[a.MapUnitID]
		return a.MapUnitID != 0 && ok && w == a.ArchiveIndex
	}
	partyAt, withdrawn := map[int]int{}, map[int]bool{}
	for i, off := range sourceOffsets {
		actorAt, ok := byOff[off]
		if _, duplicate := partyAt[off]; !ok || duplicate || !ownsOriginalEntity(actors[actorAt]) {
			return nil, nil, fmt.Errorf("original actor registry: invalid source party actor at %d", off)
		}
		partyAt[off] = i
		actor := actors[actorAt]
		if match := placements[actor.MapUnitID]; binds(actor) && len(match) == 1 {
			withdrawn[match[0]] = true
		}
	}
	units := make([]alm.Unit, 0, len(m.Units)-len(withdrawn))
	remaining := map[uint16]sim.EntityID{}
	for i, unit := range m.Units {
		if withdrawn[i] {
			continue
		}
		if unit.UnitID != 0 {
			remaining[unit.UnitID] = sim.EntityID(len(units))
		}
		units = append(units, unit)
	}
	next := uint64(len(units)) + uint64(len(party))
	for _, id := range reserved {
		if uint64(id) >= next {
			next = uint64(id) + 1
		}
	}
	if next > uint64(^sim.EntityID(0)) {
		return nil, nil, fmt.Errorf("original actor registry: entity namespace exhausted")
	}
	registry := &originalActorRegistry{byOff: map[int]int{}, groups: graph.Groups, sources: graph.Actors}
	for _, actor := range actors {
		if !ownsOriginalEntity(actor) {
			continue
		}
		binding := originalActorBinding{Source: actor}
		if index, ok := partyAt[actor.Off]; ok {
			binding.ID, binding.Party = sim.EntityID(len(units)+index), true
		} else if id, ok := remaining[actor.MapUnitID]; binds(actor) && ok {
			binding.ID = id
		} else {
			if next > uint64(^sim.EntityID(0)) {
				return nil, nil, fmt.Errorf("original actor registry: entity namespace exhausted")
			}
			binding.ID, binding.New = sim.EntityID(next), true
			next++
		}
		if actor.Class == "Humanoid" && (binding.New || binding.Party) {
			return nil, nil, fmt.Errorf("original actor %d: source-only Humanoid construction is unsupported", actor.ArchiveIndex)
		}
		registry.byOff[actor.Off] = len(registry.actors)
		registry.actors = append(registry.actors, binding)
	}
	return registry, units, nil
}

func prepareOriginalActorRegistry(m *alm.Map, graph sav.SavedActorGraph, party []mapload.PartyMember, report *OriginalSaveResume) (*originalActorRegistry, error) {
	registry, units, err := planOriginalActors(m, graph, party, report.Party.SourceOffsets)
	if err != nil {
		return nil, err
	}
	for _, binding := range registry.actors {
		a := binding.Source
		// An exact current binding retains the native position even outside
		// terrain bounds. OffMap remains the independent script state.
		if !a.CurrentEntity && !a.CurrentOffMap && (int(a.Cell&255) >= int(m.Width) || int(a.Cell>>8) >= int(m.Height)) {
			return nil, fmt.Errorf("original actor %d at %d is outside the map", a.ArchiveIndex, a.Off)
		}
		if binding.New || binding.Party {
			continue
		}
		u := &units[binding.ID]
		x, y := uint32(a.Cell&255)<<8|uint32(a.FineX), uint32(a.Cell>>8)<<8|uint32(a.FineY)
		if u.X>>8 != x>>8 || u.Y>>8 != y>>8 {
			report.Moved++
		}
		report.Joined++
		if u.Owner != uint32(a.OwnerSlot) {
			report.Reowned++
		}
		u.X, u.Y = x, y
		u.Owner = uint32(a.OwnerSlot)
	}
	report.Party.Withdrawn = len(m.Units) - len(units)
	m.Units = units
	return registry, nil
}

func admitOriginalActorRegistry(ms *Mission, registry *originalActorRegistry, table *mapload.Table, states ...*SnapshotSAVDocument) error {
	if ms == nil || ms.World == nil || registry == nil {
		return fmt.Errorf("original actor registry has no candidate")
	}
	var state *SnapshotSAVDocument
	if len(states) > 1 {
		return fmt.Errorf("original actor registry has repeated current document")
	}
	if len(states) == 1 {
		state = states[0]
	}
	profiles, err := currentProfileBases(state)
	if err != nil {
		return err
	}
	admissionHealth, err := currentAdmissionHealthByIdentity(state)
	if err != nil {
		return err
	}
	healthPlan := make(map[sim.EntityID]currentAdmissionHealth)
	absentClasses, err := currentAbsentDeadClasses(state)
	if err != nil {
		return err
	}
	var absentPeople []sim.EntityID
	byID := map[sim.EntityID]sim.Entity{}
	for _, e := range ms.World.Entities() {
		byID[e.ID] = e
	}
	manifest := &SnapshotActorManifest{Version: actorManifestVersion}
	hiredParty := map[sim.EntityID]bool{}
	for i, member := range ms.Party {
		if i < len(ms.Start.IDs) && member.Hired() {
			hiredParty[ms.Start.IDs[i]] = true
		}
	}
	var batch []sim.OriginalLivingActor
	var current []sim.EntityID
	for _, binding := range registry.actors {
		a := binding.Source
		class := uint8(1)
		if a.Class == "Human" {
			class = 2
		} else if a.Class == "Humanoid" {
			class = 3 // exact source class, not a Human definition alias
		}
		typeID := a.TypeID
		if class == 2 {
			typeID = reconciledRoodTypeID(typeID, a.ClassSelector, ms.Number, a.MapUnitID)
		}
		s := sim.SourceBinding{Class: class, ArchiveIndex: a.ArchiveIndex, Identity: a.Identity, RuntimeID: a.RuntimeID,
			TokenRow: a.DefRow, TypeID: typeID, Face: a.Face, ClassFlags: a.ClassSelector &^ sav.ActorOffMapFlag, DisplayBacking: a.DisplayBacking, GroupIndex: a.Group}
		if hiredParty[binding.ID] && s.DisplayBacking == 0 {
			if typ, ok := originalMercenaryType(a.Character, table); ok {
				s.DisplayBacking = uint32(typ)
			}
		}
		if a.Group != 0 {
			if a.Group > uint32(len(registry.groups)) {
				return fmt.Errorf("original actor has missing Group %d", a.Group)
			}
			g := registry.groups[a.Group-1]
			s.GroupSelector, s.GroupOwnerKey, s.GroupOwnerSlot, s.GroupOwnerResolved = g.Selector, g.Owner.Key, g.Owner.PlayerSlot, g.Owner.Resolved
		}
		e, exists := byID[binding.ID]
		classAbsent := a.CurrentEntity && !binding.New && absentClasses[a.Identity] == class
		if binding.New {
			var err error
			e, err = mapload.SourceActorSeed(s, table)
			if err != nil {
				return fmt.Errorf("original actor %d: %w", a.ArchiveIndex, err)
			}
		} else if !exists {
			return fmt.Errorf("original actor %d missing native binding", a.ArchiveIndex)
		}
		if !classAbsent && s.ActorClass() == 2 && table != nil && table.Humans != nil && int(s.DefinitionRow()) < table.Humans.Len() {
			seed, err := mapload.SourceActorSeed(s, table)
			if err != nil {
				return err
			}
			e.DyingTime = seed.DyingTime
			e.PotionHeadroom = [4]int32{} // source caps own potion gains
		}
		domain, err := mapload.SourceActorDomain(a.Domain)
		if err != nil {
			return fmt.Errorf("original actor %d: %w", a.ArchiveIndex, err)
		}
		if a.Character.Basis == nil {
			return fmt.Errorf("original actor %d lacks saved basis", a.ArchiveIndex)
		}
		basis := originalActorBasis(a.Character.Basis)
		basis.TypeID = typeID
		load := originalActorLoad(a.Character.LoadState, int32(int16(basis.Stats[4])))
		if load == nil {
			return fmt.Errorf("original actor %d lacks saved load", a.ArchiveIndex)
		}
		if basis.Class == 1 {
			restoreUnitLoad(load, &basis)
		}
		load.Inventory.Source = basis
		// The fresh map's weapon cache is not an independent SAV source.
		// Saved stock rebuilds it for matched and newly created actors alike.
		e.WeaponSpell, e.WeaponSpellLevel, e.WeaponSpellSource = 0, 0, sim.WeaponSpellNone
		e.ID, e.SourceBinding, e.MapUnitID = binding.ID, s, a.MapUnitID
		e.X, e.Y, e.PostX, e.PostY = int32(a.Cell&255), int32(a.Cell>>8), int32(a.Cell&255), int32(a.Cell>>8)
		if a.CurrentEntity {
			e.OffMap = a.CurrentOffMap
		} else {
			// MOVE-TICK-017: ROM1's LOAD leaves a U4C-bit-3 actor off the map.
			e.OffMap = a.ClassSelector&sav.ActorOffMapFlag != 0
		}
		e.Owner = uint32(a.OwnerSlot)
		// Group is the authored native selector used by mission checks and
		// group orders. Keep that binding for existing actors. SourceActorSeed
		// leaves new actors unbound (Group0); raw source membership/selector
		// stays separate in SourceBinding until saved Group runtime is owned.
		e.Class, e.TypeID, e.Humanoid = int32(uint8(typeID)), int32(typeID), class != 1
		// A party member's wire TypeID is in the hero band, which is no
		// units.reg class (HERO-APPEAR-041). Its class is the one its visible
		// equipment selects (HERO-APPEAR-042), which the restored member
		// already carries; sound and art look the class up.
		if binding.Party {
			if member, ok := originalPartyMember(ms, binding.ID); ok && member.Class > 0 {
				e.Class = member.Class
			}
		}
		e.Domain, e.TokenSize, e.Facing, e.DesiredFacing = domain, a.TokenSize, a.Facing, a.Facing
		e.TurnRemaining, e.TurnTotal = 0, 0
		e.HP, e.MaxHP, e.Mana, e.MaxMana = int32(int16(basis.Stats[8])), int32(int16(basis.Stats[9])), int32(int16(basis.Stats[11])), int32(int16(basis.Stats[12]))
		if a.CurrentEntity {
			if current, exists := admissionHealth[a.Identity]; exists {
				e.HP, e.MaxHP = current.HP, current.MaxHP
				healthPlan[e.ID] = current
			}
		}
		if a.CurrentEntity && e.HP > 0 && a.Stage > 1 {
			return fmt.Errorf("current actor %d has positive health with terminal decay stage %d", a.ArchiveIndex, a.Stage)
		}
		// A healed current actor overrides stale Stage 1 and ALM decay.
		if e.HP > 0 && (a.Stage == 0 || a.CurrentEntity && a.Stage == 1) {
			e.Decay, e.Dwell = sim.DecayNone, 0
		}
		if a.Dying() && e.HP <= 0 {
			if a.DyingTimer < 0 || e.MaxHP <= 0 {
				return fmt.Errorf("original dying actor %d: unsupported timer %d or maximum HP %d", a.ArchiveIndex, a.DyingTimer, e.MaxHP)
			}
			e.Decay, e.Dwell = sim.DecayFallen, uint16(a.DyingTimer)
		}
		// Current continuation supplies the final lifecycle after the detached
		// wire field/stock import. No death transition or gameplay runs here.
		if a.CurrentEntity && !a.Dying() && e.HP <= 0 {
			e.Decay, e.Dwell = sim.DecayFallen, 0
		}
		e.Reach, e.AttackCharge, e.AttackRelax = basis.Reach, int32(basis.AttackCharge), int32(basis.AttackRelax)
		e.Speed, e.Load, e.Capacity, e.ActorLoad, e.HumanMovement = load.Speed, load.Load, load.Capacity, load.Inventory, load.Movement
		// The high byte of the stored +0xa4 word is the whole-cell scan range;
		// SAV-795 (Medium) reads the low byte zero on every Unit record and the
		// six observed values whole cells of TERR-FOG-081's own scanRange span.
		e.ScanRange = uint8(basis.Sight >> 8)
		e.Reaction, e.Mind, e.Spirit = int32(int16(basis.Stats[1])), int32(int16(basis.Stats[2])), int32(int16(basis.Stats[3]))
		for i := range e.Skill {
			e.Skill[i] = int32(int16(binary.LittleEndian.Uint16(basis.Attack[2+2*i:])))
			e.SkillXP[i] = int32(basis.SkillXP[i])
		}
		batch = append(batch, sim.OriginalLivingActor{Entity: e, New: binding.New})
		if a.CurrentEntity {
			current = append(current, e.ID)
		}
		if classAbsent {
			absentPeople = append(absentPeople, e.ID)
		}
		name := a.Character.Name
		if hiredParty[binding.ID] {
			if member, ok := originalPartyMember(ms, binding.ID); ok {
				name = ordinaryActorName(member, table)
			}
		}
		manifest.Actors = append(manifest.Actors, SnapshotActor{ID: e.ID, Name: name, Constructed: binding.New})
	}
	if len(current) != 0 {
		err = ms.World.ImportCurrentLivingActors(batch, current)
	} else {
		err = ms.World.ImportOriginalLivingActors(batch)
	}
	if err != nil {
		return err
	}
	registry.profiles, registry.health = profiles, healthPlan
	ms.actorRegistry, ms.ActorManifest = registry, manifest
	return rebuildActorPeople(ms, table, absentPeople...)
}

func reconciledRoodTypeID(typeID uint16, flags uint8, mission int, mapUnitID uint16) uint16 {
	if mission == 140 && mapUnitID == 443 && typeID == 0x21 && flags&4 != 0 {
		return uint16(sim.HeroTypeID(true, false))
	}
	return typeID
}

func rebuildActorPeople(ms *Mission, table *mapload.Table, absent ...sim.EntityID) error {
	if ms.ActorManifest == nil {
		return nil
	}
	entities := map[sim.EntityID]sim.Entity{}
	for _, e := range ms.World.Entities() {
		entities[e.ID] = e
	}
	party := map[sim.EntityID]bool{}
	for _, id := range ms.Start.IDs {
		party[id] = true
	}
	for _, a := range ms.ActorManifest.Actors {
		e := entities[a.ID]
		if slices.Contains(absent, a.ID) {
			continue
		}
		if party[e.ID] {
			continue
		}
		if e.SourceBinding.Generated() && !a.Constructed && ms.Map != nil {
			if _, exists := ms.Start.Roster[e.ID]; exists && int(e.ID) < len(ms.Map.Units) && ms.Map.Units[e.ID].UnitID == e.MapUnitID {
				// Native checkpoints keep the exact ALM/NPC identity. Current
				// stats and equipment still come from World when it is carried.
				continue
			}
		}
		if e.SourceBinding.Class == 3 {
			// The uniquely matched ALM person/definition remains the native
			// reconstruction policy. Exact SAV Humanoid has no proven row.
			continue
		}
		if e.SourceBinding.ActorClass() != 2 {
			delete(ms.Start.Roster, e.ID)
			continue
		}
		if !a.Constructed && (table == nil || table.Humans == nil) {
			continue
		}
		member, err := mapload.SourceActorPerson(e, a.Name, table)
		if err != nil {
			return err
		}
		if ms.Start.Roster == nil {
			ms.Start.Roster = map[sim.EntityID]mapload.PartyMember{}
		}
		ms.Start.Roster[e.ID] = member
	}
	return nil
}

func registryTarget(ms *Mission, off int) (*sim.Entity, error) {
	binding, ok := ms.actorRegistry.actor(off)
	if !ok {
		return nil, nil
	}
	if e, ok := ms.World.Entity(binding.ID); ok {
		return &e, nil
	}
	return nil, fmt.Errorf("original actor at %d lost native binding %d", off, binding.ID)
}

// originalPartyMember is the restored party member the mission start bound to
// id.
func originalPartyMember(ms *Mission, id sim.EntityID) (mapload.PartyMember, bool) {
	for i, started := range ms.Start.IDs {
		if started == id && i < len(ms.Party) {
			return ms.Party[i], true
		}
	}
	return mapload.PartyMember{}, false
}
