package sim

import (
	"reflect"
	"slices"
	"testing"
)

// worldMethods pins the exported method set by reflection. Every addition must
// declare whether it reads or mutates deterministic state.
var worldMethods = []string{"ActiveEffects", "BindSourceDerive", "BookSpellCellRefusal", "BookSpellRefusal", "BoundarySurvivors", "Bounds", "Carried", "CarriedItems", "CarriedStacks", "CastingSpell", "CellEffects", "CellTails", "CompleteSackPickup", "CopyPotionEffects", "DeclareCellTails", "DeclareItemWeights", "DeclareStructures", "DropLanding", "Entities", "Entity", "EntityView", "EquipSourceCarried", "EquipSourceItem", "Equipped", "EquippedItems", "FireWallCount", "FormationMode", "Ghost", "GroupRateTerm", "HasEffectSpell", "HasNativeAreaEffects", "Hash", "HeadlessDamage", "HeadlessHeal", "HeadlessKill", "HeadlessKillPlayer", "HeadlessPlace", "ImportOriginalActorFacings", "ImportOriginalActorPools", "ImportOriginalActorProfiles", "ImportOriginalActorSpellbooks", "ImportOriginalActorStock", "ImportOriginalCellTails", "ImportOriginalDeadActors", "ImportOriginalLivingActors", "ImportOriginalSession", "ImportOriginalStructureHealth", "ImportSavedGroups", "InvisibleTo", "ItemWeights", "MarshalBinary", "MoveCarried", "NextEntityID", "OriginalDeadActors", "Outcome", "Purse", "Relations", "ReplaceGroundSacks", "ReplaceStock", "RestoreActorLoad", "RestoreOneShotPlayerCasts", "RestorePotionEffect", "Route", "Rules", "SackValue", "Sacks", "SavedGroupIssues", "SavedGroups", "Script", "ScriptCasts", "ScriptCounters", "ScriptLatched", "ScriptPassJustRan", "ScriptRegister", "ScorchedCells", "ScrollCasts", "SessionClock", "SetCombat", "SetDerived", "SetHumanMovement", "SetPotionHeadroom", "SetPurse", "SetRules", "RateSpeed", "ScriptTriggerBlocked", "Sight", "Spell", "Spells", "StepRate", "Stock", "Structures", "TakeSack", "Tick", "UnequipSource", "UnmarshalBinary", "UseCarriedPotion", "WeaponSpellDamage", "WithoutStructureAndItemStateSections"}

// worldWriters excludes explicit state mutations from the reader sweep.
var worldWriters = []string{"BindSourceDerive", "CompleteSackPickup", "CopyPotionEffects", "DeclareCellTails", "DeclareItemWeights", "DeclareStructures", "EquipSourceCarried", "EquipSourceItem", "HeadlessDamage", "HeadlessHeal", "HeadlessKill", "HeadlessKillPlayer", "HeadlessPlace", "ImportOriginalActorFacings", "ImportOriginalActorPools", "ImportOriginalActorProfiles", "ImportOriginalActorSpellbooks", "ImportOriginalActorStock", "ImportOriginalCellTails", "ImportOriginalDeadActors", "ImportOriginalLivingActors", "ImportOriginalSession", "ImportOriginalStructureHealth", "ImportSavedGroups", "MoveCarried", "ReplaceGroundSacks", "ReplaceStock", "RestoreActorLoad", "RestoreOneShotPlayerCasts", "RestorePotionEffect", "SetCombat", "SetDerived", "SetHumanMovement", "SetPotionHeadroom", "SetPurse", "SetRules", "TakeSack", "UnequipSource", "UnmarshalBinary", "UseCarriedPotion"}

func init() {
	worldMethods = append(worldMethods, "CheatGod", "CheatSpell", "CheatKillPlayer", "CheatPickupAll", "CheatAddGold", "CheatAddItem", "CheatCurse", "CheatSummon", "SetSafeMode")
	worldWriters = append(worldWriters, "CheatGod", "CheatSpell", "CheatKillPlayer", "CheatPickupAll", "CheatAddGold", "CheatAddItem", "CheatCurse", "CheatSummon", "SetSafeMode")
	worldMethods = append(worldMethods, "PlayerParticipants", "RestorePlayerParticipants", "SetPlayerParticipant", "CurrentPlayers", "RestoreCurrentPlayers", "RestoreCurrentPlayerIdentities", "RestoreCurrentPlayerRegistryAbsent")
	worldWriters = append(worldWriters, "RestorePlayerParticipants", "SetPlayerParticipant", "RestoreCurrentPlayers", "RestoreCurrentPlayerIdentities", "RestoreCurrentPlayerRegistryAbsent")
	worldMethods = append(worldMethods, "ActorOrderProgress", "CancelSackPickup")
	worldWriters = append(worldWriters, "CancelSackPickup", "ReleaseUnitShot")
	worldMethods = append(worldMethods, "RestoreAppliedPotionEffect", "RestoreCarriedPotionEffect")
	worldWriters = append(worldWriters, "RestoreAppliedPotionEffect", "RestoreCarriedPotionEffect")
	worldMethods = append(worldMethods, "Actions", "RestoreActions")
	worldWriters = append(worldWriters, "RestoreActions")
	worldMethods = append(worldMethods, "CurrentPolicy", "CurrentTerminalActors", "RestoreCurrentContinuation", "ReplaceCurrentObjects", "RestoreCurrentAreas", "RestoreCurrentSpellDeliveries", "RestoreCurrentPlayerSlots", "ImportCurrentTerminalActors", "RestoreCurrentTerminalActors", "RetireUnboundConstructors")
	worldMethods = append(worldMethods, "RestoreCurrentSpellDurations")
	worldWriters = append(worldWriters, "ImportCurrentTerminalActors", "RestoreCurrentTerminalActors", "RetireUnboundConstructors")
	worldWriters = append(worldWriters, "RestoreCurrentContinuation", "ReplaceCurrentObjects", "RestoreCurrentAreas", "RestoreCurrentSpellDeliveries", "RestoreCurrentPlayerSlots")
	worldWriters = append(worldWriters, "RestoreCurrentSpellDurations")
	worldMethods = append(worldMethods, "RestoreAbsentStructureCells", "RestoreActorIdentities", "RestoreCurrentClock", "RestoreGroupCarrierPresence")
	worldWriters = append(worldWriters, "RestoreAbsentStructureCells", "RestoreActorIdentities", "RestoreCurrentClock", "RestoreGroupCarrierPresence")
	worldMethods = append(worldMethods, "RestoreCurrentCarrierAbsence", "RestoreCurrentArchiveCoordinates")
	worldWriters = append(worldWriters, "RestoreCurrentCarrierAbsence", "RestoreCurrentArchiveCoordinates")
	worldMethods = append(worldMethods, "RestoreCurrentSackKeyAbsence")
	worldWriters = append(worldWriters, "RestoreCurrentSackKeyAbsence")
	worldMethods = append(worldMethods, "ActorIdentityReferences")
	worldMethods = append(worldMethods, "ImportCurrentLivingActors")
	worldWriters = append(worldWriters, "ImportCurrentLivingActors")
	worldMethods = append(worldMethods, "AppendCorpseStates")
	// Native LOAD may fill a verified legacy program's missing bindings.
	worldMethods = append(worldMethods, "RestoreScriptBindings", "RestoreScriptProgram", "ReserveEntityIDs")
	worldWriters = append(worldWriters, "RestoreScriptBindings", "RestoreScriptProgram", "ReserveEntityIDs")
	// Mission completion is an explicit boundary writer, never a tick API.
	worldMethods = append(worldMethods, "NormalizeMissionSurvivors")
	worldWriters = append(worldWriters, "NormalizeMissionSurvivors")
	worldMethods = append(worldMethods, "PublishSessionEntry", "PublishSessionEntryWithCarried")
	worldWriters = append(worldWriters, "PublishSessionEntry", "PublishSessionEntryWithCarried")
	worldMethods = append(worldMethods, "SavedWorldEffectDrivers", "ImportOriginalWorldEffectDrivers")
	worldWriters = append(worldWriters, "ImportOriginalWorldEffectDrivers")
	worldMethods = append(worldMethods, "ImportOriginalAttachedEffects")
	worldWriters = append(worldWriters, "ImportOriginalAttachedEffects")
	worldMethods = append(worldMethods, "SavedPlayerFormations", "CommandFormationMode", "ImportSavedPlayerFormations")
	worldWriters = append(worldWriters, "ImportSavedPlayerFormations")
	worldMethods = append(worldMethods, "RestoreStructureUseMetadata", "RestoreStructureBlocking", "StructureUses")
	worldWriters = append(worldWriters, "RestoreStructureUseMetadata", "RestoreStructureBlocking")
	// SavedStructures is a detached reader. ImportOriginalStructures is an
	// atomic construction-time writer, never an advancing simulation API.
	worldMethods = append(worldMethods, "ReleaseUnitShot", "SavedStructures", "ImportOriginalStructures", "SavedGroupPlayers", "ImportSavedGroupPlayers", "ActorFinePosition", "ActorMotionActive", "ImportOriginalActorMotions", "SavedActorMotionIssues", "SavedActorMotions", "SavedCellPlanes", "ImportOriginalCellPlanes", "SavedObjects", "SavedSackCellKey", "ImportSavedObjects", "ScriptRegisters", "RawSessionHead", "RawSessionMid", "SetRawSessionHead", "SetRawSessionMid", "SavedCellRecords", "ImportOriginalCellRecords", "SetSavedCellRecords", "SavedSpellEffects", "SetSavedSpellEffects", "SetSkillLevels", "SavedProjectiles", "ImportOriginalProjectiles", "SetSavedProjectiles", "SavedDiaries", "ImportOriginalDiaries", "SetSavedDiaries")
	// The burst picture's phase count is install-derived input, set once per world.
	worldMethods = append(worldMethods, "SetBurstPhases", "WindUpElapsed")
	worldMethods = append(worldMethods, "ReleaseCast", "ReleaseAreaBurst", "ProjectilePoint")
	worldWriters = append(worldWriters, "ReleaseCast", "ReleaseAreaBurst")
	worldWriters = append(worldWriters, "SetBurstPhases")
	worldMethods = append(worldMethods, "ImportOriginalDyingActors")
	worldWriters = append(worldWriters, "ImportOriginalDyingActors")
	worldMethods = append(worldMethods, "ConstructSavedCellPlanes", "ConstructSavedStructures", "ImportOriginalActorActions")
	worldWriters = append(worldWriters, "ConstructSavedCellPlanes", "ConstructSavedStructures", "ImportOriginalActorActions")
	worldMethods = append(worldMethods, "StructureOccupancy")
	worldMethods = append(worldMethods, "HasEffectArm", "SpellArm")
	worldMethods = append(worldMethods, "ResetLoadedAreaCosts", "RewriteImportedLayerCosts")
	worldWriters = append(worldWriters, "ResetLoadedAreaCosts", "RewriteImportedLayerCosts")
	// The Diary rules read install-derived rows set once per world.
	worldMethods = append(worldMethods, "SetDiaryUnits", "CopyDiaryUnits", "KnowledgeLevel")
	worldWriters = append(worldWriters, "SetDiaryUnits", "CopyDiaryUnits")
	worldMethods = append(worldMethods, "NativeAreaSaveStates", "NativeCastContinuations", "RandomState", "ImportOriginalActionClocks")
	worldMethods = append(worldMethods, "RestoreRandomState")
	worldMethods = append(worldMethods, "PendingSpellDeliveries")
	worldWriters = append(worldWriters, "RestoreRandomState")
	worldMethods = append(worldMethods, "RandomMode", "SetRandom", "OriginalRand")
	worldWriters = append(worldWriters, "SetRandom", "OriginalRand")
	worldWriters = append(worldWriters, "ImportOriginalActionClocks")
	worldMethods = append(worldMethods, "AutoHealing", "ImportAutoHealing", "SetAutoHealing")
	worldMethods = append(worldMethods, "FrozenGroupAI")
	worldWriters = append(worldWriters, "ImportAutoHealing", "SetAutoHealing")
	worldMethods = append(worldMethods, "CurrentWorldEffectOrder", "NativeSpellDeliverySaveStates", "SavedSpellGraph", "ImportSavedSpellGraph")
	worldWriters = append(worldWriters, "ImportSavedSpellGraph")
	worldMethods = append(worldMethods, "GroupHighWater", "RestoreGroupContinuations", "RestoreCurrentCellCosts")
	worldWriters = append(worldWriters, "RestoreGroupContinuations", "RestoreCurrentCellCosts")
	worldMethods = append(worldMethods, "OriginalDeadActorCount", "RetainOriginalDeadReferences")
	worldWriters = append(worldWriters, "RetainOriginalDeadReferences")
	worldMethods = append(worldMethods, "ReconcileTerminalActorRegistry")
	worldWriters = append(worldWriters, "ReconcileTerminalActorRegistry")
	worldMethods = append(worldMethods, "ReconcileNativeActorRegistry")
	worldWriters = append(worldWriters, "ReconcileNativeActorRegistry")
	worldMethods = append(worldMethods, "ActorTraversal", "RestoreActorTraversal", "RebuildLoadedActorTraversal")
	worldMethods = append(worldMethods, "SetNativeTraining", "NativeTrainingNeedsProducer", "ROM2ScenarioValue", "ROM2ScenarioState", "SetROM2ScenarioState", "SetNativeClass", "RepairNativeSkillLevels")
	worldWriters = append(worldWriters, "SetNativeTraining", "SetNativeClass", "SetROM2ScenarioState", "RepairNativeSkillLevels")
	worldMethods = append(worldMethods, "RestoreNativeActorBases", "RemovedNativeActorBases")
	worldWriters = append(worldWriters, "RestoreNativeActorBases")
	worldMethods = append(worldMethods, "RestoreHeldOrders")
	worldWriters = append(worldWriters, "RestoreHeldOrders")
	worldMethods = append(worldMethods, "RepairNativePackCells")
	worldWriters = append(worldWriters, "RepairNativePackCells")
	worldWriters = append(worldWriters, "RestoreActorTraversal", "RebuildLoadedActorTraversal")
	slices.Sort(worldMethods)
	worldWriters = append(worldWriters, "ImportOriginalStructures", "ImportSavedGroupPlayers", "ImportOriginalActorMotions", "ImportOriginalCellPlanes", "ImportSavedObjects", "SetRawSessionHead", "SetRawSessionMid", "ImportOriginalCellRecords", "SetSavedCellRecords", "SetSavedSpellEffects", "SetSkillLevels", "ImportOriginalProjectiles", "SetSavedProjectiles", "ImportOriginalDiaries", "SetSavedDiaries")
}

// worldArgReaders supplies arguments for every reader that needs them. Arity
// fails closed: an unlisted method cannot silently escape the mutation sweep.
// Entity and owner arguments refer to sample's actual population. StepRate's
// sample has no rated mover; steprate_test.go covers its rated composition.
var worldArgReaders = map[string][]reflect.Value{
	"ROM2ScenarioValue":  {reflect.ValueOf(int32(753))},
	"ActorOrderProgress": {reflect.ValueOf(EntityID(7))},
	"SackValue":          {reflect.ValueOf(int32(2)), reflect.ValueOf(int32(2))},
	"Entity":             {reflect.ValueOf(EntityID(7))},
	"KnowledgeLevel":     {reflect.ValueOf(Entity{})},
	"AutoHealing":        {reflect.ValueOf(uint32(SelfSlot))},
	"FrozenGroupAI":      {reflect.ValueOf(uint32(SelfSlot)), reflect.ValueOf(uint32(0))},
	"AppendCorpseStates": {reflect.ValueOf([]CorpseState(nil))},
	"ActorFinePosition":  {reflect.ValueOf(EntityID(7))},
	"ActorMotionActive":  {reflect.ValueOf(EntityID(7))},
	"WindUpElapsed":      {reflect.ValueOf(EntityID(7))},
	"ProjectilePoint":    {reflect.ValueOf(EntityID(7))},
	"BookSpellRefusal":   {reflect.ValueOf(EntityID(7)), reflect.ValueOf(EntityID(9)), reflect.ValueOf(uint32(1))},
	// The cell form is swept at the caster sample() holds and a cell beside
	// where it stands, so the call reaches the row and the landing predicate
	// rather than stopping at "caster absent".
	"BookSpellCellRefusal": {reflect.ValueOf(EntityID(7)), reflect.ValueOf(int32(4)),
		reflect.ValueOf(int32(5)), reflect.ValueOf(uint32(1))},
	"BoundarySurvivors": {reflect.ValueOf(uint32(1))},
	// The two effect readers (1002). Both are swept at sample()'s own caster
	// and at spell 15, `invisibility`, so the call reaches the attached-effect
	// lookup rather than stopping at "entity absent".
	"HasEffectSpell":       {reflect.ValueOf(EntityID(7)), reflect.ValueOf(uint16(15))},
	"HasEffectArm":         {reflect.ValueOf(EntityID(7)), reflect.ValueOf(uint16(15))},
	"SpellArm":             {reflect.ValueOf(uint16(15))},
	"DropLanding":          {reflect.ValueOf(EntityID(7)), reflect.ValueOf(int32(0)), reflect.ValueOf(int32(0))},
	"InvisibleTo":          {reflect.ValueOf(EntityID(7)), reflect.ValueOf(uint32(1))},
	"CastingSpell":         {reflect.ValueOf(EntityID(7))},
	"FormationMode":        {reflect.ValueOf(uint32(0))},
	"CommandFormationMode": {reflect.ValueOf(uint32(0))},
	"Carried":              {reflect.ValueOf(EntityID(7))},
	"CarriedItems":         {reflect.ValueOf(EntityID(7))},
	"CarriedStacks":        {reflect.ValueOf(EntityID(7))},
	"Equipped":             {reflect.ValueOf(EntityID(7))},
	"EquippedItems":        {reflect.ValueOf(EntityID(7))},
	"FireWallCount":        {reflect.ValueOf(int32(4)), reflect.ValueOf(int32(5))},
	"Purse":                {reflect.ValueOf(uint32(0))},
	"Route":                {reflect.ValueOf(EntityID(1))},
	"SavedSackCellKey":     {reflect.ValueOf(uint16(0))},
	"ScriptRegister":       {reflect.ValueOf(int32(0))},
	"ScriptLatched":        {reflect.ValueOf(int32(0))},
	"ScriptTriggerBlocked": {reflect.ValueOf(int32(0))},
	"Sight":                {reflect.ValueOf(uint32(0))},
	"Spell":                {reflect.ValueOf(uint32(1))},
	"RateSpeed":            {reflect.ValueOf(EntityID(7))},
	"GroupRateTerm":        {reflect.ValueOf(EntityID(7))},
	"StepRate": {reflect.ValueOf(EntityID(7)), reflect.ValueOf(int32(4)),
		reflect.ValueOf(int32(5))},
	"WeaponSpellDamage": {reflect.ValueOf(EntityID(7))},

	"NativeTrainingNeedsProducer": {reflect.ValueOf(EntityID(7))},
}

// worldState is a deep copy of everything a world holds, read straight from the
// unexported fields so that a test of a reader never has to trust that reader.
// The entity slice is copied on purpose: a plain struct copy of a World shares
// its backing array, so a comparison against one could not see a write made
// through it and would pass whatever happened.
type worldState struct {
	tick           uint64
	rng            rng
	bounds         Bounds
	mode           Mode
	grid           []byte
	entities       []Entity
	actorTraversal []EntityID
	routes         [][]cell
}

func snap(w *World) worldState {
	// The routes are copied one level deeper than the other slices, because a
	// route is itself a slice: a shallow copy would share every route with the
	// world and a comparison against one could not see a route being consumed.
	routes := make([][]cell, len(w.routes))
	for i, r := range w.routes {
		routes[i] = append([]cell(nil), r...)
	}
	return worldState{
		tick:           w.tick,
		rng:            w.rng,
		bounds:         w.bounds,
		mode:           w.mode,
		grid:           append([]byte(nil), w.grid...),
		entities:       append([]Entity(nil), w.entities...),
		actorTraversal: slices.Clone(w.actorTraversal),
		routes:         routes,
	}
}

// mustWorld builds the ordinary world of this package's tests: canonical mode,
// no grid, so every cell is passable. Its signature is deliberately the short
// one — a test that has something to say about the mode or the grid says it
// through mustWorldGrid, and the rest are not made to name two arguments they do
// not care about.
func mustWorld(t *testing.T, seed uint64, b Bounds, ents []Entity) *World {
	t.Helper()
	return mustWorldGrid(t, seed, b, ModeCanonical, nil, ents)
}

func mustWorldGrid(t *testing.T, seed uint64, b Bounds, mode Mode, grid []byte, ents []Entity) *World {
	t.Helper()
	w, err := NewWorld(seed, b, mode, grid, ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return w
}

// sample is the shared fixture: three entities given out of id order, two of
// them with a target, one position deliberately negative because nothing here
// constrains a position to the bounds.
func sample() (Bounds, []Entity) {
	return Bounds{Width: 64, Height: 48}, []Entity{
		{ID: 7, X: 3, Y: 4, TargetX: 9, TargetY: 1, HasTarget: true},
		{ID: 2, X: 11, Y: 12},
		{ID: 5, X: -1, Y: 0, TargetX: -6, TargetY: 20, HasTarget: true},
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestNewWorldOrdersByAscendingID(t *testing.T) {
	b, ents := sample()
	forward := mustWorld(t, 1, b, ents)

	reversed := make([]Entity, len(ents))
	for i, e := range ents {
		reversed[len(ents)-1-i] = e
	}
	backward := mustWorld(t, 1, b, reversed)

	got := forward.Entities()
	for i := 1; i < len(got); i++ {
		if got[i-1].ID >= got[i].ID {
			t.Fatalf("ids not strictly ascending: %v", got)
		}
	}
	if want := []EntityID{2, 5, 7}; len(got) != len(want) {
		t.Fatalf("got %d entities, want %d", len(got), len(want))
	} else {
		for i := range want {
			if got[i].ID != want[i] {
				t.Errorf("entity %d has id %d, want %d", i, got[i].ID, want[i])
			}
		}
	}
	if !reflect.DeepEqual(got, backward.Entities()) {
		t.Errorf("insertion order is observable: %v vs %v", got, backward.Entities())
	}
}

func TestNewWorldCopiesTheSliceItWasGiven(t *testing.T) {
	b, ents := sample()
	givenOrder := append([]Entity(nil), ents...)
	w := mustWorld(t, 1, b, ents)
	before := snap(w)

	// The sort ran on the copy, not on what the caller still holds.
	if !reflect.DeepEqual(ents, givenOrder) {
		t.Errorf("NewWorld reordered the caller's slice: %v", ents)
	}

	ents[0] = Entity{ID: 7, X: 999, Y: 999}
	ents[1].ID = 4
	if got := snap(w); !reflect.DeepEqual(got, before) {
		t.Errorf("mutating the constructor's argument reached the world: %+v", got)
	}
}

func TestNewWorldRefusesDuplicateIDs(t *testing.T) {
	b, _ := sample()
	cases := [][]Entity{
		{{ID: 3}, {ID: 3}},
		{{ID: 9}, {ID: 1}, {ID: 9}}, // not adjacent in the order given
	}
	for _, ents := range cases {
		w, err := NewWorld(1, b, ModeCanonical, nil, ents)
		if err == nil {
			t.Errorf("NewWorld(%v) returned no error", ents)
		}
		if w != nil {
			t.Errorf("NewWorld(%v) returned a world alongside its error", ents)
		}
	}
	if _, err := NewWorld(1, b, ModeCanonical, nil, []Entity{{ID: 1}, {ID: 2}}); err != nil {
		t.Errorf("distinct ids refused: %v", err)
	}
}

// gridBounds is the small extent every grid case here is written against: small
// enough that a whole grid can be spelled out by hand and read back.
var gridBounds = Bounds{Width: 4, Height: 3}

// gridSample is a grid over gridBounds with both defined bits in use and neither
// row uniform, so a test that gets the row-major order or the cell count wrong
// cannot pass by symmetry.
func gridSample() []byte {
	return []byte{
		0, 1, 0, 2,
		3, 0, 1, 0,
		0, 2, 0, 1,
	}
}

func TestNewWorldMaterialisesAnAbsentGrid(t *testing.T) {
	_, ents := sample()
	zeroes := make([]byte, gridBounds.Width*gridBounds.Height)

	absent := mustWorldGrid(t, 9, gridBounds, ModeCanonical, nil, ents)
	spelled := mustWorldGrid(t, 9, gridBounds, ModeCanonical, zeroes, ents)

	if !reflect.DeepEqual(snap(absent), snap(spelled)) {
		t.Errorf("a no-grid world and an all-zero-grid world differ:\n %+v\n %+v",
			snap(absent), snap(spelled))
	}
	if got := len(absent.grid); got != len(zeroes) {
		t.Fatalf("the absent grid materialised as %d cell(s), want %d", got, len(zeroes))
	}
	for i, c := range absent.grid {
		if c != 0 {
			t.Errorf("the materialised grid's cell %d is %#02x, want 0", i, c)
		}
	}

	// The same at bounds with no in-bounds cell at all: the count is zero, not a
	// negative product, and the two are still the same world.
	for _, b := range []Bounds{{Width: 0, Height: 5}, {Width: 5, Height: 0}, {Width: -4, Height: -3}} {
		w := mustWorldGrid(t, 9, b, ModeCanonical, nil, nil)
		if len(w.grid) != 0 {
			t.Errorf("bounds %+v gave a grid of %d cell(s), want 0", b, len(w.grid))
		}
		if !reflect.DeepEqual(snap(w), snap(mustWorldGrid(t, 9, b, ModeCanonical, []byte{}, nil))) {
			t.Errorf("bounds %+v: an empty grid and no grid differ", b)
		}
	}
}

func TestNewWorldStoresTheModeAndTheGridItWasGiven(t *testing.T) {
	_, ents := sample()
	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		w := mustWorldGrid(t, 1, gridBounds, mode, gridSample(), ents)
		if w.mode != mode {
			t.Errorf("mode %d was stored as %d", mode, w.mode)
		}
		if !reflect.DeepEqual(w.grid, gridSample()) {
			t.Errorf("mode %d: the grid was stored as % x, want % x", mode, w.grid, gridSample())
		}
	}
	if ModeCanonical != 0 {
		t.Errorf("ModeCanonical is %d: the zero value must be the reconstruction, not our own routing",
			ModeCanonical)
	}
}

func TestNewWorldCopiesTheGridItWasGiven(t *testing.T) {
	_, ents := sample()
	given := gridSample()
	w := mustWorldGrid(t, 1, gridBounds, ModeCanonical, given, ents)
	before := snap(w)

	given[0], given[len(given)-1] = 1, 3
	if got := snap(w); !reflect.DeepEqual(got, before) {
		t.Errorf("mutating the constructor's grid reached the world: % x", got.grid)
	}
}

// TestNewWorldRefusesAMalformedGridOrMode carries each refusal on a case of its
// own, and checks that none of them returns a world: a grid is never
// half-applied, because a construction that refuses hands back nothing to apply
// it to.
func TestNewWorldRefusesAMalformedGridOrMode(t *testing.T) {
	_, ents := sample()
	full := gridSample()

	cases := []struct {
		what string
		b    Bounds
		mode Mode
		grid []byte
	}{
		{"one cell short", gridBounds, ModeCanonical, full[:len(full)-1]},
		{"one cell long", gridBounds, ModeCanonical, append(gridSample(), 0)},
		{"a whole row short", gridBounds, ModeCanonical, full[:8]},
		{"a grid where the bounds hold no cell", Bounds{Width: 0, Height: 3}, ModeCanonical, []byte{0}},
		{"reserved bit 3 set", gridBounds, ModeCanonical, []byte{0, 1, 0, 2, 3, 8, 1, 0, 0, 2, 0, 1}},
		{"reserved bit 7 set", gridBounds, ModeCanonical, []byte{0, 1, 0, 2, 3, 0, 1, 0, 0, 2, 0, 0x80}},
		{"every reserved bit set", gridBounds, ModeCanonical, []byte{0xFC, 1, 0, 2, 3, 0, 1, 0, 0, 2, 0, 1}},
		{"mode 2", gridBounds, Mode(2), gridSample()},
		{"mode 255", gridBounds, Mode(255), gridSample()},
	}
	for _, tc := range cases {
		w, err := NewWorld(1, tc.b, tc.mode, tc.grid, ents)
		if err == nil {
			t.Errorf("%s: NewWorld returned no error", tc.what)
		}
		if w != nil {
			t.Errorf("%s: NewWorld returned a world alongside its error", tc.what)
		}
	}

	// The control: the same bounds, entities and grid with nothing wrong with
	// them are accepted, so the cases above are refused for what they carry and
	// not because this fixture cannot be built at all.
	if _, err := NewWorld(1, gridBounds, ModeOptimised, gridSample(), ents); err != nil {
		t.Errorf("the well-formed control was refused: %v", err)
	}
}

func TestNewWorldZeroesTheResidueOfAClearedTarget(t *testing.T) {
	b, _ := sample()
	w := mustWorld(t, 1, b, []Entity{
		{ID: 0, X: 1, Y: 2, TargetX: 40, TargetY: 41, HasTarget: false},
		{ID: 1, X: 3, Y: 4, TargetX: 40, TargetY: 41, HasTarget: true},
	})
	got := w.Entities()
	if got[0].TargetX != 0 || got[0].TargetY != 0 {
		t.Errorf("a targetless entity kept its coordinates: %+v", got[0])
	}
	if got[1].TargetX != 40 || got[1].TargetY != 41 {
		t.Errorf("a targeted entity lost its coordinates: %+v", got[1])
	}
}

func TestNewWorldTakesItsRNGStateFromTheSeed(t *testing.T) {
	b, ents := sample()
	for _, seed := range []uint64{0, 1, 0x0123456789ABCDEF} {
		w := mustWorld(t, seed, b, ents)
		if w.rng.state != seed {
			t.Errorf("seed %#x gave rng state %#x", seed, w.rng.state)
		}
		if w.tick != 0 {
			t.Errorf("a fresh world starts at tick %d", w.tick)
		}
		if w.bounds != b {
			t.Errorf("bounds %+v, want %+v", w.bounds, b)
		}
	}
}

// TestReadersReportTheWorldsOwnState pins that each reader returns the field
// it names rather than a constant.
func TestReadersReportTheWorldsOwnState(t *testing.T) {
	b, ents := sample()
	w := mustWorld(t, 1, b, ents)

	if w.Tick() != 0 {
		t.Errorf("a fresh world reports tick %d", w.Tick())
	}
	w.tick = 12345
	if w.Tick() != 12345 {
		t.Errorf("Tick() is %d after the field was set to 12345", w.Tick())
	}
	if w.Bounds() != b {
		t.Errorf("Bounds() is %+v, want %+v", w.Bounds(), b)
	}
	w.bounds = Bounds{Width: -3, Height: 7}
	if w.Bounds() != (Bounds{Width: -3, Height: 7}) {
		t.Errorf("Bounds() is %+v after the field was set", w.Bounds())
	}
}

func TestEntitiesHandsOutACopy(t *testing.T) {
	b, ents := sample()
	w := mustWorld(t, 1, b, ents)
	before := snap(w)

	first := w.Entities()
	first[0] = Entity{ID: 2, X: -100, Y: -100, TargetX: 7, TargetY: 7, HasTarget: true}
	if got := snap(w); !reflect.DeepEqual(got, before) {
		t.Errorf("writing through Entities() reached the world: %+v", got)
	}

	second := w.Entities()
	if reflect.DeepEqual(first, second) {
		t.Error("two calls to Entities() share a backing array")
	}
	if !reflect.DeepEqual(second, before.entities) {
		t.Errorf("Entities() returned %v, want %v", second, before.entities)
	}
}

// TestSnapshotIsIndependentOfTheWorld checks the instrument the sweep below uses
// rather than any code under test. If snap shared the world's entity array, then
// from the first call that writes one the sweep would be comparing a mutated
// world against itself, and would pass whatever that call did.
func TestSnapshotIsIndependentOfTheWorld(t *testing.T) {
	b, ents := sample()
	w := mustWorld(t, 1, b, ents)

	before := snap(w)
	wantX := before.entities[0].X
	w.entities[0].X = wantX + 1000

	if reflect.DeepEqual(snap(w), before) {
		t.Fatal("snap() did not see a write straight to the world's entities")
	}
	if before.entities[0].X != wantX {
		t.Errorf("the snapshot followed the world: X is %d, want %d", before.entities[0].X, wantX)
	}
}

// TestReadersDoNotMutateTheWorld sweeps every exported method that is not
// declared a writer and asserts the world is byte-for-byte what it was.
func TestReadersDoNotMutateTheWorld(t *testing.T) {
	b, ents := sample()
	w := mustWorld(t, 0xDEADBEEF, b, ents)
	if err := w.RestoreActorTraversal([]EntityID{7, 2, 5}); err != nil {
		t.Fatal(err)
	}
	before := snap(w)

	typ := reflect.TypeOf(w)
	val := reflect.ValueOf(w)
	swept := 0
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		if contains(worldWriters, m.Name) {
			continue
		}
		// Fail-closed: a method taking arguments cannot be swept blind, so it
		// has to have been declared a writer or given an argument to be swept
		// at first.
		args, declared := worldArgReaders[m.Name]
		if m.Type.NumIn() != 1 && !declared {
			t.Errorf("%s takes arguments and is in neither worldWriters nor worldArgReaders", m.Name)
			continue
		}
		val.Method(i).Call(args)
		swept++
		if got := snap(w); !reflect.DeepEqual(got, before) {
			t.Fatalf("%s mutated the world:\n before %+v\n after  %+v", m.Name, before, got)
		}
	}
	if swept+len(worldWriters) != len(worldMethods) {
		t.Fatalf("swept %d method(s) and skipped %d writer(s), but the pin names %d",
			swept, len(worldWriters), len(worldMethods))
	}
}

func TestWorldExportedMethodSetIsPinned(t *testing.T) {
	typ := reflect.TypeOf(&World{})
	for _, operation := range []string{"EquipSourceCarried", "EquipSourceItem", "UnequipSource"} {
		for _, suffix := range []string{"WithReceipt", "WithTopologyReceipt"} {
			if _, exists := typ.MethodByName(operation + suffix); exists {
				t.Fatalf("%s%s duplicates an equipment operation; pass SourceEquipmentOperation to %s", operation, suffix, operation)
			}
		}
	}
	got := make([]string, 0, typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		got = append(got, typ.Method(i).Name)
	}
	if !reflect.DeepEqual(got, worldMethods) {
		t.Fatalf("*World's exported methods are %v, pinned as %v — a new method must be "+
			"named here and either swept as a reader or listed in worldWriters",
			got, worldMethods)
	}
}

// ---------------------------------------------------------------- the routes

func TestAWorldBeginsWithNoRouteForAnyUnitWhateverBuiltIt(t *testing.T) {
	b, ents := sample()
	grid := make([]byte, gridCells(b))
	grid[3] = blockGround

	for _, tc := range []struct {
		what  string
		build func() *World
	}{
		{"the ordinary constructor", func() *World { return mustWorld(t, 1, b, ents) }},
		{"a world with a grid and a mode", func() *World {
			return mustWorldGrid(t, 2, b, ModeOptimised, grid, ents)
		}},
		{"a world with no entity at all", func() *World { return mustWorld(t, 3, b, nil) }},
		{"a world whose units already hold targets", func() *World {
			return mustWorld(t, 4, b, []Entity{
				{ID: 1, X: 0, Y: 0, TargetX: 5, TargetY: 5, HasTarget: true, Stall: 4},
				{ID: 2, X: 1, Y: 1, TargetX: 2, TargetY: 2, HasTarget: true},
			})
		}},
	} {
		w := tc.build()
		if len(w.routes) != len(w.entities) {
			t.Errorf("%s: %d route slot(s) for %d entities — the two slices are parallel",
				tc.what, len(w.routes), len(w.entities))
			continue
		}
		for i, r := range w.routes {
			if len(r) != 0 {
				t.Errorf("%s: unit %d begins holding the route %s", tc.what, w.entities[i].ID, fmtRoute(r))
			}
		}
	}
}

func TestAnEntityHandedOutReachesNothingOfTheWorld(t *testing.T) {
	typ := reflect.TypeOf(Entity{})
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		switch f.Type.Kind() {
		case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Chan, reflect.Func,
			reflect.Interface, reflect.UnsafePointer:
			t.Errorf("Entity.%s is a %s, so an Entity handed out is not a copy of anything",
				f.Name, f.Type.Kind())
		}
	}

	// And the world's own route slice is not reachable through one: the two
	// slices are parallel, which is a relation the World holds and an Entity
	// knows nothing about.
	w := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, X: 1, Y: 1}})
	w.routes[0] = []cell{{2, 2}}
	got := w.Entities()
	if len(got) != 1 || got[0] != (Entity{ID: 1, X: 1, Y: 1, ActorState: actorStateGuard, Reach: 1, PostX: 1, PostY: 1}) {
		t.Errorf("Entities() returned %+v, want the one unit as it stands", got)
	}
}

func TestTheConstructorFoldsAnOutOfRangeRegenRemainderToZero(t *testing.T) {
	w := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{
		{ID: 1, X: 1, Y: 1}, // names neither
		{ID: 2, X: 2, Y: 2, HealthHundredths: 50, ManaHundredths: 99}, // both in range
		{ID: 3, X: 3, Y: 3, HealthHundredths: 100},                    // health one past the top
		{ID: 4, X: 4, Y: 4, ManaHundredths: 255},                      // mana at the byte's own top
		{ID: 5, X: 5, Y: 5, HealthHundredths: 200, ManaHundredths: 100},
	})
	want := map[EntityID][2]uint8{
		1: {0, 0},
		2: {50, 99},
		3: {0, 0},
		4: {0, 0},
		5: {0, 0},
	}
	for _, e := range w.Entities() {
		if got := [2]uint8{e.HealthHundredths, e.ManaHundredths}; got != want[e.ID] {
			t.Errorf("entity %d carries remainders %v, want %v", e.ID, got, want[e.ID])
		}
	}
}

func TestEntityViewIsTheEntityList(t *testing.T) {
	b, ents := sample()
	w := mustWorld(t, 0xDEADBEEF, b, ents)
	if !reflect.DeepEqual(w.EntityView(), w.Entities()) {
		t.Fatalf("EntityView %+v differs from Entities %+v", w.EntityView(), w.Entities())
	}
}
