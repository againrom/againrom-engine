# Loading the second game's original saves

## Intent

The owner requires the second game's Load page to list each `game*.sav` the
original game wrote (magic `Bsg&`) and to load a town-like file into the
town with the hero and the money. The hero brings attributes, skills,
experience, HP, mana, worn items, the bag and the book. A mission-like file
may stay debt. After such a LOAD, SAVE writes the engine's own document. The
first game is unchanged.

## Authority

The pinned claim tool (knowledge k369) read these claims:

- R2-SESSION-017: the `Bsg&` envelope and its label.
- R2-SESSION-020: the Player's raw blocks.
- R2-SESSION-165 and R2-ENGINE-403: the Effect record.
- R2-ENGINE-208: the Humanoid without a Diary.
- R2-ENGINE-209: the Token head.
- R2-ENGINE-320: the main skill and the five experiences.
- R2-ENGINE-601: the LOAD route and CurrentState/InBattle.
- R2-ENGINE-602: the book serializer.
- R2-ENGINE-603: Building78, Pointer82 and the terrain keys.
- R2-ENGINE-604: base positions and key maps.
- R2-SESSION-233 to R2-SESSION-238: the nine EN files' partitions, the
  town-like and mission-like documents, the Player-1 Human, the books and
  the tail.

The claims name no room for a loaded town and no first frame. Those choices
are the owner's direction and stand in DIV-4153.

## As built

The SAV codec has one walk with two profiles of data, `sav.FirstGame` and
`sav.SecondGame`. The container picks the profile from the magic. The second
profile changes these record programmes:

- Token and Effect +0x0c are u16.
- Unit's sixth block is 184 bytes.
- The Humanoid has no Diary.
- The Player has 2,560 and 36 raw bytes around its groups.
- Pointer is a Building with one more DWORD.

It also has a 62-byte cell, a 10,374-byte session, an extra Token list after
the Sacks, book spells up to 29, and the main skill at the head of the
Humanoid's 24-byte block. `SecondTail` reads the physical tail (pairs, the
Scenario bank, the grid's four shop records of four category cells, the
available places, the current ordinal and nine buffers) and refuses a tail
that does not tile. A profile's `Game` is the game its envelope names:
`rom2` for `Bsg&`, none for `Asg&`, which the first game's originals and
both games' engine documents share.

The one LOAD route is restoreOriginal, validateOriginalGame,
decodeOriginalSource, then restoreOriginalTown. No step outside the codec
asks the file which game it is. The game the envelope names picks the
campaign service (`fileCampaign`, through `campaignOf`), and that service's
`fileState` answers what the file states in its own records:

- validateOriginalGame and the Load list's scan read the envelope's game
  through `ownsEnvelope`: the second game's Load page lists a `Bsg&` file
  under its label (DIV-3079); the first game's rows, its Load count and its
  save dialog leave it out, and its LOAD refuses it (DIV-2404).
- The first campaign's `fileState` reads the shortcut cells and states no
  session.
- The second campaign's `fileState` refuses two kinds of file before any
  state changes: a file saved during a mission (DIV-4155) and a book that
  holds spell 29 (DIV-4156). It reads the session from the tail
  (secondFileSession): the bank, the four shop records, the available places
  and the current place. The hero's bag becomes the campaign bag, one unit
  per stacked piece and at most 8,192 units. An empty `SpellBook/Shortcuts`
  array binds no quick spell (DIV-4154).
- decodeOriginalCampaign takes a stated session in place of the engine
  document, which such a file does not carry.
- restoreOriginalTown restores the party through the existing party restore
  and, for a stated session, leaves the bag to the campaign. The town opens
  at its square, built from the party (DIV-4153).
- The shop records go through the existing `currentSecondShop` record: a
  grid equal to the new game's writes none, as the capture does. Its
  validation bounds each cell's draws to 0..1000 and its quantity bound to
  0..100 for the engine's documents and the files alike.
- The main skill is read but not carried (DIV-4154).

`internal/archtest` ratchets the shape: outside `pkg/formats/sav` a call
of `sav.File.SecondGame`, a comparison of or switch on `sav.Profile`
values, and a comparison of a profile's `Game` with anything but the empty
string are game choices, with no debt site.

The existing SAVE producer is unchanged: after a LOAD it writes the engine's
document with the first-game magic and the second campaign's leaf.
`cmd/savecheck -party` prints the hero, the book and the second campaign.

## Proof

### Fixtures

Synthetic `Bsg&` town and mission files are built from the claims'
structure in `pkg/formats/sav/secondgame_test.go` and
`pkg/game/secondoriginalfile_test.go`. A town file is read, adopted and
compared field by field. Its round trip runs LOAD, SAVE, a cold LOAD, then
the departure for mission 10 and 20 frames
(`TestReleaseSecondOriginalTownLoadRoundTrip`). One changed byte of Mind is
the loaded Mind (`TestReleaseSecondOriginalTownLoadFollowsTheFile`). A
mission file is listed and refused, with no state changed
(`TestReleaseSecondOriginalMissionLoadIsRefused`).

The ordinary tests for the review's three findings, with the loss control
each was run against:

- `TestLiveGameChoiceStaysInTheProfile` and the four file-profile forms in
  `TestProfileFindingsRejectEveryForm`. Loss control: `if sf.SecondGame()`
  put back in `restoreOriginalTown` fails the live scan.
- `TestBsgTownShopRecordsAreTheGrids`: a town-2 file after mission 30
  (mission 40 open, bank slot 768 at 30) installs the grid's records, which
  differ from the new game's and from the bank replay in one cell; the SAVE
  capture writes them and a cold LOAD keeps them.
  `TestBsgTownShopGridIsBounded` refuses four out-of-bound cells. Loss
  control: dropping the grid from secondFileSession fails the first.
- `TestFirstGameLoadListLeavesOutABsgFile`: over one directory holding a
  `Bsg&` town and a first-game original, the first game's rows hold the
  original only, from both stores, its Load count over the `Bsg&` file alone
  is zero, and the second game's rows hold the `Bsg&` file alone. Loss
  control: dropping the envelope test from the scan fails it.

Over the nine owner copies the tail's grid equals the new game's records in
every file; the four town files install them, and the five mission files
refuse before (an uncommitted overlay over `decodeOriginalSource`).

### Witness over the nine owner files

A test kept outside the repository ran over the notes copy of the nine
files on rom2-en and rom2-ru. Each file went through the Load page, 20 town
frames, then GATES and mission 10. `cmd/savecheck -party -resave` also
loaded each file and cold-loaded our SAVE of it. Both roots gave equal
samples. No run panicked.

Column notes:

- Attributes are Body/Reaction/Mind/Spirit.
- Exp is the five experiences' sum.
- The reader's values for refused files come from the walk, not from a LOAD.

| File | Label | LOAD | Hero, attributes | HP | Mana | Exp | Worn | Book | Gold | After LOAD |
|---|---|---|---|---|---|---|---|---|---|---|
| game0000 | citybeforegame | town, square | Aldor 41/34/24/17 | 149/149 | 0/0 | 7320 | 6 | none | 1000 | 20 town frames; mission 10 not offered before the inn talk; cold LOAD of our SAVE equal |
| game0001 | firstsave | refused, mission 10 | reader: Aldor 41/34/24/17 | 149/149 | 0/0 | 7320 | 6 | none | - | state unchanged |
| game0002 | G-city | town, square | Gildarius 19/15/43/25 | 29/29 | 75/75 | 7320 | 3 | 24, 27 | 1000 | as game0000 |
| game0003 | G-city-buy | town, square | Gildarius 19/15/43/25 | 29/29 | 75/75 | 7320 | 1 | 24, 27 | 44 | as game0000; bag 0x812d |
| game0004 | G-start | refused, mission 10 | reader: Gildarius 19/15/43/25 | 29/29 | 66/75 | 7328 | 1 | 24, 27 | - | state unchanged |
| game0005 | G-all-spells | refused, mission 10 | reader: Gildarius 19/15/43/25 | 29/29 | 75/75 | 7336 | 1 | 1..29 | - | state unchanged |
| game0006 | G-particle | refused, mission 10 | reader: Gildarius 19/15/43/25 | 29/29 | 76/76 | 7498 | 1 | 1..29 | - | state unchanged |
| game9998 | Abort mission and return to town | town, square | Gildarius 19/15/43/25 | 29/29 | 75/75 | 7320 | 1 | 24, 27 | 44 | 20 town frames; GATES, mission 10, 20 map frames; bag 0x812d |
| game9999 | Restart last mission | refused, mission 10 | reader: Gildarius 19/15/43/25 | 29/29 | 75/75 | 7320 | 1 | 24, 27 | - | state unchanged |

The attributes, HP 149/149, mana 0/0 and experience 7320 of game0000 and
game0001 equal R2-SESSION-236. The other files' attributes also equal
R2-SESSION-236. The claim gives their HP, mana and experience only in its
table, so those columns are compared with nothing. The books equal
R2-SESSION-237. The tile positions the reader returns equal R2-SESSION-236:
0/0 in the town-like files, 11/41 in game0001, game0004, game0005 and
game9999, and 21/41 in game0006.

### Release witness for the release

By owner ruling, release tests run at the release, not in this lane. The
release runs these on rom2-en and on rom2-ru:

- `TestReleaseSecondOriginalTownLoadRoundTrip`,
  `TestReleaseSecondOriginalTownLoadFollowsTheFile` and
  `TestReleaseSecondOriginalMissionLoadIsRefused` in `pkg/game`.
- The owner-file witness on each root, over a fresh copy of the nine files
  in `review/rom2-owner-saves/orig` (game0000 to game0006, game9998,
  game9999; SHA-256 in `orig-sha256.txt`):
  `savecheck -assets <root> -saves <copy> -name local-sav:<file> -party -resave load`,
  then the same without `-resave` on the file our SAVE wrote. The expected
  result is the table above: four towns load and cold-load equal, and five
  files refuse with the mission message.
- The ROM2 save groups of `pipeline/check-rom2-save-tests.sh`, and
  `scripts/check-milestone2-acceptance.sh` on the EN and RU roots for the
  first game.

## Open debt

- DIV-4153 (UNKNOWN): the room a loaded town opens in and its first frame;
  the original's handling of loaded shop record values outside the writers'
  range.
- DIV-4154 (UNKNOWN): the main skill is not carried, and the empty
  Shortcuts array binds no quick spell.
- DIV-4155 (FIDELITY-DEBT): LOAD of a mission-like file refuses.
- DIV-4156 (FIDELITY-DEBT): LOAD of a book that holds spell 29 refuses.

### Research questions

1. After a cold LOAD of each town-like file in the original, which room is on
   screen at the first frame, and which controls answer?
2. Which original routines read the main skill (Humanoid+0x23c) after LOAD,
   and what do they show or change?
3. What index domain do the original's `SpellBook/Shortcuts` values take, and
   what does an empty array bind at the first frame?
4. What InBattle value does each of the nine files hold, and which branch of
   the LOAD route does each take?
5. After a cold LOAD of game0001, game0004, game0006 and game9999, what are
   the hero's position, HP, mana and book, the units' positions and the
   effects' targets at the first frame and the next ticks?
6. What does the original's book do with spell 29 after LOAD: is it listed,
   castable, and at what cost?
7. Which consumers read the Unit's 184-byte block and the Player's 2,560-byte
   and 36-byte blocks, and which values change between two saves of one
   mission?
