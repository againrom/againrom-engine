# Analysis — what a placed structure does to the plane, and what this tree already holds

| | |
|---|---|
| Intensity | **spec-anchored / static** — the derived plane is canonical hashed simulation state, which the profile puts at this tier. No watcher exists, so it is discipline |
| Terrain | **brownfield** for the plane's derivation in `pkg/mapload` (shipped behaviour changes for any map carrying a structure) · **greenfield** for the footprint resolution beside it |

## The baseline, read rather than remembered

`Passability(m *alm.Map) []byte` is the whole of the derivation today: five arms — the tile word's
impassable flag, the water range over the tile index, the mountain strip group at or above the blend
minimum, a non-zero placed-scenery byte, and the eight-cell ring — unioned into bit 0, with bit 1 set
by the ring alone. `Census` walks the same classifier, so the instrument cannot measure a second
reading. `newGrid` refuses any byte setting bits 2-7, so the plane is exactly two bits wide by
construction and not by convention.

Its two readers are `FromALMWith`, which hands it to `sim.NewWorld`, and the viewer's `Grid.Block`.
The second reads **bit 1 only** (`BorderCell`), so nothing this story does to bit 0 can move a drawn
pixel — the dimmed margin is unaffected, and that is checkable rather than argued.

`FromALM`'s own comment already discloses the gap in the terms this story closes: a structure does
not go through the ingest, it attaches after it per footprint cell, its inputs live in a table
nothing in this tree reads, and the price is the deck and doorway cells the original opens staying
closed. Two of those three have since changed: `pkg/formats/databin` parses the definition table and
hands each entry's parameters back verbatim, and `pkg/data` loads the class registries. Only the
join was missing.

`cmd/classdump -databin` is the one place that already holds both inputs at once — it opens the
archive, parses the table, and builds a `mapload.Table` from two of its collections. Nothing in
`pkg/game` opens the table at all, which is why the shipped front-end is out of this story's reach
and is named as such in the contract.

## What we did not know, and what was looked at

Every question below was taken to the research submodule at the story's pin and answered there, or
recorded as unanswered. `claims/retracted.md` and the standing-corrections table were read **first**.

- **Which routine turns a footprint into plane bytes, and whether it is the ingest.** It is not; the
  attach runs from the placed object's own constructor, strictly after the ingest, and the ingest's
  writes are assignments — so an attach that ran first would be erased. Answered, and it is what
  makes the pass an ordered second stage rather than a sixth arm.
- **Where the footprint comes from.** Four parameters of the class's definition entry: two extents
  and two 32-bit cell masks. Answered at instruction level, with the file's own column titles beside
  them.
- **Which mask decides what, and in which direction.** The attach mask is tested in the walk, the
  blocking mask in the per-cell recompute. The *direction* of the second is the one thing this story
  cannot take: it is published Medium, one `TEST`/`JZ` pair separates the two readings, and the
  corpus barely separates them at all. It is this story's R-1 and the reason the readiness verdict is
  AMBER.
- **Whether a structure can subtract.** Yes — that is the mechanism, not an edge case, and it is what
  a bridge is.
- **Whether the plane must grow a third bit.** No. The original's object bit is set and cleared only
  ever paired with bit 0, and the mover that reads it apart from bit 0 needs per-instance state no
  story has imported. Two bits stay two bits.
- **Whether structure art is a second consumer of the object-art machinery.** **It is not, and this
  is the finding that split the story.** The object and unit anchors are built from a class canvas
  and an anchor pixel — four registry keys the object and unit registries carry and the structure
  registry does not. The structure registry carries a tile-sized geometry instead, and no claim at
  the pin reads the routine that places a structure's sprite: the pin's own sprite-placement section
  covers a unit or an object, and its sprite format spec records the structure sheets' frame roles as
  still open. So the art half is not AMBER on a question in flight — it is unasked. It is not in this
  story, and the request for it is the orchestrator's to place.

## What the corpus is for here

The definition table and the maps are the owner's install. Every count this story records — cells
attached, cells blocked, cells opened, placements resolving to no entry — is a **measurement against
that install**, taken once and written into `verification.md`. None of it is a test: the tests are
synthetic tables and synthetic maps built in test code, and they must stay green with no install
present.

The measurement discriminates less than it looks. A census computed from this contract cannot test
the contract; what it can do is disagree with the figures research published for the same pass, and
that disagreement is the signal worth having.
