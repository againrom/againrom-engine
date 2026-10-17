# The fog stamp's inputs after k324

## Intent

Check the three inputs of the first game's fog stamp that DIV-3445 left on
Medium evidence (the height plane, the footprint side and the sight word)
against knowledge k324, and change the stamp only where a High claim
disagrees.

## Authority

Knowledge pin k324 (`117bab92`).

- `TERR-FOG-271` (High / Medium): the light parser reads each ALM record head
  at its own length word; the type-2 grid follows the head (20 bytes on
  `150.alm`, grid at file bytes [42184, 62920) of EN `150.alm`). Medium that
  the parser's object is the stamp's height plane.
- `TERR-FOG-272` (High / Medium): on that grid the stamp at the three heroes'
  saved cells and `+0xa4` words sets exactly the 34 cells the owner's run
  gained.
- `TERR-FOG-273` (High / Medium / Unknown): the write centre's `+0xd0` is the
  `units.reg` TileSize of the drawable's class; shipped TileSize is 1 except
  IDs 26, 27, 66, 68, 72 (2) and 71 (3). Medium for the party's class codes,
  Unknown for summons.
- `TERR-FOG-274`, `TERR-FOG-275` (High / Medium / Unknown): drawable `+0x102`
  is the server actor's `+0xa4` as of the last message with bit `0x2000`, sent
  to the owner's client; a body of class code 0x40 or above first takes its
  prototype's word. `TERR-FOG-081`'s route label is superseded; its source
  field stands.
- `TERR-FOG-276` (High / Medium / Unknown): a structure drawable's sight is
  Buildings `scanRange` x 256 and its stamp slot is bound; its presentation
  does not stamp; a clear restamp of it is Medium.
- Amended `TERR-FOG-266` to `TERR-FOG-269` and `SAV-1308`.

## As-built behaviour

No behaviour changes. Each input already reads what the High claims state on
shipped data:

- Height plane: the World height plane is the ALM type-2 payload, framed after
  a record head whose length word the decoder requires to be 20. On EN and RU
  mission 150 it equals the grid framed at the head's own length word, byte for
  byte, at [42184, 62920).
- Footprint side: the stamp adds half the body's token size, read from the
  Units row (Humans rows are 1). On EN and RU every one of the 55 Units rows
  with a class has the TileSize of its TypeID's class, and every Humans row and
  every class a Humans TypeID names is 1.
- Sight word: `Entity.SightWord` is the actor's `+0xa4` word the SAV writer
  writes; the party's words on the owner's mission-150 input are the original's.
- Structures: no structure stamps; the Buildings `scanRange` is not decoded.

`pkg/sim/sight.go` changes two comments only.

## Proof

- `TestReleaseFogHeightPlaneIsTheParserGrid` (EN, RU): fresh mission 150; the
  World height plane equals the type-2 grid framed at each head's length word;
  control: the slice 8 bytes earlier differs.
- `TestReleaseFogFootprintIsTheClassTileSize` (EN, RU): the Units and Humans
  tables against `units.reg`, and every body of fresh mission 81, which places
  TileSize 2 and 3 classes, carries its class's TileSize.
- `TestReleaseFogStampAfterColdLoad` (EN): 372 of 372 kept, 34 of 34 gained,
  0 extra, on the k324 pin.
- `pkg/sim/fogstamp_test.go`: a Human seeds with 1699; a loaded low byte wins.
- Census, fresh start of each mission in story1448's census list (the
  campaign chapter missions, 24 per first-game root; mission 81 is not in it;
  every second-game mission) with the default party, 1500 ticks,
  `35740d51` against this story (one root at a time): EN 24 of 24, RU 24 of
  24, ROM2-EN 46 of 46 and ROM2-RU 46 of 46 lines byte-equal (World hash at
  tick 0 and 1500, visible and explored planes, SAV digest).

## Rows

- DIV-3445: narrowed. Height plane, footprint and sight word now cite k324; the
  row keeps the token-size source (equal on shipped data, differs only in a
  mod), the Medium and Unknown links of `TERR-FOG-271`, `TERR-FOG-273` and
  `TERR-FOG-275`, the moment of the stamp after LOAD and one owner run.
- DIV-3477 (new, UNKNOWN): no structure stamps the fog; `TERR-FOG-276` leaves
  their restamp at Medium.
- DIV-3291, DIV-805, DIV-1184: unchanged. Their open points (sub-cell position,
  the cold-LOAD clear phase and bit 14 after LOAD) are not addressed by k324.

## Open debt

DIV-3445 and DIV-3477 as above.
