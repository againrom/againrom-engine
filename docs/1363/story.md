# Every SAVE is built from the World

## Intent and authority

Owner direction: a loaded state and an engine-made state are one kind of state.
Every SAVE, mission and town, builds its document the way a new game does, and
the copy of the loaded document as a base is removed. Only fields whose meaning
research has not established may take bytes from the loaded file; each such
field kind is listed below with its format page.

Base: public main `a975806f` (game 0.92.0, knowledge k200).

## As built

### Mission SAVE

`materializeCurrentWorld` always builds the mission document from the World,
the map and the tables. The loaded document is kept beside the Snapshot only
for the unknown-meaning spans; `graftUnknownObjects` joins each written object
to its loaded record by World identity (actor entity, Player ID, else a unique
class and identity key) and copies only those spans. A span the World holds is
written from the World: a structure's saved block and token, a registry
Effect's token, and every actor byte or value its native basis holds.

- Player roots: one root per current Player, keyed through its Group
  container; the local human Player is the hero's Player. Original mission
  saves, refused before with "current Player has no exact source or
  constructed root" and then by Group construction, now save on this path.
- Departed actors: the terminal row of a departed actor gains the worn-slot
  mask at departure (`CurrentTerminalActor.Worn`, set when the actor is removed
  and at LOAD from the record's held and worn references). Its record is built
  from its map placement and the tables, then the terminal cell, health and
  stage, the removed native basis and the worn items. On
  `saveorcsdontgo.sav` the input holds 101 Unit records, 54 Weapon records and
  77 dead roots; the SAVE writes 101, 54 and 77, and a cold LOAD of it
  restores the same World hash.
- A departed party member, such as a companion who turned and died, has no
  placement; his record is built from his party entry as a mission start
  constructs him, then the same terminal tuple.
- An actor Diary is written for a party character and for every actor whose
  Diary the World holds, an original dead actor's included.
- Actor identity: an actor keeps the Identity that joined it to its loaded
  record, read to join as a cell record's key is; else the key a World cell
  record holds for it; else a new key. A key the World holds for a structure,
  the terrain or a dead actor stays theirs, so a cell that still names a bone
  gives the actor standing on it no second copy of that key.
- Two order carriers are filled at LOAD and kept in the Snapshot's document
  state, beside the restore tick. An attack order on a body below the
  targetable floor, which the World drops, is written as LOAD read it while
  the World stays at the restore tick. The attack phase word and the
  attack-complete flag of an actor LOAD found neither alive nor dying, a late
  corpse included, are written while it stays so; no order tick runs for it.
- Original world spell effects the World still runs (a spell graph or
  original areas) are written from their loaded records, and the World writes
  every field it holds over them: spell, payload, timers, references, target,
  and an area's identity, mode, cell key, common state, countdown and stage.
  The rest of their Token comes from the loaded file (DIV-2502).
- Cell records, motion residue, and the bound slot keys follow the World's
  saved cell records and actor motions.

### Town SAVE

The town document is the party's constructed one. A bound actor or Player
takes from the loaded town only the unknown-meaning spans; a party member's own
Human tails win over the loaded ones. The town's option and view leaves are
captured with the Snapshot, as a mission captures its application state, and
are not read from the loaded town at SAVE. Retained Diary records, key
translation and the other known fields of the loaded town are no longer
copied (DIV-2500).

### Unknown-meaning kinds taken from the loaded file

`savDocumentFallbacks` (`pkg/game/savdocumentsources.go`) is the only list.

| Kind | Bytes | Format page |
|---|---|---|
| Building B52 raw | 0-13, 16-17 | `sav/objects.md` |
| Building Block12 raw | 6-7 | `sav/token.md` |
| Building B46, B48, T08, T1C | B46 0-1, B48 0, T08 2-3, T1C 0-3 | `sav/token.md` |
| Effect Block12 raw, T08, T1C | 6-7, 0-3, 0-3 | `sav/token.md` |
| GlobalDWord | 0-3 | `sav/document.md` |
| Head.Reserved 9-10 | 0-3 each | `sav/document.md` |
| Trailer 2-99 | 0-3 each | `sav/document.md` |
| Human and Unit U114, UA6 | 22-23 | `sav/human-state.md` |
| Human and Unit UBE | 4-5 | `sav/human-state.md` |
| Human and Unit UD4 | 40-41, 46-47 | `sav/human-state.md` |
| Human and Unit SpellsHeader | 0-3 | `sav/objects.md` |
| Human and Unit T08 | 2-3 | `sav/token.md` |
| Human T1C | 0-3 | `sav/token.md` |
| Player Raw10 | 0-7 | `sav/player.md` |
| Player F48, F4C, F54 | 0-3, 0-1, 0-1 | `sav/player.md` |
| World cell Residue03, Residue32 | 0, 0-1 | `terrain/cells.md` |
| World session DiplomacyHeader | 0-3 | `sav/world.md` |
| World session FlagA48, FlagA49, ValueA4C, ValueB3B0 | whole field | `sav/world.md` |
| World session Raw08 | 0-15, 24-31, 36-47 | `sav/world.md` |

Beyond this list, the Token fields of an original world spell effect come from
the loaded file (DIV-2502): on `game0125.sav` and
`original-game0011-after-beforekargallas.sav` AreaEffect `T0E`, `T18`, `T1C`
and `Block12`; on `game0018.sav` PointEffect, SpellTransport and
Effect_DirectDamage `T08`, `T0E`, `T18`, `T1C` and `Block12`, and the payload's
`T0C`. Measured by the loaded-document census restricted to those classes;
the full change refuses on these saves because a changed `Block12` breaks the
cell key order.

The Building and Effect token kinds the World holds are written from the World.
Session Raw08 is taken from the file only when the World holds no raw session
head; cell residue only for cells without a World cell record or motion cell.

## Proof

The three corpus tests below passed on the EN and the RU install.

- `TestSAVWriterCensusDeadUnits` (`sessioncorpusaudit`): `saveorcsdontgo.sav`
  writes 101 Unit records, 54 Weapon records and 77 dead roots, as loaded, and
  a cold LOAD of the SAVE restores the saved World hash.
- `TestSAVWriterCensusLoadedDocument` (`sessioncorpusaudit`, corpus-gated, no
  save bytes in the repository) writes each input three times: with the loaded
  document unchanged, with only its unknown-meaning spans changed, and with
  every leaf but the join keys and format markers changed. A leaf that differs
  between the last two is a known kind taken from the loaded file. Result:
  0 known kinds on `saveorcsdontgo.sav`, `beforekargallas.sav`,
  `game0007-original-m70-boltcoming.sav` and `game0008-original-m70-dying.sav`.
  Before, measured on the base: 253 whole and 77 mixed kinds on
  `saveorcsdontgo.sav` and 164 and 11 on `game0007-original-m70-boltcoming.sav`
  by the per-kind poison census; 46 on `beforekargallas.sav` by this test. The
  base refuses the full change on the three mission saves, so this test gives
  no base number for them.
- `TestSAVWriterCensusLossControls` (`sessioncorpusaudit`) on
  `saveorcsdontgo.sav` and `game0007-original-m70-boltcoming.sav`: a kill, an
  item drop and a hero move through the player's command queue, 400 ticks,
  then hero damage and +137 gold. SAVE and cold LOAD carry the hero's health
  and cell, the gold, the item count, the kill and the body's health and decay
  stage.
- `TestCurrentCityGraftTakesOnlyUnknownSpans`: every loaded town value is
  changed; the written town differs from the constructed one only inside
  unknown-meaning spans.

- `TestSAVRoundTrip1195OriginalCorpus` (`sessioncorpusaudit`): all 121
  readable original saves SAVE and round-trip, as at the base.

## Open debt

- DIV-2501: a departed actor's record comes from its placement or party
  entry, not its exact body; a departed actor with neither has no record and
  no Diary; its pack and effect contents beyond worn items are not written.
  An engine SAVE taken after creatures left the world now keeps their
  placements on LOAD as terminal actors instead of withdrawing them; the
  withdrawing load is tested on the owner's mission-10 save.
- DIV-2503: an actor's recipient mask `T18` follows the mask rule, not the
  loaded record.
- DIV-2504: an actor the World binds to no source record gets a new RuntimeID
  on every SAVE.
- DIV-2505: an original terminal dead actor's worn, carried and effect
  objects are not written; the World holds none of them.
- DIV-2502: an original world spell effect's Token, beyond the fields the
  World holds, comes from the loaded file.
- DIV-2500: town actor mover, order and state words, town actor Diary records
  and the town Player's F44, F3D, F50, Outcome and PRaw32 are constructor
  values. A town Player Diary with no departure from its default is written at
  the tables' length, not the loaded length.
- An item the World holds without its native record gets a minted key when
  written into a sack, and a sack the World holds as native gets a minted
  runtime ID. `TestSAVWriterCensusChangedWorlds` (`sessioncorpusaudit`)
  reports this on `oldsaves7/game0006.sav`: an item dropped into an original
  sack. The census reports 8 mismatches here and 6 at the base; the other 6
  are the base's.
- An actor with no loaded record and no World cell key gets a new key on each
  SAVE; an external reference to it does not survive the next SAVE.
- On `saveorcsdontgo.sav` milestone-2 acceptance fails 11 subtests at the base
  and here: the World of that engine-written save lacks the original carriers
  those subtests compare.
- Unknown: after play, the live World's actor native basis differs from the
  one a cold LOAD restores, so the World hash differs; the base behaves the
  same. Moving an item between party members on `saveorcsdontgo.sav` makes the
  cold LOAD refuse with "source load producer has no applicable derive rule",
  also at the base.
