# Story 1562: a second-game summoned creature leaves by the original's cap pass

## Intent

Every spell a second-game map names resolves to a rule and casts in play. Summon built a creature
that stayed until something killed it (`DIV-4151`). The original caps each caster's summoned
units with a world pass; this story makes the engine's creature leave by that pass.

## Authority

Knowledge pin k400 (`9877ee4`).

- `R2-ENGINE-455` (amended): the Summon arm stores the caster's `+4` in the creature's `+0x148`.
- `R2-ENGINE-776` (High / Medium / Unknown): the world pass. It counts the units of each key
  `+0x148`, draws r = `L2.00092(100)` = 1..100 once per key, marks the key when r < n, and gives
  the death to the first unit of each marked key: at most one per key per pass. The death stores
  health -10000 and clears `+0x148`; its corpse ageing ends at kind 5 and -10001, or at -50 and
  kind 4 for the owner's `+0x38` unit. The key order and the unit order are Medium; the time of a
  pass after LOAD and the visible effect of kind 5 are Unknown.
- `R2-ENGINE-778` (High / Medium): the session tick runs the world pass when Session `+4` mod 16
  is 12; about one pass a second (Medium); the period after LOAD is Unknown (`R2-ENGINE-669`).
- `R2-ENGINE-777` (Medium): that the stat reader runs for a Summon name.
- `R2-SESSION-326` (High that the Unit programme does not store `+0x148`; Unknown): whether a
  mission save writes a summoned creature at all and whether a loaded one is counted by the cap.
  The class factory path and the save's unit-selection path were not read. `R2-ENGINE-535`: init
  stores 0 in `+0x148`; that does not say LOAD runs init. That a loaded creature is outside the
  cap is the engine's reading of this Unknown, not a claim's statement.

## As-built behaviour

- The session tick counter is the World tick (`w.tick`, Session `+4`). It was already in the byte
  form and in SAV (the head's CounterA, the high word through `RestoreCurrentClock`). Session `+0`
  is not carried for a second-game World and the pass does not read it.
- `castSummon` gives the creature `summonKey`, `summonKeyOf(caster id)` = the caster's entity id
  plus one, on the template that `createActor` builds. `R2-ENGINE-455` stores the caster's `+4`, a
  nonzero word the original's pass reads as a key; 0 is "no key". Entity id 0 is live (the first
  placed map unit), so the engine's key is shifted by one and a caster with id 0 is capped like
  any other. The map is increasing, so ascending key is ascending caster id. The field is
  unexported, so no JSON, gob or SAV record carries it.
- The second-game arm of `stepWorld` runs `summonCapPass` on phase 12 of the 16-tick cycle
  (`worldPassPhase`), beside the script pass (6) and the report (15). The first game's session
  tick has no such arm.
- `summonCapPass` counts the keyed actors per key, draws `1 + w.rng.uniform(secondDrawSpan(100))`
  once per key in ascending key order, and gives the lowest entity id of each marked key the
  death: health -10000, then `clearFelled`.
- `clearFelled`, the one death transition, stores -10000 and clears the key for every keyed death,
  whichever route caused it. The body falls, lies for its row's dying time and leaves the World by
  the corpse ladder's removal below -600 (`decayPass`).
- The byte form carries the keys in a new outermost section (version 131, tag `SMK1`), written only
  when an actor carries a key. Every keyless form, the first game's included, is byte-identical to
  the base.
- A SAV LOAD builds every actor through `NewActor`, whose definition has no key, so the engine
  reads a loaded creature as outside the cap. This is the engine's reading: `R2-SESSION-326` is
  Unknown whether the original's mission save writes the creature and whether a loaded one is
  counted (`DIV-4388`, `DIV-2618`).
- A summoned creature killed in a fight leaves the World after its fall, as a keyed death stores
  -10000, instead of decaying to bones; a Control Spirit therefore cannot take its corpse.
- Seeded Worlds draw the 1..100 from the seeded stream (`DIV-027`).

For a caster with n summoned creatures, the original stream gives a pass chance of
ceil((n-1) * 327.68) / 32768: 0 for one creature, 328/32768 (1.0 %) for two, 984/32768 (3.0 %)
for four. The base kept all of them for any number of ticks.

`DIV-4151` is closed into `docs/DIVERGENCES-CLOSED.md`. `DIV-4388` holds what the claims leave
Medium or Unknown and the engine's reading: ascending key and id order, the phase kept through
SAV, the fall-dwell-removal standing for kind 5, no owner's `+0x38` unit, and the Units row
statistics. `DIV-2618` now cites the key, the session flag and the stat reader.

## Proof

`pkg/sim/summoncap_test.go`, synthetic Worlds, no install:

- `TestSecondSummonedCreaturesLeaveOnTheWorldPassUntilOneIsLeft`: four Summon creatures under
  three seeds; each leave is one creature on a step entered at tick 12 mod 16, three leave and one
  stays. It fails on the base ("creatures left at ticks [], 4 remain").
- `TestSecondSummonGivesItsCreatureTheCastersKey`.
- `TestSecondSummonCasterWithEntityIDZeroIsCapped`: casters with id 0 and id 1 each cast four
  times; the creatures carry a nonzero key and 1500 passes leave one each. For caster 0 it fails
  with the key set to the id ("caster 0 4, caster 1 1").
- `TestSecondWorldPassDrawsOncePerKeyAndKillsTheFirstActorOfAMarkedKey`: 400 direct passes on
  the original stream with keys of 1, 2, 4 and 101 actors, against a witness of the claim's draw
  on a copy of the stream: one draw per key, death exactly when r < n, the lowest id, health
  -10000, key 0, stage fallen; the key of one actor never loses it.
- `TestSecondWorldPassRunsOnlyOnTheTwelfthTickOfSixteen`: 96 steps; a key of 101 loses one actor
  on each step entered at 12 mod 16 and none on any other.
- `TestSecondKeyedDeathStoresTheSummonedHealthAndTheBodyLeavesAfterItsDwell`.
- `TestFirstGameKeyedActorsMeetNoWorldPass`: a first-game World with keyed actors draws the same
  stream and loses no actor, as the same World without keys.
- `TestSummonKeysRoundTripTheByteFormAndAKeylessFormHasNoSection` and
  `TestSummonKeySectionRefusesABadPair`.
- `TestALoadedActorIsOutsideTheCap`: actors rebuilt through `NewActor` carry no key; the pass
  draws nothing and four creatures stay for 1500 passes.

The first test uses only the base's API and fails there; the others read `summonKey` or
`summonCapPass`, which the base lacks, so they do not compile on it. No pinned digest moved: `go test ./pkg/sim` and
`go test ./pkg/game` pass with their pins.

`TestReleaseSecondCampaignStoneCurseAndSummonSurviveSAVEAndColdLOAD` compared the cold LOAD's
complete World hash with the saved World. The saved World's creature carries a key the SAV does
not, so the release test now compares every actor field, every effect record (`ActiveEffects()`,
the curse's remaining time, caster, kind and magnitude included), `RandomState()` and `Tick()` of
the cold LOAD with the saved World, and the complete World of a second SAVE and cold LOAD with
the first. It was compiled, not run.

Worlds with no keyed actor are unchanged: 96 ticks (six passes) of six actors with attack orders
for seeds 3 and 0x25, in a first-game World, a second-game World, a second-game World on the
original stream and a second-game World with the Summon rule and no cast, give the same SHA-256
over `MarshalBinary`, `Hash()` and `RandomState()` at every tick on `faa5e7c4a` and on the
candidate (eight digests).

## Open debt

- `DIV-4388`: the key and unit orders, the time of a pass after LOAD, the visible effect of kind 5,
  the owner's `+0x38` unit and whether the stat reader runs for a Summon name.
- `R2-SESSION-326` (Unknown): whether the original's mission save writes a summoned creature and
  whether a loaded one is counted by the cap. Settling routes: a read of the class factory path
  and of the save's unit-selection path, or a run of the original that SAVEs with two summoned
  creatures of one caster, LOADs and waits 10 minutes of game time. The engine's reading (a
  loaded creature is outside the cap) stands in `DIV-4388` until then.
- `DIV-4152` is unchanged: no dead unit is reused and no summoned mark is set.
- The release tests to run at the next release, on `rom2-en` and `rom2-ru`:
  `TestReleaseSecondCampaignStoneCurseAndSummonSurviveSAVEAndColdLOAD`, the second-game SAVE
  and cold LOAD witnesses of `secondmissionsave_release_test.go`, and the milestone-2 family
  (`scripts/check-milestone2-acceptance.sh`, `TestMilestone2*`). The SAV bytes do not change;
  the World byte form gains a section only when a creature carries a key.
