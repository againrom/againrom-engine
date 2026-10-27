# Loading the second game's original mission saves

## Intent

The owner requires a LOAD of each mission-like `game*.sav` the second game's
original wrote (game0001, game0004, game0005, game0006, game9999) to open
mission 10. It brings the eight Players with their units and groups, the
hero with his fields, items and book, the Buildings and the Pointer, the
effects, the terrain state, the Sacks and game0006's dead Unit. Terrain size,
tiles and static objects come from the mission's ALM. The first frame follows
the claims. A book holding spell 29 (Slow) loads and survives SAVE and cold
LOAD. After such a LOAD, SAVE writes the engine's own document, and a field
without a source is ledger debt, never a refused save. The four town files
load as before, and the first game is unchanged.

## Authority

Owner rulings: one builder per kind; the first game is a reduced second
game; reader before writer; SAV is the only save format, from one producer;
where the second game has no evidence it behaves as the first, with a DIV
row; first-game claims are never evidence for the second game.

The pinned claim tool (knowledge k371) read these claims:

- R2-SESSION-019: the session block's 10,374 bytes and the sparse terrain
  key list.
- R2-SESSION-020: the Player's raw blocks.
- R2-SESSION-233 to R2-SESSION-238: the nine EN files' partitions, the
  mission-like documents, the Player-1 Human, the books and the tail.
- R2-SESSION-244 and R2-SESSION-245: the five mission files' state stores
  and the Selection key.
- R2-ENGINE-209: the Token head's +0x0c word.
- R2-ENGINE-601 to R2-ENGINE-604: the LOAD route, the book serializer,
  Building78, Pointer82 and the terrain entries, base positions and key maps.
- R2-ENGINE-613 to R2-ENGINE-618: the mission entry after the LOAD, the REG
  restore, the client map from the ALM, the world half, the unit registry
  walk and the Building serializer.
- R2-ENGINE-619 and R2-ASSET-154: spell row 29 is Slow and has no row in the
  slot table and no hover line.
- R2-ENGINE-620 and R2-ENGINE-621: Speed to the tick divisor, and the first
  frame of the five files.

## As built

Each kind this change touches, with the one builder it extends:

| Kind | Builder | Change |
|---|---|---|
| SAV profile | `sav.Profile` (`pkg/formats/sav/profile.go`) | The second game's profile names its session layout, a cell entry without the trigger tail, and a state store read by grammar and root |
| Session reader | `File.SessionState` (`session.go`, `world.go`) | Reads fields through the profile's layout; counts result slots past 100; refuses a stance outside 50x50 |
| Cell reader | `File.Cells` with `decodeCell`, `File.CellTriggers`, `File.StructureCells` | The entry stride comes from the profile; an entry without the tail holds no trigger, and `SetCell` refuses it |
| Building reader | `File.Buildings` | The Token +0x0c word is kept as read; a Pointer is a Building |
| State store reader | `File.StateData` (`world_state.go`) | Gives a mid-mission file's store whichever game wrote it; `documentDataFromDocument` uses the same `parsedStateData` |
| Extra Token list | `File.ExtraTokens` (`ground.go`) | Counts the second game's list after the Sacks |
| Mission LOAD route | `secondCampaignRules.fileState` (`pkg/game/campaignsecond.go`) | Takes mission files; a file with Tokens after its Sacks refuses before any state changes |
| Structure import | `applyOriginalStructuresCurrent` (`pkg/game/originalstructures.go`) | A Pointer takes the Building policy; the structure keeps the word's low byte |
| First frame | `fileApplication` and `originalApplicationStateOf` (`savapplication.go`), read by `readFileApplication` in `originalrestore.go` | A file without a complete document gives the first frame from its own state store: view, selection, panels, pressed cell through the game's slot table, and the speed's period |
| Book import | `importedSpellbook` (`pkg/game/originalspellbook.go`), `World.ImportOriginalActorSpellbooks` (`pkg/sim`) | A book holding spells past 28 takes the native mode; the World admits it only when each such spell is a row of its table |
| World SAVE | `currentMissionDocument` (`pkg/game/currentsave.go`) with `leaveUnboundEffectsOut` | After any LOAD without a decoded document, in either game, the SAVE's copy of the World drops the carried spell effects and each projectile no driver row moves, once, before both document writers read it |
| Spell id bound | `World.admitsSpellID` and `sim.SpellIDLimit` (`pkg/sim/originalspellbooks.go`) | A spell past the book's slots stands as a book spell, an admitted selection and an attached effect only while the World's table has its row; the World's binary form, its action and effect restores and the document's effect readers take that one rule |
| Book selection in the document | `projectCurrentCastWire` (`savactionwire.go`) and `matchCurrentBookSelection` (`currentbookselection.go`) | A selection past the book's slots writes no Spell key and keeps its value at LOAD |
| LOAD report | `OriginalSaveResume` | Names the result slots past 100 it does not carry |
| Witness | `cmd/savecheck -witness N` | Prints units per Player, the hero, structures, effects, the bank and the first frame, then the hash after N ticks |

The second game's mission LOAD keeps the partial import path that the
town files use: the World is built from the mission's ALM and the
document's typed readers, and the file's own document graph is not kept.
That is why the effects in flight have no continuation (DIV-4175).

## Proof

### Fixtures

The tests build every `Bsg&` file from the claims' structure; no owner byte
enters the repository.

- `pkg/formats/sav/secondworld_test.go`: the session read by position, slots
  past 100 counted and a stance outside 50x50 refused; the cells read to
  their Building key with no trigger; an empty shortcut array in the state
  store; a Building and a Pointer keeping their Token word.
- `pkg/game/secondmissionload_test.go`:
  - `TestSecondMissionFileLoadsTheHeroAndTheFirstFrame`: at Speed 4 and 8,
    the hero at 11/21 with his pools, attributes, skills, held and carried
    item and book, selected; View 8/29; inventory closed; the book open as
    the store says; periods 62000 and 31000 us; the pressed cell.
  - `TestSecondMissionFileLoadFollowsTheFile` (sensitivity): Mind 44 instead
    of 43 and position 3/5 instead of 11/21 in the file are the loaded hero's.
  - `TestSecondMissionFileSaveColdLoadAndNextOrder` (round trip): a book
    holding 24 and 29 loads in the native mode; the game menu's SAVE writes
    our document; with the original file removed, its cold LOAD gives the
    same hero, book, view and World hash; the same move order then gives
    equal hashes for 32 ticks and the hero moves.
  - `TestSecondMissionFileSaveLeavesAnUndrivenProjectileOut`: a projectile
    without a driver, alone or beside a driven one, does not refuse the SAVE
    and is absent after the cold LOAD; the driven one is held and moves as
    it does in the live World.
  - `TestSecondMissionFileSlowCastSavesAndColdLoads`: a self-cast of Slow;
    at each of 24 ticks from the order through the effect's life the SAVE
    writes and its cold LOAD has the live World's hash, selection and next
    tick.
  - `TestSecondBookPageLeavesSlowOut`: with spell rules 1..29 the book page
    shows the slot table's 24 cells and no Slow.
- `pkg/game/secondoriginalfile_test.go`, `TestSecondMissionFileStateReads`:
  a town and a mission file with spells 24 and 29 read; a file with a Token
  after its Sacks refuses; the state store gives Speed 8, the open book,
  slot 6 pressed, View 8/29 and the selection.
- `pkg/sim/originalspellbooks_test.go`,
  `TestImportOriginalBookPastItsSlotsNeedsATableSpell`: the native mode with
  spell 29 imports when rule 29 is in the table, refuses without it and
  refuses spell 0; `TestAdmittedBookSelectionPastTheSlotsFollowsTheTable`:
  a selection of 29 is written and read back only while the table has row
  29, and 30 is refused.
- `pkg/game/secondoriginalload_release_test.go`,
  `TestReleaseSecondOriginalMissionLoadIsRefused` (not run in the lane): a
  mission file with a Token after its Sacks is listed and its LOAD refuses
  without changing the front end. The test keeps its name because the
  release population lists it.

### Witness over the nine owner files

`review/story1526-rom2-mission-load/witness.sh` builds `cmd/savecheck` from
the worktree and runs `savecheck -assets <root> -saves <copy> -name
local-sav:<file> -party -witness 64 load` over hash-checked copies of the
nine EN files. `roundtrip.sh` then LOADs each mission file, SAVEs through
the game route (`-resave`), cold-LOADs our SAVE with the witness, SAVEs again
and cold-LOADs that. rom2-en and rom2-ru gave equal output in every column.

Column notes:

- Units per Player are alive/dead counts for Players 1, 3, 5, 6, 7 and 8; the
  same in all five files except game0006's Player 3.
- Structures are 15 Buildings and one Pointer in each file.
- Hash is the World hash at the LOAD; After is the hash after 64 ticks.
- Cold is the cold LOAD of our SAVE and of our second SAVE.

| File | Label | Hero at | HP | Mana | Book | Player 3 | Cells | Effects, projectiles | View, selection | Book, inventory | Period | Hash | After | Cold |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| game0001 | firstsave | 11/41 | 149/149 | 0/0 | none | 20/0 | 125 | 0, 0 | 8/29, hero | closed, closed | 62000 us | 97e2e506ce725137 | fd3e866eadcdd6ee | equal |
| game0004 | G-start | 11/41 | 29/29 | 66/75 | 24, 27 | 20/0 | 125 | 0, 0 | 8/29, hero | closed, closed | 62000 us | 082247c51488a59b | bb7a4af63478900b | equal |
| game0005 | G-all-spells | 11/41 | 29/29 | 75/75 | 1..29 | 20/0 | 125 | 0, 0 | 8/29, hero | open, closed | 31000 us | 57cb17772cd9ba71 | 2cca172d18b27f2f | equal |
| game0006 | G-particle | 21/41 | 29/29 | 76/76 | 1..29, slot 6 (spell 21) pressed | 18/2 | 126 | 1, 1 | 8/29, hero | open, closed | 31000 us | c93b6981aa8744d0 | 4079dbc311a64c15 | 8be80b79b05cb22b, after abd2e09c3ff12e68, effects 0, 0 (DIV-4175); second cold equal to the first |
| game9999 | Restart last mission | 11/41 | 29/29 | 75/75 | 24, 27 | 20/0 | 125 | 0, 0 | 8/29, hero | closed, closed | 62000 us | 8032b980d06cbdba | eea78e5c151a892f | equal |

The other Players hold 1/0 (Player 1), 5/0, 1/0, 1/0 and 2/0 units in each
file. The hero's attributes are 41/34/24/17 in game0001 and 19/15/43/25 in
the other four, as R2-SESSION-236 states; the books equal R2-SESSION-237.
game0006's LOAD restores one late corpse and one dying actor (the dead Unit
of R2-SESSION-235). Each file sets bank 768 to 10 and 769 to 1. Speed 4 and
8 give 62 and 31 ms as R2-ENGINE-620 states.

The four town files (game0000, game0002, game0003, game9998) LOAD on both
roots with standard output and standard error byte-equal to `savecheck`
built from the base commit 91ad8a33.

### Loss control

The refusal of a mission file (`Head.Mission != 0` in `fileState`) was put
back on top of the candidate. All five `TestSecondMission*` tests in
`pkg/game` failed, three of them with "a second-game save made during
mission 10 cannot be loaded yet", and the witness refused all five mission
copies with that message while the four town copies still loaded. Each file
was restored from the candidate commit.

Three further controls ran on the corrected tree, each restored by hand
before the next; output is in the notes' `tmp/loss-ab.txt` and
`tmp/loss-c.txt`:

- With the `leaveUnboundEffectsOut` call in `currentMissionDocument` switched
  off, `TestSecondMissionFileSaveLeavesAnUndrivenProjectileOut` failed: our
  SAVE refused with "legacy native world effects have no current continuation
  bindings".
- With the `> 28` bound put back in the World's binary writer,
  `TestSecondMissionFileSlowCastSavesAndColdLoads` failed at tick 81: "our
  SAVE: invalid admitted book selection".
- With the keep rule of `matchCurrentBookSelection` switched off, the same
  test failed at tick 81: "the cold LOAD's selection is 0, want 29".

### First game unchanged

`savecheck -mission 10 -ticks 200 save`, then `load`, on the en and ru roots,
built from the base and from the candidate: the SAV files are byte-equal and
the LOAD output, World hash 104526eb035c638b included, is equal. The
`pkg/formats/sav`, `pkg/sim` and `pkg/game` packages pass.

### Release tests for the release

By owner ruling, release tests run at the release, not in this lane. The
release runs these on rom2-en and on rom2-ru:

- `TestReleaseSecondOriginalMissionLoadIsRefused`,
  `TestReleaseSecondOriginalTownLoadRoundTrip` and
  `TestReleaseSecondOriginalTownLoadFollowsTheFile` in `pkg/game`.
- `TestReleaseSpellDelivery1183InstalledRulesAndNativeSave` in `pkg/game`,
  on en and ru: it reaches the SAVE route and the spell id bound.
- The owner-file witness on each root over a fresh copy of the nine files in
  `review/rom2-owner-saves/orig`: `witness.sh <root>` and `roundtrip.sh
  <root>` from the lane's notes. The expected result is the table above.
- The lane's `probe/run.sh` and `probe/inflight.sh` over owner copies on en
  and ru: a SAVE every 4 ticks over 200 ticks on game0006 and game0005, a cast
  of each book spell 1..29 on game0006 followed by SAVEs, and the SAVE with a
  driven projectile beside the undriven one. The expected result is no
  refused SAVE and the cold LOAD holding the driven projectile alone.
- The ROM2 save groups of `pipeline/check-rom2-save-tests.sh`, and
  `scripts/check-milestone2-acceptance.sh` on en and ru for the first game.

No release test LOADs an owner mission file through the Load page: the
population of release tests is fenced in this lane. A release test that does
is a handoff.

## Open debt

- DIV-4170 (UNKNOWN): the session block read by position; result slots 901
  and 999 are not carried.
- DIV-4171 (UNKNOWN): payload +0x2c..+0x3b of each terrain entry is not
  carried.
- DIV-4172 (FIDELITY-DEBT): the Pointer's last DWORD, the Token +0x0c high
  byte and the Player and Unit raw blocks are not carried; a Token after the
  Sacks refuses the LOAD.
- DIV-4173 (UNKNOWN): the camera unit of View X/Y, the clock on the first
  frame and the receiver of 0x408.
- DIV-4174 (UNKNOWN): no book cell for Slow; an engaging caster can still
  cast it, and its selection and effect are saved and cold-loaded.
- DIV-4175 (FIDELITY-DEBT): game0006's spell effect and projectile are not
  shown, do not land and our SAVE leaves them out, for any LOAD without a
  decoded document in either game; a projectile cast after the LOAD is kept.
  The hero's saved order 4 is not continued.

DIV-4155 and DIV-4156 are closed. DIV-3079 names the mission LOAD rows.

### Research questions

1. LOAD each of the five mission-like EN saves in the original. At the first
   frame, record the map cell at the map view's top-left corner, the portrait
   marked selected and the open panels. Take two screenshots 5 s apart with
   no input and record what changed.
2. After a mission-like LOAD, which routine receives message 0x408 sent
   through app +0xd4 slot +0x48, and which fields does it write?
3. Which routine reads app +0x418 after mission setup, and in what unit does
   it use the value?
4. LOAD game0005, open the book and count its icons and hover captions.
   Select the hero, press each cast hotkey and each book cell, and record each
   spell offered or cast.
5. Which routines read and write session bytes 3604..3607 and 3996..3999
   (result slots 901 and 999)? Record both in two saves taken at different
   times of the same mission.
6. Which routines read the 60 raw bytes of a second-game terrain entry, and
   which field does each byte range hold?
7. Which routines read Pointer +0x6c and the high byte of a Building's Token
   +0x0c word, and what values do they hold in the five mission files?
8. LOAD game0006 in the original and record for 2 s with no input the
   projectile's map position on each frame and its target's HP.
