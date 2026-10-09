// SAV import binds stored characters, their current state and the object graph.
// The same format carries original and generated sessions.

package game

import (
	"errors"
	"fmt"
	"os"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/formats/textinput"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type originalCityDocument interface {
	Roster() []sav.CityCharacter
	Marshal(sav.CityUpdate) ([]byte, error)
}

// originalCitySaveState is the session bridge from one imported original
// city to a later SAVE command. document is the detached semantic provenance
// supplied by pkg/formats/sav, never a File, raw .sav, or candidate byte slice.
// bindings join its provenance-local identities to stable PartyMember IDs, so a
// live ordering change cannot update the wrong character.
// SnapshotOriginalCity persists that contract through native city saves.
type originalCitySaveState struct {
	document                       originalCityDocument
	bindings                       []originalCityBinding
	partyImportVersion             uint32
	baselineOffered                int
	baselineDifficulty             mapload.Difficulty
	baselineCampaignMarkerSelected bool
	baselineQuickSpells            [4]uint32
	unavailable                    error
	loadedState                    []sav.CityStateRecordData
	trade                          *originalCityTrade
}

type originalCityBinding struct {
	character    sav.CityCharacter
	partyID      string
	baseline     mapload.PartyMember
	training     []uint8
	sales        []SnapshotCitySale
	salesVersion uint8
	// Copied from the owning city DTO, not a new persisted field. Legacy
	// baselines compare source objects using their absent-weight import policy.
	partyImportVersion uint32
	inventory          []sav.CityInventoryItem
	inventoryErr       error
	spellRules         []sim.SpellRule
	gameRules          sim.Rules
	returned           *SnapshotCityReturn
}

// originalCityUnsupportedError names a city construction or legacy input
// validation failure. It never selects another writer or format.
type originalCityUnsupportedError struct {
	reason string
}

func (err *originalCityUnsupportedError) Error() string {
	return err.reason
}

func originalCityUnsupportedf(format string, args ...any) error {
	return &originalCityUnsupportedError{reason: fmt.Sprintf(format, args...)}
}

func originalCityCharacters(sf *sav.File, table *mapload.Table) ([]sav.Character, error) {
	characters, err := sf.Party()
	persistent := characters[:0]
	for _, character := range characters {
		if persistentOriginalCharacter(character, table) {
			persistent = append(persistent, character)
		}
	}
	persistent, _, _ = leadFirst(persistent)
	return persistent, err
}

func bindOriginalCity(document originalCityDocument, source []sav.Character,
	live []mapload.PartyMember, sourceErr error, tables ...*mapload.Table) *originalCitySaveState {
	var rules []sim.SpellRule
	var gameRules sim.Rules
	if len(tables) != 0 {
		rules, gameRules = mapload.SpellRules(tables[0]), mapload.TableRules(tables[0])
	}
	state := &originalCitySaveState{document: document, partyImportVersion: currentCityPartyImportVersion}
	if sourceErr != nil {
		state.unavailable = fmt.Errorf("original-compatible town save cannot bind the decoded roster: %w", sourceErr)
		return state
	}
	if document == nil {
		state.unavailable = fmt.Errorf("original-compatible town save has no semantic city provenance")
		return state
	}
	if len(source) != len(live) {
		state.unavailable = fmt.Errorf("original-compatible town save decoded %d source characters but restored %d live members", len(source), len(live))
		return state
	}
	descriptors := document.Roster()
	byIdentity := make(map[uint32]sav.CityCharacter, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.Identity == 0 {
			state.unavailable = fmt.Errorf("original-compatible town save has a zero character identity")
			return state
		}
		if _, duplicate := byIdentity[descriptor.Identity]; duplicate {
			state.unavailable = fmt.Errorf("original-compatible town save repeats character identity %#08x", descriptor.Identity)
			return state
		}
		byIdentity[descriptor.Identity] = descriptor
	}
	bySourceIdentity := make(map[uint32]originalCityBinding, len(source))
	usedPartyIDs := make(map[string]bool, len(live))
	for i, character := range source {
		member := live[i]
		if member.ID == "" {
			state.unavailable = fmt.Errorf("original-compatible town save restored source identity %#08x without a stable party ID", character.Key)
			return state
		}
		if usedPartyIDs[member.ID] {
			state.unavailable = fmt.Errorf("original-compatible town save restored duplicate party ID %q", member.ID)
			return state
		}
		usedPartyIDs[member.ID] = true
		descriptor, ok := byIdentity[character.Key]
		if !ok {
			state.unavailable = fmt.Errorf("original-compatible town save source identity %#08x has no semantic descriptor", character.Key)
			return state
		}
		if descriptor.Hero != member.StartingHero || descriptor.DefRow != character.DefRow {
			state.unavailable = fmt.Errorf("original-compatible town save identity %#08x changed hero or definition identity during restore", character.Key)
			return state
		}
		if _, duplicate := bySourceIdentity[character.Key]; duplicate {
			state.unavailable = fmt.Errorf("original-compatible town save repeats restored source identity %#08x", character.Key)
			return state
		}
		bindCityHuman(document, descriptor, &member)
		member = mapload.CloneParty([]mapload.PartyMember{member})[0]
		live[i] = member
		binding := originalCityBinding{
			character:          descriptor,
			partyID:            member.ID,
			baseline:           mapload.CloneParty([]mapload.PartyMember{member})[0],
			spellRules:         rules,
			gameRules:          gameRules,
			salesVersion:       1,
			partyImportVersion: currentCityPartyImportVersion,
		}
		bindCityInventory(document, &binding)
		bySourceIdentity[character.Key] = binding
	}
	if len(bySourceIdentity) != len(descriptors) {
		state.unavailable = fmt.Errorf("original-compatible town save semantic roster has %d characters but only %d were faithfully restored", len(descriptors), len(bySourceIdentity))
		return state
	}
	state.bindings = make([]originalCityBinding, 0, len(descriptors))
	for _, descriptor := range descriptors {
		binding, ok := bySourceIdentity[descriptor.Identity]
		if !ok {
			state.unavailable = fmt.Errorf("original-compatible town save semantic identity %#08x was not restored", descriptor.Identity)
			return state
		}
		state.bindings = append(state.bindings, binding)
	}
	return state
}

func originalCityBaselineUpdate(character sav.CityCharacter) sav.CityCharacterUpdate {
	return sav.CityCharacterUpdate{
		Identity:    character.Identity,
		Name:        character.Name,
		Stats:       character.Stats,
		SkillLevels: character.SkillLevels,
		SkillXP:     character.SkillXP,
		Experience:  character.Experience,
	}
}

// captureSession records the imported town's native-only presentation state.
// Unmapped fields retain their baseline for the lossless native fallback.
// Shortcut provenance remains independently validated on native LOAD, while
// CityUpdate now writes the current bindings.
func (state *originalCitySaveState) captureSession(front *CampaignSession) {
	if state == nil || front == nil {
		return
	}
	state.baselineOffered = front.Offered
	state.baselineQuickSpells = front.quickSpells
	state.baselineDifficulty = front.Difficulty
	if front.Town != nil && front.Town.progress != nil {
		state.baselineCampaignMarkerSelected = front.Town.progress.markerSelected
	}
}

func (state *originalCitySaveState) marshal(f *FrontEnd, snapshot Snapshot, label string) ([]byte, error) {
	return f.ExportCurrentSave(snapshot, label)
}

// OriginalSaveResume is what a resumed mission carried across from the save, and — as much
// as it is — WHAT IT DID NOT.
//
// It exists because a resume from this tree is a resumed PARTY inside a
// fresh mission (0147), plus the explored map (0150). The player's own
// characters come from the file: who they are, where they stand, their four
// statistics, both pool pairs, what they wear and what they carry. THE
// EXPLORED MAP COMES FROM A DIFFERENT REGION OF THE SAME FILE — the
// uncompressed tail's state store, not the compressed body's document
// (SAV-FOG-061) — which is why it arrives while the rest of the mission
// does not. Campaign progress is restored independently from the typed
// campaign tail. Ground sacks use the exact document projection. Non-party
// pools and books use the all-Player actor walk and a unique authored-map
// join after rearming (1094, 1105). The block-plane delta, unmatched world
// objects still restart.
//
// So this report is the divergence, counted. A resume that printed nothing would
// be a resume whose divergence was silent, and silent is the failure — and a
// resume that printed the WRONG units is the same failure wearing a number: this
// report said "MOVED 27" while the party it opened with was a fresh one.
type OriginalSaveResume struct {
	// Party is what the walk restored, counted. It is the axis a player checks
	// first, so it is the first field.
	Party RestoredParty

	// Mission and MapName are what the save named. MapName is the save's own
	// field, not the address the map was opened from, so a disagreement
	// between the two is visible.
	Mission int
	MapName string

	// Label is the save's slot text and Outcome its mission-outcome latch.
	Label   string
	Outcome uint8

	// Money is the human participant's purse, or zero if the save named no
	// human participant.
	Money uint32

	// HasWorld reports that the save carried a world half at all.
	HasWorld bool

	// Heads is how many placeable-object heads were located, Dead how many
	// of them carry a runtime id of 0, and Joined how many LIVING ones
	// matched a unit id the map carries.
	Heads, Dead, Joined int
	GroupsRestored      int
	GroupIssues         []string
	// Reowned counts joined placements whose saved Player slot differs.
	Reowned int
	// Stocked counts exact non-party holdings restored across all Players.
	Stocked                                                          int
	StockParty, StockDead, StockOffMap, StockUnbound, StockUnmatched int
	// Pools counts the exact all-Player pool projection separately from the
	// heuristic position-head census. Excluded covers dead/off-map/dynamic
	// records; unmatched records name no current map actor.
	PoolsRestored, PoolsParty, PoolsExcluded, PoolsUnmatched             int
	ProfilesRestored, ProfilesParty, ProfilesExcluded, ProfilesUnmatched int
	Books                                                                originalBookCounts
	CorpsesRestored, TerminalRestored                                    int
	DyingRestored                                                        int // stage-1 actors retained in Player lists, outside the late-dead manager.
	// UnboundRestored counts dead actors carried with no authored map unit at
	// all (MapUnitID 0, SAV-DEADLOAD-125): never a live entity, held forever
	// at the imported tuple. Disjoint from CorpsesRestored/TerminalRestored.
	UnboundRestored int
	// Structures counts restored source records and absent authored placements.
	Structures originalStructureCounts

	// Moved is how many of the joined heads stood somewhere other than where
	// the map places them.
	Moved int

	// BlockRecords is the size of the block-plane delta, BlockNovel how many
	// of its cells the map's OWN derived plane leaves open, and BlockOccupied
	// how many carry an occupancy bit in the runtime byte.
	//
	// These are not applied (L-3). BlockNovel is the number that says how far
	// a resumed world's terrain stands from the saved one, and it is reported
	// in the encoding-independent form — "the save blocks a cell the map does
	// not" — because the save's block bits and this tree's derived plane are
	// two different encodings of blocking and a bitwise comparison of them
	// would be a claim nobody has established.
	BlockRecords, BlockNovel, BlockOccupied int
	// CellTriggerRecords counts saved records, including repeated keys. Only
	// their six trigger bytes are restored, not the other cell payload fields.
	CellTriggerRecords  int
	CellTriggersApplied bool

	// CellRecords counts saved cell-record rows, including repeated keys: the
	// complete residue this build has no other typed carrier for — layer
	// count, both residue spans, the Sack/SpellEffect identity keys and the two
	// movement-domain actor keys with their SAV-CELLLOAD-110 rebind.
	// BaselineCost/BaselineStatic and the Building key are counted by
	// Structures; the trigger tail by CellTriggerRecords.
	CellRecords        int
	CellRecordsApplied bool

	// SpellEffects counts top-level SpellEffect-list entries: the graph carried
	// by applyOriginalSpellEffects, typed children and object identity
	// preserved. Known AreaEffect consumers are bound separately.
	SpellEffects        int
	SpellEffectsApplied bool

	// Projectiles counts distinct restored Prj<id> sections: the manager's
	// allocator word and each one's sixteen leaves, carried opaquely — this
	// build has no live projectile registry, so a restored section is never
	// bound to a live object and never simulated (docs/DIVERGENCES.md).
	Projectiles        int
	ProjectilesApplied bool

	// Diaries counts restored Diary records: the Player's own inline Diary plus
	// every living Humanoid's own referenced Diary that resolved to an admitted
	// actor, each carried as typed sparse entries (SAV-667, SAV-668) bound to
	// its owner (sim.SavedDiaryOwner) rather than left as
	// JournalChars/JournalEntries' pre-existing bare count. What an entry's
	// Count/Remaining pair means is Unknown, so nothing here acts on it beyond
	// carrying it (docs/DIVERGENCES.md).
	Diaries        int
	DiariesApplied bool

	// Latches is how many of the session block's thousand fire-once trigger
	// latches were set when the save was taken. DiplomacyCells is the complete
	// matrix population carried with them. SessionApplied says both populations
	// reached the canonical world; it is false before a world exists and for a
	// save that has no world half.
	Latches             int
	DiplomacyCells      int
	SessionApplied      bool
	WinCount, LoseCount uint32
	Sacks, GroundItems  int
	GroundApplied       bool

	UnsupportedItemEffects int

	// FogCells is how many cells the save's explored-terrain record covers and
	// FogSet how many of them it records as explored. FogApplied says the plane
	// reached the running map screen; it is false on a path that builds no map
	// screen at all, and on a save whose cell count disagrees with the map's.
	//
	// FogCells is the record's own total and not the map's, so the two being
	// unequal is visible rather than folded away.
	FogCells, FogSet int
	FogApplied       bool
}

// originalFog is a decoded explored-terrain record sized to the map it was
// read against, carried from the prepare arm to the statement that can apply
// it. The after hook is the same post-start/pre-open carrier for state that
// needs a built Mission: original session continuity and saved actor stock.
//
// done is called by that statement with the answer, which is what lets the
// counted report say whether the plane reached the screen rather than whether
// it was decoded. It is nil on a path that has no report to print.
type originalFog struct {
	cols, rows int
	cells      []byte
	after      func(*Mission) error
	done       func(applied bool)
}

// originalSessionState decodes and validates the complete supported session
// handoff before any game-level mutation. A between-mission save has no world
// half and therefore no session; that absence is not an error.
func originalSessionState(f *sav.File) (sim.OriginalSession, bool, error) {
	if f == nil || f.World == nil {
		return sim.OriginalSession{}, false, nil
	}
	state, err := f.SessionState()
	if err != nil {
		return sim.OriginalSession{}, false, err
	}
	out := sim.OriginalSession{
		HasClock:       true,
		Clock:          sim.SessionClock{SubTick: state.SubTick, FullTick: state.FullTick},
		Registers:      state.Registers,
		TriggerLatches: state.TriggerLatches,
		RawHead:        state.RawHead,
		RawMid:         state.RawMid,
		Diplomacy:      state.Diplomacy,
		Won:            state.Won,
		Lost:           state.Lost,
	}
	participants := 0
	for _, p := range f.Players {
		if p.Participant == 0 {
			participants++
			out.Outcome = sim.Outcome(p.Outcome)
		}
	}
	if participants != 1 {
		return sim.OriginalSession{}, false, fmt.Errorf("original save session: %d human participants, want exactly one", participants)
	}
	if err := sim.ValidateOriginalSession(out); err != nil {
		return sim.OriginalSession{}, false, err
	}
	return out, true, nil
}

// applyOriginalSession is the only game-to-sim mutation seam for this state.
// It runs after StartMissionFrom compiled the map script and before openMission
// constructs NewAnnouncer. ImportOriginalSession repeats complete validation
// and commits both populations together.
func applyOriginalSession(ms *Mission, state sim.OriginalSession, present bool,
	r *OriginalSaveResume) error {
	if !present {
		return nil
	}
	if ms == nil || ms.World == nil {
		return fmt.Errorf("original save session: mission has no world")
	}
	if err := ms.World.ImportOriginalSession(state); err != nil {
		return err
	}
	if r != nil {
		r.DiplomacyCells = len(state.Diplomacy)
		r.SessionApplied = true
		r.WinCount, r.LoseCount = state.Won, state.Lost
	}
	return nil
}

// exportOriginalMissionSession writes a world's hundred live trigger-result
// registers and its two carried raw session spans into a decoded original
// save's own world half, at the offsets SAV-SESS-031 gives them
// (docs/1130/story.md). It is the native-writer half of this story: the
// counterpart read is SessionState, and the counterpart mutation seam is
// applyOriginalSession above it.
//
// IT NEVER TOUCHES THE BETWEEN-MISSION FORM. A city save has no world half at
// all (SAV-SESS-031), so f.World == nil there and this refuses rather than
// silently doing nothing — a caller that reaches this function for a town
// save has already mismatched a save to a save kind.
func exportOriginalMissionSession(f *sav.File, w *sim.World) error {
	if f == nil || f.World == nil {
		return fmt.Errorf("original mission session export: this save has no world session")
	}
	if w == nil {
		return fmt.Errorf("original mission session export: nil world")
	}
	regs := w.ScriptRegisters()
	for i, v := range regs {
		if err := f.SetTriggerResult(i, v); err != nil {
			return err
		}
	}
	if err := f.SetRawHead(w.RawSessionHead()); err != nil {
		return err
	}
	return f.SetRawMid(w.RawSessionMid())
}

// readOriginalFog decodes the save's explored-terrain record against a
// decoded map, and reports nil where the save carries none.
//
// THE CELL COUNT IS CHECKED AGAINST THE MAP HERE, not at the apply. The record
// carries a count and no width — the linear order idx = col + row*width is only
// meaningful against the map's own width — so a record whose count is not
// width*height is a record for another map and is not carried. That check is
// this function's whole reason for taking a map at all.
//
// A DECODE ERROR IS SWALLOWED INTO "NO RECORD". This runs inside a mission
// opener whose failure is a mission that does not open, and a malformed Fog
// section is not a reason to refuse a save whose party, positions and items
// read perfectly. The counted report is where its absence shows.
func readOriginalFog(m *alm.Map, f *sav.File, r *OriginalSaveResume) *originalFog {
	g, ok, err := f.Fog()
	if err != nil || !ok {
		return nil
	}
	return fogForMap(m, g, r)
}

// fogForMap sizes a decoded record against a decoded map, and is where the
// count check lives. It is separate from readOriginalFog so that the sizing
// rule can be exercised over a hand-built record: the decode itself is
// pkg/formats/sav's to witness, and this is the step that turns a count into a
// plane for a particular map.
func fogForMap(m *alm.Map, g *sav.Fog, r *OriginalSaveResume) *originalFog {
	r.FogCells, r.FogSet = len(g.Cells), g.Set
	cols, rows := int(m.Width), int(m.Height)
	if cols <= 0 || rows <= 0 || cols*rows != len(g.Cells) {
		return nil
	}
	return &originalFog{cols: cols, rows: rows, cells: g.Cells}
}

// originalOccupancyBits are bits 6 and 7 of a block record's runtime byte.
const originalOccupancyBits = 0xc0

// ResumeOriginalSave starts a campaign mission from a save the ORIGINAL GAME wrote.
//
// It reads the save, opens the map the save's mission number names through this
// tree's own mission-to-map table, applies each matched actor's saved cell and
// fine position TO THAT MAP'S UNIT RECORD, and then starts the mission through
// exactly the path a fresh start uses. Nothing downstream of the map decode
// knows a save was involved: passability, placement, the script compile and the
// party drop all run as they always do.
//
// The transfer is one assignment per axis because the save's head and the map's
// unit record are the SAME FIXED POINT — cell in the high byte, fine position in
// the low — so there is no arithmetic between them to get wrong.
//
// THE PARTY IS THE SAVE'S OWN, and the party argument is what a save whose
// walk reads no character falls back to. It was the party outright until
// this story, which is why a resumed drive reported moved map units beside a
// freshly minted hero.
//
// A save taken between missions is refused, naming why: it carries mission
// number 0 and a map name left over from the previous mission, so there is
// nothing to resume and the map name would send a caller to the wrong map.
func ResumeOriginalSave(fsys entrySource, saved []byte, t *mapload.Table, diff mapload.Difficulty,
	party []mapload.PartyMember, bodies data.BodyList) (*Mission, OriginalSaveResume, error) {
	saved = repairLoadedEquipmentRows(saved)
	game := base.GameROM1
	if t != nil && t.Game != "" {
		game = t.Game
	}
	if err := validateOriginalGame(saved, game); err != nil {
		return nil, OriginalSaveResume{}, err
	}
	saved, modLayers, err := applyModMark(saved, tableModContext(t))
	if err != nil {
		return nil, OriginalSaveResume{}, err
	}
	f, err := sav.Open(saved)
	if err != nil {
		return nil, OriginalSaveResume{}, err
	}
	if _, err := f.Party(); errors.Is(err, sav.ErrSpellbook) {
		return nil, OriginalSaveResume{}, err
	}
	// The save, not the caller's diagnostic flag, owns this campaign choice.
	diff, err = campaignDifficulty(int64(f.Head.Difficulty))
	if err != nil {
		return nil, OriginalSaveResume{}, err
	}
	r := OriginalSaveResume{
		Mission:  int(f.Head.Mission),
		MapName:  f.Head.MapName,
		Label:    string(f.Label),
		HasWorld: f.World != nil,
		Heads:    len(f.Actors),
	}
	session, hasSession, err := originalSessionState(f)
	if err != nil {
		return nil, r, err
	}
	tails, hasTails, err := originalCellTails(f)
	if err != nil {
		return nil, r, err
	}
	cellRecords, hasCellRecords, err := originalCellRecords(f)
	if err != nil {
		return nil, r, err
	}
	spellEffects, hasSpellEffects, err := originalSpellEffects(f)
	if err != nil {
		return nil, r, err
	}
	projectiles, hasProjectiles, err := originalProjectiles(f)
	if err != nil {
		return nil, r, err
	}
	// A second, independent PartyWalk, on applyOriginalCellRecords' own
	// "independent reader" standing: RestoreParty above already walked and
	// discarded its own *Record, kept only the persistent-filtered, reordered
	// mapload.PartyMember view a Diary owner cannot be resolved from. Its own
	// walk error is not fatal here, the same tolerance the f.Party() call at
	// this function's own top already applies to every non-ErrSpellbook walk
	// failure: applyOriginalDiaries treats a nil diaryPlayerRec/empty
	// diaryChars as nothing to carry, not a reason to refuse the whole resume.
	diaryChars, diaryPlayerRec, _ := f.PartyWalk()
	for _, p := range f.Players {
		if p.Participant == 0 {
			r.Outcome, r.Money = p.Outcome, p.Money
			break
		}
	}
	if f.Head.Mission == 0 {
		return nil, r, fmt.Errorf("this save was taken BETWEEN missions: mission number 0, " +
			"and its map name is the previous mission's")
	}
	ground, hasGround, groundUnsupported, err := originalGroundState(f)
	if err != nil {
		return nil, r, err
	}
	document, origins := decodeSavedDocument(saved)
	currentStructures, err := readCurrentActions(document.Document)
	if err != nil {
		return nil, r, err
	}
	graph, err := currentActorGraph(f, document, origins)
	if err != nil {
		return nil, r, err
	}
	buildings, hasBuildings, err := f.Buildings()
	if err != nil {
		return nil, r, err
	}
	current := currentActorArchives(graph)
	pools, err := f.ActorPools(current...)
	if err != nil {
		return nil, r, err
	}
	profiles, err := f.ActorCurrentProfiles(current...)
	if err != nil {
		return nil, r, err
	}
	holdings, err := f.ActorHoldings(current...)
	if err != nil {
		return nil, r, err
	}
	books, err := f.ActorSpellbooks(current...)
	if err != nil {
		return nil, r, err
	}
	dead, err := f.DeadActors()
	if err != nil {
		return nil, r, err
	}
	r.HasWorld = hasGround
	addr, ok := MissionMap(r.Mission)
	if !ok {
		return nil, r, fmt.Errorf("mission %d: not a campaign mission number", r.Mission)
	}
	if fsys == nil {
		return nil, r, fmt.Errorf("read %s: no archive", addr)
	}
	b, err := fsys.ReadFile(addr)
	if err != nil {
		return nil, r, fmt.Errorf("read %s: %w", addr, err)
	}
	m, err := alm.Open(b)
	if err != nil {
		return nil, r, fmt.Errorf("%s: %w", addr, err)
	}
	for _, a := range f.Actors {
		if a.Dead() {
			r.Dead++
		}
	}
	if graph.CurrentPopulation {
		party = nil
	}
	restored, report := RestoreParty(f, party, bodies, t)
	if err := applyModLayers(restored, modLayers, tableModContext(t).Items); err != nil {
		return nil, r, err
	}
	// THE WITHDRAWAL RUNS BEFORE THE JOIN, so a placement a restored character
	// claims is gone before applyOriginalPositions goes looking for it: the
	// character carries his own position now and the record would be a second
	// entity for one person.
	r.Party = report
	if graph.CurrentPopulation {
		retainSavedActorPlacements(m, graph, dead, document)
	}
	registry, err := prepareOriginalActorRegistry(m, graph, restored, &r)
	if err != nil {
		return nil, r, err
	}
	censusOriginalBlocks(m, f, &r)
	// THE EXPLORED MAP IS COUNTED HERE AND NOT APPLIED, because this entry
	// point returns a Mission and builds no map screen: the fog plane belongs
	// to the driver openMission builds, and this path has none. The counted
	// report is therefore the whole of what a caller here gets, and
	// FogApplied stays false to say so.
	readOriginalFog(m, f, &r)
	ms, err := StartMissionFrom(m, addr, r.Mission, t, diff, restored)
	if err != nil {
		return nil, r, err
	}
	if err := applyOriginalCellTails(ms, tails, hasTails, &r); err != nil {
		return nil, r, err
	}
	if err := applyOriginalStructuresCurrent(ms, buildings, hasBuildings, t, &r, currentStructures, f); err != nil {
		return nil, r, err
	}
	if err := applyOriginalCellRecords(ms, cellRecords, hasCellRecords, registry, &r); err != nil {
		return nil, r, err
	}
	if err := applyOriginalSpellEffects(ms, spellEffects, hasSpellEffects, &r); err != nil {
		return nil, r, err
	}
	if err := applyOriginalProjectiles(ms, projectiles, hasProjectiles, &r); err != nil {
		return nil, r, err
	}
	if err := applyOriginalGround(ms, ground, hasGround, groundUnsupported, t, &r); err != nil {
		return nil, r, err
	}
	if err := admitOriginalActorRegistry(ms, registry, t, document); err != nil {
		return nil, r, err
	}
	if err := applyOriginalDiaries(ms, diaryPlayerRec, diaryChars, &r); err != nil {
		return nil, r, err
	}
	// Actor admission requires an unadvanced construction candidate. Overlay
	// the saved clocks only afterwards, still before any gameplay or adoption.
	if err := applyOriginalSession(ms, session, hasSession, &r); err != nil {
		return nil, r, err
	}
	if err := restoreOriginalActors(ms, holdings, pools, books, t, &r, profiles); err != nil {
		return nil, r, err
	}
	if err := applyOriginalFacings(ms, f.Actors, r.Party.SourceOffsets); err != nil {
		return nil, r, err
	}
	if err := applyOriginalDeadWithOrigins(ms, dead, &r, document, origins); err != nil {
		return nil, r, err
	}
	if err := applyOriginalGroups(ms, &r); err != nil {
		return nil, r, err
	}
	if err := importSavedDocument(ms, document, origins); err != nil {
		return nil, r, err
	}
	if err := restoreOriginalRandomState(ms); err != nil {
		return nil, r, err
	}
	if err := importOriginalCellPlanes(ms, fsys); err != nil {
		return nil, r, err
	}
	if err := importSavedSackObjects(ms, ms.savedDocument); err != nil {
		return nil, r, err
	}
	if err := applyOriginalDying(ms, &r); err != nil {
		return nil, r, err
	}
	if err := importOriginalActorEffects(ms); err != nil {
		return nil, r, err
	}
	if err := importOriginalWorldEffects(ms, fsys); err != nil {
		return nil, r, err
	}
	if err := restoreOriginalActions(ms, t); err != nil {
		return nil, r, err
	}
	publishSessionEntry(ms)
	currentGroups, _, _ := ms.World.SavedGroups()
	r.GroupsRestored, r.GroupIssues = len(currentGroups), ms.World.SavedGroupIssues()
	return ms, r, nil
}

func originalPartyCarriesMapUnit(party []mapload.PartyMember, mapUnitID uint16) bool {
	for _, p := range party {
		if p.Saved != nil && p.Saved.MapUnitID == mapUnitID {
			return true
		}
	}
	return false
}

// applyOriginalPositions validates the entire eligible join before changing the map.
// Only living Unit-derived owner-graph records can supply positions and owners. An absent
// map ID does not create an NPC; duplicate saved IDs or map targets are errors.
func applyOriginalPositions(m *alm.Map, f *sav.File, r *OriginalSaveResume) error {
	saved := make(map[uint16]sav.Actor)
	for _, a := range f.Actors {
		if a.Dead() || a.MapUnitID == 0 {
			continue
		}
		if _, exists := saved[a.MapUnitID]; exists {
			return fmt.Errorf("sav: ambiguous living actor map unit ID %d", a.MapUnitID)
		}
		saved[a.MapUnitID] = a
	}
	targets := make(map[uint16]bool)
	for _, u := range m.Units {
		if _, ok := saved[u.UnitID]; !ok {
			continue
		}
		if targets[u.UnitID] {
			return fmt.Errorf("sav: ambiguous map target unit ID %d", u.UnitID)
		}
		targets[u.UnitID] = true
	}
	for i := range m.Units {
		u := &m.Units[i]
		a, ok := saved[u.UnitID]
		if !ok {
			continue
		}
		r.Joined++
		x := uint32(a.Col())<<8 | uint32(a.FineX)
		y := uint32(a.Row())<<8 | uint32(a.FineY)
		if u.X>>8 != x>>8 || u.Y>>8 != y>>8 {
			r.Moved++
		}
		u.X, u.Y = x, y
		if u.Owner != uint32(a.OwnerSlot) {
			r.Reowned++
		}
		// Install the source owner before construction freezes owner-keyed
		// AI groups. Do not replay HandOver or invent saved group membership.
		u.Owner = uint32(a.OwnerSlot)
	}
	return nil
}

// censusOriginalBlocks measures the block-plane delta against what the map's own ingest
// derives, WITHOUT applying it (L-3).
//
// It counts the records naming a cell the derived plane leaves open. That is the
// encoding-independent question: the save's static byte and this tree's derived
// plane are two encodings of blocking and no claim relates them bit for bit, but
// "open" and "not open" mean the same thing in both.
func censusOriginalBlocks(m *alm.Map, f *sav.File, r *OriginalSaveResume) {
	if f.World == nil {
		return
	}
	r.BlockRecords = len(f.World.Blocks)
	plane := mapload.Passability(m)
	// The derived plane is row-major over the map's own extent, so its width
	// is the map's — but the derivation clamps an extent it cannot honour, so
	// the width is used only where it divides the plane exactly. A census is
	// not worth a wrong index.
	w := int(m.Width)
	if w <= 0 || len(plane) == 0 || len(plane)%w != 0 {
		w = 0
	}
	for _, rec := range f.World.Blocks {
		if rec.Dyn&originalOccupancyBits != 0 {
			r.BlockOccupied++
		}
		if w == 0 {
			continue
		}
		i := rec.Row()*w + rec.Col()
		if rec.Col() < w && i >= 0 && i < len(plane) && plane[i] == 0 {
			r.BlockNovel++
		}
	}
	for i := 0; i < 1000; i++ {
		v, err := f.TriggerLatch(i)
		if err != nil {
			return
		}
		if v != 0 {
			r.Latches++
		}
	}
}

// String is the report a driver prints.
//
// THE PARTY LINE COMES FIRST because it is the axis whose silence cost the owner
// a save: every count below it was already printed and correct while the party
// was a fresh one.
func (r OriginalSaveResume) String() string {
	world := "NO WORLD HALF"
	if r.HasWorld {
		continuity := "read and NOT APPLIED"
		if r.SessionApplied {
			continuity = "RESTORED (100 registers, 448 raw bytes)"
		}
		world = fmt.Sprintf("blocks %d (%d cells the map leaves open, %d occupied), "+
			"latches %d set, diplomacy %d cells, WIN/LOSE %d/%d and Player outcome %d %s",
			r.BlockRecords, r.BlockNovel, r.BlockOccupied, r.Latches,
			r.DiplomacyCells, r.WinCount, r.LoseCount, r.Outcome, continuity)
	}
	return fmt.Sprintf(
		"resume: mission %d map %q label %q outcome %d money %d\n"+
			"        %s\n"+
			"        owner-graph actors %d (%d excluded from living joins), joined %d to map units, MOVED %d, REOWNED %d, RESTOCKED %d\n"+
			"        %s\n"+
			"        %s\n"+
			"        ground sacks %d, item units %d, restored %t\n"+
			"        actor holdings %d RESTORED; party %d, dead/dying %d, off-map %d, unbound %d, unmatched %d skipped\n"+
			"        %d unsupported non-state-0 item effects (ground + holdings) reported and not applied\n"+
			"        cell-trigger records %d, six-byte overlay restored %t (operation26 relocation unsupported)\n"+
			"        cell-record residue %d rows, layer count/residue/Sack/SpellEffect keys and rebound Ground/Air carried %t\n"+
			"        spell-effect graph %d top-level records carried %t (targets, ordered delivery and known area payloads bound)\n"+
			"        projectile store %d distinct id(s), allocator/IDs/leaves carried %t (known coordinate/clock continuation bound separately)\n"+
			"        diaries %d carried (Player + Humanoid-owned), bound to their owner %t (entry meaning unknown; never simulated)\n"+
			"        actor pools %d RESTORED; party %d, excluded %d, unmatched %d skipped\n"+
			"        actor current profiles %d RESTORED; party %d, excluded %d, unmatched %d skipped\n"+
			"        actor spellbooks %d RESTORED (%d absent, %d empty, %d spells); party %d, excluded %d, unmatched %d skipped\n"+
			"        late corpses %d RESTORED, terminal actors %d RETAINED OUTSIDE PLAY, %d unbound (no authored map unit)\n"+
			"        dying owner-graph actors %d RESTORED with current pools, profile and holdings\n"+
			"        structures %d RESTORED (subclasses %d, unbound %d, unmatched %d); map-only %d removed; saved cell links retained\n"+
			"        source actor/load basis, equipment operands and container bookkeeping RESTORED\n"+
			"        supported living source actors bound by archive record; native identity/presentation retained\n"+
			"        saved Groups %d RESTORED; bounded continuation issues: %v\n"+
			"        NOT CARRIED: 18 bytes of still-unpromoted session gaps (SAV-SESS-031);\n"+
			"        Unbound Sack/SpellEffect keys, unsupported actor callbacks and\n"+
			"        effect payload components without a decoded field bridge;\n"+
			"        subclass services and structure destruction scheduling;\n"+
			"        complete original object graph and transitive Token/Position state\n"+
			"        remain outside this supported original-session handoff",
		r.Mission, r.MapName, r.Label, r.Outcome, r.Money,
		r.Party, r.Heads, r.Dead, r.Joined, r.Moved, r.Reowned, r.Stocked, world, r.fogLine(), r.Sacks, r.GroundItems, r.GroundApplied,
		r.Stocked, r.StockParty, r.StockDead, r.StockOffMap, r.StockUnbound, r.StockUnmatched,
		r.UnsupportedItemEffects,
		r.CellTriggerRecords, r.CellTriggersApplied,
		r.CellRecords, r.CellRecordsApplied,
		r.SpellEffects, r.SpellEffectsApplied,
		r.Projectiles, r.ProjectilesApplied,
		r.Diaries, r.DiariesApplied,
		r.PoolsRestored, r.PoolsParty, r.PoolsExcluded, r.PoolsUnmatched,
		r.ProfilesRestored, r.ProfilesParty, r.ProfilesExcluded, r.ProfilesUnmatched,
		r.Books.Restored, r.Books.Absent, r.Books.Empty, r.Books.Spells, r.Books.Party, r.Books.Excluded, r.Books.Unmatched,
		r.CorpsesRestored, r.TerminalRestored, r.UnboundRestored,
		r.DyingRestored,
		r.Structures.Restored, r.Structures.Subclasses, r.Structures.Unbound,
		r.Structures.Unmatched, r.Structures.Absent, r.GroupsRestored, r.GroupIssues)
}

// fogLine is the explored map's row of that report.
//
// IT STATES WHETHER THE PLANE REACHED THE SCREEN and not merely whether a
// record was read. The three cases are distinct and a reader has to be able to
// tell them apart: no record at all, which is what a between-mission save has;
// a record whose cell count disagrees with the map, which is refused; and a
// record that went in, whose cell count is printed so it can be checked against
// the file with savtool.
//
// THE PERCENTAGE IS OF THE RECORDED GRID and nothing else. How much of that the
// original DREW is not decoded — its render gate ORs neighbouring corner words,
// so the drawn extent may exceed the set cells — so this number is what the
// file holds, not a prediction of what a screen shows.
func (r OriginalSaveResume) fogLine() string {
	if r.FogCells == 0 {
		return "explored map: NO RECORD in this save"
	}
	pct := 100 * float64(r.FogSet) / float64(r.FogCells)
	if !r.FogApplied {
		return fmt.Sprintf("explored map: %d of %d cells (%.1f%%) read and NOT APPLIED "+
			"- no map screen on this path, or the count disagrees with the map",
			r.FogSet, r.FogCells, pct)
	}
	return fmt.Sprintf("explored map: %d of %d cells (%.1f%%) RESTORED",
		r.FogSet, r.FogCells, pct)
}

// String is the party half of that report, in the reader's units.
func (p RestoredParty) String() string {
	if p.Fallback {
		return fmt.Sprintf("party: NO CHARACTER RESTORED -- opened with the fresh party (%v)", p.Err)
	}
	// THE LEADER IS NAMED IN THE FILE'S OWN UNITS, AND SO IS THE RULE THAT
	// FOUND HIM. "led from 1" says the participant's character was written
	// second and now leads; "led from 0" says the file already had him first.
	// Two of the three rules are this tree's own choice, so which one applied
	// is printed beside the position: a party ordered by the fallback is a
	// party ordered on an inference, and a reader is entitled to know which he
	// is looking at.
	lead := fmt.Sprintf("led from position %d by %s", p.LeadFrom, p.LeadRule)
	if p.LeadFrom < 0 {
		lead = p.LeadRule.String()
	}
	s := fmt.Sprintf("party: %d persistent characters restored, %d temporary allies excluded, "+
		"%d with statistics and pools, %d off their own Humans row\n"+
		"        %s\n"+
		"        %d claimed a map placement and %d of those records were withdrawn,\n"+
		"        %d rebound to the script so an arm naming one measures the character\n"+
		"        wearing %d pieces (%d to the pack for want of a slot), carrying %d, armed %d of %d\n"+
		"        %d character-owned skill/experience sets restored\n"+
		"        %d unsupported non-state-0 item effects reported and not applied\n"+
		"        %d spellbooks, %d learned spell memberships RESTORED\n"+
		"        %d saved spell parameter sets RESTORED; NOT APPLIED: %d journals holding %d entries\n"+
		"        %d carry capacities, %d own-weight/load words RESTORED",
		p.Characters, p.Temporary, p.Pools, p.Definitions,
		lead,
		len(p.MapUnitIDs), p.Withdrawn, p.Rebound,
		p.Worn, p.WornSpare, p.Carried, p.Weapons, p.Armed,
		p.Skills,
		p.UnsupportedItemEffects,
		p.SpellbookChars, p.SpellRecords, p.SpellParameters, p.JournalChars, p.JournalEntries,
		p.Capacity, p.Unknowns)
	if p.Err != nil {
		s += fmt.Sprintf("\n        WALK STOPPED SHORT: %v", p.Err)
	}
	return s
}

// clearRestoredMissionPosition removes coordinates and map placement IDs before
// entering another mission. They belong to the previous map; character pools
// and other saved values remain current. The next map then uses its own drop.
func clearRestoredMissionPosition(party []mapload.PartyMember) {
	for i := range party {
		if s := party[i].Saved; s != nil {
			s.Cell, s.MapUnitID = mapload.Cell{}, 0
		}
	}
}

// RestoreOriginal opens a mission or town from the original-format SAV graph.
// Bind the imported baseline before hired-roster growth can replace its slices.
// SAV-608 excludes rebuilt hired members from bound-record counts; SAV-609's
// application map flags come from current state rather than a lazy UI cache.
func (f *FrontEnd) RestoreOriginal(saved []byte) (ui.MapOpener, bool, error) {
	open, ok, err := f.restoreOriginal(saved)
	var inn innArrayError
	if note := f.Base().Profile.Limits.OriginalSaveRefusal; err != nil && note != "" && errors.As(err, &inn) {
		err = fmt.Errorf("%w (base %s: %s)", err, f.Base().ID(), note)
	}
	return open, ok, err
}

// originalCampaignDecode is what an original SAV says about the campaign: the
// difficulty, the mission it was saved in, the town its campaign record
// describes and the current-session supplement when the file carries one.
type originalCampaignDecode struct {
	difficulty  mapload.Difficulty
	mission     int
	town        *Town
	progress    *campaignProgress
	session     *currentSessionData
	actions     *currentActionData
	document    *sav.DocumentData
	quickSpells [4]uint32
}

// decodeOriginalCampaign decodes and validates the complete campaign
// projection of an original SAV over the install's campaign, from plain
// values. It changes nothing: a malformed campaign tail cannot install a
// party, purse, town or partial progress record. quickSpells are the ordinary
// bindings the supplement may replace.
func decodeOriginalCampaign(sf *sav.File, saved []byte, campaign Campaign, quickSpells [4]uint32) (*originalCampaignDecode, error) {
	difficulty, err := campaignDifficulty(int64(sf.Head.Difficulty))
	if err != nil {
		return nil, err
	}
	// Decode and validate the complete campaign projection before changing any
	// front-end field. A malformed campaign tail therefore cannot install a
	// party, purse, town, or partial progress record.
	n := int(sf.Head.Mission)
	var currentSession *currentSessionData
	var currentActions *currentActionData
	var currentDocument *sav.DocumentData
	if doc, err := sav.DecodeDocumentData(saved); err == nil {
		a, err := readCurrentActions(&doc)
		if err != nil {
			return nil, err
		}
		if a != nil {
			currentSession = a.Session
			currentActions, currentDocument = a, &doc
		}
	}
	if currentSession != nil && currentSession.Difficulty != nil {
		difficulty = *currentSession.Difficulty
	}
	restoredTown := NewTown(campaign)
	var restoredProgress *campaignProgress
	if projection, ok, err := sf.Campaign(); err != nil {
		return nil, err
	} else if ok && currentSession != nil && currentSession.Second != nil {
		restoredTown = NewTown(Campaign{})
		if projection.Main.Mission != uint32(n) || projection.SelectedMission != uint32(n) {
			return nil, fmt.Errorf("current second campaign document does not match its mission")
		}
	} else if ok {
		progress, err := campaignProgressFromSAV(campaign, projection)
		if err != nil {
			return nil, err
		}
		restoredProgress = progress
		restoredTown = newTownFromCampaignProgress(campaign, progress)
		restoredTown.docPayload = projection.Payload.Clone()
		restoredTown.docPayloadError = projection.PayloadError
		if projection.PayloadError != "" {
			fmt.Fprintf(os.Stderr, "original save: document carrier records ignored and not written back: %s\n", projection.PayloadError)
		}
	}
	if n == 0 {
		restoredTown.lastMap = sf.Head.MapName
	}
	quickSpells, err = currentQuickSpells(quickSpells, currentSession)
	if err != nil {
		return nil, err
	}
	if currentSession != nil {
		if currentSession.Second != nil {
			restoredTown.second = currentSession.Second.restore()
		}
		for _, r := range currentSession.Taken {
			restoredTown.taken[offerRef{r.Chapter, TownBuilding(r.Building), r.Index}] = true
		}
		restoredTown.offerLabels = cloneOfferLabels(currentSession.OfferLabels)
		for _, n := range currentSession.Won {
			restoredTown.won[n] = true
		}
		if currentSession.DocumentMission != nil {
			restoredTown.docMission = *currentSession.DocumentMission
		}
		if err := validateSnapshotHeroGrants(campaign, Snapshot{HeroGrantState: true, ConsumedHeroGrants: currentSession.ConsumedHeroGrants}); err != nil {
			return nil, err
		}
		for _, chapter := range currentSession.ConsumedHeroGrants {
			if restoredTown.heroGrants == nil {
				restoredTown.heroGrants = map[int]bool{}
			}
			restoredTown.heroGrants[chapter] = true
		}
	}
	if err := validateCampaignLocation(campaign, restoredProgress, n, "original save"); err != nil {
		return nil, err
	}
	if n == 0 && campaign.completedBy(restoredTown) {
		return nil, errCompletedCampaignFile
	}
	return &originalCampaignDecode{difficulty: difficulty, mission: n, town: restoredTown, progress: restoredProgress,
		session: currentSession, actions: currentActions, document: currentDocument, quickSpells: quickSpells}, nil
}

// restoreOriginal is RestoreOriginal without the base profile's limit note.
// It decodes the file from plain values, then hands a between-mission save to
// a detached session and a mission save to the mission opener.
func (f *FrontEnd) restoreOriginal(saved []byte) (ui.MapOpener, bool, error) {
	saved = repairLoadedEquipmentRows(saved)
	if err := validateOriginalGame(saved, f.Base().Profile.GameOf()); err != nil {
		return nil, false, err
	}
	src, err := decodeOriginalSource(saved, f.modContext(), f.Campaign.Value())
	if err != nil {
		return nil, false, err
	}
	var selectedMarkers map[int]bool
	if src.campaign.town.second == nil {
		selectedMarkers, err = worldSelectedOnceFromSnapshot(worldMapMarkerMissions(src.campaign.town, coldWorldMapData(f.worldMapCache, f.Archives)))
		if err != nil {
			return nil, false, err
		}
	}
	in := f.originalInstall()
	if src.campaign.mission == 0 {
		return f.restoreOriginalTown(src, in, selectedMarkers)
	}
	return f.restoreOriginalMission(src, in, selectedMarkers)
}

// restoreOriginalTown installs the between-mission game an original SAV
// describes. The city is completed on a detached session: a hired-roster
// refusal must not replace the running campaign, counters or observer.
func (f *FrontEnd) restoreOriginalTown(src *originalSource, in originalInstall, selectedMarkers map[int]bool) (ui.MapOpener, bool, error) {
	if src.campaign.town.second != nil {
		if payload, err := readSecondTownTalk(&f.InstallResources, "npc517talk10"); err != nil || payload == nil {
			if err == nil {
				err = fmt.Errorf("initial inn conversation is unavailable")
			}
			return nil, false, err
		}
	}
	draftAudio := ui.NewAudioScope(f.SoundPlayer)
	defer draftAudio.Destroy()
	var draft CampaignSession
	report, err := draft.restoreOriginalTown(src, in, f.Campaign.Value())
	if err != nil {
		return nil, false, err
	}
	f.Units = cloneCandidateUnits(f.Units)
	f.installCandidate(&restoreCandidate{fame: draft.fame, quickSpells: src.quickSpells, offered: src.offered(),
		difficulty: src.campaign.difficulty, town: draft.Town, carried: draft.Carried,
		originalCity: draft.originalCity, townOnly: true, worldSelectedOnce: selectedMarkers, worldMapReturn: src.worldMapReturn()})
	f.CampaignSession.reportTown(src, report)
	return nil, true, nil
}

// restoreOriginalMission prepares the mission an original SAV was saved in and
// returns the opener that commits it.
func (f *FrontEnd) restoreOriginalMission(src *originalSource, in originalInstall, selectedMarkers map[int]bool) (ui.MapOpener, bool, error) {
	plan, err := src.planMission(in, cloneCandidateUnits(f.Units), selectedMarkers)
	if err != nil {
		return nil, false, err
	}
	candidate := plan.candidate
	open := f.missionOpenerMode(src.campaign.mission, plan.restored, nil, plan.savedPurse, src.prepareMission(in, plan),
		&candidate.activate, candidate.units, src.campaign.difficulty, candidate.town)
	candidate.prepared.viewer, candidate.prepared.tick, candidate.prepared.order, candidate.prepared.cadence,
		candidate.prepared.affect, candidate.prepared.advance, candidate.prepared.attack, candidate.prepared.grab,
		candidate.prepared.stance, candidate.prepared.march, err = open()
	if err != nil {
		return nil, false, err
	}
	if candidate.activate == nil {
		return nil, false, fmt.Errorf("prepared original mission has no live-driver commit")
	}
	return openPrepared(candidate.prepared, func() { f.installCandidate(candidate) }), false, nil
}

// OriginalSaveLabel is the row the LOAD GAME window shows for one such file.
//
// IT IS BUILT FROM THE HEAD ALONE so that listing a directory of saves costs one
// short read each and never a full decode — the same bargain SaveLabel makes for
// our own format. A file that will not read has no label and is dropped from the
// list by the caller, not shown as a row that would refuse when chosen.
//
// selector IS THE READER'S OWN CHOICE, not a fact read from the file — see
// asciiLabel's own doc for why.
func OriginalSaveLabel(saved []byte, selector int) (string, error) {
	sf, err := sav.Open(saved)
	if err != nil {
		return "", err
	}
	label := asciiLabel(sf.Label, selector)
	if label == "" {
		label = "(no label)"
	}
	if sf.Head.Mission == 0 {
		return fmt.Sprintf("%s - between missions", label), nil
	}
	return fmt.Sprintf("%s - mission %d", label, sf.Head.Mission), nil
}

// asciiLabel makes a label drawable, decoding it through the CURRENT
// install's own single-byte page rather than refusing every byte past ASCII.
//
// WHICH PAGE A GIVEN .sav's LABEL WAS ACTUALLY WRITTEN IN IS STILL NOT A FACT
// THIS TREE KNOWS. The format spec calls the region a string and does not say
// what encodes it; one save in the corpus begins with the byte 0xE0, which is
// a Cyrillic letter in more than one of the pages this game is known to use
// and a DIFFERENT letter in each. That is still true and still unresolved.
//
// WHAT CHANGED IS THAT THIS BUILD NOW WRITES ONE ITSELF: a SAV label this
// build's own SAVE dialog produces is encoded through textinput.EncodeRune
// under the selected install's own selector (playerCitySave), the same
// mechanism already used to draw every other installed string (DIV-019).
// Decoding through textinput.DecodeByte — its exact inverse — is
// therefore an EXACT, lossless read of what THIS BUILD wrote, not a guess.
// For a label this build did not write (a genuinely original .sav, or a
// stray byte a stray writer produced), the same decode is still only a
// CHOSEN DEFAULT: the current install's own code page, on the same DIV-019
// convention every other piece of installed text already draws through,
// rather than an invented fact about that file's own origin. A byte the
// chosen page cannot represent — a control byte, or the one Windows-1251
// gap (textinput.DecodeByte) — still becomes '?', so a name's length and
// shape survive even where a letter cannot.
//
// This is no longer an open question on the write side (golden rule 4 is
// satisfied by round-tripping our own encoder, not by guessing); it remains
// one for a label this build never wrote, which is why the fallback stays.
func asciiLabel(raw []byte, selector int) string {
	var out []rune
	for _, c := range raw {
		if c == 0 {
			// A NUL ends a C string; nothing past it is label.
			break
		}
		if r, ok := textinput.DecodeByte(c, selector); ok {
			out = append(out, r)
			continue
		}
		out = append(out, '?')
	}
	return string(out)
}
