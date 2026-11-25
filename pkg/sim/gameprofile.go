package sim

import (
	"againrom/pkg/rules"
	"againrom/pkg/scriptprofile"
)

// gameProfile is every rule the World decides by the game it runs, one field
// per difference. newGameProfile is its one chooser and the only code that
// asks the game; each branch reads its field. The zero value is the first
// game's profile, as a World with no second-game script state runs the first
// game. A flag names what the second game does. The spell rules a World
// applies are the spellRow its spellGame selects, not a field of each row.
type gameProfile struct {
	// spellGame: the game whose spell effect rules the World's driver takes.
	spellGame rules.SpellGame
	// groupRoute: the saved Group AI's routes and handler rules.
	groupRoute groupRouteProfile
	// creatureAim: the creature slot cast's aims and planner gates.
	creatureAim            creatureAimProfile
	suppressedRewards      bool
	fillsEmptyCreatureBook bool
	// deathRollAt64: the death roll admits type id 0x40 (R2-ENGINE-940).
	deathRollAt64 bool
	// deathItemRoll: the death roll draws the item roll after the gold.
	deathItemRoll bool

	// tickClock: session clock or native uint64 tick (tickClocks).
	tickClock tickClockRule
	// buildingCasters: the map's type-9 mode-4 records build casters.
	buildingCasters buildingCasterRule
	// scriptRows: the second game's script op bodies, added to the shared ones;
	// a dialect without rows runs the shared bodies only.
	scriptRows scriptOpRows
	// dialect is the script dialect this profile serves.
	dialect ScriptDialect
	// holdsScriptState: the World holds the second game's script state.
	holdsScriptState bool
	// dropsEmptyScript: a World started with an empty script holds none.
	dropsEmptyScript bool
	// scriptSeed is the non-zero register written at session start
	// (R2-SESSION-265); zero value writes none.
	scriptSeed scriptRegisterSeed
	// structureOwners: the byte form keeps structure owners; only the second
	// game's check 20 reads them.
	structureOwners bool
	// scriptCount: unit and member presence checks 1, 5 and 8
	// (scriptCountRules, R2-ENGINE-1696).
	scriptCount scriptCountRule
	// outcomeReport: how the outcome reads the lose and win counters
	// (outcomeReports).
	outcomeReport outcomeRule
	// groupActivity: how the engagement pass refreshes Group activity
	// (groupActivityRefreshers).
	groupActivity groupActivityRule
	// activityMask: actor and executor activity come from the second game's
	// script activity mask rather than first-game activity covers.
	activityMask bool
	// cullClearsActionClock: the mission-return cull clears the action-run
	// end; the first game keeps the one the mission set (SAV-1353).
	cullClearsActionClock bool

	// formHoldsConstructedGroups: the byte form writes the constructed Group
	// registry too. The second game's form has no session clock to name an
	// older engine's, so a form without a registry is one written before its
	// mission start built one, and its Groups take their native records
	// (constructedGroups, DIV-4502).
	formHoldsConstructedGroups bool
	// nativeEffectDriver: the world-effect driver table is created by the
	// first projectile, not at construction.
	nativeEffectDriver bool
	// movers: no actor holds a Mover record; byte forms carrying one
	// are refused, and the flee search keeps ScanRange (DIV-347).
	movers moverRule
	// cellPlanes: an older engine SAV without the plane mark drops the
	// cell planes its LOAD import built.
	cellPlanes cellPlaneRule
	// nativeObjects: a World without the saved-object registry needs no
	// training producer.
	nativeObjects bool
	// nativePointCasts: no stored Teleport in flight is admitted on LOAD.
	nativePointCasts bool
	// nativeTargetOrders: target orders are kept as held: no LOAD
	// normalisation (DIV-3310) and no stop when the victim leaves.
	nativeTargetOrders bool
	// shopUse: a shop structure takes the ordinary use, not the
	// recorded shop use.
	shopUse shopUseRule

	// nativeWalkRefusal keeps near-search give-up separate from a known
	// empty-route result (R2-ENGINE-1108).
	nativeWalkRefusal bool
	// castReach: unit-target reach rule for automatic and scroll casts
	// (castReachRules, R2-ENGINE-348, R2-ENGINE-1143).
	castReach castReachRule
	// castRecovery: cast recovery narrowing rule (castRecoveryRules,
	// R2-ENGINE-1261).
	castRecovery castRecoveryRule
	// castStamp: the activity stamp a started cast stores (castStamps).
	castStamp castStampRule
	// flight: the transports that wait the Euclidean distance (flightsBySpeed).
	flight flightRule
	// paysFacingAim: a book cast at another unit or a cell pays when its
	// wind-up starts facing the aim, not at the order (R2-ENGINE-543,
	// R2-ENGINE-544, R2-ENGINE-926).
	paysFacingAim bool
	// castPay: cast debit order against application refusals (castPayRules).
	castPay castPayRule
	// blowRelation: reverse-hostility gate on area blows (blowRelationRules).
	blowRelation blowRelationRule
	// rayReach: the range bound on Prismatic Spray's side targets
	// (rayReachRules, R2-ENGINE-1766).
	rayReach rayReachRule
	// chargeZero: how long a charge byte of zero lasts (chargeZeroTicks,
	// R2-ENGINE-1763).
	chargeZero chargeZeroRule
	// bookWalkNoFallback: the engage book walk has no fallback row.
	bookWalkNoFallback bool
	// postKeepsDying: a dying body stands on the guard post (AI-704).
	postKeepsDying bool
	// projectileAim: the point a projectile aims at a unit
	// (projectileAimPoints, AI-674).
	projectileAim projectileAimRule
	// areaHeading: where an area's heading comes from (areaHeadings).
	areaHeading areaHeadingRule
	// fleeOrigin: the point the withdrawal measures a unit from (fleePoints).
	fleeOrigin fleeOriginRule
	// castDropsLoadedStep: a manual cast or cast approach ends a loaded
	// step between cells instead of waiting for its centre.
	castDropsLoadedStep bool
	// withdrawList: the withdrawal square lists each hostile once by
	// its anchor cell (AI-697).
	withdrawList withdrawListRule
	// retreatMode: the rule that turns a retreat setting into thresholds
	// (retreatModeRules).
	retreatMode retreatModeRule
	// refusedFlee: the answer to a withdrawal move whose far search found no
	// route (refusedFleeAnswers).
	refusedFlee refusedFleeRule
	// supportAffinity: the support target setting's rule (supportAffinities).
	supportAffinity supportAffinityRule
	// slope: the rounding of the ground rate's slope term (slopeRules).
	slope slopeRule
	// fleeSightExtra: the Wimpy test's hostile square is this many rings wider
	// than the mover's search radius (R2-ENGINE-506); the first game adds none.
	fleeSightExtra int64
	// fleeBlockRadius: the unit block's constant radius byte that stands for
	// the mover radius in the flee test and in the Teleport arm's hostile list
	// (R2-ENGINE-1438, R2-ENGINE-1440); zero keeps the mover radius.
	fleeBlockRadius int64
	// fleeTeleports: a unit that books Teleport and can pay it casts it
	// to flee or back off, and walks otherwise (R2-ENGINE-1418).
	fleeTeleports bool
	// keepsSuppressedAcquirer: the engage refuses no suppressed acquirer
	// (AI-GUARD-007).
	keepsSuppressedAcquirer bool
	// noHealArmReach: no unit takes the heal arm's reach rule (MAGIC-320).
	noHealArmReach bool
	// castSight: a book cast at a unit asks no sight of its caster or of
	// a computer mage's Group, at the admission or the release; the spell's
	// range alone bounds it (R2-ENGINE-1136, R2-ENGINE-1143).
	castSight castSightRule
	// autoCastSight: whose sight the automatic choice takes (autoCastSights).
	// Under the side's sight its cast's admission and release ask none
	// (owner, DIV-4872).
	autoCastSight autoCastSightRule
	// weaponSpell: a worn Weapon always holds its spell, and an
	// unequip keeps it (MAGIC-379).
	weaponSpell weaponSpellRule
	// autoCastDefault: a Player without a stored setting reads
	// DefaultAutoCast (DIV-3007); the first game reads Off (DIV-3006).
	autoCastDefault autoCastDefaultRule
	computerSupport supportCastPolicy
	// computerHealOwnSide: a computer mage's automatic heal reaches its own
	// owner and allied owners alone (DIV-4756).
	computerHealOwnSide bool
	// spareMageTypes: the creature type ids an arm that spares mages spares
	// beside a human with a mana pool (sparedMage).
	spareMageTypes []int32
	// nativeRegen: regeneration runs as a pass of the step, and each actor
	// heals on its own phase (regenPass).
	nativeRegen bool
	// regenLife: the life state the pool pass restores a unit in. The second
	// game restores nothing at health zero or below, an undying unit
	// included (R2-ENGINE-1207).
	regenLife LifeState
	// regenBodyLoss: the pool pass takes one health from a unit between -9 and
	// -1 on every fourth pass (R2-ENGINE-1207).
	regenBodyLoss bool
	// decayWalk: the decay pass's walk rule (decayWalks).
	decayWalk decayWalkRule
	// deadList: a body's teardown ranks it on the dead list, which Animate
	// Dead walks (joinDeadList, R2-ENGINE-780).
	deadList bool

	// keepsCastRetain: a SAVE keeps the retain flag of a manual walk's cast
	// order (MAGIC-253 states the first game clears it).
	keepsCastRetain bool
	// residueOverBookCasts: a SAVE holds the order residue of an actor with
	// a current book cast too.
	residueOverBookCasts bool
	// savesMissionFog: a SAVE writes the current mission's fog residue.
	savesMissionFog bool
	// noCompanionObjective: a current-SAV LOAD restores no mission companion
	// objective into the script; the first game restores its VIP check.
	noCompanionObjective    bool
	keepsHealthAboveMaximum bool
	// cellTail: how instant 25 encodes its six tail bytes (cellTailEncoders).
	cellTail cellTailRule
	// cellEvent: cell event 0x17 moves the unit whose step ends centred
	// on the cell (R2-ENGINE-1778, DIV-4700).
	cellEvent cellEventRule
	// bookListsScrolls: the book shows a carried scroll at its spell's cell
	// (R2-ENGINE-766, R2-SESSION-229, DIV-4732).
	bookListsScrolls bool
	// scoreGate: the score's preference modifiers test the scoring
	// actor's size and Mind (R2-ENGINE-503), not the candidate's.
	scoreGate scoreGateRule
	// scoreEffectSpell: the lasting effect whose bearer the ordinary score
	// adds scoreEffectCost for, 0 for none (AI-COST-071, R2-ENGINE-1495,
	// R2-ENGINE-1496).
	scoreEffectSpell uint16
	// blow: the blow's draws and gates (resolveBlow).
	blow blowRule
	// skillAward: the skill award's steps (R2-ENGINE-480, R2-ENGINE-486).
	skillAward skillAwardProfile
	orders     orderProfile
	// diary: a kill is counted and a card is graded for any kind
	// whose Units row has a server id in 1..0x9ff, with no type, face, row
	// or humanoid window. The zero value is the first game's windows
	// (R2-ENGINE-772, R2-ENGINE-1710, R2-ENGINE-1712).
	diary diaryRule
	// diaryCountCap: the largest kill count the Diary stores; zero is the
	// first game's DiaryCountCap.
	diaryCountCap uint32
	// showMapFullCard: a card is drawn at the full level while the
	// show-map cheat is on (R2-ENGINE-1269).
	showMapFullCard bool
}

type scriptRegisterSeed struct{ register, value int32 }

var dialectProfiles = [...]scriptprofile.Profile{
	ScriptROM1: scriptprofile.For(scriptprofile.OpsFirst, false),
	ScriptROM2: scriptprofile.For(scriptprofile.OpsSecond, false),
}

// Profile returns the script profile of dialect d.
func (d ScriptDialect) Profile() scriptprofile.Profile {
	if int(d) < len(dialectProfiles) {
		return dialectProfiles[d]
	}
	return dialectProfiles[ScriptROM1]
}

// Known reports whether d is a dialect the engine runs.
func (d ScriptDialect) Known() bool { return int(d) < len(dialectProfiles) }

// HoldsScriptState reports whether the game of dialect d keeps the second
// game's script state: its gameProfile's holdsScriptState.
func (d ScriptDialect) HoldsScriptState() bool { return newGameProfile(d).holdsScriptState }

// scriptLimits returns the script profile of g: the dialect table's row.
func (g gameProfile) scriptLimits() scriptprofile.Profile {
	return g.dialect.Profile()
}

// diaryCap is the largest kill count the profile's Diary stores.
func (g gameProfile) diaryCap() uint32 {
	if g.diaryCountCap == 0 {
		return DiaryCountCap
	}
	return g.diaryCountCap
}

// secondGameDiaryCountCap is the second game's kill count ceiling: a count
// stops at 14, which grades 7 (R2-ENGINE-1269, R2-ENGINE-1272).
const secondGameDiaryCountCap = 14

// newGameProfile chooses the profile of a World running the script dialect
// d. It runs on every path that builds a World: mission start and the
// controlled script copy (initializeROM2Script), and the decoder (the base
// form and the second game's wrapper).
func newGameProfile(d ScriptDialect) gameProfile {
	if d != ScriptROM2 {
		return gameProfile{
			dialect:          ScriptROM1,
			dropsEmptyScript: true,
			scoreEffectSpell: 20,
			skillAward:       firstGameSkillAward,
			computerSupport:  supportCastPolicy{minimumMana: 11, requiresHeal: true, idleOnly: true, pick: supportPicksDrawn, hold: supportHeldInCombat, shield: shieldNone},
		}
	}
	return gameProfile{
		spellGame:                  rules.SpellGameSecond,
		groupRoute:                 secondGameGroupRoute,
		creatureAim:                secondGameCreatureAim,
		suppressedRewards:          true,
		fillsEmptyCreatureBook:     true,
		deathRollAt64:              true,
		deathItemRoll:              true,
		tickClock:                  nativeTickClock,
		buildingCasters:            buildingCastersBuilt,
		scriptRows:                 secondGameRows,
		dialect:                    ScriptROM2,
		holdsScriptState:           true,
		scriptSeed:                 scriptRegisterSeed{register: 901, value: 1},
		structureOwners:            true,
		scriptCount:                scriptCountListed,
		outcomeReport:              outcomeAtNonzero,
		groupActivity:              groupActivityByMask,
		activityMask:               true,
		cullClearsActionClock:      true,
		formHoldsConstructedGroups: true,
		keepsHealthAboveMaximum:    true,
		cellTail:                   cellTailTargeted,
		cellEvent:                  cellEventRelocating,
		bookListsScrolls:           true,
		scoreGate:                  scoreGateMember,
		scoreEffectSpell:           18,
		blow:                       secondGameBlow,
		skillAward:                 secondGameSkillAward,
		nativeEffectDriver:         true,
		movers:                     moversNative,
		cellPlanes:                 cellPlanesNative,
		nativeObjects:              true,
		nativePointCasts:           true,
		nativeTargetOrders:         true,
		shopUse:                    shopUseOrdinary,
		nativeWalkRefusal:          true,
		castReach:                  castReachCell,
		castRecovery:               castRecoveryLowByte,
		castStamp:                  castStampsRoutine,
		flight:                     flightEveryRow,
		paysFacingAim:              true,
		castPay:                    castPayBeforeRefusal,
		blowRelation:               blowRelationAreaGated,
		rayReach:                   rayReachKeyed,
		chargeZero:                 chargeZeroWraps,
		bookWalkNoFallback:         true,
		postKeepsDying:             true,
		projectileAim:              aimAtSavedPoint,
		areaHeading:                areaHeadingCells,
		fleeOrigin:                 fleeFromCentre,
		castDropsLoadedStep:        true,
		withdrawList:               withdrawListAnchor,
		retreatMode:                retreatModeByClass,
		refusedFlee:                refusedFleeStops,
		supportAffinity:            supportAffinityRelation,
		slope:                      slopeShiftsMagnitude,
		fleeSightExtra:             5,
		fleeBlockRadius:            5,
		fleeTeleports:              true,
		keepsSuppressedAcquirer:    true,
		noHealArmReach:             true,
		castSight:                  castSightUnasked,
		autoCastSight:              autoCastAlliedSight,
		weaponSpell:                weaponSpellKept,
		autoCastDefault:            autoCastDefaultFull,
		computerSupport:            supportCastPolicy{minimumMana: 11, requiresHeal: true, idleOnly: true, pick: supportPicksDrawn, yieldsToWithdrawal: true, shield: shieldAfterCandidateHeal},
		computerHealOwnSide:        true,
		spareMageTypes:             secondGameSpareMageTypes,
		diary:                      diaryServerKeyed,
		diaryCountCap:              secondGameDiaryCountCap,
		showMapFullCard:            true,
		nativeRegen:                true,
		regenLife:                  LifeStanding,
		regenBodyLoss:              true,
		decayWalk:                  nativeDecayWalk,
		deadList:                   true,
		keepsCastRetain:            true,
		residueOverBookCasts:       true,
		savesMissionFog:            true,
		noCompanionObjective:       true,
		orders: orderProfile{
			defendStep:       defendStepFine,
			collectSacks:     true,
			idleRanking:      targetIdleDistance,
			attackReach:      attackReachLive,
			acquisition:      acquisitionMageException,
			stopEmptyWalk:    true,
			standGroundScore: true,
			lookAround:       lookAroundOn,
			emptyRoute:       emptyRouteRepicks,
			primaryCells:     true,
			supportOrder:     true,
		},
	}
}

// paysNoReward applies the profile-gated reward mark (R2-ENGINE-535).
func (w *World) paysNoReward(e Entity) bool {
	return w.game.suppressedRewards && e.SuppressRewards
}

// SpellGame is the game whose spell effect rules this World's driver takes.
func (w *World) SpellGame() rules.SpellGame { return w.game.spellGame }

// NativeMovers reports a World whose actors hold no Mover record, so a SAVE
// writes no constructed mover byte for a body the World no longer holds.
func (w *World) NativeMovers() bool { return w.game.movers == moversNative }

// KeepsCastRetain reports whether a SAVE keeps the retain flag of a manual
// walk's armed cast order.
func (w *World) KeepsCastRetain() bool { return w.game.keepsCastRetain }

// ResidueOverBookCasts reports whether a SAVE holds the order residue of an
// actor with a current book cast too.
func (w *World) ResidueOverBookCasts() bool { return w.game.residueOverBookCasts }

// SavesMissionFog reports whether a SAVE writes the current mission's fog
// residue.
func (w *World) SavesMissionFog() bool { return w.game.savesMissionFog }

// ShowMapFullCard reports whether the show-map cheat draws every card at the
// full level.
func (w *World) ShowMapFullCard() bool { return w.game.showMapFullCard }

// RestoresCompanionObjective reports whether a current-SAV LOAD restores a
// mission companion objective into the script.
func (w *World) RestoresCompanionObjective() bool { return !w.game.noCompanionObjective }
