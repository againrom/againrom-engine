# Second-game altars, group counts, late references, cell-entry source and actions 11 and 27

## Result

In the second game, a hero can now use an altar, which switches it on or off as a lever does, and a well still heals
at once and spends one charge. Script checks now count an unconscious unit as present and stop counting it once it
dies. A trigger naming a late reference is no longer dropped: its actions run with that reference empty and its checks
are skipped. A trap set with no source now strikes from its own cell, and scripts can move an item between two units
and move a unit to a cell, after which a computer unit stands guard there.

## Authority

- Owner: «сделай как в оригинале» for remarks 1, 8, 9 and 10 on the second game.
- `R2-ENGINE-1695`: wells 15 and 16 spend one `+0x42` charge and apply the potion; altars 76 and 77 toggle 0 and 1.
- `R2-ENGINE-1696` (EN High, RU Medium): a dead unit leaves its group; checks 1, 5, 8 count living and unconscious
  units; check 20 counts buildings with `+0x42 > 0`.
- `R2-ENGINE-1697`, `R2-ENGINE-1684`: late references 8000..9999 under types 2, 3, 4 read a session table zeroed at
  start with no writer (EN High, RU Medium); the check loop skips a marked check (High; the copied mark Medium).
- `R2-ENGINE-1698` (Medium for action 25, EN read): zero source bytes put the cast at cell (0, 0), aimed at the entering
  unit; action 25 with p4 zero writes its own cell as source and target.
- `R2-ENGINE-1699`, `R2-ENGINE-1700` (High / Medium): action 11 moves one item unit from U's bag to V's; action 27 moves
  the unit, then a computer owner's unit takes Guard even when the move was refused.
- `DIV-2355` (narrowed): late references keep their nodes. `DIV-5103` (new): action 31 needs an Outpost producer.

## Rules

1. A second-game altar 76 or 77 is usable and toggles `+0x42` between 0 and 1; first-game kinds are unchanged.
2. A second-game well spends one charge while `+0x42` is above 0 and heals or restores mana at once.
3. Second-game checks 1, 5 and 8 count a unit until its body reaches the bones; the first game keeps its counts.
4. A second-game node naming 8000..9999 under a group, player or unit parameter is built with that reference absent.
5. A second-game check naming a late reference is skipped undispatched, is no gap, and keeps its mark across SAVE.
6. A second-game cell record with zero source bytes casts from cell (0, 0) at the entering unit.
7. Second-game action 25 with p4 zero stores the node's cell as both source and target.
8. Second-game action 11 moves one unit of the item from U's bag to V's bag.
9. Second-game action 27 moves the unit to (low8 p0, low8 p1); a computer unit then guards its cell, moved or not.
10. Second-game action 31 stays an unsupported gap (`DIV-5103`).

## Code

| kind | its one builder | change |
|---|---|---|
| structure use | `pkg/sim/structureuse.go` `Usable`, `applyStructureUse` | `switchKind` names switches 28, 29 and altars 76, 77 |
| script check counts | `pkg/sim/script.go` `runCheck` | `scriptAlive`, `scriptCountsMember` read profile `countsListed` |
| script compiler | `pkg/mapload/script.go` `bindParamsDialect` | `lateReference` keeps the node; `ScriptReport.Late` |
| script check pass | `pkg/sim/script.go` `runCheck`, `newScriptOn` | `ScriptCheck.Late` skip, `ScriptSilenceLateReference` |
| second-game script state | `pkg/sim/rom2statebinary.go` | optional late-check index tail after the groups |
| cell record bytes | `pkg/sim/celltail.go` `targetedCellTailBytes` | p4 zero writes the node's own cell |
| second-game script ops | `pkg/sim/rom2script.go` `secondGameScriptRows` | rows for actions 11 and 27 |
| script unit order | `pkg/sim/script.go` `standScriptedUnit` | the hand-over stand setter takes its state |

## Proof

| test | rules |
|---|---|
| `TestSecondGameAltarsToggleLikeSwitches`, `TestPointerWithoutAPositiveSlotIsNotUsable` | 1 |
| `TestSecondGameWellSpendsOneChargeAndHealsAtOnce` | 2 |
| `TestSecondGameCountsUnconsciousUnitsAndNotTheDead`, `TestTheArmCountsTheGroupsLivingMembers` | 3 |
| `TestSecondGameCompilerKeepsLateReferencesAndTheirTriggers` | 4 |
| `TestSecondGameCompilerOmitsUnresolvedReferencesAndDependentTriggers` | 4 |
| `TestSecondOmissionsAttributeEachTriggerToItsGreatestCause` | 4 |
| `TestSecondGameSkipsACheckNamingALateReference` | 5 |
| `TestSecondGameCellEntrySourceAndAction25OwnCell` | 6, 7 |
| `TestSecondGameAction11MovesOneItemBetweenBags`, `TestScriptOpRowsAreTheSecondGameOps` | 8 |
| `TestSecondGameAction27RelocatesAndGuardsAComputerUnit` | 9 |
| `TestSecondGameAction1IncrementsRegister902AndGapsRemain` | 10 |
| `TestReleaseOriginalControlResaveUnchanged` (EN, RU) | 3 |

Release tests at the release: none beyond the control resave. Evidence: `review/story1748-rom2-remarks-adopt/`.

## Open debt

- Wells recharge every 60 ticks in both games (`DIV-1031`); `R2-ENGINE-1695` bounds uses at three only if nothing else
  writes `+0x42` (Medium). Waits for a claim on a second-game writer.
- Check 8 counts every unit of the player; the original walks the player's group lists, so a unit in no group is
  counted here. No claim names such a unit.
- Action 27 rejection predicates for blocked or occupied cells are Unknown (`R2-ENGINE-1700`); `relocateActor` decides.
- Action 31 and the Outpost respawn countdown (`DIV-5103`, `R2-ENGINE-1701`, `R2-ENGINE-510`).
