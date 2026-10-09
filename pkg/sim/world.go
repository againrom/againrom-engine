package sim

import (
	"fmt"
	"sort"
)

// EntityID identifies one entity inside one world. Ids are assigned by whoever
// builds the world and are unique within it; they are not an identity any map
// record carries, so a world cannot be matched back against one.
type EntityID uint32

// skillSlots is how many per-slot experience integers an entity carries:
// this package's own mirror of pkg/data's SkillSlots (six, not five and not
// ten — that constant's own doc says why). The two cannot be one symbol:
// pkg/sim is held to STDLIB-ONLY imports, tests included — the determinism
// wall's structural half, enforced by internal/archtest's import check
// (AGENTS.md) — and pkg/data is not stdlib, so reading the count off it
// would breach the same wall a float import already cannot cross. The two
// constants are kept in step by provenance.md rather than by the compiler;
// pkg/mapload, which imports both, is where a divergence between them would
// first become visible — a slot pkg/data derives and this package refuses.
const skillSlots = 6

// Entity is one simulated thing: a stable id, a whole-cell position, and an
// optional move target.
//
// X is the column and Y the row on the map's row-major lattice from (0,0). Every
// field is an integer or a bool — no pointer, no slice, no float — so an Entity
// is copied by assignment and a caller handed one holds nothing that still
// reaches into a world.
//
// TargetX and TargetY mean nothing while HasTarget is false, and a world stores
// them as zero there (see NewWorld): a target may name any cell, so no
// coordinate value is free to stand for "none", and a cleared target must leave
// no residue behind for two otherwise identical worlds to differ by.
//
// Class is the entity's class id: an opaque signed 32-bit key set at
// construction or by unmarshalling and by nothing else. It is canonical state —
// it enters the byte form and the digest — but it is inert: a step neither
// reads, changes nor branches on it, and this package never interprets it.
// What the key resolves to is a question for whoever draws the entity.
//
// Domain is the movement domain this entity moves in: which terrain closes a
// cell to it, and which of the two occupancy layers it contends on. It is
// canonical state — it enters the byte form and the digest — and unlike Class it
// is READ by a step, at every offer a search makes.
//
// DomainGround is the zero value, so an entity built without naming one is an
// ordinary ground mover and every world assembled before this field existed
// keeps the behaviour it had.
//
// Stall counts the consecutive ticks this entity has held a target it could not
// advance toward. It is canonical state, so a world resumed from its bytes gives
// up exactly where the world it was cut from would have. Legal values are 0 to
// stallLimit-1: the count that reaches the limit is spent inside the tick that
// raised it, so no stored world carries one at or above it. Like the target
// coordinates it leaves no residue — an entity holding no target holds zero.
//
// HP and MaxHP are the entity's health and the maximum it was built with, and
// between them they are the WHOLE of its life state: THERE IS NO DEATH FLAG.
// Dead, Downed and Alive below are comparisons on these two and on nothing else,
// so a unit killed outright and one damaged to death are one state by
// construction and there is no second field for either to disagree with.
//
// HP is SIGNED AND IS NEVER CLAMPED. A killed unit's health is negative and
// stays negative, and how far below zero it has gone is state a later story
// reads: clamping it at zero would collapse every distinction below that line
// while looking, here, like tidiness.
//
// A MaxHP of zero or less is an entity with NO HEALTH SYSTEM — alive, immune to
// damage, still killable — which is what leaves a world built with neither field
// named in exactly the state, the byte form and the digest it had before they
// existed.
//
// Speed is the mover's own rate input: the per-class number the decoded rate law
// composes with the ground it is crossing. It is canonical state and it is READ
// by a step, at the start of every cell transit.
//
// A SPEED OF ZERO OR LESS IS A MOVER WITH NO RATE — one cell per tick, which is
// what every mover in this package did before a rate existed. That is ours and
// it is a widening rule, not the law's answer: the law would clamp such a speed
// to the rate floor and crawl the mover at a cell per maxTransit ticks, and
// every world built here without naming a speed names none of them. It is the
// same trade MaxHP makes above, and rated is the one predicate that decides it.
//
// Transit and TransitTotal are where the mover is between two cells: how many
// ticks of the current cell transit are still owed, and how many the whole
// transit is. A mover owing ticks neither searches, steps nor takes a route —
// it has already taken its cell and is paying for it.
//
// BOTH ARE CANONICAL, and the pair rather than one number. The original holds
// both — the ticks a transit needs and the ticks it has run — in the mover block
// its own save writes whole; and the owed count alone cannot say how far through
// a transit a mover is, which is exactly what a drawn body between two cells
// asks. The two are legal only as a pair: a total of zero means no transit at
// all, and an owed count always sits strictly below its total.
//
// A total OUTLIVES the transit it measured, staying put until the next transit
// overwrites it, and that is deliberate: the tick that brings the owed count to
// zero is the last tick of the crossing, and a total cleared there would leave
// that tick with nothing to measure the mover's position against.
//
// GroupSpeed is THE GROUP RATE TERM: the minimum speed of the group this entity
// was last ordered with, as that group stood at the moment the order was issued,
// and zero when it carries none. A nonzero one REPLACES Speed wherever a rate is
// computed — see moverSpeed, which is the only thing that reads it.
//
// IT IS NOT CLEARED BY ARRIVING, by giving up, or by another member's death, and
// that is not an oversight: it is the behaviour being reproduced. Its whole
// writer set is three sites and there is no fourth — a group order zeroes it for
// every member and then sets it on the formation arm, a plain move order zeroes
// it, and being felled zeroes it. So a group that has finished walking still
// carries its slowest member's speed, and still carries a dead member's.
//
// AttackTarget, AttackTargetKind, HasAttackTarget, AcquirePursuit, AttackPhase
// and AttackCountdown hold an attack order and its cycle.
//
// AttackCharge, AttackRelax, ToHit, Defence, Absorption, DamageBase and
// DamageSpread are the seven numbers a cycle and a resolution read, and
// AlwaysHits is the mark beside them. Each is CANONICAL and each is READ by a
// step — the first two at every phase change, the other five at every blow. They
// are int32 and range checked NOWHERE, exactly as Speed and the health pair are:
// every value is a state the constructor accepts, so refusing one here would make
// a world this package can produce a world it cannot read back.
//
// Group is the group its MAP PLACED IT IN, and it is not a rate: GroupSpeed
// above is a speed a formation order wrote and this is membership, which no
// order writes and nothing in this package ever changes. The two sit beside each
// other because that adjacency is the confusion worth pre-empting — one is
// spent when an order ends, the other outlives the order, the death and the
// save. A script's group check reads THIS one.
//
// It is the placed record's own word at its own width, and its ZERO IS A REAL
// GROUP rather than an absence: five units of the shipped corpus carry it, so no
// value is free to stand for "none". An entity built from no record carries that
// zero and is a member of group zero, which costs nothing — a check naming a
// group no entity carries answers zero either way.
//
// It is held on the MEMBER rather than on a group object the world owns. What is
// being reconstructed hangs the list off a player, and the two are the same
// observable behaviour here for the reason the rate term's doc gives: nothing in
// this tree adds a member, removes one, or moves one between groups. A story
// that can do any of the three moves the field and its readers together.
//
// Owner is the ROSTER SLOT its map placed it under, and it is the group word's
// near neighbour in every way but one. It is the placed record's own word at its
// own width, it is membership rather than residue, and a script arm writes it —
// the two hand-over arms are the only writers in this package.
//
// ITS ZERO IS NO OWNER, which is the exact opposite of the group word above, and
// the two are opposite because the two identifier spaces are: a roster slot is
// 1-based, so zero cannot name one and is free to mean absent without costing a
// presence flag. What still carries that zero is an entity built from no
// placement and given no slot — a test's own literal, and a decode of one.
//
// THE PARTY IS NOT AMONG THEM, and it was until 0094. This paragraph read "the
// hero, and everything this tree spawns" while the engagement layer indexed every
// acquisition by this field and slot 0 sat outside the matrix in both directions,
// so the sentence that looked like a note about a spare field was the statement
// that no map could ever fight the player. A mission start now writes SelfSlot.
//
// TWO THINGS THIS FIELD WAS SAID TO OWE, and where they stand:
//
//   - WHICH UNITS THE PLAYER MAY COMMAND is answered on the campaign path. The
//     start seats the party, and pkg/game hands that slot to the front end, whose
//     arming gate had been comparing against a zero meaning "none established".
//   - THE FAR-SEARCH BUDGET'S OWNER TERM is read: a one-cell unit seated on
//     SelfSlot takes the flat form and every other owner's one-cell unit the
//     computed one, and a unit larger than one cell takes the n x n arm whoever
//     owns it (humanParticipantUnit, DIV-1569, DIV-1570).
//
// A FELLED ENTITY KEEPS IT, exactly as it keeps its group: the dead stay on their
// roster, so this is not residue of a state the unit has left and the constructor's
// not-alive block takes no clause for it. Both hand-over arms write the fallen for
// the same reason — an arm that skipped them would make a hand-over depend on when
// it fired.
//
// An entity built without naming any of them attacks for nothing — no damage, a
// cycle of a tick, and a to-hit of zero against a defence of zero — which is what
// leaves a world assembled before these fields existed in exactly the state it
// had. Where a placed unit's seven numbers come from is the spawn path's
// question, not this record's.
//
// Facing is WHICH WAY THE UNIT IS POINTING: a byte in 32-unit steps over 256,
// so eight directions clockwise from north, which is the decoded field's own
// width and quantum rather than a packing of ours (MOVE-TURN-031, MOVE-DIR-034;
// facing.go holds the three conversions). It is canonical — it is carried by the
// byte form and it enters the digest — and it has exactly TWO writers: the step,
// which points a mover along the cell it took, and the attacker's turn, which
// points a unit at the victim it is standing next to.
//
// EVERY BYTE IS LEGAL, and that is the one thing about this field that differs
// from every refusal beside it. The mode byte, the movement domain and the
// attack phase each have values no world may hold, so each is refused rather
// than folded; here FacingDir is total over all 256, so there is no value to
// fold onto and nothing to reject. The constructor and the decoder carry the
// byte whole, and the symmetry the other fields buy with a shared predicate this
// one gets for free.
//
// THE ZERO IS NORTH AND NOT AN ABSENCE — the direction table's own index 0. A
// unit that has never turned faces north, which is what an entity built naming
// no facing, and every unit a map places, comes out as: the placed record
// carries no facing at all, so a facing taken from one would be invented.
//
// A FELLED ENTITY KEEPS IT, exactly as it keeps its group and its owner, and the
// constructor's not-alive block takes no clause for it. The four fields that
// block does clear are residue of a state the unit has left; a body faces the way
// it fell, which is a fact about it rather than a leftover of one.
type Entity struct {
	SourceBinding    SourceBinding
	NativeBasis      NativeActorBasis
	ID               EntityID
	X, Y             int32
	TargetX, TargetY int32
	Class            int32
	HasTarget        bool
	Stall            uint8
	HP, MaxHP        int32
	// Withdraw and Wimpy are absolute-health thresholds carried by the actor's
	// definition. They are canonical because the full-tick AI tail reads them;
	// zero is inert for every living actor, whose HP is positive.
	Withdraw, Wimpy int32
	Domain          Domain
	Speed           int32
	Transit         uint16
	TransitTotal    uint16
	Stride          NativeStride
	GroupSpeed      uint8
	Facing          uint8
	// TurnRemaining is the last call's pre-step estimate. A completed call keeps
	// one interval until the next actor update. TurnTotal is the message count;
	// later rate changes affect the server step, not the client message run.
	DesiredFacing uint8
	TurnRemaining uint8
	TurnTotal     uint8
	TurnState     TurnState
	// PotionStats are permanent single-use gains in Body, Reaction, Mind,
	// Spirit order. Headroom is the derived sheet's remaining capacity to the
	// effective attribute cap; the post-step derive owns its refresh.
	PotionStats    [4]int32
	PotionHeadroom [4]int32
	Group          uint32
	Owner          uint32

	// SuppressCorpseLoot marks an actor whose canonical template discards its
	// whole carried and worn container at death instead of creating item loot.
	// The loader derives it from template identity. The simulation carries the
	// result because definition tables are unavailable when the actor dies, and
	// the value changes deterministic sack state.
	SuppressCorpseLoot bool

	// KillCreditSource, HasKillCredit and KillCreditSpell are the surviving
	// damage attribution consumed when this actor later crosses below zero
	// health. The source has an explicit presence bit because entity id zero is
	// real. KillCreditSpell is signed because the original actor field is a
	// signed byte; zero tells a fighter route to use the current weapon skill.
	// Point and area spell envelopes rewrite this state after their payload,
	// while Drain Life and later Poison ticks deliberately leave history alone.
	KillCreditSource EntityID
	HasKillCredit    bool
	KillCreditSpell  int8

	// ScanRange is HOW FAR THIS UNIT SEES, in whole cells: the range the sight
	// march is seeded with, and the term the group notice radius adds to a
	// member's distance from the centroid.
	//
	// IT IS PER UNIT AND NOT A CONSTANT, which is the opposite of what this tree
	// held until 0091 — and the correction is worth keeping for its shape rather
	// than for its value. The reason given for the constant was that the source
	// field's complete writer set is two instructions, the actor constructor's
	// default 5 and one store in an arm nothing here reaches (AI-SIGHT-006).
	// That is a displacement sweep's own count and it is accurate. What it
	// cannot see is a write that does not carry the displacement: the Data.bin
	// spawn streamers put a column into that byte through a helper holding a
	// POINTER (UNIT-STREAM-001 slot 10, DAT-HUMANS-008 slot 8), and a hero's
	// recompute names the u16 one byte LOWER and overwrites both of its bytes
	// (HERO-SIGHT-007). The instrument saw everything it could see and the
	// conclusion drawn from it was still wrong.
	//
	// IT IS ONE BYTE, which is the field's own width and not a saving: the
	// source is a byte, its guard-leash reader is a byte compare, and the group
	// radius it feeds is stored as a byte too. A wider field could hold ranges
	// the rule that produces one cannot.
	//
	// EVERY VALUE IS LEGAL. Like the facing beside it, there is no value to fold
	// onto and none to reject, so the constructor and the decoder accept exactly
	// the same set — and ZERO IS A RANGE rather than an absence: a unit at zero
	// marches nowhere and lights only the cell it stands in, because the budget
	// a range of zero seeds is half a step and the cheapest step costs a whole
	// one. Nothing supplies a default in its place; an entity that names no
	// range has none, and every path a MAP builds one by fills it.
	ScanRange uint8

	// SeeInvisible is the actor's own whole-cell Chebyshev detector radius.
	// It is consulted only after ordinary terrain sight has admitted a target;
	// zero therefore detects an invisible actor only on the observer's cell.
	SeeInvisible uint8

	// Reach is HOW FAR A BLOW CARRIES, in whole cells: the value strikeDistance
	// (combat.go) is compared against, and the field that lets a bow-armed
	// attacker stand and shoot instead of closing to the cell it means to hit.
	//
	// IT IS PER ENTITY AND NOT A CONSTANT, on ScanRange's own reason above it: a
	// weapon's range column is a value only a placement can supply, and every
	// actor in this package stood at a fixed reach of one before that join
	// existed (0104).
	//
	// THE MAP LOADER HAS SUPPLIED IT SINCE: 0104's own third task reads a
	// placement's first resolving weapon off its row — spawn.go's unitReach,
	// carried onto the entity in the one composite literal fromalm.go builds
	// each placement from — so a caller reaching NewWorld through pkg/mapload
	// has had a way to name something other than 1 since that task landed.
	//
	// ONE BYTE, and 1 TO 255 IS THE WHOLE LEGAL RANGE: the strike distance
	// floors at 1, so a reach of 0 is an actor that can never strike anything
	// and there is no value to fold it onto. THE CONSTRUCTOR NORMALISES A ZERO
	// TO 1 rather than refusing it — transitFault's own split, stated above
	// it: every entity literal this package's tests already name carries no
	// reach, and a constructor that refused their zero would turn all of them
	// red for a reason a reader could learn nothing from. The DECODER REFUSES A
	// ZERO instead, which is reachFault's whole content: a decoded record is
	// never a caller reaching for a default, it is a claim about a saved actor,
	// and 0 is not a claim this package can have written.
	Reach uint8

	AttackTarget            EntityID
	AttackTargetKind        AttackTargetKind
	HasAttackTarget         bool
	PendingAttackTarget     EntityID
	PendingAttackTargetKind AttackTargetKind
	HasPendingAttackTarget  bool
	PendingOrder            PendingOrder
	AdmittedBookSpell       uint16
	AcquirePursuit          bool
	// PursuitIdle marks an attack order whose route was refused: the victim
	// stays held and the order does nothing until an order is written again
	// (AI-327, AI-328). It is cleared with the victim and by any reissue.
	PursuitIdle bool
	// Pursuit is the route-search state of a pursuit on a unit victim. It is
	// cleared with the victim (pursuitsearch.go).
	Pursuit         PursuitSearch
	AttackPhase     AttackPhase
	AttackCountdown int32

	AttackCharge, AttackRelax int32
	// Humanoid is the actor-class predicate used by action recovery. It is
	// canonical because the same equipment and Reaction recover differently
	// when this bit differs.
	Humanoid                 bool
	ToHit, Defence           int32
	Absorption               int32
	DamageBase, DamageSpread int32
	AlwaysHits               bool

	// Load and Capacity are the actor's carried load and his carry capacity --
	// `actor+0x90` and `actor+0x92`.
	//
	// LOAD IS DERIVED AND STORED. It is what he is wearing plus half what his
	// container holds, saturating (weight.go's loadOf), and this package's own
	// recomputeLoad is its ONE writer: every producer that moves an item calls
	// it, so the field can never name contents an actor no longer has. It is
	// stored rather than computed at each read because the original stores it,
	// because the byte form then carries it into a resume whatever table that
	// world was handed, and because the one consumer -- moverSpeed -- takes an
	// Entity and not a world.
	//
	// CAPACITY IS SUPPLIED AND NEVER DERIVED HERE: `Body x 10 + 1` for a
	// character (data.Derived.Capacity), data.UnitCapacity for a Unit. Zero is
	// an entity no spawn path stated a capacity for; overloadedSpeed leaves it
	// alone.
	//
	// BOTH ARE IN THE BYTE FORM AND THEREFORE IN THE DIGEST, and Load reaches
	// movement through moverSpeed, so a wrong one is a wrong replay and not
	// only a wrong readout.
	Load          int32
	Capacity      int32
	HumanMovement HumanMovement
	ActorLoad     ActorLoad

	Decay     DecayStage
	Dwell     uint16
	DyingTime int32

	ActorState uint8
	Retreat    RetreatContinuation

	// PatrolHeadX, PatrolHeadY, PatrolTailX, PatrolTailY and PatrolLeg are
	// THE RING AND THE LEG: the two cells a patrolling actor walks between —
	// its head, the cell it stood on when it was commanded, and its tail,
	// the cell the order named — and which of the two is its current
	// waypoint.
	//
	// ALL FIVE ARE RESIDUE ON AN ACTOR NOT IN THE PATROL STATE, on the
	// stall count's and the transit pair's own ground: a ring left over
	// from an order that is not, or is no longer, in force is a fact about
	// a state the actor has left rather than one it holds. The constructor
	// NORMALISES this away — which since ActorState above is always written
	// to guard means every ring this constructor is handed is one — and the
	// decoder REFUSES it, in the relation patrolFault names.
	//
	// THE LEG IS AN INDEX AND NOT A CELL (D-2): it names one of the two
	// coordinate pairs above rather than carrying a third pair of its own,
	// so the two can never come to disagree about where the actor is
	// walking. Only patrolLegHead and patrolLegTail are legal; any other
	// byte names no waypoint and is refused on the state byte's own ground.
	PatrolHeadX, PatrolHeadY int32
	PatrolTailX, PatrolTailY int32
	PatrolLeg                uint8

	// PostX, PostY are AN ACTOR'S POST: the cell it is anchored to. EVERY
	// ACTOR HAS ONE AT EVERY MOMENT — an entity in no group, one that is
	// not alive and one under an order that is not a stance all carry a
	// post exactly as a guarding one does.
	//
	// THERE IS NO UNSET VALUE, on the target coordinates' and the patrol
	// ring's own ground: a post may name any cell, including one outside
	// the map, so no coordinate value is free to stand for "none".
	//
	// IT IS PER-ACTOR AND NOT PER-GROUP. The two stances this build knows each
	// anchor one, and a group's members do not share a cell — a group-level
	// post would collapse a group onto a point and would be a second spelling
	// of the centroid, a different object with a different job: the centroid
	// clips a group's candidates and is nobody's destination.
	//
	// Its writers are named where they write it, not here: this task is
	// the first of the two, the world constructor.
	PostX, PostY int32

	// Mana and MaxMana are the entity's mana pool and the maximum it was built
	// with — the health pair's own shape, and carrying the health pair's
	// rules and no others: A MAXIMUM OF ZERO OR LESS IS A UNIT WITH NO MANA
	// SYSTEM, and no value of Mana is constrained against it. Every world
	// assembled before these fields existed holds every entity at the zero
	// value, which is exactly that state.
	//
	// NEITHER FIELD IS READ BY THIS TASK. What advances them is the
	// regeneration pass this story adds in a later task; this one carries
	// the pair through the record, the form and the digest, and nothing
	// else.
	Mana, MaxMana int32

	// HealthRegenPeriod and ManaRegenPeriod are the per-unit divisor of each
	// pool's regeneration rate — named as pkg/data names its own two columns,
	// so one number carries one name across the tiers. A larger period is
	// slower. NEITHER FIELD IS READ BY THIS TASK; their meaning belongs to the
	// regeneration pass alone.
	HealthRegenPeriod, ManaRegenPeriod   int32
	HealthRegeneration, ManaRegeneration int32
	ActionClock                          ActionClock
	attackNotice                         attackNotice
	RotationSpeed                        int32
	// SecondaryDamage is one ordered item-effect result, not one value per
	// school. Its Selector chooses exactly one Protection entry at a blow.
	SecondaryDamage SecondaryDamage
	// SecondBase/SecondSpread are independent of the main physical and third
	// elemental components. Both bytes are canonical state (HERO-DMG2-029).
	SecondBase, SecondSpread uint8
	CurrentProfileBasis      CurrentProfileBasis

	// HealthHundredths and ManaHundredths are each pool's own hundredths
	// remainder — the fractional part of a gain too small to move a pool
	// by a whole point on its own, carried from one qualifying tick to
	// the next in a byte of its own.
	//
	// LEGAL VALUES ARE 0 TO 99: a hundredth of a point is what the field holds
	// and 99 is its largest meaning, which is the field's own width and not a
	// saving. A value above 99 has no value to fold onto. THE CONSTRUCTOR FOLDS
	// IT TO ZERO, on reachFault's own split below: a caller handed a stale
	// accumulator has nothing it could do with an error. THE DECODER REFUSES IT
	// instead, which is regenFault's whole content: a decoded record is a claim
	// about a saved unit, and no value above 99 is a claim this package can
	// have written.
	HealthHundredths, ManaHundredths uint8

	CommandGroup uint32

	// SkillXP is the six per-slot experience integers this entity has earned,
	// in slot order — slot 0 General, 1..5 the weapon skills. It is THE WHOLE
	// OF THE EXPERIENCE STATE THIS PACKAGE HOLDS: a level is never carried,
	// because deriving one from an experience needs pow and this package bans
	// floats — S and its inverse live in pkg/data, outside the wall, and are
	// read from these integers only where a level is WANTED: the panel, and
	// nowhere on a tick path.
	//
	// EVERY SLOT IS SIGNED. Item-borne Poison may send a negative damage
	// award, and the original sink adds that signed result without a matching
	// level-decrement arm. It is
	// canonical — carried by the byte form and hashed with everything else
	// — and it has exactly one writer outside this file's own construct-
	// and-decode pair: a later task's payExperience, the only routine in
	// this package that ever adds to it.
	SkillXP [skillSlots]int32

	// Reaction, Mind and Spirit are the entity's own statistics. Mind is the
	// one statistic a gain's scaling reads. It is carried WHOLE and refused
	// nowhere, on Speed's and the health pair's own rule: every int32 is a
	// state a placement or a party mint can hand this field, so refusing one
	// here would make a world this package can produce a world it cannot read
	// back. It is a units-table column carried through rather than a literal
	// this build invented, and this package itself never reads it —
	// payExperience does. Reaction and Spirit are carried because Control
	// Spirit copies the victim's three-stat subset into its Ghost:
	// Reaction/2+1, Mind and Spirit. Keeping the source values, rather than
	// trying to invert Speed, sight or protections, is the only exact input to
	// that singular arm.
	Reaction, Mind, Spirit int32

	XPValue int32

	// TypeID and the three treasure values are the Units-row inputs to a
	// creature's death-gold roll (HERO-KILL-027). They are canonical because a
	// living unit restored from the byte form must make the same later roll.
	// Human and unresolved placements leave all four at zero; TypeID's strict
	// gate keeps them from producing gold.
	TypeID                               int32
	GoldChance, TreasureMin, TreasureMax int32

	// XPSlot is the slot a gain THIS entity earns is credited to: the skill of
	// the weapon in its hand for a class that fights, slot 0 for one that casts
	// — the fighter/mage branch, resolved at the map load or the party mint
	// into this one number rather than read again at the moment of the blow.
	// ONLY 0..5 ARE LEGAL, refused otherwise on SkillXP's own ground above: a
	// slot outside the six its array holds has no experience integer to credit
	// and nothing to fold onto.
	//
	// IT IS NOT WRITTEN ONLY ONCE, and this doc said it was until the
	// hotfix that made it false. THE WEAPON IN A UNIT'S HAND CAN CHANGE
	// after the mission opens, and when it does the recompute that follows
	// carries a new skill slot exactly as it carries new damage and a new
	// reach — so SetCombat (rearm.go) writes this field beside those,
	// through the one door, and refuses a block that would put it out of
	// range. "Resolved once" would mean a hero fighting with a mace paying
	// every blow into the sword skill he started with, which is precisely
	// the defect that hotfix fixed.
	XPSlot uint8

	// GainsXP is whether this entity's CLASS earns experience at all. EVERY
	// BYTE BUT 0 AND 1 IS REFUSED ON DECODE, on the target-presence byte's own
	// ground: a byte read as truthy would map two byte forms onto one world. A
	// Go bool has no value outside {true, false} for a fault function to
	// refuse, so that refusal is the decoder's own switch, not experienceFault
	// below.
	GainsXP bool

	// KnownSpells is the entity's own spellbook (0127 FR-4b): a bitmask
	// subscripted by spell id, bit i set meaning spell i is known. It is
	// state on Mana's own rule. The byte form and digest carry the whole mask,
	// so every world this package produces can be read back without loss.
	// Construction and decode seed the mask. KindReadBook is its one runtime
	// writer: a living mage consumes one readable carried book and ORs in that
	// book's spell bit. Nothing forgets a spell or writes the mask from UI state.
	//
	// EVERY VALUE IS LEGAL, on Class's and Mind's own rule: a caller's own
	// mask is carried whole and refused nowhere, because refusing one here
	// would make a world this package can produce a world it cannot read
	// back. A bit outside the shipped id space (1..28) names a spell no
	// table this build loads can ever hold a row for, so the cast's own
	// linear lookup already answers "no such row" to it without this
	// record having to.
	KnownSpells uint32
	Book        Spellbook

	// CreatureSpells is a creature class's three spell slots: each a spell id
	// and its Probability scaled into the draw threshold (UNIT-SPELL-007). The
	// engage routine draws against them (AI-341). Empty for every actor whose
	// class names no spell. Carried whole; no value is refused.
	CreatureSpells [CreatureSpellSlots]CreatureSpell

	// Effective skills, known trained inputs and class have independent owners.
	// None is reconstructed from XP or the current mana maximum.
	Skill          [skillSlots]int32
	NativeTraining NativeTraining
	NativeClass    NativeClass

	// WeaponSpell and WeaponSpellLevel are the spell an entity's WEAPON carries
	// and the level it casts it at: the id and the power a caster's own attack
	// releases in place of a strike, once weaponSpell (spell.go) says the
	// weapon carries one, the id names a row this world's own table loaded, and
	// the entity itself is a caster (isMage).
	//
	// BOTH ARE CARRIED WHOLE AND REFUSED NOWHERE, on Mind's and KnownSpells'
	// own rule. WeaponSpell naming no row this world's table loaded is NOT a
	// decode-time fault — it is FR-2b's own runtime answer, read by
	// weaponSpell alone: such a weapon carries a spell this build cannot
	// resolve, and the actor attacks as if it carried none. And
	// WeaponSpellLevel is never clamped: it is the attachment's own authored
	// number, carried through rather than derived from any statistic the way a
	// commanded cast's power is (spellPower, above).
	//
	// NEITHER FIELD IS WRITTEN BY THIS PACKAGE. What folds a weapon's spell
	// onto an entity is pkg/mapload's and pkg/game's road, the one every
	// other combat number already travels (CombatBlock, rearm.go) — this
	// task carries the pair through the record, the byte form and the
	// digest, and reads it only from weaponSpell and closedOn (combat.go).
	WeaponSpell      uint16
	WeaponSpellLevel int32
	// WeaponSpellSource identifies the single owner of the cached pair.
	// Item is derived from slot 1's first cast effect; Innate belongs to a
	// non-carriable weapon definition; Legacy is migration-only residue.
	WeaponSpellSource WeaponSpellSource

	// AutoSpell is the one spell this entity casts UNBIDDEN, 0 for none, and
	// CastWait is how many ticks are still to run before it may do so again.
	//
	// ZERO IS NONE, on WeaponSpell's own reserved-row ground above: the Spells
	// collection's row 0 is allocated and never written, so no id of 0 ever
	// names a spell and this field's zero value carries "no autocast" with no
	// presence flag beside it. Every entity is built holding it.
	//
	// THE AUTOCAST IS NOT A SECOND CAST ARM. autoCast (spell.go) reaches
	// castSpell — the same routine a KindCast command reaches — so an unbidden
	// cast can do nothing a player could not have ordered by hand, and the two
	// cannot come to disagree about the economy, the roll or the award.
	//
	// THE WAIT IS RESET ONLY BY AN APPLIED CAST. A refused attempt leaves it at
	// zero and is retried on the next tick, which is what keeps a caster whose
	// target stepped one cell out of range from going quiet for a whole period
	// after it stepped back in.
	AutoSpell uint16
	CastWait  uint8

	// SpellFX and SpellFXSpell are the SPELL EFFECT MARK this entity is
	// carrying: how many ticks of it are left, and which spell set it. Both
	// fall to zero together, so a world whose marks have expired is
	// byte-identical to one that never carried any.
	//
	// IT IS A PER-ENTITY PRESENTATION MARK, not effect.go's canonical attached
	// effect list and not a list of map objects. The picture a direct
	// unit-addressed cast computes is `2*spellId + 8` (`MAGIC-PIC-026`), which
	// is always EVEN, and the client's own arm answers an even picture by
	// creating no map object at all and giving the CASTER the cast action
	// instead (`MAGIC-PIC-027`). This mirror therefore sits on the actor even
	// when the canonical effect record sits in World.attached. It never decides
	// whether that record exists or owns its remaining duration.
	//
	// SpellFXSpell IS AN ID AND NOT A SCHOOL. The front end resolves it against
	// the world's own table for the colour it draws and the name it could show,
	// so this package carries the row's identity rather than a second copy of
	// one of its columns. SpellFX IS A WORD. A presentation mark may mirror an
	// attached effect's word-sized remaining duration. Instant 30 writes its
	// authored `(u16)p1` to the canonical record rather than to this mirror.
	SpellFX      uint16
	SpellFXSpell uint8

	// Protection is the five elemental protection values in school order.
	// TokenSize is the square actor footprint side; zero is normalised to one
	// by the consumers that need a footprint.
	Protection [5]int32
	// Resistance holds the five unsigned weapon damage-kind bytes in Blade,
	// Axe, Bludgeon, Pike and Shooting order. XPSlot 1..5 selects index
	// XPSlot-1; slot 0 names the original's unfilled leading byte and bypasses
	// this array. Unlike elemental Protection, these values are not clamped:
	// the table and modifier folds store their low byte modulo 256.
	Resistance [5]uint8
	TokenSize  uint8

	// EscortTarget, HasEscortTarget and EscortRange are the ESCORT ORDER a
	// member of a group takes from script group sub-command 11 (Defend) or 15
	// (Follow) — `AI-FOLLOWSET-116`'s `ord+0x10` and `AI-FOLLOWRANGE-115`'s
	// `ord+0x70`.
	//
	// HasEscortTarget is separate from the id for the reason every other
	// reference in this package carries its own presence bit: entity id
	// zero is a real entity and no id value is free to mean "none".
	//
	// THE RANGE IS ONE BYTE, which is the field's own width: the arm loads
	// the node's 32-bit value with a byte move and both helpers compare and
	// store one byte, so an authored 256 behaves as 0 and coerces to 3
	// (`TRIG-GRPLIMIT-048`). Shipped ranges are 1..6.
	//
	// THE TRIPLE IS RESIDUE ON AN ENTITY IN NO ESCORT STATE, exactly as the
	// patrol ring is on an entity not patrolling — see actorFault's rules
	// inside patrolFault.
	EscortTarget      EntityID
	HasEscortTarget   bool
	EscortRange       uint8
	EscortOrder       uint8
	EscortTurnPending bool

	// OffMap is whether the mission script has taken this entity OFF THE MAP.
	// It is the one bit instant 16 sets and instant 17 clears
	// (`TRIG-OFFMAP-041`, `TRIG-RETURN-042`).
	//
	// IT IS STATE AND NOT POSITION. An off-map entity keeps its coordinates,
	// and that is the whole reason instant 17 needs no authored cell: the cell
	// it returns to is the cell it never stopped carrying. It keeps its owner,
	// its group, its command group, its health, its container, its equipment,
	// its order and its attack target too — the arm's untouched list is
	// everything except this bit and the map occupancy it implies.
	//
	// MEMBERSHIP IS NOT PRESENCE. The script's group-count check reads Group
	// and never this field, so a group whose members are all off the map still
	// answers its full count — which is the decoded arm's own answer, because
	// that arm is an unfiltered read of the group's cached count.
	//
	// EVERY VALUE IS LEGAL and there is nothing to refuse: a Go bool has no
	// value outside {true, false}, and the byte form's 0/1 byte is refused on
	// the target-presence byte's own ground, in the decoder's own switch.
	//
	// WHETHER THE ORIGINAL SAVES IT IS UNKNOWN (spec SC-1). It is canonical
	// HERE — carried by the byte form and hashed with everything else —
	// because a save taken with a unit removed and restored with it back on the
	// map would put a unit the mission took away in front of the player.
	OffMap bool

	// MapUnitID is the authored map id this entity was placed under: the type-6
	// record's own identifier word, which is the value a script's Target_Unit
	// names below the hero band.
	//
	// It exists because check opcode 9 returns one. That arm reads the
	// subject's pursuit target and writes the target's `actor+0x08`, which
	// `TRIG-TARGETID-032` and `ALM-UNIT-018` identify as the type-6 unit id.
	// The target is chosen at run time, so no compile-time binding can supply
	// the value and the simulation has to hold it.
	//
	// It is a field on the entity and not a reverse table on the world. The id
	// never changes after placement, so a table would be a second
	// representation of a fact that never moves, it would need its own section
	// in the byte form and its own consistency rule against the entity list,
	// and every other authored per-entity column this package carries (Group,
	// Owner, TypeID, XPValue) is already a field here. pkg/mapload's
	// ScriptRefs.Units is the same fact in the other direction, built for a
	// different consumer.
	//
	// Zero means the entity carries no authored map id, on WeaponSpell's terms
	// and not Group's. A party member minted by this build was never placed by a
	// map and has no such id. The original gives every actor a runtime id at
	// that offset and this build assigns none (DIV-241). A member restored from
	// an original save does carry one, through Saved.MapUnitID, and pkg/mapload
	// writes it here.
	//
	// Every value is legal and none is refused, on Mind's rule: a map may write
	// any word into the record, and refusing one here would make a world this
	// package can produce a world it cannot read back. That includes 0, which a
	// map may place a unit under and which this field cannot then tell apart
	// from an unplaced entity (DIV-242).
	//
	// It is uint16 because that is the field's width in the map record
	// (alm.Unit.UnitID) and the key type of pkg/mapload's own table.
	MapUnitID uint16
}

// Guard is the constructor/felled default. Patrol, Defend, Follow and
// acquire-in-place are dispatched by actorPass. AI-STATE-011 and
// AI-FOLLOWSET-116 supply the state numbers; player Defend and the script
// setters share the same persisted actor states, not group-name semantics.
//
// THE DECODED ENGAGE STATE, 3, WAS DELIBERATELY ABSENT (D-10) AND 1141
// BRINGS IT BACK, because the argument that kept it out has stopped being
// true. D-10's ground was that the attack order orderAttack writes already
// carries the whole fact, so a state byte beside it would be a second
// representation free to disagree with the order it duplicates. That held
// for exactly as long as NOTHING READ THE BYTE for such a member: with the
// per-actor guard arm in place the byte is no longer a duplicate, it is the
// dispatch's own selector, and a member left at guard is a member the guard
// arm decides — which for an engaging member means its block scan finds
// the named victim outside the five cells around its post and breaks the
// order off on the next tick (`AI-BREAK-041`).
//
// IT HAS NO ARM, and that is D-1's seam rather than an omission: arm 3 is the
// law's engage routine and re-issuing an order this package already holds would
// be a behaviour, not a transcription. actorPass has no case for it, so an
// engaging member is left in every field exactly as it was before this state
// existed — which is what makes writing it a change of ownership and not a
// change of behaviour.
const (
	// Native state 2 records only post-transfer completion, not original
	// state 2's still-unimplemented walk/pending-order/progress machine.
	actorStatePickupComplete uint8 = 2
	actorStateEngage         uint8 = 3
	actorStateGuard          uint8 = 0xb
	actorStatePatrol         uint8 = 0xa
	actorStateDefend         uint8 = 8
	actorStateAcquire        uint8 = 0xc
	actorStateFollow         uint8 = 0x11
	actorStateRetreat        uint8 = 0x16
)

// escortState reports whether s is one of the two states that carry an escort
// target and a range. It is one function so that "is this an escort" cannot come
// to be spelled two ways between the fault rules and the writers.
func escortState(s uint8) bool { return s == actorStateDefend || s == actorStateFollow }

// livingOnlyState reports whether s names an order a dead actor cannot hold.
// Death resets the entity to guard (clearFelled); validation refuses these
// states on a body, and the SAV order projection writes a body's current
// state in place of a retained one.
func livingOnlyState(s uint8) bool {
	return s == actorStatePatrol || escortState(s) || s == actorStateAcquire ||
		s == actorStateRetreat || s == actorStatePickupComplete
}

// LivingOnlyActorState is livingOnlyState over a SAV order's state word.
func LivingOnlyActorState(s uint32) bool { return s <= 0xff && livingOnlyState(uint8(s)) }

// actorStateDefined reports whether s is a state this build names. It is the
// membership test patrolFault's first rule asks, written out because the set is
// only explicitly implemented values.
func actorStateDefined(s uint8) bool {
	switch s {
	case actorStateGuard, actorStatePatrol, actorStatePickupComplete, actorStateEngage,
		actorStateDefend, actorStateAcquire, actorStateFollow, actorStateRetreat:
		return true
	}
	return false
}

// patrolLegHead and patrolLegTail say which of a patrol ring's two cells is
// an actor's current waypoint. The ring is two cells and there is no third
// leg to name, which is patrolFault's own ground for refusing any other
// byte.
const (
	patrolLegHead uint8 = 0
	patrolLegTail uint8 = 1
)

// clearPatrol puts e back to the state a non-patrolling actor is in: guard,
// with an empty ring and no leg. It is the entity's own half of leaving a
// patrol, written once so the state and the four fields that make up the
// ring cannot come apart into a partial clear.
func (e *Entity) clearPatrol() {
	e.ActorState = actorStateGuard
	e.PatrolHeadX, e.PatrolHeadY = 0, 0
	e.PatrolTailX, e.PatrolTailY = 0, 0
	e.PatrolLeg = patrolLegHead
}

// clearEscort is clearPatrol's counterpart for the escort triple (0166 D-13):
// the state back to guard and the target, its presence bit and the range all
// gone. It is written once for the reason clearPatrol is — the three fields make
// up one order and a partial clear would leave a range naming a unit nothing
// escorts.
func (e *Entity) clearEscort() {
	e.ActorState = actorStateGuard
	e.EscortTarget, e.HasEscortTarget = 0, false
	e.EscortRange = 0
	e.EscortOrder, e.EscortTurnPending = escortOrderNone, false
}

// patrolFault names what is wrong with e's actor state, or nil when it is a
// state this package can produce.
//
// A STATE outside {guard, patrol} has no value to fold onto, on Domain's
// and Mode's own ground: the constructor never meets it, because ActorState
// is overwritten before this is ever asked, and a decoded byte this build
// does not define names an arm this build does not have.
//
// A LEG outside {0, 1} is the same kind of value for the same reason: the
// ring is two cells and a byte naming a third names nothing.
//
// A RING OR A LEG on an entity NOT IN THE PATROL STATE is residue in
// exactly the sense a stall count on an entity with no target already is —
// a fact left over from an order that is not, or is no longer, in force.
//
// And a PATROL STATE on an entity that is NOT ALIVE is residue on the decay
// stage's own ground: a felled actor holds no order of either layer, so a
// state naming one is a fact about a unit that has left the living.
func patrolFault(e Entity) error {
	switch {
	case !actorStateDefined(e.ActorState):
		return fmt.Errorf("actor state is %d, which is not one of guard (%d), patrol (%d), "+
			"defend (%d), acquire (%d), follow (%d), retreat (%d), engage (%d) or pickup completion (%d)",
			e.ActorState, actorStateGuard, actorStatePatrol,
			actorStateDefend, actorStateAcquire, actorStateFollow, actorStateRetreat,
			actorStateEngage, actorStatePickupComplete)
	case e.PatrolLeg != patrolLegHead && e.PatrolLeg != patrolLegTail:
		return fmt.Errorf("patrol leg is %d, which is neither head (%d) nor tail (%d)",
			e.PatrolLeg, patrolLegHead, patrolLegTail)
	case e.ActorState != actorStatePatrol && (e.PatrolHeadX != 0 || e.PatrolHeadY != 0 ||
		e.PatrolTailX != 0 || e.PatrolTailY != 0 || e.PatrolLeg != patrolLegHead):
		return fmt.Errorf("a patrol ring ((%d,%d)-(%d,%d), leg %d) on an entity in actor state %d, which is not patrol",
			e.PatrolHeadX, e.PatrolHeadY, e.PatrolTailX, e.PatrolTailY, e.PatrolLeg, e.ActorState)
	case e.ActorState == actorStatePatrol && !e.Alive():
		return fmt.Errorf("actor state is patrol on an entity at %d/%d, which is not alive", e.HP, e.MaxHP)

	// THE ESCORT TRIPLE'S OWN THREE RULES (0166 D-13), each the counterpart
	// of a ring rule directly above it.
	case !escortState(e.ActorState) && (e.HasEscortTarget || e.EscortTarget != 0 || e.EscortRange != 0):
		return fmt.Errorf("an escort order (target %d, present %v, range %d) on an entity in actor state %d, which is not defend (%d) or follow (%d)",
			e.EscortTarget, e.HasEscortTarget, e.EscortRange, e.ActorState, actorStateDefend, actorStateFollow)
	case escortState(e.ActorState) && !e.HasEscortTarget:
		return fmt.Errorf("actor state is %d, an escort state, with no escort target", e.ActorState)
	case livingOnlyState(e.ActorState) && !e.Alive():
		return fmt.Errorf("actor state is %d on an entity at %d/%d, which is not alive",
			e.ActorState, e.HP, e.MaxHP)
	case e.ActorState == actorStatePickupComplete && (e.HasTarget || e.HasAttackTarget && e.PendingOrder.Kind != PendingPickupComplete || e.GroupSpeed != 0):
		return fmt.Errorf("pickup completion carries movement, attack or group-speed residue")
	}
	return escortResidueFault(e)
}

// DecayStage is how far a body has decayed, and it is the SECOND thing in this
// record that says anything about life — the health pair being the first — so
// what keeps the two from disagreeing is written here rather than left to the
// sites that move them.
//
// The rule is one sentence: A POSITIVE STAGE AND BEING NOT ALIVE HOLD OF THE
// SAME ENTITIES. The constructor normalises either half onto the other and the
// decoder refuses both, which is the relation every residue field in this record
// already stands in, so no world this package builds, decodes or advances can
// carry a body at DecayNone or a living unit at a stage.
//
// It is state and NOT A FUNCTION OF HEALTH, and that is the whole reason it
// exists rather than being derived. A blow that overshoots leaves health far
// below the first threshold, and a stage read off health would put that body
// straight into its bones — it would never fall, because the fall is what the
// first stage draws. The first stage is set unconditionally at death and the
// walk carries health down from there, so the stage LAGS health on purpose and
// stops lagging only once the body has been torn down.
//
// STAGE 5 IS NOT A VALUE. It is what reaching the bottom of the ladder MEANS —
// the entity leaves the world — so it is never stored, never encoded, and
// refused on the way in beside every byte above it.
type DecayStage uint8

const (
	// DecayNone is a unit that has not died. It is the ZERO VALUE, so every
	// world assembled before this field existed holds living units at it, and
	// bodies in such a world are put at DecayFallen by the constructor.
	DecayNone DecayStage = 0
	// DecayFallen is a body still lying where it fell: the stage a death starts
	// at, the only stage that owes a dwell, and the only one that holds ground.
	DecayFallen DecayStage = 1
	// DecayBones is the first stage drawn from the bone block rather than from
	// the fall. It is exported for the drawer, which compares against it and
	// reads nothing else about a body.
	DecayBones DecayStage = 2
	// decayDarkStage is the first stage that stamps no sight.
	decayDarkStage DecayStage = 3
	// decayLast is the deepest stored stage. A body past it is gone.
	decayLast DecayStage = 4
)

// The ladder's thresholds, and the health a body that leaves none is pinned to.
//
// They are HEALTH values and not stage numbers: what the walk moves is health
// and the stage is read off it, so these are the only numbers in the ladder and
// there is no second table to keep level with them.
const (
	decayBonesHP  = -10
	decayThirdHP  = -20
	decayFourthHP = -40
	// decayGoneHP is the last health a body still exists at: the ladder's top
	// rung is reached BELOW it and not at it.
	decayGoneHP = -600
	// noCorpseHP is what a mover of a non-ground domain is pinned to when its
	// dwell runs out, which by the ladder removes it on that same tick.
	noCorpseHP = -1000
	// controlSpiritCorpseHP is the health a Control Spirit cast leaves on the
	// corpse it consumes and retires (MAGIC-SING-019 (c)).
	controlSpiritCorpseHP = -10001
)

// decayStageFor is the stage a torn-down body's health puts it at — the whole of
// the ladder, in one expression.
//
// It answers decayLast+1, the stage that is not a value, for a body that has
// finished; that is how the pass tells "remove this" from "write this down"
// without a second predicate free to disagree with the thresholds.
//
// A body whose health has not reached the first threshold stays at DecayFallen,
// so the answer is total over every int32 and no caller needs a floor test.
func decayStageFor(hp int32) DecayStage {
	switch {
	case hp < decayGoneHP:
		return decayLast + 1
	case hp <= decayFourthHP:
		return decayLast
	case hp <= decayThirdHP:
		return 3
	case hp <= decayBonesHP:
		return DecayBones
	}
	return DecayFallen
}

// dwellOf is the dwell a fresh body takes on: its own dying time, clamped into
// the field's width at both ends.
//
// A NEGATIVE DYING TIME IS NO DWELL, on the rule the speed and the health
// maximum already take: the column's absent value is negative, and a body whose
// class names no dwell is torn down on the tick it falls rather than lying for a
// wrapped sixty-five thousand ticks. The upper clamp is the field's own width
// and is reachable only by a column no shipped table carries.
func dwellOf(e Entity) uint16 {
	switch {
	case e.DyingTime <= 0:
		return 0
	case e.DyingTime > int32(^uint16(0)):
		return ^uint16(0)
	}
	return uint16(e.DyingTime)
}

// decayFault names what is wrong with e's decay state, or nil when it is a state
// this package can produce.
//
// It is one function with two callers, the constructor and the decoder, for the
// reason transitFault and attackFault are: a constructor that could produce what
// the decoder will not read back would build worlds this package cannot marshal
// and read again.
//
// It refuses only what has NO VALUE TO FOLD ONTO — a stage past the deepest
// stored one, which is either the stage that means removal or a byte naming
// nothing at all. The two pairing rules are the constructor's to normalise and
// the decoder's to refuse, so they are not here.
func decayFault(e Entity) error {
	if e.Decay > decayLast {
		return fmt.Errorf("decay stage is %d, the deepest a body is stored at is %d", e.Decay, decayLast)
	}
	return nil
}

// clearDecay puts e back to the state a living unit is in. It is the entity's
// own half of the pairing rule, written once so that "a living unit carries no
// stage and no dwell" cannot come apart into two assignments.
func (e *Entity) clearDecay() { e.Decay, e.Dwell = DecayNone, 0 }

// moverSpeed is the speed a rate is computed from: the group term when e carries
// one, and the entity's own otherwise.
//
// THIS IS THE SEAM. It is the ONE reader of GroupSpeed in this package, and the
// two things a later story will want to do both land here and nowhere else:
//
//   - LIFTING THE DEFECT. That the term outlives the order, the death and the
//     save is reproduced on purpose, and undoing it is a fourth call to
//     clearGroupSpeed — at restAt, where an order ends — not a rewrite. The
//     tests assert the CURRENT behaviour, so adding that call fails them loudly
//     rather than changing a rate nobody is watching.
//   - RE-HOMING THE TERM. The thing being reconstructed hangs it off a group
//     object owned by a player, not off the member. It is held per member here
//     because nothing in this tree adds a member to a group, moves one between
//     groups, or writes the term outside an order — so the two are the same
//     observable behaviour, and one byte is the shape that adds no state nothing
//     can exercise. A story that can change membership after an order moves the
//     byte onto a group and changes this function and the three writers.
//
// A NONZERO GROUP TERM IS USED RAW, for every mover. The rate law reads the
// group byte zero-extended when it is nonzero and has no load term
// (MOVE-RATE-029); the term already carries each member's overload penalty,
// because a formation order takes it as the minimum of the members' own speed
// words, and the Human derive subtracts the penalty inside that word
// (MOVE-GROUP-030, SAV-1116, MOVE-RATE-053, HERO-SPEED-008). So every member
// of one formation order moves with one term, and groupMinSpeed feeds it
// aloneSpeed rather than Entity.Speed.
//
// WITH NO GROUP TERM the mover moves at its own speed: a current retained
// source Human speed as it stands, a Unit's Speed (UNIT-DERIVE-003 derives no
// penalty), and a native Humanoid's Speed with overloadedSpeed applied after
// its modifier (DIV-1421).
func moverSpeed(e Entity) int32 {
	if e.GroupSpeed != 0 {
		return int32(e.GroupSpeed)
	}
	return aloneSpeed(e)
}

// GroupRateTerm is the group term id's transit rate reads.
func (w *World) GroupRateTerm(id EntityID) uint8 {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return 0
	}
	return w.groupRateEntity(w.entities[i]).GroupSpeed
}

// RateSpeed is the speed term id's next transit rate is computed from.
func (w *World) RateSpeed(id EntityID) (int32, bool) {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return 0, false
	}
	return moverSpeed(w.groupRateEntity(w.entities[i])), true
}

// aloneSpeed is the speed e would move at under no group term: the value a
// formation order's running minimum reads for it.
func aloneSpeed(e Entity) int32 {
	if _, ok := e.RetainedHumanSpeed(); ok {
		return int32(e.HumanMovement.RawSpeed)
	}
	if !e.Humanoid {
		return e.Speed
	}
	return overloadedSpeed(e.Speed, e.Load, e.Capacity)
}

// rated reports whether e moves at a rate at all, which is the whole of the
// widening rule Speed's doc states.
//
// It is a PREDICATE in one place, so the rule that decides who has a rate cannot
// come to differ between the tick that applies it and anything that reads it.
func rated(e Entity) bool { return e.retainedHumanMovement() || moverSpeed(e) > 0 }

// clearGroupSpeed drops e's group rate term.
//
// It had THREE callers through 0098 and has FOUR since 0096: the group-order
// arm (issueGroupDestination since this story lifted it out of groupOrder),
// which zeroes every member's term before the formation arm may set it — the
// original allocates a fresh group per order, whose term starts at zero; the
// plain move-to arm, which is that same fresh group of one; clearFelled,
// because a felled member is unlinked from its group and has none; and
// armSwarm's own walk write, on the same "fresh group of one" grounds the
// plain move-to arm stands on — an unoffset single-cell walk carries no
// formation and so no rate term either. Arriving is NOT among them, and that
// absence is the defect 0098 reproduces.
func (e *Entity) clearGroupSpeed() { e.GroupSpeed = 0 }

// transitFault names what is wrong with e's transit pair, or nil when the pair
// is a state a tick can produce.
//
// Three shapes are refused, and each is a state this package cannot write. A
// total above the law's own longest transit could not have come from transitOf.
// An owed count at or past its own total is a transit that has run past its
// length — the count is written strictly below the total and only ever falls. A
// nonzero either on a unit that is NOT ALIVE is residue of a crossing it has
// left, exactly as a target on one is; that third one is the constructor's to
// normalise and the decoder's to refuse, so it is not here.
//
// It is one function with two callers, the constructor and the decoder, for the
// reason the stall rules already are: a constructor that could produce what the
// decoder will not read back would build worlds this package cannot marshal and
// read again, and no caller could do anything about it.
func transitFault(e Entity) error {
	switch {
	case e.TransitTotal > maxTransit:
		return fmt.Errorf("transit length is %d, the longest the law can produce is %d",
			e.TransitTotal, maxTransit)
	case e.TransitTotal == 0 && e.Transit != 0:
		return fmt.Errorf("%d transit tick(s) owed with no transit to owe them to", e.Transit)
	case e.TransitTotal != 0 && e.Transit >= e.TransitTotal:
		return fmt.Errorf("%d transit tick(s) owed on a transit of %d, which no tick can leave",
			e.Transit, e.TransitTotal)
	}
	return nil
}

// experienceFault names what is wrong with e's experience state, or nil
// when it is a state this package can produce.
//
// It is one function with two callers, the constructor and the decoder, for
// the reason attackFault and transitFault already are: a constructor that
// could produce what the decoder will not read back would build worlds this
// package cannot marshal and read again.
//
// One shape is refused. A CREDITED SLOT outside 0..5 names no integer in
// SkillXP to credit — skillSlots' own ground. Slot experience itself is signed:
// negative item-Poison awards are preserved by the common sink and byte form.
//
// MIND AND THE EXPERIENCE VALUE ARE NOT HERE, on Speed's and the health
// pair's own rule: both are table columns carried whole, and every int32 is
// a state the constructor accepts, so refusing one here would make a world
// this package can produce a world it cannot read back. THE GAINS FLAG IS
// NOT HERE EITHER: a Go bool has no value outside {true, false} for this
// function to refuse, so its own decoded byte is checked by the decoder's
// switch, on the target-presence byte's own ground, before it ever becomes
// one.
func experienceSlotFault(slot uint8) error {
	if slot >= skillSlots {
		return fmt.Errorf("experience slot is %d, the six this package holds are 0..%d",
			slot, skillSlots-1)
	}
	return nil
}

func experienceFault(e Entity) error { return experienceSlotFault(e.XPSlot) }

// reachFault names what is wrong with e's reach, or nil when it is a value
// this package can produce.
//
// ONLY the decoder calls it, after source actor records have restored
// e.ActorLoad.Source.EquipmentRuntimePresent. Absent that presence, the
// constructor's own half of the split is not a question this asks —
// newWorld folds a zero to 1 unconditionally, before this function could
// have anything to refuse — so the one shape that ever reaches this call
// is exactly what a construction never produces: a decoded reach of 0. It is
// refused on the strike distance's own ground: the distance strikeDistance
// answers floors at 1, so a reach of 0 is an actor that can never strike
// anything and there is no value to fold it onto.
//
// A source (original-save-imported) actor is exempt from that refusal.
// EquipmentRuntimePresent actors carry their own saved Reach/AttackCharge/
// AttackRelax bytes, never folded by this package's own constructor, and a
// saved reach of 0 is a value the basis decode legitimately produces
// (docs/1110/handoff.md's "EquipmentRuntimePresent discriminates saved
// Reach/Charge/Relax bytes, including reach zero"). Refusing it here would
// reject an original SAV this package must still load.
func reachFault(e Entity) error {
	if e.Reach == 0 && !e.ActorLoad.Source.EquipmentRuntimePresent {
		return fmt.Errorf("reach is %d, which can never strike anything and has no value to fold onto", e.Reach)
	}
	return nil
}

// regenFault names what is wrong with e's two regeneration remainders, or nil
// when both are values this package can produce.
//
// ONLY the decoder calls it, on reachFault's own split above: the
// constructor's own half of the split is not a question this asks —
// newWorld folds each remainder above 99 to 0 unconditionally, before this
// function could have anything to refuse — so the one shape that ever
// reaches this call is exactly what a construction never produces: a decoded
// remainder above 99.
func regenFault(e Entity) error {
	if e.CurrentProfileBasis > ProfileNativeRetired {
		return fmt.Errorf("invalid current profile basis %d", e.CurrentProfileBasis)
	}
	if e.CurrentProfileBasis == ProfileOriginalCurrent && int16(e.ManaRegenPeriod) == 0 && e.MaxMana != 0 {
		return fmt.Errorf("source-current nonzero mana maximum with zero regeneration divisor")
	}
	if e.CurrentProfileBasis != ProfileNative {
		return nil
	}
	switch {
	case e.HealthHundredths > 99:
		return fmt.Errorf("health remainder is %d, the largest legal value is 99", e.HealthHundredths)
	case e.ManaHundredths > 99:
		return fmt.Errorf("mana remainder is %d, the largest legal value is 99", e.ManaHundredths)
	}
	return nil
}

// clearTransit drops the transit pair and its retained stride inputs when a
// unit is felled, a nonliving constructor input is normalised, or a headless
// relocation resets movement. Arrival does not call this: a mover that reaches
// its target mid-stride finishes paying the crossing, then keeps its inputs.
func (e *Entity) clearTransit() {
	e.Transit, e.TransitTotal = 0, 0
	e.clearStride()
}

// Dead reports whether e's health has fallen below zero.
//
// A dead entity is not advanced and is not given an order. It contributes
// occupancy while it is dwelling or remains a restorative target — see counted
// — and it IS eventually removed from the world, on the tick its decay reaches
// the bottom of the ladder; until then dying changes its health, its stage and
// its dwell, and nothing else about who the world holds.
func (e Entity) Dead() bool { return e.HP < 0 }

// Downed reports whether e stands at exactly zero health with a health system —
// the state between alive and dead.
//
// A downed entity is not advanced and is not given an order, and it KEEPS ITS
// CELL: it is a body in the way. A further blow can finish it, while Heal or a
// script health write can restore it above zero. An entity whose MaxHP is not
// positive is never downed, because zero health is not a wound to a unit that
// has no health system.
func (e Entity) Downed() bool { return e.HP == 0 && e.MaxHP > 0 }

// Alive reports whether e is neither dead nor downed, which is what the third
// state IS rather than a third comparison beside the other two.
//
// Written this way the three are pairwise exclusive and jointly total by
// construction: Dead and Downed cannot both hold — one wants a negative health
// and the other exactly zero — and this one is the complement of their union, so
// no pair of integers can be in two states or in none.
func (e Entity) Alive() bool { return e.HP >= 0 && (e.HP != 0 || e.MaxHP <= 0) }

// OrdinaryTargetable reports whether an ordinary weapon, attack order or spell
// may still name e. Bodies remain finishable through -9; -10 is the first decay
// threshold at which they stop being combat subjects. Control Spirit owns its
// separate bones-only exception at the spell call sites.
func (e Entity) OrdinaryTargetable() bool { return e.HP > decayBonesHP }

// restorativeTargetable is the health population an ordinary restorative row
// may raise. It is shared by spell admission and occupancy: a body which can
// become living at its current coordinates must keep its layer's cell, or a
// mover can take that cell first and the restoration creates two living actors
// at one hashed position. The finished-body floor remains the ordinary target
// floor, and a unit without a positive health maximum has no health system to
// restore.
func (e Entity) restorativeTargetable() bool {
	return e.MaxHP > 0 && e.OrdinaryTargetable()
}

// Dying reports whether e has fallen and its dying time has NOT yet run out:
// the window in which a body lies where it fell, before anything walks its
// health down and before it is torn down.
//
// IT IS NOT A FOURTH STATE. Alive, Downed and Dead stay pairwise exclusive and
// jointly total over the health; this is a property of a not-alive entity's
// DECAY, asked separately, and every unit it holds of is already Downed or Dead.
//
// This is the minimum occupancy window. A restorable body may keep its cell
// longer, through restorativeTargetable, so expiring Dwell cannot let a mover
// take the coordinates before Heal revives it.
func (e Entity) Dying() bool {
	return !e.Alive() && e.Decay == DecayFallen && e.Dwell > 0
}

// Restorable reports whether e has fallen and Heal can still raise it: health
// 0 through -9 with a health system (MAGIC-TARGET-017, HERO-REVIVE-068). It
// outlasts Dying and ends when a blow or the corpse walk reaches -10.
func (e Entity) Restorable() bool { return !e.Alive() && e.restorativeTargetable() }

// stallLimit is the count at which a unit gives up: the target is cleared and
// the count returns to zero inside that same tick, so this value is a bound a
// stored count never reaches and not one it sits at.
//
// It is ours and provisional — no such counter exists in what is being
// reconstructed — and it is one number in one place, so the rule that spends a
// count and the check that refuses a stored one cannot come to disagree.
const stallLimit = 16

// Bounds is the map's cell extent, in the same width and signedness as a
// position so that a clamp compares like with like. It is carried through the
// canonical form, and it is what a passability grid is sized and indexed
// against: the grid describes exactly the cells inside these bounds, row-major
// from (0,0).
type Bounds struct {
	Width, Height int32
}

// Mode selects how a route is searched for, and nothing else: the resolution
// order, the cost model and the neighbour set are the same under both.
//
// It is canonical state — it is carried by the byte form and enters the digest —
// because a value that steers routing from outside the encoding would make every
// recorded digest and every replay silently invalid across a switch, and the
// divergence would read as a determinism fault rather than as a mode change. It
// is fixed when a world is built and does not change while that world is
// advanced.
//
// ModeCanonical is the zero value deliberately: a caller who says nothing must
// get the reconstruction rather than an invention of ours, so fidelity is the
// default and quality is opt-in. Every other byte is refused — here and on
// decode alike — rather than folded into one of these two.
type Mode uint8

const (
	// ModeCanonical takes the route the reconstructed procedure yields, cheaper
	// than the alternatives or not.
	ModeCanonical Mode = 0
	// ModeOptimised takes a route of minimum total cost, which is ours by
	// choice and reproduces nothing.
	ModeOptimised Mode = 1
)

// defined reports whether m is one of the two modes this build knows. It is the
// single place that says which bytes are modes, so a construction and a decode
// cannot come to differ on it.
func (m Mode) defined() bool { return m == ModeCanonical || m == ModeOptimised }

// The passability grid's per-cell flags: one byte per in-bounds cell, bit 0
// blocking a ground mover and bit 1 blocking an air mover.
//
// gridReserved is every other bit. They are refused rather than masked away,
// which is what keeps the encoding injective: were they cleared on the way in,
// two different grids would produce one world and a pinned byte form would stop
// meaning exactly one.
const (
	blockGround       byte = 1 << 0
	blockAir          byte = 1 << 1
	blockMagicWall    byte = 1 << 2
	blockStaticObject byte = 1 << 3 // SAV-aware domain-2 predicate only

	gridReserved = ^(blockGround | blockAir | blockMagicWall)
)

// Domain is which movement domain a mover belongs to, and it is the ONE value
// this package branches on to decide where a mover may go. There are exactly
// three and there is no fourth: a byte outside them is refused by the
// constructor and by the decoder alike, never clamped, masked or folded into one
// of these — the same trade Mode makes, and for the same reason, since two
// values that behaved alike would map two byte forms onto one world.
//
// DomainGround is the zero value DELIBERATELY. A caller who names no domain gets
// the ordinary ground mover, which is what every entity in this package was
// before this type existed, so a widening cannot silently reinterpret anything
// already written.
//
// The three are not a scale and the middle one is not "half flying".
// DomainGhost is a domain of its own: no terrain of this plane closes a cell to
// it except the border, and it contends with GROUND movers rather than with
// air ones. It is here because the thing being reconstructed has three mover
// domains and not two, and a boolean would have to lose one of them.
type Domain uint8

const (
	// DomainGround is the ordinary mover: the whole derived plane closes cells
	// to it, and it contends on the ground layer.
	DomainGround Domain = 0
	// DomainGhost passes what stops a ground mover and is stopped by the border,
	// and it contends on the ground layer.
	DomainGhost Domain = 1
	// DomainAir is the flyer: the border alone stops it, and it contends on the
	// air layer.
	DomainAir Domain = 2
)

// defined reports whether d is one of the three domains this build knows. It is
// the single place that says which bytes are domains, so a construction and a
// decode cannot come to differ on it.
func (d Domain) defined() bool {
	return d == DomainGround || d == DomainGhost || d == DomainAir
}

// blocks is the grid bits that close a cell to a mover of this domain: bit 0 for
// a ground mover, bit 1 for the other two.
//
// It is a MASK and not a predicate, so the terrain test is one AND wherever it
// is asked, and the three domains' terms are written down in one place rather
// than as three arms of a test that could come to disagree.
//
// The ghost and the air mover read the SAME bit here, and that is the plane's
// doing rather than the domain's: what stops a ghost is a static object, this
// derivation carries no bit for one, and the only term of the border it does
// carry besides bit 0 is bit 1. So the two differ in this package by their layer
// alone until an object bit exists.
func (d Domain) blocks() byte {
	if d == DomainGround {
		return blockGround | blockMagicWall
	}
	if d == DomainGhost {
		return blockAir | blockMagicWall
	}
	return blockAir
}

func (d Domain) blocksForSaved(saved bool) byte {
	if saved && d == DomainGhost {
		return blockStaticObject | blockMagicWall
	}
	return d.blocks()
}

// layer is which occupancy plane a mover of this domain contends on: 1 for the
// air layer, 0 for the ground layer, which carries ground and ghost movers
// together.
//
// It is an INDEX and not a bool because it multiplies the plane's cell count to
// reach a slot, and a bool at that site would have to be converted at every
// read.
func (d Domain) layer() int {
	if d == DomainAir {
		return 1
	}
	return 0
}

// gridCells is how many cells a grid for b carries: W*H, and zero when either
// bound is not positive, since such a world has no in-bounds cell to describe.
//
// The product is taken in int64 so that two int32 extents cannot wrap into a
// small, plausible count that a wrong-length grid would then match.
func gridCells(b Bounds) int64 {
	if b.Width <= 0 || b.Height <= 0 {
		return 0
	}
	return int64(b.Width) * int64(b.Height)
}

// newGrid returns the grid a world over b holds, given the grid its builder
// passed: a copy of that grid when it names one, and an all-zero grid of
// gridCells(b) bytes when it does not.
//
// An absent grid is materialised here rather than recorded as absent. That makes
// "a world with no grid behaves as one with an all-zero grid" and "the two have
// identical byte forms" one representation instead of two kept in agreement —
// nothing downstream can ask which was passed, because nothing stores it.
//
// The grid is copied for the same reason the entity slice is: the caller's slice
// stays the caller's to reuse or mutate, and a grid cannot change under a world
// that is being advanced.
func newGrid(b Bounds, grid []byte, savedMode ...bool) ([]byte, error) {
	cells := gridCells(b)
	if len(grid) == 0 {
		return make([]byte, cells), nil
	}
	if int64(len(grid)) != cells {
		return nil, fmt.Errorf("sim: grid carries %d cell(s), want %d for bounds %dx%d",
			len(grid), cells, b.Width, b.Height)
	}
	reserved := byte(gridReserved)
	if len(savedMode) != 0 && savedMode[0] {
		reserved &^= blockStaticObject
	}
	for i, c := range grid {
		if c&reserved != 0 {
			return nil, fmt.Errorf("sim: grid cell %d is %#02x: reserved block bits must be zero", i, c)
		}
	}
	return append([]byte(nil), grid...), nil
}

// defaultCost is the cost byte an ABSENT cost plane materialises at, and it
// is derived from the rate law's own zero-mean substitute rather than
// written as a second 8.
//
// The two are the same number for the same reason — it is the value most of the
// original's own cells carry — and spelling it twice is exactly how a later
// story comes to move one and not the other.
//
// The value is what makes a world built with NO cost plane behave precisely as
// this package did before a cost plane existed (FR-1a), and the argument is
// arithmetic rather than empirical, in both consumers:
//
//   - the RATE divides by the two cells' mean, and a uniform plane of this value
//     hands it the very number its zero-mean substitute produced;
//   - the SEARCH charges a ground mover `c` orthogonally and `c + c>>1`
//     diagonally, so a uniform plane scales every label by exactly c/2 against
//     the flat 2 and 3 it charged before. Every label is a sum of step costs, so
//     the whole plane scales by that one factor, and every comparison in a
//     search is label against label or against the unlabelled marker — never
//     against a cost. A scaled order is the same order, so the same routes come
//     out, with the same ties broken the same way.
//
// The second half is the one that has to be checked rather than believed, and it
// is checked: the marker is 0 and a label is held as cost+1, so even a cost of 0
// cannot produce one.
const defaultCost = fallbackMeanCost

// Terrain is the three per-cell planes a world stands on: what closes a cell to
// a mover, what it costs a GROUND mover to enter, and how high it stands.
//
// It is a struct with named fields rather than three more parameters on the
// constructor, and that is not tidiness. Three adjacent parameters of one
// slice type are three a caller can transpose with no compiler complaint and
// no failure except a wrong route somewhere much later; named fields cannot
// be transposed. The routing mode stays positional beside it, because the
// danger there is the opposite one — a value silently DEFAULTED — and a
// field in a struct literal is exactly what makes defaulting easy.
//
// Each field is either empty — the plane is not named, and is materialised — or
// exactly one byte per in-bounds cell, row-major from (0,0), in the same order
// and at the same length as the other two. Any other length is refused.
//
// Block carries the passability bits; Cost carries the per-cell entry cost a
// ground mover pays; Height carries the altitude whose DIFFERENCE between two
// cells tilts the rate. Every byte value is legal in Cost and in Height: they
// are read as numbers, not as flags, so there is no reserved bit to refuse and
// no value that is not a state a caller may build.
type Terrain struct {
	Block  []byte
	Cost   []byte
	Height []byte
}

// newPlane returns the plane a world over b holds, given the plane its builder
// passed and the byte an absent one is filled with.
//
// It is newGrid's rule for the two planes that carry no flags: copy what was
// named, materialise what was not, refuse any other length. It does not check
// bits, because neither plane has any — which is the whole reason it is a second
// function rather than a parameter on the first.
func newPlane(b Bounds, plane []byte, absent byte, what string) ([]byte, error) {
	cells := gridCells(b)
	if len(plane) == 0 {
		out := make([]byte, cells)
		if absent != 0 {
			for i := range out {
				out[i] = absent
			}
		}
		return out, nil
	}
	if int64(len(plane)) != cells {
		return nil, fmt.Errorf("sim: %s plane carries %d cell(s), want %d for bounds %dx%d",
			what, len(plane), cells, b.Width, b.Height)
	}
	return append([]byte(nil), plane...), nil
}

// World is the canonical runtime state: a monotonic tick, one seeded integer
// RNG, the map's cell bounds, the routing mode, the three per-cell planes over
// those bounds, and the entities in ascending id order.
//
// Every field is unexported, so the type has no writer to find: nothing outside
// this package can set a position, the tick, the bounds, the mode, a grid cell
// or the RNG state individually. Readers are methods, and the ones that hand out
// entities hand out copies.
//
// The grid is always materialised — a world built with none holds an all-zero
// grid of the same size — so there is no absent case anywhere below the
// constructor, and no flag recording which way a world was built.
//
// The entities are a slice kept sorted by id rather than a map. Go randomises
// map iteration order per process, so a map anywhere on a path that touches a
// world would be a source of nondeterminism that the stdlib-only import rule
// cannot see.
//
// routes runs PARALLEL to the entities: routes[i] is the route entities[i] is
// walking, and an empty one means it holds none. It is canonical state, not a
// cache — a route is not recoverable from the cells it was computed from, so two
// worlds alike in every other field but differing in a stored route walk apart
// from each other. That is why it is carried by the byte form, enters the
// digest, and is declared in TestTheCanonicalWorldsFieldSetsArePinned's table:
// what that pin refuses is the field nobody declared, which is exactly what a
// cache here would be.
//
// It is a field of the WORLD and not of the Entity for the same reason the
// entity record is what it is: every field of an Entity is an integer or a
// bool, so an Entity is copied by assignment and Entities() can keep handing
// out copies that reach nothing of the world. A slice on the record would
// end that for every caller of Entities(). groups is the WORLD's own
// per-group record: one per owned (owner, group) pair, ascending, each
// carrying that group's notice base FROZEN when the world was built, the
// ORDER it is under, and the CELL a group command last named.
//
// The key set is every OWNED entity's (owner, group) pair, alive or not —
// an entity in roster slot 0 belongs to no group and contributes none —
// and it is fixed for the world's whole life: nothing here appends an
// entity, unlinks a felled member from its group, or moves one between
// groups, so the set the constructor builds is the set every later tick
// still holds.
//
// The base is the group's LIVING members' widest reach at the moment the
// world was built — the geometry this package has always computed, kept here
// rather than discarded at the end of one decision. IT IS WHAT THE CLIP READS:
// a guarding group's circle is this byte widened by the arm's margin, so the
// circle's centre follows the group every decision and its size does not.
//
// THE ORDER IS THE ONE THING A DECISION FORKS ON: construction writes it
// from the owner's slot — Stand Ground for the local participant's own
// groups, Guard for every other owned one — and nothing on any tick path
// derives it from an owner again. A group command is the only other writer,
// and this story builds none yet, so every record this task produces holds
// one of the two values construction gives it. THE COMMANDED CELL starts at
// the origin, unread until a command writes one.
//
// A PAIR THIS SET DOES NOT NAME decides NOTHING — not a base of zero fed
// to the clip, which is what an earlier story read this record for before an
// order existed to answer with. The mission script's hand-over arms write an
// entity's owner inside a tick, so a pair can come into existence after the
// freeze; a group object that has never had guard installed carries a zeroed
// record, and a decision for it stops at that zero rather than re-deriving
// anything from the (new) owner.
//
// It is canonical: carried by the byte form, between the routes and the
// script section, and hashed with everything else — two worlds differing only
// in one base, one order or one commanded cell ARE two worlds, because the
// divergence this record exists to remove is a fact about the world and not
// about whoever asks it a question.
//
// script, registers, latches, won, lost and outcome are THE MISSION SCRIPT AND
// ITS STATE, and admitting them is the declaration that a map's authored script
// is part of the world rather than something laid over it.
//
// The four state fields have to be canonical: a register is written by a check
// every pass and read by a trigger, a latch is what stops a one-shot trigger
// firing twice, and the two counters are the only thing that decides a mission.
// A world resumed without them re-fires every one-shot trigger it had already
// spent, which is the exact defect the original's own save avoids by writing all
// four.
//
// THE COMPILED SCRIPT IS CANONICAL TOO, AND THAT IS OURS. The original re-derives
// it from the map at every load and stores only the volatile half; a world here
// carries no map, so a script left outside the byte form would be a fact about
// how the world behaves that no digest and no round trip could see — which is
// exactly what the field-set pin exists to refuse. Carried, a world is
// self-contained and two worlds differing only in their script are two worlds.
// cost and height are the other two planes, canonical on the same terms the
// grid is and materialised on the same terms: a world that named neither holds
// a uniform defaultCost plane and an all-zero height plane, so "named none" and
// "named those" are one world in the fields, in the bytes and in the digest.
//
// They are READ while a world is advanced — cost at every offer a ground
// search makes and at every transit start, height at every transit start — which
// is what admits them to the field-set pin rather than making them a second
// source it would refuse.
// relations is WHO IS HOSTILE TO WHOM, by roster slot, and it is canonical on
// exactly the terms the three planes are: it is carried by the byte form, it
// enters the digest, it is fixed when a world is built and it does not change
// while that world is advanced. It is materialised on the same terms too, so a
// world that named none and a world that named the all-zero matrix are one world
// in the fields, in the bytes and in the digest.
//
// It is READ while a world is advanced — once per group per decision, at the
// filter that turns everything a group can see into the things it will fight —
// which is what admits it to the field-set pin rather than making it a second
// source that pin would refuse.
//
// sacks is the map's own authored loot, one entry per occupied cell,
// ascending by (Y, X) — the one-per-cell merge normaliseSacks performs
// before a world is ever built. It is canonical on the group list's own
// terms: carried by the byte form, between the group section and the script
// section, and hashed with everything else, so two worlds differing only in
// one sack's gold or contents are two worlds (AC-5) even though NOTHING ON A
// TICK PATH YET READS ONE — the criterion this record's neighbours were
// admitted on is whether a fact about the world would be invisible to the
// byte form, the digest and a replay, not whether a step consults it, and a
// sack clears that bar the day it is placed rather than the day something
// reads it.
//
// IT IS COPIED, IN THE CALLER'S OWN ORDER: unlike sacks it is NOT sorted,
// because that order is the book a front end reads spells out in, and
// reordering it here would take it away from whoever built the table. A
// repeated id, an id of 0, a negative mana cost and a negative damage column
// are refused rather than folded, on Domain's own ground: two rows
// KindCast's own linear lookup could not tell apart would map two tables
// onto one world.
//
// carried AND purses are the newest fields (0112 T1), and they are declared
// here a task AHEAD of the byte form on purpose: the plan's own reason for
// splitting the container from its byte form into two tasks is that a world
// field outside the form is a landable state, and what THIS comment records
// — what the canonical world holds — is true the moment the constructor
// and the reader exist, not the moment the bytes do.
//
// carried is entities' OWN SHAPE: one slice per entity in entity order, on
// routes' own ground — a container is not recoverable from anything else
// the world holds, so it is stored rather than derived, and an entity given
// none still has a slot, an empty slice rather than an absence, so "a world
// holds nothing" and "carried[i] belongs to entities[i]" are one
// representation instead of two kept in agreement.
//
// EACH ENTRY IS A LIST OF ELEMENTS AND NOT OF CODES: an ItemStack is one
// code with the COUNT of it held there, at least 1, and a container holds at
// most one element per code — foldContainer, in carry.go, is the one place
// that invariant is established and every act that puts an item in ends
// there. The count is canonical simulation state on the container's own
// three grounds: it is not recoverable from anything else the world holds,
// it survives the byte form, and it enters the digest.
//
// THE BYTE FORM DID NOT MOVE FOR IT AND formatVersion DID NOT EITHER. The
// carry section's bytes are "a code count, then that many codes" and a stack
// is a GROUPING of those codes rather than a new field, so the encoder
// writes the flat expansion and the decoder folds what it reads: every
// offset, every width and every existing payload's meaning are exactly what
// they were. What CAN differ from the previous build is the ORDER a
// container is left in by a take, a give-all or an equip, and where it does
// that world's digest differs — a change of behaviour, not of form.
//
// purses is money BY ROSTER SLOT and not by entity: money is Player+0x38,
// the roster's own rather than a container's, and it is indexed by the owner
// word exactly as relations is — a fixed array and not a map, on
// relations' own ground, so a world's contents never depend on Go's
// randomised iteration order. UNLIKE relations, owner 0 indexes a REAL cell
// rather than naming no slot: cell 0 is where no gold is destroyed if it is
// ever credited to an entity outside the roster, which costs nothing since a
// query naming slot 0 answers the same way either side of that choice.
//
// equipment is THE NEWEST FIELD (0124 T2): an entity's WORN item codes, on
// carried's own three grounds — not recoverable from anything else the
// world holds, so stored rather than derived; entered by the byte form and
// the digest; and canonical from the moment the constructor and the reader
// exist, which for this field is the same task rather than a task ahead of
// it, because equipment has no separate landable half the way the
// container did (0112's own two-task split existed for a byte-form task
// that had not yet been written; this one is that task already).
//
// IT IS A FIXED EquipSlots-WIDE ARRAY PER ENTITY AND NOT A SLICE, unlike
// carried: a slot's NUMBER is the whole of what it means, and an empty slot
// has to be representable AT ITS OWN INDEX — a slice compacted on every
// empty entry could not do that, the way a container compacted on every
// empty stack legitimately can. ZERO IS EMPTY, AND IT IS THE ONLY SPELLING
// OF EMPTY, the opposite of the trade a carried code makes: carryFault's own
// doc names why a container can never legitimately hold a zero, and an item
// code's own class field never resolves to a real item at row 0 either —
// so an equipment slot can never legitimately hold anything BUT zero at
// rest, and decodeEquipment refuses nothing a slot may carry.
//
// It is materialised for every entity by the constructor, on carried's own
// reason: "a world holds nothing worn" and "equipment[i] belongs to
// entities[i]" are then one representation instead of two kept in agreement,
// and no reader has a length to check first. NOTHING BUILDS ONE FILLED YET:
// the constructor takes no equipment argument at all, so every world THIS
// PACKAGE CONSTRUCTS holds every entity unarmed at every slot; what a
// mission STARTS a character wearing is the loader's own fact, read by a
// later task's recompute, and is not state this package is handed at
// construction. A DECODED world can differ, because a byte form carries
// whatever equipment it was cut with.
type World struct {
	tick              uint64
	damageObservation *damageObservation
	turnSteps         map[EntityID]struct{}
	turnStepScope     bool

	// A source session uses tick as its uint32 SubTick and keeps FullTick
	// independently. Without this presence bit the historical native uint64
	// clock and its scheduling policy remain in force (DIV-778).
	hasSessionClock bool
	fullTick        uint32

	rng            rng
	rules          Rules
	bounds         Bounds
	mode           Mode
	grid           []byte
	cost           []byte
	height         []byte
	relations      Relations
	entities       []Entity
	actorTraversal []EntityID
	// Highest initial, retired or explicitly reserved identity plus one.
	// Live later spawns also bound NextEntityID until remove retains their ID.
	entityIDFloor uint64
	routes        [][]cell
	groups        []groupAI
	savedGroups   *savedGroupState
	sacks         []Sack
	spells        []SpellRule
	ghost         GhostTemplate
	sourceDerive  SourceDerive
	safeMode      bool
	carried       [][]ItemStack
	equipment     [][EquipSlots]ItemInstance
	purses        [relationSlots]uint32

	// burst holds the Fire_Ball burst's installed phase count and the blasts of
	// the current effect walk that have not yet built their record. Neither is
	// saved: the count is install-derived input like ghost, carried across a
	// decode, and the queue is empty between ticks (burst.go).
	burst burstState

	// itemWeights is what one unit of each item code this world can name
	// weighs, sorted by code (weight.go). It is a TABLE the world is handed,
	// like the spell table above it, and not state a tick writes:
	// DeclareItemWeights is its one writer and nothing in this package calls
	// it.
	itemWeights []ItemWeight

	script    *Script
	rom2      *rom2ScriptState
	registers [scriptRegisters]int32
	latches   [scriptLatches]byte
	won       uint32
	lost      uint32
	outcome   Outcome

	// rawSessionHead and rawSessionMid are the two opaque session-block spans
	// ImportOriginalSession carries in from an original save and nothing in
	// this package reads (SAV-SESS-031, docs/1130/story.md).
	rawSessionHead [sessionRawHeadLen]byte
	rawSessionMid  [sessionRawMidLen]byte

	// THE SCRIPT'S TWO NEW KINDS OF STATE (0165). casts is the mission
	// script's pending casts — the temporary casters instants 21 and 24
	// build, resolved at the head of a later tick (scriptcast.go). effects
	// is the area effects standing on cells, kept sorted by cell key and
	// within a key in arrival order (celleffect.go).
	//
	// NEITHER IS AN ENTITY and neither is in any plane. Both are in the byte
	// form and therefore in the digest.
	casts     []scriptCast
	bookCasts []bookCast
	// engageDrew lists the creatures whose spell draw ran in the current
	// decision pass. It is scratch, reset at the start of the pass, and is in
	// neither the byte form nor the digest.
	engageDrew  []EntityID
	deliveries  []spellDelivery
	scrollCasts []ScrollCast
	effects     []cellEffect
	attached    []attachedEffect

	// THE SCRIPT'S TWO MORE KINDS OF STATE (0166). formations is the
	// per-player formation mode instant 7 writes and the group-move
	// distribution reads (formation.go); cellTails is the six-byte tail
	// instant 25 writes onto a cell's own record, kept sorted by cell key
	// (celltail.go).
	//
	// formations is an ARRAY and not a slice, subscripted exactly as purses
	// above it: the roster is a fixed width in this package and every slot
	// exists whether or not a player stands on it. Both are in the byte form
	// and therefore in the digest.
	formations    [relationSlots]uint8
	autoHealing   [relationSlots]autoHealingPolicy
	cellTails     []cellTail
	scorchedCells []uint16
	structureUses []StructureUse

	// areaCosts is the stored movement-cost byte of every cell holding an
	// applied area layer, in cell-key order (areacost.go). areaCostLive is
	// false until the first area-effect pass after construction or load.
	areaCosts    []areaCostEntry
	areaCostLive bool
	// costWindow holds a step's layer cost writes on a saved cell plane
	// (layercostwindow.go). It exists only inside one step.
	costWindow *savedCostWindow

	structures []Structure
	// Derived immutable cell aliases; HP changes do not detach a ruin.
	structureSlots        map[uint16]int
	nativeStructurePlanes bool
	// 1114: absent legacy mode derives slots from placements. Present mode
	// retains the source roster and explicit saved cell links, including empty.
	hasSavedStructures    bool
	savedStructures       []SavedStructure
	savedStructureCells   []SavedStructureCell
	originalDead          []originalDeadRecord
	currentTerminalActors []CurrentTerminalActor
	removedNativeBases    []NativeActorBasisRecord
	currentPlayers        *currentPlayerState
	savedMotion           *savedActorMotionState
	savedCellPlanes       *SavedCellPlanes
	savedObjects          *SavedObjects

	// 1131: the cell-record residue no earlier story carries in its own typed
	// form (savedcellrecord.go).
	savedCellRecords []SavedCellRecord

	// 1132: the top-level SpellEffect-list graph no earlier story carries
	// (savedspelleffect.go). DIV-938/DIV-939's own question — shared
	// SpellEffect identity inside the graph — is about what the graph means,
	// not whether it survives a decode, and stays open.
	savedSpellEffects []SavedSpellEffect

	// 1133: the Projectiles state-store subtree (savedprojectile.go).
	//
	// DIV-944
	savedProjectiles  SavedProjectiles
	savedWorldEffects *SavedWorldEffects
	savedSpellGraph   *SavedSpellGraph
	effectOrder       []WorldEffectRef
	effectWalking     bool

	// 1135: every Diary the file carries, bound to its owning Player or
	// living actor (saveddiary.go). Same class and same reason again: STORY
	// 1139 (Form85) gives it a wire position too (carriedresumebinary.go,
	// DIV-956).
	savedDiaries []SavedDiary

	diary diaryRuntime
}

// NewWorld returns a world seeded with seed, bounded by b, routing by mode, over
// grid, and holding a copy of ents sorted by ascending id.
//
// It is the only constructor, and mode and grid are positional rather than
// optional, so no caller can leave either to a default it did not choose: a
// second constructor or an options struct would let the mode be defaulted by
// accident, and the mode is the one value a replay cannot survive being wrong
// about.
//
// grid is either empty — no grid, which materialises as an all-zero grid of
// gridCells(b) bytes — or exactly that many row-major bytes, one per in-bounds
// cell. Any other length, any cell with a reserved bit set, any mode byte this
// build does not define and any entity in a MOVEMENT DOMAIN it does not define
// are refused rather than padded, truncated, masked or folded: a refused
// construction returns no world at all, so no caller can observe a world
// carrying part of a rejected input.
//
// Both slices are copied, so a caller may reuse or mutate either one it passed.
// Ids must be unique — duplicates are one of this constructor's errors — which
// with the sort makes ids strictly ascending in every world that exists. An
// entity whose HasTarget is false has its target coordinates AND its stall count
// zeroed, so a world's contents depend on the logical state alone and never on
// the residue of a target that was cleared. An entity that is NOT ALIVE has the
// whole of its order cleared by the same rule and for the same reason: such a
// unit takes no order, so a target on one is residue of a state it has left.
//
// Neither health field is defaulted. A world built here keeps exactly the pair
// it was handed, so an entity built with neither named is at 0/0 — alive, with
// no health system — and every world assembled before those fields existed keeps
// its state and its digest. Where a spawn's health comes from is the spawn
// path's question, not this constructor's.
//
// A stall count at or above stallLimit is refused rather than clamped. That
// state is one the byte form refuses, so a constructor that produced it would
// build a world this package cannot marshal and read back — an asymmetry no
// caller could do anything about.
//
// AN ATTACK ORDER is normalised away on a unit that is not alive, on one naming
// itself and on one naming a victim this world does not hold, and the cycle
// fields are zeroed on any unit holding no order. A phase this build does not
// define and a negative count owed are refused instead: those two have no value
// to fold onto, which is the same line the stall count and the transit pair
// already draw.
//
// THE ACTOR STATE is written into every entity unconditionally, at guard,
// and that is not a normalisation of an input: a caller's own value is
// discarded rather than read, because the state is not an input this
// constructor accepts. A ring or a leg named alongside it goes with it, on
// the stall count's own ground — residue of an order not, or no longer, in
// force.
//
// The RNG state is the seed, so it follows from the seed alone and from nothing
// nondeterministic: this package reads no clock and owns no process-global
// generator.
//
// A world built here holds NO route for any entity, and the parameter list is
// what makes that unconditional: there is no sixth parameter to pass one in
// with. Every caller would have to hand over a nil it did not think about, and
// what that would buy is a way to give a world a route nothing searched for —
// one whose cells no rule of this package ever checked. The two writers of a
// route are Step and UnmarshalBinary, and a decoded route is checked cell by
// cell on the way in.
// NewWorld returns a world running NO MISSION SCRIPT, and is exactly
// NewScriptedWorld with none. It is kept as its own name rather than folded away
// because a script is the one input most worlds do not have: ten of the shipped
// corpus's loose maps author no winning instant at all, and every world this
// package built before scripts existed keeps the call it was written with.
func NewWorld(seed uint64, b Bounds, mode Mode, grid []byte, ents []Entity) (*World, error) {
	return NewScriptedWorld(seed, b, mode, grid, ents, nil)
}

// NewTerrainWorld is the ROOT constructor: NewScriptedWorld over all three
// per-cell planes rather than the block plane alone.
//
// The other two are kept as their own names rather than folded away, for the
// reason NewWorld already is: most worlds this package builds name no cost and
// no height plane, every world built before either existed named none, and an
// unnamed plane is materialised to exactly the behaviour such a world had. So
// the two older names are not a compatibility shim — they are the honest
// signature for a caller with nothing to put in the other two fields.
func NewTerrainWorld(seed uint64, b Bounds, mode Mode, t Terrain, ents []Entity, s *Script) (*World, error) {
	return newWorld(seed, b, mode, t, ents, s, Relations{}, nil, nil, nil, GhostTemplate{}, nil)
}

// NewRelatedWorld is the ROOT constructor: NewTerrainWorld over a world
// whose roster slots hold a relation, rather than one in which nobody is
// hostile to anybody.
//
// It is a fourth name rather than a sixth parameter on the third, for the reason
// the third is a second name rather than three more parameters on the first:
// most worlds this package builds name no relation, every world built before one
// existed named none, and an unnamed relation is materialised to exactly the
// behaviour such a world had — nothing acquires. So the three older names are
// not a compatibility shim, they are the honest signature for a caller with
// nothing to put in this slot.
//
// The relation is POSITIONAL, like the mode, the grid and the script, and it is
// not a fourth field of Terrain. Terrain is the per-cell planes and this
// describes players; and a struct field is exactly the shape that makes a value
// easy to leave out by accident, which is the danger the planes' own doc names
// on the other side.
func NewRelatedWorld(seed uint64, b Bounds, mode Mode, t Terrain, ents []Entity, s *Script,
	rel Relations) (*World, error) {
	return newWorld(seed, b, mode, t, ents, s, rel, nil, nil, nil, GhostTemplate{}, nil)
}

// NewScriptedWorld returns a world as NewWorld does, running the compiled
// mission script s.
//
// The script is POSITIONAL like the mode and the grid, and for the same reason:
// there is no options struct to leave it out of by accident. A nil script and an
// empty one build the same world — an empty script has nothing to evaluate, so
// keeping the two apart would be a distinction the byte form, the digest and
// every tick are blind to.
//
// The build-time constants are applied HERE and only here: every constant check's
// register takes that node's own first value, which is the whole of what a
// mission variable's initial value is. Nothing else presets a register, so a
// world decoded from bytes keeps the registers it was cut with.
func NewScriptedWorld(seed uint64, b Bounds, mode Mode, grid []byte, ents []Entity, s *Script) (*World, error) {
	return NewSpelledWorld(seed, b, mode, grid, ents, s, nil)
}

// NewSpelledWorld returns a world as NewScriptedWorld does, additionally
// holding the spell table spells: KindCast's own rows.
//
// spells is POSITIONAL, on the script's own ground beside it: there is no
// options struct for a caller to leave it out of by accident, and the table
// is kept in the order the caller gave it (newWorld's own doc on the field).
func NewSpelledWorld(seed uint64, b Bounds, mode Mode, grid []byte, ents []Entity, s *Script,
	spells []SpellRule) (*World, error) {
	return newWorld(seed, b, mode, Terrain{Block: grid}, ents, s, Relations{}, nil, nil, spells, GhostTemplate{}, nil)
}

// NewLootWorld is the ROOT constructor: NewRelatedWorld over the map's own
// ground sacks rather than none.
//
// It is a fifth name rather than a seventh parameter on the fourth, for the
// reason the fourth is a fourth name rather than a sixth parameter on the
// third: most worlds this package builds name no sack, every world built
// before one existed named none, and an unnamed sack list is materialised to
// exactly the behaviour such a world had — nothing on the ground. So the four
// older names are not a compatibility shim, they are the honest signature for
// a caller with nothing to put in this slot, and every constructor still
// funnels through newWorld, so no rule about what a world may hold can differ
// by the way it was built.
//
// sacks is POSITIONAL, like the relation beside it, and for the same reason:
// an options struct or a post-construction setter would let a world exist in
// a state its own constructor did not choose. It need not be sorted or
// merged — normaliseSacks does both — so a caller may hand over the
// map's records in file order exactly as it read them.
func NewLootWorld(seed uint64, b Bounds, mode Mode, t Terrain, ents []Entity, s *Script,
	rel Relations, sacks []Sack) (*World, error) {
	return newWorld(seed, b, mode, t, ents, s, rel, sacks, nil, nil, GhostTemplate{}, nil)
}

// NewStockedWorld is the ROOT constructor: NewLootWorld over the map's own
// stock — actors handed a starting container at construction, named by
// Stock rather than left to acquire one.
//
// It is a sixth name rather than an eighth parameter on the fifth, for the
// reason the fifth is a fifth name rather than a seventh parameter on the
// fourth: most worlds this package builds name no stock, every world built
// before one existed named none, and an unnamed stock list is materialised
// to exactly the behaviour such a world had — every entity carrying
// nothing. So the five older names are not a compatibility shim, they are
// the honest signature for a caller with nothing to put in this slot, and
// every constructor still funnels through newWorld, so no rule about what a
// world may hold can differ by the way it was built.
//
// stock is POSITIONAL, like the sacks beside it, and for the same reason:
// an options struct or a post-construction setter would let a world exist
// in a state its own constructor did not choose. It need not be pre-merged
// — normaliseHoldings folds two Stock naming one id together, in argument
// order — so a caller may hand over the map's records in file order exactly
// as it read them.
func NewStockedWorld(seed uint64, b Bounds, mode Mode, t Terrain, ents []Entity, s *Script,
	rel Relations, sacks []Sack, stock []Stock) (*World, error) {
	return newWorld(seed, b, mode, t, ents, s, rel, sacks, stock, nil, GhostTemplate{}, nil)
}

// NewStockedSpelledWorld returns a world as NewStockedWorld does,
// additionally holding the spell table spells: KindCast's own rows, the same
// table NewSpelledWorld's own ladder carries.
//
// IT IS THE UNION OF THE TWO LADDERS' TOPS, and a new name rather than an
// eighth parameter on NewStockedWorld, on every constructor above's own
// ground: most worlds this package has ever built name no spell table,
// every world built before one existed named none, and an unnamed table is
// materialised to exactly the behaviour such a world had — KindCast finds
// no row and resolves nothing. So NewStockedWorld is not a compatibility
// shim to be widened, it is the honest signature for a caller with nothing
// to put in this slot, and every constructor still funnels through
// newWorld, so no rule about what a world may hold can differ by the way it
// was built.
//
// pkg/mapload's mission-building path is the caller this exists for: it
// already carries a map's own relation, sacks and stock through
// NewStockedWorld, and a mission's spell table has to travel beside them
// rather than through NewSpelledWorld, which names none of the other three.
//
// spells is POSITIONAL, on stock's own ground beside it: there is no
// options struct for a caller to leave it out of by accident.
func NewStockedSpelledWorld(seed uint64, b Bounds, mode Mode, t Terrain, ents []Entity, s *Script,
	rel Relations, sacks []Sack, stock []Stock, spells []SpellRule) (*World, error) {
	return newWorld(seed, b, mode, t, ents, s, rel, sacks, stock, spells, GhostTemplate{}, nil)
}

// NewSummoningWorld returns a world as NewStockedSpelledWorld does,
// additionally holding the definition table's own `Ghost` row: what a Control
// Spirit cast raises (MAGIC-SING-019 (c), GhostTemplate above).
//
// It is a ninth name rather than an eleventh parameter on the eighth, on every
// constructor above's own ground: every world built before this one existed
// named no template, and an unnamed template is materialised to exactly the
// behaviour such a world had — the raise is refused at admission and no mana
// is spent. pkg/mapload's mission-building path is the caller it exists for,
// the same caller NewStockedSpelledWorld exists for.
func NewSummoningWorld(seed uint64, b Bounds, mode Mode, t Terrain, ents []Entity, s *Script,
	rel Relations, sacks []Sack, stock []Stock, spells []SpellRule, ghost GhostTemplate) (*World, error) {
	return newWorld(seed, b, mode, t, ents, s, rel, sacks, stock, spells, ghost, nil)
}

// NewStructuredWorld returns a world as NewSummoningWorld does, additionally
// holding the per-structure state check opcode 21 and instant opcode 26 read
// and write (1033 B3, structure.go).
//
// It is a tenth name rather than a twelfth parameter on the ninth, on every
// constructor above's own ground: most worlds this package has ever built
// name no structure, every world built before one existed named none, and an
// unnamed structure list is materialised to exactly the behaviour such a
// world had — a check-21 or instant-26 node measures or writes nothing,
// exactly as an unresolved reference already does. pkg/mapload's
// mission-building path is the caller this exists for, the same caller
// NewSummoningWorld exists for.
//
// structs is POSITIONAL, on the ghost template's own ground beside it: there
// is no options struct for a caller to leave it out of by accident.
func NewStructuredWorld(seed uint64, b Bounds, mode Mode, t Terrain, ents []Entity, s *Script,
	rel Relations, sacks []Sack, stock []Stock, spells []SpellRule, ghost GhostTemplate,
	structs []Structure) (*World, error) {
	return newWorld(seed, b, mode, t, ents, s, rel, sacks, stock, spells, ghost, structs)
}

// newWorld is the one body all ten constructors reach, so no rule about what
// a world may hold can come to differ between the way it was built.
func newWorld(seed uint64, b Bounds, mode Mode, t Terrain, ents []Entity, s *Script,
	rel Relations, sacks []Sack, stock []Stock, spells []SpellRule, ghost GhostTemplate,
	structs []Structure) (*World, error) {
	if !mode.defined() {
		return nil, fmt.Errorf("sim: routing mode %d is not defined", uint8(mode))
	}
	// Control Spirit copies the template domain into the actor it appends. The
	// decoder refuses an Entity outside the three defined domains, so the common
	// constructor must reject the source before a world can retain it. This is
	// unconditional: the zero GhostTemplate is not raisable, but its zero domain
	// is the valid ground domain and therefore needs no special case.
	if !ghost.Domain.defined() {
		return nil, fmt.Errorf("sim: ghost template: movement domain %d is not defined",
			uint8(ghost.Domain))
	}
	// Control Spirit is the one runtime path that mints and appends an Entity
	// without returning through this constructor. Refuse an unrepresentable
	// selector on its source template here, before the world can retain it, on
	// the same six-slot boundary every initial Entity crosses below.
	if err := experienceSlotFault(ghost.XPSlot); err != nil {
		return nil, fmt.Errorf("sim: ghost template: %w", err)
	}
	if err := ghost.NativeBasis.Validate(); err != nil {
		return nil, fmt.Errorf("sim: ghost template: %w", err)
	}
	g, err := newGrid(b, t.Block)
	if err != nil {
		return nil, err
	}
	cost, err := newPlane(b, t.Cost, defaultCost, "cost")
	if err != nil {
		return nil, err
	}
	height, err := newPlane(b, t.Height, 0, "height")
	if err != nil {
		return nil, err
	}
	loot, err := normaliseSacks(b, sacks)
	if err != nil {
		return nil, err
	}
	spellTable, err := normaliseSpells(spells)
	if err != nil {
		return nil, err
	}

	cp := make([]Entity, len(ents))
	copy(cp, ents)
	sort.Slice(cp, func(i, j int) bool { return cp[i].ID < cp[j].ID })
	for i := range cp {
		for k := range cp[i].PotionStats {
			if cp[i].PotionStats[k] < 0 || cp[i].PotionStats[k] > 100 || cp[i].PotionHeadroom[k] < 0 || cp[i].PotionHeadroom[k] > 100 {
				return nil, fmt.Errorf("sim: invalid potion state for entity %d", cp[i].ID)
			}
		}
		if i > 0 && cp[i].ID == cp[i-1].ID {
			return nil, fmt.Errorf("sim: duplicate entity id %d", cp[i].ID)
		}
		if cp[i].Stall >= stallLimit {
			return nil, fmt.Errorf("sim: entity %d has stall count %d, the limit is %d",
				cp[i].ID, cp[i].Stall, stallLimit)
		}
		// Refused rather than folded into the ground domain, which is the trade
		// the mode byte already makes: a value normalised here would make two
		// entities the byte form distinguishes into one world, and a pinned form
		// would stop meaning exactly one.
		if !cp[i].Domain.defined() {
			return nil, fmt.Errorf("sim: entity %d is in movement domain %d, which is not defined",
				cp[i].ID, uint8(cp[i].Domain))
		}
		// THE ACTOR STATE IS WRITTEN HERE, UNCONDITIONALLY: it is not an input a
		// caller supplies, so whatever value ents named is overwritten rather than
		// read. What that leaves behind is exactly patrolFault's third shape — a
		// ring or a leg on an entity that is (now) not in the patrol state — and
		// it is dropped with the state rather than kept as a fact about an order
		// this build has no way to have issued yet.
		cp[i].ActorState = actorStateGuard
		cp[i].Retreat = RetreatContinuation{}
		cp[i].PostX, cp[i].PostY = cp[i].X, cp[i].Y
		// BOTH CLEARS, not one (0166 D-13): patrolFault now answers for the
		// escort triple as well as for the ring, and a fault it names in one
		// of them is not repaired by clearing the other. Together they satisfy
		// every rule it checks, since ActorState was forced to guard above.
		if err := patrolFault(cp[i]); err != nil {
			cp[i].clearPatrol()
			cp[i].clearEscort()
		}
		// REACH IS NORMALISED HERE, a zero folded to 1, on the split reachFault's
		// own doc names: every entity literal this package's tests build today
		// carries no reach, and refusing their zero would turn all of them red
		// for a reason a reader could learn nothing from. 1 to 255 are already
		// legal as given — a uint8 has no value above 255 to refuse.
		if cp[i].Reach == 0 {
			cp[i].Reach = 1
		}
		// THE TWO REMAINDERS ARE FOLDED HERE, each one above 99 to 0
		// unconditionally, on the split reachFault's own doc names above: a caller
		// handed a stale accumulator has nothing it could do with an error. 0 to
		// 99 are already legal as given.
		if cp[i].HealthHundredths > 99 {
			cp[i].HealthHundredths = 0
		}
		if cp[i].ManaHundredths > 99 {
			cp[i].ManaHundredths = 0
		}
		// A caller that supplies no active turn supplies no second facing either:
		// normalise the inactive pair to the current facing. This keeps every
		// pre-turn Entity literal usable while still making the byte decoder strict
		// about residue in a saved form.
		if cp[i].TurnRemaining == 0 {
			cp[i].DesiredFacing = cp[i].Facing
			cp[i].TurnTotal = 0
		}
		// An entity that is not alive holds no order at all, and it is
		// NORMALISED here rather than refused: a caller handed a dead unit with a
		// stale target has nothing it could do with an error, and the residue rule
		// below already normalises where it could have refused. The decoder is the
		// half that refuses, which keeps the two in the relation they are already
		// in — the constructor cannot produce what the decoder will not read back.
		if !cp[i].Alive() {
			cp[i].clearTurn()
			cp[i].clearTarget()
			// A unit that is not alive is crossing nothing, so its transit pair
			// is residue of a state it has left. NORMALISED here and refused by
			// the decoder, which is the relation the order fields already stand
			// in: the constructor cannot produce what the decoder will not read.
			cp[i].clearTransit()
			// And it belongs to no group: a felled member is unlinked from its
			// group, so a term on one is residue in exactly the sense the two
			// above are, normalised here and refused by the decoder.
			cp[i].clearGroupSpeed()
			// And it holds no attack order, by the rule that gives it no move
			// order: a unit that is not alive takes no order at all, so a victim
			// on one is residue of a state it has left.
			cp[i].clearAttack()
			// Cast recovery is action residue too. A one-shot or moving
			// retained release may have moved recovery out of its book record,
			// but a body may carry neither representation.
			cp[i].CastWait = 0
			// And it is somewhere on the decay ladder, because a positive stage
			// and being not alive hold of the same entities. A body handed over
			// at DecayNone has not taken its death tick, so it is put where that
			// tick would have put it — the first stage, owing its own dwell.
			//
			// It is a PAIRING fix and not a death: no combat number is touched
			// here, so a world holding a body whose defence was already halved
			// when that world was cut cannot have it halved a second time by
			// being built from its own entities again.
			if cp[i].Decay == DecayNone {
				cp[i].Decay, cp[i].Dwell = DecayFallen, dwellOf(cp[i])
			}
		} else {
			// And a LIVING unit carries neither, which is the other half of the
			// same rule: a stage on one is residue in exactly the sense a target
			// on a body is, normalised here and refused by the decoder.
			cp[i].clearDecay()
		}
		// A dwell on any stage but the first is residue of a dwell that ran out,
		// on the rule the target coordinates and the stall count already take.
		if cp[i].Decay != DecayFallen {
			cp[i].Dwell = 0
		}
		// And the one shape with no value to fold onto is REFUSED, as the stall
		// count and the transit pair are.
		if err := decayFault(cp[i]); err != nil {
			return nil, fmt.Errorf("sim: entity %d: %w", cp[i].ID, err)
		}
		if !cp[i].HasTarget {
			cp[i].TargetX, cp[i].TargetY, cp[i].Stall = 0, 0, 0
		}
		// An entity naming ITSELF is normalised to holding no attack order, and
		// so is one whose cycle fields are residue of an order it does not hold.
		// Both are what the decoder refuses, and normalising them here is the
		// same trade the not-alive rule above makes: a caller handed a unit with
		// a stale victim has nothing it could do with an error.
		if !cp[i].HasAttackTarget || (cp[i].AttackTargetKind == AttackTargetUnit && cp[i].AttackTarget == cp[i].ID) {
			cp[i].clearAttack()
		}
		// The two remaining shapes are REFUSED rather than normalised, as the
		// stall count and the transit pair are: a caller that named them asked
		// for a cycle this package cannot produce, and there is no value to fold
		// them onto.
		if err := attackFault(cp[i]); err != nil {
			return nil, fmt.Errorf("sim: entity %d: %w", cp[i].ID, err)
		}
		if err := turnFault(cp[i]); err != nil {
			return nil, fmt.Errorf("sim: entity %d: %w", cp[i].ID, err)
		}
		// The remaining two shapes are REFUSED rather than normalised, as the
		// stall count above is: a caller that named them asked for a crossing
		// this package cannot produce, and there is no value to fold them onto.
		if err := transitFault(cp[i]); err != nil {
			return nil, fmt.Errorf("sim: entity %d: %w", cp[i].ID, err)
		}
		if err := strideFault(cp[i]); err != nil {
			return nil, fmt.Errorf("sim: entity %d: %w", cp[i].ID, err)
		}
		// The two experience shapes are REFUSED rather than normalised too, on
		// transitFault's own ground: a caller that named them asked for a state
		// this package cannot produce, and there is no value to fold them onto.
		if err := experienceFault(cp[i]); err != nil {
			return nil, fmt.Errorf("sim: entity %d: %w", cp[i].ID, err)
		}
		if err := cp[i].Book.Validate(cp[i].KnownSpells); err != nil {
			return nil, fmt.Errorf("sim: entity %d: %w", cp[i].ID, err)
		}
		if err := secondaryDamageFault(cp[i].SecondaryDamage); err != nil {
			return nil, fmt.Errorf("sim: entity %d: %w", cp[i].ID, err)
		}
	}
	// A SECOND PASS, because an attack order is the one field whose legality
	// depends on the rest of the slice rather than on its own entity: a victim
	// this world does not hold is an order pointing at nothing, and it cannot be
	// tested until every id is known. It is normalised to no order at all, which
	// is what the decoder refuses, so the two stay in the relation the shapes
	// above already stand in.
	for i := range cp {
		if cp[i].HasAttackTarget && cp[i].AttackTargetKind == AttackTargetUnit {
			vi := indexOfEntity(cp, cp[i].AttackTarget)
			if vi < 0 {
				cp[i].clearAttack()
			} else if !cp[vi].OrdinaryTargetable() {
				cp[i].clearAttack()
				cp[i].clearTarget()
			}
		}
		// A constructor caller may hand over residue beside an absent credit or
		// a source the new world does not hold. Normalise both to the null
		// pointer shape the decoder refuses; the signed spell byte is retained,
		// because the original clear arms erase only the source pointer.
		if !cp[i].HasKillCredit || indexOfEntity(cp, cp[i].KillCreditSource) < 0 {
			cp[i].clearKillCredit()
		}
	}

	// THE STRUCTURE LIST (1033 B3), sorted and checked for a duplicate id on
	// entities' own terms above: a structure is not an entity and needs
	// neither the health, order nor group normalisation the loop above
	// applies, because this story's boundary is identity and one field alone.
	cs := make([]Structure, len(structs))
	copy(cs, structs)
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	for i := range cs {
		if i > 0 && cs[i].ID == cs[i-1].ID {
			return nil, fmt.Errorf("sim: duplicate structure id %d", cs[i].ID)
		}
		if (cs[i].Width == 0) != (cs[i].Height == 0) {
			return nil, fmt.Errorf("sim: structure %d has incomplete footprint %dx%d",
				cs[i].ID, cs[i].Width, cs[i].Height)
		}
	}

	// THE CONTAINER, one slice per entity in entity order: built here, after cp
	// is sorted and every id is settled, because a Stock names an id rather
	// than a position and normaliseHoldings's lookup is a search over the
	// final, sorted list. EQUIPMENT COMES OUT OF THE SAME CALL, off the same
	// records: a Stock states what an actor carries AND what he is wearing, so
	// one walk answers both and no world can hold a container from a record
	// whose loadout it dropped. A Stock list naming no loadout — every world
	// built before the starting-equipment hotfix — leaves every entity
	// unarmed at every slot, which is already [EquipSlots]uint16's own zero
	// value.
	carried, equipment, err := normaliseHoldings(cp, stock)
	if err != nil {
		return nil, err
	}
	if err := normaliseWeaponSources(cp, equipment); err != nil {
		return nil, fmt.Errorf("sim: %w", err)
	}

	// One slot per entity, every one of them empty. The slice is materialised
	// rather than left nil for the reason the grid is: "a world holds no route"
	// and "routes[i] belongs to entities[i]" are then one representation instead
	// of two kept in agreement, and no reader has a length to check first.
	if s.Empty() && s.Dialect() == ScriptROM1 {
		s = nil
	}
	w := &World{
		rng: rng{state: seed}, bounds: b, mode: mode, grid: g, cost: cost, height: height,
		relations: rel.materialised(),
		entities:  cp, routes: make([][]cell, len(cp)), groups: freezeGroups(cp), sacks: loot,
		spells:  spellTable,
		ghost:   ghost,
		carried: carried, equipment: equipment, script: s,
		structures: cs,
	}
	w.rebuildStructureSlots()
	if err := w.pendingAttackFault(); err != nil {
		return nil, err
	}
	for i := range w.entities {
		if e := w.entities[i]; e.HasAttackTarget && e.AttackTargetKind == AttackTargetStructure {
			if _, ok := w.attackStructure(e); !ok {
				w.entities[i].clearAttack()
				w.clearOrder(i)
			}
		}
	}
	// THE FORMATION DEFAULT IS WRITTEN HERE AND NOT AT THE READ (0166 D-2).
	// A Go array's zero value is all zeroes and mode 0 means NEVER IN
	// FORMATION, so a build that read the field without filling it would
	// silently take every group in every world out of formation.
	w.resetFormations()
	w.initializeROM2Script()
	w.presetRegisters()
	w.reserveObservableEntityIDs()
	// THE LOAD IS WRITTEN AT GUARD, unconditionally, and a caller's own value
	// is discarded rather than read (on the actor state's own rule stated in
	// this constructor's doc). It is derived from the containers and worn sets
	// this constructor just settled, against a weight table no constructor
	// takes: a world nobody declares one to therefore opens with every load at
	// the empty sum, which is what every world built before this story held.
	// DeclareItemWeights writes them again.
	w.recomputeLoads()
	for _, s := range stock {
		if s.LoadState != nil {
			if err := w.RestoreActorLoad(s.ID, *s.LoadState); err != nil {
				return nil, err
			}
		}
	}
	return w, nil
}

// costAt and heightAt are cell c's bytes on the two planes, and ZERO for a
// cell the planes do not describe.
//
// The out-of-bounds answer is a real one and not a guard. A route search may
// begin from a cell OFF THE MAP — that is deliberate, so that a unit standing
// outside the grid can still be routed onto it — so the rate's call site can be
// handed such a cell. Zero is what the plane says where it holds no cell, and
// the law's own arithmetic then does the rest: a zero height contributes no
// tilt, and a zero cost enters the mean, which may then take the substitute.
//
// The SEARCH never reaches that arm, because a step is only ever into an open
// cell and every open cell is in bounds. The two share one accessor anyway:
// two would be two places for "the plane holds no such cell" to be answered
// differently.
func (w *World) costAt(c cell) uint8 {
	if key, ok := savedPlaneKey(c.x, c.y); ok && w.savedCellPlanes != nil && w.savedCellPlanes.CostKnown[key] != 0 {
		return w.savedCellPlanes.Cost[key]
	}
	if i, ok := w.cellIndex(c.x, c.y); ok {
		return w.searchCostAt(i, c)
	}
	return 0
}

// describes reports whether the planes hold a cell at c at all — the bounds
// test, named where the rate's own rule is written rather than spelled out
// twice at the one site that asks it.
func (w *World) describes(c cell) bool {
	_, ok := w.cellIndex(c.x, c.y)
	return ok
}

func (w *World) heightAt(c cell) uint8 {
	if key, ok := savedPlaneKey(c.x, c.y); ok && w.savedCellPlanes != nil {
		return w.savedCellPlanes.Height[key]
	}
	if i, ok := w.cellIndex(c.x, c.y); ok {
		return w.height[i]
	}
	return 0
}

// Tick returns the number of ticks the world has advanced. It is an index, not a
// duration.
func (w *World) Tick() uint64 { return w.tick }

// Bounds returns the map's cell extent.
func (w *World) Bounds() Bounds { return w.bounds }

// CastingSpell reports one entity's canonical spellbook wind-up. It is a
// read-only projection of the state repeated input, persistence and the digest
// already consume; callers cannot retain or mutate the underlying record.
func (w *World) CastingSpell(id EntityID) (spell uint16, remaining uint8, ok bool) {
	if i, exists := w.scrollIndex(id); exists && w.scrollCasts[i].Started {
		spell, _, _ = ScrollSpell(w.scrollCasts[i].Item)
		return spell, w.scrollCasts[i].Remaining, true
	}
	i, ok := w.bookCastIndex(id)
	if !ok || w.bookCasts[i].Phase != bookCharging {
		return 0, 0, false
	}
	return w.bookCasts[i].Spell, w.bookCasts[i].Remaining, true
}

// Spell returns the normalized rule installed in this world. The value is a
// copy, making it safe for developer instruments and presentation readouts.
func (w *World) Spell(id uint32) (SpellRule, bool) { return w.findSpell(id) }

// Relations returns the world's relation as a fresh copy, on Entities()' own
// rule and for its reason: mutating the result cannot reach the world it came
// from, so flipOnBlow — the type's one writer — stays the only one a caller
// could ever reach.
func (w *World) Relations() Relations { return w.relations.materialised() }

// hostileTo reports whether me's roster slot treats him's as an enemy. It is the
// one bridge from an entity to the relation, so the two places that could
// differ about which of the pair indexes the row cannot.
func (w *World) hostileTo(me, him Entity) bool {
	return w.relations.Hostile(me.Owner, him.Owner)
}

// flipOnBlow remembers the attacker's cell and attempts both diplomacy flips
// (AI-RETAL-056, HERO-AGGRO-028). Each direction retains its allied-bit gate.
// It does not assign an order or target: normal acquisition consumes the notice.
func (w *World) flipOnBlow(ai, ti int) {
	if ai < 0 || ti < 0 || ai >= len(w.entities) || ti >= len(w.entities) {
		return
	}
	a, t := w.entities[ai], w.entities[ti]
	if a.Owner == 0 || t.Owner == 0 {
		return
	}
	w.rememberAttacker(ai, ti)
	w.relations.turnHostile(a.Owner, t.Owner)
	w.relations.turnHostile(t.Owner, a.Owner)
}

// Entities returns the world's entities in ascending id order, as a fresh slice
// of a pointer-free value type: mutating the result, or anything in it, cannot
// reach the world it came from.
func (w *World) Entities() []Entity {
	out := make([]Entity, len(w.entities))
	copy(out, w.entities)
	for i := range out {
		out[i].NativeBasis = w.nativeBasisNow(out[i])
	}
	return out
}

// EntityView returns the world's own entity slice in ascending id order,
// without the copy Entities makes. It is for a reader that walks the list
// once and changes nothing: the caller must not write through it, must not
// retain it, and must not use it after anything mutates the world. A range
// loop's element is still a copy, so modifying that local is safe.
func (w *World) EntityView() []Entity {
	return w.entities
}

// Entity returns one entity by id, the same shallow copy Entities hands out,
// without copying the whole list.
func (w *World) Entity(id EntityID) (Entity, bool) {
	if i := indexOfEntity(w.entities, id); i >= 0 {
		e := w.entities[i]
		e.NativeBasis = w.nativeBasisNow(e)
		return e, true
	}
	return Entity{}, false
}

// Structures returns the world's structures in ascending id order, as a fresh
// slice a caller may mutate without reaching the world it came from — the
// entity list's own rule (1033 B3).
func (w *World) Structures() []Structure {
	out := make([]Structure, len(w.structures))
	copy(out, w.structures)
	return out
}
