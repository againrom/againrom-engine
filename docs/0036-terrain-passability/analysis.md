# Analysis — the block plane a map already carries, and the one it does not

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** — the derived grid is hashed simulation state, which the profile puts at this tier; no watcher exists, so it is discipline |
| Terrain — the derivation itself | **greenfield**: nothing in this tree classifies a tile word for movement, in any spelling |
| Terrain — `mapload.FromALM` and every test that walks a map-built world | **brownfield**: the loader's grid argument is shipped behaviour, and changing it moves those worlds' trajectories and digests |

## The baseline describes a tree that is not this one

Each sentence was re-derived against the code and against the pin, never translated:

| The baseline says | What is actually here |
|---|---|
| "the shipped derivation (`pkg/mapload/passability.go`)" computes a grid from a type-class split and a **64-value table transcribed from a reference**, which this spec demotes | **No such file, and no derivation anywhere.** There is nothing to demote: `FromALM` passes `nil`. The story is greenfield, not a cleanup, and its whole risk is what it asserts rather than what it withdraws |
| `WorldFromALM` | `mapload.FromALM` |
| 0029 "verified the sim-side grid contract (canonical state, enterability, **A\*/occupancy**)" | there is **no A\***: canonical mode is a label-correcting wave with no distance term, optimised mode a Dijkstra over a clipped region (0029 FR-5, FR-6) |
| FR-2: a map with no blocked cell yields a **`nil` grid**, "byte-identical to the pre-grid World" | unreachable and inert. The decoded border blocks the outer eight rings of **every** map, so no non-degenerate map has an empty grid; and `sim` already makes no-grid and all-zero **one representation** (0029's own design), so returning `nil` would assert nothing |
| FR-5: `blockedAir` is **never** set here — "flyers cross water" | **false against the decoded plane.** The air bit is set by the border and by nothing else, so the border is its only writer; declining to set it leaves the bit dead and a later flyer story with no seam |
| R-1/R-2/R-3: the class split, the mountain rule and "water blocks ground" are **not yet derived from the game** | all three are closed at High. The split, the strip-pair table, the five-level blend, the Mountain compare and the water range are named instructions with their immediates |
| the mountain rule is a threshold on a 64-value **fill** table | there is a 4x14 table, and it is a **blend level** selecting primary over secondary — not a fill fraction, not mountain-specific, and consulted for every non-water cell |
| "water family strips `0x20`-`0x2F`" | the same set as the decoded range `[512,768)`, arrived at differently. The one baseline claim that survives contact |

## Two decline-by-pointing clauses were chased to what they point at

- **"Any `pkg/sim` change — the grid contract already exists (0029)."** Located and **true**: the
  grid, its two bit meanings, the reserved-bit refusal, `terrainOpen` and the grid section of the
  version-5 byte form are all shipped. The clause stands, and this story touches no file under
  `pkg/sim`.
- **"Structures / blocking objects — need the class footprint / block flag decoded first."**
  Located, and true for a sharper reason than the baseline had: the footprint *is* decoded — a
  rectangle plus a `Passability` and a `BuildingPresent` mask — but it lives in the `Data.bin`
  Buildings table, and **nothing in this tree reads `Data.bin`**. A search over the whole module
  finds it only inside the research submodule's own tools. So the clause survives as a non-goal on a
  premise that is now precise instead of vague.

## What the fresh research changed about the shape of the answer

The rival this story expected to settle — "buildings are baked into the plane at load, like water"
— is **refuted for structures and true for scenery**. The `.alm` type-3 layer is baked by the
ingest, one assignment per cell. A placed type-4 structure never goes through it: it attaches in
its own constructor, strictly after the ingest, to a per-cell record, and the plane byte is then a
**cache recomputed from that record** — which is why a footprint cell can *subtract*, and why a
bridge crosses water at all.

So there are two models, not one, and a bake-once grid can neither demolish a building nor lay a
bridge. This story bakes, because the attach model's inputs are two undecoded tiers away, and the
foreclosure is stated in the contract rather than discovered later. The measured price is 1113
shipped cells the original opens and we leave blocked, 908 of them bridge decks.

## Where a criterion can discriminate, and where nothing can

Four of the rules are **not** separable by the shipped corpus, and each is separable by a word a
test can write:

- **assignment versus union.** The ingest's four block writes are assignments, so the scenery arm
  overwrites the terrain arms. Under a two-bit projection the overwritten and the OR-ed byte are
  equal, so no criterion here discriminates them — and none in the corpus does either, `1 | 5` being
  `5`. Stated once and never asserted as behaviour.
- **the water arm read raw versus read through the classifier.** The two agree on all 880 704
  shipped cells and are algebraically one set over the reachable groups. They part on a word inside
  the range whose low nibble is 8 or more, which **0** shipped cells carry and a test builds in one
  line.
- **the sim split versus the render split.** They differ only on words setting bits 10-12; 0 shipped
  cells do, and a synthetic word does.
- **the reject arm and the unwritten groups.** A low nibble of 14 or 15, and a strip group of 13 or
  more, occur 0 times in the corpus; the second reads uninitialised heap in the original, so there
  is nothing to be faithful to and the choice is ours.

The corpus is therefore the wrong instrument for four rules and the right one for the whole: a
census of the derived plane over a real install reproduces published per-value counts cell for
cell, which is the only end-to-end check this story can have.

## Left open for the plan

Where the derivation lives and what it is called; how the border is expressed without an underflow
on a narrow map; whether the fixtures that walk a map-built world grow or the border is
conditioned on extent; and what the instrument prints. None is a contract question.
