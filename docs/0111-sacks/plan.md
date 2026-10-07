# Plan — a sack on the ground is drawn

## Approach

A sack is cell-anchored content standing on the ground, so it becomes a **fourth stream of the
existing drawn band** rather than a pass of its own: one sheet decoded at start-up and handed to
each window that opens, a per-refresh list of cells pushed beside the entity list, and one
placement expression turning each into the same placed rectangle the band already merges, culls,
tints and paints. Lighting, culling and depth are inherited by joining the band rather than written
twice. The simulation is not touched at all.

## Facts verified during planning

- `pkg/sim` holds the ground-sack list, orders it ascending by (Y, X), serializes it and hands it
  out by copy through `(*World).Sacks()`. **This story adds no field to the serialized form**, so
  the byte-form version is unchanged. `sackFault` refuses a sack outside the world's bounds at
  construction, so no world this build makes can hold one.
- `terrain.DepthOrder` takes **four lists and an order**: `[]PlaneRef` (the once-per-map
  structure/object arrangement `PlaneOrder` fixed), `[]StructurePlacement`, and two
  `[]StaticPlacement` — the objects and the per-frame entities. It returns index references.
- That merge has **three parts**, and only the middle one is a merge: a leading run of
  ground-decoration (`Flat`) structure refs emitted untouched and **first whatever their rows**; a
  walk over the remaining order that flushes pending entities while their row is **strictly less**
  than the current ref's, guarded by whether that ref's row could be read at all; and a **tail
  drain** of every entity left over. The entity list is **sorted inside** `DepthOrder`, over an
  index permutation, so no caller carries half the ordering rule.
- The decoration prefix is painted by a **separate earlier pass** and skipped inside the merge
  walk, so it is not merely first in the order — it is a different pass.
- `pkg/ui.Viewer.planeSprites` walks the order, culls each entry through the package's one
  `spriteScreenRect`, and `drawPlane` paints it. Every frame goes through `staticImage` and
  `spritePixels`, which apply the day/night light row and tint and cache the texture by frame
  pointer, row and tint. `spritePixels` returns a CPU image and is the observable seam; an
  `*ebiten.Image`'s pixels cannot be read back before the game starts.
- The shadow pass reads the same three class-driven lists as the content plane and deliberately
  builds no fourth; it does not walk the depth order at all.
- `terrain.StaticAnchor` takes the canvas size, the centre and **the drawn frame's own size** as
  separate positional integers. `terrain.UnitPlace` is the existing example of building a
  `StaticPlacement` for a per-frame list from them.
- `pkg/game.sheetCache` decodes a `.256` and converts **every** frame to `[]*terrain.StaticFrame`,
  returning nil for an absent, undecodable or palette-less sheet. It **prefixes the container
  identity at the read**, so what it takes is a container-relative path and not a full address.
- **`NewFrontEnd` builds no viewer.** A viewer is constructed per map, inside `LoadMapViewer`, and
  every start-up-loaded resource a viewer needs is installed by a setter on **both** front-end
  paths — the map picker's and the mission's — as the font and the attack pointer already are.
  `cmd/mapview` reaches `LoadMapViewer` directly, with no front end; `cmd/terraintool` builds no
  viewer at all.
- `mapWorld.push` is the **entity** hand-off — `SetEntities` from a list `entityDraws` builds from
  the world, and `SetLightClock`. It is not the only per-tick hand-off; the readout is another.
- `cmd/missionrun` is headless and builds no viewer, so no code path this story adds is reachable
  from the milestone drive.
- `pkg/render/terrain`'s tests are an **external** test package; `pkg/ui`'s and `pkg/game`'s are
  in-package. So a render-tier signature change reddens `go vet` and `go test` until its own test
  call sites move with it, and a `pkg/game` test can read the pushed list without an accessor.

## Design decisions

**DD-1 — The drawn sacks cross the seam per refresh, from the world's own list.** `mapWorld.push`
gains a second setter call beside `SetEntities`, fed by a builder that reads `(*World).Sacks()`
(FR-3, FR-4, FR-11). *Rejected:* building them once at map load as extra static-object placements —
cheaper today, since nothing creates or destroys a sack, but it makes the drawn set a second copy
of state `pkg/sim` owns, which the first pick-up would have to invalidate, and it would put sacks
behind the map's art switch.

**DD-2 — A sack joins the band as a fourth stream.** `terrain.PlaneKind` gains `PlaneSack`
**after** `PlaneEntity`, so the existing constants and the zero value are unmoved, and
`DepthOrder` takes a fifth list. Its three parts change as follows (FR-7):

- the decoration prefix is emitted **unchanged and first**, and no sack is ever emitted into it;
- in the walk, a ref of row `R` is preceded by every sack and every entity of row `< R`, emitted
  **interleaved by row with the sack first at an equal row**, under the same readability guard the
  entity flush already carries;
- the **tail drain** applies the same interleave to everything left of both streams — not one
  stream exhausted before the other, which would put a near sack in front of a far entity.

The sack stream is **sorted stably inside** `DepthOrder`, over an index permutation, on the ground
the entity stream is: a caller must not carry half the ordering rule, and a production list that
happens to arrive sorted would hide an unsorted implementation. *Rejected:* appending sacks to the
entity list, which is indexed by the window's own entity records for selection, hit-testing and the
path overlay — a member of it that is not an entity would have to be excluded from each by hand.

**DD-3 — The sack's placement is its own function in the render tier**, taking a cell, a frame and
the lift pair, and calling `StaticAnchor` with the canvas equal to the drawn frame's size and the
centre equal to that canvas's centre, so the frame's centre pixel lands on the ground point (FR-5,
FR-6). *Rejected:* reusing `UnitPlace` with a synthetic class, which would need a `UnitClass` whose
only true fields are a canvas and a centre and every other field of which would be a lie.

**DD-4 — The front end owns the sheet; each viewer is handed it; the game tier owns the rule.**
The frames are decoded once in `NewFrontEnd`, kept in a `FrontEnd` field beside the other bundles,
and installed by a viewer setter on **both** opener paths, at the point the font and the attack
pointer are installed on each. The pushed sack record carries **a cell and a frame index**, and the
index comes from one named function in `pkg/game` that today answers a constant (FR-1, FR-2).
*Rejected:* handing the frames through `LoadMapViewer`, which would change a signature `cmd/mapview`
also calls and would decide the diagnostic tool's behaviour as a side effect; *rejected:* the pushed
record carrying a resolved frame pointer as an entity's does, since the frames would then have to
reach `mapWorld`, whose constructors are called from about forty test sites; *rejected:* the window
selecting the frame, which would put an authored decision in the one tier that cannot see the
payload it is about.

**DD-5 — A viewer opened without the front end is handed no sheet and draws no sack** (FR-1, FR-10).
That is one setter not called, not a branch. *Rejected:* wiring the tool's path too — a map viewer
is an instrument, and world content it cannot interact with is scope with no reader.

**DD-6 — The address is a container-relative path constant in the game tier**, of the form the
sheet cache takes — **not** joined to the container identity, which that cache prepends itself
(FR-1). The loader returns frames and no error, on `LoadHeroBody`'s precedent rather than
`LoadStatics`'s: a missing registry is an install missing a piece the game needs, a missing
cosmetic sheet is not. *Rejected:* a registry lookup — no registry this build reads names the sheet.

**DD-7 — An out-of-range index and an ungridded cell are excluded where the placement is built**,
in the window tier, exactly as an off-map entity and a frameless one already are: they contribute
no placement, so they are absent from the merge rather than present and skipped (FR-10).
*Rejected:* refusing them at the push, which would make the push decide what is drawable.

**DD-8 — No shadow entry and no overlay composite are added** (FR-8, FR-9). The shadow pass keeps reading
the three class-driven lists it reads now, and a sack's painted pixels are the sheet's frame under
the band's own light row and tint and nothing else. *Rejected:* compositing the overlay sibling,
which would make the sack the only sprite in the build that draws one.

## Files to touch

**`pkg/render/terrain/`** — `structures.go` `MODIFY` (`PlaneSack`, the fifth list, the three
changed merge parts); `sack.go` `ADD` (the placement); `structures_test.go` `MODIFY` (its three
existing call sites); `sack_test.go` `ADD`.

**`pkg/ui/`** — `viewer.go` `MODIFY` (the frames and the pushed list as state, and their two
setters); `statics.go` `MODIFY` (the sack layer, the `PlaneSack` arm, the real fifth argument);
a test file `ADD`.

**`pkg/game/`** — `sacks.go` `ADD` (the path constant, the loader, the frame rule); `world.go`
`MODIFY` (the drawn-sack builder and its call in `push`); `frontend.go` `MODIFY` (the field, the
load, and the setter call on **both** opener paths); a test file `ADD`.

`pkg/game/mapload.go` and `cmd/mapview` are deliberately **not** touched: DD-4 and DD-5 keep the
viewer constructor's signature and the diagnostic path's behaviour.

## Risks

**R-1 — The chosen frame is not the one the original draws.** Every sack is then drawn at the wrong
size. *Mitigation:* the rule is one function behind one seam field; the cost is cosmetic and no
cell, list or order depends on it.

**R-2 — The sheet is not a sack's art at all.** The player then sees something that is not a sack
lying on the ground. *Mitigation:* SC-8 is the criterion, and it is cheap — four known cells on one
shipped map.

**R-3 — A fourth stream reorders the three already there.** The regression is subtle: a unit drawn
through a wall, or ground decoration jumping in front of the map. *Mitigation:* the merge's existing
tests already pin the decoration prefix ahead of a lower row and the entity behind a same-row
structure; those frozen expectations must not move.

**R-4 — The setter is called on one opener path and not the other.** Mission 10 then draws nothing
while the map picker's path works. This is the sharp one: FR-1 and FR-10 make a viewer with no sheet
a silent, lawful state, so **no automated criterion here can detect it** and SC-8 is the only thing
that can. *Mitigation:* the two call sites are named in the same design decision as the field, and
SC-8 is run on the mission path specifically.

**R-5 — The path constant is spelled as a full address.** The sheet then never opens, and FR-1
turns that into silence too. *Mitigation:* the same as R-4 — SC-8; and SC-1 is written against a
loader whose fixture is addressed the way the real one is.

## Success criteria

| # | Condition | How |
|---|---|---|
| **SC-1** | Every frame of a synthetic multi-frame sheet is loaded in order, addressed as the real one is; an absent, an undecodable and a palette-less sheet each yield none, and none is an error | automated (AC-1, AC-2) |
| **SC-2** | The pushed sack list equals the world's own list, entry for entry and in order, and two refreshes of an unchanged world agree | automated (AC-3, AC-4) |
| **SC-3** | A placed sack's centre pixel is at the cell's ground point, and its flat and displaced placements differ only in Y, by exactly the lift and the origin | automated (AC-5, AC-6) |
| **SC-4** | The merge's existing frozen expectations are unchanged with an empty sack stream; with sacks, the decoration prefix still leads, the rest is ascending by row, ties resolve art, sack, entity, and a leftover far sack precedes a leftover near entity | automated (AC-7) |
| **SC-5** | A sack's frame and a static frame beside it produce the same light row and tint key, and the same CPU pixels, at a dark and at a bright clock | automated (AC-8) |
| **SC-6** | The shadow pass produces nothing for a sack, and a sack's CPU pixels are the decoded frame's own under that row and tint | automated (AC-9) |
| **SC-7** | An ungridded cell and an out-of-range frame index each yield no placement, raise nothing, and leave the rest of the band untouched | automated (AC-10) |
| **SC-8** | Mission 10 opened **from the mission path** against a lawful install draws four sacks, at the four ground cells, and none at the stock cell | developer-run (AC-11) |
| **SC-9** | The headless mission-10 drive reports the same outcome at the same tick as before this story, on both roots | developer-run (AC-12) |
