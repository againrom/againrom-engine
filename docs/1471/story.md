# A spell cast at a target out of reach

## Intent

A player's book cast at a unit or a cell beyond the spell's reach is kept.
The caster walks until the target is in reach, then casts. A computer
player's mage casts at the victim its engage routine was given.

## Authority

Knowledge k328. Claims read: `MAGIC-340` (High bodies / Medium outcome),
`MAGIC-348` (High / Medium), `MAGIC-239` (High / Medium), `AI-487`, `AI-554`,
`MAGIC-AI-012`, `MAGIC-311`, `MAGIC-376`, `MAGIC-377`, `AI-617`, `AI-618`,
`MAGIC-TELEPORT-364`, `MAGIC-TELEPORT-365`, `MENU-CAST-153`, `TERR-CLICK-262`.

- The manual setters park the Spell as pending kind 8 (unit) or 9 (cell) and
  test only for a parked Spell. The executor gate tests facing and reach; a
  failing gate calls the approach wrapper and keeps the order. The admission
  refuses on mana alone and retries (`MAGIC-340`). The admission callees test
  no range or sight (`MAGIC-348`).
- The actor approach passes the target to the pursuit step; the point
  approach walks to the cell (`MAGIC-239`).
- The client's cast click has no own fog test; its cursor gate reads the
  four-corner aggregate (`MENU-CAST-153`, `TERR-CLICK-262`). The point builder
  includes only members holding the spell bit (`MAGIC-TELEPORT-365`).
- The computer mage's engage routine `R0393` takes the victim as an argument.
  Above Mind 59 a 30-in-100 draw hands back to the engage routine and casts
  nothing. Otherwise it draws uniformly from the affordable non-defensive ids
  1..28 and passes the victim to the cast-order writer. With an empty list and
  id 28 in the book it casts id 28 unfiltered (`AI-554`, `MAGIC-AI-012`).

## Behaviour

### A player's cast out of reach

1. The player arms a spell from the book or a quick-spell key and clicks a
   unit or a cell beyond the spell's reach.
2. The caster keeps the cast. Nothing is paid. The caster walks: to the cell,
   or to the target unit's cell, re-aimed each time that unit changes cell.
3. When the target is in reach and the caster sees it, the caster stands,
   turns, pays and charges as an in-reach cast does, and releases the spell.
4. Another order ends the walk and the cast: move, pick up, attack, use of a
   structure or a scroll, another cast, a group order, stand, guard, defend
   and retreat. A target that dies, leaves the map by a script or can no
   longer take the spell ends the cast unpaid; the caster stands where it is.

A cast clicked while the caster's strike winds up waits behind the strike,
in reach or not; when the strike ends the click runs and walks if it must.
A controlled creature with class spells walks its book casts the same way.

Teleport out of reach is this rule's cell case. EN and RU are the same: the
rule reads no text.

### A computer player's mage

When its engage routine attacks a victim, a computer mage with a book casts at
that victim. In reach it stands and charges at once. Out of reach it pursues
the victim with the attack's pursuit and casts when in reach. A mage above
Mind 59 attacks without casting on 30 of 100 engages. A mage that can afford
none of its spells and holds id 28 keeps an id-28 order at the victim and
casts it when its mana allows. A retained order that reaches its victim
without the mana stands there and waits for it.

### Click refusals

The engine's checks at a player's book-cast click, against the claims.

| Refusal at the click | Original | Decision |
| --- | --- | --- |
| caster absent, off map or dead | the client includes only a living member (`MAGIC-TELEPORT-365`) | kept |
| spell not known, row absent, no applicable arm | the client includes only a member holding the spell bit (`MAGIC-TELEPORT-365`) | kept |
| spell takes no unit, or no cell | the cursor selects the unit or the point builder (`MENU-CAST-153`) | kept |
| target absent | no target to send | kept |
| target state (untargetable, a Prismatic Spray body, a damaging self cast, Heal without health) | the setters and the admission test no target state (`MAGIC-340`, `MAGIC-348`) | kept, `DIV-3266` |
| target not seen by the caster | no sight test at the click or the admission (`MAGIC-348`, `MENU-CAST-153`) | moved: a target a unit of the caster's player sees arms the walk; the caster casts once it sees the target itself. A target no unit of the player sees is refused, `DIV-3266` |
| target or cell out of reach | the executor's gate walks and keeps the order (`MAGIC-340`, `MAGIC-239`) | moved to the walk |
| point effect refused (`pointEffectRefusal`) | the admission callees refuse nothing (`MAGIC-348`) | kept, `DIV-3266` |
| area landing refused (`areaLandingRefusal`) | the admission callees refuse nothing (`MAGIC-348`) | kept, `DIV-3266` |
| caster turned to stone | Unknown | kept, `DIV-3266` |
| insufficient mana | the admission refuses and retries until the progress-2 exit (`MAGIC-340`, Medium outcome) | kept, `DIV-3266` |
| cell outside the map | the client builds no such cell | kept |

A busy caster is not tested at the click (`checkAction` is false): a cast
behind a strike waits as a pending row and runs when the strike ends; a
running paid wind-up keeps its present rule (`DIV-3267`).

## As built

One builder arms every approach: `armCastApproach`
(`pkg/sim/creaturecast.go`). Its caller picks the walk: a creature or computer
cast pursues with the attack's pursuit (`attachAttack`); a human
participant's book order walks with a move order (`attachMoveOrder`).
`castApproachPursues` reads the walk back from the caster: a computer
caster, or a controlled creature that holds the attack its class-spell cast
attached. `approachHeld` reads the order each kind holds;
`stepBookCasts` (`pkg/sim/spell.go`) is the one mover.

Sites that changed builder:

| Site | Before | Now |
| --- | --- | --- |
| `armCreatureCast` (creature and computer mage) | `armCreatureApproach` | `armCastApproach` |
| `stepBookCasts`, retained pending and release re-arm | `armCreatureApproach` | `rearmCastApproach`, which calls `armCastApproach` for pursuers |
| player click, `beginManualCast` | `beginCellCastApproach`, Teleport only, through `manualCellApproach` | `beginCastApproach` through `manualCastApproach`, every unit and cell cast |

Other changes:

- `manualCastApproach` admits any human participant's caster, a controlled
  creature with class spells included.
- `castAfterStrike` (`pkg/sim/manualcast.go`) queues a click behind a running
  strike or step, out of reach as in reach; `takePendingOrders` runs it
  through `beginManualCast` when the strike ends.
- `approachTargetGone` ends an approach whose target a script took off the
  map, before any payment.
- An approach refused for mana alone in reach turns into the waiting pending
  record for a retained pursuing cast.
- `startApproachedCast` stands a walking player caster (`standCaster`) before
  it charges. `walkApproach` re-aims the unit form's walk at the target's
  current cell.
- `dropCastApproach` (renamed from `dropCreatureApproach`) runs at every
  player order that replaces the cast: move and pick up, attack, use of a
  structure, a scroll, group orders, defend and retreat.
- `clearInvalidTargetReferences` (`pkg/sim/target.go`) stands a walking
  player caster whose cast it removes.
- `approach` (`pkg/sim/combat.go`) keeps pursuing a closed-on victim while an
  approach is armed, so a pursuer reaches its reach instead of striking.
- `bookSpellAdmissionRange` with `skipRange` skips the caster's sight as well
  as the reach. Creature callers ask it only after a reach refusal, so their
  result is unchanged.
- `aiCast` and its calls in the engage pass and the saved-group decision are
  removed. `mageEngageCast` (`pkg/sim/engage.go`) runs from
  `creatureEngageCast`, inside `orderAttack`, the engine's engage routine.

Scroll casts keep their path (`AI-617`). The release re-test (`DIV-3269`) is
unchanged: a player's cast whose target leaves reach during the wind-up ends.

The SAV format is unchanged. The approach record was already part of the
book-cast state (`castbinary.go`). The one SAV producer writes a walking
player cast as action 0xd or 0xe with order kind 8 or 9 at progress 0 and the
retention flag 0x60 at zero; the walk itself is the engine's move order
(`DIV-3681`).

## Proof

Focused tests, `pkg/sim/castapproach_test.go`:

| Test | Shows |
| --- | --- |
| `TestAPlayerCastAtAUnitOutOfReachWalksAndCasts` | unit 11 cells off: armed, unpaid, walks, casts, pays 3, stands |
| `TestAPlayerCastAtACellOutOfReachWalksAndCasts` | cell form lands on the clicked cell |
| `TestATeleportOutOfReachStillWalksAndTeleports` | Teleport ends on its cell |
| `TestAnotherOrderReplacesTheCastApproach` | move, stand, attack and another cast end the cast; it never lands |
| `TestATargetThatDiesEndsTheApproachAndStandsTheCaster` | the record goes, the caster stands unpaid |
| `TestACastInReachStartsOnTheClickTick` | in reach: charging on the click tick, no walk |
| `TestACastApproachSavesAndLoads` | `MarshalBinary` round trip, 300 ticks in lockstep, the cast lands |
| `TestACastAtAUnitOnlyAnotherMemberSeesWalksUntilTheCasterSees` | arms when another unit of the player sees; refused when none does |
| `TestAComputerMageCastsAtItsEngageVictim` | near victim: charging at it; far victim: approach at it |
| `TestAComputerMageWithAnEmptyListCastsIdTwentyEight` | id 28 waits for mana at the victim; without id 28, the plain engage |
| `TestAHighMindMageHandsThreeInTenEngagesBack` | over 200 seeds both outcomes occur, hand-backs at most 90 |
| `TestATargetTakenOffTheMapEndsTheApproachUnpaid` | script removal (`takeOffMap`): no cast, mana 100, target unhurt, caster stands |
| `TestADistantCastClickedDuringAStrikeIsKeptAndLands` | unit and cell clicks during a charging strike wait and land after it |
| `TestAPlayerCreatureWalksItsDistantBookCast` | a controlled creature with a class slot walks unit and cell casts and lands them |
| `TestAFarMageWithIdTwentyEightWaitsForManaInReach` | a far id-28 order walks, waits pending at zero mana for 300 ticks, casts once mana comes |

Loss controls: with `manualCastApproach` forced false, the unit, cell,
Teleport, replacement, dead-target and SAV tests fail. With
`dropCastApproach` disabled, move, stand and attack-the-same-unit fail. With
`mageEngageCast` disabled, the three computer-mage tests fail. Each of the
last four tests fails alone with its own fix reverted: the off-map end, the
strike queue, the creature admission and the mana wait.

Release test `TestReleasePlayerCastApproachSaveColdLoad`
(`pkg/game/castapproach_release_test.go`, save corpus `2026-08-24/game0021.sav`,
mission 10), EN and RU: Fergard is clicked to cast spell 9 at (23,66), beyond
reach, by the book and a ground click. F2 SAVE while he walks writes action
0xe, order kind 9, progress 0, flag 0. Cold LOAD keeps the record; both
sessions keep one hash until he casts from (20,66), 38 ticks after LOAD.

Headless mission scenarios, EN and RU, both pass:

- `scenarios/cast-approach-mission41.json`: the mage hero at (44,68) casts
  Fire Arrow at `u37`, 16 cells off and seen by no unit of the player: refused,
  the hero stays. `u65` is placed at (42,57); Heal at it arms the walk on
  tick 13. The hero walks to (42,62), unpaid at tick 74, and has paid by
  tick 94.
- `scenarios/computer-mage-mission41.json`: the hero is placed at (61,67)
  beside the computer mage `u38`; by tick 31 `u38` has spent mana and the
  hero's health is 43 of 72.

Hashed-state census, `pipeline/check-mission-scenarios.sh` on EN and RU, base
`f5c724f2` against this head: 76 of 76 runs pass on the head, 72 of 72 on the
base. 42 runs keep every report hash. 30 runs differ, 15 scenarios on each
root: `0159-mission40-join`, `1193-mission121-bridge-guard-leash` and
campaign 040, 041, 061, 080, 091, 100, 110, 111, 120, 130, 131, 141, 150.
Each of those missions holds a computer mage (a mage hero drive of every
campaign mission found one in 40, 41, 61, 80, 90, 91, 100, 110, 111, 120,
121, 130, 131, 141, 150 and 151); no scenario on a mission without one
differs. None of these scenarios orders a player cast. The differences are
attributed to the computer mage's new target; that attribution is inference
from the population, not a per-tick trace. `pipeline/check-milestone.sh`
measures the same script census for the base and the head build.

`TestAUnitCastFollowsTheEditedManaRangeAndDamage` (`pkg/mapload`) pinned
the dropped cast: a target beyond an edited reach of 2 cost nothing. It now
requires the click's reach refusal, a walk, the edited cost of 9 and the
edited damage of 60; the caster casts from (2,0) at the target at (5,0), two
cells between them.

The second game's computer mages use the same engage rule: `mageEngageCast`
draws from the whole book there (the id-28 case is first-game only), and the
removed `aiCast` no longer casts in the decision pass. This rests on ROM1
evidence; no second-game claim was read. It moves the world hash of the
second game's profile witness (`TestReleaseSecondGameProfileWitnessIsUnchanged`,
`pkg/game/testdata/profilewitness/rom2-en.txt` and `rom2-ru.txt`) from line 21,
`run screen=map mission=10`, `world=5084e4e83d49d852/7` to
`world=2789a05484450c02/7`, and the eight lines after it; every frame hash
is equal. With `aiCast`, its two call sites and the old auto target restored
and `mageEngageCast` disabled (a build overlay over `pkg/sim/engage.go` and
`pkg/sim/savedgroupsai.go`), the witness passes unchanged on EN and RU;
disabling `mageEngageCast` alone keeps the new hash. The files were rewritten
with `AGAINROM_PROFILE_WITNESS_WRITE`. No second-game divergence row records
computer mage AI, so no row was added.

Two release tests changed their choreography, not their claim:

- `TestReleaseAnAttackedGroupAtKadaganPostFightsAsAGroup` counts a member
  holding a book cast at a hostile unit as fighting. The garrison's mage
  (member 12) now casts at its engage victim instead of striking it.
- `TestReleaseMission81QuestItemHandoverAndColdSAV/club` kills the hostiles
  within 4 of the club again after the holder reaches it, and finishes the
  loaded cycle before the grab. The three book-less computer mages of
  owner 4 (mana 750, no known spell) now draw the Mind hand-back at each
  engage (`AI-554` draws before the book walk). The shifted draws send the
  owner-4 guards at the holder while he walks to the club; with the draw
  skipped for an empty book the original choreography passes (one
  experiment, reverted).

## Open debt

- `DIV-3266`: the click still refuses target state, point effect, area
  landing, a stoned caster, mana and a target no unit of the player sees.
- `DIV-3268`: the hand-back does not re-enter the engage routine's draws; the
  full tick composition stays Medium.
- `DIV-3681`: the player's unit-form walk is a move to the target's cell, not
  the actor pursuit; the SAV carries that walk.
- `DIV-3682`: what ends an approach whose target dies or leaves is the
  engine's rule.
- `DIV-1310`: the compressed fog's mixed aggregate stays.
- A controlled creature with class spells saves its walking book order with
  the retention flag 0x60 at 1, since the SAV writer sets it for any retained
  cast of a slot holder; the original's manual setter writes 0. The order
  round-trips; the byte differs.
- Script orders given to a walking player caster were not traced; only the
  player order arms listed above drop the approach.
