package game

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

var errSavingUnavailable = fmt.Errorf("saving is not available for this game yet")

// ExportCurrentSave constructs one document from a captured session. The save
// point determines its shape; provenance supplies only unmodelled residue.
func (f *FrontEnd) ExportCurrentSave(s Snapshot, label string) ([]byte, error) {
	if s.game == "" {
		s.game = base.GameROM1
	}
	if s.game != f.Base().Profile.GameOf() {
		return nil, fmt.Errorf("captured save game differs from installed game")
	}
	if s.game == base.GameROM2 {
		if s.second == nil || s.noticeOpen {
			return nil, errSavingUnavailable
		}
		if s.Mission == 0 {
			if s.second.validateTown() != nil || len(s.World) != 0 || len(s.Party) == 0 || s.Residue.hasMissionState() {
				return nil, errSavingUnavailable
			}
		} else if !secondSaveMission(int(s.Mission)) || s.second.Current != (currentSecondLocation{1, int(s.Mission)}) || s.Residue.MissionLost || s.Residue.FogVisible == nil {
			return nil, errSavingUnavailable
		}
		if err := s.second.validate(); err != nil {
			return nil, err
		}
		if s.Mission != 0 {
			var w sim.World
			if err := w.UnmarshalBinary(s.World); err != nil {
				return nil, err
			}
			if w.Script().Dialect() != sim.ScriptROM2 || w.Outcome() != sim.OutcomeUndecided {
				return nil, errSavingUnavailable
			}
		}
	}
	if s.WorldMapReturn != nil {
		return nil, fmt.Errorf("return to the city before saving")
	}
	if s.Mission == 0 && s.second == nil && f.Campaign.Value().completedBy(restoreTown(f.Campaign.Value(), s)) {
		return nil, errCompletedCampaignSave
	}
	s, err := f.resolveShopSnapshot(s)
	if err != nil {
		return nil, err
	}
	if s.CampaignState {
		fame := fameFromSnapshot(s)
		s.Fame = &fame
		s.Campaign.MissionTime, s.Campaign.ScoreEvents, s.Campaign.ScoreEventsKnown = fame.Time, fame.Events, true
	}
	if vanilla := max(len(s.Documents), len(s.Campaign.Documents)); !sav.DocPayloadFits(s.DocPayload, vanilla) || !sav.DocPayloadFits(s.Campaign.Payload, vanilla) {
		f.Town.dropDocPayload(vanilla)
		s.DocPayload, s.Campaign.Payload = nil, nil
	}
	s.Campaign = f.cityCampaignMarkerPaths(f.currentCampaignMarkers(s.Campaign, s.WorldSelectedOnce))
	if !s.CampaignState || s.Mission != 0 {
		s.Campaign = f.campaignMapObjects(s.Campaign)
	}
	if s.Mission != 0 && f.live != nil && f.live.mission != nil && f.live.mission.state != nil {
		m := f.live.mission.state.Map
		base := sim.Terrain{Cost: mapload.Cost(m), Height: mapload.Height(m)}
		s.terrainBase = &base
	}
	var doc sav.DocumentData
	if s.Mission == 0 {
		doc, err = f.currentCityDocument(s)
	} else {
		doc, err = f.currentMissionDocument(s)
	}
	if err != nil {
		return nil, err
	}
	if err := projectCurrentSession(&doc, s); err != nil {
		return nil, err
	}
	if err := sav.ProjectDocumentPayload(&doc, s.DocPayload); err != nil {
		return nil, err
	}
	doc.Label = []byte(label)
	doc.Head.MapName = originalMapName(doc.Head.MapName)
	repairSavedParticipants(&doc)
	doc, _, err = sav.ReindexDocumentData(doc)
	if err != nil {
		return nil, err
	}
	if err := projectCurrentRuntimeIDs(&doc, s); err != nil {
		return nil, err
	}
	doc, err = sav.CompleteDocumentKeys(doc)
	if err != nil {
		return nil, err
	}
	if s.Mission == 0 {
		finalizeCurrentTownEffectOwners(&doc)
	}
	if err := finalizeCurrentObjectIdentityAbsence(&doc); err != nil {
		return nil, err
	}
	if err := finalizeCurrentOrderSpellAbsence(&doc); err != nil {
		return nil, err
	}
	if err := finalizeCurrentCarrierAbsence(&doc, s.World); err != nil {
		return nil, err
	}
	if err := f.finalizeCurrentPlaneResidue(&doc, s); err != nil {
		return nil, err
	}
	repairEquipmentDefinitionRows(&doc)
	if f.ModSet().Empty() {
		projectOriginalAttack(&doc)
	}
	if err := markDocumentForMods(&doc, f.ModSet(), f.modContext().Items, modMarkLayers(s.Party, f.Table)); err != nil {
		return nil, err
	}
	clampOriginalLevels(&doc)
	return sav.EncodeDocumentData(doc)
}

func (f *FrontEnd) currentMissionDocument(s Snapshot) (sav.DocumentData, error) {
	if s.ghost == nil && f.live != nil && f.live.world != nil {
		ghost := f.live.world.Ghost()
		s.ghost = &ghost
	}
	var w sim.World
	if err := w.UnmarshalBinary(s.World); err != nil {
		return sav.DocumentData{}, err
	}
	// SAV-BLOCK-011/TERR-PASS-053: the caller's own NativeMissionTerrain
	// choice, not whether a prior document exists -- an ordinary mission's
	// own first-ever SAVE also has no prior document, and that save's round
	// trip through this engine still needs currentWorldDocument's baseline
	// per-cell fallback to reload byte for byte. Only NativeMissionTerrain
	// switches this document's own currentSpatial/currentBlockPlaneDelta
	// scope to the sweep-window delta the original writer keeps, and gates
	// the same recompute pass's Capacity/own-weight repair (UNIT-CTOR-004/
	// SAV-792, worldsave.go's own doc comment on currentWorldDocument).
	native := s.NativeMissionTerrain
	var err error
	s.SavedDocument, err = f.materializeCurrentWorld(s, &w)
	if err != nil {
		return sav.DocumentData{}, err
	}
	if s.second != nil {
		s.Campaign = sav.CampaignProjection{Main: sav.CampaignRecord{Mission: uint32(s.Mission)}, SelectedMission: uint32(s.Mission), AutoGetMission: ^uint32(0)}
		s.CampaignState = true
	}
	if !s.CampaignState {
		var err error
		s.Campaign, err = nativeCampaignProjectionForChapter(f, s, f.generatedCampaignChapter(s))
		if err != nil {
			return sav.DocumentData{}, err
		}
		mission := int(s.Mission)
		campaign := f.Campaign.Value()
		if mission != 0 && !containsMission(campaign.Main, mission) &&
			(containsMission(campaign.Side, mission) || containsMission(campaign.Offered, mission)) {
			present := false
			for _, child := range s.Campaign.Children {
				if int(child.Mission) == mission {
					present = true
					break
				}
			}
			if !present {
				child := progressRecordFromCampaign(campaign, mission).savRecord(true)
				child.Age = 0
				s.Campaign.Children = append(s.Campaign.Children, child)
			}
		}
		s.Campaign.SelectedMission, s.CampaignState = uint32(s.Mission), true
		// SAV-1094/REG-SCN-063: every original mission-10 save carries the
		// next mission (20) under AutoGetMission (Campaign+0x110); every
		// other mission, including 20 and every mission from 30 on, carries
		// the original's own no-successor sentinel -1, or the party is stuck
		// on the world map. scenario.reg's own [Mission<n>] section is the
		// current-state source (Campaign.AutoAdvance), not a literal table.
		s.Campaign.AutoGetMission = autoGetMissionValue(campaign, mission)
		// Newly constructed missions start without a town map point.
		s.Campaign.FirstMapPoint = false
	}
	return currentWorldDocument(s, native, f.Table)
}

func (f *FrontEnd) currentCityBase(s Snapshot) (sav.DocumentData, []int, error) {
	return cityBaseDocument(f.Table, s)
}

// cityBaseDocument is the city document skeleton of a snapshot's party: its
// units, groups and application state, over the definition table.
func cityBaseDocument(table *mapload.Table, s Snapshot) (sav.DocumentData, []int, error) {
	if s.Gold < 0 || uint64(s.Gold) > uint64(^uint32(0)) {
		return sav.DocumentData{}, nil, fmt.Errorf("city purse is outside uint32")
	}
	empty := func(objects []sav.CityObjectData, u *sav.CityUnitData, p mapload.PartyMember, t *mapload.Table, owner uint32, seq *int) ([]sav.CityObjectData, error) {
		u.ContainerFlag = 1
		u.Equipment = make([]uint16, 13)
		if p.Book.WirePresent(p.KnownSpells) {
			u.SpellbookFlag, u.SpellbookCount = 1, 1
		}
		return objects, nil
	}
	applyHuman := func(unit *sav.CityUnitData, member mapload.PartyMember, table *mapload.Table) error {
		if err := applyCurrentCityHuman(unit, member, table); err != nil {
			return err
		}
		if _, current := currentCityHumanTails(member); current {
			return nil
		}
		if s.OriginalCity != nil && s.CityObjects != nil {
			if tails, retained := cityHumanTailsByParty(s.CityObjects.HumanTails, member.ID); retained {
				copy(unit.RawA6[22:24], tails[0][:])
				copy(unit.Raw114[22:24], tails[1][:])
				copy(unit.RawD4[40:42], tails[2][:])
			}
		}
		return nil
	}
	city, err := nativeCityDataConstruct(s.Party, nil, table, s.Difficulty, empty, applyHuman)
	if err != nil {
		return sav.DocumentData{}, nil, err
	}
	order := writeCityGroups(&city, s.Party, s.cityGroups)
	city.MapName = s.LastMap
	if err := nativeCityPlayerFormation(&city, s.ApplicationState, nil); err != nil {
		return sav.DocumentData{}, nil, err
	}
	var loaded []sav.CityStateRecordData
	if s.OriginalCity != nil {
		loaded = s.OriginalCity.Document.State.ValueRecords
		if s.OriginalCity.Unavailable != "" {
			loaded = s.OriginalCity.loadedState
		}
	}
	if err := projectCityApplication(&city, s.ApplicationState, loaded); err != nil {
		return sav.DocumentData{}, nil, err
	}
	base, err := sav.CityFromData(city)
	if err != nil {
		return sav.DocumentData{}, nil, err
	}
	doc, err := base.DocumentData()
	if err != nil {
		return sav.DocumentData{}, nil, err
	}
	doc, err = graftSnapshotCityResidue(doc, s)
	if err != nil {
		return sav.DocumentData{}, nil, err
	}
	return doc, order, nil
}

func (f *FrontEnd) currentCityDocument(s Snapshot) (sav.DocumentData, error) {
	doc, order, err := f.currentCityBase(s)
	if err != nil {
		return sav.DocumentData{}, err
	}
	projection, err := newCityObjectProjection(s.CityObjects, s.Party, f.Table)
	if err != nil {
		return sav.DocumentData{}, err
	}
	if err := projection.constructSpellRecords(doc); err != nil {
		return sav.DocumentData{}, err
	}
	keys, err := reserveCurrentCityDocumentKeys(doc, projection.graph, 65536)
	if err != nil {
		return sav.DocumentData{}, err
	}
	b := generatedDocumentBuilder{doc: doc, table: f.Table, reservedKeys: keys, runtimeIDs: savedRuntimeIDs(doc.Objects)}
	a := currentActionData{Version: 1}
	var actors []uint16
	for i, r := range doc.Objects {
		if r.Class == "Human" || r.Class == "Unit" {
			actors = append(actors, uint16(i+1))
		}
	}
	if len(actors) != len(s.Party) {
		return sav.DocumentData{}, fmt.Errorf("city actor construction lost a party member")
	}
	if order != nil {
		// The document lists actors group by group. Bind each back to its
		// party member, which is the order everything below reads.
		byMember := make([]uint16, len(actors))
		for k, member := range order {
			byMember[member] = actors[k]
		}
		actors = byMember
	}
	if err := projection.emit(&b, &a); err != nil {
		return sav.DocumentData{}, err
	}
	for i, index := range actors {
		if err := projection.holdings(&b, index, s.Party[i]); err != nil {
			return sav.DocumentData{}, err
		}
	}
	b.doc.GlobalDWord = cityHireCost(s, f.Table)
	for _, index := range b.doc.Players {
		p := &b.doc.Objects[index-1]
		mustSetValue(p, "Money", uint32(s.Gold))
		mustSetText(p, "Name", nativeCityHeroOf(s.Party).Name)
	}
	if err := projectCurrentCityParty(&b.doc, s.Party, actors, &a, f.Table); err != nil {
		return sav.DocumentData{}, err
	}
	campaign := s.Campaign
	if s.second != nil {
		campaign = sav.CampaignProjection{Main: sav.CampaignRecord{Mission: uint32(s.Mission)}, SelectedMission: uint32(s.Mission), AutoGetMission: ^uint32(0)}
		s.CampaignState = true
	}
	if !s.CampaignState {
		campaign, err = nativeCampaignProjection(f, s)
		if err != nil {
			return sav.DocumentData{}, err
		}
	}
	campaign.FirstMapPoint = true
	b.doc, _, err = sav.ReindexDocumentData(b.doc)
	if err != nil {
		return sav.DocumentData{}, err
	}
	campaign = f.cityCampaignMarkerPaths(campaign)
	if !s.CampaignState {
		campaign = f.campaignMapObjects(campaign)
	}
	doc, err = sav.ProjectDocumentCampaign(b.doc, campaign)
	if err != nil {
		return sav.DocumentData{}, err
	}
	shortcuts, err := quickSpellsToOriginalIndices(s.QuickSpells)
	if err != nil {
		return sav.DocumentData{}, err
	}
	for i := range doc.State.ValueRecords {
		r := &doc.State.ValueRecords[i]
		if r.Path == "/SpellBook/Shortcuts" {
			r.Value.Bytes = make([]byte, 16)
			for j, v := range shortcuts {
				binary.LittleEndian.PutUint32(r.Value.Bytes[j*4:], uint32(v))
			}
		}
	}
	return doc, nil
}

func applyCurrentCityHuman(unit *sav.CityUnitData, p mapload.PartyMember, table *mapload.Table) error {
	if p.Carry != nil && p.Carry.LiveLoad != nil {
		load := p.Carry.LiveLoad
		if err := load.Validate(); err != nil {
			return fmt.Errorf("city Human %q invariant: invalid current load: %w", p.ID, err)
		}
		if load.Inventory.Source.Class == 2 {
			current := mapload.SourceHumanState(load.Inventory.Source, load.Inventory.Accumulator)
			if current.Hero() != p.Hero {
				return fmt.Errorf("city Human %q invariant: Hero disagrees with current actor load", p.ID)
			}
		}
	}
	var h data.HumanState
	var runtime *sav.CityHumanRuntime
	if current, currentRuntime, ok := currentCityHuman(p); ok {
		h, runtime = current, currentRuntime
	} else if current, ok := p.OriginalHumanState(); ok {
		h = current
	} else {
		d, hp, mp := mapload.PartyDisplayWithTable(p, table)
		var err error
		h, err = nativeCityHumanFromDerived(p, table, *unit, d, hp, mp)
		if err != nil {
			return err
		}
	}
	h.HasSpellbook = p.Book.WirePresent(p.KnownSpells)
	if p.Carry != nil {
		for i, xp := range p.Carry.SkillXP {
			h.SkillXP[i] = uint32(xp)
		}
	}
	if p.Carry != nil && p.Carry.LiveLoad != nil {
		load := p.Carry.LiveLoad
		h.Weight, h.Load, h.Capacity = uint16(load.Inventory.OwnWeight), uint16(load.Load), uint16(load.Capacity)
		h.Speed = uint16(load.Speed)
		if load.Movement.Present {
			h.Speed = uint16(load.Movement.RawSpeed)
		}
		if runtime == nil {
			runtime = &sav.CityHumanRuntime{HealthHundredths: load.HealthHundredths, ManaHundredths: load.ManaHundredths}
			d, _, _ := mapload.PartyDisplayWithTable(p, table)
			runtime.Reach, runtime.AttackCharge, runtime.AttackRelax = byte(d.Combat.Reach), byte(d.Combat.AttackChargeTime), byte(d.Combat.AttackRelaxTime)
		}
	}
	if err := nativeCityApplyHumanState(unit, p, h); err != nil {
		return err
	}
	d, _, _ := mapload.PartyDisplayWithTable(p, table)
	unit.Scalar2[34], unit.Scalar2[39], unit.Scalar2[40] = byte(d.Combat.Reach), byte(d.Combat.AttackChargeTime), byte(d.Combat.AttackRelaxTime)
	if runtime != nil {
		unit.Scalar2[28], unit.Scalar2[29] = runtime.HealthHundredths, runtime.ManaHundredths
		unit.Scalar2[34], unit.Scalar2[39], unit.Scalar2[40] = runtime.Reach, runtime.AttackCharge, runtime.AttackRelax
	}
	return nil
}

func (b *generatedDocumentBuilder) currentCityHoldings(index uint16, member mapload.PartyMember, a *currentActionData, locations ...*[]currentPartyHoldingLocation) error {
	projection, err := newCityObjectProjection(nil, []mapload.PartyMember{member}, b.table)
	if err != nil {
		return err
	}
	if err := projection.emit(b, a); err != nil {
		return err
	}
	return projection.holdings(b, index, member, locations...)
}
func currentCityItem(item sim.ItemInstance, t *mapload.Table) sim.ItemInstance {
	kind := item.Kind
	item = constructedGeneratedItem(item, t)
	item.Kind = kind
	item.ObjectID = 0
	return item
}
