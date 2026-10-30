package game

import (
	"errors"
	"slices"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// maxLabel bounds a SAV label and every bounded name carried in a persisted
// record, so a document naming a gigabyte of text is refused before anything is
// allocated for it.
const maxLabel = 4096

// maxSaveBytes is the largest complete save file this build will read or
// publish. It bounds raw-byte allocation; maxSaveGobElements separately bounds
// the container counts a persisted snapshot may carry.
const maxSaveBytes = 16 << 20

// maxSaveGobElements is the aggregate container-count budget for a persisted
// snapshot record.
const maxSaveGobElements uint64 = 1 << 16

// maxSaveGobDepth bounds how deep a persisted snapshot record may nest.
const maxSaveGobDepth = 128

// Snapshot is everything a saved game holds.
//
// Exported fields retain the legacy checkpoint shape. Private fields are
// capture-only constructor inputs; the SAV producer writes their applicable
// values through the ordinary document and current continuation policy.
//
// Town.camp remains the registry's immutable campaign definition. Campaign
// below is the mutable projection imported from an original save: a save
// carries the player's progress through that definition, not a second registry.
type Snapshot struct {
	game base.Game
	// guest is the session's mod mission or waiting guest.
	guest *guestMission
	// randomSession is the random session at capture; the SAV records it.
	randomSession random.Session
	second        *currentSecondCampaign
	// towns is the first game's town record: nil while the party belongs to
	// town 1 and every other town keeps its fresh state.
	towns        *currentTowns
	noticeOpen   bool
	shop         *Shop
	terrainBase  *sim.Terrain
	mapAnimation *ui.AnimationState
	mapMotion    []currentAnimationClock
	CityObjects  *cityObjectTopology
	// cityGroups is the town's live Player group membership at capture.
	cityGroups       []cityLiveGroup
	cityGroupsLoaded bool
	// knowledge is the town's carried Player Diary at capture (UNIT-148,
	// SAV-847). The town SAV writes it into the Player's Diary record.
	knowledge []sim.SavedDiary
	// townOptions are the loaded town's option and view leaves, captured with
	// the Snapshot as a mission captures its application state.
	townOptions []sav.CityStateRecordData
	// playerColor is the town's carried local Player +0x44 at capture.
	playerColor uint8
	// tavernStockIDs are the town's tavern stock unit ids at capture.
	tavernStockIDs []uint32
	// NativeMissionTerrain requests the original writer's own scoped
	// block-plane row set (a delta inside its own sweep window, only where
	// a cell's dynamic byte exceeds the ingested baseline) instead of the
	// full per-cell census every other export keeps. It is the caller's own
	// explicit choice, never inferred from whether a prior document exists:
	// a first-ever SAVE of an ordinary, continuing mission carries no prior
	// document either, and that save's own round trip through this engine
	// still needs the full census to reload byte for byte. Only a generated
	// mission SAV meant for the original game itself sets this true. Ordinary
	// current-state capture leaves it false.
	NativeMissionTerrain bool
	// The opaque World binary omits this constructor input. Capture it with
	// the World so a later save cannot substitute another mission's template.
	// Nil on older checkpoints uses the installed mission constructor.
	ghost *sim.GhostTemplate
	// loadedRecord is the live World's loaded-record carrier, which the byte
	// form does not carry (worldcarriers.go).
	loadedRecord sim.LoadedRecord
	// loadedDocument is the loaded document while the writer replaces
	// SavedDocument with the World-built one; only unknown spans read it.
	loadedDocument *SnapshotSAVDocument
	// Current mission actor labels bind the ordered entry party and per-actor
	// derive inputs. They are captured with World, never inferred from names.
	CurrentPartyIDs []sim.EntityID
	CurrentRoster   map[sim.EntityID]mapload.PartyMember
	// Fame is additive campaign score history. Nil means an older writer never
	// recorded it, so restore does not invent a complete earned score.
	Fame *SnapshotFame
	// ApplicationState is current mission UI state. Nil is the legacy default.
	ApplicationState *SnapshotApplicationState
	// CameraSet distinguishes an explicitly captured mission camera from the
	// constructor default. Coordinates use cells, as in SaveApplicationState.
	CameraSet                    bool
	CameraX, CameraY, CameraZoom float64
	// QuickSpells holds spell IDs in F5-F8 order. Current spell, mode and book
	// visibility are captured by their own application fields.
	QuickSpells [4]uint32
	// Difficulty is campaign state. World is already scaled and is never
	// adjusted during restore (UNIT-GATE-013).
	Difficulty mapload.Difficulty
	// MainMission is the native campaign main; ordinary Campaign is authoritative
	// when present. Otherwise the active mission and history supply it.
	MainMission     int
	SelectedMission int
	Open            bool
	Gold            int
	Won             []int
	Available       []int
	Taken           []SnapshotOffer
	OfferLabels     *currentOfferLabels

	// HeroGrantState marks captured consumption of native AddHero arrays.
	HeroGrantState     bool
	ConsumedHeroGrants []int

	// MercenaryState marks captured tavern stock and hired types.
	MercenaryState   bool
	MercenaryPool    [16]int
	MercenaryEnabled [16]bool
	MercenaryHired   [16]bool

	// Documents is the campaign document collection. DocumentMission is the
	// monotonic guard that prevents collecting it twice on mission entry.
	Documents       []SnapshotDocument
	DocumentMission int

	// LastMap is Town.lastMap. An older payload decodes it empty, the value
	// every town SAV was written with before it existed.
	LastMap string

	// CampaignState marks a captured campaign projection.
	// CampaignMarkerSelected carries the presentation latch that becomes true
	// when a player selects a mission after import. It is deliberately separate
	// from Campaign.Markers: no synthetic original-format marker record or
	// meaning for its unnamed dwords is invented merely to persist an againrom
	// session fact.
	CampaignState          bool
	Campaign               sav.CampaignProjection
	CampaignMarkerSelected bool
	// DocPayload is the Againrom-only payload carried in the SAV document
	// list. Nil on every payload that predates it, and when none is held.
	DocPayload *sav.DocPayload
	// Absent records recover current candidates at age0 and an accepted
	// previous-chapter child at age1. Historical entry ages are not recoverable.
	CampaignRecords *SnapshotCampaignRecords

	// WorldSelectedOnce is the set of missions selected at least once on this
	// build's world map, in the order they entered it. It is separate from Campaign.Markers for the
	// same reason as CampaignMarkerSelected: a native snapshot persists its
	// own presentation gate without inventing an original-format record.
	WorldSelectedOnce []int

	// WorldMapReturn reads legacy travel saves. LOAD resolves it to the city;
	// current saves never write it.
	WorldMapReturn *SnapshotMapReturn

	// OriginalCity is bounded semantic provenance for current city state.
	// It holds no source bytes or original file path.
	OriginalCity *SnapshotOriginalCity

	// Party is what FrontEnd.Carried held and Offered is the mission the
	// campaign last offered. Both are the front-end's own half of what
	// survives a mission.
	Party   []mapload.PartyMember
	Offered int

	// Mission is which mission the save was taken in, and ZERO IS THE TOWN. A
	// snapshot with no mission has no world half at all, which is the shape a
	// save taken between missions has to have; a snapshot with one carries both
	// halves.
	Mission int

	// World is sim.World.MarshalBinary's own bytes, carried OPAQUE (plan
	// D-1). This package authors nothing inside them and the simulation's
	// version discipline stays exactly where it already is.
	World []byte

	// ActorManifest is presentation/reconstruction metadata for original actors
	// not described by the initial party alone. Nil preserves old native saves.
	// Canonical actor values remain in World, never in this manifest.
	ActorManifest *SnapshotActorManifest

	// SavedDocument retains the complete detached SAV graph and exact native
	// object bindings. Nil is the deterministic pre-document/native-new mode.
	// World remains the authority for every admitted live value.
	SavedDocument *SnapshotSAVDocument

	// Residue is the front-end's per-map memory that is neither derivable on
	// resume nor cosmetic. It is empty for a town save.
	Residue SnapshotResidue
}

// SnapshotMapReturn reconstructs a homeward route from registry points. It
// carries no completed mission world, reward, or original-save byte layout.
type SnapshotMapReturn struct {
	Mission int
	Shown   int
}

// SnapshotOffer is one consumed offer: the three coordinates offerRef holds.
type SnapshotOffer struct {
	Chapter  int
	Building int
	Index    int
}

// SnapshotDocument is one element of the campaign document collection.
type SnapshotDocument struct {
	Value int
	Kind  int
}

type SnapshotResidue struct {
	// Pending choices and visuals are captured without executing a tick. The
	// ordinary SAV producer writes their typed application supplement.
	PendingMessages    []int32
	PendingGameOptions []PendingGameOption
	PendingQueue       *SnapshotPendingQueue
	SpellBolts         []SnapshotSpellBolt
	HealBursts         []SnapshotHealBurst
	TargetMarks        []SnapshotTargetMark
	CastRuns           []SnapshotCastRun
	VisualIdentities   []SnapshotVisualIdentity
	VisualNext         sim.EntityID

	// MissionLost preserves the frontend loss latch, including a reserved
	// failure message without a simulation loss counter. Old saves default
	// false and still derive death/script outcomes from their saved world.
	MissionLost bool

	// DepartedCharacters retains ownership transfers after the entity's final
	// removal. Extant actors also supply their current owner.
	DepartedCharacters []uint32

	// Commanded is every entity the front end has ordered. A resumed world
	// that forgot this puts those units back under the placeholder script.
	Commanded []uint32

	// Swing and Phase are one memory in two maps: how many ticks each
	// entity's current attack run has played, and the phase the run is
	// detected against. Carried together because a zero Phase over a
	// mid-swing world restarts the run and re-fires its sound.
	Swing map[uint32]int
	Phase map[uint32]uint8

	// GroupTag is the tag the NEXT group order will carry.
	GroupTag uint32

	// Native saves retain both accumulated exploration and the last scheduled
	// visibility sample. Nil FogVisible is the legacy pre-sample format; an
	// original SAV carries exploration only and still uses its own LOAD path.
	FogCols     int
	FogRows     int
	FogExplored []byte
	FogVisible  []byte

	// deathAges is how long each fallen actor has lain.
	deathAges []currentDeathAge
	// keyed holds a carrier with no field of its own, by its SAV key name.
	keyed map[string]any
}

// snapshotTown reads the town's five fields out. It is a function in this
// package rather than a method elsewhere because those fields are unexported
// and stay that way: a save is not a reason to open the town's state to
// every caller.
func snapshotTown(t *Town, s *Snapshot) {
	if t == nil {
		return
	}
	s.second = captureSecondCampaign(t.second)
	s.towns = t.townsRecord()
	s.noticeOpen = t.second.noticePending()
	s.Open, s.Gold = t.open, t.gold
	s.MainMission = t.currentMain()
	s.SelectedMission = t.selectedMission()
	s.OfferLabels = savedOfferLabels(t)
	s.CityObjects = t.cityObjects.Clone()
	s.cityGroups, s.cityGroupsLoaded = slices.Clone(t.cityGroups), t.cityGroupsLoaded
	s.knowledge = slices.Clone(t.knowledge)
	s.playerColor = t.playerColor
	s.tavernStockIDs = slices.Clone(t.tavernStockIDs)
	s.HeroGrantState = true
	for chapter, consumed := range t.heroGrants {
		if consumed {
			s.ConsumedHeroGrants = append(s.ConsumedHeroGrants, chapter)
		}
	}
	sortInts(s.ConsumedHeroGrants)
	s.MercenaryState = true
	s.MercenaryPool, s.MercenaryEnabled, s.MercenaryHired = t.mercPool, t.mercEnabled, t.mercHired
	for n := range t.won {
		s.Won = append(s.Won, n)
	}
	for n := range t.available {
		s.Available = append(s.Available, n)
	}
	for r := range t.taken {
		s.Taken = append(s.Taken, SnapshotOffer{Chapter: r.chapter, Building: int(r.building), Index: r.index})
	}
	// THE DOCUMENT COLLECTION IS NOT SORTED and must not be. Won, Available
	// and Taken come off Go maps and are sorted below so that one game state
	// writes one set of bytes; the collection is a SLICE whose order is the
	// order the campaign granted its elements in, which is what the panel
	// pages through. Sorting it would silently reorder the player's
	// documents.
	for _, d := range t.documents {
		s.Documents = append(s.Documents, SnapshotDocument{Value: d.Value, Kind: d.Kind})
	}
	s.DocumentMission = t.docMission
	s.DocPayload = t.docPayload.Clone()
	s.LastMap = t.lastMap
	if t.progress == nil && t.main > 0 {
		s.CampaignRecords = t.records.snapshot()
	}
	if t.progress != nil {
		s.CampaignState = true
		s.Campaign = t.progress.projection()
		s.CampaignMarkerSelected = t.progress.markerSelected
		s.Campaign.MercenaryWorking = s.Campaign.MercenaryWorking[:0]
		s.Campaign.MercenaryPristine = s.Campaign.MercenaryPristine[:0]
		s.Campaign.MercenaryHired = s.Campaign.MercenaryHired[:0]
		for i := 1; i < 16; i++ {
			s.Campaign.MercenaryWorking = append(s.Campaign.MercenaryWorking, uint16(t.mercPool[i]))
			s.Campaign.MercenaryPristine = append(s.Campaign.MercenaryPristine, uint16(t.mercCapacity[i]))
			s.Campaign.MercenaryHired = append(s.Campaign.MercenaryHired, t.mercHired[i])
		}
		s.Campaign.Payload = t.docPayload.Clone()
		s.Campaign.Documents = s.Campaign.Documents[:0]
		for _, d := range t.documents {
			s.Campaign.Documents = append(s.Campaign.Documents,
				sav.CampaignDocument{Value: uint32(d.Value), Kind: uint32(d.Kind)})
		}
	}
	sortInts(s.Won)
	sortInts(s.Available)
	// Taken comes off a Go map, same as Won and Available above; unlike
	// those two this ordering went unsorted until round-2 adversarial review
	// (1017 pass 1) found it live. A save holding two or more taken offers
	// otherwise writes nondeterministic bytes for the same game state.
	sortOffers(s.Taken)
}

// snapshotWorldSelectedOnce copies townScreen's set into Snapshot in the order
// its members entered it; a member installed without an order follows in
// ascending order. False map values are not members of the set and are not
// serialized merely because a test or future caller left a key behind.
func snapshotWorldSelectedOnce(t *townScreen, s *Snapshot) {
	if t == nil || s == nil {
		return
	}
	listed := make(map[int]bool, len(t.worldSelectedOrder))
	for _, mission := range t.worldSelectedOrder {
		if t.worldSelectedOnce[mission] && !listed[mission] {
			listed[mission] = true
			s.WorldSelectedOnce = append(s.WorldSelectedOnce, mission)
		}
	}
	var rest []int
	for mission, selected := range t.worldSelectedOnce {
		if selected && !listed[mission] {
			rest = append(rest, mission)
		}
	}
	sortInts(rest)
	s.WorldSelectedOnce = append(s.WorldSelectedOnce, rest...)
}

// restoreTown builds a detached current town over c. Ordinary campaign records
// own represented fields; legacy inputs recover native identity from history.
// A town record the campaign cannot restore is an error.
func restoreTown(c Campaign, s Snapshot) (*Town, error) {
	t := NewTown(c)
	t.second = s.second.restore()
	restoredCampaign := false
	if s.CampaignState {
		if progress, err := campaignProgressFromSAV(c, s.Campaign); err == nil {
			progress.markerSelected = progress.markerSelected || s.CampaignMarkerSelected
			t = newTownFromCampaignProgress(c, progress)
			restoredCampaign = true
		}
	}
	if !restoredCampaign {
		t.open = s.Open
	}
	t.cityObjects = s.CityObjects.Clone()
	t.guest = s.guest.clone()
	t.cityGroups, t.cityGroupsLoaded = slices.Clone(s.cityGroups), s.cityGroupsLoaded
	t.knowledge = slices.Clone(s.knowledge)
	t.playerColor = s.playerColor
	t.tavernStockIDs = slices.Clone(s.tavernStockIDs)
	t.gold = s.Gold
	if s.MercenaryState && !restoredCampaign {
		t.mercPool, t.mercEnabled, t.mercHired = s.MercenaryPool, s.MercenaryEnabled, s.MercenaryHired
	}
	for _, n := range s.Won {
		t.won[n] = true
		if !s.MercenaryState {
			t.applyMercenaryUnlocks(n)
		}
	}
	if !restoredCampaign {
		if containsMission(c.Main, s.MainMission) {
			t.main = s.MainMission
		} else {
			for _, mission := range c.Main {
				if t.won[mission] {
					t.advanceMain(mission)
				}
			}
		}
		t.activateMission(s.Mission)
		t.selectMission(s.SelectedMission)
		if s.Open {
			t.Arrive()
		}
	}
	if !restoredCampaign {
		t.restoreCampaignRecords(s)
	}
	for _, r := range s.Taken {
		t.taken[offerRef{chapter: r.Chapter, building: TownBuilding(r.Building), index: r.Index}] = true
	}
	t.offerLabels = cloneOfferLabels(s.OfferLabels)
	if err := t.restoreTowns(s.towns); err != nil {
		return nil, err
	}
	if !restoredCampaign {
		restoreNativeHeroGrants(t, s)
	}
	// THE COLLECTION IS RESTORED THROUGH THE SAME APPEND THE CAMPAIGN USES,
	// so a payload that somehow carried the same pair twice restores as one
	// element rather than as two the panel would page through twice.
	if !restoredCampaign {
		for _, d := range s.Documents {
			t.addDocument(Document{Value: d.Value, Kind: d.Kind})
		}
		t.docMission = s.DocumentMission
	}
	t.lastMap = s.LastMap
	t.docPayload = s.DocPayload.Clone()
	return t, nil
}

// sortInts is sort.Ints under another name, kept here so this file states its
// own ordering rather than importing a package for one call in a file that
// otherwise reaches only the encoders.
func sortInts(v []int) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}

// offerLess orders two SnapshotOffer values by chapter, then building, then
// index — the same field order snapshotTown reads them off offerRef in.
func offerLess(a, b SnapshotOffer) bool {
	if a.Chapter != b.Chapter {
		return a.Chapter < b.Chapter
	}
	if a.Building != b.Building {
		return a.Building < b.Building
	}
	return a.Index < b.Index
}

// sortOffers is sortInts's own insertion sort, over SnapshotOffer.
func sortOffers(v []SnapshotOffer) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && offerLess(v[j], v[j-1]); j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}

// worldBytes marshals a world for the world half, or answers nothing for a
// world that is not there.
func worldBytes(w *sim.World) ([]byte, error) {
	if w == nil {
		return nil, errors.New("no world to save")
	}
	return w.MarshalBinary()
}
