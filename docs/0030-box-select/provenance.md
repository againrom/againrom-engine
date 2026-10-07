# Provenance — selecting a group and ordering it from the running game

Pinned at research `ce40c15`, frozen for the story; `claims/retracted.md` and the registry's standing
corrections were read first; no row bearing here has been taken back.

**Appended 2026-07-31.** The papers were authored at `ce40c15` and the code lands at `a13b3b8` — the
0029→0030 boundary bump (`6a1154a`) fell between the two, which is where S-5 puts it. Nothing here
moves: the one claim this file cites, `MENU-MASK-004`, is untouched across that range, and the 44
ids the bump added were all created after the older pin, so none could have been consumed. The
sentence above records where the reading was done; this one records where the story ran.

## Backing

**No row.** Not one anchor of this contract is derived from a research claim, and the search
establishing that is on the record. The pinned ledgers — ALM, DAT, FAME, INV, MENU, MOVE, REG, RES,
SAV, SHOP, TERRAIN — hold **no selection, marquee, drag, modifier or group-command machinery at all**;
the one decoded pointer pick in the corpus is the main menu's (`MENU-MASK-004`, High), a per-pixel
8-bit index mask over eight buttons. What they *do* hold is the original's ordering, reservation and
wait machinery, every part of it touching this story differing. Those claims appear below as
divergences from a choice of ours, never as backing.

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| **A drag draws a selection box** (FR-2, FR-6) | Nothing decoded says the original had one. The gesture set is a modern RTS convention we adopt deliberately — not a reconstruction, so there is nothing here to be faithful to |
| **`Shift` takes panning; the unmodified drag becomes the box** (FR-1, FR-7, C-2) | Nothing decoded touches the original's camera gestures or any modifier key. Entirely ours; the front-end reads none today |
| **A unit is caught by cell coverage** (FR-2, C-1) | Nothing decoded. Worth naming beside it: the one picking technique on the record (`MENU-MASK-004`) is **shape-accurate to the pixel** where ours is accurate to the cell. No claim says how, or whether, units were picked at all |
| **Every ordered unit is sent to the same cell** (FR-4, C-3) | Nothing decoded says the original had a group order. `ALM-GRP-020` / `ALM-GRP-041` are the map file's **player/faction roster** — `Self, Monsters, Villagers, …`, authored data — so the only decoded "group" in the corpus is not a player-formed control group, and reading it as one is the mistake this row exists to prevent |
| **Commands are emitted in ascending `EntityID`** (FR-4, P-4) | **Flatly divergent, and the claims are explicit.** `MOVE-TICK-009` (High) — the per-tick update imposes **no priority**, so update order alone decides who gets a contested cell. `MOVE-TICK-013` (High) — the container is a pooled `CObList` whose only insert is `AddTail`, walked head→tail, its order pure insertion history, fixed at load as the campaign party then the type-6 records in record order (`MOVE-TICK-014`, High) and not preserved across save/load (`MOVE-TICK-017`, standing correction). `MOVE-ID-016` (High) — the runtime id is a lowest-free-bit bitmap allocation, freed at corpse decay, and **never the walk order**. So there an id is explicitly *not* an ordering key. We make it one, for reproducibility, with the divergence known |
| **The selection is replaced, never added to** (FR-2, FR-3) | Nothing decoded. Ours; the modifier that would add is spent on panning (C-2) |
| **A group ordered to one cell arrives one unit deep, the surplus giving up** (FR-4, disclosed limitation) | **Half convergent, and the divergent half is ours and worse.** `MOVE-WAIT-008` (High) agrees on the near half: a unit whose next cell is occupied **waits facing it**, and neither pushes, swaps nor steps aside. `MOVE-TERM-003` (High) carries the far half we lack — when the wave's slack budget fails "the caller then substitutes a nearby goal", through the `altTarget` parameter of `MOVE-SEARCH-001`'s own signature. Ours stalls sixteen ticks and clears the target instead. Not fidelity; disclosed as ours. **Two corrections appended 2026-07-31 at the `8c92427` pin, not substituted, because this is a dated record.** The quoted clause is **retracted at High** — the search substitutes in its own tail and nothing outside it calls either picker (`MOVE-ALT-018`) — and `altTarget` is **not** the substitute cell: it is a target **actor**, which selects the contact-ring picker instead of the ring-around-a-cell one (`MOVE-ALT-020`). The rule itself is now read: rings from the requested cell, **minimum label** inside a ring, so the substitute is the cell cheapest to reach *from the mover* and "free" means *labelled* (`MOVE-ALT-019`/`-021`). And the far half we lack is smaller than this row implies — **no multi-unit distribution exists anywhere** (`MOVE-ORDER-023`, Medium): a group order writes one cell to every member there too, and the original's spread is emergent from movers claiming cells in turn |
| **Selection, rectangle and press state are front-end only, never hashed** (FR-8, P-2) | The original keeps comparable per-actor state *in* the world: `MOVE-CLAIM-007` (High) has `mover+0x80` holding the cell a unit intends to enter, marked on the shared plane before arrival. Ours has no reservation at all, and nothing this story adds reaches a world field, a byte or a digest |

## Open / undecoded

- **Whether the original had a box-select or a group order at all.** Nothing in the pinned ledgers
  speaks to it, and no research item was opened — a decision, not an omission: the gesture set is a
  convention we chose, so a decoded answer could not change the contract, only say afterwards how far
  from the game it lands.
- **The substitute goal is decoded and deliberately not consumed.** `MOVE-TERM-003` gives the
  mechanism, `MOVE-SEARCH-001` its parameter. Consuming it changes what a unit does when its route
  fails — `pkg/sim`'s business, forbidden here outright (FR-8). A later story makes it ours to
  schedule; until then the give-up is disclosed, not presented as recovered behaviour.
- **Contention after a load.** `MOVE-TICK-017` warns a consumer not reproducing the regrouping that
  it diverges after any load. We diverge from the first tick anyway, by choice.

## Confidence, and the High threshold

Policy sets a **High** threshold for anything reaching hashed simulation state. Nothing here reaches
it: the story adds no line to `pkg/sim`, no world field and no canonical byte, so no anchor carries a
confidence and a graded row would be a fabrication. Every claim above is High, and each is cited as a
**divergence** — constraining nothing we build, recording what we are not.

## Removed from the baseline and why

- Its provenance-basis preamble and its closing section reporting that no research item is needed —
  both refused outright by the doc-budget content bans.
- **`pickUnitAt`, and its "sprite rect, or a `32x32` cell rect when undrawable" geometry.** No such
  function, and no sprite-rectangle pick either; the shipped pick is cell equality and the undrawable
  glyph is a 10-pixel square. C-1 records cell coverage as a choice, the drawn rectangle as its
  rejected alternative.
- **`!in.Left`, `g.sim != nil`, `pkg/game.MapScene`, `sim.MoveTo(id, cell)`.** None is a name here:
  the first is right in substance under another spelling, the rest describe a build this repo does not
  contain.
- **"The surplus stop on the nearest reachable free cells."** False against this tree, where they stop
  where the refusal found them and then give up — and the original's answer is a different mechanism
  again (`MOVE-TERM-003`), so the sentence was neither ours nor the game's.
- **Its "Reference model" framing**, which presented the genre convention as a settled reference. It
  is a choice here, and no sentence of the contract implies the original is its source.

## Appended 2026-08-01 — pin `130bb79`: `MOVE-TICK-014` is now `● active (contested)`

The FR-4 row above cites `MOVE-TICK-014` for what fixes the tick order at map load — the campaign
party first, then the type-6 records in record order. At this pin the registry's new **Open
contradictions** section names that row in conflict **C-2**, and neither side has been picked:

- `MOVE-TICK-014` says the tick list's members are **Humans, Units and Sacks**, sack creation
  included.
- `ITEM-SACK-010` / `ITEM-SACK-011` say a sack is **not** in that list at all — it lives only in
  the per-cell sack registry, takes no walk position, and gets its id from a direct allocator call
  with no `AddTail`.

The clause named in the conflict is **lowered to Medium** until one side is re-derived. What
changes if the other arm wins: the tick population is two classes, not three, so the insertion
history the FR-4 row quotes contains no sacks, and `MOVE-TICK-009`'s contention model has fewer
members in it. **This story leans on the arm that says three**, and does not need to — the clause
it actually uses is *the order is insertion history and an id is not an ordering key*, which both
arms agree on, and this tree has no sack. The divergence FR-4 discloses is unaffected either way.
No side is picked here; the conflict is disclosed and left to research.

Two smaller notes. `MOVE-TERM-003`'s substitution clause is now classed **REFUTED**, and
`verification.md`'s "the original, which substitutes a nearby goal when its own search budget
fails" is right about *that* it substitutes and silent about the two things the refutation
settles — the search substitutes, not the caller, and the substitute is the cell with the smallest
**label**, cheapest to reach from the mover rather than nearest to the request (`MOVE-ALT-018`,
`MOVE-ALT-021`). The *deliberately not consumed* row is unaffected: it is still not consumed.
