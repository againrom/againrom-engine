# A second-game spell in flight continues after LOAD

## Intent

A second-game mission file that holds a spell in flight now continues it
after LOAD. The spell lands, does its damage and retires. A SAVE taken while
it lives writes it from the current state, and the cold LOAD of that SAVE
continues it. Before, the LOAD carried the records and the
SAVE dropped them: the transport hung, never landed, and the next SAVE wrote no effect. The first
game's LOAD, SAVE and hashed state are unchanged.

## Authority

- Owner: one builder per kind. The continuation is the first game's
  (`currentWorldEffects` and the `SavedWorldEffectDrivers` rows), and the
  second game's records enter it as data.
- R2-SESSION-260: game0006 holds one top-level SpellTransport (countdown 1,
  +0x4c null) nesting a PointEffect with target key 347558336, which nests an
  Effect_DirectDamage.
- R2-SESSION-244: Prj16 has picture 40, action target 24598 and zero
  action segments.
- R2-ENGINE-659: the LOAD restores the typed records into the effect list. A
  key that misses the lookup stores zero.
- R2-ENGINE-660 (Medium): the transport decrements its countdown, and at zero
  or below it queues its child. Same tick or next tick for the queued point,
  the damage formula and the target binding are Unknown (DIV-4423, DIV-4424).
- R2-ENGINE-661 (Medium): a projectile with zero segments is removed on its
  first update. Its link to the transport is Unknown (DIV-4425).
- R2-ENGINE-621: of the owner's second-game saves, game0006 alone restores a
  projectile.
- R2-ENGINE-662, R2-ENGINE-663, R2-SESSION-259: the hero's saved command
  selectors. The first action they deliver is Unknown (DIV-4426).

## As built

- `pkg/game/savdocument.go`: `decodeSavedRecord` returns the decoded
  document and its origins as a `heldDocument` when the document decodes but
  `savedFormationBlock` refuses it. A second-game profile places no
  PlayerFormation, so its LOAD keeps a partial `SnapshotSAVDocument`.
  `savedActorBindings` is the actor binding factored out of
  `importSavedDocument`. Both callers use it. The partial state gains
  `LoadedEffects`, the effect family (a document holding the graph records
  and their WorldEffects metadata).
- `pkg/game/originalrestore.go` passes the held document to
  `importOriginalWorldEffects`.
- `pkg/game/savworldeffects.go`: for a partial state with a held document,
  `importOriginalWorldEffects` runs the existing binder on a view whose
  actors come from `savedActorBindings`. It then keeps the family with
  `loadedEffectFamily`. `worldEffectSource` gives the SAVE the family
  of a partial state and the whole document otherwise.
  `leaveUnboundEffectsOut` now drops only effects that nothing bound.
- `pkg/game/worldcarriers.go`: the effect carrier and its key reservation
  read `worldEffectSource`.
- The continuation is the first game's: the transport's child joins the
  list tail and acts on the next tick. Direct damage uses the first game's
  payload rule.
- `cmd/savecheck`: `-witness` prints the effect roots with their nesting, the
  transport countdown, each point's key, target health and payload, and
  each projectile. It prints them after LOAD and after each tick while an
  effect or projectile lives.

## Proof

- Failing, then passing: `TestSecondMissionFlightContinuesAfterLoad` and
  `TestSecondMissionFlightSavesAndColdLoads`
  (`pkg/game/secondmissionflight_test.go`). Both fail at 76b9e59bc and pass at
  the candidate.
- First-game record test: `TestSpellEffect1152IndependentDAGAndNativeCycle`,
  with the `TestSpellEffect1152`, `TestRetiredSpellGraph`,
  `TestProjectiles1157`, `TestWorldEffects1162`, `TestWorld1170` and
  `TestCurrentWorldEffectToken` families, unchanged.
- Witness, read-only, game0006 from the owner's 2026-10-30 second-game
  saves, EN and RU installs (`savecheck -witness 4`, then `-resave` and the
  cold LOAD of the written file). EN and RU print the same lines:
  - After LOAD, tick 2257, World hash 488e60ced459c298: SpellTransport
    countdown 1 nesting PointEffect key 347558336 on actor 13 (hp 30/30),
    payload Effect_DirectDamage 8+8, school 4. Prj16 picture 40 at
    6272/10624, segments 0, target 24598.
  - After tick 1: the PointEffect alone, no projectile.
  - After tick 2: no effect; actor 13 has hp 22/30.
  - Four ticks after LOAD: hash 330d8aab6fe289a1.
  - The SAVE taken right after LOAD writes the same bytes on EN and RU. Its
    cold LOAD prints the same lines and the same two hashes.
  - All 16 owner second-game saves in that set LOAD. Only game0006
    holds effects.
- Release tests for the next release, EN and RU: the first game's
  spell-in-flight and projectile SAV set (`TestReleaseWorldEffectsContinue1162`,
  `TestReleaseRetiredWorldEffectsColdLoad`,
  `TestReleaseMilestone2SpellEffects1152`,
  `TestReleaseCurrentWorldEffectSAV1170`,
  `TestReleaseCurrentWorldProjectileContinuation`,
  `TestReleaseMilestone2Projectiles1157` and the others the result names) and
  the second game's mission-file LOAD set.

## Open debt

- DIV-4175, narrowed: a LOAD whose document did not decode (a first-game
  partial legacy input) binds no effect, and its SAVE drops them.
- DIV-4427: a held document whose effects or projectiles the binder refuses
  (game0026's area cloud with cells) keeps them carried and unbound; the LOAD
  succeeds, the World holds none of the binder's imports, the SAVE drops them.
- DIV-4423: the queued point acts on the next tick.
- DIV-4424: direct damage takes the first game's health rule.
- DIV-4425: Prj16 runs on its own driver and is removed on its first tick.
  Its target 24598 is detached.
- DIV-4426: the hero's saved command (order 4) is not continued.
- A cold LOAD after the transport retired renumbers the graph, because the
  retired node is not written. Its World hash then differs from the live
  one, as on the first game.
