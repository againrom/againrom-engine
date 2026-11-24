# The second-game map editor builds, reads and changes a teleport

## Result

In the second game's trigger window a cell event action of event 23 reads as
a teleport: its entry cell, its exit cell, the source cell shown as the entry
cell, and the power named as unread. The map maker changes the entry and exit
cells with a map click and picks a new teleport from the operation list, which
writes the operands the shipped teleports hold.

On a second-game map the editor adds, changes and removes a type-9 record of
event 23 as a teleport from its cell to its target. A new one saves as the
original's six-byte cell event with power 0; a loaded one keeps its power.

Game rule: a unit moves by a teleport when its step ends at the centre of its
own cell and that cell holds event 23, not on the tick its stride crosses into
any cell under its footprint.

## Authority

- Owner: «нужно понять как работает телепорт в rom2, миссия 53.alm и уметь это настраивать».
- R2-ENGINE-1775: action 25 writes event p0, power p1, source and target on cell (p2, p3).
- R2-ENGINE-1776: one event per cell, a later write replaces it; two writers.
- R2-ENGINE-1778: the step body at the centre reads the unit's own cell and calls the cell move.
- R2-ENGINE-1779 (Medium): power and source are not read for event 23.
- R2-ASSET-386, R2-ASSET-387: mission 53's node and every shipped teleport; no type-9 power default.

## Rules

1. When an action 25 node holds event 23, it reads as a teleport: entry, source and exit cell, unread power.
2. When the source cell equals the entry cell, it shows as the entry cell and Apply writes it from that cell.
3. When the teleport is picked from the operation list, the node takes event 23, power 255, the entry as source.
4. When the plain action 25 is picked again, the event clears and every slot shows as before.
5. When a unit's step ends centred on a cell of event 23, it moves to the target; the event stays.
6. When a unit's stride crosses into such a cell, or its footprint covers it beside its own cell, it is not moved.
7. When a teleport is added on a second-game map, it saves as event 23, power 0, source at its cell, the target given.
8. When a type-9 teleport is changed, its cell and target change and its event stays 23; another event is refused.
9. When a type-9 teleport is removed, its cell event goes and the other records stay.
10. When a type-9 record is not a cell record of event 23 or 0, it lists no field and the palette does not offer it.

## Code

| kind | its one builder | change |
|---|---|---|
| second-game operation rows | `pkg/scriptprofile/ops.go` secondActions, Op.Slots | action 25 gains a teleport form |
| trigger window node | `pkg/ui/mapeditortrigger.go` nodeFields, retypeNode, scriptValues | form fields, retype, following cell |
| type-9 record section | `pkg/mapedit/enchantment.go` enchantmentSection | both formats' rows built from it with their field table |
| second-game type-9 fields | `pkg/mapedit/second.go` secondEnchantmentFields | the teleport's five fields |
| format edit row | `pkg/game/mapeditrow.go` editRow | second row admits KindEnchantment |
| type-9 palette and choices | `pkg/game` cellSpellPalette, castSpellChoices | cell record only; event 23 as Teleport |
| type-9 record view | `pkg/game/mapinspectioncasts.go` inspectionEnchantment | teleport label from CellMove |
| cell event rule | `pkg/sim/cellentry.go` cellEventBodies | centred body; entry queues nothing |

## Proof

| test | rules |
|---|---|
| `TestSecondGameActionTwentyFiveEventTwentyThreeReadsAsTeleport` | 1, 2 |
| `TestSecondGameTeleportBuiltFromTheOperationList` | 3, 4 |
| `TestSecondGameCellMoveWaitsForTheStepsCentre` | 5, 6 |
| `TestSecondGameCellMoveReadsTheUnitsOwnCell` | 6 |
| `TestSecondGameTrapMovesTheUnitWhoseStepEndsOnIt` | 5 |
| `TestSecondMapTeleportRecordAddedChangedAndRemoved` | 7, 8, 9, 10 |

Release witness, EN and RU: `TestReleaseSecondMapSavesEveryInstalledMapUnchanged`.

## Open debt

- The executor retry through command word +0xb0 and event 23 on air units are not modelled (DIV-5154).
- Order paths other than the walk's step are not hooked; the saved-motion path is not hooked.
- The bit-0 refusal of +0x20000 in the writer is not modelled (R2-ENGINE-1776).
- A second-game map with no type-9 section takes no new teleport; the section has no constructor.
- The type-9 record list's teleport label has no focused test; the spell step's Teleport label has one.
- Unknown: whether the original plays an edited teleport; not testable headless.
