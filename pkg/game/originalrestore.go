package game

import (
	"errors"
	"fmt"
	"os"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// originalSource is what an original SAV says, decoded and validated from the
// file's bytes, the install's mod context and its campaign alone. Decoding it
// changes no game state. The mission members are zero for a between-mission
// save.
type originalSource struct {
	saved       []byte
	modLayers   []modMarkLayer
	sf          *sav.File
	quickSpells [4]uint32
	campaign    *originalCampaignDecode

	tails             []sim.CellTail
	hasTails          bool
	cellRecords       []sav.Cell
	hasCellRecords    bool
	spellEffects      []sav.SpellEffect
	hasSpellEffects   bool
	projectiles       sav.ProjectileStore
	hasProjectiles    bool
	diaryChars        []sav.Character
	diaryPlayerRec    *sav.Record
	session           sim.OriginalSession
	hasSession        bool
	ground            []sim.Sack
	hasGround         bool
	groundUnsupported int
	pools             []sav.ActorPools
	profiles          []sav.ActorCurrent
	holdings          []sav.ActorHoldings
	books             []sav.ActorSpellbook
	dead              []sav.DeadActor
	graph             sav.SavedActorGraph
	buildings         []sav.Building
	hasBuildings      bool
	missionDocument   *SnapshotSAVDocument
	documentOrigins   []sav.DocumentObjectOrigin
}

// decodeOriginalSource decodes the complete original SAV in saved over the
// install's mod context and campaign. A mission number of zero is a
// between-mission save.
func decodeOriginalSource(saved []byte, mods mapload.ModContext, campaign Campaign) (*originalSource, error) {
	saved = repairLoadedEquipmentRows(saved)
	saved, modLayers, err := applyModMark(saved, mods)
	if err != nil {
		return nil, err
	}
	sf, err := sav.Open(saved)
	if err != nil {
		return nil, err
	}
	quickSpells, err := originalQuickSpells(sf)
	if err != nil {
		return nil, err
	}
	tails, hasTails, err := originalCellTails(sf)
	if err != nil {
		return nil, err
	}
	cellRecords, hasCellRecords, err := originalCellRecords(sf)
	if err != nil {
		return nil, err
	}
	spellEffects, hasSpellEffects, err := originalSpellEffects(sf)
	if err != nil {
		return nil, err
	}
	projectiles, hasProjectiles, err := originalProjectiles(sf)
	if err != nil {
		return nil, err
	}
	// A malformed book must never remove a character or publish the old
	// template fallback. Validate before either town or mission changes state.
	if _, err := sf.Party(); errors.Is(err, sav.ErrSpellbook) {
		return nil, err
	}
	// A second, independent PartyWalk, on ResumeOriginalSave's own precedent
	// immediately above sf.Party(): a Diary owner cannot be resolved from the
	// persistent-filtered, reordered mapload.PartyMember view alone. Its own
	// walk error is not fatal here, the same tolerance the sf.Party() call
	// immediately above already applies to every non-ErrSpellbook walk failure.
	diaryChars, diaryPlayerRec, _ := sf.PartyWalk()
	decoded, err := decodeOriginalCampaign(sf, saved, campaign, quickSpells)
	if err != nil {
		return nil, err
	}
	n := decoded.mission
	quickSpells = decoded.quickSpells
	session, hasSession, err := originalSessionState(sf)
	if err != nil {
		return nil, err
	}
	src := &originalSource{saved: saved, modLayers: modLayers, sf: sf, quickSpells: quickSpells, campaign: decoded,
		tails: tails, hasTails: hasTails, cellRecords: cellRecords, hasCellRecords: hasCellRecords,
		spellEffects: spellEffects, hasSpellEffects: hasSpellEffects, projectiles: projectiles,
		hasProjectiles: hasProjectiles, diaryChars: diaryChars, diaryPlayerRec: diaryPlayerRec,
		session: session, hasSession: hasSession}
	if n != 0 {
		src.missionDocument, src.documentOrigins = decodeSavedDocument(saved)
		if err := validateOriginalApplicationSource(src.missionDocument.Document); err != nil {
			return nil, err
		}
		src.graph, err = currentActorGraph(sf, src.missionDocument, src.documentOrigins)
		if err != nil {
			return nil, err
		}
		src.ground, src.hasGround, src.groundUnsupported, err = originalGroundState(sf)
		if err != nil {
			return nil, err
		}
		current := currentActorArchives(src.graph)
		src.pools, err = sf.ActorPools(current...)
		if err != nil {
			return nil, err
		}
		src.profiles, err = sf.ActorCurrentProfiles(current...)
		if err != nil {
			return nil, err
		}
		src.holdings, err = sf.ActorHoldings(current...)
		if err != nil {
			return nil, err
		}
		src.books, err = sf.ActorSpellbooks(current...)
		if err != nil {
			return nil, err
		}
		src.dead, err = sf.DeadActors()
		if err != nil {
			return nil, err
		}
		src.buildings, src.hasBuildings, err = sf.Buildings()
		if err != nil {
			return nil, err
		}
	}
	return src, nil
}

// originalInstall is the install's values a restore of an original SAV reads:
// the definitions table and body list, the default starting weapon, the
// archives and the text selector labels are drawn in.
type originalInstall struct {
	table       *mapload.Table
	bodies      data.BodyList
	startWeapon *data.Weapon
	archives    *Archives
	selector    int
}

// originalInstall is this install's values for a restore of an original SAV.
// The selector is read here, when the restore starts.
func (in *InstallResources) originalInstall() originalInstall {
	selector := 0
	if in.Font.Value() != nil {
		selector = in.Font.Value().Selector
	}
	return originalInstall{table: in.Table, bodies: in.Bodies, startWeapon: in.StartWeapon.Value(),
		archives: in.Archives, selector: selector}
}

// restoreOriginalTown builds the between-mission game an original SAV
// describes on this session, which it first clears. A hired-roster refusal
// returns with the error and nothing else changed, because the caller hands in
// a session that is not the running one.
func (s *CampaignSession) restoreOriginalTown(src *originalSource, in originalInstall, campaign Campaign) (RestoredParty, error) {
	sf, saved, modLayers := src.sf, src.saved, src.modLayers
	quickSpells, difficulty, restoredTown := src.quickSpells, src.campaign.difficulty, src.campaign.town
	currentSession, currentActions, currentDocument := src.campaign.session, src.campaign.actions, src.campaign.document
	var cityDocument originalCityDocument
	var cityCharacters []sav.Character
	var cityErr error
	cityDocument, cityErr = sf.CityProvenance()
	var partyErr error
	cityCharacters, partyErr = originalCityCharacters(sf, in.table)
	if cityErr == nil && partyErr != nil {
		cityErr = partyErr
	}
	s.clear(campaign)
	s.quickSpells, s.Difficulty, s.Town = quickSpells, difficulty, restoredTown
	s.fame = fameFromCurrentTown(restoredTown, currentSession)
	restored, report := RestoreParty(sf, s.nextParty(func() []mapload.PartyMember {
		return MissionParty(in.startWeapon, in.bodies, in.table)
	}), in.bodies, in.table)
	// The current action supplement carries the live city policy (including
	// native-only Book modes). Provenance baselines must remain the semantic
	// source reconstruction used by cold LOAD; otherwise a BookLegacy current
	// value is persisted as a source baseline and the next source-free import
	// rejects its own current-state save. Keep this source party before the
	// current action replaces restored below; NameParty stabilizes its IDs.
	semanticBaseline := mapload.CloneParty(restored)
	mapload.NameParty(semanticBaseline)
	clearRestoredMissionPosition(semanticBaseline)
	var cityObjects *cityObjectTopology
	current, currentChars, hasCurrent, err := restoreCurrentCityParty(currentDocument, currentActions, in.table, &cityObjects)
	if err != nil {
		return report, err
	}
	if hasCurrent {
		restored, cityCharacters = current, currentChars
	} else {
		mapload.NameParty(restored)
		clearRestoredMissionPosition(restored)
		if !report.Fallback {
			doc, err := sav.DecodeDocumentData(saved)
			if err != nil {
				return report, err
			}
			cityObjects, err = captureOriginalCityTopology(&doc, restored, cityCharacters)
			if err != nil {
				return report, err
			}
		}
	}
	cityObjects.publishSessionEntry()
	s.Town.cityObjects = cityObjects
	if err := applyModLayers(restored, modLayers, tableModContext(in.table).Items); err != nil {
		return report, err
	}
	s.Carried = restored
	// Bind before rebuilding hires: the overlay mutates this party slice,
	// and the hired roster then copies those overlaid values.
	if cityErr != nil {
		s.originalCity = &originalCitySaveState{unavailable: fmt.Errorf("original-compatible town save provenance: %w", cityErr)}
		if doc, err := sav.DecodeDocumentData(saved); err == nil {
			s.originalCity.loadedState = doc.State.ValueRecords
		}
	} else {
		bindingParty := restored
		if hasCurrent {
			bindingParty = semanticBaseline
		}
		s.originalCity = bindOriginalCity(cityDocument, cityCharacters, bindingParty, nil, in.table)
		s.originalCity.captureSession(s)
	}
	for _, p := range sf.Players {
		if p.Participant == 0 {
			s.Town.gold = int(p.Money)
			break
		}
	}
	s.Town.knowledge = savedPlayerDiary(src.diaryPlayerRec)
	// A current SAV binds its group actors through its document, an
	// original file through its provenance.
	s.Town.cityGroups = nil
	if hasCurrent && cityErr == nil {
		if p, ok := cityDocument.(interface{ Data() sav.CityData }); ok {
			s.Town.cityGroups = cityGroupsFromCurrent(p.Data(), currentActions)
		}
	} else if !hasCurrent {
		s.Town.cityGroups = cityGroupsFromLoaded(s.Carried, s.originalCity.snapshot())
	}
	if err := s.restoreHiredMercenaries(in.table); err != nil {
		return report, fmt.Errorf("original between-mission save: %w", err)
	}
	s.Town.settleCityGroups(s.Carried)
	return report, nil
}

// restoredTownOffer is the number of missions the saved session had offered.
func (src *originalSource) offered() int {
	if src.campaign.session != nil {
		return src.campaign.session.Offered
	}
	return 0
}

// worldMapReturn is the world-map position the saved town was returning from,
// copied, or nil.
func (src *originalSource) worldMapReturn() *SnapshotMapReturn {
	if a := src.campaign.actions; a != nil && a.WorldMapReturn != nil {
		state := *a.WorldMapReturn
		return &state
	}
	return nil
}

// reportTown prints the between-mission restore line for the town the session
// now holds.
func (s *CampaignSession) reportTown(src *originalSource, report RestoredParty) {
	if src.campaign.progress != nil {
		fmt.Fprintf(os.Stderr, "original between-mission save: %s; campaign main %d selected %d restored; money %d\n",
			report, s.Town.Chapter(), s.Town.selectedMission(), s.Town.Gold())
	} else {
		fmt.Fprintf(os.Stderr, "original between-mission save: %s; no campaign record; fresh campaign state; money %d\n",
			report, s.Town.Gold())
	}
}

// originalMissionPlan is the mission half of a restore before the map exists:
// the candidate, the party restored from the file and the saved purse.
type originalMissionPlan struct {
	candidate  *restoreCandidate
	restored   []mapload.PartyMember
	report     RestoredParty
	savedPurse *uint32
}

// planMission builds the candidate of a mission SAV and its restored party.
// units is the candidate's unit bundle. A refusal changes nothing.
func (src *originalSource) planMission(in originalInstall, units *terrain.UnitSet, selectedMarkers map[int]bool) (*originalMissionPlan, error) {
	sf, n := src.sf, src.campaign.mission
	restoredTown, currentSession := src.campaign.town, src.campaign.session
	if _, ok := MissionMap(n); !ok {
		return nil, fmt.Errorf("mission %d: not a campaign mission number", n)
	}
	// Map construction and map-dependent validation can still refuse this
	// load. Keep the active session untouched until the complete candidate is
	// ready, using the same prepare/commit boundary as native Restore. Its
	// fallback party must be fresh, never the previous game's Carried roster.
	candidate := &restoreCandidate{
		quickSpells:       src.quickSpells,
		difficulty:        src.campaign.difficulty,
		town:              restoredTown,
		fame:              fameFromCurrentTown(restoredTown, currentSession),
		units:             units,
		worldSelectedOnce: selectedMarkers,
	}
	fresh := MissionParty(in.startWeapon, in.bodies, in.table)
	if src.graph.CurrentPopulation {
		fresh = nil
	}
	restored, report := RestoreParty(sf, fresh, in.bodies, in.table)
	if err := applyModLayers(restored, src.modLayers, tableModContext(in.table).Items); err != nil {
		return nil, err
	}
	var savedPurse *uint32
	for _, p := range sf.Players {
		if p.Participant == 0 {
			money := p.Money
			savedPurse = &money
			break
		}
	}
	if savedPurse != nil {
		candidate.town.gold = int(*savedPurse)
	}
	if currentSession != nil {
		candidate.offered = currentSession.Offered
		if currentSession.MissionGold != nil {
			candidate.town.gold = *currentSession.MissionGold
		}
	}
	return &originalMissionPlan{candidate: candidate, restored: restored, report: report, savedPurse: savedPurse}, nil
}

// prepareMission is the hook the mission opener runs between decoding the map
// and starting the world. The opener installs the returned fog plane after it.
func (src *originalSource) prepareMission(in originalInstall, plan *originalMissionPlan) func(*alm.Map) (*originalFog, error) {
	sf, n, restored, report := src.sf, src.campaign.mission, plan.restored, plan.report
	graph, dead, missionDocument, documentOrigins := src.graph, src.dead, src.missionDocument, src.documentOrigins
	tails, hasTails, cellRecords, hasCellRecords := src.tails, src.hasTails, src.cellRecords, src.hasCellRecords
	spellEffects, hasSpellEffects, projectiles, hasProjectiles := src.spellEffects, src.hasSpellEffects, src.projectiles, src.hasProjectiles
	buildings, hasBuildings, ground, hasGround, groundUnsupported := src.buildings, src.hasBuildings, src.ground, src.hasGround, src.groundUnsupported
	diaryPlayerRec, diaryChars, session, hasSession := src.diaryPlayerRec, src.diaryChars, src.session, src.hasSession
	holdings, pools, books, profiles := src.holdings, src.pools, src.books, src.profiles
	// The opener installs the returned fog plane after prepare.
	return func(m *alm.Map) (*originalFog, error) {
		r := OriginalSaveResume{
			Party:    report,
			Mission:  n,
			MapName:  sf.Head.MapName,
			HasWorld: hasGround,
			Label:    asciiLabel(sf.Label, in.selector),
			Heads:    len(sf.Actors),
		}
		for _, p := range sf.Players {
			if p.Participant == 0 {
				r.Outcome, r.Money = p.Outcome, p.Money
				break
			}
		}
		for _, a := range sf.Actors {
			if a.Dead() {
				r.Dead++
			}
		}
		// Retain the saved population before joining exact authored placements.
		if !report.Fallback || graph.CurrentPopulation {
			retainSavedActorPlacements(m, graph, dead, missionDocument)
		}
		registry, err := prepareOriginalActorRegistry(m, graph, restored, &r)
		if err != nil {
			return nil, err
		}
		censusOriginalBlocks(m, sf, &r)
		// Report after the opener applies the fog plane.
		report := func(applied bool) {
			r.FogApplied = applied
			fmt.Fprintln(os.Stderr, r)
		}
		g := readOriginalFog(m, sf, &r)
		if g == nil {
			// Optional fog does not suppress actor/equipment restoration.
			g = &originalFog{}
		}
		g.after = func(ms *Mission) error {
			if err := applyOriginalCellTails(ms, tails, hasTails, &r); err != nil {
				return err
			}
			if err := applyOriginalStructures(ms, buildings, hasBuildings, in.table, &r, sf); err != nil {
				return err
			}
			if err := applyOriginalCellRecords(ms, cellRecords, hasCellRecords, registry, &r); err != nil {
				return err
			}
			if err := applyOriginalSpellEffects(ms, spellEffects, hasSpellEffects, &r); err != nil {
				return err
			}
			if err := applyOriginalProjectiles(ms, projectiles, hasProjectiles, &r); err != nil {
				return err
			}
			if err := applyOriginalGround(ms, ground, hasGround, groundUnsupported, in.table, &r); err != nil {
				return err
			}
			if err := admitOriginalActorRegistry(ms, registry, in.table, missionDocument); err != nil {
				return err
			}
			if err := applyOriginalDiaries(ms, diaryPlayerRec, diaryChars, &r); err != nil {
				return err
			}
			if err := applyOriginalSession(ms, session, hasSession, &r); err != nil {
				return err
			}
			if err := restoreOriginalActors(ms, holdings, pools, books, in.table, &r, profiles); err != nil {
				return err
			}
			if err := applyOriginalFacings(ms, sf.Actors, r.Party.SourceOffsets); err != nil {
				return err
			}
			if err := applyOriginalDeadWithOrigins(ms, dead, &r, missionDocument, documentOrigins); err != nil {
				return err
			}
			if err := applyOriginalGroups(ms, &r); err != nil {
				return err
			}
			if err := importSavedDocument(ms, missionDocument, documentOrigins); err != nil {
				return err
			}
			if err := restoreOriginalRandomState(ms); err != nil {
				return err
			}
			if err := importOriginalCellPlanes(ms, in.archives.Containers); err != nil {
				return err
			}
			if err := importSavedSackObjects(ms, ms.savedDocument); err != nil {
				return err
			}
			if err := applyOriginalDying(ms, &r); err != nil {
				return err
			}
			if err := importOriginalActorEffects(ms); err != nil {
				return err
			}
			if err := importOriginalWorldEffects(ms, in.archives.Containers); err != nil {
				return err
			}
			if err := restoreOriginalActions(ms, in.table); err != nil {
				return err
			}
			publishSessionEntry(ms)
			currentGroups, _, _ := ms.World.SavedGroups()
			r.GroupsRestored, r.GroupIssues = len(currentGroups), ms.World.SavedGroupIssues()
			return nil
		}
		g.done = report
		return g, nil
	}
}
