# The second game's hero uncovers the fog the way the original does

## Intent

Brief: the second game's units uncover the fog by the original's walk and
stamp (seed, step cost, height rise, edge skip, block offset and owner gate
as the claims state), through one fog stamp for both games with the first
game as data. Where the claims leave the time of a stamp Unknown the engine
keeps its own timing. The gate tests the 16-bit sight word in both games
(TERR-TILE-079, TERR-FOG-275, HERO-MOD-016), where the base tested the
whole-cell scan range. Seat decision: one gate serves both games, with no
per-game gate. On the 148 shipped maps no body's gate result differs from the
base, so the first game's fog cells, World hash and SAV bytes stay equal
there. One first-game case changes (see "What changes in the first game").

## Authority

Research EXP-2081, published in knowledge k376; the pin moves from k374 to
k376. Claims read with the pinned claim tool:

- R2-ENGINE-643 (High / Medium): the stamp returns unless the owner gate
  passes and the sight word is nonzero; the refresh every 32nd client tick;
  a second route (Medium) through the per-tick slot.
- R2-ENGINE-644 (High / Medium): the seed (sight >> 1) + 64, s 7, K 128.
- R2-ENGINE-645 (High / Medium): rings 1 to 19, the step cost
  trunc(128·sqrt(i²+j²)/max(i,j)), the type-2 height rise over the centre,
  ring cells outside 7..W-8 skipped, an empty ring ends the walk.
- R2-ENGINE-646 (High / Medium): the 2x2 block per positive budget cell, at
  the walk centre plus trunc(size/2).
- R2-ENGINE-647 (Medium): the hero at (11,41) uncovers 155 cells for sight
  1648 and 154 for 1617; the time of the first stamp is Unknown.
- R2-ENGINE-648 (High / Unknown): the owner word's bit 0x8 and a unit's state
  byte below 3; the bit's writer is Unknown.
- R2-ENGINE-649 (High): EN and RU equal. R2-ASSET-162 (High): the first map's
  planes, equal in both roots.
- TERR-FOG-266 (High / Medium) and TERR-TILE-079: the first game's stamp
  takes the same walk, block and write centre, gated by the local record's
  bit 3 for the owner, the 16-bit sight word at `+0x102` compared with zero,
  and the decay stage below 3.
- TERR-FOG-275 (High / Medium / Unknown): `+0x102` is the server actor's
  `+0xa4` word. HERO-MOD-016 (High): the actor's modifier fold adds the sight
  modifier word `+0xe4` to `+0xa4`, a 16-bit add. A Darkness effect lowers the
  word by whole cells (a modifier times 256); a fraction that stays nonzero
  keeps the gate open.

DIV-4182 was UNKNOWN for want of these claims; it is narrowed below.

## As built

| Kind | Builder |
|---|---|
| Fog reveal (gate, walk and stamp) | `World.fogStamp` in `pkg/sim/sight.go`; `Sight` and `SightOf` call it for every body |
| Sight word | `Entity.SightWord`, unchanged |
| SAV fog plane writer | the one SAV producer, unchanged |
| Minimap painter | unchanged |

- The first game's stamp is the second game's walk with the same data: the
  seed, tables, edge and block offset are one set of values, and no branch on
  the running game exists in the stamp.
- `fogStamp` holds the gate the claims state for both games: `fogRevealed`
  (the viewer's reveal bit for the owner, which the engine knows only for the
  viewer's own slot), a decay stage below 3 and a nonzero sight word. Before,
  `Sight` and `SightOf` each held their own copy of the gate, and it tested
  the whole-cell scan range where the claims test the sight word.
- The gate change reaches both games. A body whose whole-cell sight is 0 and
  whose sight word is nonzero stamps its 2x2 block, in the first game as in
  the second. Three bodies can be so: a Human whose Darkness effect took his
  whole cells to 0 and whose Mind and Reaction leave a fraction
  (`(Mind+Reaction)*256/25 mod 256`, nonzero unless Mind+Reaction is a
  multiple of 25); a loaded body whose saved word has high byte 0 and a
  nonzero low byte; a second-game source actor whose derived word is 1..255.
  A non-human body without a saved word has word `ScanRange << 8`, so its
  gate is the base's.
- The timing is unchanged: the opening refresh at tick 0, the refresh every
  32nd world tick, the teleport arrival; the second game takes no per-move
  stamp (`fogStampOnMove` stays a first-game profile value).
- `internal/archtest/fogwalk.go`: only `fogStamp` in `pkg/sim/sight.go` names
  the fog reader `fogSight`, it marches once, and it reads no `SecondGame`
  or `rom2`.

## Proof

Census, headless, an instrument run through `go test -overlay` from the
lane's notes, never committed. The engine's stamp is compared with the
cells each owner save's own Fog plane holds, and with a walk written from
the claim text alone. Hero (11,41) unless named; EN and RU rows are equal.

| State | Root | Our count | Original count | Only ours | Only theirs |
|---|---|---:|---:|---:|---:|
| new game, first stamp (hero word 1443) | rom2-en, rom2-ru | 131 | — | — | — |
| unmodified first hero, word 1648 | rom2-en, rom2-ru | 155 | 155 (R2-ENGINE-647) | 0 | 0 |
| game0001.sav, word 1617 | rom2-en, rom2-ru | 154 | 154 | 0 | 0 |
| game0004.sav, word 1617 | rom2-en, rom2-ru | 154 | 154 | 0 | 0 |
| game0005.sav, word 1617 | rom2-en, rom2-ru | 154 | 154 | 0 | 0 |
| game9999.sav, word 1617 | rom2-en, rom2-ru | 154 | 0 | 154 | 0 |
| game0006.sav (21,41), word 1617 | rom2-en, rom2-ru | 172 | 301 | 0 | 129 |

- The claim-text walk equals ours cell for cell on every row.
- The headless new game's hero carries Mind 15, Agility 26 and sight byte 5
  (word 1443); 131 is the claims' walk at that word. R2-ENGINE-647's 155 is
  for the unmodified template, which the second row builds on the mission's
  own planes.
- game9999 holds no explored cell: a save before the first stamp
  (R2-ENGINE-647). game0006's plane holds the hero's path from (11,41); our
  stamp at (21,41) lies inside it.
- Base `f7247810` gives the same table.

First game, base `f7247810` and head, missions 30 (the first) and 60, at the
first stamp and after 96 world ticks:

| Mission | Point | Root | Explored | World hash | SAV bytes | Fog, World, SAV, viewer fog, minimap |
|---|---|---|---:|---|---:|---|
| 30 | first stamp | en | 131 | 21b44069c4a3c204 | 212331 | equal |
| 30 | 96 ticks | en | 131 | 67c6b2a3e304d6ef | 214849 | equal |
| 60 | first stamp | en | 146 | a776e4f2b9c749ce | 401806 | equal |
| 60 | 96 ticks | en | 146 | e143e6090747fe81 | 404118 | equal |
| 30 | first stamp | ru | 131 | 21b44069c4a3c204 | 212365 | equal |
| 30 | 96 ticks | ru | 131 | 67c6b2a3e304d6ef | 214875 | equal |
| 60 | first stamp | ru | 146 | a776e4f2b9c749ce | 400865 | equal |
| 60 | 96 ticks | ru | 146 | e143e6090747fe81 | 403205 | equal |

"Equal" compares SHA-256 of the explored and visible planes, the World bytes,
the SAV document, the viewer's fog plane and the composed minimap picture.

Focused test `TestTheFogStampIsTheSecondGameWalk` (`pkg/sim/fogwalk_test.go`),
synthetic planes, against the claim-text witness: flat ground; a ridge of
height 127 east of the hero darkens column 29; a hill under the hero lights
more cells; near the west edge the stamp ends at column 7; a body of size 2
writes the size-1 figure one cell to +x and +y; a body of another owner
stamps nothing for the viewer and its own figure for its owner, and the same
body under the viewer's owner adds cells.

## What changes in the first game

A viewer's Human whose whole-cell sight falls to 0 and whose sight word stays
nonzero now uncovers his 2x2 block, in the first game as in the second. The
base uncovered nothing for him. Input: the review's Darkness witness on the
first mission of en and ru through the game's own mission door (hero Mind 24,
Reaction 27, ScanRange 6, word 1546; the mission's Darkness row at power 150
leaves ScanRange 0 and word 10 from tick 32). Base: visible plane 0 cells,
minimap `722673b00807…`. Head: visible plane 4 cells, minimap
`bcf3a429c0fa…`. The World hash and the SAV bytes equal the base at every
sampled tick, because the block lay inside cells already explored.

The explored plane does change when a darkened walk leaves the explored
cells. Input: a synthetic open 45x45 plane, a Human with Mind 25 and
Reaction 36 at ScanRange 0 (word 112) walking 24 cells east over 192 world
ticks through `mapWorld`. Head: 4 visible cells at the start, 52 explored
cells at the end. Base's gate: 0 and 0. The SAV's fog record is the explored
plane (`FogExplored` into the SAV's explored-terrain record), so its bytes
change in that case. The same stamp serves the first game's per-move fold,
the period refresh and the stamp after LOAD.

Authority: the first game's gate is the 16-bit word at `+0x102` compared with
zero (TERR-TILE-079); that word is the server actor's `+0xa4` (TERR-FOG-275),
which the modifier fold lowers by whole cells and leaves with its fraction
(HERO-MOD-016). The original's word for the review's hero is 1546 - 1536 =
10, nonzero, so the original stamps. The head is closer to the claims than the
base. No owner decision is needed: one gate serves both games.

Focused tests, synthetic planes, against the claim-text witness:
`TestAWordNotAWholeCellOpensTheFog` (`pkg/sim/fogdarkword_test.go`, both
games): a darkened Human at ScanRange 0 with word 112 stamps exactly the 4
cells the claim walk gives; Mind+Reaction a multiple of 25 (word 0) stamps
nothing; a loaded body with a saved low byte 0x80 and ScanRange 0 stamps the
claim walk at word 128, and with a saved low byte 0 nothing.
`TestAStampAfterLoadOpensTheBlockOfAWordWithNoWholeCell`
(`pkg/game/fogdarkword_test.go`): the stamp after LOAD of that loaded body
lights 4 visible and 4 explored cells, and of one with word 0 none.

Loss control, gate: with the base's gate (`ScanRange == 0`) put back in
`fogStamp` through an overlay of the committed file, both tests fail
("darkened Human: cell 1012 is 0, want 1"; "the stamp after LOAD lights 0
visible cells, want 4"). The committed source was never changed for it.

Gate census over the shipped maps, `ArchiveMaps` of one root per game opened
through `StartMissionFrom` and run for 600 ticks: the number of bodies below
decay stage 3 whose scan-range gate and sight-word gate differ is 0 at the
start and at the end (the review's census over all four roots: 148 maps, 0).
The shipped maps hold no body at ScanRange 0 below stage 3.

## Loss control of the walk

With the former second-game branch (the whole-cell scan range,
one cell per lit accumulator cell) put back in `fogStamp` through an overlay,
`TestTheFogStampIsTheSecondGameWalk`,
`TestTheFogStampLeavesTheAIReaderAndServesTheSecondGame` and
`TestTheSecondGameHeroStampsHisSightWordBlock` fail; the census gives 120
cells for each 154-cell save, 34 only theirs; and the guard reports
"fogStamp reads the running game (SecondGame)" and "fogStamp marches 2
times". The committed source was never changed for it.

## Open debt

- DIV-4182 (UNKNOWN, narrowed): the original's first stamp time and its
  second route; which owners and statics pass the owner gate; the six start
  minimap cells on rows 34 and 35, which lie outside both claimed figures.
- The headless new game's hero is not the template hero (word 1443 against
  1648). The hero producer is not this story's kind.
- A dead body at health exactly -20 is at the engine's decay stage 3 and at
  the second game's state byte 2 (R2-SESSION-195: 3 below -20), so it stamps
  in the original and not here. The decay ladder is not this story's kind.
- Unknown, the sight-word producer's kind: when a modifier exceeds the whole
  part, the engine clamps ScanRange at 0 and keeps the fraction (word 10 in
  the review's witness), where HERO-MOD-016's 16-bit add wraps the original's
  word (1546 - 1792 gives 65290). How the original's stamp reads a wrapped
  word is unread. Not fixed here; the gate and the producer are separate
  kinds.
- Unknown: a second-game source hero whose derived modifier leaves the word in
  1..255 differs from the base's gate by the code (`humanderive.go`); not run.
- Not reproduced on an install: a first-game cold LOAD of a body with saved
  high byte 0, and a second-game Darkness that lowers the hero's sight (the
  mission's spell 17 left ScanRange 5 in the review's run).
- `CheckFogWalk` (`internal/archtest/fogwalk.go`) misses four shapes, none in
  the tree: a game read in a callee of `fogStamp`; a second fog walk built
  from a `sightReader` literal with `fogInset`; a second march with `aiSight`
  inside `fogStamp`; a second-game-only field such as `w.rom2ActivityMask`. It
  also does not pin the gate term; the two tests above do.
