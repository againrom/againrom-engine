# The second game's building casters

## Intent

In the second game the buildings a map binds with a type-9 record of mode mask 4
cast at units. A destroyed building stops its casters. The list survives a save
and a load. The first game does not change.

## Authority

Owner direction, verbatim in `DIV-4718`: the towers of mission 72 («Исчезнувший
караван», `scenario/72.alm`) must attack. Claims at the k444 pin:
`R2-ENGINE-1006` (construction), `R2-ENGINE-1007` (turn pass), `R2-ENGINE-1008`
(selection), `R2-ENGINE-1009` (fields), `R2-ENGINE-1010` (owner word and
removal), `R2-ENGINE-1011` (document list), `R2-ENGINE-054` (the active-effect
word), `R2-ASSET-230` (the type-9 grammar).

## As-built behaviour

Claimed behaviour: the construction gates (mask 4, nonzero cell, no sack, a key
naming a building), the once-per-turn pass, the one draw among survivors, the
spell and power bytes, the removal on a hit not below the building's health over
the cell word, and the list as the document's last list.

Decisions under the owner's direction, each listed in `DIV-4718` and open until
the bodies are read:

- The record's byte +0x00 is a reach in cells, a Chebyshev square. The fill
  routine is unread (`R2-ENGINE-1008`).
- An attack spell keeps units hostile to the owner, a defensive spell the
  others. `R2-ENGINE-1008` gives the comparison; the meaning of the spell byte
  and of the relation bit is Unknown.
- The cast runs through the engine's script-cast route, no mana, no cooldown.
  The cast body is unread.
- The owner is the low word of the building's type-4 `+0x0e`, read as a roster
  slot as a unit's owner word is. `R2-ENGINE-1010` resolves the word to a list
  object; `R2-ENGINE-1009` is silent on owner.
- Bit 0x1000 of the unit's active-effect word is read as attached spell 12
  (inference, Medium).
- A physical blow removes casters as a spell hit does. `R2-ENGINE-1010` names
  the resolver; the owner's direction is that a destroyed building stops its
  casters.
- A file whose list is empty on a map with such records rebuilds from the map
  for the buildings still alive at load. `R2-ENGINE-1011` leaves the original's
  behaviour Unknown.

Persistence: the world's list goes to the `Bsg&` file through the one document
producer (`secondFormKeeps`, `DocumentInProfile`) and to the engine document's
actions leaf, whose `BuildingCastersWritten` field separates an emptied list
from a save of an older build. The LOAD of an original file reads the
VirtualCaster records (`sav.VirtualCasters`); other classes in the list after
the Sacks are read past and not carried (`DIV-4172`). The refusal of a file with
a Token in that list is gone.

## Proof

Headless measurement, `cmd/missionrun -towers`, mission 72, both roots, one
hero, written to the lane's notes directory. Before the fix the world held 0
casters of 27 records: `mapload.startMissionScripted` rebuilds the world and did
not carry the list. After the fix: 27 casters on `rom2-en` and `rom2-ru`, and
byte-identical logs on the two roots.

| Run | First cast | Tower, spell, power | Casts at the hero |
|---|---|---|---|
| Idle at the start, 3200 ticks | none | none | 0 |
| Walk to (23,109) | tick 231 | (22,111), 10, 255 | 3 at ticks 231, 247, 263; hero's health before each 145, 79, 24, and the hero fell |
| Walk to (33,116) | tick 231 | (33,111), 6, 255 | one every 16 ticks while the hero stood in reach; hero's health 145 at the first, 133 at the second, back to 145 by tick 471 |

Fixture tests name the route they prove and the base's result:
`TestBuildingCastersSackBindingGate`, `TestBuildingCastersKeyOwnerAndCellWord`,
`TestBuildingCasterStepQueuesOnTheTurnPhaseOnly`,
`TestBuildingCasterSkipsTheExcludedEffect`,
`TestBuildingCasterRemovedThroughBothHitPaths`,
`TestBuildingCasterSAVWriteThenReadWithLossControls`,
`TestBuildingCasterFreshEntryCarriesTheMapsList` (with the rebuild's line removed
it fails: a fresh entry holds an empty list),
`TestBuildingCasterEmptyFileListRebuildsFromTheMap`,
`TestApplyOriginalBuildingCastersRules`. The base does not compile them.

The actions leaf's new field moves the bytes of a second-game SAVE:
`pkg/game/testdata/missionmessages.txt`, the pinned delivery record, moved on 91
of its 226 lines, each a second-game save hash; no first-game line moved.

Release test owed, not run in the lane, no root column on its population row:
`TestReleaseSecondMissionTowersCastAtTheParty`.

## Open debt

`DIV-4718`: the fill and cast bodies, the empty-list rebuild, the owner word's
list object, and the VirtualCaster Token words written as zero. `DIV-4719` was
reserved and is returned unused.
