# The fog stamp and the stamp after LOAD

## Intent

The explored area the engine stamps around a first-game party matches the
original's, and a LOAD stamps the party at once, so the explored layer gains
what the stamp lights.

## Authority

Knowledge pin k322 (`4b390231`).

- `TERR-FOG-266` (High / Medium): the client stamp ORs `0xc0` into the 2x2 tile
  block (x,y)..(x+1,y+1) for each positive accumulator cell, around a write
  centre at drawable `+0x34`/`+0x38` plus half of the class `+0xd0`; the walk
  centres on `+0x34`/`+0x38`.
- `TERR-FOG-265`, `SAV-1308` (High): in the owner's mission-150 run the original's
  SAVE taken at once after a cold LOAD kept 372 of 372 explored cells and added
  34 at Chebyshev 5..7 from the three heroes; a SAVE 5 s later was equal.
- `TERR-FOG-267` (Medium / Unknown): the transcribed stamp at the heroes' saved
  cells, with their `+0xa4` sight words as drawable `+0x102`, explains 28 of the
  34 and sets 3 the original lacks.
- `TERR-FOG-268` (Medium): a freshly created drawable stamps on its first
  admitted presentation call; the cold-LOAD clear phase is Unknown.
- `HERO-SIGHT-007` (High): a Human's sight word is `((mind + reaction)/25 + 4)`
  in 1/256 cell, whose high byte is the whole-cell range.
- `SAV-1307` (High on the root): the SAVE root writes the live plane masked
  `0x8000` unchanged. `SAV-FOG-061`, `TERR-FOG-145`: a SAV carries bit 15 only.

## As-built behaviour

- `sim.World.Sight` and `SightOf` are the fog reader. In the first game each
  body's stamp walks from its cell with the fog inset, seeded with the body's
  sight word, and marks the 2x2 block per positive accumulator cell around the
  cell plus half the footprint side (`fogStamp`). The second game keeps one cell
  per positive accumulator cell. The AI reader (`aiInset`, `actorSight`,
  `groupSight`) marches unchanged: `marchSight` now delegates to `marchInto`,
  which marks no cell when the stamp is nil.
- `Entity.SightWord` is the one producer of the sight word: a source-derived
  actor's own word, else the scan range with a loaded low byte, else the Human
  rule, else the whole range. The SAV writer (`currentActorSource`) and the fog
  stamp both read it. The Human rule moved to `pkg/rules` (`SightWord`,
  `SightSubCells`); `pkg/data` delegates.
- Every first-game stamp uses it: the opening refresh, the period-32 refresh,
  the per-move stamp (DIV-3291), the teleport arrival and the stamp after LOAD.
- `settleMission` calls `stampAfterLoad` on every SAV LOAD after the Fog plane
  and the application state are restored: the party's stamp ORs into the
  visible and explored layers before the first visible frame. The second game
  is skipped.
- Across an ordinary LOAD the visible layer becomes the party's stamp at the
  loaded state and exploration becomes the saved plane ORed with it. A SAVE
  taken between two samples loses the cells the last sample still showed behind
  a moved body; the next period-32 sample makes the continuations equal.

## Proof

- Owner run, EN, `TestReleaseFogStampAfterColdLoad` (inputs read from
  `AGAINROM_FOG_STAMP_DIR/en/`, SHA-256 `8f349384…` and `5683fdca…`; RU has no
  input and skips): the SAVE before any tick keeps 372 of 372, gains
  34 of 34 and sets 0 cells the original lacks; the SAVE after 96 ticks is equal.
  The heroes stand at (14,103), (14,104), (13,106) with sight words 1699, 1546,
  1648, the original's words. Controls on the same input: base `2b5ff4f4` gains
  0; the 2x2 block with the whole-cell range gains 16; the sight word without the
  block gains 17. `TERR-FOG-267`'s transcription gives 28 / 6 / 3.
- Synthetic, `pkg/sim/fogstamp_test.go`: the flat-ground stamp equals the
  independently dilated disc (16, 32, 60, 88, 128, 172 cells for range 1..6);
  the write centre moves by half the footprint while the walk stays on the
  body's cell on uneven ground, with a control that a walk from the write
  centre differs; a Human seeds with 1699 and a loaded low byte wins; the AI
  reader and the second game's fog light the plain 69-cell disc at range 4.
- Census, fresh start of every campaign mission with the default party, 1500
  ticks, base against story (one root at a time): EN 24 of 24 and RU 24 of 24
  World hashes equal at tick 0 and tick 1500; explored cells at tick 1500 total
  2909 before and 4053 after on each root. ROM2-EN 46 of 46 and ROM2-RU 46 of
  46 missions keep World hash, visible and explored planes byte-equal.
- N4, `TestNativeFog1115CheckpointBetweenSamples`: a SAVE one tick after a
  move, cold LOAD; the visible layer equals the party's stamp, exploration is the
  saved plane ORed with it, the stale sample cells are gone, World hashes stay
  equal for 64 ticks and the visible layers agree from the first period sample.

## Open debt

DIV-3445: the footprint side stands in for the class `+0xd0`, the altitude
plane for the client height plane and the actor sight word for drawable
`+0x102`, each on Medium evidence; the engine stamps inside the LOAD, where the
original stamps at its first presentation call; the match is one mission in one
owner run. DIV-805 and DIV-1184 state the visible layer after LOAD.
