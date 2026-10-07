# provenance — 0150 the explored map

Research pin: `research` submodule at `1f8902a`. Every fact below is cited by claim id.

## Claims this story builds on

| Claim | Confidence | What is taken from it |
|---|---|---|
| `SAV-FOG-061` | High | The original save carries a per-cell record of explored terrain in the uncompressed tail. The embedded `&YA1` state store holds section `Fog` with two leaves: `FirstState` (int type, int32) and `Data` (int-array type, int32[]). `Data` is a run-length encoding of tile-plane bit 15 over exactly `W x H` cells in linear order `idx = col + row*W`. The first run carries the state in `FirstState`; the state flips between runs. Both the store routine `R0084` and the load routine `R0099` were read instruction by instruction. |
| `SAV-TAILEXT-062` | High | The uncompressed tail is three regions. The state store ends at `0x18 + 32R + 4 + poolLen`, where `R` is the record count. A further 268 bytes follow on a mid-mission save and 310 on a between-mission save. The store is 28 records / 19 leaves on a mid-mission save and 22 / 15 on a between-mission save; the two absent sections are `Fog` and `Projectiles`. A reader that parses to EOF reads 268 bytes of a different structure as pool. |
| `TERR-FOG-145` | High for the writer; Medium for "bit 15 is the only tile bit a save restores" | The load arm at `L07966 OR word ptr [ESI],DI` only ORs the bit in and never clears it. A consumer must persist explored terrain across a save and a load, and must not persist bit 14. |
| `SAV-EMB-004` | Medium; extent clause retracted | The tail store is the REG-style inline key-value `&YA1` container, and the nine top-level key names include `Fog`. The `0x100`-byte label region ahead of it is untouched. The extent clause ("running to EOF") is refuted and is replaced here by `SAV-TAILEXT-062`. |
| `TERR-EDGE-024` | High | Linear cell order over a `W x H` grid is `idx = col + row*W`. This is the order `Fog/Data` runs in, and it is already the order `pkg/sim`'s `cellIndex` uses. |
| `TERR-FOG-087` | High for the census; persistence clause refuted | The shipped-map census stands: no shipped map authors bit 15 outside the section head's overlay artefact at index 3. That is what makes an OR-only load arm reproduce the saved state exactly. |
| `SAV-DOC-053` | High | The compressed body's document `Serialize` does not write the tile plane. The record is in the tail and not in the body, so nothing in this story touches the body walk. |

## Claims that were believed and are now refuted

`SAV-SEEN-055` ("a save carries no per-cell record of what the human participant has seen") and the
persistence clause of `TERR-FOG-087` ("loading restores a fully unexplored map") are both REFUTED in
the pin. This tree contained no reference to either claim id and no code written against them, so
nothing had to be removed. The prose that had been written on their strength is corrected by FR-5.

## What is ours by choice

- **The explored plane stays out of `pkg/sim`.** The original keeps bit 15 in the tile plane, which
  is world state. This tree keeps exploration in `pkg/game`'s `fogPlane` (story 0118), because it is
  per-participant view rather than simulation state. That is a pre-existing structural choice this
  story does not revisit. The consequence is that the serialized byte form does not change: see
  `plan.md` DD-6.
- **Reuse of `pkg/formats/reg` to parse the tail store.** The store's signature dword is
  `0x31415926`, the same value `reg` accepts, and its framing is `reg`'s framing exactly. Parsing it
  with a second copy of the same parser inside `pkg/formats/sav` was rejected.
- **The trailing region is carried verbatim and not decoded.** `SAV-TAILEXT-062` grades its contents
  past the first dword as Unknown. Nothing here reads it.

## What is open

- What the original **draws** from the restored bit is Unknown in `SAV-FOG-061`: the render gate
  `TERR-TILE-079` ORs four neighbouring corner words, so the drawn extent may exceed the set cells.
  This story does not author a rule for that. It restores the recorded cells into the plane and
  leaves the existing visibility path to decide what is drawn.
- The 16.8 % of cells `game0009.sav` records against the owner's own 40-50 % estimate of the
  original's screen is recorded in `SAV-FOG-061` as an open discrepancy and is not resolved here.
  `verification.md` reports the restored counts and says nothing was tuned to close the gap.
- Whether the trailing region's length is fixed or merely constant across two map extents is
  Unknown. This reader does not depend on the length: it takes whatever follows the store's own
  declared end.
