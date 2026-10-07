# 0138-stacks — plan

## Approach

`pkg/sim`'s `World.carried` changes element type from `uint16` to a new exported `ItemStack`, and
one folding helper becomes the single place a container is normalised. Every existing reader of a
container keeps its shape by expanding on the way out, so `pkg/mapload` compiles untouched, and one
new reader carries the counts to the three places that need them: the inventory window's pack area,
the equip command's own gate, and the mission drive.

The byte form is left alone. `pkg/ui` gains a parallel count array on its subject and draws the
number; `cmd/missionrun` gains a take arm so the merge can be driven against a real install.

## Design decisions

**D-1 — `ItemStack{Code uint16; Count uint32}`, exported, in `pkg/sim`.** A struct, not a parallel
counts slice: the two halves of an element cannot then drift apart in a compaction, a decode or a
give-all, and every walk that has to keep them together is a walk over one value. It is named
`ItemStack` rather than `Stack` because `Stock` and `Sack` are already types in this package and a
one-letter difference between three container-ish names is a defect waiting for a typo (chosen over
`Stack`, `Held` and `Carried`).

`Count` is **`uint32` and not the `u16` of `ITEM-STACK-003`'s own field**, and that is the one width
that makes D-2 total. The carry record's length is a `uint32` bounded only by the buffer, and
`normaliseHoldings` takes a `Stock.Items` of any length, so a container of 70 000 of one code is
constructible today and is a payload the previous build both writes and reads. A `uint16` count
would fold it to 4 464 — silent loss, and it would make D-5's bijection false, which is the whole
argument for not moving the version. The count here is not a copy of the original's field, it is a
grouping of a list whose length this form already writes at 32 bits, so it takes that width.

**D-2 — One folding helper, `foldContainer([]ItemStack) []ItemStack`, and every mutation ends with
it.** Chosen over a merging `add(code, count)` used at each site: `equip`'s write-back puts a code
back at a *named index* rather than appending, so an add-only helper would leave that one path to
merge by hand, which is exactly the path that would rot. Folding a whole container is O(n²) over
elements in the worst case and n is a container; a map would be faster and is forbidden on any path
that builds a world (`normaliseHoldings`' own rule).

**D-3 — Serves FR-4: `Carried` and `Stock.Items` keep `[]uint16` and answer the flat expansion.**
Chosen over changing both to `[]ItemStack`. `Stock` is the constructor's *input* as well as its
output, so changing it would push the new type through `pkg/mapload/fromalm.go` and `start.go` for
no gain: an authored map states a list of codes, which is precisely a flat expansion. It also keeps
`Stock()` the exact inverse of the stock argument (FR-4's last sentence) without either side
knowing about counts.

**Keeping the signature is NOT a licence to leave a caller alone**, and one caller must move
BECAUSE it compiles: `pkg/game/world.go`'s `enqueueEquip` bounds-checks the pack cell index against
the flat codes and resolves `codes[idx]` to decide whether the item may be worn, then sends that
same index as the equip command's `cindex` — which after FR-9 names an ELEMENT. With
`[(a,3),(b,1)]`, cell 1 is `b` while `codes[1]` is `a`, so the gate would judge the wrong item and
the bounds check would pass indices no element has. It moves to D-4's reader.

**D-4 — Serves FR-5: one new reader, `(*World).CarriedStacks(id) ([]ItemStack, bool)`**, beside
`Carried` and shaped exactly like it, copy included.

**D-5 — Serves FR-10: `encode` expands, `decodeCarried` folds, `formatVersion` does not move.** The
carry section's own bytes are "a code count, then that many codes"; a stack is a grouping of those
codes rather than a new field, so writing the expansion leaves every offset, every width and every
existing payload's meaning exactly as they were. This is the whole reason the story does not need a
version number, and it holds only because of FR-2 and D-1's width: with at most one element per
code the expansion is injective on every world this package can build, and with a 32-bit count no
expansion this form can carry is too long to fold, so `decode(encode(w)) == w`. Rejected: widening
the record to `code,count` pairs, which is a version bump for state the form already carries.

`encode`'s own preallocation walks the same containers to size the buffer and must count `Σ Count`
rather than `len(codes)`, or the section it sizes is shorter than the section it writes.

**D-6 — Serves AC-10: the decoder folds rather than refuses.** `carryFault`'s split — the
constructor drops a zero code, the decoder refuses one — does not extend here, and the line between
them is what a payload can *claim*: a zero is not an item, so a record naming one is a claim this
package could not have written, while a code appearing twice is two items, which is a claim the
form makes perfectly well and merely writes in an arrangement the constructor would have folded.
The decoder therefore runs the same `foldContainer` the constructor does. Refusing was rejected: it
would make a payload the previous build wrote fail to load for saying something true.

**D-7 — Serves FR-9: `equip` names an element and moves one unit.** Read the element at `cindex`;
write its code into the slot; if its count is above 1 decrement it and add the displaced code (when
non-zero) through the container's own merge; otherwise keep 0112's exact move — the displaced code
overwrites the element in place, or the element is removed when the slot was empty — and fold.

**The fold is what makes this a behaviour change even at every count 1**, and the honest statement
is narrower than "byte-for-byte as before": a container `[a,b,c]` with `c` already worn used to
become `[c,b,c]` and now becomes `[(c,2),(b,1)]`, whose expansion is `c,c,b`. The duplicate arrives
from the equipment slot rather than from the container, which is why FR-10 states the order change
in terms of the acts rather than in terms of what a container held beforehand. 0112's AC-3 and AC-4
tests use distinct codes and stay green on their own terms; a new test names this case rather than
letting it ride on their silence.

**D-8 — Serves FR-7: the corpse drop expands.** `step.go`'s drop builds the sack's list from the
flat expansion, then appends the twelve slots as it already does. A sack stays `[]uint16` and
`pourSack` is not touched. 0132's DD-3 note there — that the container slice is HANDED OVER rather
than copied, so a second drop is unrepresentable — stops being true the moment the list is a fresh
expansion; the field is still nilled, so the guarantee holds for a new reason and the note says the
new one rather than the old one.

**D-9 — Serves FR-11: `InventorySubject` gains `PackCount [invPackCells]uint32`** — an array and
not a slice, because the subject is held by value and compared for equality as a cache key, which a
slice field would stop compiling. `RenderInventory` takes the font as a second argument (its one
non-test caller is inside the viewer, which already holds one; twelve more are in that package's
own tests), draws a count of 2 or more into the cell's lower-right corner over the icon, and draws
nothing at 1, at 0, or with a nil font — so the window never fails to compose (0112 P-2). Chosen
over a second exported renderer, which would double the composition path for one string.

**The font joins `inventoryKey`.** The presented picture is now a function of the font as well as
the subject and the area, and the viewer composes before `SetFont` arrives as readily as after, so
a key without it would hold a countless window past the moment a font existed to draw one.

**D-10 — Serves FR-11: `buildInventoryPack` takes `[]sim.ItemStack` and answers the counts beside
the pictures.** It already walks one entry per cell and stops at the array's length; the entry
becomes an element rather than a code, and the icon is still keyed by code, so the cache is
untouched. `pkg/game/inventory.go`'s figure composition and its icon loader are not in this story.
`mapWorld`'s `invCodes` — the field `refreshPack` compares against to decide whether to recompose —
becomes `[]sim.ItemStack` with it, so a container whose ELEMENTS are unchanged but whose COUNTS
moved still recomposes.

**D-11 — Serves FR-12: `-take TAKER:X:Y` in `cmd/missionrun`,** repeatable, driven after the
attacks, parsed by the same three-field shape `-waypoint` already uses. Each take prints what
`TakeSack` answered and then the taker's elements. `victimHoldings`' carried line prints elements
too, so the before-report and the take-report can be read against each other.

## Risks

- **A digest pinned in a test moves** because the world it pins holds one code twice and now folds
  it. That is a real behaviour change, not a form change; a pin that moves is re-taken and the move
  is recorded rather than papered over, and a pin that does not move is not touched.
- **`equip` is shared ground with another story running in parallel.** The change here is inside
  the existing function body and adds no argument, so a guard added at its head merges cleanly; the
  branch is re-merged with master and rebuilt before the final gate.
- **The drawn count is unreadable at 48 pixels.** Mitigated by drawing it in the window's own font
  with the same shadow the damage numerals use, and witnessed by a pixel-difference test rather
  than by eye.

## Success criteria

1. Every AC in `spec.md` has a test in the package that owns the behaviour, and `AC-9`'s digest
   evidence names the constant it compares against.
2. `formatVersion` is unchanged, and the byte-form offset table is unchanged.
3. `go test -count=1 -trimpath ./...` is green, and the four repository scripts pass.
4. A mission drive on **both** installs fells a unit, takes its sack and prints one element at a
   count above 1.
5. No package outside `pkg/sim`, `pkg/ui`, `pkg/game/inventory.go`, `pkg/game/world.go` and
   `cmd/missionrun` changes.

## Traceability

| Design | Serves |
|---|---|
| D-1, D-2 | FR-1, FR-2, FR-3, P-1, P-4 |
| D-3 | FR-4 |
| D-4 | FR-5 |
| D-5, D-6 | FR-10, AC-9, AC-10, P-3 |
| D-7 | FR-9, AC-7, AC-8, P-2 |
| D-8 | FR-7, AC-5 |
| D-2 | FR-6, FR-8, AC-4, AC-6 |
| D-9, D-10 | FR-11, AC-11, AC-12, P-5 |
| D-11 | FR-12, AC-13 |
</content>
</invoke>
