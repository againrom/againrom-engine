# Terminal loot verification

## Focused contracts

On base `972e37f`, `TestTerminalLoot*` reproduces blocked-origin drops in all
four terminal producers and missing no-cell retry. The corrected simulation
run covers native/source ownership, static/dynamic blockers, magic walls,
occupied footprints, existing sacks, rectangle edges and expanded searches.
It checks flying-body removal, original item identity, no-cell RNG retention
and cold continuation. Ordinary same-cell merging, suppression and source
teardown controls pass separately.

`TestCurrentTerminalLootRelocationKeepsSAVOwnersAndNextTicks` checks two actual
SAV cycles and 16 next ticks per cycle. Ordinary Sack position/gold and the
original weapon reference are checked before LOAD. World hash, body position,
object identity and the new Sack token survive both cycles.

`TestCurrentTerminalLootWithoutGroundKeepsPendingSAVDeath` saves a terminal
body with no free ground, compares cold continuation, opens one cell through
an ordinary SAV Block edit, then observes the retained weapon transfer.

After reconciling `main` at `8af74d4`, the final focused run passes: simulation
0.071 s, both SAV tests 0.514 s, and `TestLiveTreeClean` 1.988 s. The command is
`go test -trimpath -count=1 ./pkg/sim ./pkg/game ./internal/storyguard -run 'Test(TerminalLoot|CurrentTerminalLoot|LiveTreeClean)'`.
The resource receipt is
`review/owner-dragon-loot-141/story1234-focused-final.json` outside the
repository. It records 78.879 s including compilation, four logical CPUs,
BelowNormal priority, exit code zero, no timeout and zero remaining child
processes. `go fmt ./pkg/sim ./pkg/game` passes through the same bounded
launcher. New tests use version-free fixture helpers; the legacy-identifier
and comment budgets do not rise.

## Installed observable result

The EN `TestReleaseTerminalLootGroundCurrentSAV` uses the owner's
`save141.sav`, SHA-256
`94105d209cc27eba5086d78bab4c883315824e8bab8d5059eb53c8c522f157ab`.
Its actual map is mission 151. A script-health write and ordinary ticks kill
flyer 113, type 71, on (86,9), with source Block masks `21/a1`. The new sack
lands on (83,12) with 3,103,807 gold. All seven existing sacks stay unchanged.
Ordinary menu SAVE, cold LOAD and four next ticks agree on World hash and sack
value (4.17 s test, 4.750 s package). Input/output ordinary Block records are
the mask oracle; this native-current save has no retained source cell planes.

This witness runs the frontend without assigning a Sack or corpse result. It
does not witness the GUI. It ran before the main reconciliation and the
equivalent rectangle-search enumeration refinement. The later focused
contracts cover that refinement; a final installed run remains a landing gate.

## Remaining gates

No full suite, RU installed gate, milestone-2 chain or GUI drive ran in this
lane. The mission 10/20 script census did not run; no script-count change is
claimed. `builds/current` is unchanged. Review and final gates remain with the
landing coordinator. Original-runtime placement remains unmeasured.
