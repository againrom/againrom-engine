# An undying state and a health-below condition for map scripts

## Intent

A map script can keep a unit from dying and react when a unit's health falls below a value the map
chooses, which may be negative. The action "Set undying (Againrom)" sets or clears the state on a
named unit. The condition "Health below (Againrom)" is 1 while the named unit's current health is
below X. The map editor offers both nodes with their parameters and marks a map that holds either
as an Againrom-only map. The game plays such a map, and SAVE and cold LOAD keep the state. A map
without the nodes, the base game of both titles, and every save without an undying unit are
unchanged.

## Authority

- Owner direction: BACKLOG T7 and the brief of this story. The original has no such rule.
- `HERO-DEATH-026`: the death arm is taken on health <= 0.
- `HERO-DWELL-065`: the teardown comes when the dwell has expired and health is at or below -10.
- `HERO-DECAY-069`: a body not yet torn down loses one health point every fourth full tick while
  its health is strictly negative.
- `HERO-ZERO-070`: health 0 is a fixed point of that decay. `HERO-REVIVE-068`: a heal revives.
- `HERO-HEALTH-032`: health is written as a 16-bit subtraction; the field is signed 16-bit.
- `ALM-TRIG-045`, `ALM-TRIG-046`: a node is a 796-byte record with an opcode and ten typed
  parameters; type codes 1..9.
- `TRIG-EVAL-001`: one trigger pass per full tick, on phase 6 of the 16 sub-ticks.
- `TRIG-BIND-010`: a check opcode in 23..0x10001 is built when its references resolve; a
  condition above 0x10002 is never built and aliases the next register (retraction), and an
  action at or above 0x10004 is discarded. `TRIG-COND-003`: a check opcode past the dispatch
  bound writes no register. `TRIG-STORE-002`: registers are zero at session construction.
  `TRIG-ACT-004`: an action opcode 35..0x10001 logs "Script: Bad instant %d" and returns.
- `ALM-SEC-003`: an ALM holds exactly ten sections of the fixed types 0..9.
  `ALM-RDR-059`: four shipped readers with three different acceptance tests.
- Divergence: `DIV-4009` in `docs/divergences/mods.md`. `DIV-4010`, `DIV-4011` and `DIV-4012` are
  unused and returned.

## As built

### The nodes

`pkg/sim/script.go`: `ScriptInstantUndying = 0x4100` (p0 unit, p1 state 1 or 0) and
`ScriptCheckHealthBelow = 0x4101` (p0 unit, p1 X). Both are rows of the one support table pair and
the one dispatch of the trigger evaluator (`runCheck`, `runInstantObserved`). Both opcodes lie in
the ranges the claims give a defined original outcome (check 23..0x10001, action 35..0x10001) and
outside the second game's opcodes 23..27 and 35..39. `AgainromNodeValue`
(`pkg/sim/undying.go`), the one value check, refuses at `NewScript` (and so at SAV LOAD) an X outside
-32768..32767 and a state other than 0 or 1. The binder (`pkg/mapload/script.go`,
`againromNodeNamesUnit`) refuses either node when it names no unit. The condition writes its
register only when the unit resolves; an unresolved unit leaves the register as it stands, as the
other unit checks do.

### What the undying state does

`Entity.Undying` (`pkg/sim/world.go`). While it is set:

- `Alive()` is true and `Dead()` and `Downed()` are false at any health, so no death arm is taken,
  no dwell starts and no teardown follows. The live decay pass skips the unit
  (`pkg/sim/sessionclock.go`), and `setCurrentHealth` floors its health at -32768
  (`pkg/sim/sourceactor.go`), the signed 16-bit field's least value.
- The unit takes orders, walks, fights and casts. It is struck and healed as any unit; a heal adds
  health without a revival, since no death was taken. Rules that test "living" on positive health
  (`Entity.Living()`: picking a fight, reacquiring, using a potion or scroll, withdrawing,
  retreating, skill use, the weapon rider's living-victim flag, Prismatic Spray's refusal of a
  body as its primary) treat it as living. Outside `pkg/sim` the same predicate decides the voice
  acknowledgement speaker (`pkg/game/commandvoice.go`), the local Teleport fog reveal
  (`pkg/game/fog.go`) and the headless scenario's "dead" condition, "alive" assertion and
  alive/fallen counts (`pkg/game/scenario.go`). Without an undying unit each decides as the
  `HP > 0` test it replaces.
- Target pickers that rank by health keep reading health: at health below 1 the engagement and
  escort pickers rank it as a dying body, it is not counted as an engaged foe, and it does not
  hold a guard post (`engage.go`, `escort.go`, `groupengaged.go`, `guardarm.go`). It stays
  ordinarily targetable.
- A blow pays blow experience only while the victim's health stays above -10 (`resolveBlow`,
  `pkg/sim/combat.go`), the base rule for a body. An undying unit at -10 or below therefore pays
  its striker no blow experience.
- Setting the state on a unit that has already taken the death transition changes nothing.
  Clearing it on a unit at health <= 0 runs `clearFelled` in the same trigger pass, as a script
  health write does: the fallen stage at once, then the dwell and teardown of the ordinary path.
- The kill award is the one delayed death processor, `processNewKillCredits`
  (`pkg/sim/itemattribution.go`): a unit not dead at the head of the tick and dead at its tail
  gives its killer the kill experience, the kill credit and the Diary count once. `Step` records an
  undying unit as not dead before the session wrapper's script pass, so a clear below zero in that
  pass is a death of that tick, awarded as a plain unit's death on the blow.

### The condition

`healthBelow` writes 1 while the unit's current health is below X, else 0, in every trigger pass.
A map pairs it with a constant 1 by "equals". The pass runs once per 16 world ticks, on phase 6.

### Persistence

- World byte form: bit 1 of the entity record's always-hits byte (+82), whose bit 0 is
  `AlwaysHits`. A world with no undying unit encodes and hashes as before; the decoder refuses
  any other bit.
- SAV: `ActorValues.Undying` (`pkg/sim/currentvalues.go`) in the native actions leaf, the existing
  Againrom mission-state carrier; `omitempty`, so a save without an undying unit holds the same
  bytes, and an old leaf restores no unit undying. The script program and registers travel in the
  same leaf as before and are recompiled through `NewScript` on LOAD.

### Editor and the mark

- `pkg/mapload/scriptvocabulary.go` and `scriptsignature.go`: names and parameter signatures
  (unit, health; unit, undying); `AgainromScriptOperation` and `ScriptAgainromOnly`.
- `pkg/ui/mapeditortrigger.go`: the node popup shows "Againrom operation: the original game does
  nothing with it." under the operation; the trigger popup shows X signed.
  `pkg/ui/mapeditorwindow.go`: "Againrom-only map" in the status strip; the script notice gains
  "/ Againrom-only map" (`pkg/game/mapinspection.go`).
- `pkg/game/mapinspectionscriptauthor.go`: the editor asks `sim.AgainromNodeValue` on each
  plain field and again on the node a whole set leaves.
- Words: EN and RU in `pkg/words/text/*/strings.toml`.

The mark is derived and adds no byte: a map is Againrom-only exactly when its script holds either
opcode. A header flag or an eleventh section would be bytes every reader must accept, while
`ALM-SEC-003` gives the section set as fixed and `ALM-RDR-059` gives the shipped readers different
acceptance tests. A map without the nodes is written byte for byte as before.

What the original does with such a map, by the cited claims: it reads it as any map; the
health-below check is built and its register stays 0, so a trigger waiting for it to be 1 never
fires; the undying action logs "Script: Bad instant" and does nothing. No unit is ever undying.
Whether the original's other readers (`ALM-RDR-059`) treat the nodes differently is not stated by
the claims; the original was not run.

## Proof

See `RESULT.md` of the lane for the exact runs. Tests:

- `pkg/sim/undying_test.go`: the byte form (one bit moves; hash and bytes restore on clear), the
  SAV leaf (no key without the state; old leaf restores not undying), the health floor, the
  strict comparison, attack and heal at negative health, setting on the dead and clearing at
  negative health.
- `pkg/game/scriptundying_test.go`: a fixture map built through `pkg/mapedit`. The undying unit is
  struck to -20, keeps no corpse stage and walks; struck to -35 on tick 83, health-below -25
  fires and clears on tick 87; the unit falls and is torn down 7 ticks later. Control: the same
  map without the undying action; the unit falls on the blow to -20 and is torn down 7 ticks
  later. Refusals: either node naming no unit; X of -32769 and 32768; state 2.
- `pkg/game/mapeditorundying_test.go`: both nodes authored in the node popups, saved, reopened
  with the nodes, the mark and the note; editor refusals.
- `pkg/game/scriptundying_release_test.go` (EN and RU installs): campaign map 10 edited in the
  editor and played as the guest mission. The hero is undying by tick 32, struck to -20 on tick
  40, walks at negative health; SAVE on tick 80; struck to -35 on tick 81; health-below fires on
  tick 87 with and without the SAVE, and in a separate process after a cold LOAD; the hero falls.

## Open debt

- The second game: the nodes compile in a second-game world, but no second-game map, frame or
  release run exercises them.
- Regeneration at negative health follows the unit's ordinary rules; it was observed (a fixture
  unit gained 2 health in 65 ticks at -20, the hero none in 40 ticks) and not specified.
