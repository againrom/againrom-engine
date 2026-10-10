package game

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// materializeCurrentWorld builds the whole SAV document of a mission from the
// captured World, the installed tables and the map. A loaded game and a new
// game take this one path; the loaded document is never its base.
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
	if err := b.reserveCurrentWorld(w); err != nil {
		return nil, err
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		return nil, err
	}
	// Every key the World holds stays reserved, so a minted key never
	// collides with an identity a current object or reference names.
	b.reserveCurrentForm(raw)
	if err := b.reserveWorldEffectKeys(s.SavedDocument, w); err != nil {
		return nil, err
	}
	// joined is the identity that joins each actor to its loaded record. It
	// is read to join, as a cell record's key is; a cell key can name a body
	// another actor stands on. A key the World holds for a structure, the
	// terrain or a dead actor stays theirs. No minted key takes a joined one.
	held := map[uint32]bool{currentTerrainKey(w): true}
	if rows, _, present := w.SavedStructures(); present {
		for _, row := range rows {
			held[row.SourceKey] = true
		}
	}
	for _, d := range w.OriginalDeadActors() {
		held[d.Source.Identity] = true
	}
	joined := map[sim.EntityID]uint32{}
	if l := s.SavedDocument; l != nil && l.Document != nil {
		for _, a := range l.Actors {
			if a.ObjectIndex != 0 && int(a.ObjectIndex) <= len(l.Document.Objects) {
				if key, _ := savedStructureValue(&l.Document.Objects[a.ObjectIndex-1], "Identity"); key != 0 && !held[key] {
					joined[a.EntityID], b.currentKeys[key] = key, true
				}
			}
		}
	}
	terrain := currentTerrainKey(w)
	if terrain == 0 {
		terrain = b.identity()
	}
	b.doc = sav.DocumentData{Version: sav.DocumentDataVersion, FileVersion: sav.MinVersion, Marker: sav.GeneratedCityMarker,
		Head:  sav.DocumentHeadData{Mission: uint32(s.Mission), Difficulty: uint32(s.Difficulty), CounterA: uint32(w.Tick()), MapName: originalMapName(ms.Address), PlayerListField: 1},
		World: &sav.DocumentWorldData{TerrainIdentity: terrain}}
	state := &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion,
		GroupBindings: &SnapshotSAVGroupBindings{Version: 1, PlayersPresent: true}, ActorEffects: &SnapshotSAVActorEffects{Version: 1}, Objects: &SnapshotSAVObjectBindings{Version: 2}}
	state.Document = &b.doc
	if _, _, groups := w.SavedGroups(); groups {
		_, players := w.SavedGroupPlayers()
		state.GroupBindings.PlayersConstructed = !players
	}
	b.state = state
	currentPlayers, haveCurrentPlayers := w.CurrentPlayers()
	// One Player record is written per current Player; two Players may share
	// a Slot. An owner Slot no current Player names gets a constructed root.
	type playerRoot struct{ slot, id uint32 }
	var roots []playerRoot
	covered := map[uint32]bool{}
	if haveCurrentPlayers {
		seen := map[uint32]bool{}
		for _, p := range currentPlayers {
			if seen[p.ID] {
				return nil, fmt.Errorf("current generated Player has ambiguous exact identities")
			}
			seen[p.ID], covered[p.Slot] = true, true
			roots = append(roots, playerRoot{p.Slot, p.ID})
		}
	} else {
		for i := range ms.Map.Groups {
			slot := uint32(i + 1)
			covered[slot] = true
			roots = append(roots, playerRoot{slot: slot})
		}
	}
	party := map[sim.EntityID]mapload.PartyMember{}
	for i, id := range s.CurrentPartyIDs {
		if i < len(s.Party) {
			party[id] = s.Party[i]
		}
	}
	departed, err := currentTerminalBodies(w, ms.Map, t, s.Difficulty, party)
	if err != nil {
		return nil, err
	}
	owners := w.Entities()
	for _, row := range w.CurrentTerminalActors() {
		if body, held := departed[row.ID]; held && !haveCurrentPlayers {
			owners = append(owners, body.entity)
		}
	}
	for _, e := range owners {
		if !covered[e.Owner] {
			covered[e.Owner] = true
			roots = append(roots, playerRoot{slot: e.Owner})
		}
	}
	if !haveCurrentPlayers {
		slices.SortStableFunc(roots, func(a, b playerRoot) int { return cmp.Compare(a.slot, b.slot) })
	}
	// An actor's Player is the current Player containing its Group, else the
	// first Player of its owner Slot.
	containers := map[sim.EntityID]uint32{}
	if groups, _, present := w.SavedGroups(); present {
		for _, g := range groups {
			for _, m := range g.Members {
				if m.Bound && g.ContainerID != 0 {
					containers[m.Entity] = g.ContainerID
				}
			}
		}
	}
	local := currentLocalPlayerRoot(w, s, containers, len(roots), func(i int) (uint32, uint32) { return roots[i].slot, roots[i].id })
	keys, objects := map[uint32]uint32{}, map[uint32]uint16{}
	idKeys, idObjects := map[uint32]uint32{}, map[uint32]uint16{}
	formations, formationsPresent := w.SavedPlayerFormations()
	state.GroupBindings.FormationsPresent = formationsPresent
	for rootIndex, root := range roots {
		slot, currentID := root.slot, root.id
		key := currentPlayerKey(w, slot, currentID)
		if key == 0 || b.claimHeldKey(key) {
			key = b.identity()
		}
		trigger, mode := slot, w.FormationMode(slot)
		if currentID != 0 {
			for _, f := range formations {
				if f.PlayerID == currentID {
					trigger, mode = f.TriggerID, f.Mode
				}
			}
		}
		r := mustNewRecord("Player")
		name, color, participant := "", uint32(1), uint32(1)
		if slot > 0 && int(slot) <= len(ms.Map.Groups) {
			g := ms.Map.Groups[slot-1]
			name, color, participant = g.Name, g.Color+1, g.Participant
		}
		if rootIndex == local {
			name, participant = nativeCityHeroOf(s.Party).Name, 0
		} else if participant == 0 {
			// Exactly one Player is the local human.
			participant = 1
		}
		mustSetText(&r, "Name", name)
		percent, present := w.AutoHealing(slot)
		if !present {
			percent = currentActorOwnerReserve(w, slot)
		}
		for _, v := range []sav.DocumentValueData{{Name: "Slot", Value: slot}, {Name: "SlotAgain", Value: trigger}, {Name: "This", Value: key}, {Name: "F44", Value: color}, {Name: "Participant", Value: participant}, {Name: "F58", Value: percent}, {Name: "Money", Value: w.Purse(slot)}} {
			mustSetValue(&r, v.Name, v.Value)
		}
		if participant == 0 {
			mustSetValue(&r, "F2C", 1<<(slot%16))
			mustSetValue(&r, "F3D", 1)
		}
		tail := make([]byte, sav.PlayerTailLen)
		tail[sav.PlayerTailFormationByte] = mode
		mustSetRaw(&r, "PRaw32", tail)
		r.Inline[0].Record = mustNewDiaryRecord(diaryRows, key)
		index, err := b.append(r)
		if err != nil {
			return nil, err
		}
		if objects[slot] == 0 {
			keys[slot], objects[slot] = key, index
		}
		if currentID != 0 {
			idKeys[currentID], idObjects[currentID] = key, index
		}
		b.doc.Players = append(b.doc.Players, index)
		id := uint32(1)
		for _, p := range state.GroupBindings.Players {
			if p.ID == ^uint32(0) {
				return nil, fmt.Errorf("current Player identity namespace exhausted")
			}
			id = max(id, p.ID+1)
		}
		if currentID != 0 {
			id = currentID
		}
		// A record built for a current Player is that Player's exact
		// container; only a record with no current identity is constructed.
		_, nativePlayers := w.SavedGroupPlayers()
		state.GroupBindings.Players = append(state.GroupBindings.Players, SnapshotSAVGroupPlayerBinding{ID: id, ObjectIndex: index, Constructed: nativePlayers && currentID == 0})
	}
	if err := bindCurrentPlayerRoots(state, w); err != nil {
		return nil, err
	}
	// The player-list dword is one past the list count in every corpus SAV.
	b.doc.Head.PlayerListField = uint32(len(b.doc.Players) + 1)
	bound := map[sim.EntityID]bool{}
	for _, a := range state.Actors {
		bound[a.EntityID] = true
	}
	ownerKey := func(e sim.Entity) (uint32, uint16) {
		if id := containers[e.ID]; idObjects[id] != 0 {
			return idKeys[id], idObjects[id]
		}
		return keys[e.Owner], objects[e.Owner]
	}
	// writeActor writes one actor record from a World body: a live entity or
	// the body a terminal row holds.
	writeActor := func(e sim.Entity, member mapload.PartyMember, retained *sim.OriginalDeadRecord, diary bool) (uint16, uint32, error) {
		var placement *alm.Unit
		for i := range ms.Map.Units {
			if e.MapUnitID != 0 && ms.Map.Units[i].UnitID == e.MapUnitID {
				placement = &ms.Map.Units[i]
				break
			}
		}
		key := joined[e.ID]
		if key == 0 {
			key = heldActorKey(w, e.ID)
		}
		if key == 0 || b.claimHeldKey(key) {
			key = b.identity()
		}
		basis, name, err := currentRecordActor(e, member, nativeCityHeroOf(s.Party), placement, t, key, b.runtime())
		if err != nil {
			return 0, 0, err
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
		playerKey, _ := ownerKey(e)
		mustSetToken(&r, nativeCityToken(source.Identity, playerKey, source.TokenRow, source.TypeID))
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
			return 0, 0, err
		}
		// A party character always has an actor Diary; another actor has one
		// only when the World holds its Diary.
		if r.Class != "Unit" && diary {
			index, err := b.append(mustNewDiaryRecord(diaryRows, 0))
			if err != nil {
				return 0, 0, err
			}
			mustSetRefs(&r, "Diary", []uint16{index})
		}
		if retained != nil {
			if err := b.currentDeadRecord(&r, *retained); err != nil {
				return 0, 0, err
			}
		}
		index, err := b.append(r)
		if err != nil {
			return 0, 0, err
		}
		return index, source.Identity, nil
	}
	dead := w.OriginalDeadActors()
	// A dead actor's identity is its own; a cell record that still names it
	// gives no other actor that key.
	for _, d := range dead {
		if d.Source.Identity != 0 {
			b.claimHeldKey(d.Source.Identity)
		}
	}
	diaries := map[sim.EntityID]bool{}
	for _, d := range w.SavedDiaries() {
		if !d.Owner.Player {
			diaries[d.Owner.Actor] = true
		}
	}
	for _, e := range w.Entities() {
		if bound[e.ID] {
			continue
		}
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
		index, identity, err := writeActor(e, member, retained, isParty || diaries[e.ID])
		if err != nil {
			return nil, err
		}
		state.Actors = append(state.Actors, SnapshotSAVActor{EntityID: e.ID, ObjectIndex: index})
		state.GroupBindings.Members = append(state.GroupBindings.Members, SnapshotSAVGroupMemberBinding{EntityID: e.ID, ObjectIndex: index, Bound: true})
		if _, playerObject := ownerKey(e); member.StartingHero {
			mustSetValue(&b.doc.Objects[playerObject-1], "Hero", identity)
		}
	}
	// A departed actor's record is written from its constructed body at the
	// terminal row's cell, health and stage.
	for _, row := range w.CurrentTerminalActors() {
		body, held := departed[row.ID]
		if !held || bound[row.ID] {
			continue
		}
		// A body the World still holds as an original dead actor keeps that
		// record's identity and dead tuple.
		var retained *sim.OriginalDeadRecord
		for i := range dead {
			if dead[i].ID == row.ID {
				retained = &dead[i]
			}
		}
		member, isParty := party[row.ID]
		index, _, err := writeActor(body.entity, member, retained, isParty || diaries[row.ID])
		if err != nil {
			return nil, err
		}
		state.Actors = append(state.Actors, SnapshotSAVActor{EntityID: row.ID, ObjectIndex: index, Retired: true})
		if err := b.currentTerminalWorn(index, body.worn, w); err != nil {
			return nil, err
		}
	}
	if _, err := b.appendAbsentDeadRecords(w, ms, nativeCityHeroOf(s.Party), s.Difficulty, diaries, diaryRows); err != nil {
		return nil, err
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
	if err := b.currentWorldEffects(s.SavedDocument, w); err != nil {
		return nil, err
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

// currentLocalPlayerRoot is the root of the local human Player: the Player
// holding the starting hero, by its Group container or owner Slot, else the
// first root of the local Slot, else the first root.
func currentLocalPlayerRoot(w *sim.World, s Snapshot, containers map[sim.EntityID]uint32, n int, root func(int) (uint32, uint32)) int {
	first := func(match func(slot, id uint32) bool) int {
		for i := range n {
			if match(root(i)) {
				return i
			}
		}
		return -1
	}
	for i, id := range s.CurrentPartyIDs {
		if i >= len(s.Party) || !s.Party[i].StartingHero {
			continue
		}
		if c := containers[id]; c != 0 {
			if at := first(func(_, rid uint32) bool { return rid == c }); at >= 0 {
				return at
			}
		}
		for _, e := range w.Entities() {
			if e.ID == id {
				if at := first(func(slot, _ uint32) bool { return slot == e.Owner }); at >= 0 {
					return at
				}
			}
		}
	}
	if at := first(func(slot, _ uint32) bool { return slot == sim.SelfSlot }); at >= 0 || n == 0 {
		return at
	}
	return 0
}

// heldActorKey is the key a World cell record holds for its bound actor; it
// is the actor's written identity, so the cell and the actor stay joined.
func heldActorKey(w *sim.World, id sim.EntityID) uint32 {
	var key uint32
	for _, row := range w.SavedCellRecords() {
		for _, slot := range []sim.SavedCellActorSlot{row.Ground, row.Air} {
			if slot.Bound && slot.Entity == id && slot.Key != 0 {
				if key != 0 && key != slot.Key {
					return 0
				}
				key = slot.Key
			}
		}
	}
	return key
}

// currentTerrainKey is the terrain identity the World holds: the key every
// structure position names, else the one every actor motion names. Zero means
// the World holds none and the writer mints one.
func currentTerrainKey(w *sim.World) uint32 {
	var keys []uint32
	if sources, _, present := w.SavedStructures(); present {
		for _, source := range sources {
			keys = append(keys, binary.LittleEndian.Uint32(source.Position[8:]))
		}
	}
	if len(keys) == 0 {
		if motions, _, _, present := w.SavedActorMotions(); present {
			for _, m := range motions {
				keys = append(keys, m.Position.TerrainKey)
			}
		}
	}
	if len(keys) == 0 {
		return 0
	}
	for _, key := range keys {
		if key != keys[0] {
			return 0
		}
	}
	return keys[0]
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
			// An installed Humans row with no parameters is no definition.
			if row >= 0 && row < t.Humans.Len() && len(t.Humans.EntryParams(row)) != 0 {
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
	basis.NativeBasis, basis.SpeedModifier = sim.NativeActorBasis{}, 0
	basis.HumanMovement = sim.HumanMovement{Present: true, RawSpeed: int16(e.SpeedWord()), NativeSpeed: e.Speed, Load: e.Load, Capacity: e.Capacity}
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
	values := map[int]int32{1: e.Reaction, 2: e.Mind, 3: e.Spirit, 4: e.SpeedWord(), 5: int32(e.ActorLoad.OwnWeight), 6: e.Load, 7: e.Capacity, 8: e.HP, 9: e.MaxHP, 10: e.HealthRegenPeriod, 11: e.Mana, 12: e.MaxMana, 13: e.ManaRegenPeriod}
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

// currentPlayerKey is the identity key the World holds for a Player slot: the
// key its Group references name, else the owner key most of its actors' native
// scalars name. Zero means the World holds none and the writer mints one.
func currentPlayerKey(w *sim.World, slot, id uint32) uint32 {
	groups, _, _ := w.SavedGroups()
	if id != 0 {
		// A Group a current Player contains names that Player's key as its
		// owner reference when the owner Slot is the Player's own.
		for _, g := range groups {
			if g.ContainerID == id && g.Owner.Class == 1 && g.Owner.Owner == slot && g.Owner.Key != 0 {
				return g.Owner.Key
			}
		}
	}
	for _, g := range groups {
		for _, ref := range []sim.SavedGroupReference{g.Owner, g.Reference} {
			if ref.Class == 1 && ref.Owner == slot && ref.Key != 0 {
				return ref.Key
			}
		}
	}
	votes := map[uint32]int{}
	for _, e := range w.Entities() {
		if e.Owner == slot && e.NativeBasis.ScalarIsKnown(sim.ScalarReference) {
			if key := e.NativeBasis.Scalars[sim.ScalarReference]; key != 0 {
				votes[key]++
			}
		}
	}
	var key uint32
	for k, n := range votes {
		if n > votes[key] || n == votes[key] && k < key {
			key = k
		}
	}
	return key
}

// claimHeldKey claims a World-held key for one object. It reports true when
// another object already claimed the key, which then cannot be reused.
func (b *generatedDocumentBuilder) claimHeldKey(key uint32) bool {
	if !b.documentKeysReserved {
		b.reserveCurrentDocumentKeys()
	}
	if b.heldKeys == nil {
		b.heldKeys = map[uint32]bool{}
	}
	if b.heldKeys[key] {
		return true
	}
	b.heldKeys[key], b.currentKeys[key] = true, true
	return false
}
