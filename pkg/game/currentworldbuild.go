package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Materialization fills missing representation from the captured World.
func (f *FrontEnd) materializeCurrentWorld(s Snapshot, w *sim.World) (*SnapshotSAVDocument, error) {
	if f.live == nil || f.live.mission == nil || f.live.mission.state == nil {
		return nil, fmt.Errorf("current mission lacks installed descriptors")
	}
	ms, t := f.live.mission.state, f.Table
	diaryRows := 0
	if t != nil && t.Units != nil {
		diaryRows = t.Units.Len()
	}
	b := &generatedDocumentBuilder{nextKey: 0x51000000, table: t}
	state, err := cloneSavedDocument(s.SavedDocument)
	if err != nil {
		return nil, err
	}
	fresh := state == nil || state.Document == nil
	if !fresh {
		b.doc = *state.Document
	}
	if err := b.reserveCurrentWorld(w); err != nil {
		return nil, err
	}
	if fresh {
		terrain := uint32(0)
		if sources, _, present := w.SavedStructures(); present && len(sources) != 0 {
			terrain = binary.LittleEndian.Uint32(sources[0].Position[8:])
			for _, source := range sources {
				if binary.LittleEndian.Uint32(source.Position[8:]) != terrain {
					terrain = 0
					break
				}
			}
			raw, err := w.MarshalBinary()
			if err != nil {
				return nil, err
			}
			b.reserveCurrentForm(raw)
		}
		if terrain == 0 {
			terrain = b.identity()
		}
		b.doc = sav.DocumentData{Version: sav.DocumentDataVersion, FileVersion: sav.MinVersion, Marker: sav.GeneratedCityMarker,
			Head:  sav.DocumentHeadData{Mission: uint32(s.Mission), Difficulty: uint32(s.Difficulty), CounterA: uint32(w.Tick()), MapName: originalMapName(ms.Address), PlayerListField: 1},
			World: &sav.DocumentWorldData{TerrainIdentity: terrain}}
		state = &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion,
			GroupBindings: &SnapshotSAVGroupBindings{Version: 1, PlayersPresent: true}, ActorEffects: &SnapshotSAVActorEffects{Version: 1}, Objects: &SnapshotSAVObjectBindings{Version: 2}}
	} else {
		b.reservedKeys, err = sav.ReserveDocumentKeys(b.doc, 65536)
		if err != nil {
			return nil, err
		}
	}
	terrainMissing := b.doc.World.TerrainIdentity == 0
	if terrainMissing {
		b.doc.World.TerrainIdentity = b.identity()
	}
	state.Document = &b.doc
	if state.GroupBindings == nil {
		state.GroupBindings = &SnapshotSAVGroupBindings{Version: 1, PlayersPresent: true}
	}
	if _, _, groups := w.SavedGroups(); groups {
		_, players := w.SavedGroupPlayers()
		state.GroupBindings.PlayersConstructed = !players
	}
	b.state = state
	currentPlayers, haveCurrentPlayers := w.CurrentPlayers()
	rootIDs := map[uint16]uint32{}
	for _, row := range currentPlayerRoots(state) {
		rootIDs[row.ObjectIndex] = row.ID
	}
	keys, objects := map[uint32]uint32{}, map[uint32]uint16{}
	for _, index := range b.doc.Players {
		if index == 0 {
			continue
		}
		r := &b.doc.Objects[index-1]
		slot, _ := savedStructureValue(r, "Slot")
		key, _ := savedStructureValue(r, "This")
		keys[slot], objects[slot] = key, index
		found := false
		for _, p := range state.GroupBindings.Players {
			found = found || p.ObjectIndex == index
		}
		if !found {
			id := uint32(index)
			if rootIDs[index] != 0 {
				id = rootIDs[index]
			} else if players, present := w.CurrentPlayers(); present {
				for _, p := range players {
					if p.Slot == slot {
						id = p.ID
					}
				}
			}
			state.GroupBindings.Players = append(state.GroupBindings.Players, SnapshotSAVGroupPlayerBinding{ID: id, ObjectIndex: index})
		}
	}
	var slots []uint32
	if haveCurrentPlayers {
		for _, p := range currentPlayers {
			if !slices.Contains(slots, p.Slot) {
				slots = append(slots, p.Slot)
			}
		}
	} else if fresh {
		for i := range ms.Map.Groups {
			slots = append(slots, uint32(i+1))
		}
	}
	for _, e := range w.Entities() {
		if !slices.Contains(slots, e.Owner) {
			slots = append(slots, e.Owner)
		}
	}
	if !haveCurrentPlayers {
		slices.Sort(slots)
	}
	changed := fresh || terrainMissing
	for _, slot := range slots {
		if objects[slot] != 0 {
			continue
		}
		changed = true
		key := b.identity()
		r := mustNewRecord("Player")
		name, color, participant := "", uint32(1), uint32(1)
		if slot > 0 && int(slot) <= len(ms.Map.Groups) {
			g := ms.Map.Groups[slot-1]
			name, color, participant = g.Name, g.Color+1, g.Participant
		}
		if slot == sim.SelfSlot {
			name, participant = nativeCityHeroOf(s.Party).Name, 0
		}
		mustSetText(&r, "Name", name)
		percent, present := w.AutoHealing(slot)
		if !present {
			percent = currentActorOwnerReserve(w, slot)
		}
		for _, v := range []sav.DocumentValueData{{Name: "Slot", Value: slot}, {Name: "SlotAgain", Value: slot}, {Name: "This", Value: key}, {Name: "F44", Value: color}, {Name: "Participant", Value: participant}, {Name: "F58", Value: percent}, {Name: "Money", Value: w.Purse(slot)}} {
			mustSetValue(&r, v.Name, v.Value)
		}
		if participant == 0 {
			mustSetValue(&r, "F2C", 1<<(slot%16))
			mustSetValue(&r, "F3D", 1)
		}
		tail := make([]byte, sav.PlayerTailLen)
		tail[sav.PlayerTailFormationByte] = w.FormationMode(slot)
		mustSetRaw(&r, "PRaw32", tail)
		r.Inline[0].Record = mustNewDiaryRecord(diaryRows, key)
		index, err := b.append(r)
		if err != nil {
			return nil, err
		}
		keys[slot], objects[slot] = key, index
		b.doc.Players = append(b.doc.Players, index)
		// Current IDs bind roots independently of Slot, including Slot zero.
		id := uint32(1)
		currentID := uint32(0)
		if haveCurrentPlayers {
			for _, p := range currentPlayers {
				if p.Slot == slot {
					if currentID != 0 {
						return nil, fmt.Errorf("current generated Player Slot has ambiguous exact identities")
					}
					currentID = p.ID
				}
			}
		}
		for _, p := range state.GroupBindings.Players {
			if p.ID == ^uint32(0) {
				return nil, fmt.Errorf("current Player identity namespace exhausted")
			}
			id = max(id, p.ID+1)
		}
		if currentID != 0 {
			id = currentID
		}
		_, nativePlayers := w.SavedGroupPlayers()
		state.GroupBindings.Players = append(state.GroupBindings.Players, SnapshotSAVGroupPlayerBinding{ID: id, ObjectIndex: index, Constructed: nativePlayers})
	}
	if err := bindCurrentPlayerRoots(state, w); err != nil {
		return nil, err
	}
	// The player-list dword is one past the list count in every corpus SAV.
	b.doc.Head.PlayerListField = uint32(len(b.doc.Players) + 1)
	party := map[sim.EntityID]mapload.PartyMember{}
	bound := map[sim.EntityID]bool{}
	for _, a := range state.Actors {
		bound[a.EntityID] = true
	}
	for i, id := range s.CurrentPartyIDs {
		if i < len(s.Party) {
			party[id] = s.Party[i]
		}
	}
	dead := w.OriginalDeadActors()
	for _, e := range w.Entities() {
		if bound[e.ID] {
			continue
		}
		changed = true
		member, isParty := party[e.ID]
		if !isParty {
			member = s.CurrentRoster[e.ID]
		}
		identity, class := e.SourceBinding.Identity, e.SourceBinding.Class
		var retained *sim.OriginalDeadRecord
		for _, body := range dead {
			if body.ID == e.ID {
				identity, class = body.Source.Identity, body.Source.Class
				retained = &body
				break
			}
		}
		if identity != 0 {
			var existing uint16
			for i, record := range b.doc.Objects {
				key, err := savedStructureValue(&record, "Identity")
				if err != nil || key != identity || record.Class != savedActorClass(class) {
					continue
				}
				if existing != 0 {
					return nil, fmt.Errorf("current actor %d has ambiguous retained identity", e.ID)
				}
				existing = uint16(i + 1)
			}
			if existing != 0 {
				for _, actor := range state.Actors {
					if actor.ObjectIndex == existing {
						return nil, fmt.Errorf("current actor %d shares another actor's retained node", e.ID)
					}
				}
				state.Actors = append(state.Actors, SnapshotSAVActor{EntityID: e.ID, ObjectIndex: existing})
				found := false
				for _, b := range state.GroupBindings.Members {
					found = found || b.ObjectIndex == existing
				}
				if !found {
					state.GroupBindings.Members = append(state.GroupBindings.Members, SnapshotSAVGroupMemberBinding{EntityID: e.ID, ObjectIndex: existing, Bound: true})
				}
				continue
			}
		}
		var placement *alm.Unit
		for i := range ms.Map.Units {
			if e.MapUnitID != 0 && ms.Map.Units[i].UnitID == e.MapUnitID {
				placement = &ms.Map.Units[i]
				break
			}
		}
		basis, name, err := currentRecordActor(e, member, nativeCityHeroOf(s.Party), placement, t, b.identity(), b.runtime())
		if err != nil {
			return nil, err
		}
		if retained != nil {
			basis.SourceBinding.Identity, basis.SourceBinding.Class = retained.Source.Identity, retained.Source.Class
		}
		flags := []string{"HasInventory"}
		if e.Book.WirePresent(e.KnownSpells) {
			flags = append(flags, "HasSpellbook")
		}
		r := mustNewRecord(savedActorClass(basis.SourceBinding.Class), flags...)
		source := basis.SourceBinding
		mustSetToken(&r, nativeCityToken(source.Identity, keys[e.Owner], source.TokenRow, source.TypeID))
		// SAV-1093/SAV-678: ordinary mask 2; observed off-map records 0.
		mask := uint32(2)
		if e.OffMap {
			mask = 0
		}
		mustSetValue(&r, "T18", mask)
		mustSetValue(&r, "RuntimeID", source.RuntimeID)
		mustSetValue(&r, "T08", uint32(e.MapUnitID))
		mustSetRaw(&r, "Block12", constructedPositionBlock(e.X, e.Y, b.doc.World.TerrainIdentity))
		mustSetText(&r, "Name", name)
		for _, v := range []sav.DocumentValueData{{Name: "U4B", Value: uint32(source.Face)}, {Name: "U4C", Value: uint32(source.ClassFlags)}, {Name: "U148", Value: source.DisplayBacking}, {Name: "U49", Value: uint32(e.TokenSize)}, {Name: "U4A", Value: uint32(e.Domain) + 1}, {Name: "Inventory1C", Value: e.ActorLoad.InsertIndex}, {Name: "Inventory20", Value: uint32(e.ActorLoad.Accumulator)}} {
			mustSetValue(&r, v.Name, v.Value)
		}
		r, err = savedActorValueRecord(r, basis, true)
		if err != nil {
			return nil, err
		}
		if r.Class != "Unit" {
			index, err := b.append(mustNewDiaryRecord(diaryRows, 0))
			if err != nil {
				return nil, err
			}
			mustSetRefs(&r, "Diary", []uint16{index})
		}
		if retained != nil {
			if err := b.currentDeadRecord(&r, *retained); err != nil {
				return nil, err
			}
		}
		index, err := b.append(r)
		if err != nil {
			return nil, err
		}
		state.Actors = append(state.Actors, SnapshotSAVActor{EntityID: e.ID, ObjectIndex: index})
		state.GroupBindings.Members = append(state.GroupBindings.Members, SnapshotSAVGroupMemberBinding{EntityID: e.ID, ObjectIndex: index, Bound: true})
		if member.StartingHero {
			mustSetValue(&b.doc.Objects[objects[e.Owner]-1], "Hero", source.Identity)
		}
	}
	if added, err := b.appendAbsentDeadRecords(w, ms); err != nil {
		return nil, err
	} else {
		changed = changed || added
	}
	if !fresh {
		added, err := b.currentStructureRoots(w)
		if err != nil {
			return nil, err
		}
		changed = changed || added
	}
	if !changed && state.PlayerPurses != nil {
		if !w.CurrentPolicy().MotionCarrier {
			if err := b.currentActorCells(w, ms.Map, f.Archives.Containers); err != nil {
				return nil, err
			}
		}
		return state, nil
	}
	slices.SortFunc(state.Actors, func(a, b SnapshotSAVActor) int { return int(a.EntityID) - int(b.EntityID) })
	slices.SortFunc(state.GroupBindings.Members, savedGroupMemberCompare)
	if err := rootDeadActors(state, w); err != nil {
		return nil, err
	}
	if groups, _, present := w.SavedGroups(); present {
		gap, err := projectSavedGroupRoster(state, w, groups)
		if err != nil {
			return nil, err
		}
		if gap != "" {
			return nil, fmt.Errorf("current Group construction: %s", gap)
		}
		projectSavedPlayerCounts(&b.doc)
	} else {
		if err := projectCurrentGroups(state, w); err != nil {
			return nil, err
		}
	}
	if fresh {
		if err := b.currentSpatial(w, ms.Map, t, f.Archives.Containers); err != nil {
			return nil, err
		}
		base := sav.NewCityStateData(nativeCityHeroOf(s.Party).Name)
		b.doc.State = sav.DocumentStateData{RootKind: base.RootKind, DirectoryRecords: base.DirectoryRecords, ValueRecords: base.ValueRecords}
		b.doc.State.DirectoryRecords = append(b.doc.State.DirectoryRecords, sav.CityStateDirectoryData{Path: "/Fog", Kind: 1})
		fog := make([]byte, 4)
		binary.LittleEndian.PutUint32(fog, uint32(ms.Map.Width*ms.Map.Height))
		b.doc.State.ValueRecords = append(b.doc.State.ValueRecords, sav.CityStateRecordData{Path: "/Fog/FirstState", Value: sav.CityStateValueData{Kind: 2}}, sav.CityStateRecordData{Path: "/Fog/Data", Value: sav.CityStateValueData{Kind: 6, Bytes: fog}})
		current := w.SavedProjectiles()
		ids := make([]uint16, len(current.Items))
		for i, p := range current.Items {
			ids[i] = p.ID
		}
		if err := constructProjectileLeaves(&b.doc, current, ids, map[string]bool{}); err != nil {
			return nil, err
		}
		b.doc.Campaign.Scalars[1] = 1
	} else if !w.CurrentPolicy().MotionCarrier {
		if err := b.currentActorCells(w, ms.Map, f.Archives.Containers); err != nil {
			return nil, err
		}
	}
	rows, err := savedPlayerPurseRows(state.Document, state.GroupBindings)
	if err != nil {
		return nil, err
	}
	state.PlayerPurses = &SnapshotSAVPlayerPurses{Version: 1, Players: rows}
	// Roster projection can detach removed actors while filling missing
	// representation. Retire their exact closure before validating reachability.
	if err := retireCurrentActors(state, w); err != nil {
		return nil, err
	}
	indexed, perm, err := sav.ReindexDocumentData(*state.Document)
	if err != nil {
		return nil, err
	}
	state.Document = &indexed
	if err := remapSavedSackDocument(state, perm); err != nil {
		return nil, err
	}
	return state, nil
}

func currentRecordActor(e sim.Entity, member, hero mapload.PartyMember, placement *alm.Unit, t *mapload.Table, key, runtime uint32) (sim.Entity, string, error) {
	if e.SourceBinding.Class != 0 {
		return e, ordinaryActorName(member, t), nil
	}
	var human *data.HumanState
	if e.Humanoid {
		// A native current actor may have no roster portrait at all. SAV still
		// needs the same installed Human constructor row that spawned it.
		if member.FigureFace == 0 && t != nil && t.Humans != nil {
			row := -1
			if placement != nil {
				resolved := mapload.Resolve(*placement, t)
				if resolved.Arm != mapload.ArmUnits && resolved.Found() {
					row = resolved.Index
				}
			} else if member.StartingHero {
				_, n, ok := data.ChargenBase(t.Humans, member.Mage, data.FigureDir(member.FigureDir).Female())
				if ok {
					row = n
				}
			}
			if row < 0 {
				row = data.FindHumanByType(t.Humans, e.TypeID)
				if member.Class > 0 {
					row = data.FindHumanByType(t.Humans, member.Class)
				}
			}
			if row >= 0 && row < t.Humans.Len() {
				def, err := data.NewHumanDef(t.Humans.EntryName(row), t.Humans.EntryParams(row))
				if err != nil {
					return e, "", err
				}
				dir, face := data.FigureFor(def.TypeID, def.Face, def.Gender)
				if placement != nil {
					if d, f, ok := mapload.PlacedFigure(*placement, t); ok {
						dir, face = d, f
					}
				}
				member.Name, member.Class, member.Mage = t.Humans.EntryName(row), def.TypeID, dir.Mage()
				member.Profile, member.FigureDir, member.FigureFace = def.Profile(), string(dir), face
				member.Hero, member.KnownSpells = def.Hero(), def.KnownSpells
			}
		}
		member.Hero = mapload.PotionHero(member.Hero, e.PotionStats)
		unit := nativeCityUnitData(key, 0, member, member, t, [sav.UnitStatWords]uint16{}, [sav.CharacterSkillSlots]uint16{})
		d, hp, mp := mapload.PartyDisplayWithTable(member, t)
		h, err := nativeCityHumanFromDerived(member, t, unit, d, hp, mp)
		if err != nil {
			return e, "", err
		}
		human = &h
	}
	// A companion or hired Human's row is the one the city writer resolves
	// for the same member (nativeCityDefRow); ROM1's reader recognises a
	// persistent companion by that PC_ row.
	humanRow := 0
	if placement == nil && e.Humanoid && (member.CompanionNPC != 0 || member.MercenaryType != 0) {
		humanRow = int(nativeCityDefRow(member, hero, t))
	}
	basis, name, err := mapload.ConstructActorBasis(e, member, placement, t, key, runtime, human, humanRow)
	if err != nil {
		return e, "", err
	}
	// Only the record adapter uses the constructed unknown fields. Native
	// current values and arithmetic policy never change in the live World.
	source := basis.ActorLoad.Source
	source = currentActorSource(e, source)
	binding := basis.SourceBinding
	basis = e
	basis.SourceBinding = binding
	basis.ActorLoad.Source, basis.ActorLoad.Present = source, true
	basis.NativeBasis = sim.NativeActorBasis{}
	basis.HumanMovement = sim.HumanMovement{Present: true, RawSpeed: int16(e.Speed), NativeSpeed: e.Speed, Load: e.Load, Capacity: e.Capacity}
	if member.Hired() {
		name = ordinaryActorName(member, t)
	}
	return basis, name, nil
}

func currentActorSource(e sim.Entity, source sim.SourceActor) sim.SourceActor {
	if e.ActorLoad.Source.Class != 0 {
		return e.SourceNow()
	}
	if source.Class == 0 {
		source.Class = 1
		if e.Humanoid {
			source.Class = 2
		}
	}
	if e.NativeBasis.BodyPresent && e.NativeBasis.BodyKnown {
		source.Stats[0] = e.NativeBasis.Body
	}
	for n := range source.Attack {
		if e.NativeBasis.AttackByteKnown(n) {
			source.Attack[n] = e.NativeBasis.Attack[n]
		}
	}
	for n := range source.Defence {
		if e.NativeBasis.DefenceByteKnown(n) {
			source.Defence[n] = e.NativeBasis.Defence[n]
		}
	}
	for n := range source.Base {
		if e.NativeBasis.BaseByteKnown(n) {
			source.Base[n] = e.NativeBasis.Base[n]
		}
	}
	for n := range source.Modifier {
		if e.NativeBasis.ModifierByteKnown(n) {
			source.Modifier[n] = e.NativeBasis.Modifier[n]
		}
	}
	source.TypeID = uint16(e.TypeID)
	if e.NativeClass.Present {
		source.Fighter = e.NativeClass.Fighter
	}
	values := map[int]int32{1: e.Reaction, 2: e.Mind, 3: e.Spirit, 4: e.Speed, 5: int32(e.ActorLoad.OwnWeight), 6: e.Load, 7: e.Capacity, 8: e.HP, 9: e.MaxHP, 10: e.HealthRegenPeriod, 11: e.Mana, 12: e.MaxMana, 13: e.ManaRegenPeriod}
	for i, v := range values {
		source.Stats[i] = uint16(v)
	}
	if !e.ActorLoad.Present && e.NativeBasis.ScalarIsKnown(sim.ScalarU8E) {
		source.Stats[5] = uint16(e.NativeBasis.Scalars[sim.ScalarU8E])
	}
	if e.NativeBasis.ScalarIsKnown(sim.ScalarUA0) {
		source.ManaFloor = uint16(e.NativeBasis.Scalars[sim.ScalarUA0])
	}
	for i, v := range e.Skill {
		binary.LittleEndian.PutUint16(source.Attack[2+2*i:], uint16(v))
		source.SkillXP[i] = uint32(e.SkillXP[i])
	}
	if source.Class == 2 {
		// Training takes priority over supplied raw words; legacy effective
		// levels fill only bytes whose independent base is unavailable.
		levels := e.Skill
		if e.NativeTraining.Present {
			levels = e.NativeTraining.Levels
		}
		for j := 1; j < data.SkillSlots; j++ {
			at := 2 + 2*j
			word := uint16(levels[j])
			if e.NativeTraining.Present || !e.NativeBasis.BaseByteKnown(at) {
				source.Base[at] = byte(word)
			}
			if e.NativeTraining.Present || !e.NativeBasis.BaseByteKnown(at+1) {
				source.Base[at+1] = byte(word >> 8)
			}
		}
	}
	if source.Class == 2 {
		// HERO-XP-077: actor+0x130 is the wrapping u32 sum of the six
		// per-skill experience counters, maintained at that value from
		// creation, not derived fresh at read time. A Human or Humanoid
		// with no source basis yet (a freshly minted mission actor) has no
		// loaded aggregate to keep, so it is built here from the live
		// SkillXP slots the same way famestate.go's campaign score already
		// sums them (DIV-1267). A record with a source basis returns
		// earlier through SourceNow and keeps its own loaded aggregate.
		var experience uint32
		for _, xp := range e.SkillXP {
			experience += uint32(xp)
		}
		source.Experience = experience
	}
	if e.NativeBasis.ScalarIsKnown(sim.ScalarU130) {
		source.Experience = e.NativeBasis.Scalars[sim.ScalarU130]
	}
	binary.LittleEndian.PutUint16(source.Attack[:], uint16(e.ToHit))
	source.Attack[14], source.Attack[15], source.Attack[16] = uint8(e.DamageBase), uint8(e.DamageSpread), e.XPSlot
	source.Attack[17], source.Attack[18] = e.SecondBase, e.SecondSpread
	source.Attack[19], source.Attack[20], source.Attack[21] = e.SecondaryDamage.Base, e.SecondaryDamage.Spread, 0
	if e.SecondaryDamage.Base != 0 || e.SecondaryDamage.Spread != 0 {
		for i, selector := range data.ElementalSelectorOrder {
			if selector == e.SecondaryDamage.Selector {
				source.Attack[21] = uint8(i + 1)
				break
			}
		}
	}
	binary.LittleEndian.PutUint16(source.Defence[:], uint16(e.Defence))
	binary.LittleEndian.PutUint16(source.Defence[2:], uint16(e.Absorption))
	for i, v := range e.Protection {
		binary.LittleEndian.PutUint16(source.Defence[6+2*i:], uint16(v))
		source.Defence[17+i] = e.Resistance[i]
	}
	source.MoverSpeed, source.Sight = uint8(e.RotationSpeed), uint16(e.ScanRange)<<8
	if source.Class == 2 {
		source.Sight = data.SightWord(e.Mind, e.Reaction, int32(e.ScanRange))
	}
	if e.NativeBasis.ScalarIsKnown(sim.ScalarUA4) {
		source.Sight = uint16(e.ScanRange)<<8 | uint16(e.NativeBasis.Scalars[sim.ScalarUA4]&0xff)
	}
	source.Reach, source.AttackCharge, source.AttackRelax, source.EquipmentRuntimePresent = e.Reach, uint8(e.AttackCharge), uint8(e.AttackRelax), true
	binary.LittleEndian.PutUint16(source.Modifier[10:], uint16(e.HealthRegeneration))
	binary.LittleEndian.PutUint16(source.Modifier[14:], uint16(e.ManaRegeneration))
	return source
}

func projectCurrentGroups(state *SnapshotSAVDocument, w *sim.World) error {
	state.GroupBindings.Groups = nil
	entities := w.Entities()
	for _, p := range state.GroupBindings.Players {
		r := &state.Document.Objects[p.ObjectIndex-1]
		slot, _ := savedStructureValue(r, "Slot")
		key, _ := savedStructureValue(r, "This")
		r.Groups = nil
		var selectors []uint32
		for _, e := range entities {
			if e.Owner == slot && e.Decay < sim.DecayBones && !slices.Contains(selectors, e.Group) {
				selectors = append(selectors, e.Group)
			}
		}
		var count uint32
		for _, selector := range selectors {
			g := sim.SavedGroup{ID: uint32(len(state.GroupBindings.Groups) + 1), Selector: selector, Owner: sim.SavedGroupReference{Class: 1, Owner: slot}}
			order, base, _ := w.FrozenGroupAI(slot, selector)
			g.AI[0x20], g.AI[0x38], g.AI[0x45] = order, base, 1
			var refs []uint16
			for _, e := range entities {
				if e.Owner != slot || e.Group != selector || e.Decay >= sim.DecayBones {
					continue
				}
				for _, a := range state.Actors {
					if a.EntityID == e.ID {
						refs = append(refs, a.ObjectIndex)
						g.Members = append(g.Members, sim.SavedGroupMember{Entity: e.ID, Bound: true})
						break
					}
				}
			}
			group := newSavedGroupRecord()
			if err := projectSavedGroupFields(&group, g, refs, 0, key); err != nil {
				return err
			}
			inline := uint32(len(r.Groups))
			r.Groups = append(r.Groups, group)
			count += uint32(len(refs))
			state.GroupBindings.Groups = append(state.GroupBindings.Groups, SnapshotSAVGroupBinding{ID: g.ID, PlayerObject: p.ObjectIndex, InlineIndex: inline, ContainerID: p.ID, Authored: true, Owner: SnapshotSAVGroupReferenceBinding{Class: 1, Owner: slot, Key: key, ObjectIndex: p.ObjectIndex}})
		}
		mustSetCount(r, "Groups", uint32(len(r.Groups)))
		mustSetCount(r, "Actors", count)
	}
	return projectCurrentPatrols(state, w)
}

// projectCurrentPatrols writes the actor order of every living patroller from
// current state: state 0x0a, its ring in U158_90 in ring order and a cursor
// that is a ring member. Every other actor keeps its written order (DIV-1382).
func projectCurrentPatrols(state *SnapshotSAVDocument, w *sim.World) error {
	entities := w.Entities()
	byID := make(map[sim.EntityID]sim.Entity, len(entities))
	for _, e := range entities {
		byID[e.ID] = e
	}
	for _, a := range state.Actors {
		if a.Retired || a.ObjectIndex == 0 || int(a.ObjectIndex) > len(state.Document.Objects) {
			continue
		}
		order, err := savedActorRaw(&state.Document.Objects[a.ObjectIndex-1], "U158", 148)
		if err != nil {
			continue
		}
		var raw [144]byte
		copy(raw[:], order)
		e, live := byID[a.EntityID]
		if !live {
			continue
		}
		o, patrol := sim.PatrolOrder(e, raw)
		if !patrol {
			continue
		}
		if err := projectNativeOrder(state, w, entities, a.ObjectIndex, o); err != nil {
			return fmt.Errorf("actor %d patrol order: %w", a.EntityID, err)
		}
	}
	return nil
}
