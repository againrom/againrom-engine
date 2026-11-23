# The second game's town save fills more of the original's fields

## Result

A town save of the second game now writes the fields the original's town files hold, where the engine
used to write a constant or leave them empty. Each member's experience word follows his Humans row,
the player's kill count follows the kills recorded in the save, and the saved selection names the hero.
A save loaded from an original keeps the player's raw block. The first game's town save is unchanged.

## Authority

- Owner: a field with no source is ledger debt, never a guess, and never refuses a save.
- R2-ENGINE-320: template load writes a Human's experience word from his row.
- R2-ENGINE-772: a kill of a victim outside class 0x21..0x3f adds 1 to the owner's +0xa48 with no cap; a victim in
  that class adds 1 to +0xa4c; the victim's grade byte rises only below 14.
- R2-ENGINE-675: a new Player's bytes are zero.
- R2-SESSION-322: the town files differ between stages in the Player words and the selection.
- DIV-4305, DIV-4391: the rows that name these fields.

## Rules

1. A town save writes every party member's experience word from his Humans row.
2. A town save writes the player's kill count uncapped, outside class 0x21..0x3f, with grade bytes at most 14.
3. A town save writes an item's word at +0xe as 0.
4. A town save writes the hero's key as its selection.
5. A player block loaded from an original is written back unchanged, also after a load of records alone.

## Code

| kind | its one builder | change |
|---|---|---|
| second-game town record writer | `pkg/game/savform.go` secondFormHolders | holders and a raw-block site as data |
| loaded campaign tail | `pkg/game/currentsecondcampaign.go` secondTailKept | the player's raw block kept |

## Proof

| test | rules |
|---|---|
| `TestSecondTownSaveWritesTheClaimedFields` | 1, 3, 4 |
| `TestSecondSaveKillCountPastTheGradeCap` | 2 |
| `TestSecondTownSaveWritesTheLoadedPlayerBlock` | 5 |

Release tests at the release: none.

## Open debt

- The other fields of DIV-4305, DIV-4391, DIV-4324, DIV-4306 and DIV-4416 wait for claims on the town writer.
