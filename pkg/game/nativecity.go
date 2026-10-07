package game

import (
	"encoding/binary"
	"reflect"
	"slices"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// This file builds an original-compatible city SAV for a campaign that never
// imported one. originalCitySaveState/bindOriginalCity/marshal (originalsave.go)
// is the import-continuity refusal table: it binds once at SAV import time and
// refuses to export whenever a native mutation happened that it does not
// understand. A native campaign has no import baseline to diverge from, so
// that machinery is not used here. Instead every SAVE call rebuilds a
// complete sav.CityData from current live state, validates it through the
// existing sav.CityFromData, then writes final bytes through the existing
// (*CityProvenance).Marshal — the same two calls the roster-editing test in
// pkg/formats/sav/city_test.go already proves valid. Because the roster and
// campaign record are rebuilt fresh every time, hiring, dismissal and the
// mercenary pool arithmetic need no special-case code: they are simply
// whatever f.Carried/f.Town currently hold.
//
// Known Human fields are written together by nativeCityApplyHuman. Current
// attack/defence, maintained class-skill bases, equipment/effect modifiers,
// period words, load, sight and movement rate have separate owners. Original
// LOAD consumes these bytes; our DefRow fallback is not evidence that zero
// values are valid. Unpromoted motion/allocator tails retain explicit defaults.
//
// DIV-890, SAV-607, DIV-884, DIV-903, DIV-904, SAV-614

// nativeCityPlayerIdentity and nativeCityIdentity are placeholder identity
// keys. (*CityProvenance).Marshal reassigns every identity in the document
// before the bytes are final (remintCityIdentities), so these only need to be
// distinct and internally consistent, not globally meaningful.
const nativeCityPlayerIdentity = uint32(0x10)

func nativeCityIdentity(index int) uint32 { return uint32(0x20 + 0x10*index) }

// nativeCityToken builds the placeable-object head shared by every generated
// Human, matching the same known-valid construction
// pkg/formats/sav/city_test.go's cityTestToken already proves
// (SAV-TOKENPOS-074, SAV-OBJ-014).
//
// HERO-SKILLGATE-074
func nativeCityToken(identity, owner uint32, defRow byte, typeWord uint16) []byte {
	b := make([]byte, 37)
	b[4], b[5] = 0x80, 0x80
	binary.LittleEndian.PutUint32(b[12:16], identity>>4)
	b[16] = defRow
	binary.LittleEndian.PutUint16(b[17:19], typeWord)
	binary.LittleEndian.PutUint32(b[29:33], identity)
	binary.LittleEndian.PutUint32(b[33:37], owner)
	return b
}

// nativeCityRosterTypeWord carries the generated player character's class
// and sex (HERO-CLASS-013, nonzero constructor mode). The drawable reads
// these bits from typeID-0x21 (HERO-APPEAR-041/043), independently of DefRow.
// Writing a fighter word for a mage makes ROM1 request heroes_l/mage_st,
// which neither lawful install contains. Hired Humans use their own arm.
func nativeCityRosterTypeWord(member mapload.PartyMember) uint16 {
	word := uint16(0x21)
	if member.Mage {
		word += 2
	}
	if data.FigureDir(member.FigureDir).Female() {
		word++
	}
	return word
}

// SAV-614, SAV-627, DIV-918
func nativeCityHumanTypeWord(member mapload.PartyMember) uint16 {
	return uint16(member.Class)
}

// nativeCityDefRow resolves the installed Humans-table row a generated
// character's reload should read its cosmetic figure/profile/mage flags
// from (DIV-887, amended).
//
// A CHARGEN'D HERO IS NOT ROWLESS IN ROM1. Research settles this: the
// original chargen command constructs a new player character from exactly
// one of four shipped rows selected by class/sex — SESS-HERO-013 and
// SESS-HERO-014 (both High) decode the command byte and its
// "PC_Danath"/"PC_Naira"/"PC_Fergard"/"PC_Reniesta" + archetype-suffix name
// construction; HERO-START-081 (promoted) and the companion cross-checks
// REG-SCN-098 and HERO-JOIN-122/123 (all High) corroborate the same four
// names for a player-built character elsewhere in the same table. This is a
// fact about the ROM1 SAV format, not a claim that Againrom's own
// mapload.PartyMember carries a row — it deliberately does not
// (data.ChargenBase's own doc: "A GENERATED CHARACTER HAS NO ROW OF HIS
// OWN"), because his stats come from his spread and trained slot, not the
// row. What DefRow needs is which row ROM1 itself would have written, and
// that is recoverable without guessing: data.ChargenBase already resolves
// this exact mapping from (mage, female) at chargen time to seed
// Profile/Face/Book/Cells; this function calls the same published function
// again, from the same two facts the live member still carries
// (member.FigureDir, decoded through data.FigureDir.Mage/.Female), and
// reports the row it lands on. restoredDefinition (originalparty.go) then
// derives the SAME Profile/FigureDir/Face/Mage from that row on reload that
// chargen itself derived when the character was made — not a template
// substituted for the player's choice, the player's own choice re-read.
// restoredParty still overwrites Hero.Body/.Reaction/.Mind/.Spirit and every
// other tracked stat from the character's own written Stats regardless of
// what the row supplies, so this only ever affects cosmetics.
//
// A HIRED MERCENARY IS FOUND BY NAME, THE SAME WAY HE WAS HIRED:
// buildMercenarySquad (tavern.go) resolves his one row with
// data.FindHumanByName(table.Humans, name) before it ever builds the
// PartyMember, and member.Name is that identical "NPC%02d_%d" string
// (mercenaryHireTypeFromName, originalparty.go, reads it back the same way
// on load). Re-deriving the row from Class/TypeID instead, as the legacy
// fallback below still does for whatever it does not name here, would reopen
// exactly the imprecision that fallback's own doc admits (many-to-one over
// the Humans collection): a wrong sibling row with the same class art but a
// different Face, HiredRotationSpeed or Book, silently swapped in under a
// name that still reads correctly.
//
// A companion or a CampaignServerID this table cannot place, npc22 aside —
// or a hired mercenary whose own name this table no longer carries — falls
// to the legacy TypeID lookup RotationSpeedBase's own fallback already uses
// elsewhere: tavern.go's own buildMercenarySquad sets Class from a resolved
// Humans row's TypeID (def.TypeID), and mapload.CampaignNPCMember itself
// sets a companion's Class from that same column (d.TypeID) off the row
// FindHumanByServerID already resolved for it at join time — TypeID is
// many-to-one over the Humans collection, so this can land on a row other
// than the one actually hired or joined from. A Class no row declares (0, or
// a template with no TypeID at all) safely falls out of the row<=0 bound
// below the same way it always did.
func nativeCityDefRow(member, hero mapload.PartyMember, table *mapload.Table) byte {
	if table == nil || table.Humans == nil {
		return 0
	}
	if member.StartingHero {
		fig := data.FigureDir(member.FigureDir)
		_, row, ok := data.ChargenBase(table.Humans, fig.Mage(), fig.Female())
		if !ok || row <= 0 || row > 255 {
			return 0
		}
		return byte(row)
	}
	if member.CompanionNPC != 0 {
		heroFemale := data.FigureDir(hero.FigureDir).Female()
		if serverID, ok := table.NPC.CampaignServerID(int32(member.CompanionNPC), 0, hero.Mage, heroFemale); ok {
			if row := data.FindHumanByServerID(table.Humans, serverID); row > 0 && row <= 255 {
				return byte(row)
			}
		}
	}
	if member.MercenaryType != 0 && !nativeCitySiegeMember(member) {
		if row := int(member.DefinitionRow); row > 0 && row < table.Humans.Len() {
			return byte(row)
		}
		if row := data.FindHumanByName(table.Humans, member.Name); row > 0 && row <= 255 {
			return byte(row)
		}
	}
	row := data.FindHumanByType(table.Humans, member.Class)
	if row <= 0 || row > 255 {
		return 0
	}
	return byte(row)
}

// nativeCityHeroOf returns party's own starting hero, or the zero value when
// none is present. nativeCityDefRow needs the hero's own sex to resolve a
// companion's REG-SCN-098 selector; every call site that reaches a companion
// at all has already required a party carrying exactly one hero
// (nativeCityData's own refusal below), so the zero-value fallback here is
// only ever live for a malformed party ExportNativeCitySave refuses before
// this answer would matter.
func nativeCityHeroOf(party []mapload.PartyMember) mapload.PartyMember {
	for _, member := range party {
		if member.StartingHero {
			return member
		}
	}
	return mapload.PartyMember{}
}

func nativeCityRosterMembers(party []mapload.PartyMember) []mapload.PartyMember {
	out := make([]mapload.PartyMember, 0, len(party))
	for _, member := range party {
		if member.MercenaryType != 0 {
			continue
		}
		out = append(out, member)
	}
	return out
}

// nativeCitySiegeMember is Catapult/Ballista (tavern type 1 or 2) — the
// Units-table Unit(0x198) hire buildSiegeSquad (tavern.go) builds, a
// different wire class from the Humans-table Human(0x1e8) hire
// buildMercenarySquad's typ>=3 branch builds (MERC-LEVEL-005).
func nativeCitySiegeMember(member mapload.PartyMember) bool {
	return member.MercenaryType == 1 || member.MercenaryType == 2
}

// nativeCityHumanHiredMembers narrows the live party to the Human-type hired
// members (tavern type 3..15) this writer now binds a real actor record to
// in a second group (SAV-616/617), preserving their own live relative order
// — no rebuild, so no reordering rule is needed for them.
func nativeCityHumanHiredMembers(party []mapload.PartyMember) []mapload.PartyMember {
	out := make([]mapload.PartyMember, 0, len(party))
	for _, member := range party {
		if member.MercenaryType == 0 || nativeCitySiegeMember(member) {
			continue
		}
		out = append(out, member)
	}
	return out
}

// nativeCityUnitData builds one generated Human object at the correct raw
// block lengths for every field city_semantic.go's archive reader requires.
// This allocates the record; nativeCityAttachItems supplies its owned graph
// and nativeCityApplyHuman then fills the known Human fields together. Only
// the remaining unpromoted tails retain zero defaults (DIV-879).
func nativeCityUnitData(identity, owner uint32, member, hero mapload.PartyMember, table *mapload.Table, stats [sav.UnitStatWords]uint16, skillLevels [sav.CharacterSkillSlots]uint16) sav.CityUnitData {
	scalar2 := make([]byte, 55)
	if member.Hired() && !nativeCitySiegeMember(member) {
		binary.LittleEndian.PutUint32(scalar2[47:51], uint32(member.MercenaryType))
	}
	for i, v := range stats {
		binary.LittleEndian.PutUint16(scalar2[2*i:], v)
	}
	rawA6 := make([]byte, 24)
	for i, v := range skillLevels {
		binary.LittleEndian.PutUint16(rawA6[2+2*i:], v)
	}
	typeWord := nativeCityRosterTypeWord(member)
	if member.Hired() {
		typeWord = nativeCityHumanTypeWord(member)
	}
	// SAV-678: every install Human token holds publication mask 2.
	token := nativeCityToken(identity, owner, nativeCityDefRow(member, hero, table), typeWord)
	binary.LittleEndian.PutUint16(token[23:25], 2)
	return sav.CityUnitData{
		Token:      token,
		RawA6:      rawA6,
		RawBE:      make([]byte, 22),
		Raw114:     make([]byte, 24),
		RawD4:      make([]byte, 64),
		Raw154:     make([]byte, 180),
		Raw158:     make([]byte, 148),
		Scalar1:    make([]byte, 19),
		Name:       ordinaryActorName(member, table),
		Scalar2:    scalar2,
		ScalarTail: make([]byte, 17),
		XP:         make([]byte, 24),
	}
}

// nativeCityData builds a complete, structurally valid sav.CityData from
// current live session state. The state store follows sav.NewCityStateData's
// proven exact shape; the campaign section is otherwise left zero-valued
// because applyCityCampaignProjection overwrites every field it establishes
// except scalars[1]/[6] (DIV-888).
//
// SAV-ORIGMIN-397, SAV-599, AI-DIFF-016, SAV-600
func nativeCityData(roster, hired []mapload.PartyMember, table *mapload.Table, difficulty mapload.Difficulty) (sav.CityData, error) {
	return nativeCityDataConstruct(roster, hired, table, difficulty, nativeCityAttachItems, nativeCityApplyHuman)
}

// nativeCityDataApproximate is nativeCityData's DIV-1320/DIV-1321 fallback:
// the same construction, but a member's Human basis is written as the
// current live-derived approximation instead of refusing when no exact
// source continuity survived (nativeCityApplyHumanApproximate).
func nativeCityDataApproximate(roster, hired []mapload.PartyMember, table *mapload.Table, difficulty mapload.Difficulty) (sav.CityData, error) {
	return nativeCityDataConstruct(roster, hired, table, difficulty, nativeCityAttachItems, nativeCityApplyHumanApproximate)
}

func nativeCityDataConstruct(roster, hired []mapload.PartyMember, table *mapload.Table, difficulty mapload.Difficulty,
	attach func([]sav.CityObjectData, *sav.CityUnitData, mapload.PartyMember, *mapload.Table, uint32, *int) ([]sav.CityObjectData, error),
	human func(*sav.CityUnitData, mapload.PartyMember, *mapload.Table) error) (sav.CityData, error) {
	const maxActors = 0x7fff - 1
	if len(roster) == 0 || len(roster) > maxActors || len(hired) > maxActors-len(roster) {
		return sav.CityData{}, originalCityUnsupportedf("native city actor population exceeds archive object space")
	}
	heroIndex := -1
	for i, member := range roster {
		if member.StartingHero {
			if heroIndex >= 0 {
				return sav.CityData{}, originalCityUnsupportedf("native city save has more than one starting hero")
			}
			heroIndex = i
		}
	}
	if heroIndex < 0 {
		return sav.CityData{}, originalCityUnsupportedf("native city save has no starting hero")
	}
	// combined is group 1 then group 2, in that order, walked once through
	// the SAME per-member construction below — sav.newCityProvenance reads a
	// document's actors in group-then-actor order (city.go), so this is
	// exactly the order (*CityProvenance).Roster() and ExportNativeCitySave's
	// own update.Characters must agree on afterwards.
	combined := make([]mapload.PartyMember, 0, len(roster)+len(hired))
	combined = append(combined, roster...)
	combined = append(combined, hired...)

	heroIdentity := nativeCityIdentity(heroIndex)
	fixed, playerSettings := nativeCityPlayerState(heroIdentity)
	if percent, present, err := carriedAutoHealing(combined); err != nil {
		return sav.CityData{}, err
	} else if present {
		binary.LittleEndian.PutUint32(fixed[39:43], percent)
	}

	objects := make([]sav.CityObjectData, 0, len(combined)+1)
	objects = append(objects, sav.CityObjectData{Class: "Player", Player: &sav.CityPlayerData{
		Name: "Player", Fixed: fixed, Raw32: playerSettings,
		Groups: []sav.CityGroupData{{Raw80: make([]byte, 80), F44: nativeCityPlayerIdentity}},
		Diary:  nativeCityDiary(table),
	}})
	actors := make([]uint16, 0, len(combined))
	itemSeq := 0
	for i, member := range combined {
		identity := nativeCityIdentity(i)
		if nativeCitySiegeMember(member) {
			unit, err := nativeCitySiegeUnit(identity, nativeCityPlayerIdentity, member, table)
			if err != nil {
				return sav.CityData{}, err
			}
			binary.LittleEndian.PutUint32(unit.Token[12:16], nextSavedRuntimeID(cityRuntimeIDs(objects)))
			objects = append(objects, sav.CityObjectData{Class: "Unit", Unit: &unit})
			actors = append(actors, uint16(len(objects)))
			continue
		}
		unit := nativeCityUnitData(identity, nativeCityPlayerIdentity, member, combined[heroIndex], table,
			[sav.UnitStatWords]uint16{}, [sav.CharacterSkillSlots]uint16{})
		binary.LittleEndian.PutUint32(unit.Token[12:16], nextSavedRuntimeID(cityRuntimeIDs(objects)))
		if err := nativeCityInitializeHumanMovement(&unit, table); err != nil {
			return sav.CityData{}, err
		}
		var err error
		if objects, err = attach(objects, &unit, member, table, identity, &itemSeq); err != nil {
			return sav.CityData{}, err
		}
		if err := human(&unit, member, table); err != nil {
			return sav.CityData{}, err
		}
		objects = append(objects, sav.CityObjectData{Class: "Human", Unit: &unit})
		actors = append(actors, uint16(len(objects)))
	}
	objects[0].Player.Groups[0].Actors = actors[:len(roster)]
	if len(hired) > 0 {
		// SAV-617: the hire creates one new group under the Player holding
		// the new actors; the Player's own record is unchanged.
		hire := cityHireGroup()
		hire.Actors = actors[len(roster):]
		objects[0].Player.Groups = append(objects[0].Player.Groups, hire)
	}

	head := [13]uint32{}
	switch difficulty {
	case mapload.DifficultyEasy, mapload.DifficultyNormal, mapload.DifficultyHard:
		head[12] = uint32(difficulty)
	default:
		head[12] = uint32(mapload.DifficultyNormal)
	}

	// The player-list dword is one past the list count in every original SAV,
	// the same rule the mission writer applies.
	players := []uint16{1}
	return sav.CityData{
		Version: sav.CityDataVersion, FileVersion: sav.MinVersion,
		MapName: "", Head: head, PlayerList: uint32(len(players) + 1),
		Players: players, Marker: sav.GeneratedCityMarker,
		TrailerState: make([]byte, generatedCityTrailerLen),
		Objects:      objects,
		State:        sav.NewCityStateData(combined[heroIndex].Name),
		// scalars[1] = 1 (DIV-888; see the doc comment above). scalars[6] and
		// every other campaign scalar stay 0.
		Campaign: sav.CityCampaignData{Scalars: [7]uint32{1: 1}},
	}, nil
}

const generatedCityTrailerLen = 400

// nativeCampaignProjection builds the campaign record a native campaign's
// current chapter, offers, mercenary and marker-history state describe,
// reusing the same registry helpers advanceMain itself uses to build a
// chapter's records (progressRecordFromCampaign, candidateSides — both
// pkg/game/campaignprogress.go) rather than re-deriving that mapping.
// Field-by-field justification and the consumption trace behind it are
// recorded in docs/1125/story.md; the short version is: Main's
// Payment/ShopMin/ShopMax/AddHero/EnableMercenary and the mercenary/offer
// arrays are read by real gameplay (Town.ChapterData/Offers) and are
// populated from the registry and t.available/t.taken. Announced mirrors
// t.available (both mean "taken from a building or routed to by a declared
// successor, not yet won" — confirmed against newTownFromCampaignProgress's
// own seeding).
// AutoGetMission is now the scenario's own declared successor, or -1 where it
// declares none (REG-SCN-063); LastMission is still written zero
// because no method in campaignprogress.go reads it (DIV-882). scalars[1]/[6]
// are NOT part of sav.CampaignProjection and applyCityCampaignProjection never
// touches them (DIV-888) — they are set once, in nativeCityData's own
// sav.CityData literal, before this projection is ever applied.
//
// DIV-890, DIV-884, DIV-903
func nativeCampaignProjection(f *FrontEnd, s Snapshot) (sav.CampaignProjection, error) {
	return nativeCampaignProjectionForChapter(f, s, restoreTown(f.Campaign.Value(), s).currentMain())
}

// The first playable world precedes the town-offer boundary. Its active main
// record is the mission being played; all arrays use this same producer.
func nativeCampaignProjectionForChapter(f *FrontEnd, s Snapshot, chapter int) (sav.CampaignProjection, error) {
	fame := fameFromSnapshot(s)
	t := restoreTown(f.Campaign.Value(), s)
	ch := f.Campaign.Value().Chapters[chapter]
	main := progressRecordFromCampaign(f.Campaign.Value(), chapter).savRecord(false)
	if record := t.currentRecords().record(chapter); record != nil {
		main = record.savRecord(false)
	}
	main.AddHero = u16sFromInts(t.pendingNativeHeroGrants(chapter))

	buildingData := ch
	if chapter == t.currentMain() {
		// currentMain retains the terminal main identity for the SAV record,
		// while ChapterData reports that a won terminal chapter has no active
		// town building rows.
		buildingData = t.ChapterData()
	}

	var children []sav.CampaignRecord
	for _, child := range t.currentRecords().children {
		mission := child.mission
		record := child.savRecord(true)
		record.AddHero = u16sFromInts(t.pendingNativeHeroGrants(mission))
		children = append(children, record)
	}

	filterOffered := func(building TownBuilding, list []int) []int {
		var out []int
		for i, mission := range list {
			if t.taken[offerRef{chapter, building, i}] {
				continue
			}
			out = append(out, mission)
		}
		return out
	}
	innMissions := filterOfferedPaired(t, chapter, TownTavern, buildingData.Inn, buildingData.InnNPC)

	out := sav.CampaignProjection{
		Main: main, Children: children,
		// The shelf is the current chapter's own declared list (SAV-CAMPAIGN-083/
		// SAV-606); the permanent unlock array is the session's live
		// accumulated set instead (MERC-SHELF-002) — two different fields,
		// closing DIV-886.
		Mercenaries: u16sFromInts(buildingData.Mercenaries), PermanentMercenaries: mercEnabledU16s(f.Campaign.Value(), t.permanentLoaded(), t.mercEnabled),
		InnNPC: u16sFromInts(innMissions.npc), InnMission: u16sFromInts(innMissions.mission),
		TCMission: u16sFromInts(filterOffered(TownSchool, buildingData.School)), ShopMission: u16sFromInts(filterOffered(TownShop, buildingData.Shop)),
		SelectedMission: uint32(t.selectedMission()),
		// A town SAVE can only ever be taken at home: WorldMapReturn != nil
		// already refuses every away-from-home shape before this function is
		// called (DIV-905).
		FirstMapPoint: true,
		// REG-SCN-063: the scenario's own declared successor, or -1
		// where [Mission<chapter>] declares none.
		AutoGetMission: autoGetMissionValue(f.Campaign.Value(), chapter),
	}
	if len(out.PermanentMercenaries) == 0 {
		merc := t.mercEnabled
		for _, implied := range impliedMercenaryUnlocks(f.Campaign.Value(), chapter) {
			for _, typ := range f.Campaign.Value().Chapters[implied].EnableMercenary {
				if typ > 0 && typ < len(merc) {
					merc[typ] = true
				}
			}
		}
		out.PermanentMercenaries = mercEnabledU16s(f.Campaign.Value(), nil, merc)
	}
	out.MissionTime, out.ScoreEvents, out.ScoreEventsKnown = fame.Time, fame.Events, true
	out.Payload = t.docPayload.Clone()
	for _, d := range t.documents {
		out.Documents = append(out.Documents, sav.CampaignDocument{Value: uint32(d.Value), Kind: uint32(d.Kind)})
	}
	for i := 1; i < 16; i++ {
		// The campaign record working array is authoritative even while a
		// type is hired. Hiring only toggles its flag; mission end writes the
		// live tally back to this cell (MERC-POOL-012, SAV-1085).
		working := t.mercPool[i]
		out.MercenaryWorking = append(out.MercenaryWorking, uint16(working))
		out.MercenaryPristine = append(out.MercenaryPristine, uint16(t.mercCapacity[i]))
		out.MercenaryHired = append(out.MercenaryHired, t.mercHired[i])
	}
	return f.campaignMapObjects(f.currentCampaignMarkers(out, s.WorldSelectedOnce)), nil
}

func (f *FrontEnd) currentCampaignMarkers(campaign sav.CampaignProjection, selected []int) sav.CampaignProjection {
	campaign.Markers = slices.Clone(campaign.Markers)
	for _, mission := range selected {
		if slices.ContainsFunc(campaign.Markers, func(m sav.CampaignMarker) bool { return m.Value == uint32(mission) }) {
			continue
		}
		picture, _ := nativeCityMarkerPicture(f, mission)
		campaign.Markers = append(campaign.Markers, sav.CampaignMarker{Value: uint32(mission), Picture: picture})
	}
	return campaign
}

// mercEnabledU16s converts the tavern's own live per-type unlock set into the
// permanent list PermanentMercenaries carries (DIV-886), in ROM1's append
// order: the loaded list, then campaign declarations by mission, then the rest.
func mercEnabledU16s(c Campaign, loaded []int, enabled [16]bool) []uint16 {
	var out []uint16
	var seen [16]bool
	add := func(typ int) {
		if typ > 0 && typ < len(enabled) && enabled[typ] && !seen[typ] {
			seen[typ] = true
			out = append(out, uint16(typ))
		}
	}
	for _, typ := range loaded {
		add(typ)
	}
	missions := make([]int, 0, len(c.Chapters))
	for mission := range c.Chapters {
		missions = append(missions, mission)
	}
	slices.Sort(missions)
	for _, mission := range missions {
		for _, typ := range c.Chapters[mission].EnableMercenary {
			add(typ)
		}
	}
	for typ := 1; typ < len(enabled); typ++ {
		add(typ)
	}
	return out
}

// impliedMercenaryUnlocks answers exactly the main missions ROM1's own
// campaign data says MUST have been won for a settled town to exist AT ALL,
// regardless of which chapter that town currently sits at — never a guess,
// and never the literal mission numbers a probe happens to show. It walks
// forward from the campaign's own first main mission (camp.Main[0]) along the
// successor chain each [Mission<n>] section declares under its own
// AutoGetMission key (Campaign.AutoAdvance — the same chain
// `againrom.exe -check` reports as "winning mission 10 opens mission 20"),
// which is the campaign's own automatic, no-town-choice prologue: every
// mission on it happens in a fixed order with no building ever offering it
// (Campaign.Offered's own doc). The walk stops at camp.TownBegins() itself
// (the campaign's own first town-offered mission) without adding it: nothing
// but finishing the entire automatic prologue ever reaches a town at all, so
// everything visited strictly before that boundary is implied by the town
// existing, independent of how many town-offered missions have been won
// since.
//
// This is safe to apply at any chapter at or after TownBegins() because
// Town.Arrive (frontend.go) only ever sets the open flag — it never runs
// Town.Won for a pre-town mission, and this engine's Town object does not
// exist to run one earlier than that. So a mission on the automatic prologue
// never has its own EnableMercenary drained by the ordinary Town.Won path
// (town.go) the way a town-offered mission's does, on ANY route that reaches
// a settled town, not only the one that just crossed the boundary.
//
// chapter itself must sit at TownBegins() or later in camp.Main (or be 0,
// meaning every main mission is already won) for the walk to mean anything
// about it: a chapter that is still one of the automatic-prologue missions
// (mid-play, never yet won) implies nothing about its own predecessors
// finishing, and neither does a chapter camp.Main does not carry at all
// (e.g. a side mission) — the caller's own refusal stays live for both.
func impliedMercenaryUnlocks(camp Campaign, chapter int) []int {
	begins, ok := camp.TownBegins()
	if !ok || len(camp.Main) == 0 {
		return nil
	}
	if chapter != 0 {
		reachable := false
		for _, m := range camp.Main {
			if m == chapter {
				reachable = m >= begins
				break
			}
		}
		if !reachable {
			return nil
		}
	}
	var implied []int
	m := camp.Main[0]
	// Bounded by the campaign's own declared main-mission count: the
	// automatic chain visits each [Mission<n>] section at most once, so a
	// walk still going after that many steps is a cycle in the shipped data,
	// not a route to the town boundary — imply nothing rather than loop
	// forever.
	for step := 0; step <= len(camp.Main); step++ {
		if m == begins {
			return implied
		}
		implied = append(implied, m)
		next, hasNext := camp.AutoAdvance(m)
		if !hasNext {
			// The declared chain ends here with no further automatic
			// successor: nothing names begins itself as an AutoGetMission
			// target (the town offers it; nothing auto-advances into it), so
			// this — not walking m up to equal begins — is how stock data
			// marks the boundary. Everything visited so far is implied.
			return implied
		}
		m = next
	}
	return nil
}

// nativeCityMarkerPicture resolves the complete ROM1 resource path. TOWN-041
// constructs it during selection; TOWN-123 and SAV-609 restore the saved path
// when entering the world map. A bare registry name cannot be loaded there.
// It resolves mission's own MapObject picture name
// through the same registry markWorldSelected itself reads
// (worldmap.go) — the exact gate that already decided mission belongs in
// Snapshot.WorldSelectedOnce in the first place, so this call is expected to
// succeed for every mission the caller passes; it can only fail if the
// install's own world-map registry changed underneath a live session.
func nativeCityMarkerPicture(f *FrontEnd, mission int) (string, bool) {
	if f == nil {
		return "", false
	}
	assets := f.worldMapAssets()
	if assets == nil || assets.data == nil {
		return "", false
	}
	object, ok := assets.data.Missions[mission]
	if !ok || object < 0 || object >= len(assets.data.Objects) {
		return "", false
	}
	row := assets.data.Objects[object]
	if !row.Valid || row.Picture == "" || strings.EqualFold(row.Picture, "nothing") {
		return "", false
	}
	return originalCityMarkerPath(row.Picture), true
}

func originalCityMarkerPath(name string) string {
	return `main\graphics\Global.Map\` + name + ".256"
}

// cityCampaignMarkerPaths repairs the bare names written by older Againrom
// saves. Only an exact match with that mission's installed registry entry is
// changed. Qualified and foreign paths, marker fields and the captured state
// are retained. The decoder itself remains a literal format reader.
func (f *FrontEnd) cityCampaignMarkerPaths(campaign sav.CampaignProjection) sav.CampaignProjection {
	copied := false
	for i, marker := range campaign.Markers {
		path, ok := nativeCityMarkerPicture(f, int(marker.Value))
		if !ok || path != originalCityMarkerPath(marker.Picture) {
			continue
		}
		if !copied {
			campaign.Markers = append([]sav.CampaignMarker(nil), campaign.Markers...)
			copied = true
		}
		campaign.Markers[i].Picture = path
	}
	return campaign
}

type pairedOffers struct{ mission, npc []int }

func filterOfferedPaired(t *Town, chapter int, building TownBuilding, mission, npc []int) pairedOffers {
	n := len(mission)
	if len(npc) < n {
		n = len(npc)
	}
	var out pairedOffers
	for i := 0; i < n; i++ {
		if t.taken[offerRef{chapter, building, i}] {
			continue
		}
		out.mission = append(out.mission, mission[i])
		out.npc = append(out.npc, npc[i])
	}
	return out
}

// A pending native grant whose member is absent would change the current
// party during SAV LOAD. Consumed grants are written empty and no longer
// require this refusal, even if the granted member has since left or died.
// The historical function name is retained for the existing refusal seam.
func nativeCityDismissedCompanion(t *Town, party []mapload.PartyMember) int {
	if t == nil {
		return 0
	}
	for _, npc := range t.pendingNativeHeroGrants(t.Chapter()) {
		if npc == 0 {
			continue
		}
		present := false
		for _, member := range party {
			if member.CompanionNPC == npc {
				present = true
				break
			}
		}
		if !present {
			return npc
		}
	}
	return 0
}

// nativeCityEqualUint16s and nativeCityEqualItemInstance(s) compare by
// content, treating a nil slice and an allocated zero-length slice as equal.
// buildMercenarySquad (tavern.go) always allocates Carried via
// make([]uint16, len(carriedItems)) — non-nil even for a type whose row
// carries nothing — while a live PartyMember's own zero value leaves the same
// field nil; reflect.DeepEqual treats the two as different. Measured: a first
// version of nativeCityHiredEquipmentMismatch built on reflect.DeepEqual
// refused every single hire of a type whose row carries nothing, including
// one the player never touched, because "carried item codes" mismatched
// nil-vs-empty on both sides of that one comparison alone.
func nativeCityEqualUint16s(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func nativeCityEqualItemInstance(a, b sim.ItemInstance) bool {
	if a.ObjectID != b.ObjectID || a.Code != b.Code || a.Kind != b.Kind || a.Price != b.Price ||
		a.Weight != b.Weight || a.WeightPresent != b.WeightPresent || a.SourceEquipment != b.SourceEquipment {
		return false
	}
	if len(a.Effects) != len(b.Effects) {
		return false
	}
	for i := range a.Effects {
		if a.Effects[i] != b.Effects[i] {
			return false
		}
	}
	return true
}

func nativeCityEqualItemInstances(a, b []sim.ItemInstance) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !nativeCityEqualItemInstance(a[i], b[i]) {
			return false
		}
	}
	return true
}

func nativeCityEqualWornItems(a, b [sim.EquipSlots]sim.ItemInstance) bool {
	for i := range a {
		if !nativeCityEqualItemInstance(a[i], b[i]) {
			return false
		}
	}
	return true
}

func nativeCityHiredEquipmentMismatch(live, template mapload.PartyMember) string {
	switch {
	case live.Worn != template.Worn:
		return "worn item codes"
	case !nativeCityEqualWornItems(live.WornItems, template.WornItems):
		return "worn item instances"
	case !nativeCityEqualUint16s(live.Carried, template.Carried):
		return "carried item codes"
	case !nativeCityEqualItemInstances(live.CarriedItems, template.CarriedItems):
		return "carried item instances"
	case !reflect.DeepEqual(live.Weapon, template.Weapon):
		return "weapon"
	case live.WeaponMaterialized != template.WeaponMaterialized:
		return "weapon materialization"
	case live.KnownSpells != template.KnownSpells:
		return "known spells"
	case live.Book != template.Book:
		return "spellbook"
	case live.SpellbookRestored != template.SpellbookRestored:
		return "spellbook-restored flag"
	case live.SpellbookPresent != template.SpellbookPresent:
		return "spellbook-present flag"
	case live.Hero != template.Hero:
		return "tracked stats"
	}
	return ""
}

// nativeCitySiegeHireOrderMismatch reports the first siege type (tavern type
// 1 Catapult or 2 Ballista; nativeCitySiegeMember) whose live party position
// restoreHiredMercenaries (nativecityrestore.go) could not reproduce, or 0
// when none is hired or every one already sits where a reload would place
// it. A siege engine is written as its own Unit actor (nativecitysiege.go) and
// read back from it on a LOAD of an original file; a file without that actor
// is rebuilt by restoreHiredMercenaries from the campaign record's hire flag.
// Neither carries where the engine stood in the party (DIV-907's own gap, still
// open).
func nativeCitySiegeHireOrderMismatch(party []mapload.PartyMember) int {
	lastSiegeType := 0
	sawHumanHire := false
	for _, member := range party {
		if nativeCitySiegeMember(member) {
			typ := int(member.MercenaryType)
			if sawHumanHire {
				return typ
			}
			if typ < lastSiegeType {
				return typ
			}
			lastSiegeType = typ
			continue
		}
		if member.MercenaryType != 0 {
			sawHumanHire = true
		}
	}
	return 0
}

func nativeCityRefuseUnbindable(members []mapload.PartyMember, hero mapload.PartyMember, table *mapload.Table) error {
	for _, member := range members {
		if member.ID == "" {
			return originalCityUnsupportedf("native city save contains a party member without a stable ID")
		}
		if nativeCityDefRow(member, hero, table) == 0 {
			return originalCityUnsupportedf("native city save has no Humans definition row to bind %s's figure, profile and mage flag; that identity requires lossless .ags", member.Name)
		}
	}
	return nil
}

// ExportNativeCitySave retains the established API name. Provenance does not
// select the producer or restrict the captured save point.
func (f *FrontEnd) ExportNativeCitySave(s Snapshot, label string) ([]byte, error) {
	return f.ExportCurrentSave(s, label)
}
