# Story 1411: the second game's Blizzard

## Intent

A Blizzard cast in the second game (spell table row 7) plays instead of being refused at landing
(the former DIV-2617). The area pulses over its target, each pulse striking one sampled cell;
actors there take water-resisted damage and structures the structure roll less 5. The campaign maps
whose creatures hold Blizzard have no spell without a rule. A SAVE during the pulses keeps the area,
and a cold LOAD continues it to the same remaining pulses and damage.

## Authority

Second-game claims at pin k260, read with the claims they cite:

- `R2-ENGINE-390` (Medium): positive Blizzard damage builds a mode-2 AreaEffect owning a damage
  object, row 7 at +0x0c, countdown +0x50 zero.
- `R2-ENGINE-391` (Medium): one cell per pulse from two inclusive RNG(5) draws, x then y, minus 2,
  so offsets -2..3; byte coordinates; a cell outside 8..dimension-9 is rejected but its draws are
  spent and the pulse counts. Range call floor(r*(n+1)/32768) over the MSVC core.
- `R2-ENGINE-392` (Medium): the countdown is decremented and its old value tested; an old zero
  resets it to 2 and attempts a pulse; 32 attempts on area updates 1, 4, ..., 94.
- `R2-ENGINE-393` (Medium / Unknown): row 7 damage 10..30 filled as B = trunc(10f),
  S = trunc(30f - B), f = 1 + P/30; actors take B+U[0,S] cut by water resistance with no hit roll,
  no caster or ally exclusion, repeatable; structures max(B+U[0,S]-5, 0) when S is positive.
- `R2-ENGINE-394` (Medium / Unknown): opcode 0x86, identifier 23, argument 16; art and sound
  Unknown.
- `R2-ENGINE-395` (High): row 7's stored parameters are equal on EN and RU.
- `R2-SESSION-161` (Medium / Unknown): the AreaEffect serializer's six-byte suffix holds
  initialization, radius, direction, pulse counter and countdown. `R2-SESSION-162`: caster counts
  Unknown.

## As built

- **Builder.** The engine's one area-effect builder (`pkg/sim/celleffect.go`, `landAreaAimed` and
  `tickCellEffect`) already runs Distribution 5 rows as staged areas whose program is keyed by arm.
  Arm 21's program is the claims' loop: 32 stages three ticks apart, one cell per stage from
  `rng.uniform(5) - 2` drawn x then y, byte arithmetic, the 8..dimension-9 inset. Row 7 now runs
  arm 21 (`pkg/sim/secondspell.go`); no new builder, object kind or game branch.
- **Clock.** The landing tick is area update 1 (DIV-3049). The SAV pulse counter is
  `phase/3 + 1` and the countdown `2 - phase%3`, which equal the claims' counter and countdown
  after every update (`TestSecondBlizzardPulseScheduleAndCellDraws`).
- **Draws.** The cell draws come from the World's one stream: seeded mode draws through
  SplitMix64 only when a pulse runs, original mode consumes the shared MSVC stream in the claims'
  order. No other consumer's draws move.
- **Damage.** Actors take the existing second-game roll (`secondSpellDamage`): base plus
  U[0,spread], water protection held to 0..100, `(v*(100-p)+75)/100`. Structures take the existing
  structure resolver (`applyStructureSpellAt`). No caster or ally exclusion (AreaHits all).
- **SAVE and LOAD.** The SAV area writer and reader key the staged program and the cell layer on
  the row's arm (`pkg/game/savworldeffects.go`, `savspellgraph.go`, `savarea_current.go`). The
  byte form gives a second-game spell table its arms before it validates retained areas
  (`World.rowArms`, set by the second-game form). A sampled program's last painted cell rides in
  the engine's action leaf (`CurrentAreaPolicy.RingCells`); LOAD admits at most one cell inside
  the stage window and draws nothing (DIV-3052).
- **Census.** Missions 53, 87, 92, 102 and 110 name Blizzard and now have no spell without a rule;
  mission 110 has no blocker left (Ready 8 to 9), 53, 92 and 102 keep the census party blocker,
  87 the party and entry blockers.

### Changes outside Blizzard that its proof needed

- A second-game SAVE while a Poison Cloud (row 6, arm 8) stood was refused: the cloud's cell layer
  was looked up by row id. It is now looked up by arm. First-game rows are their own arm. The same
  change ends the refusal for rows 14 and 15 (row 14 is cast by creatures in 15 missions) and moves
  the second game's Earth Wall (row 17) to cell slot 3 from slot 5, so a SAV written on main with a
  standing second-game Earth Wall would load without its cells; no creature in missions 1 to 130
  holds row 3, 15 or 17 (review note 4, `pipeline/reviews/story1411-adversarial-review.md`).
- A first-game SAVE taken during Meteor Storm (arm 21) now carries the storm's last painted cell,
  and its LOAD draws nothing. Before, the LOAD drew two values to repaint a different cell, so
  the loaded World's stream and cell differed from the saved one (inference from
  `RestoreCurrentAreas`, which called `ringStageCells` for arm 21). That SAV gains one JSON field;
  no other first-game SAV byte, World hash or cast result moves.

## Proof

- `pkg/sim/secondblizzard_test.go`: the schedule (32 attempts on updates 1..94, nothing between,
  the counter and countdown of the claims' loop, the area gone after update 94), the draws (two per
  attempt, x then y, offsets -2..3 with both ends reached, rejected edge cells still drawing) on
  the seeded stream and on an independently computed MSVC stream; actor damage exact per pulse
  with water protection 0, 40, 100, -20 and 150, the caster struck, an actor struck twice;
  structure loss exact for spread 20 and nothing for spread 0; the carried cell admission.
- `TestReleaseSecondGameBlizzardCast` (rom2-en, rom2-ru): mission 102's creatures cast Blizzard at
  the hero through their own cast route; every area follows the schedule and window, struck actors
  lose at least their resisted base, and a second run repeats every pulse and the World hash.
- `TestReleaseSecondGameBlizzardSaveContinuation` (rom2-en, rom2-ru): a named SAV at pulse 12 on
  mission 102, a cold LOAD in the process and one in a fresh process, 120 identical World hashes
  until the saved area runs out its pulses; loss control: the same SAV without the carried cell
  loads to another World.
- `TestReleaseSecondGameSupportCensus`: NoSpellMaps 5 to 0, Ready 8 to 9, the per-mission
  blockers above.

## Open debt

- DIV-3049 first-pulse delay, DIV-3050 presentation, DIV-3051 damage admission and EN parity,
  DIV-3052 the original SAV form of a standing area, DIV-3053 Ice Missile and Diamond Dust.
- Other row-id checks in the SAV area import remain keyed on first-game ids (the Earth Wall test
  `d.Spell != 19` in `savworldeffects.go`).
