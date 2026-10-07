# spec — 0150 the explored map

**Contract: the explored map survives a save and a load.**

A player who loads a save resumes with the ground he had already uncovered still uncovered. Before
this story a loaded save showed only what the party revealed after the load.

This document is self-contained. Research provenance is in `provenance.md`.

## Terms

- **Original save**: a `game####.sav` file the original game wrote, at an install root. Magic
  `Asg&`.
- **Our save**: a file this tree wrote. Magic `AGRMSAVE`.
- **Tail**: the part of an original save after the compressed body and after the `0x100`-byte label
  region.
- **State store**: the key-value container at the start of the tail. Its first dword is `0x31415926`
  and its framing is the same framing the `.reg` registry format uses: a `0x18`-byte header, a node
  table of `recordCount` records of `0x20` bytes each, a `u32` pool length, then the pool.
- **Trailing region**: whatever follows the state store's own declared end, to the end of the file.
- **Explored plane**: one byte per map cell, non-zero where the cell has ever been seen, in linear
  order `idx = col + row*width`.

## Functional requirements

**FR-1 — The state store is opened, to its own declared extent.**
`pkg/formats/sav` parses the tail into two parts: the state store, whose length is
`0x18 + 0x20*recordCount + 4 + poolLen` read from the store's own header, and the trailing region,
which is everything after it. The store is exposed as a parsed key-value tree. The trailing region
is exposed as bytes and is not decoded.

A reader that parsed the tail to the end of the file would read the trailing region as pool. That is
the failure this requirement exists to prevent.

**FR-2 — A tail that does not frame as a store is still carried.**
Where the tail is too short, carries the wrong signature dword, or declares an extent past the end of
the tail, the whole tail is carried as the trailing region and no store is reported. `Open` does not
fail on it. A file that opened before this story still opens.

**FR-3 — Re-emitting a save is byte-identical.**
`Marshal` writes the store bytes and then the trailing region. Splitting the tail changes no byte of
the output for any file this tree can read.

**FR-4 — The explored plane is decoded from `Fog/FirstState` and `Fog/Data`.**
Where the store carries section `Fog` with an int leaf `FirstState` and an int-array leaf `Data`, the
save yields:

- the run lengths, verbatim;
- a per-cell plane of `sum(Data)` bytes, in linear order, holding 1 where the cell is explored and 0
  where it is not;
- the polarity taken from `FirstState`: the first run carries the state `FirstState` records, and
  the state flips at every run boundary. `FirstState` non-zero means the first run is explored.

A negative run length is refused, naming the element. A store with no `Fog` section reports that
there is none, which is not an error: a save taken between missions carries no `Fog` section.

**FR-5 — A loaded original save restores the explored plane into the running map screen.**
When the LOAD GAME window opens a row from the original format, the mission opens with every cell the
save records as explored already explored. The restore ORs into the plane and clears nothing, which
is what the original's own load arm does.

The plane is applied only where the save's cell count equals the opened map's `width * height`. A
disagreement is refused, not clamped: the save was taken on a different map and painting one map's
exploration onto another's cells would be a wrong picture with nothing to signal it.

**FR-6 — The render gate is not changed.**
Nothing in this story changes which cells are drawn for a given plane state, how the three fog states
are shaded, or how a drawable is gated. The restored bytes enter the same plane a march would have
lit, and the existing visibility path decides the rest.

**FR-7 — Our own save carries the explored plane and round-trips it.**
A save this tree writes, and the load that reads it back, carry the explored plane with the same
dimensions. A plane restored from an original save and then written into one of our saves comes back
from that save.

**FR-8 — The tools report what was restored.**
`savtool` gains a verb that prints, for each file: whether a store was found, its record and leaf
counts, its declared extent, the trailing region's length and first dword, and — where a `Fog`
section is present — the run count, the total cell count and the explored cell count with its
percentage.

`savecheck load` prints, for a restored original save, how many cells the explored plane holds.

**FR-9 — The prose that says the explored map is not carried is corrected.**
Every sentence this tree shows a player or a developer stating that exploration is not carried is
corrected to state that it is. Named sites: the LOAD GAME window's caveat line, and the counted
resume report a driver prints.

## Acceptance criteria

- **AC-1** For each of the 19 preserved original saves, the store's declared extent plus the
  trailing region's length equals the tail's length exactly.
- **AC-2** `sum(Data)` equals the map's cell count for every save that carries a `Fog` section.
- **AC-3** Re-emitting each of the 19 files produces the identical bytes.
- **AC-4** A save taken between missions is reported as carrying no `Fog` section rather than as an
  error.
- **AC-5** Loading an original save through the LOAD GAME seam yields a map screen whose explored
  plane holds the save's own explored cell count, and never fewer.
- **AC-6** The explored plane restored from an original save survives a save into our own format and
  a load back out of it.
- **AC-7** Nothing clears an explored cell. The plane only ever grows.
- **AC-8** The script-gap census is unchanged: this story runs no script node.

## Out of scope

- Anything else in the save's compressed body, including the `Unit` subtree.
- The trailing region's contents past carrying them verbatim.
- Writing a `Fog` section into an original save. This tree never writes one.
- Any rule for how the original expands the recorded cells at draw time.
- Restoring bit 14. It is runtime state the original re-derives, and the save masks `0x8000` alone.
