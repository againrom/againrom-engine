# City group membership as current state

## Intent

A city SAVE writes the Player's group membership from the live town. A hire, a
return of a squad, a mission, a companion's arrival and a cold LOAD each leave
the membership the next SAVE writes, over any number of hire cycles, with no
constructed fallback group where the loaded file's groups are readable.

## Authority

- Owner ruling: SAV is the only format and has one producer. A field is written
  from current state, then from a rule derived from evidence, then from the
  loaded document's bytes, then from a research constructor value. A field with
  none of these is named debt, never a refused save.
- `SAV-617`: a tavern hire creates one new group under the Player holding
  exactly the new actors. `MERC-HIRE-003`: one hire flag per type, committed
  as one command naming every hired type. `SAV-GRPLOAD-560`: LOAD appends each
  saved actor to its saved group. `SAV-GRPALLOC-577`: a new group's payload is
  not cleared, so `+1c` has no constructor value. Read from the pinned snapshot
  with `go run ./tools/claim <ID>` and quoted in DIV-1411.
- Grouping of several hire types, siege hire groups and the group a joiner
  enters have no claim. They stay inferences and are named in DIV-1683,
  DIV-1684, DIV-1685.

## As built

Before: `writeCityGroups` rebuilt the groups at each SAVE from the loaded
town's provenance and the current party. A town whose provenance was
unavailable (a source companion matching no installed definition, a population
mismatch) wrote one constructed group of every non-hire member, so a loaded
group order was lost on the next SAVE. DIV-1411 debt (a), (b) and (c) described
this.

After:

- `Town.cityGroups` holds each group's field values and its members by party
  ID (`pkg/game/citygroupcarry.go`).
- LOAD of a current SAV builds it from the file's own groups, binding each
  group actor to its party member through the current supplement's object
  bindings (`cityGroupsFromCurrent`). It needs no original provenance. LOAD of
  an original file builds it through the provenance's identity bindings
  (`cityGroupsFromLoaded`).
- `settleCityGroups` reconciles it against the party at the tavern hire and
  return, at a mission's return, at a companion's arrival and at the end of a
  load. A member who left leaves its group, an emptied group is dropped, a
  member in no group enters the first group (a hire enters the group of its
  mercenary type, appended in party order).
- `writeCityGroups` writes the reconciled groups with their own field values.
  Snapshot capture and the mission-to-city snapshot carry the membership.
- The AGS reader is not used.

## Obligation 11: error returns of `originalcity_current.go`

Instrument: `git grep` over `f9786837` for every caller of each function, in
Go sources including tests.

| line | return | function | callers |
|---|---|---|---|
| 62, 68 | Human identity repeats or is missing | `cityDocumentHuman` | `memberItems` only |
| 216, 232 | item record does not resolve; graph overflow | `copyCityRecord` | `construct`, `writeValue` only |
| 268, 330 | written item differs; graph overflow | `write`, `construct` | `currentGraphs` only |
| 416, 419, 422, 433, 438, 451 | return integrity, historical placement, skill XP, no Human, current items | `currentUpdate` | none |
| 485 | document has no unit | `derivedHuman` | `currentUpdate` only |

`memberItems` and `currentGraphs` are called only from `currentUpdate`, which
has no caller. The city SAVE route (`ExportCurrentSave`, `currentCityBase`,
`currentCityDocument`) reaches none of them, so no player action reaches any of
the six named returns. The unreachable chain is removed. `currentCityHuman`
and `replayedMember`, which production calls, stay. The route's own error
returns (a Gold outside uint32, an actor-construction mismatch) are not in this
file and are not claimed by this table.

## Proof

Receipts are in `review/story1258-sav-m8/`.

- `TestReleaseCityGroupsRepeatedHireCyclesF2SAV` (EN and RU): three cycles of
  seed SAVE, cold LOAD through the main menu, hire, F2 SAVE, cold LOAD,
  mission entry on the live and the cold town, mission return. The first load
  adds the companion. Hire plans are `14, 1, 14, 6`, `14, 6, 10, 6, 1` and
  `6, 2, 14` over Human types and siege types 1 and 2, covering siege before
  and after Human and a squad returned before the visit ends. Each F2 SAVE
  holds one group per unhired member and one group per hired squad in hire
  order; the cold town holds the same membership and field values; a second
  SAVE from the cold town writes the same groups; World hashes of the live and
  the cold mission agree for four ticks after each entry. Loss control: with
  the live membership dropped the same SAVE folds the party into one group and
  the comparison fails, in every cycle.
- `TestCityGroupsSurviveUnavailableProvenance`: the retained
  GROUP-COUNTEREXAMPLE input (synthetic source companion with no matching
  definition, ten hires, loaded group order `[Companion Leader]`) keeps its
  groups over three SAVE and LOAD cycles with provenance unavailable on every
  load. Loss control: dropping the live membership writes `[Leader Companion]`.
- Unit tests `TestReconcileCityGroupsFollowsPartyChanges`,
  `TestReconcileCityGroupsDropsAnEmptiedGroup` and
  `TestCityGroupsFromCurrentBindActorsThroughTheDocument`.
- The existing group, roster, hired identity and native-town release tests pass
  on both roots (88 tests per root, no skip).

## Open debt

DIV-1411 is narrowed to the live-state mechanism. DIV-1683 (joiner placement),
DIV-1684 (several hire types and siege groups), DIV-1685 (an emptied group) and
DIV-1686 (a file with no readable group record) name what remains. Original
acceptance of a multi-type hire SAVE and of a join stays with the owner kits.
