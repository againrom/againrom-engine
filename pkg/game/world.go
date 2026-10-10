package game

import (
	"fmt"
	"image"
	"slices"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// mapWorld is the running world under one open map screen: the world itself,
// the script it is advanced against, the unit-art bundle its entities resolve
// against, and the viewer that draws them.
//
// It lives HERE, in the one tier that may see both the simulation and the
// window, and it is unexported because nothing outside this package may hold
// one. The window tier gets a parameterless func() and cannot name a single
// type in this struct; the simulation tier gains nothing at all. What
// crosses to the viewer is a slice of plain cells with render-tier art
// beside them, which is why the draw path cannot reach a world even by
// accident.
//
// The bundle rides in this struct because resolution — pairing an entity's
// class id with a drawn frame — belongs to the ONE tier that sees both a
// world and a bundle: the window tier must not learn what a class id is, and
// the simulation must not learn what art is. It is read on every push and
// written never; nil is the hand-assembled front-end with no bundle, and
// every entity then crosses art-less.
//
// The fields are held for the life of one opened map and are replaced by
// nothing: the world is ADVANCED, never rebuilt, and the schedule is derived
// once, when the map opens. A world rebuilt per tick would stand at tick 0
// forever and a schedule re-derived per tick would be the same slice
// computed again — the first is a defect, the second is waste, and holding
// both closes the question.
//
// It is dropped with the viewer when the map screen is left: the front-end
// holds it only through the tick closure the loader handed back, so leaving
// releases the world in the same statement it releases the viewer.
//
// clock and last are the CADENCE, and they live here for the same reason
// scene does: born with the map, dropped with it, and read by nothing
// canonical. clock is the decoded logic-tick model — terrain's own Ticker
// at the speed index map load selects, the one the water layer already paces
// by — and last is the wall-clock instant the previous paced call was made
// at. They decide only WHEN a tick fires. What a tick does is tick()'s,
// which reads neither.
//
// pending is the queue of orders the front-end has issued and no advance has
// applied yet, and it lives here for the third time for the same reason:
// born with the map, dropped with it, read by nothing canonical. It is NOT
// world state — it is not in pkg/sim, it enters neither Hash nor
// MarshalBinary, and an order sitting in it has changed nothing at all until
// a tick assembles it. Orders keep ISSUE ORDER here, so the last one issued
// for an entity is the last one Step sees and therefore the one that wins;
// and it is emptied by the tick that applies it, which is what makes
// "exactly one advance and no later one" a property of one statement's
// position rather than a rule.
//
// commanded is every entity the front-end has ever ordered, and it lives
// here for the fourth time for the same reason: born with the map, dropped
// with it, read by nothing canonical. It is WRITTEN BY THE STATEMENT THAT
// ENQUEUES and read only by lookup — facing's precedent — so a Go map's
// per-process range order has no path into a command stream and therefore
// none into a world. NOTHING CLEARS IT while the map is open, and that
// absence is C-4's "for good": a commanded unit leaves the placeholder
// script for the life of the screen rather than for the one tick a
// last-write would have bought, which matters because a fresh scripted
// target arrives every twenty-four ticks and would otherwise overwrite the
// player's order inside two seconds.
//
// prev is the STEP MEMORY: the cell each entity stood on before the most
// recent advance, keyed by entity id. It lives here for the same reason
// every field above it does — born with the map, dropped with it, read by
// nothing canonical — and the difference between it and the cell an entity
// stands on now is the step that entity took, which is both what says
// whether it is walking and which way it faces and the segment the drawing
// interpolates along.
//
// THE ADVANCE WRITES IT AND THE PUSH ONLY READS IT, which is what keeps building
// a snapshot a pure, repeatable read: asked twice with no tick between, the push
// answers with the same steps, directions and frames. Written from the push it
// would consume its own input, and the second call would report a mover as
// stationary — silently, with no digest to catch it.
//
// PRESENCE IS PART OF THE ANSWER. An id this map has not seen has taken NO step,
// which is what makes the constructor's tick-0 push carry no step rather than a
// delta from the map's corner, and an entity first appearing in a snapshot start
// at rest. It is looked up by key and NEVER ranged, so a Go map's per-process
// range order has no path into a snapshot.
//
// died is the DEATH CLOCK: the scene tick each entity was first observed not
// alive on, keyed by entity id. It lives here for the same reason every
// field above it does — born with the map, dropped with it, read by
// nothing canonical — and the difference between it and the scene clock is
// how far into its fall a body is.
//
// IT IS SET ONCE PER FALL and never rewritten while that body remains fallen,
// so the snapshot build may write it: a build that finds an entry leaves it,
// and repeated builds without a tick answer the same elapsed count. That is what
// separates it from prev, which is a DELTA memory — a build that wrote that
// one would consume its own input and report every mover as standing still —
// and it is why prev is written in the tick and this is not. The tick death is
// first observed on is only knowable AFTER the step, so prev's placement would
// be wrong here by exactly one tick.
//
// A living actor clears its old stamp before drawing. Healing and a later
// fall of the same identity then start a new death run.

// hurt is the short presentation clock for a finishable fallen body struck by
// a causal damage event (DIV-488). Unlike died it is deliberately REWRITTEN by
// every new hit, because each hit restarts the twitch. Unlike HP comparison in
// the viewer it is fed by sim.Report.Damages, so the corpse decay ladder cannot
// manufacture a hit every time it lowers health. It reaches no world byte or
// digest and expired entries are pruned on the tick path, never by a draw.
//
// stopped is the world's own STOP, and unpaced is the decoded owner-loop
// selector (`SESS-CLOCK-005`, `SESS-IDLE-007`). They live here for the same
// reason every field above does: born with the map, dropped with it, read by
// nothing canonical. It is a switch BESIDE the rate and never a value of it
// — the shipped index clamp reads a value below the table as the slowest
// speed, so a stop written as a rate would be a slow game — and it is a
// field of this struct rather than of the ticker, because the water instance
// would then carry a switch it must never read and cmd/mapview would gain
// one it cannot honour.
type mapWorld struct {
	cheats           cheatConsole
	fame             fameObserver
	applicationState *SnapshotApplicationState
	world            *sim.World
	sched            [][]sim.Command
	units            *terrain.UnitSet
	view             *ui.Viewer
	scene            int
	prev             map[sim.EntityID]image.Point
	died             map[sim.EntityID]int
	hurt             map[sim.EntityID]int
	pendingDamage    []sim.DamageEvent
	soundEntities    []ui.MapEntity
	clock            *terrain.Ticker
	last             time.Time
	stopped          bool
	unpaced          bool
	pending          []sim.Command
	pendingIgnored   []bool
	commanded        map[sim.EntityID]bool

	// Diagnostic damage-message counts; voice dispatch consumes the ordered
	// pending stream. Neither counter enters simulation or saves.
	blows map[sim.EntityID]uint32

	// strikes counts the unit strikes that took no health from each entity
	// (ANIM-125), beside blows and on the same terms.
	strikes map[sim.EntityID]uint32

	// drawnMoving is whether each entity's last pushed frame drew the move
	// action, the draw state a move or swarm reply tests on its speaker
	// (VIDEO-067). It is presentation only and reaches no world field, save or
	// digest.
	drawnMoving map[sim.EntityID]bool

	// ownerFrames is the presentation-only frame identity for one base frame
	// through one owner palette. The indexed pixels stay shared; a distinct
	// pointer keeps the window's frame-keyed texture cache from handing two
	// factions one uploaded texture. It reaches no simulation field or digest.
	ownerFrames map[ownerFrameKey]*terrain.StaticFrame

	// structureStateScratch is the presentation projection rebuilt by push.
	// Reusing it keeps a long headless mission from allocating one slice per
	// tick merely to carry structure health across the viewer seam.
	structureStateScratch []ui.MapStructure

	// fog is the local participant's whole fog-of-war state: which cells
	// sim.SelfSlot's own living entities can see right now and which cells have
	// ever been seen, sized from the world's own bounds. It is built and
	// refreshed once in newMapWorldWith, BEFORE that constructor's tick-0 push,
	// so a mission opens closed by construction rather than by an ordering this
	// struct's caller has to get right; refreshed again every fogPeriod world
	// ticks in tick, below. push reads its three-state projection for
	// rendering, and attackOrCast reads explored for the local participant's
	// Teleport admission (pkg/game/fog.go).
	fog *fogPlane

	// groupTag is the tag the NEXT group order will carry, bumped whenever an
	// enqueued order cannot join the one being assembled — see enqueue. It is
	// bookkeeping of the front-end's, exactly as the four memories above it are:
	// the simulation reads a tag inside one advance and stores it nowhere, so
	// this counter reaches no world field, no byte form and no digest, and a
	// world resumed from its bytes needs nothing of it.
	groupTag uint32

	// tiers is which TIER each entity's art is drawn in, keyed by entity id. It
	// lives here for the sixth time for the reason every memory above it does
	// — born with the map, dropped with it, read by nothing canonical — and
	// for one more: a tier is a fact about a placement's definition, not about
	// a world, so putting it on an entity would widen the byte form and the
	// digest for a colour.
	//
	// It is FIXED, not a memory. The four maps above record what the world did
	// and are written as it runs; this is resolved once, when the map opens,
	// and never written again, so the push may read it as freely as it reads
	// the bundle.
	//
	// AN ID WITH NO ENTRY STATES NO TIER, which reads as the zero tier and
	// draws the sheet's own colours — the answer for every placement that took
	// a non-stat arm, for one whose key reached no entry, and for every entity
	// of a map opened with no definition table. Looked up by key and NEVER
	// ranged, facing's precedent.
	tiers      map[sim.EntityID]int
	actorNames map[sim.EntityID]string

	// walk is the WALK ODOMETER: how far each entity has walked since it last
	// stood still, and how many ticks of its current crossing have run. It
	// lives here for the seventh time for the reason every memory above it does
	// — born with the map, dropped with it, read by nothing canonical — and
	// for the reason the walk needs one at all: the engine's own count is on
	// the CLIENT DRAWABLE, outside the state a save carries, and putting ours
	// in pkg/sim would make a quantity nothing in the simulation reads into
	// hashed state that every digest has to agree on.
	//
	// THE ADVANCE WRITES IT AND THE PUSH ONLY READS IT, prev's own rule and for
	// prev's own reason: written from the push it would consume its own input,
	// and the second build of one tick's picture would differ from the first
	// with no digest to catch it.
	//
	// Looked up by key and NEVER ranged, facing's precedent. An id the map has
	// not seen reads the zero value — no distance walked and no crossing under
	// way — which is what an entity appearing in a snapshot for the first time
	// has done.
	walk map[sim.EntityID]walkClock

	// chars is the CHARACTER each entity was placed with, keyed by entity id.
	// It lives here for the reason tiers does, and it is the same shape: a fact
	// about a PLACEMENT rather than about a world, resolved once when the map
	// opened and never written again, so the push may read it as freely as it
	// reads the bundle.
	//
	// A hero's statistics are a loader input. They reach no entity field, no
	// byte form and no digest — which is exactly why they cannot be read back
	// off the world and have to be remembered here instead.
	//
	// AN ID WITH NO ENTRY STATES NO CHARACTER, which is every unit a map placed
	// and every entity of a world opened with no party. Looked up by key and
	// NEVER ranged, tiers' own precedent.
	chars map[sim.EntityID]ui.UnitCharacter

	// derives is the loader-side half of every live character sheet. Skill
	// awards live in pkg/sim, while the raw statistics, profile and equipment
	// definitions deliberately do not cross that determinism wall. Keeping the
	// two halves paired here lets one fixed post-step pass replace the complete
	// derived block in the same tick a level rises.
	derives       []characterDerive
	derivedSkills map[sim.EntityID]derivedSkillState
	// skillBonus is the skill bonus of the worn items that the entity's levels
	// include, per member. A native entity holds base plus bonus, so the base
	// a recompute starts from is the levels less this.
	skillBonus     map[sim.EntityID][data.SkillSlots]int32
	derivedPotions map[sim.EntityID][4]int32

	// art is the class an entity is DRAWN AS when its own class key is not the
	// answer, keyed by entity id.
	//
	// It lives here for the reason tiers and chars do, and it is the same shape:
	// a fact about a PLACEMENT rather than about a world, resolved once when the
	// map opened and never written again, so the push may read it as freely as
	// it reads the bundle.
	//
	// WHY IT EXISTS AT ALL is that the two populations are drawn by two
	// different rules. An actor a map places is drawn as the class its record
	// names; a player's character is drawn as whatever its visible equipment
	// says, from a sheet composed out of a name — and one code path cannot serve
	// both. This lookup is that second rule's whole footprint in the driver.
	//
	// AN ID WITH NO ENTRY IS DRAWN THROUGH ITS OWN CLASS KEY, which is every
	// unit a map placed, every entity of a map opened with no party, and every
	// party member whose body the loader could not resolve. Looked up by key and
	// NEVER ranged, tiers' own precedent.
	art map[sim.EntityID]*terrain.UnitClass

	// swing is the SWING CLOCK: how many ticks each entity's current attack RUN
	// has been playing, keyed by entity id, and phase is what the run is
	// started from — the attack phase this seam last saw the entity in.
	//
	// A RUN IS ONE PASS OF THE ART, and both the length and the restart are
	// decoded (ANIM-RUN-004, ANIM-PHASE-003, ANIM-STATE-023). The engine's attack
	// arm counts one per tick from zero, indexes the expanded track with no
	// modulo, runs for exactly the track's own length and is then forced back to
	// the standing state; a fresh run begins when the swing starts, which is the
	// tick the attacker's countdown is loaded. So this clock counts from zero at
	// the transition INTO the charging phase and rises by one per tick after it,
	// and the selection refuses once it reaches the track's length — which is
	// what returns the unit to its idle drawing until the next swing.
	//
	// It is NOT the attack cycle's countdown and must not become one: the swing
	// frame, the swing sound and the damage are scheduled from three different
	// numbers in the original, and binding any two together reproduces it on at
	// most 2 of the 157 shipped pairs that answer (ANIM-CLOCK-024).
	//
	// Neither map is canonical: neither is in the world, the byte form or the
	// digest, exactly like the death clock and the odometer beside them. Looked
	// up by key and NEVER ranged, the step memory's precedent.
	swing map[sim.EntityID]int
	phase map[sim.EntityID]sim.AttackPhase

	// bolts is this tier's own memory of the spells in flight (spellbolt.go).
	// Not canonical either: not in the world, the byte form or the digest,
	// beside the two maps above.
	bolts []spellBolt

	// shots is the memory of ranged swings and the shots they released
	// (unitshot.go), presentation only and not saved.
	shots unitShots

	// healBursts are presentation-only particles born from a positive semantic
	// Heal result. They retain the application anchor and target owner rather
	// than an entity pointer, so target motion/removal cannot leave a stale
	// reference and no cosmetic state reaches the simulation hash or save.
	healBursts []healBurst

	// projectiles is the projectile art bundle, keyed by picture id. It is
	// FIXED, not a memory: loaded once with the archives and assigned when the
	// mission opens, so the push may read it as freely as it reads the unit
	// bundle.
	//
	// NIL DRAWS NO SPELL ART, which is every hand-assembled front end in this
	// tree and every driver built before this story. Its own lookup answers nil
	// on a nil receiver, so no caller here tests for it.
	projectiles *terrain.EffectSet

	// castRun is each book caster's own attack run: the ticks left and the span
	// it stands for (spellbolt.go).
	castRun         map[sim.EntityID]castRun
	visualIDs       map[sim.EntityID]sim.EntityID
	visualNext      sim.EntityID
	visualLocalNext sim.EntityID

	// markElements is the client's own effect-mark element list (1002;
	// MAGIC-MARK-060, effectmark.go): one entry per (actor, kind) the
	// simulation currently carries, holding the phase countdown the rebuild
	// decrements. Cosmetic, on bolts' and castRun's own terms — not in the
	// world, the byte form or the digest.
	markElements []markElement

	// stoneHold is the scene tick each stone-cursed actor's animation was held
	// at (1002; MAGIC-ACTOR-066). Cosmetic beside markElements.
	stoneHold map[sim.EntityID]int

	// sounds, swingSound and spellSound are the map's class-timed and
	// positional sound seam, installed together by setSwingSound: sounds is
	// LoadUnitSounds' own per-class table (sound.go), holding the sound-slot
	// array and the attack delay swingSlot's own emission below reads, and
	// swingSound is the play callback bound to a running viewer's PlaySlotAt.
	//
	// BOTH ARE NIL UNTIL setSwingSound RUNS, and neither newMapWorld nor
	// newMapWorldWith ever calls it: openMapWorld and openMission — the only
	// two production callers of either constructor — leave both fields at
	// their zero value on return, and frontend.go installs them afterwards,
	// at the same two sites that hand a viewer its font (loadMap,
	// MissionOpener). That is what keeps close to forty existing tests in
	// this package, every one of which builds a mapWorld through
	// newMapWorld or newMapWorldWith directly, untouched by this story: a
	// nil callbacks are the sound paths' own guard for "say nothing", not a
	// state those tests have to arrange.
	sounds        map[int32]UnitSound
	swingSound    func(slot int, owner uint32, cell image.Point)
	spellSound    func(slot int, owner uint32, cell image.Point)
	semanticSound func(source string, slot int, owner uint32, cell, fine image.Point)

	// spellSoundCues are delayed one-shot selectors owned by simulation-tick
	// events: cast-animation AttackDelay, travelling-impact arrival and Meteor's
	// phase-8 arm. They advance only in tick, never in a draw or frame push.
	spellSoundCues []spellSoundCue

	// mission is everything this driver needs to answer for a MISSION rather
	// than a bare map: which mission it is, where its words are read from, what
	// its script raised, and what is on screen because of it.
	//
	// IT IS NIL FOR EVERY MAP THE PICKER OPENS, which is what keeps that whole
	// path byte-identical: no announcer is built, no outcome is watched and no
	// address is ever composed, so a map opened from the list is the map screen
	// this story inherits.
	//
	// It is ONE POINTER rather than six fields, because the six are meaningless
	// apart: a driver holding a part number and no payload, or an announcer and
	// no mission number, is a state nothing could act on. Together they are
	// present or absent.
	mission *missionNotices

	// invSubject is the inventory window's own subject, exactly as openMission
	// built it (docs/0110-inventory T4; plan D-7, D-8) — held HERE, not
	// re-read off the viewer, so paced's own per-frame refresh can hand the
	// viewer back the SAME Figure and Slots and change only Pack. It lives here
	// for the reason every memory above it does — born with the map, dropped
	// with it, read by nothing canonical — and it is a Viewer setter's whole
	// argument rather than a Viewer field this package reaches into, because
	// pkg/ui's own fields stay unexported.
	//
	// invSubjectSet is whether there IS one, and it is a SEPARATE flag rather
	// than a test of invSubject.ID (0112 spec: guard the empty-party case). A
	// mission with no party or no start ids leaves buildInventorySubject's
	// own ZERO subject, whose ID is 0 — a real entity id an unrelated actor
	// may hold — so an id comparison could not tell "no subject" from "the
	// subject is entity 0", and this flag is what does instead.
	invSubject    ui.InventorySubject
	invSubjectSet bool

	// DIV-330
	pickup pickupIntent

	invCodes []sim.ItemStack

	// invIconCache is buildInventoryPack's own cache, keyed by item code: built
	// once, at mission open, and never reset while the map is open, so a
	// recompose after a grab rereads only the codes it has not resolved before.
	invIconCache map[uint16]*image.RGBA

	// portraits is pushPortrait's own cache (portrait.go), keyed by the
	// picture NAME rather than by a class id — two classes naming one picture
	// is the shipped case, and the name is what the address is built from. A
	// nil value is a name that resolved to nothing and is cached alike, so an
	// install missing a node costs one read for the session.
	portraits map[string]*image.RGBA

	// figures is which figure each PLACED HUMAN composes its doll from
	// (figures.go), keyed by entity id and absent for every creature; and
	// figurePics is the composed picture, keyed by that figure TOGETHER with
	// the equipment it was composed over, because a doll is a function of both.
	figures    map[sim.EntityID]figureID
	figurePics map[figureCacheKey]*image.RGBA

	// figureMasks is unitFigure's own mask, cached under the SAME key its
	// picture is (1005 contract, item 1) — see figures.go's own comment on
	// unitFigure for why this stays a sibling map rather than folding the two
	// into one cached struct.
	figureMasks map[figureCacheKey]*ui.SlotMask

	// npcFaces is what picture each npc RECORD calls for, keyed by the
	// subscript a dialogue's `npc=` tag names (speakers.go). It is built once
	// per process and handed in, because a speaker's record is a fact about the
	// campaign and not about this map — which the measurement behind it is
	// emphatic on: half the shipped speakers are not placed on the map that
	// quotes them.
	npcFaces map[int32]data.NPCFace

	// speakerActors is who a dialogue can be ABOUT: this map's placed persons
	// followed by the mission party, with the values the speaker predicate
	// reads (speakeractors.go). It lives here for figures' own reason — born
	// with the map, dropped with it, reaching no world field, no byte form and
	// no digest — and, like tiers, it is FIXED: resolved once when the map
	// opens and never written again.
	speakerActors []speakerActor

	// spellAtlasImg/spellAtlasTried/spellIcons are spellicon.go's own: the one
	// decoded icon strip, whether it has been attempted, and one 36x36 cut per
	// spell id. All three hold their MISSES — a spell with no slot and an
	// install with no atlas both answer nil once rather than on every frame.
	spellAtlasImg   *bmp.Image
	spellAtlasTried bool
	spellIcons      map[uint16]*image.RGBA

	invEquipment data.Equipment
	// invEquipmentItems is the complete-instance half of invEquipment's
	// change tracker. Equal codes can still carry different magic or value.
	invEquipmentItems [sim.EquipSlots]sim.ItemInstance

	// invWeaponEverEquipped records whether the subject's starting-weapon
	// fallback has been spent since he was last seeded (openMission,
	// switchInventorySubject): slot 1 was occupied, or its starting code was
	// already in the pack. rearm (below) uses it to decide
	// whether an empty slot 1 means "the starting weapon was never written
	// into the array" (hotfix b51b439+1) or "the weapon was taken off".
	//
	// The bug this exists to fix: Rearm's fallback argument,
	// mw.invParty.startWeapon, is meant only for a character whose starting
	// weapon has never reached the array (0124's disclosed limit). But
	// ResolveEquipmentLoadout (pkg/mapload/loadout.go) applies that fallback
	// on every call where slot 1 reads empty, with no memory of whether it
	// was ever real. Once 0134-what-he-wears started writing a generated
	// character's starting weapon into the array at spawn, slot 1 is
	// occupied from tick 1 for the ordinary case, and the fallback's only
	// remaining effect was to reapply the starting weapon's damage after an
	// explicit unequip left slot 1 empty again — the hero never actually
	// fought bare-handed. This field remembers that slot 1 has been real at
	// least once, so rearm can stop passing the fallback from then on.
	invWeaponEverEquipped bool

	// invFigureEquipment is refreshEquipment's own tracker, the same shape as
	// invEquipment and for the identical reason — it exists as a separate
	// field rather than one shared with rearm's — but comparing what the
	// FIGURE was last composed from rather than what the COMBAT BLOCK was last
	// recomputed from. The zero value at mission open is on invEquipment's own
	// reasoning above, WITH ITS CORRECTION: openMission's call to
	// buildInventorySubject paints the figure from the member's whole worn set
	// (0136 FR-10d, inventory.go), and the world's array now carries that same
	// set, so the first paced frame recomposes once from a picture that already
	// matches. It is a recompose that changes nothing rather than one that is
	// skipped, and the difference costs one frame's work at the open and buys
	// the tracker one rule instead of two.
	invFigureEquipment data.Equipment
	// invFigureEquipmentItems is refreshEquipment's complete-instance tracker.
	// It keeps worn popups and enchantment marks live across equal-code swaps.
	invFigureEquipmentItems [sim.EquipSlots]sim.ItemInstance

	// invLayers and invFigureLayers are the clothing layers rearm and
	// refreshEquipment last derived from and composed. Both stay nil without a
	// mod that adds layers.
	invLayers, invFigureLayers []uint16

	// invComposedEquipment is what the doll ACTUALLY DREW FROM last, distinct
	// from invFigureEquipment above (C1b, round-2 adversarial review, twelfth
	// pass). invFigureEquipment is seeded from currentFigureEquipment — a
	// LIVE re-derivation off the running sim.World, correct by construction and
	// blind to whatever buildInventorySubject itself composed the figure from
	// at mission open or member switch. This field is set from that same
	// composition's own answer (inventory.go's missionDollEquipment, called a
	// second time at the same two call sites rather than threaded out of
	// buildInventorySubject's fenced signature) and is kept in step by
	// refreshEquipment's own recompose branch below, which updates both
	// trackers together once a live change actually earns one. A caller asking
	// "what does the doll show right now" reads this field; a caller asking
	// "has the live entity's equipment changed since the doll was last
	// composed" — refreshEquipment's own question — still reads
	// invFigureEquipment, unchanged.
	invComposedEquipment data.Equipment

	// invDollSuppressSlot is refreshDollDrag's own tracker, invFigureEquipment's
	// shape restated for one integer instead of an equipment array: which
	// slot (1..12) the viewer currently reports mid-drag off the doll, or 0
	// for none, LAST TOLD to the viewer (1005, "the interactive doll", item
	// 5). The guard-then-skip this buys is the same one refreshEquipment
	// already relies on: a drag that has not moved to a new slot since last
	// frame costs one compare and no recompose.
	invDollSuppressSlot int

	invSkill    [data.SkillSlots]int32
	invTraining sim.NativeTraining
	invSkillSet bool

	skillPosted map[sim.EntityID][data.SkillSlots]int32

	bodyEquipment data.Equipment

	// invParty is everything T4's recompute and resolvability test need about
	// the subject and the installed definitions, beyond the equipment array
	// itself — resolved once, at mission open, and never written again,
	// chars' and art's own shape (plan D-5). See invPartyGear's own doc for
	// each field and why it lives here rather than being re-read from ms or
	// re-parsed from an archive on every tick.
	invParty invPartyGear

	// spellNames is the book's own NAME resolver: id to Name, resolved once off
	// the SAME Spells collection the world's own []sim.SpellRule table was
	// built from (mapload's spellsFor), and never written again. It lives here
	// for tiers' and chars' own reason: a fact about the INSTALL rather than
	// about the world.
	//
	// PKG/SIM CARRIES NO NAME (SpellRule's own doc, pkg/sim/spell.go): the
	// determinism wall admits no string a save need not hash, so a name
	// never reaches the world and this map is the only place in pkg/game
	// one is read at all. AN ID WITH NO ENTRY STATES NO NAME, which is
	// every map opened with no Spells collection and answers exactly like
	// one holding an empty string for it — spellbookOf (spell.go) never
	// treats an absent name as an absent spell.
	spellNames map[uint16]string
}

// invPartyGear is invParty's own type (mapWorld, above): what rearm's
// recompute and equipFromPack's resolvability test both need beyond the
// world's own bytes (docs/0124-equip-from-the-pack T4; plan D-5, R-1).
//
// hero IS THE SUBJECT'S OWN Hero, read back off ms.Party[0] rather than
// recomputed from the derived UnitCharacter chars already holds (world.go's
// own field): Hero.Recompute needs the four raw statistics and the six skill
// levels to run the derived-stat graph again for a NEW loadout, and
// UnitCharacter carries only the OLD graph's output.
//
// figureDir and figureFace ARE memberFigure's OWN PAIR (inventory.go),
// recomputed here from the same party member rather than threaded out of
// buildInventorySubject: that function's signature is a T4 fence this task
// does not widen (buildInventorySubject still takes only an entrySource and
// a *Mission), so refreshEquipment gets the pair the same way openMission's
// own call to buildInventorySubject already did, one statement earlier.
//
// table IS THE THREE COLLECTIONS data.WeaponFromCode RESOLVES A SLOT'S CODE
// AGAINST — Shapes, Materials and Weapons — carried on the SAME *mapload.Table
// openMission already receives as its own t parameter (world.go) and had
// nowhere to keep before this story: nothing before T4 needed a definition
// table once the tick-0 push was over. IT MAY BE NIL, exactly as openMission
// already treats a nil t as "resolve nothing" for entityTiers, and every
// caller here refuses before dereferencing it rather than risk a nil-pointer
// read on a table's own Shapes or Materials field.
//
// list IS READ HERE, off the mission's own archive source, RATHER THAN
// THREADED FROM THE FRONT END: openMission holds no field for a FrontEnd's
// Bodies (that value lives one tier up, in Definitions and FrontEnd, and
// this task's own boundary is world.go alone), so ReadBodyList is called a
// SECOND time, over the SAME address LoadDefinitions already read once at
// construction. That is not a second answer — it is the same bytes read
// again, for the one caller sitting where the front end's own copy cannot
// reach — and it costs one small text parse per mission open, never per
// frame. A nil or empty list is not an error, ReadBodyList's and
// HeroAppearance's own shared contract: a mission whose archive holds no
// heropicture.txt refreshes into no name, which draws the subject through
// his class record exactly as an unresolvable body already does.
//
// mage IS READ STRAIGHT OFF THE PARTY MEMBER'S OWN FIELD, hero's and
// startWeapon's own precedent applied to the class axis: mapload.PartyMember
// already carries it (plan D-5's own rejection of reading a contested
// profile flag instead), so this is a plain copy and not a second
// derivation.
type invPartyGear struct {
	hero        data.Hero
	profile     data.Profile
	startWeapon *data.Weapon
	figureDir   data.FigureDir
	figureFace  int
	figureHero  bool
	table       *mapload.Table
	list        data.BodyList
	mage        bool
	primary     bool

	class              int32
	hired              bool
	hiredRotationSpeed int32
}

func primaryPlayerHero(p mapload.PartyMember) bool {
	return p.StartingHero
}

// guardedEntities is the mission-loss set: the entity of every party member who
// is a player character rather than a hired unit.
//
// THE TWO CONDITIONS ARE THE PARTY MODEL'S OWN (mapload.PartyMember).
// PlayerCharacter distinguishes a full character from a mercenary, and
// MercenaryType is the hire's own 1-based type; a member is guarded when it is
// the first and not the second. Both are tested rather than one, because the
// two fields are written by different constructors — the tavern writes the type
// and leaves PlayerCharacter clear, an original save's restore writes
// PlayerCharacter — and a future constructor that sets both would otherwise
// make a hire lose the mission.
//
// StartingHero IS NOT TESTED. It names the one primary character, which is
// exactly the narrowing this hotfix removes.
func guardedEntities(party []mapload.PartyMember, ids []sim.EntityID) []sim.EntityID {
	var out []sim.EntityID
	for i, p := range party {
		if i >= len(ids) {
			break
		}
		if p.PlayerCharacter && p.MercenaryType == 0 {
			out = append(out, ids[i])
		}
	}
	return out
}

// isGuarded reports character identity for the UI. Ownership transfers change
// automatic loss eligibility, not whether the actor is a player character.
func (mw *mapWorld) isGuarded(id sim.EntityID) bool {
	// A BARE MAP HAS NO MISSION AT ALL: `newMapWorld` leaves mw.mission nil
	// until a mission opens, and `entityDraws` is reached from `push` inside
	// the constructor, so this is asked once before the field exists on every
	// map the editor and the tests load.
	if mw.mission == nil {
		return false
	}
	for _, g := range mw.mission.guarded {
		if g == id {
			return true
		}
	}
	return false
}

// FaceSource is where the picture of a speaker named in a mission's event
// text comes from.
//
// IT IS ONE METHOD BECAUSE ONE HOP WAS MISSING. Everything else about the
// portrait was settled — the pane's rectangle, which of the window's two shapes
// a file gets, which part changes the face, and what happens to a part that
// names nobody — while where the art is kept was not: the picture-bearing keys
// of the registry the speaker numbers reach were published with their domains
// measured and their meaning graded Unknown. So the answer was a seam rather
// than a guess. `REG-NPC-088` has since named the fields, and the seam stays: it
// is still the driver that holds the mission's archive, its unit classes and its
// party subject, and none of those can cross into the drawing tier.
//
// IT ALSO CARRIES THE WINDOW, which is the second half of what the pane draws.
// `REG-NPC-089` makes the crop a per-speaker rectangle stated in the registry,
// so the picture alone no longer answers "what does this pane show"; the
// rectangle is in the PICTURE's own top-down pixels and the zero value means the
// record states none. Returning it beside the picture rather than through a
// second method is what keeps the two one statement about one speaker.
//
// NOT FINDING ONE IS NOT AN ERROR and there is no error return to say it was.
// The window draws its pane empty and everything else about it works, which is
// the only thing a caller could do with a failure anyway; an error here would be
// an invitation to log or to fall back, and each of those shows the player
// something the original does not.
//
// The picture is BORROWED. The window reads it while composing and does not copy
// it, so an implementation that reuses one buffer must hand it over again after
// changing it.
type FaceSource interface {
	SpeakerFace(speaker int) (*image.RGBA, image.Rectangle, bool)
}

// speakerFace is the picture for the speaker of the part being shown and the
// window the pane cuts out of it, or nil.
//
// A NIL SOURCE AND A SOURCE THAT MISSES ANSWER THE SAME, deliberately: both mean
// "this window cannot show whose face this is", which is the one thing the
// window can act on, and the shipped path is the first of the two. A miss drops
// the window with the picture — a rectangle with nothing to cut it out of is not
// a partial answer.
func (m *missionNotices) speakerFace(speaker int) (*image.RGBA, image.Rectangle) {
	if m.faces == nil {
		return nil, image.Rectangle{}
	}
	img, win, ok := m.faces.SpeakerFace(speaker)
	if !ok {
		return nil, image.Rectangle{}
	}
	return img, win
}

// dialogueOf is the push for part n of the open file: its words, this file's
// shape, whether this part names a speaker and that speaker's picture.
//
// IT IS ONE PLACE AND NOT TWO. Opening a window and paging one differ in the
// part number and in nothing else, so building the push here is what keeps a
// paged window from being assembled by a second rule that could disagree with
// the opening one about the shape.
func (m *missionNotices) dialogueOf(body string, n int) ui.Dialogue {
	speaker, named := EventPartSpeaker(m.payload, n, m.dialogueAudience)
	d := ui.Dialogue{Text: body, Portrait: m.portrait, Speaks: named}
	if named {
		d.Face, d.FaceWindow = m.speakerFace(speaker)
	}
	return d
}

// missionNotices is the driver's whole notice state.
//
// THE DRIVER HOLDS IT AND THE DRIVER ALONE DECIDES. The viewer holds what it
// draws and a flag saying whether it draws one; which part of an event text is
// showing, whether a second announcement was dropped and whether the mission is
// over live here. So "a second announcement while one is open is discarded" is
// one test in one place rather than a rule two tiers must agree about.
type missionNotices struct {
	// number is the mission the world was built for, and src is what its event
	// text is read out of. Both are fixed when the mission opens.
	number int
	src    entrySource
	party  []mapload.PartyMember
	ids    []sim.EntityID
	// entryCount is the prefix minted at mission entry. Joined map actors live
	// in the world bytes of an in-mission save and must not be minted a second
	// time from Snapshot.Party when that save is restored.
	entryCount int
	// entryQuickSpells are the front end's quick-spell bindings at the moment
	// this driver became the live one; leaving the mission without completing
	// it puts them back.
	entryQuickSpells [4]uint32
	// resumed is set when the driver was restored from a SAV, which holds the
	// mission's party and no town party.
	resumed bool
	state   *Mission
	table   *mapload.Table
	list    data.BodyList

	// ann derives, from the world's own latch array, which announcements each
	// step raised. It reads the world and writes nothing to it.
	ann *Announcer

	// audience is who the event text's eight conditional tag arms test: the
	// mission's own subject, and the closure that answers for a tag's speaker.
	//
	// IT IS BUILT ONCE WHEN THE MISSION OPENS. The hero half cannot change
	// inside a mission, and settling it here is what makes paging safe: part
	// n+1 is selected by exactly the audience part n was, so a page cannot show
	// what the same person was refused a moment earlier.
	audience         EventAudience
	dialogueAudience EventAudience

	// open is whether a notice is up, kind which surface it is, payload the
	// event text a dialogue notice is paging through and part which part of it
	// is showing.
	//
	// The PAYLOAD is held rather than the part's text, because paging asks the
	// same file for part n+1 — and re-reading the archive on every press would
	// put a file read on the input path and could answer differently the second
	// time.
	open            bool
	kind            ui.NoticeKind
	payload         []byte
	part            int
	tips            int
	pendingMessages []int32
	objectiveLabels []string
	npcKeys         map[sim.EntityID]uint16
	failureText     []string

	// pages is a notice whose text this tier composed and paged ITSELF, rather
	// than one read out of an event file (1032 return 1). The load disclosure
	// is the only one: it is longer than the window draws, it has no event file
	// to page out of, and it must be readable in full. part indexes it from 1,
	// the same way it indexes an event text's parts, so advanceNotice counts
	// the same number for both shapes.
	//
	// A nil pages is the event-text shape and is what every other notice opens
	// with, so the two cannot be confused: payload and pages are never both set.
	pages []string

	// portrait is whether THIS FILE's window carries a portrait pane, settled
	// once when the window opens and carried unchanged across every page of it.
	//
	// It is held HERE rather than re-derived per page for the reason the payload
	// is: the test is over the whole file, so asking it again at each part is
	// asking the same question of the same bytes, and asking it of anything
	// smaller would be asking a different one.
	portrait bool

	// faces is where a speaker's picture comes from, and it is ALLOWED TO BE
	// NIL — which is what it is on every path that ships today.
	//
	// A nil source resolves nothing, so every named speaker reaches the window
	// with no picture and the pane stands empty. That is the whole of this
	// story's open hop: no published claim identifies the resource a speaker's
	// face is kept in, so nothing in this tree can implement this interface yet
	// without guessing, and guessing is what golden rule 4 refuses. When the
	// decode lands, one implementation of one method closes it and no contract,
	// geometry, test or call site moves.
	faces FaceSource

	// MISSION-END-013
	guarded []sim.EntityID

	// departed remembers a character's transfer after its body disappears.
	// The entering party is retained for stable save IDs, not loss eligibility.
	// Native residue carries this membership across a cold load.
	departed map[sim.EntityID]bool

	// announced is whether the outcome has been latched, outcome which way it
	// went, and outcomeShown whether its notice has reached the screen.
	//
	// A MISSION IS DECIDED ONCE AND THE NOTICE IS SHOWN ONCE. The outcome
	// latches in the simulation, so without this flag every step after the
	// decision would re-open the banner and no press could ever dismiss it.
	announced    bool
	outcome      sim.Outcome
	outcomeShown bool
	// delayedVictory is the live-session permission created by Continue. It is
	// presentation/session state, not simulation state, and is deliberately
	// reconstructed false when a save opens. victoryTaken makes the campaign
	// completion boundary one-shot even if a caller repeats the action.
	delayedVictory bool
	victoryTaken   bool
}

// The words the outcome notice states when the install does not supply them, and
// the sentence the win path leaves on the map list.
//
// THE FIRST TWO ARE NOW A FALLBACK AND NOT THE ANSWER. The original's panels
// state global string slots 140 and 141 (`MENU-STRTAB-008`), which this
// build reads through ui.Words; these are what it draws where the install
// states no line there. The third is ours outright: the destination the win
// path cannot reach does not exist to have a sentence.
const (
	// MissionWonText and MissionLostText are the two authored endings, and they
	// are what makes the two distinguishable (0066 AC-14) — the frame and the
	// geometry are the same box.
	//
	// They are ALIASES of the ui values since 0168, which is where the eleven
	// authored words live together. The name is kept because callers already use
	// it and because the value has not changed.
	MissionWonText  = ui.MissionWonText
	MissionLostText = ui.MissionLostText

	// TownNotBuiltMessage is stated on the map list when a WON mission is
	// dismissed, IN THE RUNNING PROGRAM. It is THIS DRIVER'S OWN answer, and
	// this driver reads no campaign — a silent return to the list would read
	// as an ordinary one.
	//
	// SINCE 0142 THERE IS A TOWN and the sentence no longer says there is
	// not. What still reaches this is a win that got no further: a front end
	// with no campaign, a loose map picked off the list, or a run that has
	// not reached the town's own boundary. continuity (frontend.go) is what
	// overrides it wherever the campaign has more to say, and the name is
	// kept because what it names — the destination that is not a town — is
	// the same one.
	TownNotBuiltMessage = "mission won - nothing here leads on from it: choose a row"
)

// missionOutcomeText is what the outcome notice says for each ending. An
// undecided outcome has no words, and it cannot reach here: the notice is opened
// only on a transition out of undecided.
//
// THE WORDS COME OFF THE VIEWER. It holds the resolved set the front end
// read from the install, so the sentence this states and the word the
// notice's own button states are one value. A nil viewer — every headless
// driver — answers the authored English, which is exactly what this
// function returned before the seam existed.
func missionOutcomeText(v *ui.Viewer, o sim.Outcome) string {
	w := ui.AuthoredWords()
	if v != nil {
		w = v.Words()
	}
	return w.OutcomeText(o == sim.OutcomeLost)
}

// openMission builds the driver under a started campaign mission: the world
// the start built, that map's schedule and tiers, and the notice state the
// mission needs.
//
// IT BUILDS NO WORLD. StartMission already did, from the same map, and a second
// build here would be a second answer to "what does this mission start as".
//
// src is where the mission's event text is read from — the archive set the front
// end holds — and it is a parameter for the reason the definition table is one:
// this tier opens nothing, and a driver that reached for an install itself would
// give one mission two behaviours depending on when it was called.
func openMission(ms *Mission, t *mapload.Table, units *terrain.UnitSet, v *ui.Viewer,
	src entrySource, faces FaceSource, npcFaces map[int32]data.NPCFace) *mapWorld {
	campaignOf(tableGame(t)).openMission(v)
	normalizeMissionShieldLoadouts(ms, t)
	// NO SCHEDULE. A mission's units are moved by its script and by the AI, and
	// by nothing else. The owner saw it and asked for it out; the generator now
	// exists only as test support, in schedule_test.go, where its one remaining
	// use is written down. THE DRAWN BODY IS DERIVED HERE, NOT TRUSTED FROM THE
	// RECORD (owner defect reports). It has to run BEFORE partyArt below, which
	// keys the bundle off PartyMember.Body and PartyMember.BodyDir.
	list, _ := ReadBodyList(src)
	canonicalizePartyAppearance(ms, units, src, list)
	tiers := savedActorTiers(ms.Map, entityTiers(ms.Map, t), ms.World.Entities(), ms.savedDocument)
	mw := newMapWorldWith(ms.World, nil, units,
		tiers, missionCharacters(ms.Map, t, ms), missionAppearanceArt(ms, units, list, src), v)
	mw.installActorManifest(ms.ActorManifest)
	mw.installCharacterDerivations(ms, t)
	v.SetItemCastSink(mw.useScroll)
	v.SetMusicAreas(missionMusicAreas(ms.Map), mw.musicHero)
	// The human band's own lookup, beside the creature band's tier and resolved
	// from the same map and table (figures.go). It is assigned rather than
	// passed because the constructor's parameter list already carries seven and
	// an eighth would be a second thing every test fixture has to state.
	mw.figures = partyFigures(savedActorFigures(ms.Map, t, ms.World.Entities(), ms.savedDocument), ms.Party, ms.Start.IDs)
	for id, member := range ms.Start.Roster {
		if mw.figures == nil {
			mw.figures = map[sim.EntityID]figureID{}
		}
		mw.figures[id] = rosterFigureID(member, mw.world, id)
	}
	// THE BOOK'S OWN NAMES, openMapWorld's own reasoning applied to the mission
	// door: resolved once off the same table t, and set directly on the struct
	// so newMapWorldWith's own signature stays untouched.
	mw.spellNames = spellNamesFrom(t, v.Words().SpellBookNames)
	// WHO CAN SPEAK AND WHAT THEY LOOK LIKE (speakers.go), carried in rather
	// than resolved here: the table is the campaign's and is read once.
	mw.npcFaces = npcFaces
	// WHO A DIALOGUE CAN BE ABOUT (speakeractors.go), resolved off the same map
	// and table the figures above came from. A record for which no live actor
	// answers uses the original's bare synthesised arm.
	mw.speakerActors = missionSpeakers(ms.Map, t, ms.Start.Roster, ms.Party, ms.Start.IDs,
		placedEntities(ms.Map, ms.World.Entities(), ms.Start.Roster, ms.savedDocument))
	// THE DRIVER IS ITS OWN FACE SOURCE WHERE NOTHING ELSE IS SUPPLIED (0141).
	// The seam was left open because no published claim named a resource for
	// that pane. `REG-NPC-088` has since named the two fields each key is
	// stored into (data/npcface.go), and the seam stays anyway: this driver is
	// the one object that holds both the campaign's speaker table and the party
	// subject a speaker record can refer to instead of carrying a picture, and
	// neither of those can cross into the drawing tier. A caller that supplies
	// its own FaceSource still wins, which is what keeps the seam a seam.
	if faces == nil {
		faces = mw
	}
	mw.mission = &missionNotices{
		number:     ms.Number,
		src:        src,
		party:      ms.Party,
		ids:        ms.Start.IDs,
		entryCount: len(ms.Party),
		state:      ms,
		table:      t,
		list:       list,
		faces:      faces,
		// The raise list is the COMPILE'S and not the compiled program's — see
		// NewAnnouncer for the aliasing that makes the difference load-bearing.
		ann: NewAnnouncer(ms.World, ms.Raises),
	}
	mw.mission.audience = speakerAudience(HeroAudience(ms.Party), npcFaces)
	if ms.ActorManifest != nil {
		for _, actor := range ms.ActorManifest.Actors {
			if _, present := mw.chars[actor.ID]; !present {
				mw.rememberCheatCharacter(actor.ID, actor.Name)
			}
		}
	}
	mw.mission.campaign().loadMissionText(mw, src, ms, t)
	// Start.IDs is parallel to the entering party. Later joiners are added by
	// syncJoinedHeroes; guardedCharacterLost reads their current ownership.
	mw.mission.guarded = guardedEntities(ms.Party, ms.Start.IDs)
	// A restored world may already contain a map actor handed to the player.
	// Discover it before the inventory and first post-open projection are built;
	// the save's party prefix deliberately did not mint it a second time.
	mw.syncJoinedHeroes()

	subject, _ := buildInventorySubject(src, ms)

	// THE PACK, COMPOSED ONCE ALONGSIDE THE FIGURE. buildInventoryPack is
	// called from here and from refreshPack (paced, below) and nowhere else —
	// T4's own rule for the figure, extended to the one part of the subject a
	// tick can change.
	//
	// invSubjectSet REPEATS buildInventorySubject's OWN CONDITION rather than
	// reading it back off subject, because subject's zero value cannot say
	// which case it is: a party-less mission and a mission whose subject
	// really is entity 0 both leave subject.ID at 0 (spec: guard the
	// empty-party case).
	mw.invSubjectSet = len(ms.Party) > 0 && len(ms.Start.IDs) > 0
	if !mw.invSubjectSet {
		subject = ui.InventorySubject{}
	}
	if mw.invSubjectSet {
		mw.invIconCache = map[uint16]*image.RGBA{}
		// CarriedStacks, not Carried: the pack draws one cell per ELEMENT with its
		// count, and only the elements reader carries the count back out of the
		// world at all.
		mw.invCodes, _ = mw.world.CarriedStacks(sim.EntityID(subject.ID))
		subject.Pack, subject.PackCount, _ = buildInventoryPack(src, mw.invCodes, mw.invIconCache)
		subject.PackStars = itemStackStarFlags(mw.invCodes)

		// invParty IS RESOLVED HERE, ALONGSIDE THE FIGURE (docs/0124-equip-
		// from-the-pack T4; plan D-5, R-1): the same party member
		// buildInventorySubject just drew the figure from, and the same t
		// this constructor already received and had nowhere to keep before
		// this story. figureDir and figureFace are memberFigure's own pair,
		// recomputed rather than threaded out of buildInventorySubject's
		// fenced signature — see invPartyGear's own doc.
		//
		// list AND mage JOIN THE SAME LITERAL (docs/0134-what-he-wears T4;
		// plan D-6): list off src, the SAME archive source buildInventorySubject
		// was just handed, and mage straight off the member — see
		// invPartyGear's own doc for why each is read the way it is.
		member := ms.Party[0]
		figureDir, figureFace := memberFigure(member)
		mw.invParty = invPartyGear{
			hero: member.Hero, profile: member.Profile, startWeapon: mapload.MemberWeapon(member, t),
			figureDir: figureDir, figureFace: figureFace, figureHero: memberFigureID(member).Hero, table: t,
			list: list, mage: member.Mage, primary: primaryPlayerHero(member),
		}
	}
	mw.invSubject = subject
	if mw.invSubjectSet {
		// Seed the history before currentFigureEquipment is read. A real opening
		// slot 1 disables the display fallback immediately; an empty slot leaves
		// it available for a starting weapon that never reached the array.
		// resolveWeaponMaterialized also folds in ms.Party[0]'s own PERSISTED
		// history (round-2 adversarial review, fifth pass): a subject who resolved
		// the fallback in a PREVIOUS mission and then sold or dropped the
		// resulting item carries no trace of that into THIS mission's own
		// equipment or stock, which is exactly what a present-state read of eq and
		// mw.invCodes alone cannot see.
		eq := mw.currentEquipment()
		mw.invWeaponEverEquipped = mw.resolveWeaponMaterialized(&mw.mission.party[0], eq)
		figureEq := mw.currentFigureEquipment()
		// THE POPUP'S OWN TEXT (0151, defect 5), composed off the SAME equipment
		// buildInventorySubject just composed the figure from —
		// currentFigureEquipment (round-2 adversarial review, counterexample 2),
		// not the raw array alone: buildInventorySubject applies member.Weapon's
		// own slot-1 fallback to the figure and mask, and SlotInfo has to agree,
		// or a subject drawn wearing a fallback weapon (a companion whose starting
		// weapon was never folded into the array — rosterTemplate,
		// pkg/mapload/spawn.go) shows a popup and an icon over an empty slot.
		figureItems := mw.currentFigureEquipmentItems()
		mw.invSubject.SlotInfo = slotItemInfoLinesWithWeaponDamage(figureItems, t, mw.itemWeaponDamage, mw.view.Words())
		// WeaponFallback (counterexample 4, round-2 adversarial review, third
		// pass), recomputed here rather than trusted from buildInventorySubject's
		// own return: mw.invParty is now set, so currentWeaponFallbackActive reads
		// the live entity the same way SlotInfo just did, one statement up — the
		// single source of truth this file already keeps for the fallback-aware
		// equipment itself.
		mw.invSubject.WeaponFallback = mw.currentWeaponFallbackActive()
		mw.invSubject.PackInfo = packInfoLinesWithWeaponDamage(mw.invCodes, t, mw.itemWeaponDamage, mw.view.Words())
		if mw.invParty.primary {
			appendInventoryGold(&mw.invSubject, mw.previewPurse(), loadInventoryGoldIcon(src))
		}
		// Seed every change tracker from this same opening state: the first
		// paced frame is presentation-only, not a synthetic equipment change
		// that re-arms or repaints a restored character.
		//
		// invEquipment (rearm's own guard) SEEDS FROM THE RAW ARRAY, not the
		// figure equipment above (counterexample 2's own scoping): rearm
		// resolves the fallback itself, gated on invWeaponEverEquipped, and a
		// tracker seeded wide here would mask a real first equip from it.
		// invFigureEquipment (refreshEquipment's own guard) seeds from the
		// SAME fallback-aware equipment the figure and SlotInfo above were
		// just built from, so a later real change to slot 1 is still detected
		// (currentFigureEquipment stops adding the fallback the moment the
		// array itself carries a code) and a frame where nothing changed
		// costs the one compare refreshEquipment's own doc promises.
		mw.invEquipment = eq
		mw.invEquipmentItems = mw.currentEquipmentItems()
		mw.invLayers, mw.invFigureLayers = mw.activeLayers(sim.EntityID(subject.ID)), nil
		mw.invFigureEquipment = figureEq
		mw.invFigureEquipmentItems = figureItems
		// invComposedEquipment (C1b) seeds from missionDollEquipment's own answer
		// over mw.world/subject.ID/ms.Party[0] — the SAME three arguments
		// buildInventorySubject just passed it, above, to compose
		// subject.Figure/Slots itself — rather than from eq or figureEq
		// directly. Since missionDollEquipment itself now prefers the live world
		// (round-2 adversarial review, twelfth pass, C1's own second correction,
		// see that function's doc), this seed and eq/ figureEq typically agree in
		// practice, but this call is kept SEPARATE rather than reused: it is what
		// would still catch a future buildInventorySubject that stops calling
		// missionDollEquipment for its own figure, which a live re-derivation
		// alone could never witness (see invComposedEquipment's own field doc).
		mw.invComposedEquipment, _ = missionDollEquipment(mw.world, sim.EntityID(subject.ID), ms.Party[0])
		// partyArt already installed a saved/generated body when that body was
		// resolvable at open. Seed the appearance tracker only in that case: it
		// prevents a synthetic first-frame repaint of a restored hero. A caller
		// with no opening body (notably the focused compositor fixtures) still
		// gets the ordinary first refresh from its equipment.
		//
		// figureEq, NOT eq (round-2 adversarial review, fifth pass, counterexample
		// I): refreshAppearance's own comparison widened to currentFigureEquipment
		// alongside this seed. Seeding the narrow eq here for a fallback-only
		// member (DIV-070's population) would disagree with that comparison on the
		// very first later call — still resolving the SAME body correctly, but
		// through a redundant recompute rather than the skip this seed exists to
		// buy, exactly the "synthetic first-frame repaint" this comment already
		// says to prevent.
		if _, ok := mw.art[sim.EntityID(mw.invSubject.ID)]; ok {
			mw.bodyEquipment = figureEq
		}
		if e, ok := mw.entity(sim.EntityID(mw.invSubject.ID)); ok {
			mw.invSkill, mw.invSkillSet = e.Skill, true
			mw.invTraining = e.NativeTraining
		}
	}
	v.SetInventorySubject(mw.invSubject)
	// newMapWorldWith's constructor push necessarily ran before mission state
	// existed. Repeat the pure projection now that the guarded/player-character
	// set is installed, including any joined actor recovered from a saved world.
	mw.push()
	words := v.Words()
	announce(v, pickupLinesForItems(ms.pendingPickups, t, &words))
	ms.pendingPickups = nil

	// A saved Player outcome is already reported state, not a new script edge.
	// Present it before a tick can run; NewAnnouncer has independently consumed
	// the saved message latches, so no old dialogue or reward is replayed.
	if outcome := mw.world.Outcome(); outcome != sim.OutcomeUndecided {
		mw.mission.announced, mw.mission.outcome = true, outcome
		mw.showOutcome()
	}
	mw.seedSkillBaseline()
	return mw
}

// syncJoinedHeroes promotes every newly owned persistent roster actor into the
// running mission's ordinary party surfaces. The simulation is the producer:
// BoundarySurvivors supplies only living, self-owned human actors. The loader's
// roster supplies identity and immutable row inputs; CarryRosterIDs supplies
// the actor's live skill, pack and equipment state without reconstructing it.
func (mw *mapWorld) syncJoinedHeroes() {
	if mw == nil || mw.world == nil || mw.mission == nil || mw.mission.state == nil {
		return
	}
	if !mapload.RosterJoinPending(mw.world, mw.mission.ids, mw.mission.state.Start.Roster) {
		return
	}
	party, ids := mapload.CarryRosterIDs(mw.mission.party, mw.world, mw.mission.ids,
		mw.mission.state.Start.Roster)
	if len(party) <= len(mw.mission.party) {
		return
	}
	first := len(mw.mission.party)
	mw.mission.party, mw.mission.ids = party, ids
	mw.mission.state.Party = party
	mw.mission.state.Start.IDs = ids
	for i := first; i < len(party) && i < len(ids); i++ {
		id := ids[i]
		member := &mw.mission.party[i]
		// A joined actor's roster member is the canonical mission-local
		// resolution of its figure identity. entityFigures was built before the
		// handover from the install-wide table, which cannot resolve composed
		// npc23/npc24 rows without the entering hero. Install the same directory
		// and face used by the inventory compositor before this tick's push.
		if mw.figures == nil {
			mw.figures = make(map[sim.EntityID]figureID)
		}
		mw.figures[id] = memberFigureID(*member)
		if slots, ok := mw.world.Equipped(id); ok {
			body, dir, class, matched := data.HeroAppearance(mw.mission.list,
				equipmentFromSlots(slots), member.Mage, false)
			if matched {
				member.Body, member.BodyDir, member.Class = string(body), dir, class
				LoadHeroBody(mw.mission.src, mw.units, dir, body)
				if mw.units != nil {
					if art := mw.units.Bodies[data.HeroBodyKey(dir, body)]; art != nil {
						if mw.art == nil {
							mw.art = make(map[sim.EntityID]*terrain.UnitClass)
						}
						mw.art[id] = art
					}
				}
			}
		}
		if mw.chars == nil {
			mw.chars = make(map[sim.EntityID]ui.UnitCharacter)
		}
		mw.chars[id] = partyPanelSubject(*member, mw.mission.table).Char
		mw.mission.guarded = append(mw.mission.guarded, id)
	}
	// state.Party must receive the appearance writes made through mission.party.
	mw.mission.state.Party = mw.mission.party
}

// partyCharacters is the character each party member was placed with, keyed
// by the entity id the start minted for him.
//
// EVERY MEMBER IS THE PERSON BAND: a party member is a person for every
// purpose of the sheet contract, and this is that rule applied to the one
// path this story does not otherwise touch — the band is the only field
// this story adds here, and no number below it changes.
//
// IT PAIRS THE START'S OWN TWO SLICES and derives neither: the ids come from
// Start.IDs, the statistics from the member the mission was started with, and
// both are in party order. Recomputing an id here — "the party is appended, so
// it is the last one" — would be this package holding a copy of a rule that
// lives in the loader.
//
// partyPanelSubject computes the complete table-backed projection once. The
// generator preview calls the same helper before a mission exists.
//
// A mission started with no party yields an EMPTY LOOKUP rather than a nil one
// only when there is something to put in it; either way every id misses, and a
// miss states no character. The two slices are walked to the SHORTER of the two,
// so a Start from before ids were recorded pairs nothing rather than panicking.
func partyCharacters(ms *Mission, tables ...*mapload.Table) map[sim.EntityID]ui.UnitCharacter {
	var table *mapload.Table
	if len(tables) > 0 {
		table = tables[0]
	}
	n := len(ms.Party)
	if len(ms.Start.IDs) < n {
		n = len(ms.Start.IDs)
	}
	if n == 0 {
		return nil
	}
	out := make(map[sim.EntityID]ui.UnitCharacter, n)
	for i := 0; i < n; i++ {
		p := ms.Party[i]
		out[ms.Start.IDs[i]] = partyPanelSubject(p, table).Char
	}
	return out
}

// partyPanelSubject is the one projection from a generated party member to a
// sheet subject. The generator preview and a newly started mission use this
// expression, so Card values cannot drift from the values the first panel
// receives.
func partyPanelSubject(p mapload.PartyMember, t *mapload.Table, installed ...ui.Words) ui.PanelSubject {
	d, health, mana := mapload.PartyDisplayWithTable(p, t)
	words := ui.AuthoredWords()
	if len(installed) > 0 && installed[0] != (ui.Words{}) {
		words = installed[0]
	}
	// THE PAIR'S SECOND NUMBER IS THE DERIVED MAXIMUM, NOT THE CURRENT POOL
	// AGAIN. A generated member spawns at his maximum, so reading the pool
	// twice was indistinguishable from reading the maximum and stood here
	// unnoticed. A source-restored character carries his own current health
	// and mana, and for him the two differ: the card printed 92/92 for a
	// character whose maximum was 98, and -- the owner-visible half -- the
	// pair then stayed 92/92 when an equipped Body item raised that maximum
	// to 98, so the only number on the card that moved was Body itself.
	// A subject with no derived pool at all keeps the pool it had, which is
	// what leaves the row's own positivity gate (panelPoolPositive) alone.
	maxHP, maxMana := health, mana
	if d.HealthMax > 0 {
		maxHP = d.HealthMax
	}
	if d.ManaMax > 0 {
		maxMana = d.ManaMax
	}
	s := ui.PanelSubject{
		Name: p.Name, HP: int(health), MaxHP: int(maxHP), Mana: int(mana), MaxMana: int(maxMana),
		UnitNameIndex: int(p.Class), DetailLevel: 7, DetailSet: true, Words: words,
		OriginalPanel: originalPanelActor(p.PlayerCharacter, p.Class, 0),
		Combat: ui.UnitCombat{Known: true, DamageBase: int(d.Combat.DamageBase),
			DamageSpread: int(d.Combat.DamageSpread), ToHit: int(d.Combat.ToHit),
			Defence: int(d.Combat.Defence), Absorption: int(d.Combat.Absorption),
			AttackCharge: int(d.Combat.AttackChargeTime), AttackRelax: int(d.Combat.AttackRelaxTime),
			AlwaysHits: d.Combat.AlwaysHits},
		Speed: int(d.Speed),
		// The carried load for a member who is in no mission. There is no
		// Entity.Load to read outside one, so it is resolved off his own worn set
		// and carried list through the same single statement of the law a live
		// actor's load goes through (mapload.PartyLoad).
		Weight: int(mapload.PartyLoad(p, t)), WeightKnown: true,
		Char: ui.UnitCharacter{Known: true, UnitNameIndex: int(p.Class), Name: p.Name, Mage: p.Mage, Band: ui.CharacterBandPerson,
			Body: int(d.Body), Reaction: int(d.Reaction), Mind: int(d.Mind), Spirit: int(d.Spirit),
			Experience: int(d.Experience), Sight: int(d.Sight)},
	}
	for i := range s.Char.Skills {
		s.Char.Skills[i] = int(d.Skill[i])
	}
	if t != nil {
		if name, ok := t.Mods.CharacterName(p.Name); ok {
			s.Name, s.Char.Name = name, name
		}
	}
	if p.Carry != nil {
		s.Char.Experience = 0
		for _, xp := range p.Carry.SkillXP {
			s.Char.Experience += int(xp)
		}
	}
	s.Char.Sight256 = data.SightWord(d.Mind, d.Reaction, d.Sight)
	if h, ok := p.OriginalHumanState(); ok {
		s.MaxHP, s.MaxMana = int(int16(h.HealthMax)), int(int16(h.ManaMax))
		s.Char.Sight256 = h.Sight
		s.Weight, s.Speed = int(int16(h.Load)), int(int16(h.Speed))
		s.Char.Experience = int(int32(h.Experience))
	}
	if p.Carry != nil && p.Carry.LiveLoad != nil {
		s.Weight, s.Speed = int(p.Carry.LiveLoad.Load), int(p.Carry.LiveLoad.DisplaySpeed())
	}
	skillXP := d.SkillXP
	if p.Carry != nil {
		skillXP = p.Carry.SkillXP
	}
	// A generated person's original +0x1c value is one percent of the five
	// trained skill-XP dwords (slots 1..5). Only its nonzero state is consumed here.
	// Player characters are already excluded by flags bit 0; the value still
	// follows the actor projection for non-player generated people.
	s.OriginalPanel.XPValue = partyOriginalPanelXPValue(p, t, originalHumanPanelXPValue(skillXP))
	for i := range s.Char.Protection {
		s.Char.Protection[i] = int(d.Protection[i])
		s.Char.Resistance[i] = int(d.Resistance[i])
	}
	if weapon := mapload.MemberWeapon(p, t); weapon != nil {
		s.Char.Weapon = weapon.Name
	}
	spellID, _ := mapload.SpellIDByToken(t, d.Combat.SpellName)
	spell, ok := sim.WeaponSpellCharacteristicsFor(mapload.TableRules(t), sim.Entity{MaxMana: mana, WeaponSpell: spellID,
		WeaponSpellLevel: d.Combat.SpellPower}, mapload.SpellRules(t))
	if ok {
		s.Combat.WeaponSpellKnown = true
		if spell.HasDamage {
			s.Combat.WeaponSpellDamageKnown = true
			s.Combat.SpellDamageBase, s.Combat.SpellDamageSpread = spell.DamageMin, spell.DamageMax-spell.DamageMin
		}
	}
	return s
}

func originalPanelActor(playerCharacter bool, typeID int32, xpValue int32) ui.OriginalPanelActor {
	actor := ui.OriginalPanelActor{Known: true, XPValue: xpValue}
	if playerCharacter {
		actor.Flags |= 0x1
	}
	if typeID >= 0 && typeID < 0x1a {
		actor.Flags |= 0x10
	}
	if typeID == 0x49 { // UNIT-PANEL-010
		actor.Byte14A = 2
	}
	return actor
}

func originalHumanPanelXPValue(skillXP [data.SkillSlots]int32) int32 {
	return sim.HumanExperienceValue(skillXP)
}

// partyOriginalPanelXPValue returns actor+0x1c for the two party shapes. A
// generated person stores one percent of his six skill-XP values there
// (ITEM-ARMFOLD-033). A hired flat unit retains its Units-row XPValue, the
// same value mapload puts on its mission entity (AI-WITHDRAW-027). The lookup
// is needed only before a mission exists; once placed, entityDraws reads the
// live Entity.XPValue directly.
func partyOriginalPanelXPValue(p mapload.PartyMember, t *mapload.Table, humanXP int32) int32 {
	if p.PlayerCharacter || p.Class < 0x1a || t == nil || t.Units == nil {
		return humanXP
	}
	for i := 1; i < t.Units.Len(); i++ {
		def, err := data.NewUnitDef(t.Units.EntryName(i), t.Units.EntryParams(i))
		if err == nil && def.TypeID == p.Class {
			return def.XPValue
		}
	}
	return humanXP
}

// canonicalizePartyAppearance re-derives every party member's DRAWN BODY from
// the equipment his entity actually holds, and loads that body's art, before
// partyArt below reads the two fields it keys the bundle on.
//
// THE DEFECT IT CLOSES (owner, two reports in one message: a mage with no
// staff equipped drawn holding one, and a fighter who bought and wore a full
// set in the shop walking onto the map bare). PartyMember.Body, .BodyDir and
// .Class were derived ONCE, when the party was assembled — assembleParty
// for a generated hero, restoredPartyMember for a saved one,
// canonicalizeJoinedRoster for a joined NPC — and never again. Every later
// equipment change left them naming the loadout the member was assembled in:
// the shop's own tables, a mid-mission equip, a sale, and
// mapload.CarryParty, which carries the new worn set across a mission
// boundary and does not touch the three appearance fields. partyArt then
// painted the stale body at the next open. The map's refreshAppearance
// masked it for the ONE member who was the inventory subject and only after
// his equipment changed again, which is why taking a single piece off
// corrected the picture at once.
//
// IT IS A DERIVATION AND NOT A TRACKER, which is what closes the class rather
// than one instance: no producer of an equipment change has to remember to
// update an appearance field, because no producer owns one. The same reasoning
// canonicalizeJoinedRoster (frontend.go) already applies to a joined roster
// entry, applied to the party and moved to the point where the world is built.
//
// THE WORLD IS THE SOURCE, NOT PartyMember.Worn OR Carry.Equipped: by this
// point the start has stocked every entity and a resumed save has replaced the
// started world outright, so the entity's own slots are the only reading that
// cannot be one boundary out of date. A member the world does not hold is left
// exactly as his record stands.
//
// SLOT 1 IS WIDENED BY THE STARTING-WEAPON FALLBACK, currentFigureEquipment's
// own rule (world.go) restated for an arbitrary member rather than for the
// inventory subject: DIV-070's population is a member whose starting weapon
// never reached the equipment array, and he is drawn holding it until it
// materializes. Reading the raw array here would draw that member bare-handed
// on the map while his own doll shows him armed, which is the disagreement
// refreshAppearance was widened to fix.
//
// AN UNMATCHED NAME LEAVES THE THREE FIELDS ALONE, canonicalizeJoinedRoster's
// own precedent: a body that fell back on the law's default is not worth
// overwriting a resolved one with. An empty body list — every synthetic
// fixture, which has no archive to read heropicture.txt from — matches
// nothing, so this function is a no-op there and changes no existing test.
func canonicalizePartyAppearance(ms *Mission, units *terrain.UnitSet, src entrySource, list data.BodyList) {
	if ms == nil || ms.World == nil {
		return
	}
	n := len(ms.Party)
	if len(ms.Start.IDs) < n {
		n = len(ms.Start.IDs)
	}
	for i := 0; i < n; i++ {
		member := ms.Party[i]
		// A HIRED MAN IS NOT REDRESSED AS A HERO HERE (owner). This walk is the
		// player character's appearance law and it ran over every party member, so
		// it restored the composed hero sheet at every mission open even for a
		// member the tavern had built correctly. He is drawn from his own class
		// record; see buildMercenarySquad.
		if member.Hired() {
			continue
		}
		slots, ok := ms.World.Equipped(ms.Start.IDs[i])
		if !ok {
			continue
		}
		eq := equipmentFromSlots(slots)
		if occupied, _ := eq.Occupied(1); !occupied && member.Weapon != nil && !member.WeaponMaterialized {
			eq.SetCode(1, member.Weapon.Code)
		}
		body, dir, class, matched := data.HeroAppearance(list, eq, member.Mage, false)
		if !matched {
			continue
		}
		ms.Party[i].Body, ms.Party[i].BodyDir, ms.Party[i].Class = string(body), dir, class
		LoadHeroBody(src, units, dir, body)
	}
}

// partyArt is the class each party member is DRAWN AS, keyed by the entity
// id the start minted for him.
//
// IT IS partyCharacters' TWIN and pairs the same two slices in the same order,
// for the same reason: the ids come from the start, the body name from the
// member the mission was started with, and recomputing an id here would be this
// package holding a copy of a rule that lives in the loader.
//
// A MEMBER WITH NO BODY, AND A BODY THE BUNDLE DID NOT RESOLVE, BOTH STATE
// NOTHING — no entry, rather than an entry holding nil. The two are the same
// answer downstream, and the difference matters for what this map MEANS: it
// holds the members whose picture was actually resolved, so its length is
// readable as that count rather than as the party's size.
//
// The two slices are walked to the SHORTER of the two, partyCharacters' own
// rule, so a Start from before ids were recorded pairs nothing rather than
// panicking.
func partyArt(ms *Mission, units *terrain.UnitSet) map[sim.EntityID]*terrain.UnitClass {
	if units == nil {
		return nil
	}
	n := len(ms.Party)
	if len(ms.Start.IDs) < n {
		n = len(ms.Start.IDs)
	}
	var out map[sim.EntityID]*terrain.UnitClass
	for i := 0; i < n; i++ {
		member := ms.Party[i]
		body := units.Bodies[data.HeroBodyKey(member.BodyDir, data.HeroBody(member.Body))]
		if body == nil {
			continue
		}
		if out == nil {
			out = make(map[sim.EntityID]*terrain.UnitClass, n)
		}
		out[ms.Start.IDs[i]] = body
	}
	return out
}

// missionAppearanceArt keeps party art and resolves placed humans: a Hero type
// is drawn from worn equipment on the PC sheet; any other type keeps the class
// art of its Humans row.
// The retained roster body and archived dead types preserve corpse art on LOAD.
func missionAppearanceArt(ms *Mission, units *terrain.UnitSet, list data.BodyList, src terrain.EntrySource) map[sim.EntityID]*terrain.UnitClass {
	out := partyArt(ms, units)
	if ms == nil || ms.World == nil || units == nil {
		return out
	}
	for id, member := range ms.Start.Roster {
		if slices.Contains(ms.Start.IDs, id) {
			continue
		}
		if e, ok := ms.World.Entity(id); ok && e.Decay >= sim.DecayBones {
			if class, matched := data.HeroBodyClass(data.HeroBody(member.Body)); matched {
				if resolved := rosterBodyArt(units, src, e.TypeID, member.BodyDir, data.HeroBody(member.Body), class); resolved != nil {
					if out == nil {
						out = make(map[sim.EntityID]*terrain.UnitClass)
					}
					out[id] = resolved
					continue
				}
			}
		}
		if e, held := ms.World.Entity(id); held && !data.FigureIsHero(e.TypeID) {
			// The entity's type id is the Humans-row class (ANIM-106); the roster
			// class is a placement key after a SAV load.
			if resolved := units.Classes[e.TypeID]; resolved != nil {
				if out == nil {
					out = make(map[sim.EntityID]*terrain.UnitClass)
				}
				out[id] = resolved
			}
			continue
		}
		slots, ok := ms.World.Equipped(id)
		if !ok {
			continue
		}
		body, dir, class, matched := data.HeroAppearance(list, equipmentFromSlots(slots), member.Mage, false)
		if !matched {
			continue
		}
		e, _ := ms.World.Entity(id)
		resolved := rosterBodyArt(units, src, e.TypeID, dir, body, class)
		if resolved == nil {
			continue
		}
		if out == nil {
			out = make(map[sim.EntityID]*terrain.UnitClass)
		}
		out[id] = resolved
		member.Body, member.BodyDir = string(body), dir
		ms.Start.Roster[id] = member
	}
	for id, typeID := range ms.DeadArt {
		class := int32(uint8(typeID))
		if class >= 0x20 && class < 0x40 {
			// HERO-APPEAR-041/042: the wire hero axes select a dying body;
			// they are not a units.reg class index.
			body := data.HeroBodyName(data.BodyUnarmed, false, (class-0x21)&2 != 0, true)
			class, _ = data.HeroBodyClass(body)
		}
		resolved := units.Classes[class]
		if resolved == nil {
			continue
		}
		if out == nil {
			out = make(map[sim.EntityID]*terrain.UnitClass)
		}
		out[id] = resolved
	}
	return out
}

// rosterBodyArt is the art of a placed person drawn from worn equipment. A rider
// keeps his own mounted class record: the equipment body class names a foot
// sheet (UNIT-APPEAR-030, HERO-DOLL-078).
func rosterBodyArt(units *terrain.UnitSet, src terrain.EntrySource, typeID int32, dir string, body data.HeroBody, class int32) *terrain.UnitClass {
	if data.FigureHasHorse(typeID) && !data.FigureIsHero(typeID) {
		if own := units.Classes[typeID]; own != nil {
			return own
		}
	}
	if data.FigureIsHero(typeID) {
		LoadHeroBody(src, units, dir, body)
		if pc := units.Bodies[data.HeroBodyKey(dir, body)]; pc != nil {
			return pc
		}
	}
	return units.Classes[class]
}

// settleNotices decides, for the step that has just run, what the player
// should be looking at.
//
// IT IS CALLED ONCE PER sim.Step AND FROM tick() ALONE. Not at the frame
// seam: one frame advances up to a catch-up bound of whole ticks — some
// sixteen script passes at the ladder's fast end — and a repeating trigger
// that fires and then does not inside one such call has its latch cleared
// before a frame-level sampler could look. That raise would not be merged;
// it would be lost.
//
// THE OUTCOME IS LATCHED FIRST AND SHOWN LAST. A dialogue raised by the same
// trigger therefore owns the screen until its final page is dismissed; only
// then does the already-decided outcome appear. A decision with no readable
// dialogue still appears on the deciding step.
//
// THE DISCARD APPLIES TO RAISES AND NOTHING ELSE. An outcome transition is never
// dropped: it latches in the simulation and would never be offered again, so
// dropping it would lose the banner permanently and with it the only way out of
// the mission.
//
// Sample is called on EVERY step whatever is open, because it is what advances
// the announcer's memory: skipping it while a notice was up would leave a rising
// edge unread and raise it later, which is a deferral and not a discard.
//
// NOTHING HERE REACHES THE WORLD. It reads the outcome and the latch array,
// reads an archive entry, and writes to the viewer — so a run that shows
// notices and a run that does not have the same ticks and the same digests.
func (mw *mapWorld) settleNotices() {
	m := mw.mission
	if m == nil {
		return
	}
	if !m.announced || m.outcome == sim.OutcomeWon {
		// THE CHARACTERS ARE TESTED FIRST, AHEAD OF Outcome() ITSELF. The engine's
		// own reporter tests a loss before a win (see Outcome() below), and this
		// test placed after it would let a winning trigger latch on the very step
		// a character died — so this is not a rule beside the engine's two, it
		// is that same three-test order carried one test further up, ahead of both
		// counters.
		//
		// A FALLEN CHARACTER WHO CAN STILL BE HEALED IS NOT LOST. The guarded
		// set is the party, and a member's fall defers the loss while his body
		// dwells or Heal can still raise it. A blow or the corpse walk to -10
		// ends that window; a body at exactly 0 never decays (HERO-ZERO-070).
		// The original loses on the primary's fall instead: DIV-219.
		//
		// ONE DEAD CHARACTER IS ENOUGH and the walk stops at it. Which one it
		// was is not recorded: the notice says the same words whoever fell, and
		// a field naming him would be state nothing reads.
		o := sim.OutcomeUndecided
		if mw.guardedCharacterLost() {
			o = sim.OutcomeLost
		}
		// THE OUTCOME COMES FROM Outcome() AND NEVER FROM THE COUNTERS, when the
		// character test above left the question open. The counters only rise,
		// nothing clears them inside a mission, and one repeating check drives the
		// lose counter up without bound — so `> 0` is the wrong predicate and
		// nothing above pkg/sim re-derives an outcome.
		if o == sim.OutcomeUndecided {
			o = mw.world.Outcome()
		}
		if o != sim.OutcomeUndecided {
			if m.outcome != o && m.outcomeShown {
				mw.closeNotice()
				m.outcomeShown, m.delayedVictory = false, false
			}
			m.announced, m.outcome = true, o
		}
	}

	raised := m.ann.Sample(mw.world)
	if m.campaign().settleNotices(mw) {
		return
	}
	if m.open {
		return
	}
	for _, e := range raised {
		// MESSAGE 255 IS RESERVED AND IS NOT AN EVENT TEXT. The original's
		// mission-lost arm posts the same window message with the same value, and
		// the handler compares the number against that sentinel before it composes
		// any address, so a script raising 255 gets the lose panel and no file is
		// ever opened for it. The shipped campaign raises 0..25, so nothing that
		// ships reaches this.
		if e == ReservedMessageNumber {
			m.announced, m.outcome = true, sim.OutcomeLost
			continue
		}
		// The FIRST raise that yields text takes the screen and every later one
		// on the same step meets an open notice — which is the discard rule
		// applied to a step that raised several, not a second rule beside it.
		if mw.openDialogue(int(e)) {
			return
		}
	}
	if m.announced && !m.outcomeShown {
		mw.showOutcome()
	}
}

func (mw *mapWorld) guardedCharacterLost() bool {
	if mw.mission != nil {
		for _, id := range mw.mission.guarded {
			e, ok := mw.entity(id)
			if ok && e.Owner != sim.SelfSlot {
				if mw.mission.departed == nil {
					mw.mission.departed = make(map[sim.EntityID]bool)
				}
				mw.mission.departed[id] = true
				continue
			}
			if ok && mw.mission.departed[id] && !sim.InPersistBand(e.TypeID) {
				continue // reused id, not a genuine returned party actor: exemption stands
			}
			if ok {
				delete(mw.mission.departed, id)
			}
			if mw.mission.departed[id] {
				continue
			}
			if !ok || (!e.Alive() && !e.Dying() && !e.Restorable()) {
				return true
			}
		}
	}
	return false
}

func (mw *mapWorld) showOutcome() {
	m := mw.mission
	kind := ui.NoticeFailure
	if m.outcome == sim.OutcomeWon {
		kind = ui.NoticeSuccess
	}
	m.open, m.kind, m.payload, m.part, m.portrait = true, kind, nil, 0, false
	m.outcomeShown = true
	body := missionOutcomeText(mw.view, m.outcome)
	body = m.campaign().outcomeText(mw, body)
	mw.view.SetNotice(body, kind)
	// VIDEO-SFX-016 places the fixed cue after the outcome child is created.
	// SetNotice above is this front end's child creation boundary.
	if m.outcome == sim.OutcomeLost {
		mw.view.PlayUISound(ui.UISoundMissionFailed)
	} else {
		mw.view.PlayUISound(ui.UISoundMissionComplete)
	}
}

// openDialogue reads the event text an announcement names and shows its part
// 1, and reports whether there was anything to show.
//
// A FILE THAT DOES NOT SHIP PRODUCES NOTHING AT ALL — no notice, no placeholder,
// no message, no error and no change to anything else — and neither does one
// that ships and yields no part 1. Of the 242 numbers the shipped campaign
// scripts raise, 19 name no file, so silence is the authored behaviour of about
// one raise in thirteen and not a defect to report.
//
// The read happens HERE, at the moment the announcement fires, and nothing
// reads any event text when a map is loaded.
func (mw *mapWorld) openDialogue(event int) bool {
	m := mw.mission
	payload, ok := ReadEventTextFor(m.src, tableGame(m.table), m.number, event)
	if !ok {
		return false
	}
	audience := mw.eventAudience(event)
	body, ok := dialoguePart(payload, 1, audience)
	if !ok {
		return false
	}
	m.open, m.kind, m.payload, m.part = true, ui.NoticeDialogue, payload, 1
	m.dialogueAudience = audience
	m.tips = 0
	m.noteTips(1)
	// THE SHAPE IS SETTLED HERE, ONCE, from the whole file — which is the one
	// place it can be settled from, because that is what the test is over. It
	// is then carried across every page of this window and re-derived at none
	// of them.
	m.portrait = EventHasSpeaker(dialoguePayload(payload))
	mw.view.SetDialogue(m.dialogueOf(body, 1))
	return true
}

// advanceNotice is the seam the front-end's three dismiss inputs reach —
// the ui.MapAdvance the map screen holds.
//
// IT IS THE ONLY PLACE A NOTICE CLOSES. A dialogue notice PAGES: part n+1 if the
// file has one, and closed if it does not. An outcome notice ENDS THE MISSION,
// and where it leads is where this story stops — a lost mission to the main
// menu, a won one to the map list carrying the sentence that says the town is
// not built.
//
// It reads no clock and pushes nothing into the simulation. The part number is
// this driver's and the payload is the one already read, so paging opens no
// archive entry and cannot answer differently the second time.
//
// NOTHING HERE PRODUCES A SUCCESSOR (0131 T2's own limit). Every return above
// hands back a nil ui.MapOpener: this method does not read a campaign and
// does not know one exists. It is the mechanical half of ui.MapAdvance's
// widened shape alone — the third result exists so this method keeps
// compiling against the seam continuity wraps it in — and every answer it
// gives is exactly the one it gave before this story. Reading a mission's own
// successor and turning it into a NoticeToMission is T3's, at the tier that
// holds the campaign.
func (mw *mapWorld) advanceNotice(actions ...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) {
	m := mw.mission
	if m == nil {
		return ui.NoticeStay, "", nil
	}
	action := ui.NoticeAdvance
	if len(actions) > 0 {
		action = actions[0]
	}
	// End Quest's later Victory reaches this same one-shot transition after
	// Continue has closed the panel. No other action is admitted while closed.
	if !m.open {
		if action == ui.NoticeVictory && m.delayedVictory && !m.victoryTaken && m.outcome == sim.OutcomeWon {
			m.delayedVictory, m.victoryTaken = false, true
			return ui.NoticeToMapList, TownNotBuiltMessage, nil
		}
		return ui.NoticeStay, "", nil
	}
	if m.kind == ui.NoticeSuccess {
		if action == ui.NoticeAdvance {
			action = ui.NoticeVictory // the initially focused first control
		}
		switch action {
		case ui.NoticeContinue:
			mw.closeNotice()
			m.delayedVictory = true
			return ui.NoticeStay, "", nil
		case ui.NoticeVictory:
			mw.closeNotice()
			if m.victoryTaken {
				return ui.NoticeStay, "", nil
			}
			m.delayedVictory, m.victoryTaken = false, true
			return ui.NoticeToMapList, TownNotBuiltMessage, nil
		default:
			return ui.NoticeStay, "", nil
		}
	}
	if m.kind == ui.NoticeFailure {
		switch action {
		case ui.NoticeLoadGame:
			return ui.NoticeToLoad, "", nil
		case ui.NoticeAdvance, ui.NoticeExitMain:
			mw.closeNotice()
			return ui.NoticeToMenu, "", nil
		default:
			return ui.NoticeStay, "", nil
		}
	}
	if m.kind == ui.NoticeOutcome {
		mw.closeNotice()
		if m.outcome == sim.OutcomeLost {
			return ui.NoticeToMenu, "", nil
		}
		return ui.NoticeToMapList, TownNotBuiltMessage, nil
	}
	m.part++
	// A NOTICE THIS TIER PAGED ITSELF comes first, because it carries no event
	// payload and EventPart would answer "no next part" for it on the first
	// dismiss — which is exactly how the load disclosure came to be one
	// screen long (1032 return 1).
	if len(m.pages) > 0 {
		if m.part <= len(m.pages) {
			mw.view.PageDialogue(ui.Dialogue{Text: m.pages[m.part-1]})
			return ui.NoticeStay, "", nil
		}
	} else if body, ok := dialoguePart(m.payload, m.part, m.dialogueAudience); ok {
		m.noteTips(m.part)
		mw.view.PageDialogue(m.dialogueOf(body, m.part))
		return ui.NoticeStay, "", nil
	}
	mw.raiseMissionTip()
	if m.campaign().acknowledgeNotice(mw) {
		return ui.NoticeStay, "", nil
	}
	if m.announced && !m.outcomeShown {
		mw.showOutcome()
		return ui.NoticeStay, "", nil
	}
	mw.closeNotice()
	return ui.NoticeStay, "", nil
}

// noteTips keeps the `tips=` value of the part just shown, as the dialogue
// panel keeps the last one its parser stored (TRIG-TIPS-087).
func (m *missionNotices) noteTips(part int) {
	if n, ok := dialoguePartTips(m.payload, part, m.dialogueAudience); ok {
		m.tips = n
	}
}

// raiseMissionTip is the dialogue's close on its last page: a nonzero stored
// value shows that tip in the mission popup while TipsMode is set, replacing
// an open one (TRIG-TIPS-087). The ROM2 dialogue is outside the claim.
func (mw *mapWorld) raiseMissionTip() {
	m := mw.mission
	n := m.tips
	m.tips = 0
	if n == 0 || m.payload == nil || !mw.view.MissionTipsMode() {
		return
	}
	path, ok := MissionTipPath(m.edition(), m.number, n)
	if !ok {
		return
	}
	text, ok := ReadShopTip(m.src, path, InstallTextCode(m.src, m.edition()))
	if !ok {
		return
	}
	mw.view.ShowMissionTip(text)
}

// closeNotice drops what the driver was holding and tells the viewer to stop
// drawing it, in that order and in one place, so the two cannot come to disagree
// about whether a notice is up.
func (mw *mapWorld) closeNotice() {
	m := mw.mission
	m.open, m.payload, m.part, m.portrait, m.tips = false, nil, 0, false, 0
	m.pages = nil
	m.dialogueAudience = EventAudience{}
	mw.view.ClearNotice()
}

// walkClock is one entity's walk state: the distance it has walked since it
// last stood still, in 1/256 of a cell, and which tick of its current crossing
// has just run, counted from 0.
//
// THE TICK IS COUNTED AND NOT DERIVED FROM TransitTotal. That total outlives
// the transit it measured — pkg/sim writes the pair only for a mover its
// rate law rates — so an entity that stops being rated keeps the previous
// crossing's total while crossing a cell per tick, and an index derived from
// it would read the last tick of a long crossing and pay a whole cell the
// last share of one. Transit cannot go stale that way: it is zero whenever
// nothing is owed, so tick + 1 + Transit is the crossing's own length at
// every tick of it, and TransitTotal is never read here at all.
type walkClock struct {
	dist int
	tick int
}

// maxCatchUpMicros bounds ONE paced call: the most WORLD TIME a single
// front-end call may run, however long the front-end was away.
//
// A stall — a minimised window, a breakpoint, a slow first frame — leaves the
// ticker holding whole seconds of elapsed time, and running all of it would
// teleport every walking entity across the map in one frame. Past the bound the
// time is DROPPED, never queued for later, so a stall costs world time once
// instead of a lurch that lasts as long as the stall did.
//
// It is a SPAN and no longer a fixed count of four, which was a rate ceiling
// wearing a stall bound's name: four ticks a call at a 60-per-second front-end
// is 240 ticks a second whatever period is selected, so every rate this story
// exists to reach was unreachable while it stood. A span converts to a different
// count at every period and to the SAME span at all of them.
const maxCatchUpMicros = 250_000

// maxCatchUp is how many ticks one paced call may run at the period in force.
//
// It is DERIVED AT THE CALL from the period the clock is holding and stored
// nowhere, so there is no second value to fall out of step with the rate: 4 at
// the map-load period — the shipped number re-derived rather than moved — 64 at
// 256 ticks a second, and 1 at rate 1, where the bound is shorter than a single
// tick and a floor of zero would be a stop nobody asked for.
func maxCatchUp(periodUS int) int {
	if n := maxCatchUpMicros / periodUS; n > 1 {
		return n
	}
	return 1
}

func newMapWorld(w *sim.World, sched [][]sim.Command, units *terrain.UnitSet, v *ui.Viewer) *mapWorld {
	return newMapWorldWith(w, sched, units, nil, nil, nil, v)
}

// newMapWorldWith is newMapWorld with the per-placement tier lookup beside
// the bundle: the tier each entity's art is drawn in, keyed by the id the
// world builder minted.
//
// tiers is nil for a caller that has none — every hand-assembled front-end, and
// every map opened with no definition table — and an id with no entry states no
// tier. It is ADOPTED and never written: this constructor is handed a lookup
// somebody else resolved, and nothing in this file adds to it, so the tier of a
// placement is fixed at the moment the map opened.
//
// chars is the same shape and the same rule, one story later, and it is a
// CONSTRUCTOR ARGUMENT rather than something installed afterwards for a
// reason worth stating: the tick-0 push below is the only push a mission
// that is opened and left stopped ever gets, so a lookup arriving after
// construction would leave that mission's panel stating no character until
// its world ran.
//
// art is the third of the shape, two stories later, and it is a constructor
// argument for the SAME reason with the stakes raised: a picture installed
// after construction would leave a mission opened and left stopped drawing
// its party member as the class key its record names, which is the one
// drawing this argument exists to replace. A nil one overrides nobody, so
// the two constructors above and every caller holding no party keep exactly
// the screen they had.
func newMapWorldWith(w *sim.World, sched [][]sim.Command, units *terrain.UnitSet,
	tiers map[sim.EntityID]int, chars map[sim.EntityID]ui.UnitCharacter,
	art map[sim.EntityID]*terrain.UnitClass, v *ui.Viewer) *mapWorld {
	mw := &mapWorld{world: w, sched: sched, units: units, tiers: tiers, chars: chars,
		art: art, view: v,

		// The portrait cache is built HERE and not beside the pack's own,
		// which is built only for a mission that has a party: a portrait is
		// drawn for whatever is SELECTED, and every map has selectable actors
		// whether or not it has a party (portrait.go).
		portraits:  make(map[string]*image.RGBA),
		figurePics: make(map[figureCacheKey]*image.RGBA),

		prev:        make(map[sim.EntityID]image.Point),
		died:        make(map[sim.EntityID]int),
		hurt:        make(map[sim.EntityID]int),
		swing:       make(map[sim.EntityID]int),
		phase:       make(map[sim.EntityID]sim.AttackPhase),
		castRun:     make(map[sim.EntityID]castRun),
		commanded:   make(map[sim.EntityID]bool),
		ownerFrames: make(map[ownerFrameKey]*terrain.StaticFrame),
		walk:        make(map[sim.EntityID]walkClock),
		scene:       int(w.Tick()),
		// The cadence starts at the speed index map load selects — the water
		// layer's own default — and its first paced call only takes the
		// baseline, so opening a map fires no tick from the time before it was
		// open.
		clock: terrain.NewTicker(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex))}
	// THE FOG PLANE IS BUILT AND REFRESHED HERE, BEFORE THE TICK-0 PUSH BELOW:
	// every cell starts unseen and this one refresh already lights
	// sim.SelfSlot's own surroundings, so the screen a mission opens on is
	// closed by construction — the push a few lines down is what a developer
	// would expect to be the first opportunity to see anything, and by the time
	// it runs the plane already holds the party's opening view rather than the
	// fresh-allocated all-unseen one.
	//
	// Sized from the WORLD's own bounds and never the render grid's: the
	// plane indexes the same cells w.Sight's stamp does, and the two must
	// agree without either side converting.
	b := w.Bounds()
	mw.fog = newFogPlane(int(b.Width), int(b.Height))
	mw.fog.refresh(w, sim.SelfSlot)
	// The first push uses the restored world's tick, or zero for a fresh map.
	mw.push()
	// The readout's own tick-0 push, beside it and for its reason: a map that
	// has been opened and not yet ticked already states the cadence it opened
	// at and the digest of the state it starts from, rather than a box of zeros
	// until the first frame lands.
	mw.pushReadout()
	// WHICH ROSTER SLOT THE LOCAL PARTICIPANT HOLDS, and since 0094 it is
	// answered rather than declined.
	//
	// It is written HERE, once, on the statement that builds the world, rather
	// than left at the viewer's zero value — for pushReadout's own reason. A
	// value that is never written is indistinguishable from one that is written
	// and happens to be zero, and this is the one line a session join changes; a
	// viewer left at its zero value would need somebody to notice that the line
	// was missing.
	//
	// IT WAS A ZERO UNTIL 0094 and the reason it stopped being one is not that the
	// question got easier. This build's mission start puts the party on
	// sim.SelfSlot, so on the campaign path the answer is no longer the map's to
	// invent — it is the start's, already made, and this line reads it rather than
	// deriving anything. A session join replaces the source without touching the
	// consumer.
	//
	// AND LEAVING THE ZERO WOULD NOT HAVE BEEN NEUTRAL, which is what settled it.
	// The damage-numeral rule asks `e.Owner != localOwner` to decide which way a
	// figure drifts, and that comparison came out "mine" for a party member only
	// because BOTH sides were zero. Move one and every figure over the player's
	// own units flips. The other consumer is the arming gate, which its own
	// comment says goes live the day a nonzero value is pushed "with no other
	// change anywhere": a player may now arm an attack with his own units
	// selected and not with somebody else's.
	v.SetLocalOwner(sim.SelfSlot)
	// WHAT A STEP ONTO A CELL WOULD COST A UNIT, handed over as the question
	// and not as an answer.
	//
	// It is installed HERE, on the statement that builds the world, for
	// pushReadout's own reason one line up: a mission opened and left stopped gets
	// no further push, and a question installed on the frame path would leave such
	// a mission's box stating an absence until its world ran.
	//
	// The closure dereferences mw at the CALL and captures no world, so a driver
	// that comes to open a second map answers about the map it is holding rather
	// than about the one that was open when this line ran.
	//
	// It composes nothing and interprets nothing: the world's own read decides
	// what is answerable, and this converts widths. THAT ORDER MATTERS — every
	// refusal is the simulation's, so the front-end cannot acquire a rule of its
	// own about which movers have a rate.
	v.SetStepCost(func(id uint32, col, row int) (ui.StepCost, bool) {
		rate, transit, adjacent, ok := mw.world.StepRate(sim.EntityID(id), int32(col), int32(row))
		if !ok {
			return ui.StepCost{}, false
		}
		return ui.StepCost{Rate: int(rate), Transit: int(transit), Adjacent: adjacent}, true
	})
	return mw
}

// openDifficulty is the standalone diagnostic map viewer's identity setting.
// Campaign mission doors use FrontEnd.Difficulty, selected during creation.
const openDifficulty = mapload.DifficultyNormal

// openMapWorld builds the world under one opened map: one entity per placed
// unit and that map's own placeholder schedule, both from THE MAP ALREADY
// DECODED and not from a second decode of the same bytes.
//
// It resolves every placement against t, so two things the table holds reach the
// running game together: a unit's MOVEMENT DOMAIN, and the HEALTH its own class
// entry names in place of the one provisional constant every placement carried
// before. A nil t resolves nothing and yields exactly the world this package
// built before it took a table at all.
//
// A table the loader refuses ends the map open, wrapped by the caller with the
// map's name. It does NOT fall back to a table-free build: that would answer a
// broken table with the all-ground, one-health world this seam exists to
// remove, and it would do it silently.
//
// Both derivations come from mapload, which owns the id convention and the
// one unit-record-to-cell conversion they share.
//
// units is the unit-art bundle every push under this map resolves against.
// It is a parameter and not a lookup because the loader that owns one is the
// front-end, loaded once at startup; the map itself contributes nothing to
// it, so a world stays a function of the decoded map alone whatever the
// bundle holds.
func openMapWorld(m *alm.Map, t *mapload.Table, units *terrain.UnitSet, v *ui.Viewer) (*mapWorld, error) {
	w, err := mapload.FromALMWith(m, t, openDifficulty)
	if err != nil {
		return nil, err
	}
	// The tier lookup comes off THE SAME MAP AND THE SAME TABLE the world was
	// just built from, in this one statement, which is what keeps its keys and
	// that world's ids the same numbers rather than two derivations that agree.
	// A nil table resolves no tier, exactly as it resolves no health and no
	// domain. A plain map places no party, but it still resolves a character
	// for every placement that resolves to a definition entry: the picker's own
	// path is the map screen it always was, with each unit's eight numbers, and
	// now its sheet, added and nothing else. Before this story this line passed
	// no lookup at all and the picker's every unit stated no character;
	// placedCharacters below is the map-loading tier's own PlacedSheets,
	// converted, and does not read ms — there is no mission here to read one
	// from. NO SCHEDULE — see openMission, where the same removal is
	// explained. THIS PATH COMPILES NO SCRIPT and advances under the AI alone
	// — not "its own script", which was never true of it and is corrected
	// here. A picker row naming a campaign mission does not reach this function
	// at all; it opens through MissionOpener, which does compile one.
	mw := newMapWorldWith(w, nil, units, entityTiers(m, t), placedCharacters(m, t, placedEntities(m, nil, nil, nil)), nil, v)
	mw.figures = entityFigures(m, t)
	// THE BOOK'S OWN NAMES, off the SAME table t already resolved a spell
	// column from: resolved once, here, and never again — set directly on the
	// struct rather than through the constructor, on setSwingSound's own
	// reasoning one call below (frontend.go): widening newMapWorldWith for one
	// value only this function and openMission ever supply would touch every
	// one of the package's own tests that build a mapWorld through it directly.
	mw.spellNames = spellNamesFrom(t, v.Words().SpellBookNames)
	return mw, nil
}

// paced is the advance the front-end holds — ui.MapTick, called exactly
// once per map-screen tick as 0020 requires and unchanged in signature, so
// the window tier still names nothing and still counts one call per tick.
// What it runs behind that seam is this file's business: the wall clock is
// read HERE, on the one line of the cadence a test cannot drive.
//
// IT ALSO REFRESHES THE PACK, on every call and not only on one that fires a
// tick: the pack can change on a key press that carries no tick of its own
// (grab, below), so "per frame" and not "per tick" is the seam the refresh
// rides. refreshPack's own compare-then-skip is what keeps that
// unconditional call cheap.
//
// UNEQUIPFROMWORN RUNS BESIDE IT, on the same reasoning (0151, defect 4): a
// worn-box double-click's request must reach mw.pending before this call's
// own paceTo, or it too would sit a frame before an advance ever saw it.
//
// REFRESHEQUIPMENT RUNS AFTER, beside refreshPack: both read the state an
// equip command already reached through this same call's own paceTo, so both
// belong after it and neither before.
func (mw *mapWorld) paced() {
	mw.frameAdvance(func() { mw.paceTo(time.Now()) })
}

// deterministicFrame replaces only the physical clock adapter: one running
// ui.App frame is one simulation tick and a stopped frame is none.
func (mw *mapWorld) deterministicFrame() {
	mw.frameAdvance(func() {
		if !mw.stopped {
			mw.tick()
		}
		mw.view.SetPhase(0, mw.clock.Period())
	})
}

func (mw *mapWorld) frameAdvance(advance func()) {
	mw.refreshInventorySelection()
	mw.equipFromPack()
	mw.unequipFromWorn()
	mw.unequipFromDoll()
	mw.dropFromInventory()
	mw.dropGoldFromPurse()
	advance()
	mw.refreshPack()
	mw.refreshEquipment()
	mw.refreshAppearance()
	mw.refreshDollDrag()
	// Selection and quick keys still work while paused. The simulation push
	// alone cannot refresh their book or portrait on a frame with no tick.
	mw.pushSpellbook()
	mw.pushPortrait()
}

// refreshInventorySelection switches the inventory subject when exactly one
// persistent party character is selected. Each subject is rebuilt from that
// member and that entity's own stock; no inventory state is copied.
func (mw *mapWorld) refreshInventorySelection() {
	if mw == nil || mw.mission == nil || mw.view == nil {
		return
	}
	selected, ok := mw.view.SelectedUnit()
	if !ok || mw.invSubjectSet && mw.invSubject.ID == selected {
		return
	}
	mw.switchInventorySubject(selected)
}

// switchInventorySubject changes presentation only. Every value is read from
// the selected member and entity; no party member, entity, container,
// equipment record, experience or skill is written. The three change trackers
// are seeded to the selected entity's current values so the following paced
// refresh cannot misread a subject change as an equipment or skill change.
func (mw *mapWorld) switchInventorySubject(selected uint32) {
	if mw == nil || mw.mission == nil || mw.world == nil || mw.view == nil {
		return
	}
	index := -1
	for i, id := range mw.mission.ids {
		if uint32(id) == selected && i < len(mw.mission.party) {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	member := mw.mission.party[index]
	// MERC-TYPE-001
	if member.Hired() || !data.ComposesFigure(member.Class) {
		return
	}
	// World IS SET HERE (round-2 adversarial review, twelfth pass, C1's own
	// second correction): missionDollEquipment now prefers a live
	// w.Equipped(id) read over the party record, and this throwaway Mission is
	// buildInventorySubject's only route to one for a member switch. Left nil,
	// the switched-to member's doll would fall back to member.Worn/Carry
	// exactly as the pre-fix code did, stale for any member whose live
	// equipment has changed during THIS mission (a companion re-armed by the
	// player, then switched away from and back).
	one := &Mission{Party: []mapload.PartyMember{member}, Start: mapload.Start{IDs: []sim.EntityID{sim.EntityID(selected)}}, World: mw.world}
	subject, _ := buildInventorySubject(mw.mission.src, one)
	stacks, _ := mw.world.CarriedStacks(sim.EntityID(selected))
	if mw.invIconCache == nil {
		mw.invIconCache = make(map[uint16]*image.RGBA)
	}
	subject.Pack, subject.PackCount, _ = buildInventoryPack(mw.mission.src, stacks, mw.invIconCache)
	subject.PackStars = itemStackStarFlags(stacks)
	figureDir, figureFace := memberFigure(member)
	mw.invParty.hero, mw.invParty.profile, mw.invParty.startWeapon = member.Hero, member.Profile, mapload.MemberWeapon(member, mw.invParty.table)
	mw.invParty.class, mw.invParty.hired = member.Class, member.Hired()
	mw.invParty.hiredRotationSpeed = member.HiredRotationSpeed
	mw.invParty.figureDir, mw.invParty.figureFace = figureDir, figureFace
	mw.invParty.figureHero = memberFigureID(member).Hero
	mw.invParty.mage = member.Mage
	mw.invParty.primary = primaryPlayerHero(member)
	mw.invSubject, mw.invSubjectSet, mw.invCodes = subject, true, stacks
	// eq IS THE RAW ARRAY, read for invEquipment — rearm's own tracker, which
	// must stay narrow (its own doc on currentFigureEquipment). SlotInfo,
	// invFigureEquipment AND bodyEquipment go through currentFigureEquipment
	// instead (counterexample 2, and round-2 adversarial review, fifth pass,
	// counterexample I for bodyEquipment specifically), matching the subject
	// figure buildInventorySubject just composed with member.Weapon's own
	// fallback.
	eq := mw.currentEquipment()
	// Seed this subject's history before resolving its display fallback.
	// Reading currentFigureEquipment with the previous subject's sticky flag
	// would make a selection change inherit the previous member's slot-1
	// history. resolveWeaponMaterialized also folds in this member's own
	// PERSISTED history (round-2 adversarial review, fifth pass), the same
	// reason openMission's own seed reads it.
	mw.invWeaponEverEquipped = mw.resolveWeaponMaterialized(&mw.mission.party[index], eq)
	figureEq := mw.currentFigureEquipment()
	figureItems := mw.currentFigureEquipmentItems()
	mw.invSubject.SlotInfo = slotItemInfoLinesWithWeaponDamage(figureItems, mw.invParty.table, mw.itemWeaponDamage, mw.view.Words())
	// WeaponFallback (counterexample 4, round-2 adversarial review, third
	// pass): mw.invParty was reassigned above in this function, so
	// currentWeaponFallbackActive's read of mw.invParty.startWeapon reflects
	// the NEW subject, matching figureEq just resolved one line up.
	mw.invSubject.WeaponFallback = mw.currentWeaponFallbackActive()
	mw.invSubject.PackInfo = packInfoLinesWithWeaponDamage(stacks, mw.invParty.table, mw.itemWeaponDamage, mw.view.Words())
	if mw.invParty.primary {
		appendInventoryGold(&mw.invSubject, mw.previewPurse(), loadInventoryGoldIcon(mw.mission.src))
	}
	mw.invEquipment, mw.invFigureEquipment, mw.bodyEquipment = eq, figureEq, figureEq
	mw.invLayers, mw.invFigureLayers = mw.activeLayers(sim.EntityID(selected)), nil
	mw.invEquipmentItems = mw.currentEquipmentItems()
	mw.invFigureEquipmentItems = figureItems
	// invComposedEquipment (C1b) seeds from the SAME three arguments
	// buildInventorySubject passed missionDollEquipment over member, above,
	// through one (mw.world, sim.EntityID(selected), member) — not from eq
	// or figureEq directly; see openMission's own seed for why this call is
	// kept separate rather than reused, and missionDollEquipment's own doc
	// for why mw.world is now what it reads in preference to member.
	mw.invComposedEquipment, _ = missionDollEquipment(mw.world, sim.EntityID(selected), member)
	if e, ok := mw.entity(sim.EntityID(selected)); ok {
		mw.invSkill, mw.invSkillSet = e.Skill, true
		mw.invTraining = e.NativeTraining
	} else {
		mw.invSkill, mw.invSkillSet = [data.SkillSlots]int32{}, false
		mw.invTraining = sim.NativeTraining{}
	}
	mw.view.SetInventorySubject(mw.invSubject)
}

func (mw *mapWorld) refreshPack() {
	if !mw.invSubjectSet {
		return
	}
	stacks, _ := mw.world.CarriedStacks(sim.EntityID(mw.invSubject.ID))
	var gold uint32
	if mw.invParty.primary {
		gold = mw.previewPurse()
	}
	if itemStacksEqual(stacks, mw.invCodes) && gold == inventoryGold(mw.invSubject) {
		return
	}
	mw.invSubject.Pack, mw.invSubject.PackCount, _ = buildInventoryPack(mw.mission.src, stacks, mw.invIconCache)
	mw.invSubject.PackStars = itemStackStarFlags(stacks)
	// THE POPUP'S OWN PACK TEXT MOVES WITH THE PACK (0151, defect 5), on the
	// SAME stacks read this compares against invCodes — a second container
	// read here would be a second answer to what refreshPack has already
	// asked once.
	mw.invSubject.PackInfo = packInfoLinesWithWeaponDamage(stacks, mw.invParty.table, mw.itemWeaponDamage, mw.view.Words())
	appendInventoryGold(&mw.invSubject, gold, loadInventoryGoldIcon(mw.mission.src))
	mw.invCodes = stacks
	mw.view.SetInventorySubject(mw.invSubject)
}

func itemStacksEqual(a, b []sim.ItemStack) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !sim.StackStateEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func (mw *mapWorld) equipFromPack() {
	idx, ok := mw.view.TakeInventoryEquip()
	if !ok {
		return
	}
	mw.enqueueEquip(idx)
}

func (mw *mapWorld) enqueueEquip(idx int) {
	if !mw.invSubjectSet {
		return
	}
	id := sim.EntityID(mw.invSubject.ID)
	stacks, _ := mw.world.CarriedStacks(id)
	if idx < 0 || idx >= len(stacks) {
		return
	}
	code := data.ItemCode(stacks[idx].Code)
	// THE CAMPAIGN DOCUMENTS ITEM OPENS THE PANEL INSTEAD OF EQUIPPING
	// (ITEM-DOC-053, MENU-DOC-009).
	//
	// IT DOES NOT ALSO EQUIP, and the original's routine does: its other
	// work is a store into the actor's own slot array. DAT-DOC-021
	// establishes that the Humans equipment grammar has no MagicItems arm,
	// so this build has no slot to store it in, and EquipTarget below would
	// refuse the code anyway (data.EquipSlotFor refuses class 14). Inventing
	// a slot for it is the one thing that would make the item worn, which is
	// what the contract's B5 forbids. DIV-297.
	//
	// The early return is what stops the drained request from also reaching
	// the equip questions below with a code none of them can answer for.
	if data.RaisesDocuments(code) {
		mw.view.RaiseDocuments()
		return
	}
	item := stacks[idx].Instance()
	if spell, _, scroll := sim.ScrollSpell(item); scroll {
		mw.view.ArmItemCast(uint32(id), idx, uint32(spell), fmt.Sprintf("%#v", item))
		return
	}
	if item.Kind == 3 {
		mw.pending = append(mw.pending, sim.UsePotion(id, sim.ItemSlot(idx)))
		return
	}
	if _, readable := item.BookSpell(); item.Kind == 5 {
		if !readable || !mw.invParty.mage {
			return
		}
		mw.pending = append(mw.pending, sim.ReadBook(id, sim.ItemSlot(idx)))
		return
	}
	if layer, isLayer := mapload.LayerItem(mw.invParty.table, uint16(code)); isLayer {
		mw.toggleLayer(id, layer)
		return
	}
	slot, ok := EquipTarget(code, mw.invParty.table)
	if !ok {
		return
	}
	if !mw.wearAllows(code) {
		return
	}
	var displace int
	switch slot {
	case 1:
		// Any weapon that is not a resolved one-handed melee weapon frees the
		// shield hand in the same canonical command that wears it.
		if !mapload.WeaponAllowsShield(code, mw.invParty.table) {
			if worn, ok := mw.world.Equipped(id); ok && worn[1] != 0 {
				displace = 2
			}
		}
	case 2:
		worn, ok := mw.world.Equipped(id)
		if !ok {
			return
		}
		switch {
		case worn[0] != 0:
			// A weapon that fills both hands leaves for the pack in the same
			// canonical command that wears the shield.
			if !mapload.WeaponAllowsShield(data.ItemCode(worn[0]), mw.invParty.table) {
				displace = 1
			}
		case mw.currentWeaponFallbackActive() && mw.invParty.startWeapon != nil:
			// The drawn fallback becomes a real item in the same tick.
			// materializeFallbackWeapon first gives it an owned container
			// instance. A weapon that leaves a hand free is worn by the first
			// queued command and the shield command follows it before any frame
			// can observe the pair; any other stays in the pack, where the shield
			// alone is then worn.
			element, materialized := mw.materializeFallbackWeapon()
			if !materialized {
				return
			}
			if mapload.WeaponAllowsShield(mw.invParty.startWeapon.Code, mw.invParty.table) {
				mw.pending = append(mw.pending, sim.Equip(id, sim.ItemSlot(element), 1))
			}
		}
	}
	mw.pending = append(mw.pending, sim.EquipDisplacing(id, sim.ItemSlot(idx), sim.EquipSlot(slot), sim.EquipSlot(displace)))
}

// wearAllows is the wear rule asked of the inventory subject: whether the
// class this subject is may use the item code names.
//
// A TABLE THAT CANNOT SUPPLY THE THREE COLLECTIONS ANSWERS YES, on FR-2a's own
// polarity: a column that was never read is not a refusal. That is the opposite
// leniency from the EquipTarget question above it, which answers NO on a nil
// table — and the two are opposite on purpose. EquipTarget is asked "where does
// this go", and without an answer there is no slot to name in a command; this is
// asked "may he", and without an answer the rule simply does not apply.
func (mw *mapWorld) wearAllows(code data.ItemCode) bool {
	t := mw.invParty.table
	if t == nil {
		return true
	}
	allowed, _ := data.AllowsItem(code, mw.invParty.mage, t.Weapons, t.Shields, t.Armors)
	return allowed
}

// unequipFromWorn is defect 4's own drain (0151), equipFromPack's shape run
// the other way: the viewer's one-shot unequip request, taken and cleared
// in the same call by TakeInventoryUnequip (pkg/ui/inventory.go), so a
// request left undrained this frame cannot surface again next frame under a
// stale slot.
func (mw *mapWorld) unequipFromWorn() {
	idx, ok := mw.view.TakeInventoryUnequip()
	if !ok {
		return
	}
	mw.enqueueUnequip(idx)
}

// unequipFromDoll is unequipFromWorn's own drain, restated for the doll
// figure's own gesture (1005, "the interactive doll", item 3 — press to take
// off — and item 4's drag-doll-to-pack release): the SAME one-shot pattern,
// over the SAME enqueueUnequip this file already states once, so a tap on
// the doll and a double-click on the worn box are two ways to raise the
// identical command rather than two rules that could disagree.
func (mw *mapWorld) unequipFromDoll() {
	idx, ok := mw.view.TakeInventoryDollUnequip()
	if !ok {
		return
	}
	mw.enqueueUnequip(idx)
}

// enqueueUnequip turns idx, the zero-based worn-box slot index a drained
// double-click named, into a sim.Command when a subject is set. idx is
// widened by one into the original's own 1..12 slot numbering — equip.go's
// convention, and KindUnequip's — because Slots, SlotInfo and
// wornSlotRects (pkg/ui) all index a worn cell zero-based while pkg/sim
// counts slots from 1.
//
// UNLIKE enqueueEquip THIS FUNCTION RESOLVES NOTHING AGAINST THE TABLE:
// there is no code to look up and no applicability question to ask, because
// sim.World's own unequip already refuses an out-of-range slot and a slot
// already at the zero code (equip.go) — the same totality enqueueEquip
// leans on rather than duplicates for the equip direction. The guard here
// is IS THERE A SUBJECT AT ALL, enqueueEquip's own first question, because
// idx would otherwise name an entity id (0) that may be a real, unrelated
// actor; the bounds check beside it is defensive, since every idx
// TakeInventoryUnequip can hand back already names one of the twelve worn
// cells.
func (mw *mapWorld) enqueueUnequip(idx int) {
	if !mw.invSubjectSet {
		return
	}
	if idx < 0 || idx >= sim.EquipSlots {
		return
	}
	if idx == 0 {
		if _, ok := mw.materializeFallbackWeapon(); ok {
			return
		}
	}
	id := sim.EntityID(mw.invSubject.ID)
	mw.pending = append(mw.pending, sim.Unequip(id, sim.EquipSlot(idx+1)))
}

// materializeFallbackWeapon turns the subject's DRAWN starting weapon into a
// real unit in his own container and raises the persisted latch, reporting the
// container element it landed in. It answers false — changing nothing — for a
// subject whose slot 1 is a genuine array cell, who has no starting weapon at
// all, or whose latch is already up: in each of those the drawn item is backed
// by something a sim command can already move.
//
// IT WRITES THE CONTAINER AND NOT THE EQUIPMENT ARRAY. Filling slot 1 would
// arm the weapon — Rearm would resolve stats from it and death would drop it
// with the rest of the worn set — and a member drawn with the fallback is
// precisely one the loader deliberately left unarmed (startingLoadout's own
// NPC gate, pkg/mapload/spawn.go; DIV-070). Appending to the container is the
// same effect the shop's own fallback unequip has had since the fifth pass,
// and it leaves every stat where it stood.
//
// THE DOOR IS ReplaceStock, sim's own out-of-Step restore door, already used
// by this package at a mission boundary. The equipment array is read and
// written back unchanged, so the only difference this call makes to the
// world's state is one more complete item in one container. Existing carried
// and equipped instances cross the replacement door unchanged; rebuilding
// either side from its compatibility code projection would erase enchantment.
func (mw *mapWorld) materializeFallbackWeapon() (int, bool) {
	if !mw.invSubjectSet || mw.world == nil || !mw.currentWeaponFallbackActive() {
		return 0, false
	}
	id := sim.EntityID(mw.invSubject.ID)
	items, ok := mw.world.CarriedItems(id)
	if !ok {
		return 0, false
	}
	equipped, ok := mw.world.EquippedItems(id)
	if !ok {
		return 0, false
	}
	code := uint16(mw.invParty.startWeapon.Code)
	grown := append(items, mapload.ItemInstanceFromCode(code, mw.invParty.table))
	mapload.DeclareCodeWeights(mw.world, mw.invParty.table, []uint16{code})
	if !mw.world.ReplaceStock(sim.Stock{ID: id, ItemInstances: grown, EquippedItems: equipped}) {
		return 0, false
	}
	materializeStartingWeapon(mw.missionPartyMember(id))
	mw.invWeaponEverEquipped = true
	stacks, _ := mw.world.CarriedStacks(id)
	for i, stack := range stacks {
		if stack.Code == code && stack.Count > 0 {
			return i, true
		}
	}
	return 0, true
}

// dropFromInventory is 1005 round 2's own drain, equipFromPack's shape
// restated for a release that lands on neither box: the viewer's one-shot
// ground-drop request, taken and cleared in the same call by
// TakeInventoryDrop (pkg/ui/inventory.go), so a request left undrained this
// frame cannot surface again next frame under a stale cell.
func (mw *mapWorld) dropFromInventory() {
	worn, idx, x, y, ok := mw.view.TakeInventoryDrop()
	if !ok {
		return
	}
	mw.enqueueDrop(worn, idx, x, y)
}

// enqueueDrop turns a drained ground-drop request into a sim.Command
// (`ITEM-DROP-008`, `ITEM-CMD-007`'s destination code 3, 1005 round 2).
//
// UNLIKE enqueueEquip THIS FUNCTION RESOLVES NOTHING AGAINST THE TABLE and
// asks no wear rule: dropping an item to the ground is never refused on
// what it is, only on where it came from, and `pkg/sim`'s own
// dropFromContainer/dropFromEquipment already refuse an out-of-range
// source and an empty slot (drop.go) — the same totality enqueueEquip and
// enqueueUnequip lean on rather than duplicate.
//
// worn TELLS THE TWO SOURCES APART, the doll's own equipment slot (index
// widened by one into the 1..12 numbering, enqueueUnequip's own convention)
// against the pack's own container element (idx, CarriedStacks ordering,
// enqueueEquip's own convention) — the same distinction TakeInventoryEquip
// and TakeInventoryDollUnequip already keep as two separate requests,
// merged into one here only because both resolve into a Spell field with
// the same shape: an origin index, not a destination.
//
// x, y ARE THE CELL THE CURSOR NAMED AT RELEASE, in world cell units
// (pkg/ui's own groundCellAt). `pkg/sim`'s own dropToGround (drop.go) applies
// ITEM-DROP-008's Chebyshev window and the bounds fallback against the
// entity's OWN position at application time, so this function states no
// window of its own. It aims the command through sim.DropLanding alone, so
// that a sack which the window would plant beneath a living unit, where it
// cannot be seen, is commanded to a visible cell beside the thrower.
func (mw *mapWorld) enqueueDrop(worn bool, idx int, x, y int32) {
	if !mw.invSubjectSet {
		return
	}
	id := sim.EntityID(mw.invSubject.ID)
	if worn {
		if idx < 0 || idx >= sim.EquipSlots {
			return
		}
		// A DRAWN SLOT 1 BECOMES REAL FIRST, enqueueUnequip's own rule for the
		// same origin: the weapon materializes into the container and the drop
		// then leaves it from there, so one gesture on the doll reaches the
		// ground in one frame rather than needing a second press.
		if idx == 0 {
			if element, ok := mw.materializeFallbackWeapon(); ok {
				mw.pending = append(mw.pending, sim.DropCarried(id, sim.ItemSlot(element), mw.dropAim(id, x, y)))
				return
			}
		}
		mw.pending = append(mw.pending, sim.DropWorn(id, sim.EquipSlot(idx+1), mw.dropAim(id, x, y)))
		return
	}
	if idx < 0 {
		return
	}
	mw.pending = append(mw.pending, sim.DropCarried(id, sim.ItemSlot(idx), mw.dropAim(id, x, y)))
}

// dropAim is the cell enqueueDrop commands a drop at: sim.DropLanding's answer,
// or the release cell itself for a thrower the world does not hold.
func (mw *mapWorld) dropAim(id sim.EntityID, x, y int32) sim.CellPoint {
	at, _ := mw.world.DropLanding(id, x, y)
	return at
}

// currentEquipment is the subject's twelve equipment slots, read off the
// world through sim.World.Equipped and turned into a data.Equipment through
// hero.go's own equipmentFromSlots — the ONE conversion rearm's own tracking
// compare, refreshEquipment's own figure recompose and (since 0134)
// refreshAppearance's own body recompose all build from, so the world's
// array is read one way and not three (docs/0134-what-he-wears T4; plan D-6
// "one conversion serves both").
//
// A SUBJECT THE WORLD NO LONGER HOLDS ANSWERS THE ZERO VALUE, Equipped's own
// reading of an id it does not recognise (pkg/sim/equip.go): every slot
// empty, which is what a subject entity removed from the world (an unusual
// but not impossible state — pkg/sim's decay ladder can remove one) leaves
// rearm, refreshEquipment and refreshAppearance believing about it rather
// than a panic.
func (mw *mapWorld) currentEquipment() data.Equipment {
	slots, ok := mw.world.Equipped(sim.EntityID(mw.invSubject.ID))
	if !ok {
		return data.Equipment{}
	}
	return equipmentFromSlots(slots)
}

func (mw *mapWorld) currentEquipmentItems() [sim.EquipSlots]sim.ItemInstance {
	items, ok := mw.world.EquippedItems(sim.EntityID(mw.invSubject.ID))
	if !ok {
		return [sim.EquipSlots]sim.ItemInstance{}
	}
	return items
}

func itemEquipmentStateEqual(a, b [sim.EquipSlots]sim.ItemInstance) bool {
	for i := range a {
		if a[i].Code != b[i].Code || a[i].Kind != b[i].Kind || a[i].Price != b[i].Price ||
			len(a[i].Effects) != len(b[i].Effects) {
			return false
		}
		for k := range a[i].Effects {
			if a[i].Effects[k] != b[i].Effects[k] {
				return false
			}
		}
	}
	return true
}

// weaponFallbackSpent is a PRESENT-STATE OBSERVATION with exactly one term:
// slot 1 of the equipment array is occupied, so the fallback picture is
// standing in for nothing and the weapon is real right now.
//
// IT NO LONGER SCANS THE PACK (round-2 adversarial review, seventh pass,
// R3). It used to answer true when the starting weapon's own CODE sat in the
// container, as a way of remembering an unequip across a mission boundary. A
// code is not an identity: a member carrying a second unit of the same code
// — bought, looted, or handed over by another member — read as
// already-materialized, and since the fifth pass that reading was written
// back onto the member permanently. The history it was standing in for is
// now kept where history belongs, in PartyMember.WeaponMaterialized itself,
// which survives both a mission boundary (mapload.CarryParty) and a save
// (FrontEnd.Snapshot, since the same pass made the live party the one the
// mission mutates). Nothing else in this file should call this function
// directly; resolveWeaponMaterialized is the door.
func (mw *mapWorld) weaponFallbackSpent(eq data.Equipment) bool {
	occupied, _ := eq.Occupied(1)
	return occupied
}

// missionPartyMember is a pointer to id's own record in the mission's own
// party slice — Mission.Party (pkg/game/mission.go), built once by
// mapload.OwnParty and never reallocated element-wise for the life of the
// mission, and the SAME slice FinishMission later hands to
// mapload.CarryParty. A write through this pointer — PartyMember.
// WeaponMaterialized — therefore survives the mission boundary exactly as
// Weapon itself already does, with no separate reconciliation step needed at
// the mission's own close. Nil if id names no member of this mission's party
// (mw.mission is nil, or id is not one of mw.mission.ids).
func (mw *mapWorld) missionPartyMember(id sim.EntityID) *mapload.PartyMember {
	if mw == nil || mw.mission == nil {
		return nil
	}
	for i, mid := range mw.mission.ids {
		if mid == id && i < len(mw.mission.party) {
			return &mw.mission.party[i]
		}
	}
	return nil
}

// resolveWeaponMaterialized folds member's own PERSISTED history together
// with weaponFallbackSpent's present-state reading of eq and stacks, and
// writes the observation back onto member the first time it turns true
// (round-2 adversarial review, fifth pass, the root cause behind
// counterexamples A and B). Once PartyMember.WeaponMaterialized is true this
// always returns true, regardless of what eq and stacks say this instant —
// that is the whole difference between this call and weaponFallbackSpent
// alone. member may be nil (a subject missionPartyMember could not resolve,
// or a caller with no party pointer at all), in which case this degrades to
// weaponFallbackSpent's own present-state answer, unable to remember it
// across a mission boundary.
func (mw *mapWorld) resolveWeaponMaterialized(member *mapload.PartyMember, eq data.Equipment) bool {
	if member == nil {
		return mw.weaponFallbackSpent(eq)
	}
	if mw.weaponFallbackSpent(eq) {
		materializeStartingWeapon(member)
	}
	return member.WeaponMaterialized
}

// currentFigureEquipment is currentEquipment widened by the slot-1 starting
// weapon only while currentWeaponFallbackActive says the starting weapon has
// never materialized in the equipment or pack. The ordinary doll,
// refreshDollDrag's suppressed doll, SlotInfo, the hover popup AND the
// world-sprite body (refreshAppearance, round-2 adversarial review, fifth
// pass, counterexample I) all read this value, so the doll, the popup and
// the figure standing on the map use one rule between them.
// refreshAppearance used to read currentEquipment directly and draw a
// fallback-only member bare-handed on the map while the doll standing beside
// it, composed from this function, showed him holding the starting weapon
// — the same DIV-070 population, two disagreeing pictures. HeroAppearance
// is a purely visual derivation with no stat consequence, unlike the two
// callers below that still need the raw array.
func (mw *mapWorld) currentFigureEquipment() data.Equipment {
	eq := mw.currentEquipment()
	if mw.currentWeaponFallbackActive() {
		eq.SetCode(1, mw.invParty.startWeapon.Code)
	}
	return eq
}

func (mw *mapWorld) currentFigureEquipmentItems() [sim.EquipSlots]sim.ItemInstance {
	items := mw.currentEquipmentItems()
	if mw.currentWeaponFallbackActive() && mw.invParty.startWeapon != nil {
		if member := mw.missionPartyMember(sim.EntityID(mw.invSubject.ID)); member != nil {
			items[0] = mapload.MemberItemEquipment(*member, mw.invParty.table)[0]
		}
		if items[0].Empty() {
			items[0] = mapload.ItemInstanceFromCode(uint16(mw.invParty.startWeapon.Code), mw.invParty.table)
		}
	}
	return items
}

// currentWeaponFallbackActive reports currentFigureEquipment's own condition
// directly, for a caller that needs the BOOL rather than the widened
// equipment (counterexample 4, round-2 adversarial review, third pass):
// ui.InventorySubject.WeaponFallback, which command.go reads at press time
// to refuse arming a drag or a tap-to-unequip for a slot 1 nothing in the
// live entity backs, so sim.unequip and dropFromEquipment both refuse it
// outright. Asked separately rather than inferred from comparing
// currentFigureEquipment's own result against currentEquipment's, which
// would cost the same two reads under a less direct name.
func (mw *mapWorld) currentWeaponFallbackActive() bool {
	occupied, _ := mw.currentEquipment().Occupied(1)
	return !occupied && mw.invParty.startWeapon != nil && !mw.invWeaponEverEquipped
}

// itemWeaponDamage is the inventory presenter's one route to a weapon-spell
// interval. An equipped weapon is answered only by the live World, including
// a false answer; an unequipped starting weapon may still describe its stored
// spell while it sits in the pack.
func (mw *mapWorld) itemWeaponDamage(code data.ItemCode) (weaponDamageInterval, bool) {
	if mw == nil || mw.world == nil {
		return weaponDamageInterval{}, false
	}
	id := sim.EntityID(mw.invSubject.ID)
	if equipped, ok := mw.world.Equipped(id); ok && equipped[0] == uint16(code) {
		return liveWeaponSpellDamage(mw.world, id, code)
	}
	if mw.currentWeaponFallbackActive() {
		return storedWeaponSpellDamage(code, mw.invParty.startWeapon, mw.invParty.mage, mw.invParty.table)
	}
	return weaponDamageInterval{}, false
}

func (mw *mapWorld) refreshEquipment() {
	if !mw.invSubjectSet {
		return
	}
	// eq GOES THROUGH THE SAME FALLBACK invFigureEquipment WAS SEEDED WITH
	// (counterexample 2, round-2 adversarial review): currentEquipment alone
	// would compare unequal to a fallback-aware seed on every single frame for
	// a member whose slot 1 the array never fills for real, recomposing every
	// tick instead of costing the one compare this method's own doc promises
	// — and worse, would recompose the figure WITHOUT the fallback, erasing
	// the weapon layer buildInventorySubject drew at open.
	eq := mw.currentFigureEquipment()
	fallback := mw.currentWeaponFallbackActive()
	figureItems := mw.currentFigureEquipmentItems()
	layers := mw.activeLayers(sim.EntityID(mw.invSubject.ID))
	if eq == mw.invFigureEquipment &&
		itemEquipmentStateEqual(figureItems, mw.invFigureEquipmentItems) &&
		fallback == mw.invSubject.WeaponFallback &&
		slices.Equal(layers, mw.invFigureLayers) {
		return
	}
	composed, _ := composeInventoryPortraitLayered(mw.mission.src, mw.invSubject.ID, eq, figureLayersFor(layers, mw.invParty.table),
		figureID{Dir: mw.invParty.figureDir, Face: mw.invParty.figureFace, Hero: mw.invParty.figureHero})
	mw.invFigureLayers = layers
	mw.invSubject.Figure, mw.invSubject.Slots = composed.Figure, composed.Slots
	// THE MASK MOVES WITH THE FIGURE, on the SAME reasoning as the popup text
	// below it (1005, "the interactive doll"): a mask left standing from the
	// PREVIOUS equipment would answer a hover or a press over the new figure
	// with the old one's slot boundaries the moment the two disagree — the
	// mask is a function of eq exactly as the picture is, and this is the one
	// call that recomposes both.
	mw.invSubject.SlotMask = composed.SlotMask
	// THE POPUP'S OWN WORN TEXT MOVES WITH THE FIGURE AND THE SLOTS (0151,
	// defect 5), on the SAME eq this call already resolved — a second read
	// of the equipment array here would be a second answer to a question
	// this method has already asked once.
	mw.invSubject.SlotInfo = slotItemInfoLinesWithWeaponDamage(figureItems, mw.invParty.table, mw.itemWeaponDamage, mw.view.Words())
	// WeaponFallback IS RECOMPUTED HERE TOO (counterexample 4, round-2
	// adversarial review, third pass): it is a pure function of the SAME eq
	// this call just resolved (currentFigureEquipment's own condition), so it
	// can only change inside this guarded branch — command.go reads it at
	// press time to refuse arming a drag for a slot nothing in the live entity
	// backs.
	mw.invSubject.WeaponFallback = fallback
	mw.invFigureEquipment = eq
	mw.invFigureEquipmentItems = figureItems
	// invComposedEquipment MOVES WITH invFigureEquipment HERE (C1b): this
	// branch just recomposed mw.invSubject.Figure/Slots from eq, above, so eq
	// is genuinely what the doll now shows, and the two trackers agree again
	// until the next live change. Between mission open and whichever recompose
	// (if any) reaches this line, invComposedEquipment stays at whatever
	// missionDollEquipment answered at open — which is the whole reason this
	// field can witness a wrong composition where invFigureEquipment cannot:
	// this branch's own guard, two statements up, compares eq against
	// invFigureEquipment (both live-derived), never against
	// invComposedEquipment, so a composition that started wrong here is never
	// "corrected" by a live-vs-live agreement that has nothing to say about
	// what was drawn.
	mw.invComposedEquipment = eq
	mw.view.SetInventorySubject(mw.invSubject)
}

// refreshDollDrag is item 5's own drain (1005, "the interactive doll"): the
// doll redraws without the dragged layer THE MOMENT A DRAG PICKS UP ONE OF
// ITS OWN SLOTS, by composing a DIFFERENT equipment set — that slot cleared —
// rather than by teaching the compositor a layer to skip (DLG-FIGURE-021:
// the compositor takes no layer selector). want is 0 for no drag, or the
// dragged slot's 1..12 number; invDollSuppressSlot is refreshEquipment's own
// guard-then-skip shape, so a drag that has not changed slot since last
// frame costs one compare.
//
// THE SUPPRESSED PICTURE NEVER REACHES mw.invSubject.Figure OR
// invFigureEquipment: it is handed to the viewer through
// SetDollSuppressedFigure, a SEPARATE field the doll box's own cache key
// (dollSource.suppressSlot) folds in, so ending the drag needs no recompose
// of the ordinary figure — refreshEquipment already holds the answer it
// should return to.
//
// composed.SlotMask TRAVELS WITH composed.Figure (1005 round-1 adversarial
// review): the two are the same call's own pair, and the viewer reads the
// mask back through dollFigureSlotAt whenever this suppression is the one in
// force — a hover or a press over the picture this call composed is
// answered from ITS OWN mask, never invSubject's stale one, which is exactly
// the staleness refreshEquipment's own SlotMask line above guards against
// for the ordinary figure.
func (mw *mapWorld) refreshDollDrag() {
	if !mw.invSubjectSet {
		return
	}
	slot0, dragging := mw.view.DraggedDollSlot()
	want := 0
	if dragging {
		want = slot0 + 1
	}
	if want == mw.invDollSuppressSlot {
		return
	}
	mw.invDollSuppressSlot = want
	if want == 0 {
		mw.view.SetDollSuppressedFigure(mw.invSubject.ID, 0, nil, nil)
		return
	}
	// eq GOES THROUGH THE SAME FALLBACK invFigureEquipment IS BUILT WITH
	// (counterexample 3, round-2 adversarial review, third pass):
	// currentEquipment alone leaves a fallback-only slot 1 (a member whose worn
	// set never folds his starting weapon in) at its raw, unoccupied code, so a
	// drag on any OTHER slot suppressed a figure with the weapon layer missing
	// while the ordinary figure — recomposed through currentFigureEquipment
	// by refreshEquipment above — still drew it, the exact
	// hit-test-vs-picture disagreement DIV-085 states must never happen,
	// restated for the suppressed doll rather than the ordinary one (the
	// round-1 review's own suppressed-mask fix already closed this disagreement
	// once, for the mask alone; this is the same disagreement's mission-side
	// figure, reopened by counterexample 2's own fallback).
	composed := mw.suppressedDollSubject(want)
	mw.view.SetDollSuppressedFigure(mw.invSubject.ID, want, composed.Figure, composed.SlotMask)
}

// suppressedDollSubject is refreshDollDrag's composition seam. Keeping the
// equipment-source choice here makes the source itself directly testable
// without exporting ui.Viewer's private pointer-drag state to pkg/game.
func (mw *mapWorld) suppressedDollSubject(want int) ui.InventorySubject {
	eq := mw.currentFigureEquipment()
	eq.SetCode(want, 0)
	composed, _ := composeInventoryPortraitLayered(mw.mission.src, mw.invSubject.ID, eq,
		figureLayersFor(mw.activeLayers(sim.EntityID(mw.invSubject.ID)), mw.invParty.table),
		figureID{Dir: mw.invParty.figureDir, Face: mw.invParty.figureFace, Hero: mw.invParty.figureHero})
	return composed
}

func (mw *mapWorld) refreshAppearance() {
	if !mw.invSubjectSet {
		return
	}
	eq := mw.currentFigureEquipment()
	if eq == mw.bodyEquipment {
		return
	}
	mw.bodyEquipment = eq

	body, dir, class, _ := data.HeroAppearance(mw.invParty.list, eq, mw.invParty.mage, false)
	key := data.HeroBodyKey(dir, body)
	LoadHeroBody(mw.mission.src, mw.units, dir, body)

	// A NIL BUNDLE RESOLVES NOTHING, LoadHeroBody's own guard restated for
	// the read below: that call already skipped over a nil mw.units without
	// touching it, and this function must too before indexing its own Bodies
	// field.
	if mw.units == nil {
		return
	}
	resolved := mw.units.Bodies[key]
	if resolved == nil {
		return
	}
	if mw.art == nil {
		mw.art = make(map[sim.EntityID]*terrain.UnitClass)
	}
	id := sim.EntityID(mw.invSubject.ID)
	mw.art[id] = resolved
	if char, ok := mw.chars[id]; ok {
		char.UnitNameIndex = int(class)
		mw.chars[id] = char
	}
}

func (mw *mapWorld) rearm() {
	if !mw.invSubjectSet {
		return
	}
	id := sim.EntityID(mw.invSubject.ID)
	e, held := mw.entity(id)

	// Seed the live-level comparison every call, including unchanged equipment.
	var levelMoved, trainingMoved bool
	if held {
		levelMoved = mw.invSkillSet && e.Skill != mw.invSkill
		trainingMoved = !mw.invSkillSet || e.NativeTraining != mw.invTraining
		mw.invSkill, mw.invSkillSet = e.Skill, true
		mw.invTraining = e.NativeTraining
	}

	eq := mw.currentEquipment()
	items := mw.currentEquipmentItems()
	layers := mw.activeLayers(id)
	if eq == mw.invEquipment && itemEquipmentStateEqual(items, mw.invEquipmentItems) && !levelMoved && slices.Equal(layers, mw.invLayers) {
		return
	}
	if eq == mw.invEquipment && itemEquipmentStateEqual(items, mw.invEquipmentItems) && slices.Equal(layers, mw.invLayers) &&
		(mw.world.NativeTrainingNeedsProducer(id) || held && e.NativeTraining.Present && !trainingMoved) {
		return
	}
	mw.invEquipment = eq
	mw.invEquipmentItems = items
	mw.invLayers = layers

	// THE HERO Rearm RECOMPUTES FROM IS A LOCAL COPY carrying the LIVE LEVELS,
	// never mw.invParty.hero itself — invParty is resolved once at mission
	// open and never written again (its own doc) — so the ten numbers a blow
	// resolves against follow a raise exactly as they already follow an equip.
	// A subject the world no longer holds keeps the hero's own seeded levels,
	// currentEquipment's identical reasoning for the equipment array applied to
	// the levels beside it.
	hero := mw.invParty.hero
	if held {
		hero.Skill = e.TrainedSkills(mw.skillBonus[id])
	}

	// invWeaponEverEquipped ADVANCES HERE, ahead of the call, on invSkill's
	// own precedent above: whether slot 1 is occupied THIS tick is a fact
	// about eq, already read, and the tracker has to see every occupied
	// read to stay sticky — see its own doc on the struct field, and
	// ResolveEquipmentLoadout's (pkg/mapload/loadout.go) for what the flag
	// changes once it is true.
	//
	// THE SAME OBSERVATION ALSO WRITES THE PERSISTED BIT (round-2 adversarial
	// review, fifth pass): a real occupation seen here is a live equip,
	// mid-mission, that this subject's own opening seed could not have known
	// about — the one place besides the two seeds that resolves the fallback
	// into something real, so it is the one place besides them that has to
	// write PartyMember.WeaponMaterialized.
	if occupied, _ := eq.Occupied(1); occupied {
		mw.invWeaponEverEquipped = true
		materializeStartingWeapon(mw.missionPartyMember(id))
	}
	w, wrote := RearmWithLayers(mw.world, id, hero, mw.invParty.profile, mw.invParty.startWeapon, mw.invWeaponEverEquipped, mw.invParty.table,
		mapload.RotationSpeedBase(mw.invParty.hired, mw.invParty.hiredRotationSpeed, mw.invParty.class, mw.invParty.table), layers)
	if !wrote {
		return
	}
	mw.settleSkillBonus(id, mapload.EquippedSkillBonus(items, mw.invParty.profile.Fighter))
	if now, ok := mw.entity(id); ok {
		mw.invSkill = now.Skill
	}
	mw.refreshWornCharacter(id)

	// THE PANEL'S NAME, off the SAME resolved w Rearm derived the block
	// from. It is the third thing that did not arrive at an equip, after
	// the credited slot: mw.chars is written once, by partyCharacters at
	// the open, and nothing else in this tree wrote it again — so the four
	// numbers a new weapon changed moved and the row naming the weapon that
	// changed them went on naming the one he started the mission with.
	//
	// IT IS WRITTEN HERE RATHER THAN OVERLAID IN entityDraws, the way Skills
	// and Experience are: the overlay would have to re-resolve a code to a
	// definition on every tick for every entity, and only the inventory
	// SUBJECT has equipment this tree can change or an invPartyGear to resolve
	// it against — so the uniform-looking shape is a per-tick resolve that
	// answers nothing for every entity but this one. The cost of writing it
	// here instead is stated plainly: this covers the subject and no other
	// party member, which is exactly the set whose weapon can move.
	//
	// A NIL w IS A BARE HERO and clears the name rather than leaving the last
	// one standing — partyCharacters' own rule at the other end of this map,
	// applied again. w falls back to invParty.startWeapon, which is nil for a
	// character generated holding nothing, and "no weapon" is a real state
	// (inventory.go) that must not read as the previous weapon's name.
	//
	// The lookup is a comma-ok: openMission pairs Party[0] with Start.IDs[0]
	// for both this map and the subject, so the entry is there — reading it
	// back keeps that pairing a fact of openMission's rather than an
	// assumption of this function's, and a miss writes nothing instead of
	// minting a character the load knows none of.
	if char, ok := mw.chars[id]; ok {
		char.Weapon = ""
		if w != nil {
			char.Weapon = w.Name
		}
		mw.chars[id] = char
	}
}

// seedSkillBaseline records every known character's current levels as the
// level a later skillRises call compares against. A mission calls it once when
// it finishes opening, so a raise in the first simulated tick posts its notice
// like any later one and a restored mission never posts the levels its save
// already held.
func (mw *mapWorld) seedSkillBaseline() {
	for _, e := range mw.world.EntityView() {
		if !mw.chars[e.ID].Known {
			continue
		}
		if mw.skillPosted == nil {
			mw.skillPosted = make(map[sim.EntityID][data.SkillSlots]int32)
		}
		mw.skillPosted[e.ID] = e.Skill
	}
}

func (mw *mapWorld) skillRises() []string {
	var rows []string
	var words *ui.Words
	ents := mw.world.EntityView()
	for i := range ents {
		e := &ents[i]
		char := mw.chars[e.ID]
		if !char.Known {
			continue
		}
		last, seen := mw.skillPosted[e.ID]
		if mw.skillPosted == nil {
			mw.skillPosted = make(map[sim.EntityID][data.SkillSlots]int32)
		}
		mw.skillPosted[e.ID] = e.Skill
		if !seen || e.Skill == last {
			continue
		}
		if words == nil {
			words = mw.skillRaiseWords()
		}
		rows = append(rows, skillRiseRows(words, e.MaxMana > 0, last, e.Skill)...)
	}
	if mw.view != nil {
		announce(mw.view, rows)
	}
	return rows
}

// skillRaiseWords is the word set a raise is stated in: the viewer's install
// words, already in the alphabet its font draws, or the authored set when no
// viewer is attached.
func (mw *mapWorld) skillRaiseWords() *ui.Words {
	w := ui.AuthoredWords()
	if mw.view != nil {
		w = mw.view.Words()
	}
	return &w
}

// skillRiseRows is skillRises' own row-building half, factored out for the
// SAME reason pickupLinesForItems is: it is a PURE function of a word set, a
// class flag and two level snapshots, so a test can assert precisely what one
// raise states without reaching into pkg/ui's opaque Viewer.
//
// last IS WHAT skillRises LAST SAW for this entity, and current IS THIS
// TICK'S OWN LIVE READING — never a before/after pair inside one tick, so
// passing the same array for both answers nil (AC-9's "a tick with no raise
// posts nothing", read down to this function's own domain).
//
// Each row is the slot's raise line and its new level (DIV-1449), in slot
// order. Slot 0 has no raise line and posts nothing: no mission award raises
// it (HERO-SKILLGATE-074, HERO-SKILLBUY-076).
func skillRiseRows(words *ui.Words, mage bool, last, current [data.SkillSlots]int32) []string {
	var rows []string
	for slot := int32(0); slot < data.SkillSlots; slot++ {
		if current[slot] <= last[slot] {
			continue
		}
		line, ok := words.SkillRaisedLine(mage, int(slot))
		if !ok {
			continue
		}
		rows = append(rows, fmt.Sprintf("%s: %d", line, current[slot]))
	}
	return rows
}

// enqueue is the ui.MapOrder the loader hands the front-end beside the tick:
// it converts one order into the simulation's own command, appends it to the
// queue and marks its entity commanded, and it does NOTHING ELSE.
//
// The two writes are ONE STATEMENT. An entity that is queued for but not
// marked keeps taking scripted targets and the feature reads as broken; one
// marked but not queued leaves the script for an order that never arrives.
//
// The conversion is the seam's own widths read back down: the uint32 the
// window tier carries is the id's own width, so sim.EntityID(entity)
// recovers precisely the id entityDraws minted, and the two ints are a map
// cell, which is what a command's X and Y already are. It names an entity
// the world may no longer hold — a Step over a command for an absent
// entity is already a no-op, which is why nothing here validates one. Such
// an entity is marked all the same: the set is a record of what the
// FRONT-END ordered, and asking a world what it still holds would put a
// world read on the issuing path. THE KIND IS THE GROUP MOVE-TO, and that is
// 0059's whole change here. Every order queued between two advances lands in
// ONE command slice and is applied in ONE tick, and every one of them comes
// from one LEFT click over one selection — one click, one slice, one tick,
// one tag — so they carry the tag's zero value and the simulation reads
// them as a single group order. That is what the thing being reconstructed
// does with a selection: a player order builds a group out of it, computes
// the group's centroid and its slowest member once, and issues to every
// member from there.
//
// A selection of ONE takes no other path. A group of one is its own centroid, its
// offsets are zero and its rate term is its own speed, so a single unit ordered
// through this seam goes exactly where it went before — which is why the widening
// needs no arm here and no new seam above it.
func (mw *mapWorld) enqueue(entity uint32, x, y int) {
	id := sim.EntityID(entity)
	cx, cy := int32(x), int32(y)
	// The tag is the CLICK, and the queue is what says where one ends. A right
	// click emits one order per selected unit, all naming the clicked cell and
	// each unit once, so an order joins the group being assembled exactly when
	// that group already has a member, names the same cell, and does not already
	// name this entity. Anything else is a second press, and a second press is a
	// second order — which is what keeps "the last order issued wins" true of two
	// clicks between one pair of advances.
	//
	// It is derived from the queue rather than remembered beside it, so a drain
	// ends a group for free: nothing pending carries the tag any more, and the
	// next order opens a new one. The counter is the only state, and it never
	// reaches a world — the simulation reads a tag inside one advance and stores
	// it nowhere.
	mw.queueGroup(func(tag uint32) sim.Command { return sim.GroupMoveTo(id, sim.CellPoint{X: cx, Y: cy}, tag) })
}

// stance is the ui.MapStance the loader hands the front-end: it turns one
// cell-free standing order into the simulation's own command, appends it to
// the SAME QUEUE the orders, the attacks and the blows use, and marks its
// entity commanded — and it does NOTHING ELSE.
//
// THIS IS WHERE THE BOOL BECOMES A NUMBER, and it is the only place in this
// tree that it does. The seam names two things the player can ask for; which of
// the law's own order bytes each is, is decided here, on the world's side of the
// tier boundary, exactly as MapAffect's own "what a chip takes off" is.
//
// The order rides in the command's X because that is where the simulation's
// stance arm reads it — the same 32 bits under another name KindAttack's victim
// already is. Y is unread and is left at zero, which is not a cell: this kind
// carries none.
//
// It marks, on strike's own grounds: a unit told to hold ground has been given
// something to do, exactly as one told where to walk has, so the placeholder
// script must not overwrite it. And it steps no world — sim.Step is reached
// from tick and from nowhere else in this package.
func (mw *mapWorld) stance(entity uint32, guard bool) {
	order := sim.OrderStandGround
	if guard {
		order = sim.OrderGuard
	}
	mw.queueGroup(func(tag uint32) sim.Command { return sim.GroupStance(sim.EntityID(entity), order, tag) })
}

// march is the ui.MapMarch the loader hands the front-end: stance's sibling
// for the two orders that name a CELL, on the same queue, with the same mark
// and the same silence about the world.
//
// The two ints are a map cell, which is what a command's X and Y already are —
// enqueue's own conversion, unchanged.
func (mw *mapWorld) march(entity uint32, patrol bool, x, y int) {
	id, at := sim.EntityID(entity), sim.CellPoint{X: int32(x), Y: int32(y)}
	if patrol {
		mw.queueGroup(func(tag uint32) sim.Command { return sim.GroupPatrolTo(id, at, tag) })
		return
	}
	mw.queueGroup(func(tag uint32) sim.Command { return sim.GroupSwarmTo(id, at, tag) })
}

// queueGroup is the one body enqueue, stance and march share: open a group
// tag or join the one being assembled, append the command, mark the entity
// commanded.
//
// THE TAG IS THE PRESS, and the queue is what says where one ends — enqueue's
// own rule, now asked of the kind as well as of the cell. A key press emits one
// command per selected unit, all of the same kind and all naming the same two
// numbers, so a command joins the group being assembled exactly when that group
// already has a member OF THIS KIND, names the same pair, and does not already
// name this entity. Anything else is a second press, and a second press is a
// second order.
//
// Asking the kind is what keeps a stance press and a move press in one frame
// from ever merging onto one tag: the counter is small and a tag is reused the
// moment a drain empties the queue, so without it two orders of different sorts
// could share one and the simulation would resolve membership across both.
// It takes a BUILDER and not a kind with two loose numbers, so each of the four
// presses names its own group constructor and the two numbers stop being a pair
// this function has to carry without knowing what they mean. The order is built
// twice: the first build is a probe whose kind and aim the membership test
// reads, and only the tag differs between the two. Building is pure field
// assignment, so the probe costs nothing and disturbs nothing it measures.
func (mw *mapWorld) queueGroup(build func(tag uint32) sim.Command) {
	probe := build(0)
	id := probe.Entity
	mw.cancelPickup(id)
	if !mw.joinsGroup(probe.Kind, mw.groupTag, id, probe.X, probe.Y) {
		mw.groupTag++
	}
	mw.pending, mw.commanded[id] = append(mw.pending, build(mw.groupTag)), true
}

// joinsGroup reports whether a command of this kind for id carrying (x,y)
// belongs to the group order tagged tag as the pending queue holds it: that tag
// has at least one member of this kind, every one of them carries this pair, and
// none of them names this entity.
//
// A MEMBER OF THIS TAG UNDER ANOTHER KIND REFUSES THE JOIN rather than being
// skipped: the tag is already spoken for by a different order, so this command
// opens a new one.
func (mw *mapWorld) joinsGroup(kind uint8, tag uint32, id sim.EntityID, x, y int32) bool {
	found := false
	for _, c := range mw.pending {
		if c.Group != tag {
			continue
		}
		if c.Kind != kind || c.X != x || c.Y != y || c.Entity == id {
			return false
		}
		found = true
	}
	return found
}

// strike is the ui.MapAttack the loader hands the front-end beside the tick,
// the order, the cadence and the blow: it turns one attack order into the
// simulation's own command, appends it to the SAME QUEUE the orders and the
// blows use, and marks its entity commanded — and it does NOTHING ELSE.
//
// IT MARKS, WHERE affect DOES NOT, and that one line is the whole difference
// between the two far sides. The commanded set exists to keep the placeholder
// script from overwriting a player's ORDER, and an attack IS an order: a unit
// told whom to fight has been given something to do, exactly as one told where to
// walk has. A blow is not — hitting a unit says nothing about what it should do —
// which is why that seam leaves the set alone and this one writes it, in the same
// tuple assignment enqueue makes, so no path performs one write and not the other.
//
// IT LOOKS NOTHING UP, where affect does. The blow seam has to read the victim to
// decide the amount; an order carries no amount, and what the world still holds is
// a question the advance already answers — a command naming an absent entity, an
// absent victim or the attacker itself is a no-op at the arm that applies it. So
// there is no world read on this path at all, and no ownership test either: who
// may be attacked is not decided here, and in what is being reconstructed it is
// not decided at the click at all.
//
// The conversion is the seam's own widths read back down: both uint32 are the id's
// own width, and the victim rides in the command's X because that is where the
// simulation's attack arm reads it — the same 32 bits under another name.
//
// It steps no world. sim.Step is reached from tick and from nowhere else in this
// package, so "an attack order reaches the world only through an advance" is a
// property of there being one call site rather than a rule someone keeps: a press
// with no advance behind it leaves every world field, the byte form and the digest
// exactly where they were.
func (mw *mapWorld) strike(entity, victim uint32) {
	id := sim.EntityID(entity)
	mw.cancelPickup(id)
	cmd := sim.Attack(id, sim.EntityID(victim))
	mw.pending, mw.commanded[id] = append(mw.pending, cmd), true
}

// castAt is strike's SIBLING for a CAST: one spell order in, one
// sim.KindCast command on the SAME queue strike, enqueue and affect all use,
// and nothing else touched.
//
// THE CONVERSION IS STRIKE'S OWN, one field wider: the victim rides in X
// exactly as strike's does, and the spell rides in Y — the same two 32-bit
// widths crossing the seam under other names, KindCast's own doc (step.go).
//
// IT MARKS, exactly as strike does and for strike's own reason: a cast is an
// order, so a caster told what to cast at has been given something to do,
// where affect's blow is not one and leaves the set alone.
func (mw *mapWorld) castAt(entity, victim, spell uint32) {
	id := sim.EntityID(entity)
	mw.cancelPickup(id)
	cmd := sim.Cast(id, sim.EntityID(victim), sim.SpellID(spell))
	mw.pending, mw.commanded[id] = append(mw.pending, cmd), true
}

// attackOrCast is the ui.MapAttack the loader now hands the front-end: it
// dispatches to castAt when spell is nonzero and to strike otherwise, spell
// 0 being ui.MapAttack's own "no spell selected, this is an attack" — the
// split this seam's whole doc block describes, made concrete in the one
// place it is decided.
func (mw *mapWorld) attackOrCast(entity, victim, spell uint32, x, y int, cell bool) {
	if spell != 0 {
		if cell {
			// Teleport is allowed into an unseen destination so mission 80's
			// disconnected grave can be reached. Keep the populated fog plane's
			// map bounds guard; range and terrain remain simulation refusals.
			const teleportSpellID = 26
			if spellArmOf(mw.world.Spells(), spell) == teleportSpellID && !mw.fog.contains(x, y) {
				return
			}
			id := sim.EntityID(entity)
			mw.cancelPickup(id)
			mw.pending = append(mw.pending, sim.CastAt(id, sim.SpellID(spell), sim.CellPoint{X: int32(x), Y: int32(y)}))
			mw.commanded[id] = true
			return
		}
		mw.castAt(entity, victim, spell)
		return
	}
	if cell {
		id := sim.EntityID(entity)
		mw.cancelPickup(id)
		mw.pending = append(mw.pending, sim.AttackStructure(id, sim.StructureID(victim)))
		mw.commanded[id] = true
		return
	}
	mw.strike(entity, victim)
}

// chipDivisor is what a chip takes off: a tenth of the unit's own maximum. It is
// a DEBUG amount and not a formula — nothing is rolled, no class is read, and no
// absorption or resistance exists to reduce it — so ten presses fell a unit at
// full health whatever that health is.
const chipDivisor = 10

// affect is the ui.MapAffect the loader hands the front-end beside the tick,
// the order and the cadence: it turns one blow into the simulation's own
// command and appends it to the SAME QUEUE the orders use, and it does
// NOTHING ELSE.
//
// THE AMOUNT IS DECIDED HERE, on the side that owns the world. A chip is a tenth
// of the named unit's own maximum and never less than 1, so a unit with a
// maximum below ten still loses something and a chip is never a heal; the seam
// carries a bool, so the front-end holds no damage rule and this side takes no
// number it cannot check.
//
// IT LOOKS THE ENTITY UP, and a blow naming one the world no longer holds
// appends nothing at all. A command for an absent entity is already a no-op, so
// this is not a correctness guard — it is what keeps a queue the next advance
// drains from filling with commands that cannot do anything, and it is the read
// the amount needs in any case.
//
// It does NOT mark the entity commanded. That set exists to keep the placeholder
// script from overwriting a player's ORDER, and a blow is not an order: hitting
// a unit says nothing about where it should walk, so a chipped unit stays on
// whatever script it was on.
//
// It steps no world. sim.Step is reached from tick and from nowhere else in this
// package, so "a blow reaches the world only through an advance" is a property
// of there being one call site rather than a rule someone keeps: a key press
// with no advance behind it leaves every world field, the byte form and the
// digest exactly where they were.
func (mw *mapWorld) affect(entity uint32, kill bool) {
	id := sim.EntityID(entity)
	e, ok := mw.entity(id)
	if !ok {
		return
	}
	cmd := sim.Kill(id)
	if !kill {
		amount := e.MaxHP / chipDivisor
		if amount < 1 {
			amount = 1
		}
		cmd = sim.Damage(id, amount)
	}
	mw.pending = append(mw.pending, cmd)
}

func (mw *mapWorld) grab(entity uint32, col, row int, aimed bool) {
	if !aimed {
		if !mw.invSubjectSet {
			return
		}
		mw.takeSackFor(sim.EntityID(mw.invSubject.ID))
		return
	}
	mw.orderPickup(sim.EntityID(entity), int32(col), int32(row))
}

// pickupIntent is one standing pick-up order: which entity was told to take a
// sack, and the cell the sack was standing on when the order was given. set
// tells an armed order from the zero value, on invSubjectSet's own grounds --
// entity 0 is a real id.
type pickupIntent struct {
	id   sim.EntityID
	x, y int32
	set  bool
}

// orderPickup refuses absent sacks (ITEM-PICK-016), then queues the cell in
// canonical state beside any physical progress (AI-356). Arm presentation
// after queueGroup, which cancels its predecessor's presentation latch.
func (mw *mapWorld) orderPickup(id sim.EntityID, x, y int32) {
	if _, ok := mw.sackAt(x, y); !ok {
		return
	}
	if e, ok := mw.entity(id); !ok || !e.Alive() || e.OffMap {
		return
	}
	mw.queueGroup(func(tag uint32) sim.Command { return sim.PickUp(id, sim.CellPoint{X: x, Y: y}) })
	mw.pickup = pickupIntent{id: id, x: x, y: y, set: true}
}

// settlePickup reconstructs the presentation latch from canonical pending
// state after LOAD. Transfer waits for both arrival and physical completion.
// Removal, refusal or arrival disarms the request; a failed transfer never loops.
func (mw *mapWorld) settlePickup() {
	mw.pickup = pickupIntent{}
	for i, actors := 0, mw.world.EntityView(); i < len(actors); i++ {
		if p := actors[i].PendingOrder; p.Kind == sim.PendingPickup {
			mw.pickup = pickupIntent{id: actors[i].ID, x: p.X, y: p.Y, set: true}
			break
		}
	}
	if !mw.pickup.set {
		return
	}
	e, ok := mw.entity(mw.pickup.id)
	if !ok || !e.Alive() || e.OffMap {
		mw.world.CancelSackPickup(mw.pickup.id)
		mw.pickup = pickupIntent{}
		return
	}
	if _, ok := mw.sackAt(mw.pickup.x, mw.pickup.y); !ok {
		mw.world.CancelSackPickup(mw.pickup.id)
		mw.pickup = pickupIntent{}
		return
	}
	if e.X != mw.pickup.x || e.Y != mw.pickup.y || !sackPickupArrived(mw.world, e) {
		return
	}
	id := mw.pickup.id
	mw.pickup = pickupIntent{}
	mw.takeSackFor(id)
	mw.world.CancelSackPickup(id)
}

// cancelPickup drops a standing pick-up order for one entity. It is called from
// queueGroup, the one place move, stance, march and the two attack arms all
// reach, and from strike and castAt, which append their own command.
//
// A LATER ORDER CANCELS AN EARLIER PICK-UP because the original's does: `0x21`
// writes `actor+0x50 = 2` and every other order arm writes that same field, so
// a second order overwrites the pick-up state rather than queuing behind it
// (`AI-CMD-032`, `AI-STATE-011`). Here the field is on the driver, so the
// overwrite has to be performed rather than inherited.
func (mw *mapWorld) cancelPickup(id sim.EntityID) {
	if mw.pickup.set && mw.pickup.id == id {
		mw.pickup = pickupIntent{}
	}
}

// takeSackFor is the transfer half of the pick-up, for ONE NAMED ENTITY: the
// sack under that entity's own current cell, taken all-or-nothing, with the
// message line posted only when TakeSack actually succeeded.
//
// IT IS NAMED BY ITS CALLER AND NAMES NOBODY ITSELF, which is what closed
// `DIV-292`. Until this story's round 4 the whole function was grab's body and
// answered the "which character" question once, for the inventory window's
// subject, whatever the click had named. The KEY still acts for the window's
// subject, which is MapGrab's own doc and correct; the CLICK now acts for the
//
// EACH LINE IS A CARRIED ITEM THE TAKE REACHED. The taker's container, and
// the primary hero's when quest documents move to him, are read before and
// after the take; each stack it created or merged units into posts one line
// stating that Item's count (SAV-1114, reachedStacks).
//
// THE LOG IS POSTED ONLY WHEN TakeSack ACTUALLY SUCCEEDED. A read that found
// a sack's codes but a take that then refused them — the owner-past-
// relationSlots refusal TakeSack's own doc states, the only one reachable
// once this function's own two guards below have already passed — must not
// tell the player something was picked up when the world moved nothing, so
// the error gates the post exactly where the caller-visible failure used to
// be silently discarded.
//
// EACH ITEM IS NAMED THROUGH itemName, below (0151): the code's raw bits
// looked up in invParty.table's own Names first — the shipped table an
// armour, a shield or a magic item is named through and a weapon usually
// is too — then, when Names carries no line for it, the same recovery the
// equip half uses, data.WeaponFromCode against Shapes, Materials and
// Weapons, the identical call enqueueEquip and rearm already make. A code
// that resolves through neither prints its own seven-digit digits
// (data.ItemCode.Name) rather than being dropped, so a gap in what this
// build can name is visible on screen instead of silently missing a line.
//
// AN ERROR IS STILL DISCARDED as a caller-visible failure. "There is no sack
// here" remains the ordinary answer to pressing the key with nothing under
// the subject, not a failure this driver reports anywhere — the same silence
// affect keeps for an entity the world no longer holds.
// unit the cursor's gate named.
func (mw *mapWorld) takeSackFor(id sim.EntityID) {
	if e, ok := mw.entity(id); ok && e.HasAttackTarget && e.AttackPhase != sim.AttackReady && !(e.PendingOrder.Kind == sim.PendingPickup && e.PendingOrder.RowAdmitted) {
		mw.orderPickup(id, e.X, e.Y)
		return
	}
	primary, primaryOK := mw.primaryPartyID()
	holders := []sim.EntityID{id}
	if primaryOK && primary != id {
		holders = append(holders, primary)
	}
	before := make([][]sim.ItemStack, len(holders))
	for k, holder := range holders {
		before[k], _ = mw.world.CarriedStacks(holder)
	}
	purse := mw.world.Purse(sim.SelfSlot)
	if _, ok := takeSackUnderfoot(mw.world, id, primary, primaryOK); !ok {
		return
	}
	mw.cancelPickup(id)
	mw.cancelQueuedPickupPredecessors(id)
	var reached []sim.ItemStack
	for k, holder := range holders {
		after, _ := mw.world.CarriedStacks(holder)
		reached = append(reached, reachedStacks(before[k], after)...)
	}
	words := mw.view.Words()
	announce(mw.view, pickupLinesForTake(reached, purseGain(purse, mw.world.Purse(sim.SelfSlot)), mw.invParty.table, &words))
}

// PickUpUnderfoot is the pickup key's world half for a mission driven without
// a map: the actor takes the sack on its own cell, quest documents go to the
// starting hero, and the pickup completes on the next Step.
func (m *Mission) PickUpUnderfoot(id sim.EntityID) (sim.Sack, bool) {
	var primary sim.EntityID
	primaryOK := false
	for i, p := range m.Party {
		if i < len(m.Start.IDs) && primaryPlayerHero(p) {
			primary, primaryOK = m.Start.IDs[i], true
			break
		}
	}
	return takeSackUnderfoot(m.World, id, primary, primaryOK)
}

// takeSackUnderfoot is the world half of takeSackFor, shared with the headless
// scenario's pick-up step: it reports the sack it took and whether it took one.
func takeSackUnderfoot(w *sim.World, id, primary sim.EntityID, primaryOK bool) (sim.Sack, bool) {
	e, ok := w.Entity(id)
	if !ok || !e.Alive() || e.OffMap || !sackPickupArrived(w, e) {
		return sim.Sack{}, false
	}
	var sack sim.Sack
	for _, s := range w.Sacks() {
		if s.X == e.X && s.Y == e.Y {
			sack = s
			break
		}
	}
	var documents uint32
	for _, code := range sack.Items {
		if code == uint16(data.QuestDocumentCode) {
			documents++
		}
	}
	if documents > 0 {
		if !primaryOK {
			return sim.Sack{}, false
		}
		if _, ok := w.Carried(primary); !ok {
			return sim.Sack{}, false
		}
	}
	if err := w.TakeSack(id, e.X, e.Y); err != nil {
		return sim.Sack{}, false
	}
	if documents > 0 && id != primary {
		if err := w.MoveCarried(id, primary, uint16(data.QuestDocumentCode), documents); err != nil {
			return sim.Sack{}, false
		}
	}
	w.CompleteSackPickup(id)
	return sack, true
}

func sackPickupArrived(w *sim.World, e sim.Entity) bool {
	if e.Transit != 0 || e.AttackPhase != sim.AttackReady && !(e.PendingOrder.Kind == sim.PendingPickup && e.PendingOrder.RowAdmitted) || w.ActorOrderProgress(e.ID) != 0 {
		return false
	}
	x, y, present := w.ActorFinePosition(e.ID)
	return !present || x == 128 && y == 128
}

// The underfoot key transfers synchronously. Older orders still queued for
// this actor must not overwrite its successful pickup on the following tick.
// Other actors and inventory/parameter commands retain their original order.
func (mw *mapWorld) cancelQueuedPickupPredecessors(id sim.EntityID) {
	kept := mw.pending[:0]
	ignored := make([]bool, 0, len(mw.pending))
	for i, c := range mw.pending {
		if c.Entity == id {
			switch c.Kind {
			case sim.KindMoveTo, sim.KindPickUp, sim.KindGroupMoveTo, sim.KindGroupSwarmTo,
				sim.KindGroupStance, sim.KindGroupPatrolTo, sim.KindGroupDefend,
				sim.KindGroupRetreat, sim.KindAttack, sim.KindAttackStructure,
				sim.KindCast, sim.KindCastAt, sim.KindUseScroll, sim.KindUseScrollAt:
				continue
			}
		}
		kept = append(kept, c)
		ignored = append(ignored, i < len(mw.pendingIgnored) && mw.pendingIgnored[i])
	}
	mw.pending = kept
	mw.pendingIgnored = ignored
}

// primaryPartyID is the explicitly marked starting player's entity, paired by
// the mission start's parallel party/id slices. No position is inferred when
// the marker is absent.
func (mw *mapWorld) primaryPartyID() (sim.EntityID, bool) {
	if mw == nil || mw.mission == nil {
		return 0, false
	}
	for i, p := range mw.mission.party {
		if i >= len(mw.mission.ids) {
			break
		}
		if primaryPlayerHero(p) {
			return mw.mission.ids[i], true
		}
	}
	return 0, false
}

// sackAt is the ground sack at (x, y), or the zero value when none stands
// there. The returned sack is already a copy from World.Sacks.
func (mw *mapWorld) sackAt(x, y int32) (sim.Sack, bool) {
	for _, s := range mw.world.Sacks() {
		if s.X == x && s.Y == y {
			return s, true
		}
	}
	return sim.Sack{}, false
}

// sackItemsAt is the item codes the ground sack at (x, y) carries, or nil for
// a cell holding none — grab's OWN READ, taken BEFORE TakeSack consumes the
// sack (plan R-4), over (*sim.World).Sacks' own fresh copy, so nothing here
// can reach back into the world it read from.
//
// IT IS HANDED THE SAME (x, y) TakeSack IS THEN CALLED WITH, both taken from
// one read in the caller, which is the whole of R-4's "the two lookups must
// agree": one sack per cell (sim/sack.go's own construction-time fold) means
// naming that cell twice can only ever find the one sack standing on it.
func (mw *mapWorld) sackItemsAt(x, y int32) []uint16 {
	s, _ := mw.sackAt(x, y)
	return s.Items
}

func itemName(code data.ItemCode, table *mapload.Table) string {
	if table != nil {
		if s, ok := table.Names.NameFor(code); ok {
			return s
		}
		if w, err := data.WeaponFromCode(code, table.Shapes, table.Materials, table.Weapons); err == nil {
			return w.Name
		}
	}
	return code.Name()
}

// entity is the world's own entry for id, and whether it holds one. It reads
// through the copy-handing entity read, so what comes back reaches nothing of
// the world it came from.
//
// It walks the slice rather than searching it: this is asked once per marked
// unit per key press, where the walk's cost is a rounding error, and a second
// copy of the package's binary search would be a second thing to keep in
// agreement with the ordering it assumes.
func (mw *mapWorld) entity(id sim.EntityID) (sim.Entity, bool) {
	return mw.world.Entity(id)
}

// paceTo advances the world by however many WHOLE logic ticks have elapsed
// since the previous call, and reports how many it ran AND how many pending
// orders those ticks applied. It is the tested spelling of paced: now is an
// argument, so the whole cadence is decidable without a clock.
//
// A call that fires NO tick applies none and keeps the queue: the drain happens
// inside a tick, so an advance the cadence decided was worth nothing leaves an
// order exactly where it was, for the advance that does fire one. Within a call
// that fires several, only the first finds a queue, so the count is the queue's
// length at that first tick and zero at every later one — which is what makes it
// the number of orders THIS ADVANCE applied rather than a running total (0028
// AC-8).
//
// The FIRST call only takes the baseline and advances nothing — the standalone
// viewer's own pinned rule for its water ticker, and the reason a slow startup
// cannot fire a burst out of the time before the map opened.
//
// Only the WHOLE MICROSECONDS handed to the ticker are consumed: the
// sub-microsecond tail is left on the baseline rather than dropped, exactly
// where the sub-millisecond one was left before the clock's unit changed, so a
// 60 fps front-end (16666.67 us a frame) loses nothing to truncation. The
// ticker carries the sub-TICK remainder the same way, so no elapsed time is
// lost at either scale.
//
// A non-positive elapsed advances nothing and does not move the baseline: a
// clock that jumped backwards must not rewind the cadence, and time.Now()'s
// monotonic reading makes it unreachable in production anyway.
//
// STOPPED, IT TAKES THE BASELINE AND RETURNS ZERO. The elapsed span is
// consumed by that write — the same write the running path makes, on the
// same line — so nothing is owed when the stop clears and the resume
// cannot burst: the first advance after it runs ONE tick, not a backlog,
// whether the stop was held for a frame or for an hour. The elapsed never
// reaches the accumulator, which is what makes that a property of where the
// return stands rather than of how large the catch-up bound happens to be.
//
// The stop gates the TICK LOOP alone. It changes no rate, empties no queue and
// reaches nothing inside tick(): what a tick does is unopened here, so orders
// issued while stopped are simply orders no tick has drained yet.
//
// The tick count is bounded by maxCatchUp at the period in force and then
// run as that many tick() calls — the ONE advance, unedited. Wall-clock
// decides when a tick fires; a tick is the same integer step whatever paced
// it, which is why k paced ticks reach the state and the digest of k direct
// ones.
//
// AFTER THE TICK LOOP IT PUSHES THE PHASE: the clock's own remainder and
// period, from THIS clock — the pacing accumulator that decided how many
// ticks just fired — and never from the viewer's water instance, which is
// a second instance of the same type re-rated through its own call site and
// carrying its own remainder. It is pushed on every call that reaches this
// line, ticks or none, because a frame that fired no tick is exactly the
// frame that has advanced within one. Every earlier return — the baseline,
// a non-positive elapsed, the stop — is BEFORE the elapsed reaches the
// accumulator, so the pair last pushed is still the one in force and a
// stopped world's picture holds still without a rule of its own.
func (mw *mapWorld) paceTo(now time.Time) (ticks, applied int) {
	// THE READOUT PUSH, deferred so that it happens on EVERY exit. Three of the
	// four returns below are early — the baseline call, a non-positive
	// elapsed, and the stop — and the stop is exactly the state the readout
	// most needs to report: a push placed after the tick loop would never fire
	// for a stopped world, so the box would never learn about Space. Deferred,
	// "the readout reports every frame" is a property of one statement rather
	// than of four returns being kept in step.
	//
	// It runs AFTER this call's ticks, so what is pushed is the state the frame
	// about to be drawn will be drawn from.
	defer mw.pushReadout()

	if mw.unpaced {
		// No elapsed-time or deadline test exists on this arm. One eligible
		// owner callback is one tick, even at the same timestamp or after a
		// long stall. last is still refreshed so the paced arm can never see
		// the span spent here as debt.
		mw.last = now
		if mw.stopped {
			return 0, 0
		}
		mw.clock.AdvanceOne()
		applied = mw.tick()
		mw.view.SetPhase(0, mw.clock.Period())
		return 1, applied
	}

	if mw.last.IsZero() {
		mw.last = now
		return 0, 0
	}
	elapsed := now.Sub(mw.last)
	if elapsed <= 0 {
		return 0, 0
	}
	mw.last = now.Add(-(elapsed % time.Microsecond))
	if mw.stopped {
		return 0, 0
	}

	n := mw.clock.AdvanceMicros(int(elapsed / time.Microsecond))
	if bound := maxCatchUp(mw.clock.Period()); n > bound {
		n = bound
	}
	for i := 0; i < n; i++ {
		applied += mw.tick()
	}
	mw.view.SetPhase(mw.clock.Remainder(), mw.clock.Period())
	return n, applied
}

// setCadence is the ordinary-paced convenience used by cadence-focused tests.
// Production crosses setCadenceMode below, whose third scalar selects the
// separate owner-loop arm.
//
// The period arrives as a plain int and the stop as a plain bool. Neither names
// a simulation type, so a direct cadence test still cannot spell anything
// behind this driver.
//
// IT COMPUTES NOTHING. What arrives is already a tick length in
// microseconds, taken from the one cadence ladder on the statement that
// moved it, and this function adopts it verbatim. It used to divide a rate
// here, and the viewer's own setter divided the same rate again on the far
// side of the same statement — one number, two quotients — which is what
// made the shipped speeds unreachable: the game truncates 1000/16 to 62 ms
// and no microsecond quotient of 16 comes back to it.
//
// The clock is written on EVERY call, including one that only moved the stop. A
// cadence is a period and a switch beside it and there is no third state for "no
// cadence yet" to occupy, so the front-end's own cadence is what is in effect
// from the first call on — and because that cadence is now a rung of the same
// ladder the map opened on, a call that moved only the stop writes back the very
// period the clock was already holding. The accumulator's remainder survives the
// write, so a re-rate mid-run neither loses the fraction of a tick already
// elapsed nor fires a spurious one.
func (mw *mapWorld) setCadence(periodUS int, stopped bool) {
	mw.setCadenceMode(periodUS, stopped, false, false)
}

// setCadenceMode is MapCadence's production far side. In addition to the
// ordinary period and stop, it selects the genuine unpaced owner loop.
// Returning to paced, or an explicit clear command while already paced, resets
// both parts of this driver's cadence phase: the ticker remainder and the
// wall-clock baseline. The selected period and the logical counter survive, so
// the existing ladder/readout stay where the player left them and no animation
// rewinds.
func (mw *mapWorld) setCadenceMode(periodUS int, stopped, unpaced, reset bool) {
	mw.clock.SetPeriod(periodUS)
	mw.stopped = stopped
	if reset || mw.unpaced && !unpaced {
		mw.clock.ResetPhase()
		mw.last = time.Time{}
	}
	mw.unpaced = unpaced
}

// pushReadout hands the window tier what the readout states about the WORLD:
// the period this clock is holding, this advance's own stop, and the world's
// own tick and digest.
//
// ALL FOUR ARE READ HERE, on this statement, from the authority for each. The
// period comes off the very ticker paceTo divides elapsed time by, not from the
// rate the front-end last asked for and not from the viewer's water instance;
// the stop is the flag paceTo itself tests; the tick and the digest are the
// world's own. So a clamp, a truncation or a refusal anywhere below the key that
// asked for a change is already in what is pushed, because what is pushed was
// never the request.
//
// That is the whole of why this is not the cached copy the contract forbids.
// What it forbids is a value MAINTAINED beside an authority by the code that
// moves the authority; nothing here is maintained — it is re-derived on the
// frame path, and there is no statement anywhere that changes this clock and
// leaves this behind, because this is not on the changing path at all. A push
// that stopped happening freezes the readout visibly rather than letting it
// drift plausibly.
//
// THE DIGEST IS THE ONE EXPENSIVE READ and it is taken only for a viewer
// that reports the readout shown: it is a full encode of the world's byte
// form, hashed, and a front-end that hid the box should not be paying for a
// number nobody can see. Everything else here is a field read.
func (mw *mapWorld) pushReadout() {
	if !mw.view.ReadoutShown() {
		return
	}
	mw.view.SetReadout(ui.Readout{
		PeriodUS: mw.clock.Period(),
		Stopped:  mw.stopped,
		Tick:     mw.world.Tick(),
		Digest:   mw.world.Hash(),
	})
}

// commands is the stream one tick is advanced on: the schedule's entry for
// the world's own tick, with every command naming a COMMANDED entity cut out
// of it, and the pending orders spliced in AFTER what is left.
//
// THE INDEX IS THE WORLD'S OWN TICK, read here and never a counter this
// driver keeps. Reading w.Tick() is what makes "the schedule's entry for the
// world's tick as it stood before that step" the same statement on both
// sides. Past the schedule's end no command is applied AT ALL — not the
// last entry repeated, not an empty one padded on.
//
// THE EXCLUSION IS A LOOKUP PER SCRIPTED COMMAND and never a walk of the
// set. That is not a micro-optimisation: the set is a Go map, whose range
// order is randomised per process, so ranging it here would put a
// nondeterministic order into a command stream and, through Step's last
// write, into a world. Read by key, the stream stays a function of the
// schedule and the queue alone.
//
// THE SPLICE PUTS PENDING AFTER THE SCRIPT, and after is the direction that
// matters: Step applies commands in slice order with the last write winning,
// so a player's order behind the script wins a tick the two share. With the
// exclusion above in place, no world state can tell the two splice orders
// apart — the scripted command the order would have beaten is already gone
// — so what witnesses this direction is THIS SLICE and nothing else.
//
// WITH NEITHER SET NON-EMPTY THE SCHEDULE'S OWN SLICE IS RETURNED, uncopied.
// Both of the other forms are built FRESH, so no append reaches a schedule
// entry's spare capacity and no tick writes into an entry a later tick has
// still to apply.
func (mw *mapWorld) commands() []sim.Command {
	var cmds []sim.Command
	if t := mw.world.Tick(); t < uint64(len(mw.sched)) {
		cmds = mw.sched[t]
	}
	if len(mw.commanded) > 0 {
		kept := make([]sim.Command, 0, len(cmds))
		for _, c := range cmds {
			if !mw.commanded[c.Entity] {
				kept = append(kept, c)
			}
		}
		cmds = kept
	}
	if len(mw.pending) > 0 {
		joined := make([]sim.Command, 0, len(cmds)+len(mw.pending))
		joined = append(joined, cmds...)
		for i, c := range mw.pending {
			if i < len(mw.pendingIgnored) && mw.pendingIgnored[i] || mw.pendingEndpointAbsent(c) {
				continue
			}
			joined = append(joined, c)
		}
		cmds = joined
	}
	return cmds
}

// tick consumes pending commands once, records pre-step cells for interpolation,
// advances simulation, and projects the resulting state to the viewer. Queue
// clearing precedes Step; repeated draws cannot replay orders or consume motion.
func (mw *mapWorld) tick() int { return mw.tickWithCastSink(nil) }

// tickWithCastSink is tick with an optional read-only observation seam for the
// real-install no-window instrument. The sink sees the same immutable values
// observeCasts consumes and owns no driver or simulation state.
func (mw *mapWorld) tickWithCastSink(sink func([]sim.CastEvent)) int {
	return mw.tickStep(sink, true)
}

// tickStep is one driver tick; project false skips the viewer projection and
// keeps only the state a projection would have written, for a no-window
// instrument advancing several ticks before it reads the viewer.
func (mw *mapWorld) tickStep(sink func([]sim.CastEvent), project bool, reportSink ...func(sim.Report)) int {
	cmds := mw.commands()
	applied := len(mw.pending)
	mw.pending = mw.pending[:0]
	mw.pendingIgnored = mw.pendingIgnored[:0]
	mw.recordCells()
	mw.noteShotPreMoves()
	deadBefore := mw.world.OriginalDeadActorCount()
	// THE STEP REPORTS ITS APPLIED CASTS. StepObserved is the same step Step is
	// — the same code path with a sink handed down — and what comes back is
	// a value this tier keeps, never state on the world.
	report := sim.StepReported(mw.world, cmds)
	mw.observeScriptMessages(report.ScriptMessages)
	for _, observe := range reportSink {
		observe(report)
	}
	if deadAfter := mw.world.OriginalDeadActorCount(); deadAfter > deadBefore {
		mw.retainNewSourceDeadReferences(deadBefore)
	}
	mw.observeFame()
	// Damage observation is consumed before the scene clock advances. The
	// recorder stamps the NEXT scene, which is the one this tick will push, so
	// the first jolt phase is visible rather than already one tick old.
	mw.observeDamage(report.Damages)
	// A GiveUnit/GiveGroup instant becomes an ordinary hero in the same tick it
	// changes owner, before rearm, notice settlement or the viewer projection can
	// observe an in-between state.
	mw.syncJoinedHeroes()
	events := report.Casts
	born := len(mw.bolts)
	mw.observeScriptCasts(report.ScriptCasts)
	// A successful spell can raise a school level inside StepObserved. Fold
	// that new level through the complete installed character graph before any
	// later decision, save, hash or projection can observe a stale sheet.
	mw.recomputeRaisedSkills()
	mw.observeCasts(events)
	// AND ITS PAINTED STAGES (1003). A staged area effect sends one message per
	// accepted cell and the client builds one short-lived object from each, so
	// an earlier stage is still drawn while a later one lights up.
	mw.observeAreaPaints(report.AreaPaints)
	if sink != nil {
		sink(events)
	}
	mw.rearm()
	mw.refreshAppearance()
	mw.skillRises()
	// THE NOTICES ARE SETTLED HERE, once per step and immediately after it.
	// This is the granularity the latch array answers at: the pass writes it
	// and the very next statement reads it, so a repeating trigger that fires
	// and then does not inside one paced call is seen rather than lost. It
	// reads the world and writes nothing to it, so a run that shows notices and
	// a run that does not are the same run, tick for tick and digest for
	// digest.
	mw.settleNotices()
	mw.settlePickup()
	// The odometers advance for the tick that just ran, AFTER the step and
	// BEFORE the push, so what the viewer receives is selected from the
	// distance this tick produced and the push stays a pure read.
	mw.advanceOdometers()
	// The swing clocks advance for the tick that just ran, on the same terms
	// and in the same window as the odometers: AFTER the step, so the state
	// read is the state the push will select on, and BEFORE the push, so what
	// the viewer receives is selected from the count this tick produced.
	mw.advanceSwings()
	mw.advanceUnitShots()
	// The bolts age in the same window and for the same reason: after the step,
	// so the age read is the age the push will select on, and before the push,
	// so the viewer receives this tick's own.
	mw.advanceBoltsBornFrom(born)
	mw.advanceHealBursts()
	mw.advanceSpellSoundCues()
	// And the effect-mark elements, in the same window and for the same
	// reason: one rebuild is one tick here (1002; effectmark.go).
	mw.advanceEffectMarks()
	// The mission item grid advances on the same paced message as one world
	// tick (ITEM-STARPHASE-099). deterministicFrame and paceTo both enter this
	// function only for ticks they actually run, so ordinary pause holds the
	// visible-slot phases while repainting the existing frame.
	if mw.view != nil {
		mw.view.AdvanceInventoryStars()
	}
	// And the cast runs, in the same window: the frame counter they index is
	// advanced by advanceSwings immediately above, so the run's own countdown
	// comes down beside it.
	mw.advanceCastRuns()
	// One map-screen tick has run, so the scene clock advances exactly once,
	// BEFORE the push: what the viewer receives for the state this tick
	// produced is selected at this tick's scene, following the restored World
	// tick. It advances here and nowhere else; nothing canonical reads it.
	mw.scene++
	// Teleport arrivals reveal their surroundings before the same frame is
	// pushed. Ordinary updates retain the periodic world-tick cadence.
	if mw.world.Tick()%fogPeriod == 0 || mw.localTeleportArrived(events) {
		mw.fog.refresh(mw.world, sim.SelfSlot)
		mw.view.RefreshCheatFog()
	}
	if project {
		mw.push()
	} else {
		mw.pushHeldState()
	}
	return applied
}

// recordCells writes the step memory: every entity's current cell, keyed by
// its id, as the cell it will have STOOD ON before the advance that follows.
//
// It walks the world's own entity slice rather than the memory, so the write
// order is the world's ascending id and never a Go map's range order. An id the
// world has dropped keeps its last entry, which nothing reads: the push only
// looks up ids the world still hands back.
//
// AN ENTITY THAT OWES TRANSIT TICKS IS SKIPPED, and that one condition is
// the whole of what a crossing longer than a tick needs on this side. A
// mover takes its next cell at the crossing's first tick and then stands on
// it while it pays, so a memory written on every tick would record that cell
// on the second tick of the crossing and the derived step would collapse to
// zero for the rest of it — the unit would read as idle, face wherever it
// faced before, and jump between cells instead of walking. Skipped, the
// memory keeps the cell the crossing BEGAN at, and the step vector, the
// walking classification and the facing all hold until the next crossing
// starts.
//
// The transit is read BEFORE the advance, like every other value here: at
// the crossing's first tick the entity still owes nothing, so that cell is
// recorded and the crossing's own ticks are the ones skipped. THE CROSSING
// TICK IS COUNTED HERE, off the same condition and in the same walk. "Owes
// transit ticks" IS "the tick about to run continues the crossing this
// memory is holding a cell for", so the entry that keeps its cell keeps
// counting and the entry that gets a fresh cell starts a fresh crossing at
// tick 0. Writing it anywhere else would be a second reading of the same
// condition, free to disagree with this one.
func (mw *mapWorld) recordCells() {
	ents := mw.world.EntityView()
	for i := range ents {
		e := &ents[i]
		w := mw.walk[e.ID]
		if e.Transit > 0 {
			w.tick++
			mw.walk[e.ID] = w
			continue
		}
		w.tick = 0
		mw.walk[e.ID] = w
		mw.prev[e.ID] = image.Point{X: int(e.X), Y: int(e.Y)}
	}
}

// advanceOdometers pays each entity the distance it walked on the tick that
// has just run, and resets the count of one that stood still and has no idle
// cycle to draw.
//
// THE STEP IS DERIVED EXACTLY AS THE PUSH DERIVES IT — the entity's cell now
// less the cell the memory holds, under the memory's own PRESENCE test — so the
// unit the odometer pays is the unit the push draws walking, and there is one
// rule and not two kept in step. An id the memory has not seen has taken no
// step, which is what keeps an entity first observed part way through a crossing
// from being paid the whole distance from wherever the map's corner is.
//
// THE CROSSING IS READ FROM Transit AND THE COUNTED TICK, never from
// TransitTotal (walkClock). At tick k of an n-tick crossing the simulation still
// owes n-1-k, so k + 1 + Transit is n at every tick of it, and it is 1 for a
// mover that crosses a cell per tick whatever a stale total says.
//
// THE RESET IS THE ENGINE'S OWN FORK. A tick on which an entity covered no
// ground pins the count to zero for a class with no idle cycle and leaves it for
// a class with one, which is the engine's test on the same class scalar. It is a
// SECOND lookup of the bundle — the push resolves classes on the far side of
// this, and resolving here is what keeps the push a pure read — and it reads the
// entity's OWN class, never the corpse class the death path substitutes: the
// substitution is a fact about drawing a body, not about which unit walked. A
// class that resolves to nothing has no idle cycle and resets, the answer the
// seam gives a nil class everywhere else; its count is never read either way.
func (mw *mapWorld) advanceOdometers() {
	var classes map[int32]*terrain.UnitClass
	if mw.units != nil {
		classes = mw.units.Classes
	}
	for _, e := range mw.world.EntityView() {
		w := mw.walk[e.ID]

		var step image.Point
		if was, seen := mw.prev[e.ID]; seen {
			step = image.Point{X: int(e.X), Y: int(e.Y)}.Sub(was)
		}

		if step == (image.Point{}) {
			// Standing still — blocked, waiting, ordered nowhere, downed or
			// dead. A class that fidgets keeps its place in the walk; one that
			// does not restarts from the cycle's first step.
			if c := classes[e.Class]; c == nil || !c.Anim.IdleOK || len(c.Anim.IdleTrack) == 0 {
				w.dist = 0
			}
			mw.walk[e.ID] = w
			continue
		}

		w.dist += terrain.WalkAdvance(step.X, step.Y, w.tick+1+int(e.Transit), w.tick)
		mw.walk[e.ID] = w
	}
}

// advanceSwings counts one more tick for every entity that is drawing a
// swing and returns every other entity's count to zero.
//
// The condition is the PUSH'S OWN — holding a victim, and not moving under the
// step memory's own presence test — written here a second time rather than
// shared through a field, because a field would be state this tier holds between
// two functions and the conjunction is two reads of values both of them already
// have. What must not differ is the RULE, and it is one line in each place.
//
// It walks the world's own entity slice rather than the memory, so the write
// order is the world's ascending id and never a Go map's range order. An id the
// world has dropped keeps its last entry, which nothing reads: the push only
// looks up ids the world still hands back.
func (mw *mapWorld) advanceSwings() {
	// ONE VIEW, taken once. The emission below looks up the attacker's
	// victim in the same list; nothing in this walk mutates the world, so
	// the view needs no copy. It is also the RIGHT slice to look the victim
	// up in: the attacker and the target must be read from the same
	// instant, or a swing could be judged against a cell one of them no
	// longer stands on.
	ents := mw.world.EntityView()
	for i := range ents {
		e := &ents[i]
		was := mw.phase[e.ID]
		mw.phase[e.ID] = e.AttackPhase

		// A CAST RUN HOLDS THE CLOCK OPEN FOR A CASTER THAT HOLDS NO VICTIM
		// (`MAGIC-CASTANIM-029`): a book cast plays the caster's own Attack run
		// and a book cast sets no attack target, so without this the clock is
		// returned to zero on the very tick the run starts.
		if !e.Alive() || e.Turning() || (!e.HasAttackTarget && !mw.casting(e.ID)) {
			mw.swing[e.ID] = 0
			mw.shots.end(e.ID)
			continue
		}
		// A FRESH RUN AT THE SWING START, which is the tick the attacker leaves
		// the ready phase and loads its countdown — the instant the engine sends
		// the message that enters the drawn attack state. Every other tick of the
		// cycle advances the run this one began. A CASTING WIND-UP STARTS A RUN
		// EXACTLY AS A CHARGING ONE DOES. It did not, and that is the owner's
		// report that the staff animation does not correspond to the moments the
		// spell is applied: a mage whose weapon carries a spell loads
		// AttackCasting and never AttackCharging, so this counter was never reset
		// for one and simply ran free from the tick it acquired a victim. The
		// attack frame is selected by that counter and the release resolves on the
		// countdown, so the two clocks had no relation at all.
		//
		// THE TWO PHASES ARE THE TWO WIND-UPS and the run starts at whichever
		// one this tick entered (advanceAttack, pkg/sim/combat.go: a READY
		// attacker loads AttackCasting when weaponSpell says it is eligible and
		// AttackCharging otherwise). A DIVERT — loaded toward one and now the
		// other — is a change between the two and restarts the run, which is
		// right: the actor returns to READY and loads a fresh wind-up.
		if windUp(e.AttackPhase) && e.AttackPhase != was {
			mw.swing[e.ID] = 0
			mw.shots.begin(e.ID, e.AttackPhase)
		} else {
			mw.swing[e.ID]++
			// A run restored mid-wind-up has its clock and phase back but no
			// memory of which wind-up opened it: it continues as that phase.
			if _, held := mw.shots.run[e.ID]; !held && windUp(e.AttackPhase) {
				mw.shots.begin(e.ID, e.AttackPhase)
			}
		}

		// The swing sound rides this walk, which owns the counter: the counter
		// passes each value once per run, so testing the value it just took is
		// "once per run". The gate is the wind-up (charging or casting), not the
		// attack target, which also holds after the swing has left. The class's own
		// AttackDelay picks the tick; it is a separate number from the animation
		// cycle (ANIM-CLOCK-024). The attacker must also be able to reach its
		// victim: the cycle runs out of reach, and a sound that cannot match a blow
		// is not played. That narrows the voicing only; no phase, countdown or
		// hashed state changes. mw.swingSound is nil in hand-built test worlds.
		if windUp(e.AttackPhase) {
			if snd, ok := mw.sounds[mw.spellClientClass(e.ID, e.Class)]; ok && mw.swing[e.ID] == int(snd.AttackDelay) {
				// A weapon replacement is the cast arm, not a physical
				// swing. Its selector is emitted at the class AttackDelay
				// even when the spell's own range exceeds weapon reach.
				if e.AttackPhase == sim.AttackCasting {
					mw.playActorSound("weapon-cast-animation", castSpellSoundSlot(e.WeaponSpell), *e)
					continue
				}
				if mw.swingSound == nil && mw.semanticSound == nil || !mw.swingTargetInReach(ents, *e) {
					continue
				}
				// A SLOT OF 0 OR LESS IS SILENCE, and this is where that is
				// honoured rather than left to playSlotAt's own guard
				// (pkg/ui/sound.go) three calls away: "an emission" means a
				// play, and a class whose swing slot is silent (spec Terms;
				// three shipped classes ship exactly this) makes none —
				// AC-14's "once per run" would otherwise count a silent tick
				// as an emission it is not.
				if slot := swingSlot(snd); slot > 0 {
					mw.playActorSound("unit-swing", slot, *e)
				}
			}
		}
	}
}

// swingInReach reports whether a's attack victim is one this snapshot holds
// AND one a could actually strike from where it stands — sim.InReach, the
// exported spelling of the predicate resolveBlow itself consults, rather than
// a distance expression written a second time on this side of the wall.
//
// A VICTIM THE SNAPSHOT DOES NOT HOLD ANSWERS false, so nothing is voiced for
// it. That is the conservative arm of the same rule and not a separate one: a
// swing at an entity the world no longer carries can correspond to no blow,
// and advanceAttack ends such an order at the attacker's own next turn
// anyway. It is a linear scan because the caller's snapshot is a handful of
// entities and this walks it once per charging attacker; the alternative,
// asking the world per id, allocates a fresh slice copy every time.
// windUp reports that phase is one of the two an attacker LOADS A COUNTDOWN
// IN: AttackCharging, which resolves a blow, and AttackCasting, which releases
// a weapon-borne spell (advanceAttack, pkg/sim/combat.go). It is the drawn
// attack state — the swing clock's run and the swing's own voicing both key off
// it — and it is written once here so the two cannot come to disagree about
// which phases are a swing.
func windUp(p sim.AttackPhase) bool {
	return p == sim.AttackCharging || p == sim.AttackCasting
}

func swingInReach(ents []sim.Entity, a sim.Entity) bool {
	if a.AttackTargetKind != sim.AttackTargetUnit {
		return false
	}
	for _, t := range ents {
		if t.ID == a.AttackTarget {
			return sim.InReach(a, t)
		}
	}
	return false
}

func (mw *mapWorld) swingTargetInReach(ents []sim.Entity, a sim.Entity) bool {
	if a.AttackTargetKind == sim.AttackTargetStructure {
		for _, s := range mw.world.Structures() {
			if uint32(s.ID) == uint32(a.AttackTarget) {
				return sim.InStructureReach(a, s)
			}
		}
		return false
	}
	return swingInReach(ents, a)
}

func (mw *mapWorld) attackTargetCell(a sim.Entity) (image.Point, bool) {
	if a.AttackTargetKind == sim.AttackTargetStructure {
		for _, s := range mw.world.Structures() {
			if uint32(s.ID) == uint32(a.AttackTarget) {
				return image.Pt(int(s.Col), int(s.Row)), true
			}
		}
		return image.Point{}, false
	}
	t, ok := mw.entity(a.AttackTarget)
	return image.Pt(int(t.X), int(t.Y)), ok
}

// swingSlot is a class's own swing sound — index 0 of its Sound array — or 0
// for silence: a nil or short array and a zero element all fold to the same
// answer (spec Terms "Sound slots of a class": "A zero element is silence
// and not an error ... An array shorter than an index, or absent
// altogether, is silence by the same rule"), pkg/ui's own classSlot applying
// the identical rule one package over (sound.go, T2).
func swingSlot(snd UnitSound) int {
	if len(snd.Slots) == 0 {
		return 0
	}
	return int(snd.Slots[0])
}

// setSwingSound installs the per-class sound table and the map's positional
// swing/spell play callback: sounds is LoadUnitSounds' own answer (sound.go)
// and play is (*ui.Viewer).PlaySlotAt, bound to the running map's own viewer
// — frontend.go's loadMap and MissionOpener are the two production
// callers, at the same two sites that hand that viewer its font.
//
// IT IS A SETTER AND NOT A CONSTRUCTOR ARGUMENT, on purpose. newMapWorldWith
// already takes six positional arguments and close to forty call sites
// across this package's own tests build a mapWorld through it or through
// newMapWorld, its narrower wrapper; widening either for two values only
// openMapWorld's and openMission's production caller ever supplies would
// touch every one of those tests for a feature none of them exercises.
// Called from neither constructor, both fields this method fills stay at
// their zero value — nil — for every one of them, and advanceSwings' own
// nil guards make that a silent map rather than a broken one.
func (mw *mapWorld) setSwingSound(sounds map[int32]UnitSound, play func(slot int, owner uint32, cell image.Point)) {
	mw.sounds = sounds
	mw.swingSound = play
	mw.spellSound = play
}

// pushHeldState is what push writes besides the viewer's replaceable lists:
// the scorch and light clocks (each keeps history across calls) and the
// first-seen-dead stamps entityDraws records. A tick that
// skips the projection still runs these, so the state after a run of skipped
// ticks and one projection equals the state after a projection every tick.
func (mw *mapWorld) pushHeldState() {
	ents := mw.world.EntityView()
	for i := range ents {
		e := &ents[i]
		if e.Alive() {
			delete(mw.died, e.ID)
		} else {
			mw.observeDeath(e.ID)
		}
	}
	mw.view.SetScorchedCellsAt(mw.world.ScorchedCells(), uint64(mw.world.Tick()))
	mw.view.SetLightClock(mw.world.Tick())
}

// push hands the viewer this world's entities as they stand now: one
// ui.MapEntity per entity, cell beside art, in the order the world hands
// them back.
//
// The slice is freshly built on every call and handed over rather than kept:
// the setter adopts it, so nothing on this side still refers to what the
// renderer holds, and the previous tick's slice is dropped rather than
// accumulated. THE LIGHTING CLOCK RIDES THIS STATEMENT, and it is INSIDE
// push rather than beside push's callers because there are two of them —
// the constructor and the tick. Beside them it would be two lines free to
// drift; here, "the sun's clock is the world's tick" is one line.
//
// Both callers are load-bearing. The tick's is the cycle itself. The
// constructor's is why a mission opens already lit for its opening minute rather
// than for the viewer's seed — a map opened and left stopped never ticks, and
// tick 0 is a relight instant, so the opening push is the one that lights it.
//
// THE WORLD'S TICK IS THE SUB-TICK, unconverted: it is the same unit the mission
// script and the engagement decision already take modulo 16, and handing it over
// raw is what keeps this seam carrying one number rather than a conversion two
// sides could disagree about.
//
// And it rides the PER-TICK push, not a per-frame one, for a reason rather than
// for symmetry: a paced call runs up to maxCatchUp whole ticks in one frame, so
// a frame-level push of only the last tick's clock would step straight over a
// relight instant and hold the sun still for a whole period.
//
// It is a pure read of the world and writes nothing to it, so a run that draws
// and a run that does not are the same run, tick for tick and digest for digest.
func (mw *mapWorld) push() {
	mw.pushSavedStructureRoster()
	draws := mw.entityDraws()
	if len(mw.pendingDamage) != 0 {
		byID := make(map[uint32]ui.MapEntity, len(draws)+len(mw.soundEntities))
		for _, draw := range mw.soundEntities {
			byID[draw.ID] = draw
		}
		for _, draw := range draws {
			byID[draw.ID] = draw
		}
		messages := make([]ui.DamageMessage, 0, len(mw.pendingDamage))
		for _, event := range mw.pendingDamage {
			if draw, ok := byID[uint32(event.Target)]; ok {
				draw.HP = int(event.BeforeHP)
				messages = append(messages, ui.DamageMessage{Entity: draw, AfterHP: int(event.AfterHP)})
			}
		}
		mw.view.AppendDamageMessages(messages)
		mw.pendingDamage = nil
	}
	mw.view.SetEntities(draws)
	mw.soundEntities = draws
	mw.view.SetUnitInspectionPictureSource(mw.inspectionUnitPicture)
	mw.view.SetStructures(mw.structureDraws())
	mw.view.SetScorchedCellsAt(mw.world.ScorchedCells(), uint64(mw.world.Tick()))
	mw.view.SetLightClock(mw.world.Tick())
	mw.view.SetSpellLighting(mw.spellLighting())
	mw.view.SetAmbientWallFire(wallFireAmbientCells(mw.cellEffectsByArm()))
	mw.view.SetSacks(mw.sackDraws())
	mw.view.SetFog(mw.fog.project(), mw.fog.cols, mw.fog.rows)
	// THE SPELLBOOK, beside SetSacks and SetFog on push's own reasoning: the
	// selected unit's own book can change on any frame that carries no tick at
	// all, exactly as the sacks and the fog can.
	mw.pushSpellbook()
	// THE SELECTED UNIT'S FLAT PORTRAIT, beside the book and for its reasons
	// (0141; `UNIT-PICT-035`): what is selected changes on frames that carry
	// no tick, and this picture follows the selection rather than the party's
	// own subject.
	mw.pushPortrait()
	// THE SPELLS IN FLIGHT, last of the per-frame pushes: the book-cast bolts
	// this tier remembers and the weapon-borne ones it reads live off the
	// attack cycle, both resolved fresh from the world's own entities so a
	// frame with no tick behind it draws the same picture.
	ents := mw.world.EntityView()
	mw.view.SetSpellBolts(mw.boltDraws(ents))
	mw.view.SetLightStamps(mw.objectLightStamps(ents))
	mw.view.SetHealSprites(mw.healSpriteDraws())
}

// wallFireAmbientCells projects canonical spell-3 coverage into the
// presentation-only loop seam. Viewer de-duplicates overlaps, so this read
// cannot mutate or normalise the world.
func wallFireAmbientCells(effects []sim.CellEffect) []image.Point {
	var out []image.Point
	for _, effect := range effects {
		if effect.Spell != 3 {
			continue
		}
		for _, cell := range effect.Cells {
			out = append(out, image.Pt(int(cell[0]), int(cell[1])))
		}
	}
	return out
}

func (mw *mapWorld) structureDraws() []ui.MapStructure {
	structures := mw.world.Structures()
	out := mw.structureStateScratch[:0]
	for _, s := range structures {
		out = append(out, ui.MapStructure{ID: uint32(s.ID), Health: s.Field42,
			MaxHealth: s.MaxHealth, Cell: image.Pt(int(s.Col), int(s.Row))})
	}
	mw.structureStateScratch = out
	return out
}

const (
	lightSpellID           = 12
	darknessSpellID        = 17
	lightTerrainBrightness = float32(3)
	lightSpriteBrightness  = float32(2)
	darknessBrightness     = float32(0.5)
)

// spellLighting derives the presentation plane from live area records. The
// simulation's layer-conflict and expiry rules remain its single canonical
// state; a push rebuilding this list restores ordinary light without residue.
func (mw *mapWorld) spellLighting() []ui.SpellLightCell {
	return spellLightingCells(mw.cellEffectsByArm(), mw.world.Bounds())
}

// cellEffectsByArm is the live area records with each spell id replaced by the
// arm its row runs, so a second-game row lights and burns as its own arm.
func (mw *mapWorld) cellEffectsByArm() []sim.CellEffect {
	effects := mw.world.CellEffects()
	for i := range effects {
		effects[i].Spell = mw.world.SpellArm(effects[i].Spell)
	}
	return effects
}

// spellLightingCells is the Light and Darkness population of the cell-bit
// stage (MAGIC-UNITLIGHT-057): each writes its cell-local actor level and the
// separate terrain value carried to the shared vertex plane. Wall of Fire is a
// light-stamp source instead (wallFireLightStamps). Every other spell is
// absent from the cell-bit dispatch.
func spellLightingCells(effects []sim.CellEffect, bounds sim.Bounds) []ui.SpellLightCell {
	var out []ui.SpellLightCell
	indices := make(map[image.Point]int)
	put := func(cell image.Point, terrainBrightness, spriteBrightness float32) {
		if cell.X < 0 || cell.Y < 0 || cell.X >= int(bounds.Width) || cell.Y >= int(bounds.Height) {
			return
		}
		if i, ok := indices[cell]; ok {
			if terrainBrightness > 0 {
				out[i].Terrain = terrainBrightness
			}
			// Max on the gain scale makes an intersecting Light/Darkness cell
			// independent of effect-record order.
			if spriteBrightness > out[i].Sprite {
				out[i].Sprite = spriteBrightness
			}
			return
		}
		indices[cell] = len(out)
		out = append(out, ui.SpellLightCell{Cell: cell, Terrain: terrainBrightness, Sprite: spriteBrightness})
	}
	for _, effect := range effects {
		terrainBrightness, spriteBrightness := float32(0), float32(0)
		switch effect.Spell {
		case lightSpellID:
			terrainBrightness, spriteBrightness = lightTerrainBrightness, lightSpriteBrightness
		case darknessSpellID:
			terrainBrightness, spriteBrightness = darknessBrightness, darknessBrightness
		default:
			continue
		}
		for _, cell := range effect.Cells {
			put(image.Pt(int(cell[0]), int(cell[1])), terrainBrightness, spriteBrightness)
		}
	}
	return out
}

// wallFireLightLevel is the level a Wall of Fire cell stamps, the flicker
// between 0 and 12 seeded by scene/2 + worldX*worldY (TERR-LIGHT-061). This
// build selects level 0 when abs(seed)/5 is even and 12 when it is odd.
func wallFireLightLevel(scene int, cell image.Point) uint8 {
	value := scene/2 + cell.X*cell.Y
	if value < 0 {
		value = -value
	}
	if (value/5)&1 == 0 {
		return 0
	}
	return 12
}

// entityDraws is the world's entities as the window tier receives them —
// the entity's own id, a plain map cell and, where the entity's class id
// resolves, the render-tier art beside the SELECTED frame and mirror bit;
// nil art and frame for one that draws as the square — one ui.MapEntity
// each, in the order the world hands them back, which is ascending id and so
// the map's own unit-slice order.
//
// THE ID IS MINTED HERE AND NOWHERE ELSE. It is the simulation's own
// sim.EntityID, widened to the uint32 the seam carries — the id's own
// width, so the conversion loses nothing in either direction — and it is
// deliberately NOT the loop index: a window tier handed slice positions
// would address its orders at whatever happened to be that far down the
// snapshot, which is the same value only while a world has never lost an
// entity. The window tier cannot construct one, so an id it names is always
// an id that crossed here.
//
// THE PUSH CLASSIFIES OFF THE STEP THE ENTITY TOOK, not off the order it
// holds: the step is this entity's cell now minus the cell the memory
// recorded before the last advance, and the entity is moving exactly when
// that delta is nonzero. The order was the wrong quantity — pkg/sim sets a
// target and clears it on arrival inside ONE advance, so an order to an
// adjacent cell is in force at no tick boundary at all and neither is the
// last step of any longer walk, and both were drawn idle facing wherever the
// unit faced before.
//
// A mover's sign-octant is WRITTEN to the facing memory; an idle entity READS
// its own entry, octant 0 if it never moved. Both memories are looked up by id
// and NEVER iterated, so a Go map's nondeterministic range order cannot reach
// the viewer through any path here, and the write is idempotent under a
// repeated build. Classification runs for every entity, resolved or not: the
// memories are the seam's record of what the world did, not of what happened
// to draw.
//
// AN ENTITY THE MEMORY HAS NOT SEEN HAS TAKEN NO STEP — the constructor's
// tick-0 push, and an entity appearing in a snapshot for the first time. That
// is presence answering, not a zero cell: a previous CELL of (0,0) would read
// as a unit that just walked in from the map's corner.
//
// THE DELTA CROSSES THE SEAM as ui.MapEntity.Step, zero for an entity that
// did not move, and the window tier draws the entity along it. Nothing about
// it is world state: it is derived here, per push, from a memory pkg/sim
// cannot see.
//
// SELECTION IS REACHED ONLY THROUGH THE RENDER TIER'S OWN SELECTIONS, and
// there is one per life state. A living entity takes SelectUnitFacingFrame at its
// effective tick, scene + int(id) — a cosmetic de-sync from deterministic
// snapshot data — AND at its own walk odometer, which is what the moving
// arm actually reads. The two clocks are the engine's own asymmetry:
// everything but the walk advances by one per tick, and the walk advances by
// ground covered. Neither is state beyond the seam's own. One that is not
// alive takes SelectDeathFrame at its own elapsed count. THIS FUNCTION IS
// THE ONE PLACE THAT CHOOSES between them — the only site holding both the
// life state and the class — so there is one rule and not two kept in
// step.
//
// WHICH TIER'S COLOURS a resolved class is drawn in is decided here too, and
// by one lookup read by both selections. A tier is a colour table over the
// same frames, so it moves neither the selection, the mirror, the anchor nor
// the count — only which slice the selected index is taken out of — and
// the seam therefore does not widen: what crosses is still a class and a
// frame.
//
// RESOLUTION HAPPENS HERE, in the one tier that sees both a world and a
// bundle: Art is the bundle's own entry exactly when that entry holds a
// frame, and nil otherwise — a frameless entry (an excluded class), an id
// naming no class, and a nil bundle all cross as nil, because at this seam
// every one of them draws the same square. The lookup reads the set and the
// world and mutates neither of them: the world is read through the
// copy-handing Entities, so resolving cannot write back; what it does write
// is the seam's own facing memory, which nothing canonical reads.
//
// IT PERFORMS NO CELL ARITHMETIC. An entity's X and Y are already the whole
// cell it stands on; this widens two int32s to int and nothing else. The
// delta the classification reads is target minus position, cells against
// cells — a direction, not a placement.
//
// THE LIFE STATE AND THE HEALTH PAIR CROSS AS THE WORLD'S OWN ANSWER. The
// classification is made HERE, through pkg/sim's own three predicates, so the
// rule that says what alive, downed and dead mean exists once, in the package
// that owns the two integers it is read off. The window tier receives the
// answer; it cannot re-derive one, and there is nothing there for a re-derivation
// to disagree with. The pair rides beside it because a health bar needs the
// ratio, and both are read through the same copy-handing entity read as
// everything else here, so nothing this function does can write back.
//
// A CORPSE STILL CROSSES, art and all, and no drawn thing is dropped here.
// It crosses as the CORPSE CLASS'S art beside a dying frame, so the seam
// does not widen by one field and every glyph, placement, cull and texture
// path downstream receives exactly what it always received.
//
// ITS FACING IS FROZEN BY CONSTRUCTION. pkg/sim does not advance an entity
// that is not alive, so a body's step is (0, 0) at every later tick; it
// therefore reads the facing memory rather than writing one, and the
// direction it died in is the direction it keeps. Nothing here had to
// arrange that, and the criterion exists to keep it true rather than to make
// it so.
//
// An entity whose cell lies outside the map is passed on unchanged, art and
// all. A world permits one to walk off the grid, and dropping it here would
// be this tier deciding what is drawable; the render tier's own off-map
// rejection is what leaves it undrawn. It is the drawing seam's own
// simplified distance test and not a copy of the simulation's strikeDistance
// (combat.go, unexported): that routine also folds in a footprint term
// neither this seam nor its spec reads, and the gate here decides only
// whether a mark is worth drawing, never whether a blow can land.
func chebyshevDist(dx, dy int) int {
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// soundSlots is class's sound-slot array, widened to []int for the entity
// seam (0126 spec Terms; plan T3): ui.MapEntity.Sound is a plain []int —
// Speed's own shape, one door over — so this package's own []int32
// (UnitSound.Slots, sound.go) is converted here, once, in the one place
// both types are in scope.
//
// A CLASS THE TABLE DOES NOT NAME RESOLVES TO NIL, sounds itself nil
// among them (a front-end that never wired setSwingSound, or one whose
// LoadUnitSounds came back empty): nil crosses exactly like every entity
// built before this story, and ui.MapEntity's own comment states that a
// nil slice answers "silence at every index" with no caller having to
// tell the two apart.
func soundSlots(sounds map[int32]UnitSound, class int32) []int {
	snd, ok := sounds[class]
	if !ok || len(snd.Slots) == 0 {
		return nil
	}
	out := make([]int, len(snd.Slots))
	for i, s := range snd.Slots {
		out[i] = int(s)
	}
	return out
}

func shotPoint(from, to image.Point, num, den int) image.Point {
	return image.Point{
		X: from.X*ui.ShotScale + (to.X-from.X)*ui.ShotScale*num/den,
		Y: from.Y*ui.ShotScale + (to.Y-from.Y)*ui.ShotScale*num/den,
	}
}

func (mw *mapWorld) entityDraws() []ui.MapEntity {
	var classes map[int32]*terrain.UnitClass
	if mw.units != nil {
		classes = mw.units.Classes
	}
	ents := mw.world.EntityView()
	draws := make([]ui.MapEntity, len(ents))
	// READ ONCE FOR THE WHOLE TICK (1031 B3), not once per entity: Relations
	// materialises its whole matrix on every call, and every entity's own
	// Hostile field below is the local participant's relation toward the same
	// small set of owners. The word table is read once for the same reason.
	relations := mw.world.Relations()
	spells := mw.world.Spells()
	var unitNames []string
	if mw.view != nil {
		names := mw.view.Words().UnitNames
		unitNames = names[:]
	}
	for i, e := range ents {
		if e.Alive() {
			delete(mw.died, e.ID)
		}
		cell := image.Point{X: int(e.X), Y: int(e.Y)}
		fineX, fineY, finePosition := mw.world.ActorFinePosition(e.ID)

		var step image.Point
		if was, seen := mw.prev[e.ID]; seen {
			step = cell.Sub(was)
		}
		moving := step != image.Point{} || mw.world.ActorMotionActive(e.ID)
		if mw.drawnMoving == nil {
			mw.drawnMoving = make(map[sim.EntityID]bool)
		}
		mw.drawnMoving[e.ID] = moving

		// Keep the entity's own class before corpse-art substitution. Display
		// captions below use its type or an explicit actor name.
		c := classes[e.Class]
		category := terrain.UnitCategoryFor(c, uint8(e.Decay))

		// AND THE SECOND RULE, in one statement and before anything reads the
		// result. An actor a map places is drawn as the class its record names,
		// which is the lookup above; a player's character is drawn as whatever its
		// visible equipment says, from a sheet composed out of a name, which is
		// this. One code path cannot serve both, and putting the choice HERE is
		// what makes the name, the live selection, the swing and the fall all
		// follow from it by construction rather than by four edits that agree
		// today.
		//
		// It replaces the class WHOLE, as the death path's substitution does and
		// for the same reason: a body is a canvas, a descriptor, a name, a
		// corpse link and a sheet, and taking any of them from the other side
		// would draw one class's pixels through another class's geometry.
		//
		// An id with no entry keeps the lookup above, so this is inert for every
		// entity the map placed.
		if body := mw.art[e.ID]; body != nil {
			c = body
		}

		name := ""
		if c != nil {
			name = c.Name
		}
		if saved, ok := mw.actorNames[e.ID]; ok {
			name = saved
		}

		// THE TWO EXPERIENCE-BEARING MEMBERS ARE OVERLAID FROM THE ENTITY OF THIS
		// TICK, not from what the load computed — Combat below already reads e
		// live for the same reason, and this is that same rule reaching the two
		// fields on Char that move after a mission starts. The four statistics and
		// the two families stay exactly what mw.chars gave them; Experience is
		// replaced off e.SkillXP, except for a retained original aggregate whose
		// six source XP slots still match (1099). Guarded on Known so a unit the
		// load knows no character for does the work for nothing: its zero-value
		// Char already reads as "no character" and this loop could not change
		// that. That set USED TO BE every unit this tree did not place from a
		// party and is not any more — it is now only a placement that reached no
		// definition entry at all.
		//
		// THE WEAPON NAME IS NEITHER OVERLAID HERE NOR STABLE. It used to be
		// listed above as one of the members that "stay exactly what mw.chars
		// gave them", and that sentence is why an equipped weapon's numbers
		// moved in the panel while its name did not: it asserted a stability
		// nothing was maintaining. rearm now writes the resolved name into
		// mw.chars at the moment the inventory subject's equipment changes
		// (world.go), so the map itself moves for that one field and this loop
		// reads the current name by reading the map — no second resolve on a
		// tick, and no per-entity work for the entities that cannot re-arm.
		ordinary := (e.SourceBinding.ActorClass() == 1 || !sim.InPersistBand(e.TypeID)) &&
			mw.missionPartyMember(e.ID) == nil
		definitionName := ""
		if ordinary && mw.mission != nil {
			definitionName = sourceActorDefinitionName(mw.mission.table, e.SourceBinding)
		}
		sourceClassName := definitionName != "" && mw.actorNames[e.ID] == definitionName
		char := mw.chars[e.ID]
		if ordinary && definitionName != "" && char.Name == definitionName {
			char.Name = ""
		}
		if char.Name != "" {
			name = char.Name
		}
		if char.Known {
			for k := range char.Protection {
				char.Protection[k] = int(e.Protection[k])
				char.Resistance[k] = int(e.Resistance[k])
			}
			var total int
			for slot, xp := range e.SkillXP {
				if char.Band != ui.CharacterBandCreature {
					char.Skills[slot] = int(e.Skill[slot])
				}
				total += int(xp)
			}
			char.Experience = mw.retainedHumanExperience(e, total)
		}

		spell, weaponSpellKnown := sim.WeaponSpellCharacteristicsFor(mw.world.Rules(), e, spells)
		spellBase, spellSpread := int64(0), int64(0)
		if weaponSpellKnown && spell.HasDamage {
			spellBase, spellSpread = spell.DamageMin, spell.DamageMax-spell.DamageMin
		}
		nameIndex := int(e.TypeID)
		nativeClassName := e.SourceBinding.Class == 0 && e.ActorLoad.Source.Class == 0 &&
			e.NativeBasis.HasValues() && c != nil && mw.actorNames[e.ID] == c.Name
		if !ordinary && char.UnitNameIndex != 0 {
			nameIndex = char.UnitNameIndex
		}
		if (ordinary || char.Known) && char.Name == "" && (mw.actorNames[e.ID] == "" || nativeClassName || sourceClassName) {
			if ordinary {
				name = ""
			}
			if nameIndex >= 0 && nameIndex < len(unitNames) && unitNames[nameIndex] != "" {
				name = unitNames[nameIndex]
			}
		}
		playerCharacter := mw.isGuarded(e.ID)
		panelXPValue := e.XPValue
		if e.TypeID >= 0 && e.TypeID < 0x1a {
			panelXPValue = originalHumanPanelXPValue(e.SkillXP)
		}
		clientClass := mw.spellClientClass(e.ID, e.Class)
		draws[i] = ui.MapEntity{ID: uint32(e.ID), UnitNameIndex: nameIndex,
			DrawCategory:    category,
			SpellStateKnown: true, KnownSpells: e.KnownSpells,
			CastCapable: (clientClass == 0x17 || clientClass == 0x18) && e.KnownSpells != 0,
			Cell:        cell, Step: step, Route: mw.routeOf(e.ID),
			Name: name, Life: lifeOf(e), HP: int(e.HP), MaxHP: int(e.MaxHP),
			Untargetable: !e.OrdinaryTargetable(),
			// Life keeps the corpse rendering/depth answer for negative HP.
			// Selectable is the independent owner-authored exception through -9;
			// the simulation still refuses every move from a not-alive actor.
			Selectable: e.MaxHP > 0 && e.Dead() && e.OrdinaryTargetable(),
			// The population ordinary Heal may target (DIV-033): a positive
			// maximum and health above the finished-body floor. The minimap
			// drops a fallen unit's mark once this is false (DIV-1455).
			Restorable: e.MaxHP > 0 && e.OrdinaryTargetable(),
			DamageJolt: mw.damageJolt(e.ID),
			Stone:      mw.world.HasEffectArm(e.ID, 20), Translucent: mw.world.HasEffectArm(e.ID, 15),
			// The mana pair beside it, read off the same copy-handing entity read as
			// everything else here and carried whole — no period, no remainder: the
			// panel states a pool, not a rate.
			Mana: int(e.Mana), MaxMana: int(e.MaxMana), TokenSize: int(e.TokenSize),
			// THE SPELL EFFECT MARK: the ticks the simulation says still stand, and
			// the SCHOOL of the spell that set it — resolved here, off the world's
			// own table, because pkg/ui may not name a spell row. A mark whose spell
			// no longer resolves carries school 0, which draws the palette's own
			// no-school colour rather than dropping the mark.
			SpellFX: int(e.SpellFX), SpellFXSchool: spellSchool(uint16(e.SpellFXSpell), spells),
			// The roster slot the map placed it under, carried whole for the arming
			// gate to compare and interpreted nowhere on either side. It is the
			// simulation's own field at its own width — zero for an entity no map
			// placed, which is every entity this tree spawns.
			Owner:     e.Owner,
			Knowledge: mw.cardKnowledge(e), KnowledgeKnown: true,
			// THE LOCAL PARTICIPANT'S OWN RELATION TOWARD IT (1031 B3;
			// UNIT-VPLAYER-021, UNIT-VISBIT-044), read off the same relation
			// matrix `pkg/sim/engage.go` already reads for engagement — not
			// re-derived here, so a hostility rule changing has one place to
			// change. `sim.SelfSlot` is the same local-participant slot
			// `flow.go`'s own `v.SetLocalOwner(sim.SelfSlot)` already names.
			Hostile: relations.Hostile(sim.SelfSlot, e.Owner),
			// THE MISSION CURSOR'S PLAYER-CHARACTER GATE (`PARTY-FLAG-003`, Medium,
			// read as `CUnit+0x18c` bit `0x1`). The mission-loss set is already
			// exactly "every party member who is a player character and not a
			// mercenary" — guardedEntities' own two conditions — so the flag is
			// read off it rather than recomputed from the party a second time. A
			// mission opened from the picker has an empty guarded set and every
			// entity answers false, which is the state this build had before the
			// field existed.
			PlayerCharacter: playerCharacter,
			// The exact four-field actor projection read by the two conditional
			// installed captions. It deliberately does not reuse Mage or
			// AlwaysHits: neither is part of either original predicate.
			OriginalPanel: originalPanelActor(playerCharacter, e.TypeID, panelXPValue),
			// THE EIGHT NUMBERS A BLOW READS, for the unit panel to state. They are
			// read off the entity on the tick they are pushed, through the same
			// copy-handing entity read as everything else here, so the panel states
			// the simulation's own answer and never a value accumulated beside it.
			// Known is set for every entity of a running world, because every entity
			// of a running world has them.
			Combat: ui.UnitCombat{Known: true,
				DamageBase: int(e.DamageBase), DamageSpread: int(e.DamageSpread),
				WeaponSpellKnown:       weaponSpellKnown,
				WeaponSpellDamageKnown: weaponSpellKnown && spell.HasDamage,
				SpellDamageBase:        spellBase, SpellDamageSpread: spellSpread,
				ToHit: int(e.ToHit), Defence: int(e.Defence), Absorption: int(e.Absorption),
				AttackCharge: int(e.AttackCharge), AttackRelax: int(e.AttackRelax),
				AlwaysHits: e.AlwaysHits},
			// AND THE CHARACTER THEY WERE DERIVED FROM, where the load knew one. An
			// id with no entry states none, which is the tier lookup's own rule —
			// and the set that misses is no longer "every unit a map placed": a
			// placement reaching a definition entry now states its row's own sheet,
			// and only one reaching none misses. Two of its members are then overlaid
			// live — char, built just above.
			Char: char,
			// The speed input, carried whole for the readout to state and interpreted
			// nowhere on either side. It is read through the same copy-handing entity
			// read as everything else here, so it is the simulation's own answer and
			// not a second derivation.
			Speed: humanSpeedStat(e),
			// The group rate term beside it, carried and never composed with it.
			// Which of the two actually moves the entity is a rule this package does
			// not hold and must not restate: the seam that composes them is the
			// simulation's own, and the readout states both numbers precisely so that
			// the composition can be read rather than copied.
			GroupSpeed: int(mw.world.GroupRateTerm(e.ID)),
			// The carried load, on the speed's own terms: the simulation's own field,
			// carried whole for the sheet to state and interpreted nowhere on either
			// side. The overload penalty it feeds is applied inside the simulation
			// and is already inside e.Speed's consumers, so nothing here composes the
			// two.
			Load: int(e.Load),
			// The crossing this step is part of, carried whole so the window tier can
			// run the displacement over it. Both are the simulation's own, read
			// through the same copy-handing entity read as everything else here, and
			// neither is state of this seam's.
			Transit: int(e.Transit), TransitSpan: int(e.TransitTotal),
			FinePosition: finePosition, FineX: fineX, FineY: fineY,
			// The live client class's own Sound array, including an
			// equipment-derived hero class; corpse art does not participate.
			// A human's wounds and fall take his voice bank instead, whatever
			// that class is (ANIM-096).
			Sound: soundSlots(mw.sounds, clientClass), Voice: mw.voiceBank(e),
			CorpseStage: uint8(e.Decay),
			// THE DECODED EFFECT MARKS (1002; MAGIC-MARK-059,
			// MAGIC-MARK-060). Rebuilt every tick from the client's own
			// element list, each record beside the sheet its record index
			// named, and scaled by the actor class's own units.reg
			// TileSize. A class the bundle does not answer for scales by 1,
			// which is that key's own registry default.
			Marks: mw.markDraws(e.ID, markTileSize(classes[e.Class]))}

		// ANIM-DIR-006
		facing := ((int(e.DrawnFacing()) >> 4) + 8) & 15
		oct := facing >> 1

		// The death path, and it is the first thing tried for an entity the world
		// reports not alive — downed and dead alike. The frames come out of the
		// CORPSE class the loader resolved: its sheet, its descriptor and its
		// canvas, since the substitution is a whole class. The elapsed count
		// carries NO per-entity offset, unlike the live selection's effective
		// tick: a fall begins at its own first frame.
		//
		// THE TIER IS THIS PLACEMENT'S, looked up once and read by both arms
		// below. It selects WHICH FRAME SLICE of whichever class supplies the
		// frames — the entity's own here, the corpse class's on the death path
		// — so a substitution can never draw one class's pixels through another
		// class's colours. An id with no entry states no tier, which the selector
		// answers the sheet's own frames for.
		tier := mw.tiers[e.ID]

		// A SWING IS DRAWN FIRST among the living selections, and the test is
		// holding a victim and not `moving`. `moving` is the step memory's, which
		// holds its cell for the whole of a crossing — so it means "is
		// mid-stride, or moved", and an attacker walking to a distant victim draws
		// its walk rather than a swing. In the engine the two states are exclusive
		// by construction, one action code being current at a time; here they are
		// two independent readings, and this is what orders them.
		//
		// It does NOT ask whether the victim is in reach. The reach is a
		// simulation law behind an unexported predicate, and a copy of it here is
		// what the crossing and the rate composition are already refused for; the
		// cost is that an attacker standing still and OUT of reach — blocked, or
		// its victim fled — draws a swing too.
		//
		// WHETHER THE RUN IS STILL PLAYING is not asked here either: the swing
		// clock passes the track's length and the selection refuses, which is the
		// engine's own "the run counter reached zero, force the state to 0" seen
		// from the fall-through side.
		//
		// A unit that is not alive holds no victim, so this can never contend with
		// the death path below: a blow clears the order it fells a unit out of.
		// The ordering is still written as it is, death first, because that is the
		// contract and not an accident of which flag happens to be set. AND A BOOK
		// CASTER SWINGS TOO, for as long as its cast run stands:
		// `MAGIC-CASTANIM-029` fixes that a cast plays the caster's own Attack
		// run, once, one frame per tick, and that run is the same track
		// SelectAttackFrame already indexes for a swing.
		swinging := e.Alive() && !e.Turning() && (e.HasAttackTarget || mw.casting(e.ID)) && !moving

		// THE ORANGE MARK IS THE FALLBACK for a shot this build cannot draw: a
		// physical ranged swing whose class names no projectile, or one whose
		// sheet did not load. Every other ranged swing draws its class's own
		// shot (unitshot.go), and a casting wind-up draws its spell's picture
		// (weaponBoltDraws). The remaining gates are the contract's own: an
		// actual attack target held, a reach past plain adjacency, and a victim
		// the world still answers for at two or more cells away by
		// chebyshevDist's own simplified test.
		//
		// THE PROGRESS IS THE SWING CLOCK OVER THE CHARGE, both counted in
		// ticks: mw.swing[e.ID] counts ticks since the charging phase began —
		// the same clock SelectAttackFrame indexes below — and e.AttackCharge
		// is the countdown chargeTicks (pkg/sim, unexported) loads at that same
		// instant and floors at 1. num is capped at den so a clock that has run
		// past its own charge never overshoots the victim's cell.
		_, _, classDrawsShot := mw.classShot(clientClass)
		if swinging && e.HasAttackTarget && e.Reach > 1 && mw.shots.physical(e.ID) && !classDrawsShot {
			if vcell, ok := mw.attackTargetCell(e); ok {
				if chebyshevDist(vcell.X-cell.X, vcell.Y-cell.Y) >= 2 {
					den := int(e.AttackCharge)
					if den < 1 {
						den = 1
					}
					num := mw.swing[e.ID]
					if num > den {
						num = den
					}
					shot := shotPoint(cell, vcell, num, den)
					draws[i].Shot = &shot
				}
			}
		}

		if !e.Alive() {
			elapsed := mw.observeDeath(e.ID)
			if c != nil && c.Corpse != nil {
				d := c.Corpse
				frames := d.TierFrames(tier)
				// THE BONES FIRST, and the test is the simulation's own decay stage. A
				// body past its fall is drawn from the bone block and not from the last
				// frame of the dying one, and the stage is the ONLY thing consulted:
				// this tier is handed an integer and never asks what a body is, which is
				// what keeps the stage the simulation's to advance and nothing else's.
				//
				// It stands above the fall for the reason the fall stands above
				// the live selection: the chain is read top to bottom and each
				// link refuses rather than answering a frame the caller cannot
				// tell from a real one. So a corpse class with no bone block
				// falls through to its own last dying frame, and one with no
				// dying block through that to the drawing it had before either
				// path existed.
				if frame, mirror, ok := terrain.SelectBoneFrame(d.Anim, len(frames), oct,
					int(e.Decay)); ok {
					draws[i].Art = d
					draws[i].Frame = frames[frame]
					draws[i].Mirror = mirror
					continue
				}
				if frame, mirror, ok := terrain.SelectDeathFrame(d.Anim, len(frames), oct, elapsed); ok {
					draws[i].Art = d
					draws[i].Frame = frames[frame]
					draws[i].Mirror = mirror
					continue
				}
			}
			// NEVER VANISH, and it is a fall-through rather than a decision: no
			// corpse link, a corpse class holding no frames and an index that sheet
			// cannot hold all arrive here, and here is the drawing this entity had
			// before the death path existed.
		}

		frames := c.TierFrames(tier)
		if e.DrawingTurn() && len(frames) > 0 {
			frame, mirror := terrain.SelectStandingFrame(c.Anim, len(frames), facing)
			draws[i].Art, draws[i].Frame, draws[i].Mirror = c, frames[frame], mirror
			continue
		}

		// THE SWING PATH, and it falls through exactly as the death path above
		// does: a class with no attack block, an empty track, a failed gate or an
		// index its own sheet cannot hold all arrive at the live selection, which
		// is the drawing this entity had before a swing could be drawn at all.
		if swinging && len(frames) > 0 {
			// THE RUN INDEX IS SCALED FOR A CAST AND RAW FOR A BLOW (the owner's own
			// rule): a cast's swing is stretched or compressed to cover exactly the
			// interval its projectile crosses in, so every projectile has one
			// complete swing behind it however fast the casts come. A melee blow
			// keeps the one-frame-per-tick clock it has always had.
			run := mw.swing[e.ID]
			if span, casting := mw.castSwingSpan(e); casting {
				run = scaleRun(run, span, len(c.Anim.AttackTrack))
			}
			if frame, mirror, ok := terrain.SelectAttackFrame(c.Anim, len(frames), oct,
				run); ok {
				draws[i].Art = c
				draws[i].Frame = frames[frame]
				draws[i].Mirror = mirror
				continue
			}
		}

		if len(frames) > 0 {
			// THE STONE-CURSE ANIMATION HOLD (1002; MAGIC-ACTOR-066): the
			// clock stops at the scene tick the effect landed on and the
			// actor keeps that pose, walk cycle included, until it expires.
			clock, held := mw.scene, false
			if at, ok := mw.stoneHeld(e.ID); ok {
				clock, held = at, true
			}
			frame, mirror := terrain.SelectUnitFacingFrame(c.Anim, len(frames), moving && !held, facing,
				clock+int(e.ID), mw.walk[e.ID].dist)
			draws[i].Art = c
			draws[i].Frame = frames[frame]
			draws[i].Mirror = mirror
		}
	}
	// AN OFF-MAP UNIT IS NOT DRAWN, and so cannot be picked: the window tier
	// resolves a click against the entities in this list and against nothing
	// else, so one rule removes the unit from the picture and from selection
	// together.
	//
	// THE COMPACTION IS DONE HERE, AFTER THE LOOP, and not as a `continue`
	// inside it: every one of the twenty-odd writes above addresses draws[i]
	// by the entity's own index, so a skipped entity would leave a zero-value
	// MapEntity at its slot — an entity of id 0 drawn at the map's corner —
	// rather than no entry at all. Filtering the built slice keeps the loop's
	// indexing exactly as it was and states the rule once.
	//
	// THE ORDER WITHIN THIS FILTER IS UNCHANGED: this loop is a stable filter
	// over draws, which is built in the world's ascending entity id order,
	// and the compaction above still relies on that per-index correspondence.
	// The slice this FUNCTION returns is no longer globally ascending by id,
	// though: virtualDeadDraws' entries are appended below, after this loop
	// finishes, so they trail every live entity's draw regardless of id.
	kept := draws[:0]
	for i, e := range ents {
		if e.OffMap {
			continue
		}
		// AND AN INVISIBLE ACTOR THIS PARTICIPANT DOES NOT DETECT IS NOT
		// DRAWN (1002; MAGIC-ACTOR-066). The original gates the whole actor
		// sprite on a per-player bit and skips the sprite when it is clear;
		// dropping the entry here rather than blanking its art removes it
		// from the picture and from selection together, which is the rule
		// OffMap above already states. The local participant is SelfSlot,
		// the slot this front end pushes as its own (SetLocalOwner).
		if mw.world.InvisibleTo(e.ID, sim.SelfSlot) {
			continue
		}
		draw := draws[i]
		draw.Boundary = draw.Art.BoundaryOf(draw.Frame)
		// STONE OVERRIDES THE OWNER SHADE ONLY THROUGH THE FIRST BONE STAGE
		// (MAGIC-STONEDRAW-084, REG-UNITS-050, PAL-BLIT-024). Both original
		// Stone draw overrides share the drawable+0x15a <= 2 gate; that field is
		// the corpse stage: 0 live, 1 fallen and 2 the first bone. The attached
		// effect can outlive that boundary, so the effect fact carried into the
		// selection loop is narrowed here to the presentation fact the viewer
		// receives. At stages 3 and 4 the ordinary frame and owner-palette paths
		// resume even while the effect remains attached.
		//
		// This gate and guard sit after every frame-selection arm converges, so
		// they cover the normal, attack, dying, bone, hero-body and substituted
		// corpse populations together. A presented Stone entry reaches the
		// viewer's fixed neutral grayscale path from the selected base frame; an
		// entry outside the gate keeps the ordinary owner rule below.
		draw.Stone = draw.Stone && e.Decay <= sim.DecayBones
		if !draw.Stone {
			draw.Frame = mw.ownerFrame(draw.Art, draw.Frame, e.Owner)
		}
		kept = append(kept, draw)
	}
	kept = append(kept, mw.virtualDeadDraws()...)
	return kept
}

// virtualDeadDraws renders current stage-2..4 records outside the live entity
// population. Their dead-list clock advances independently of occupancy.
// Facing stays at the unobserved default; only a resolved body is drawn.
func (mw *mapWorld) virtualDeadDraws() []ui.MapEntity {
	var out []ui.MapEntity
	owners := map[uint32]uint32{}
	if mw.mission != nil && mw.mission.state != nil && mw.mission.state.savedDocument != nil && mw.mission.state.savedDocument.Document != nil {
		for _, record := range mw.mission.state.savedDocument.Document.Objects {
			if record.Class != "Player" {
				continue
			}
			identity, keyErr := savedStructureValue(&record, "This")
			slot, slotErr := savedStructureValue(&record, "Slot")
			if keyErr == nil && slotErr == nil && identity != 0 {
				owners[identity] = slot
			}
		}
	}
	oct := sheetOctant(0)
	for _, r := range mw.world.OriginalDeadActors() {
		if r.Source.MapUnitID != 0 || r.Current.Stage < 2 || r.Current.Stage > 4 {
			continue
		}
		c := mw.art[r.ID]
		if c == nil || c.Corpse == nil {
			continue
		}
		d := c.Corpse
		frames := d.TierFrames(mw.tiers[r.ID])
		frame, mirror, ok := terrain.SelectBoneFrame(d.Anim, len(frames), oct, int(r.Current.Stage))
		if !ok {
			continue
		}
		draw := ui.MapEntity{
			ID:    uint32(r.ID),
			Owner: owners[r.Source.OwnerKey],
			Cell:  image.Point{X: int(r.Current.Cell & 255), Y: int(r.Current.Cell >> 8)},
			// Life crosses as dead so the viewer takes the corpse depth tie
			// and suppresses the health bar (overlay.go's own rule); HP is
			// the archived tuple's own answer and MaxHP stays 0, this
			// record's own honest "never observed alive this session".
			Life: ui.LifeDead, HP: int(r.Current.HP),
			// No live entity backs this id: it cannot be ordered, attacked
			// or targeted, and Combat/Char stay their zero value (Known
			// false) because nothing here derived either sheet.
			Untargetable: true,
			FinePosition: true, FineX: r.Current.FineX, FineY: r.Current.FineY,
			Art: d, Frame: frames[frame], Mirror: mirror,
			DrawCategory: terrain.UnitCategoryFor(c, r.Current.Stage), CorpseStage: r.Current.Stage,
		}
		draw.Boundary = draw.Art.BoundaryOf(draw.Frame)
		draw.Frame = mw.ownerFrame(draw.Art, draw.Frame, draw.Owner)
		out = append(out, draw)
	}
	return out
}

type ownerFrameKey struct {
	frame *terrain.StaticFrame
	shade uint8
}

// ownerFrame resolves one already-selected production body frame through its
// owner's shared palette. Selection of the class, tier, body replacement,
// corpse and animation frame has already happened; only the table consulted by
// the canonical indexed-pixel blit changes here.
func (mw *mapWorld) ownerFrame(c *terrain.UnitClass, frame *terrain.StaticFrame, owner uint32) *terrain.StaticFrame {
	if frame == nil || mw == nil || mw.units == nil {
		return frame
	}
	palette := mw.units.OwnerPalette(c, owner)
	if palette == nil || frame.Palette == *palette {
		return frame
	}
	key := ownerFrameKey{frame: frame, shade: uint8(owner & 0x0f)}
	if tinted := mw.ownerFrames[key]; tinted != nil {
		return tinted
	}
	tinted := *frame
	tinted.Palette = *palette
	if mw.ownerFrames == nil {
		mw.ownerFrames = make(map[ownerFrameKey]*terrain.StaticFrame)
	}
	mw.ownerFrames[key] = &tinted
	return &tinted
}

// markTileSize is a class's own footprint scale for the mark builders, or 1 for
// a class the unit bundle does not answer for — the units.reg key's own
// registry default (MAGIC-MARK-059).
func markTileSize(c *terrain.UnitClass) int {
	if c == nil || c.TileSize < 1 {
		return 1
	}
	return c.TileSize
}

// sackFrame reads the sack's value from its world, where the item records a
// stack is written as are known, and falls back to the record-free sum.
func (mw *mapWorld) sackFrame(s sim.Sack) int {
	if value, ok := mw.world.SackValue(s.X, s.Y); ok {
		return sackFrameForValue(value)
	}
	return sackFrameIndex(s)
}

func (mw *mapWorld) sackDraws() []ui.MapSack {
	sacks := mw.world.Sacks()
	draws := make([]ui.MapSack, len(sacks))
	for i, s := range sacks {
		draws[i] = ui.MapSack{
			Cell:       image.Point{X: int(s.X), Y: int(s.Y)},
			FrameIndex: mw.sackFrame(s),
		}
	}
	return draws
}

// observeDeath stamps the scene tick an entity was first seen not alive and
// answers how many ticks have run since.
//
// SET ONCE PER FALL. A second call in the same build, or in any later one,
// finds the entry and leaves it, so the count it answers is a function of the
// scene clock alone and a snapshot built twice with no advance between carries
// the same fall both times. Loading current actions restores a saved stamp;
// drawing an actor who recovered clears it before that actor can fall again.
//
// The first call answers 0 — the fall's own first frame, on the very tick the
// world first reported the entity not alive.
func (mw *mapWorld) observeDeath(id sim.EntityID) int {
	died, seen := mw.died[id]
	if !seen {
		died = mw.scene
		mw.died[id] = died
	}
	return mw.scene - died
}

// fallenDamageJolt is the three-tick sprite-only twitch requested by the owner.
// It is authored and recorded as DIV-488: ANIM-BLOW-019 finds no second sprite,
// flash or overlay in ROM1's take-damage client arm. Alternating two pixels is
// enough to read at native scale while returning exactly to the corpse anchor.
var fallenDamageJolt = [...]image.Point{
	{X: -2},
	{X: 2},
	{X: -1},
}

// observeDamage retains application order for delivery and diagnostic counts.
// A decrease on a fallen body restarts its short jolt; expiry removes old ids.
func (mw *mapWorld) observeDamage(events []sim.DamageEvent) {
	mw.pendingDamage = append(mw.pendingDamage, events...)
	nextScene := mw.scene + 1
	for id, at := range mw.hurt {
		if nextScene-at >= len(fallenDamageJolt) {
			delete(mw.hurt, id)
		}
	}
	for _, event := range events {
		if event.BeforeHP == event.AfterHP {
			if mw.strikes == nil {
				mw.strikes = make(map[sim.EntityID]uint32)
			}
			mw.strikes[event.Target]++
			continue
		}
		if mw.blows == nil {
			mw.blows = make(map[sim.EntityID]uint32)
		}
		mw.blows[event.Target]++
		e, ok := mw.entity(event.Target)
		if !ok || e.MaxHP <= 0 || event.BeforeHP > 0 || event.BeforeHP <= -10 || event.AfterHP >= event.BeforeHP {
			continue
		}
		if mw.hurt == nil {
			mw.hurt = make(map[sim.EntityID]int)
		}
		mw.hurt[event.Target] = nextScene
	}
}

// damageJolt answers the sprite offset for the current scene without mutating
// the clock. The unsigned-looking bounds are written as two comparisons because
// a hand-built mapWorld may ask before a stored stamp; such a value is silence,
// not an index wrapped through the pattern.
func (mw *mapWorld) damageJolt(id sim.EntityID) image.Point {
	at, ok := mw.hurt[id]
	if !ok {
		return image.Point{}
	}
	age := mw.scene - at
	if age < 0 || age >= len(fallenDamageJolt) {
		return image.Point{}
	}
	return fallenDamageJolt[age]
}

// lifeOf translates the simulation's own three predicates into the one byte the
// seam carries. It is a TRANSLATION and never a derivation: every arm asks
// pkg/sim, so the rule that decides what alive, downed and dead mean stays in
// the package that owns the health fields, and this tier holds no copy of it.
//
// It is total by the predicates' own construction — they are pairwise exclusive
// and jointly total over every pair of integers — so the last return is
// unreachable rather than a default anything falls into. It is written as the
// alive arm because a state that somehow satisfied none of the three would be a
// unit with no wound to report, which is the answer that changes least.
func lifeOf(e sim.Entity) uint8 {
	switch {
	case e.Dead():
		return ui.LifeDead
	case e.Downed():
		return ui.LifeDowned
	default:
		return ui.LifeAlive
	}
}

// sheetOctant converts the simulation's nearest compass octant to sheet order.
// The live renderer instead halves the stored client sixteenth (ANIM-DIR-006).
func sheetOctant(facing uint8) int { return (sim.FacingDir(facing) + 4) & 7 }

// routeOf is one entity's remaining route as the seam carries it: the
// world's own answer, converted cell for cell into the window tier's point
// type and into nothing else.
//
// IT IS A CONVERSION AND NOT A DERIVATION. The route is state the simulation
// already holds and already maintains — the far search stores one, the advance
// consumes its walked head — so this tier neither computes a path nor remembers
// one between ticks, and there is no cache here to go stale against a world that
// re-routed. That is why the render side of this story is a field and not a
// subsystem: the question "is the stored route still the right one" is the
// simulation's, answered by its own staleness tests, and a second answer on this
// side could only disagree with it.
//
// IT IS BUILT FOR EVERY ENTITY, and the alternative was measured against the
// seam rather than against the cost. The tier that knows which units are
// SELECTED is the window tier, on the far side of a boundary that exists so this
// one cannot be asked about drawing; reaching back through it for the selection,
// to skip the copy for everybody else, would put the overlay's own scoping rule
// on both sides of the seam. The copy is per TICK and not per frame — sixteen a
// second against the display's rate — and an entity holding no order copies
// nothing at all, which is every entity that is not under orders.
func (mw *mapWorld) routeOf(id sim.EntityID) []image.Point {
	route := mw.world.Route(id)
	if len(route) == 0 {
		return nil
	}
	out := make([]image.Point, len(route))
	for k, c := range route {
		out[k] = image.Point{X: int(c[0]), Y: int(c[1])}
	}
	return out
}
