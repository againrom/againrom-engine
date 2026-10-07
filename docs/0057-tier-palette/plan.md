# Plan — feed the existing builder different bytes

## Approach

Five layers, each the smallest thing that can carry its own half of the contract, and one
instrument.

A new formats leaf takes a byte stream to 256 colours. The data tier, which already owns a class's
sprite address and its registry keys, gains the name construction and the bounded tier count. The
render tier's unit class, which already carries the whole sheet as a frame slice, gains **a slice
per tier** and one total selector over it. The loader in the top tier — the only tier that may open
a container and decode a sheet — reads the named entries and builds those slices. The game's map
screen resolves each placement's tier once when the map opens, beside the world and outside it, and
hands the selector that number at the one site that already picks a frame.

Nothing on the draw path changes shape: the tier decides **which frame slice** a class is indexed
in, and everything after that — the ramp, the blit, the anchor, the texture cache keyed on frame
identity — is untouched.

## Facts verified during planning

- `StaticFrame` carries `Palette [256]color.RGBA` inline and a `Pixels` slice; a blit takes a frame
  alone. The window tier's texture cache is keyed on the **frame pointer** and the ramp row, so two
  frames differing only in palette already upload as two textures and one frame shared by two
  classes already uploads once.
- `pkg/game`'s sheet cache memoises a converted `[]*terrain.StaticFrame` per sprite path, so two
  classes naming one sheet already receive one slice, pointer for pointer.
- `data.UnitClass` already parses `Palette` (int, absent-default 0) and already exposes
  `SpritePath()` built from a private extension-less base.
- `mapload.FromALMWith` documents that the i-th unit record takes entity id i; `mapload.Resolve`
  and `data.NewUnitDef` are both exported, so a tier can be resolved without touching `pkg/mapload`.
- `sim.Entity` has no owner and no face field, and `pkg/game/world.go` already holds four
  per-entity memories born with a map and read by nothing canonical.
- On both lawful roots: 16 classes carry a `Palette` above 0, the 55 addresses the construction
  builds all exist, all begin `BM`, all are at least `0x436` bytes, and tier 1's table equals its
  sheet's own on 16 of 16.
- The whole gate passes on the branch point at `EXIT=0` with an empty FAIL set.

## Files to touch

| File | Change |
|---|---|
| `pkg/formats/pal/{pal.go,doc.go,pal_test.go}` | new leaf: the decoder |
| `internal/archtest/dag.go` | register the leaf; deny it the text module |
| `docs/ARCHITECTURE.md` | the tier row and the DAG row for the new package |
| `pkg/data/sprite.go`, `pkg/data/classes.go` | the built name; the bounded tier count |
| `pkg/render/terrain/units.go` | the per-tier frame slices and the selector |
| `pkg/game/units.go` | read the entries, build the slices, count the fallbacks |
| `pkg/game/tiers.go` | a map's placements to their tiers |
| `pkg/game/world.go` | carry the tiers; select with them at the one draw site |
| `pkg/game/frontend.go` | hand the map's own table and map to that resolution |
| `cmd/terraintool/main.go` | the `tiers` subcommand: census and still |
| tests beside each | |

## Design decisions

**DD-1 — the decoder reads a fixed window and parses nothing.** `Decode([]byte) (Table, error)`
where `Table` is `[256]Color` of three bytes. It checks length and magic, then reads 1024 bytes at
`0x36`. It does **not** parse the BMP header — no width, no height, no `clrUsed`, no offset field —
because the engine does not either, and a decoder that honoured a header would disagree with the
engine on exactly the file whose header disagrees with its content. Rejected: reusing
`terrain.DecodeBMP8`, which is a full 8-bpp image decoder in a tier the formats leaf may not import
and which would refuse files the engine reads.

**DD-2 — the magic check is a positive exclusion, and it is the only rule that is ours.** The one
file of the other shape begins `00 00`; every address this story builds begins `BM` on both roots.
Refusing on the magic makes handing the wrong file a named error rather than 1024 bytes of
someone's pixels. It was measured before it was written (55/55 on each root), not assumed.

**DD-3 — the name lives beside `SpritePath`, and the count beside the key.** `PalettePath(tier
int) string` is a pure string function of the same private base the sprite path is built from, so
the separator, the prefix and the directory rule exist once; `TierCount() int` clamps the parsed
key against `TierLimit`. Both are in the tier that owns the registry, so the loader above states no
convention of its own. Rejected: building the name in the loader — it would put half of one
formatter two tiers above the other half.

**DD-4 — `TierLimit` is a constant in one place and the clamp is the only reader.** That is what
makes FR-3's seam real: raising the number is one edit and no other line of ours mentions 4. It is
in the data tier rather than the render tier because it bounds a **registry key**, not a picture.

**DD-5 — a tier is a frame slice on the class, and the selector is total.** `UnitClass` gains
`Tiers [][]*StaticFrame` and `TierFrames(tier int) []*StaticFrame`, which answers `Tiers[tier-1]`
when that is in range and non-empty and `Frames` otherwise — so tier 0 (no tier stated), a negative
tier, a tier past the count and a tier that failed to load are one arm and not four. Rejected: a
palette argument threaded through the blit and the texture key. That is a second draw path beside
the one that works, it moves the colour off the frame that FR-5's identity clause depends on, and
the claim says the tier feeds the existing builder different bytes rather than adding a path.

**DD-6 — the recolour copies the table and shares the pixels, and returns the base slice
unchanged when the table already matches.** One helper takes the base slice and a table: it
compares (a `[256]color.RGBA` is comparable, so this is one array equality per frame), returns the
base slice itself on equality, else builds one new frame per base frame by struct copy with the
table overwritten — which shares `Pixels` by construction rather than by remembering to. On both
lawful roots the equality arm is what every class's tier 1 takes, so tier 1 costs no memory and no
texture; a modded `palette.pal` that differs would be honoured rather than silently ignored, which
is the customisation half of the same decision.

**DD-7 — the loader memoises per (sheet address, table address) pair.** The existing per-path memo
answers the base slice; a second map keyed on both addresses answers the recoloured one, so two
classes naming one sheet and one table share one slice and one texture, exactly as they already do
for the base. Failures are remembered like successes — presence, never the value, says an address
was tried.

**DD-8 — the fallback is a shape, not a counter.** `LoadUnits` keeps its signature, and the per-tier
slice carries **one entry per tier the class declares**, nil for one that could not be built. So
`len(Tiers)` is the declared count and a nil entry is a fallback, and FR-6's "reportable" is
answered by the bundle itself: the instrument sums what is there rather than trusting a number
somebody kept in step with it. Rejected: a count field beside the slices, which is a second fact
about the same slices and can disagree with them; and rejected: a loader that logs, which is a
loader with an output.

**DD-9 — the tier is resolved at map open into a lookup beside the world.** A `map[sim.EntityID]int`
built from the decoded map and the definition table through `mapload.Resolve` +
`data.NewUnitDef`, keyed by the id that loader mints, held on the same struct the facing, step,
death and command memories are held on: born with the map, dropped with it, looked up and never
ranged, read by nothing canonical. That is what makes FR-9 structural — there is no field of
`sim.Entity` to widen and no line of `pkg/sim` to touch. Rejected: carrying the tier across the
draw seam as a new field. The seam already carries the resolved frame; adding the tier would make
the window tier hold a number it must not interpret.

**DD-10 — the constructor gains a sibling, not a parameter.** `newMapWorld` keeps its four
arguments and is defined in terms of a five-argument form taking the lookup, so there is one
construction implementation and "a screen with no tiers behaves exactly as before" cannot decay
into two paths that agree today — `FromALM`/`FromALMWith`'s own shape, one tier down.

**DD-11 — the draw site reads the selector, and both arms read it.** `entityDraws` already picks
between a live class and a substituted corpse class; each arm replaces its `c.Frames` with
`c.TierFrames(tier)` and indexes the answer. One number, looked up once per entity, used by
whichever class supplies the frames — which is FR-11 with no branch of its own.

**DD-12 — the instrument is a `terraintool` subcommand with two outputs and one load.** `tiers`
opens the install's archives once, loads the bundle, prints the census, and — only when a class and
an output path are named — composes the still. Panels are the drawn frame's own size with a fixed
gutter on a neutral opaque field, each drawn through the same lit blit the window builds textures
with, so the picture is the game's pixels rather than a second palette walk. The frame index is the
class's own idle selection at octant 0 and tick 0, identical for every tier, so the panels differ
in colour alone. Rejected: a second tool, and rejected: putting it in `classdump`, which may not
reach the render tier.

## Risks

- **R-1 — memory.** A tiered class holds up to four frame slices, each frame carrying a 1 KB table
  beside a shared pixel slice. Bounded by DD-6 (tier 1 costs nothing on shipped data) and by the
  texture cache being lazy: a texture exists only once a frame is drawn.
- **R-2 — a wrong colour has no ground truth on screen.** Nobody can see that index 37 resolved
  wrongly. Mitigated by AC-9's byte comparison against the file itself and by the published
  discriminating word for one class, entry and level — an exact integer, not a look.
- **R-3 — the tier lookup and the world's ids drifting apart.** The lookup is keyed by the id
  `FromALMWith` mints, and both are built from the same map in the same function; a placement the
  world never built cannot be looked up, and a lookup miss is `TierFrames`' first arm.
- **R-4 — the sibling lane.** `pkg/sim` and `pkg/mapload` are untouched: the resolution reads
  `mapload`'s exported entry points and adds nothing to that package.

## Success criteria

| # | Criterion | Serves |
|---|---|---|
| SC-1 | The decoder is total over generated streams and exact over a synthesised table | FR-1, FR-1a |
| SC-2 | The built name and the bounded count are exact over their whole stated domains | FR-2, FR-3 |
| SC-3 | The selector is total and answers the base slice for every out-of-range tier | FR-8 |
| SC-4 | A synthetic container exercises all four tier outcomes: equal, differing, absent, refused | FR-5, FR-6 |
| SC-5 | Two placements of one class at two tiers reach the draw seam with different tables and equal pixels | FR-7, FR-11 |
| SC-6 | A world's entities, byte form and digest are unchanged by tier resolution | FR-9 |
| SC-7 | Both lawful roots: every built address exists, decodes and equals the file's own bytes; tier 1 equals its sheet; no fallback | FR-4, FR-6, FR-12 |
| SC-8 | The still exists outside both repositories and shows one class's tiers side by side | FR-10, FR-12 |
| SC-9 | The gate's FAIL set is empty and the deletion set of the landing is empty | — |
