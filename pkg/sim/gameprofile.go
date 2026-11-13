package sim

import "againrom/pkg/rules"

// gameProfile is every rule the World decides by the game it runs, one field
// per difference. newGameProfile is its one chooser and the only code that
// asks the game; each branch reads its field. The zero value is the first
// game's profile, as a World with no second-game script state runs the first
// game. A flag names what the second game does. A spell row's Second is no
// profile field: it is the row's own format, the second game's columns, set
// by AssignSecondGameArms where its table is loaded or decoded, and an empty
// table or a cross-game controlled copy keeps the table's own answer.
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

	// nativeTick: the World keeps its native uint64 tick and no session
	// clock; the step ends by raising it and runs no session tick arms.
	nativeTick bool
	// buildingCasters: the map's type-9 mode-4 records build casters.
	buildingCasters bool
	// bankedScript: the second game's script dialect: its checks, instants,
	// message and outcome rules, and an inert bank without a program.
	bankedScript bool
	// activityMask: Group and actor activity come from the second game's
	// script activity mask; no first-game activity cover holds an actor.
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
	// nativeMovers: no actor holds a Mover record; byte forms carrying one
	// are refused, and the flee search keeps ScanRange (DIV-347).
	nativeMovers bool
	// nativeCellPlanes: an older engine SAV without the plane mark drops the
	// cell planes its LOAD import built.
	nativeCellPlanes bool
	// nativeObjects: a World without the saved-object registry needs no
	// training producer.
	nativeObjects bool
	// nativePointCasts: no stored Teleport in flight is admitted on LOAD.
	nativePointCasts bool
	// nativeTargetOrders: target orders are kept as held: no LOAD
	// normalisation (DIV-3310) and no stop when the victim leaves.
	nativeTargetOrders bool
	// nativeShopUse: a shop structure takes the ordinary use, not the
	// recorded shop use.
	nativeShopUse bool

	// nativeWalkRefusal keeps near-search give-up separate from a known
	// empty-route result (R2-ENGINE-1108).
	nativeWalkRefusal bool
	// cellCastReach: the automatic cast's candidate test and a scroll cast
	// at a unit compare whole Chebyshev cells (R2-ENGINE-348), not the
	// footprint gap. A book order at a unit keeps the footprint gap
	// (R2-ENGINE-1143).
	cellCastReach bool
	// releaseRevalidates: a release asks the whole admission again: reach
	// and range (MAGIC-403 states the first game asks none).
	releaseRevalidates bool
	// paysFacingAim: a book cast at another unit or a cell pays when its
	// wind-up starts facing the aim, not at the order (R2-ENGINE-543,
	// R2-ENGINE-544, R2-ENGINE-926).
	paysFacingAim bool
	// blowOrdersAttack: an area blow orders a victim of another owner against
	// the caster (DIV-1600); otherwise it runs the blow hook (answerAreaBlow).
	blowOrdersAttack bool
	// bookWalkAll: the engage book walk takes every row and has no fallback.
	bookWalkAll bool
	// postKeepsDying: a dying body stands on the guard post (AI-704).
	postKeepsDying bool
	// savedProjectileAim: a projectile aims at the saved target point, not
	// the mover's fine point (AI-674).
	savedProjectileAim bool
	// withdrawMoverPoint: the withdrawal measures from the mover's fine
	// point, not the saved target point.
	withdrawMoverPoint bool
	// withdrawAnchorCells: the withdrawal square lists each hostile once by
	// its anchor cell (AI-697).
	withdrawAnchorCells bool
	// classThresholds: a computer Human's Withdraw and Wimpy come from its
	// class at mission start (classThresholds).
	classThresholds bool
	// keepsSuppressedAcquirer: the engage refuses no suppressed acquirer
	// (AI-GUARD-007).
	keepsSuppressedAcquirer bool
	// noHealArmReach: no unit takes the heal arm's reach rule (MAGIC-320).
	noHealArmReach bool
	// castsUnseen: a book cast at a unit asks no sight of its caster or of
	// a computer mage's Group, at the admission or the release; the spell's
	// range alone bounds it (R2-ENGINE-1136, R2-ENGINE-1143).
	castsUnseen bool
	// autoCastSideSight: the automatic choice takes an actor a living unit of
	// the caster's player or of a player allied to it sees for a cast, not
	// the cells the caster lights (autoCastObservers); its cast's admission
	// and release ask none (owner, DIV-4872).
	autoCastSideSight bool
	// keepsWeaponSpell: a worn Weapon always holds its spell, and an
	// unequip keeps it (MAGIC-379).
	keepsWeaponSpell bool
	// autoCastFullDefault: a Player without a stored setting reads
	// DefaultAutoCast (DIV-3007); the first game reads Off (DIV-3006).
	autoCastFullDefault bool
	computerSupport     supportCastPolicy
	// computerHealOwnSide: a computer mage's automatic heal reaches its own
	// owner and allied owners alone (DIV-4756).
	computerHealOwnSide bool
	// spareMageTypes: the creature type ids an arm that spares mages spares
	// beside a human with a mana pool (sparedMage).
	spareMageTypes []int32
	// nativeRegen: regeneration runs as a pass of the step, and each actor
	// heals on its own phase (regenPass).
	nativeRegen bool
	// nativeDecay: the decay ladder walks held dead and decaying health on
	// the native tick; no session-clock dead ladder runs.
	nativeDecay bool

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
	// cellEventRelocates: instant 25 stores a source and target, and a cell
	// event of the Teleport kind moves the unit (R2-ENGINE-673, DIV-4700).
	cellEventRelocates bool
	// bookListsScrolls: the book shows a carried scroll at its spell's cell
	// (R2-ENGINE-766, R2-SESSION-229, DIV-4732).
	bookListsScrolls bool
	// memberScoreGate: the score's preference modifiers test the scoring
	// actor's size and Mind (R2-ENGINE-503), not the candidate's.
	memberScoreGate bool
	// blow: the blow's draws and gates (resolveBlow).
	blow blowRule
	// skillAward: the skill award's steps (R2-ENGINE-480, R2-ENGINE-486).
	skillAward skillAwardProfile
	orders     orderProfile
}

// newGameProfile chooses the profile of a World running the script dialect
// d. It runs on every path that builds a World: mission start and the
// controlled script copy (initializeROM2Script), and the decoder (the base
// form and the second game's wrapper).
func newGameProfile(d ScriptDialect) gameProfile {
	if d != ScriptROM2 {
		return gameProfile{}
	}
	return gameProfile{
		spellGame:                  rules.SpellGameSecond,
		groupRoute:                 secondGameGroupRoute,
		creatureAim:                secondGameCreatureAim,
		suppressedRewards:          true,
		fillsEmptyCreatureBook:     true,
		deathRollAt64:              true,
		deathItemRoll:              true,
		nativeTick:                 true,
		buildingCasters:            true,
		bankedScript:               true,
		activityMask:               true,
		cullClearsActionClock:      true,
		formHoldsConstructedGroups: true,
		keepsHealthAboveMaximum:    true,
		cellEventRelocates:         true,
		bookListsScrolls:           true,
		memberScoreGate:            true,
		blow:                       secondGameBlow,
		skillAward:                 secondGameSkillAward,
		nativeEffectDriver:         true,
		nativeMovers:               true,
		nativeCellPlanes:           true,
		nativeObjects:              true,
		nativePointCasts:           true,
		nativeTargetOrders:         true,
		nativeShopUse:              true,
		nativeWalkRefusal:          true,
		cellCastReach:              true,
		releaseRevalidates:         true,
		paysFacingAim:              true,
		blowOrdersAttack:           true,
		bookWalkAll:                true,
		postKeepsDying:             true,
		savedProjectileAim:         true,
		withdrawMoverPoint:         true,
		withdrawAnchorCells:        true,
		classThresholds:            true,
		keepsSuppressedAcquirer:    true,
		noHealArmReach:             true,
		castsUnseen:                true,
		autoCastSideSight:          true,
		keepsWeaponSpell:           true,
		autoCastFullDefault:        true,
		computerSupport:            supportCastPolicy{minimumMana: 11, requiresHeal: true, selfOnly: true, idleOnly: true},
		computerHealOwnSide:        true,
		spareMageTypes:             secondGameSpareMageTypes,
		nativeRegen:                true,
		nativeDecay:                true,
		keepsCastRetain:            true,
		residueOverBookCasts:       true,
		savesMissionFog:            true,
		noCompanionObjective:       true,
		orders: orderProfile{
			defendFineStep:           true,
			collectSacks:             true,
			idleRanking:              targetIdleDistance,
			liveAdmission:            true,
			mageAcquisitionException: true,
			stopEmptyWalk:            true,
			standGroundScore:         true,
			lookAround:               true,
			repickEmptyRoute:         true,
			primaryCells:             true,
			supportOrder:             true,
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
func (w *World) NativeMovers() bool { return w.game.nativeMovers }

// KeepsCastRetain reports whether a SAVE keeps the retain flag of a manual
// walk's armed cast order.
func (w *World) KeepsCastRetain() bool { return w.game.keepsCastRetain }

// ResidueOverBookCasts reports whether a SAVE holds the order residue of an
// actor with a current book cast too.
func (w *World) ResidueOverBookCasts() bool { return w.game.residueOverBookCasts }

// SavesMissionFog reports whether a SAVE writes the current mission's fog
// residue.
func (w *World) SavesMissionFog() bool { return w.game.savesMissionFog }

// RestoresCompanionObjective reports whether a current-SAV LOAD restores a
// mission companion objective into the script.
func (w *World) RestoresCompanionObjective() bool { return !w.game.noCompanionObjective }
