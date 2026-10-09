# Every SAVE is built from the World

## Intent and authority

Owner direction: a loaded state and an engine-made state are one kind of state.
Every SAVE, mission and town, builds its document the way a new game does, and
the copy of the loaded document as a base is removed. Only fields whose meaning
research has not established may take bytes from the loaded file; each such
field kind is listed below with its format page.

Base: public main `a975806f` (game 0.92.0, knowledge k200), reconciled with
main `71847832` (game 0.97.0, knowledge k204).

## As built

### Mission SAVE

`materializeCurrentWorld` always builds the mission document from the World,
the map and the tables. The loaded document is kept beside the Snapshot only
for the unknown-meaning spans; `graftUnknownObjects` joins each written object
to its loaded record by World identity (actor entity, Player ID, else a unique
class and identity key) and copies only those spans. A span the World holds is
written from the World: a structure's saved block and token, a registry
Effect's token, and every actor byte or value a native basis holds, a bound
entity's, retired or not, and a departed actor's removed basis on its terminal
or dead record.

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
- The World holds an actor's order words it does not run
  (`sim.HeldOrder`, byte form 117). LOAD fills it. A frozen order is the
  nonzero attack phase word and attack-complete flag of an actor LOAD found
  neither alive nor dying, a late corpse included; it ends when the actor
  lives again. A body order is a live actor's attack order on a body below
  the targetable floor, which the World drops; it ends when the World
  advances. The SAVE writes both from the World.
- Original world spell effects the World still runs (a spell graph or
  original areas) are written from their loaded records, and the World writes
  every field it holds over them: spell, payload, timers, references, target,
  and an area's identity, mode, cell key, common state, countdown and stage.
  The rest of their Token comes from the loaded file. Those are known fields,
  so this is a loaded-file dependency the owner's rule does not admit, kept as
  debt (DIV-2502): the World holds no Token and no record graph for these
  records, and carrying them needs that model in the World.
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

Beyond this list, the known Token fields of an original world spell effect
come from the loaded file (DIV-2502). On `game0018.sav` the loaded-document
census names 19 kinds: `Block12`, `RuntimeID`, `T08`, `T0E`, `T18` and `T1C`
of PointEffect, SpellTransport and Effect_DirectDamage, and the payload's
`T0C`. A per-class census measured AreaEffect `T0E`, `T18`, `T1C` and
`Block12` on `game0125.sav` and `original-game0011-after-beforekargallas.sav`.

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
  between the last two is a known kind taken from the loaded file. In a
  value holding an unknown span, the full change also flips its first known
  byte below the span (the low word of `T08`). When the full change is
  refused, each pattern is changed alone and every pattern that alters the
  SAVE or is refused is named. Result on EN and RU: 0 known kinds on
  `saveorcsdontgo.sav`, `beforekargallas.sav`,
  `game0007-original-m70-boltcoming.sav` and `game0008-original-m70-dying.sav`;
  on `game0018.sav`, which holds original world spell effects, the 19 Token
  kinds of DIV-2502 and no other. Actor order words are poisoned with the rest
  of each record; the World's held orders carry them.
  Before, measured on the base: 253 whole and 77 mixed kinds on
  `saveorcsdontgo.sav` and 164 and 11 on `game0007-original-m70-boltcoming.sav`
  by the per-kind poison census; 46 on `beforekargallas.sav` by this test. The
  base refuses the full change on the three mission saves, so this test gives
  no base number for them.
- `TestReleaseConsumedCorpseSAVLoad` (EN and RU): the World holds 23 and 29
  in base byte 22 and modifier byte 40 of a creature before the first SAVE and
  LOAD, then 96 and 100 before Control Spirit consumes its corpse. SAVE and
  cold LOAD restore 96 and 100 and the removed basis; before the graft took
  the removed basis as World-held, cold LOAD gave 23 and 29.
- `TestHeldOrdersSurviveBinaryAndLapseAsTheWorldAdvances` and
  `TestHeldOrdersRefuseAnImpossibleOrder` (`pkg/sim`).
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
  World holds, comes from the loaded file. These are known fields; this does
  not meet the owner's unknown-only rule.
- DIV-2500: town actor mover, order and state words, town actor Diary records
  and the town Player's F44, F3D, F50, Outcome and PRaw32 are constructor
  values. A town Player Diary with no departure from its default is written at
  the tables' length, not the loaded length.
- DIV-2506: an original Sack whose cell has no key in the file is held by
  the World without its Token, because the object registry binds a Sack only
  through its cell key. A SAVE gives it and its items new runtime IDs and
  identities. `TestSAVWriterCensusChangedWorlds` (`sessioncorpusaudit`)
  reports this on `oldsaves7/game0006.sav` (the Sack at 71,116, runtime ID 168
  in the file): 8 mismatches here, 6 at the base; the other 6 are the base's.
  Its cell, gold and items are kept.
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
