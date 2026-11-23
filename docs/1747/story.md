# Second-game script registers 0..999, checks 20 and 22, and action 1

## Result

In the second game, mission scripts now run with all one thousand registers from zero through nine hundred ninety-nine,
starting each session with register 901 set to one and counting each script tick in register 999 instead of register 93.
Saving and loading a second-game mission preserves registers 100 through 999 in both the native save block and the
engine document. Second-game maps can also evaluate check 20 to count a player's standing buildings, check 22 to test
and relocate a ground unit on a cell, and action 1 to increment register 902, while the first game stays unchanged.

## Authority

- `R2-SESSION-265`: session start sets registers 900 and 902 to 0, 901 to 1, and 999 to 0.
- `R2-ENGINE-1451`: every second-game script tick adds 1 to register 999.
- `R2-ENGINE-1450`: check 22 tests cell `p1*256+p0` and relocates its occupant to `(low8 p2, low8 p3)`
  returning 1 else 0; action 1 adds 1 to session `+0xc710` and register 902.
- `R2-ENGINE-1452`: check 20 counts structures of player P whose signed word at `+0x42` is greater than 0.
- `R2-ENGINE-1288`, `R2-ENGINE-1010`, `SAV-OWNER-048`: ALM type-4 word `+0x0e` and a saved Player key give
  `Structure.Owner`; that it equals check 20's player number is Medium.
- `R2-ASSET-247`: installed campaign maps use neither check 20 nor check 22, though custom maps may.
- `DIV-4352` (updated): second-game mission SAV writes and restores registers 100..999; `+0xa4c` and
  `+0xa8c0` remain zero.
- `DIV-5097`: session counter `+0xc710` has no claimed reader and is neither kept nor saved.
- Owner remarks 7, 8, 10: register count and tick register are profile data, not game-identity branches; checks 20,
  22 and action 1 use the second-game row builder; invent no meaning for `+0xc710`.

## Rules

1. A second-game session runs with 1000 registers, starts 901 at 1, and ticks 999 on every script tick, program or not.
2. Second-game checks, triggers, and variable actions read and write registers 0..999 and ignore out-of-range slots.
3. The first game stays unchanged at 100 registers, increments register 93 on each pass, and rejects wide registers.
4. Register count and tick register come from `scriptprofile.Profile` data rather than branches on game identity.
5. Second-game mission SAV writes registers 100..999 from live state and restores them on both leaf and native loads.
6. Second-game check 20 counts a player's structures with positive signed `Field42` and stays unsupported in ROM1.
7. Second-game check 22 tests a ground occupant on `p1*256+p0`, relocates it to `(p2&0xff, p3&0xff)`, returns 1 else 0.
8. Second-game action 1 adds 1 to register 902 and stays unsupported in ROM1; actions 11, 27, and 31 stay gaps.
9. Only the second game writes `Structure.Owner` (ALM `+0x0e`) to the World form; the first game's form is unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| script profile limits | `pkg/scriptprofile/profile.go` `Profile` | added `TickRegister` (93 first/extended, 999 second) and `ForOps` |
| world game profile | `pkg/sim/gameprofile.go` `newGameProfile` | added `dialect`, `scriptProfile`, `scriptSeed`, `scriptLimits`, `Dialect.Profile` |
| script runtime | `pkg/sim/script.go` `newScriptOn`, `scriptPassObserved` | sized registers and tick slot from profile; seeded 901=1; added `ScriptRegistersBeyond` |
| second-game script ops | `pkg/sim/rom2script.go` `secondGameScriptRows` | added rows for checks 20 and 22 and action 1 |
| world byte form | `pkg/sim/formwrapper.go` `formWrappers` | added optional wrappers `WRG1` (142) for wide registers and `SOW1` (143) for structure owners |
| structure owner | `pkg/mapload/fromalm.go` `Structures`, `pkg/game/originalstructures.go`, `pkg/sim/structureownerbinary.go` | populated `Structure.Owner` from ALM `+0x0e` and SAV Player references; the form writes it by profile `structureOwners` |
| SAV session registers | `pkg/formats/sav/engine_form.go`, `session.go`, `pkg/game/savform.go`, `originalsave.go` | wrote live `SessionResults` into `Bsg&`, read `WideRegisters` and `CurrentROM2Policy.Registers` |

## Proof

| test | rules |
|---|---|
| `TestProfilesHoldTheEnginesLimits`, `TestExtendedProfileHoldsTheOwnersLimits` | 4 |
| `TestSecondGameSessionStartRegistersAndTick999`, `TestSecondGameEmptyProgramCountsRegister999` | 1, 4 |
| `TestSecondGameWideRegistersReadWriteAndRoundTrip` | 1, 2, 5 |
| `TestFirstGameKeepsHundredRegistersAndSlot93Tick` | 3, 4 |
| `TestSecondGameCheck20CountsPlayerStructuresWithPositiveHealth` | 6 |
| `TestSecondGameCheck22CellObjectRelocatesAndReturnsOneOrZero` | 7 |
| `TestSecondGameAction1IncrementsRegister902AndGapsRemain` | 8 |
| `TestScriptOpRowsAreTheSecondGameOps` | 6, 7, 8 |
| `TestTheWorldAsksItsGameOnce`, `TestScriptOperationReadersHaveNoGameBranch` | 4 |
| `TestSecondMissionSessionResultsRoundTripFromProfileKeeps` | 5 |
| `TestSecondMissionSaveRoundTripsRegisters900To999`, `TestSecondMissionSaveWritesTickRegisterWithEmptyProgram` | 1, 5, 8 |
| `TestReleaseOriginalControlResaveUnchanged` (EN, RU) | 3, 9 |

Release round trip on the installs: `TestReleaseSecondOldMissionSaveTakesKindWordsFromRecords` (rom2-en, rom2-ru).

## Open debt

- Session counter `+0xc710` incremented by action 1 (`DIV-5097`): only `R2-ENGINE-1450` mentions it, it lies outside
  the 10,374-byte SAV session block (`R2-SESSION-019`), and no reader is claimed; waits for a claim naming one.
- Second-game script actions 11, 27, and 31 remain unsupported gaps (`ScriptGapInstant`) in `pkg/sim`; wait
  for dedicated second-game script action stories.
- `DIV-4352` fields `+0xa4c` and `+0xa8c0` remain zero in second-game mission SAV.
- Check 20 with player 0: the engine counts ownerless structures for player 0; the original may resolve slot 0 to no
  player. No claim reads it.
- Check 22 "present" (`R2-ENGINE-1450`, helper `L2.07790`, Medium): read here as a cell holding an occupant, a
  body with 0 health included; no claim settles whether a dead body counts.
