# One SAVE capture

## Intent and authority

Owner decision on architecture audit rank 3 (SAVE capture transforms): make
the SAVE capture one builder. This is a unification; behaviour stays as it
was. What each route writes stays as `SAV-1251`, `SAV-1234` and
`docs/1477/story.md` set it; DIV-3790 and DIV-1685 hold the difference from
the original.

| Claim | Confidence | Used for |
|---|---|---|
| `SAV-1251` | High predicate / Medium population | SAVE removes the empty Groups of a `+28`-zero Player from the live list before the file is built |
| `SAV-1234` | High | the serializers write every Group still listed, an empty one empty |

## As built

### The capture

`FrontEnd.captureSave(mode, onMap, build)` (`pkg/game/savecapture.go`) is the
one SAVE capture. It applies the mode's transforms, calls `FrontEnd.Snapshot`,
and calls `build` with the snapshot and its label while the transforms hold.
A nil `build` applies the transforms and captures nothing; the save dialog
uses it to commit the prune once its file is written. An unknown mode is
refused before any transform.

`saveCaptureTransforms` is the one table:

| Mode | Routes | Prune live Groups (SAV-1251) | Restore after build |
|---|---|---|---|
| `captureTentative` | save dialog `Prepare`, before the file is written | yes | yes |
| `captureCommitted` | game-menu SAVE (`SaveSeams`), quicksave (F4), save dialog `Commit` after a successful write (nil build) | yes | no |
| `captureTimer` | mission-start autosave, timed autosave, the mod route's waiting-party stash, the fidelity and speech-focus witnesses | no | no |

The prune is the one before this story: on a mission map the World's
`PruneGroupsForSaveUndo`, in the town `Town.pruneCityGroupsForSave`; each
applies only to lists a LOAD set.

Gone: `playerSnapshot`, `playerSnapshotUnwritten`, the dialog's own
post-write `pruneGroupsForSave` call and `World.saveDropsGroup`.
`pruneGroupsForSave` remains as the private transform step only
`captureSave` calls; it now returns only the undo.

### The Group drop rule

`sim.KeepGroupsForSave` (`pkg/sim/savedgroupplayers.go`) is the one applier of
`sim.SaveKeepsGroup`. The World prune (`PruneGroupsForSaveUndo`), the town
prune (`Town.pruneCityGroupsForSave`, participant word zero) and the town
writer (`writeCityGroups`, the written Player's word, only when a LOAD set
the groups) decide through it.

### The rule that holds it

`internal/archtest/savecapture.go` (`CheckSaveCapture`) names, for each
capture and drop function, the production functions that may reference it:
`FrontEnd.Snapshot` and `pruneGroupsForSave` only from `captureSave`; the two
live prunes only from `pruneGroupsForSave` (and the sim delegate);
`SaveKeepsGroup` only from `KeepGroupsForSave`. One listed exception:
`cmd/scenariofixture.prepare` builds a fixture with the exported `Snapshot`,
without a transform, as the timer mode does. `openCityBase`
(`pkg/game/missionports.go`) reads the game through its
`gameSnapshotSource` interface to project a city graph; it writes no file and
is not a SAVE route.

## Proof

### Focused tests, each with a loss control

- `TestSaveCaptureModesOnTheWorld` (`pkg/game/savecapture_test.go`): on a
  LOADed World whose human Player has `+28` zero, a tentative capture builds
  from the pruned lists and leaves the live lists and World hash as they were;
  a committed capture leaves them pruned; a timer capture changes nothing.
  Loss controls: each mode's row replaced in turn by a wrong row (tentative
  without restore, committed with restore, timer with prune) makes exactly
  one of the three checks fail.
- `TestSaveCaptureModesOnTheTown`: a LOADed town's emptied group leaves the
  live list only under the committed mode; the control, a town no LOAD set,
  keeps it under every mode.
- `TestSaveCaptureRefusesAnUnknownMode`: mode 0 builds nothing and changes
  nothing.
- `TestSaveDialogPrunesOnlyWhenTheFileIsWritten`,
  `TestMissionSaveDropsOnlyParticipantZeroEmptyGroups`,
  `TestCitySaveDropsAnEmptiedGroupOfTheHumanPlayer` and
  `TestCitySaveRemovesAnEmptiedGroupFromALoadedTown` pass unchanged in
  substance; the second now captures through `captureSave`.
- `TestSaveRoutesCaptureThroughOneFunction` (`internal/archtest`) runs the
  rule over every production package of the module and checks that each
  named function and owner exists. `TestSaveCaptureRuleRejectsAFourthVariant`
  is its loss control: eight synthetic packages; the route through the
  capture passes, and a route that calls `Snapshot`, holds it as a value,
  calls the prune itself, prunes the World or the town, or applies
  `SaveKeepsGroup` in the game or a second time in sim, is reported.

### Unchanged against 86ee5494

| Instrument | Population | Result |
|---|---|---|
| `pipeline/check-release-tests.sh`, `AGAINROM_HEADLESS_ONLY=1`, EN then RU, at 28f8cdeb | 1,340 gated tests per root | all pass but two per root, both refused by environment (below); 1 skipped by `AGAINROM_HEADLESS_ONLY` |
| `scripts/check-milestone2-acceptance.sh` EN, RU at 28f8cdeb, and at a `git archive` of 86ee5494 | 50 instruments per root | exit 0, 0 refusals; the 3,287 sorted instrument lines per root equal base's except 8 timing lines |
| `pipeline/check-mission-scenarios.sh` EN+RU at 28f8cdeb and at 86ee5494 | 76 runs | 76 of 76 both; all 76 logs byte-identical, World hashes included; no run writes a save |
| ROM2 SAVE selection of `pipeline/check-rom2-save-tests.sh`, by hand, EN and RU | 107 `pkg/game` + 3 `pkg/secondshop`/`pkg/video/smacker` tests per locale | all pass, none skipped |

The release-set refusals: `TestReleaseCompletedExtraTavernTalkerDisappearsThroughAppAndCurrentSAV`
and `TestReleaseKeptTurtleOfferAvailabilitySurvivesCurrentSAV` accept outputs
only under their own fixed review directories, and the lane pointed every
output variable under its notes directory; each fails at that check before
any game work.

The second game shares the changed code: its SAVE routes are the same
`pkg/game` routes and call `captureSave`. Nothing moved for it.

## Open debt

- A player SAVE and a timer SAVE of the same LOADed mission write different
  Group lists: the player's file drops the empty Groups of a `+28`-zero
  Player, the autosave's file keeps them. DIV-3790 already holds this; no
  new row. In the town the files agree, because the town writer applies the
  rule in every mode; only the live list differs after a timer SAVE until the
  next player SAVE.
- `cmd/scenariofixture` captures with the exported `Snapshot`; it is a
  fixture tool, listed in the rule as the one exception.
