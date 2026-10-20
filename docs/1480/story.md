# Engine state after an original resave

## Intent and authority

The original game drops the engine-only store leaves `/CurrentState/AgainromActions`,
`AgainromRng` and `AgainromSeed` when it resaves an engine SAV (SAV audit G1). Owner decision,
`pipeline/BACKLOG.md` item 16: on LOAD, derive what the original's own bytes carry; what cannot
be derived is named debt. The DIV-2342 document carrier is not used. A derivation cites a
promoted claim or a rule from evidence in hand; Againrom code is not ROM1 evidence.

One decision says "this file has no engine-state block": `readCurrentActions`
(`pkg/game/savactioncurrent.go`) returns nil when `sav.NativeActions` finds no leaf, and every
LOAD route reads the session, the actions and the document through it. This story adds no LOAD
route.

G20 (DIV-2761) was a writer fault: a native Human without retained load state was written with
the pack running weight summed from instance weights, so a pack of table-weighted items was
written as 0. SAVE now writes the world's container sum (`World.ContainerWeight`,
`pkg/sim/weight.go`) from `pkg/game/savcurrentitems.go` and from the mission-open seed in
`pkg/game/citymissionseed.go`. `ITEM-STACK-003` (High): the container keeps a running sum of
weight times count over its elements; `SAV-794` (promoted): LOAD keeps the stored word.

## Field by field

| Field | Without the leaf | Authority or row |
|---|---|---|
| Taken tavern, shop and school offers | derived: the shortened InnMission/InnNPC, ShopMission and TCMission arrays | `SAV-CAMPAIGN-081`, `TAVERN-024` (promoted); the engine already wrote them |
| Done main missions | derived: every main below the campaign record's current main | `SAV-CAMPAIGN-077` |
| Won list: done side missions, finished count | debt: lost; the finished count loads 0 and feeds the shop stock seed | `DIV-3813` |
| Offered | debt: loads 0; written again at the next mission end before any read | `DIV-3815`, `DIV-1322` |
| DocumentMission | debt: loads the current main; a restored campaign does not read it | `DIV-3815` |
| Live load speed | derived: the saved speed and load words are kept | `SAV-794` (promoted); fixed before this story, `TestReleaseOverloadedHumanSpeedAfterOriginalTownLoad` |
| Pack running weight (G20) | derived: the saved word, now the container sum | `ITEM-STACK-003`, `SAV-794`; `DIV-2761` closed |
| Quick spells bound to an original spell | derived: the Shortcuts leaf | `AI-QUICKSAVE-281` |
| Quick spell without an original book cell | debt: the slot loads empty | `DIV-3814` |
| Fog, explored and visible planes | derived through the original-file route | `DIV-805` |
| Script registers, counters and program | derived through the original-file route; the program from the installed map | `DIV-1369` |
| Cast before its release | debt: lost; the caster stands | `DIV-3816` |
| Actor motion, order and patrol | debt: rebuilt from the ordinary records, not continued | `DIV-3817` |
| Random session (`AgainromSeed`, `AgainromRng`) | seeded from the file's own bytes | `DIV-2736`, `DIV-1202` (owner-ruled) |

## Proof

- `TestReleaseOriginalResaveReloadsTheSameTown` reads the owner pair `citysaved.sav` and
  `cityresaved.sav` from `AGAINROM_CITY_RESAVE_DIR` (skips when unset) and compares one town
  view: chapter, selection, gold, every building's offers, the tavern roster, done mains,
  documents, mercenary pools, quick spells and each member's sheet speed, weight, health and
  mana. The view is equal. The log names the debt: done sides `[131]`, finished 2 with the
  leaf; `[]`, 0 without.
- `TestReleaseLeaflessTownReloadKeepsTheTown` takes a tavern offer in chapter 50 and a shop and
  a school offer in chapter 130, saves, removes the three leaves (`withoutEngineState`) and
  compares the view of the two reloads. Loss control: the leafless file with the chapter's
  registry arrays in place of the shortened ones must reload a different view.
- `TestReleaseLeaflessReloadKeepsTableWeightedPackLoad`, mission 101: the SAV running weight
  equals the container sum; the leafless reload keeps the running weight and the load after
  one tick. Loss control: the instance-weight word reloads a different running weight.
- `TestReleaseOverloadedHastedHeroMovesInTheClaimedOrder` and
  `TestReleaseNativeAndOriginalHumanShareOneDerivedBlock` now assert the written running weight
  instead of patching it into the file.

Measured on EN with test-only instruments, not committed: an engine resave of each owner file
writes speeds 20, 20, 18 and 16, equal to the owner's original resave. Cast and motion figures
are in `DIV-3816` and `DIV-3817`.

## Open debt

- `DIV-3813`: done side missions and the finished count after a leafless LOAD.
- `DIV-3814`: a quick spell outside the original 24-cell map.
- `DIV-3815`: Offered and DocumentMission take constructor values.
- `DIV-3816`: a cast before its release.
- `DIV-3817`: actor motion, order and patrol continuation.
- Observed, outside the named fields: in the owner pair a weapon's name reads
  "Meteoric Long Sword" with the leaf and "Very Rare Meteoric Long Sword" without it. Unknown
  which side the original shows.
