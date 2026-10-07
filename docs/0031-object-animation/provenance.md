# Provenance — the object cycle, its counter and the gate

Pinned at research `8c92427` (`8c92427e06f2d5432f5a50aec9f4bd7e099939c9`), frozen for the story.
`claims/retracted.md` read first: nothing cited below is overturned, and `TERR-ANIM-009`'s
amendment is the clause this story consumes.

**Threshold: Medium.** Nothing this story adds reaches hashed simulation state: `ANIM-CLOCK-001`
puts the frame outside the simulation's state and inside its clock, and `ANIM-OBJ-008` gives the
object arm no per-instance state at all, so the drawn frame is a pure function of a front-end
counter and a cell, held nowhere and never serialized, and `pkg/sim` is neither imported nor
changed.

## Backing

| `spec.md` anchor | Claim | Confidence as published |
|---|---|---|
| The frame is presentation state stepped by the game's own tick, outside the simulation (FR-8; P-2) | `ANIM-CLOCK-001` | High for the field census, the caller chain and the tick's identity / **Medium** that no `memcpy`-shaped write reaches those fields |
| Objects animate off that same counter by a different mechanism, holding no per-instance state — a pure function of `(counter, cell)` (FR-5, FR-6; P-1) | `ANIM-OBJ-008` | High |
| The frame arms and their gates; the phase `counter + col*(row+1)` reduced by the timeline's length, the counter **unshifted**; the disabled flag forcing frame 0 (FR-2, FR-3, FR-4) | `TERR-SPR-042` | High — each arm, each gate and the modulus a named instruction in a function read end to end, the slot's writer set exhaustive / Medium for the corpus ranges / **Unknown** whether the animated arm is ever entered |
| The timeline is **built, not stored** — `AnimationFrame[i]` appended `AnimationTime[i]` times, a non-positive time contributing nothing, both heads dropped per round, ending when either track empties — and its length is the modulus (FR-1) | `REG-OBJ-046` | High for the expansion loop, reproduced instruction by instruction / Medium for the lengths over the one registry: 0 x52, 24 x2, 28 x21, 105 x7 across 82 classes |
| The gate's second half — the cell's four corner tile words ORing to `0xc000` in bits 15..14 — and **0 of 880 704** shipped cells setting either bit (FR-4; P-5) | `TERR-TILE-044` | High for the gate and the exhaustive census, the reader enumeration re-run on a repaired function table / **Unknown** what writes those bits |
| One flag disables animation, and it forces a static object's frame to 0 as well as water's phase (FR-2's disabled arm) | `TERR-ANIM-009` (amended) | High |
| The counter the object arm reads is the one the water phase reads (FR-5) | `ANIM-OBJ-008`, `TERR-SEM-004` | High for the shared increment — both jobs in one pass of one function |
| An animated frame is lit by the one sprite rule, through its own sheet's palette (FR-7) | `TERR-LIGHT-064`, `TERR-SPR-065` | High |

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| **The diagnostic that opens the gate** (FR-8) | Ours entirely: the engine has no such switch, and any writer of bits 15..14 is Unknown (`TERR-TILE-044`). Opening them ourselves asserts nothing about the original — it makes an arm we implemented drawable |
| The gate **closed** on the last column and row (FR-4) | A **divergence**: the engine reads `idx+1`, `idx+W`, `idx+W+1` unguarded, so at the far edge it reads past the grid and there is no defined behaviour to reproduce |
| An out-of-range selection draws **sheet frame 0** (FR-2) | Unmeasured in the sense that matters: `TERR-SPR-042`'s census (Medium, one registry) puts `Index` and every `Index + T[k]` inside the sheet on all 75 art-bearing classes, so nothing shipped reaches the guard. Frame 0 is this tree's own selector answer and the frame the engine substitutes when animations are off |
| **Euclidean** reduction of the phase (FR-3) | A **divergence** where the sums differ: the engine's is a signed `IDIV` remainder. The two agree on every non-negative sum, which is every sum a `uint32` counter and non-negative cells produce |
| Objects keep cycling **while the world is stopped** (FR-5) | A **divergence** from `ANIM-CLOCK-001`, whose counter rises inside the tick a stop halts. 0041 exempted that counter for water and its rate setter takes no stop; one counter for two consumers is what buys the exemption |
| The raster tool renders at a **chosen counter value** (FR-8) | Ours: the tool draws one still and the engine has no such selection |
| The animated placement's rectangle is its **drawn frame's own** (FR-6) | Engineering; no engine fact is claimed about culling. The anchor arithmetic it feeds is `TERR-SPR-040`'s, unchanged |

## Open / undecoded

- **Whether the animated arm is ever entered at runtime.** `TERR-SPR-042` and `TERR-TILE-044` both
  publish it Unknown, and research's own next-question list names it as what stands between us and
  knowing whether ROM1's trees animate at all. This story assigns it no answer: under the decoded
  gate our renderer draws every shipped map exactly as it draws it today (P-5).
- **The dead-object arm**, blocked on us and not on research, which is why a decoded arm is out of
  scope here: `REG-OBJ-047` measures `DeadObject` as -1 on 54 classes, `pkg/data` yields -1 on 21
  and **0** on 33, and 0 is the valid id of `Object0` (0017's own open item).
- **Whether object animation slows with the game speed and stops with the tick**, as
  `ANIM-CLOCK-001` predicts from the caller chain alone. No runtime session exists; two stopwatch
  observations are named there as the witnesses.

## Removed from the baseline and why

- **The `Provenance basis.` preamble and the `## Research needed` section** — refused by
  `check-doc-budget.sh` check 0; what they said is this file.
- **`R-1`, "what the paired arrays mean at draw time is unproven"** — answered at High since it was
  written: `TERR-SPR-042` reads the arm, `REG-OBJ-046` the loader. 0017 dropped its own two research
  items on the same grounds, and this story carries none.
- **"Fire forms, the burning variant referenced by `FireObject`, a `Phases = 17` cycle"** —
  `FireObject` is never a subscript, never arithmetic, never stored; its value space is
  `{-2 x21, -1 x61}` and its one reader image-wide is an ambient-sound ratio (`REG-OBJ-047`, High).
  It references no frame, and the fire-variant classes ship no art (`ALM-CLS-042`).
- **The stagger `x*13 + y*7`** — an invention standing beside a decoded value.
- **"the cull is sized by the class's maximum cycle-frame extent"** — unnecessary once the placement
  is re-anchored with the frame: the rectangle is exact at every counter.
- **"a bad frame falls back to `Index`, diverging from the nil-sentinel cache-miss idiom"** — no such
  idiom exists here, and the fallback taken is frame 0.
- **"Classes without a table stay static, byte-for-byte the current output"** — false with
  animations off, where every object draws frame 0 (`TERR-SPR-042`(d)).
- **Six named symbols and two document citations** that do not exist in this tree — `analysis.md`
  names each and what stands in its place.

## Appended 2026-08-01 — pin `130bb79`: `TERR-SPR-042`'s census denominator moved, and this story already carries the corrected one

`retracted.md` classes the row's figure **SUPERSEDED**: "0/82 out of the sprite's own frame count"
should read *over the **75** classes whose sprite ships* — the other seven have no frame count to
be out of. The result is unchanged and the figure simply read as a stronger negative than it is.

The *Ours by choice* row above already states the census as "all **75** art-bearing classes", so
nothing here moves, and the tree measured the complement independently: `0017`'s `verification.md`
records **7 of 82** loaded classes carrying no drawable frame, which is the same seven. Recorded
because the correction is the kind that arrives silently — a denominator, not a claim — and the
next reader should be able to see it was checked rather than missed.

## Appended 2026-08-02 — pin `01c64e2`: the writer of bits 15..14 is found, this story's first open item is half-answered, and FR-8's label falls

- **`TERR-TILE-044` — `● active (amended)`; its writer dichotomy REFUTED.** The row held that no
  instruction ORs an immediate `0x4000`/`0x8000` into memory, so the writer was "a non-immediate
  store, or the path is dead". Both horns were wrong: it is an **immediate store of the wrong
  width**. `R0554` (`CUnit`/`CAirUnit` `vt+0x48`) executes `OR byte ptr [...],0xc0` at
  `L02581`, `L02582`, `L02583`, `L02584` — **the gate's own four corners** — over a 41×41
  window around every drawable, and `R0374` clears bit 14 map-wide every 32 ticks
  (`ANIM-TICK-011`). A sweep for 16-bit operands could not see a byte store on the word's high half.
  **The census is untouched**: 0 of 880 704 is still exactly true *of the shipped file*, the bits
  being runtime state. The *Backing* row above is cited for the gate's shape and that census, and
  both stand; only its `**Unknown** what writes those bits` is now dated.
- **`ANIM-OBJ-008` — `● active (amended)`; its unreachability clause REFUTED as an assertion about
  play.** "On a shipped map the object arm is unreachable anyway" is **withdrawn to unestablished**,
  not to *reachable*: the stamp's two guards (`[[mapView+0x9b4]+0x38][player·2] & 8` and
  `unit+0x102 != 0`) and its `CMapView+0x17cc` mask are unread, and no census of the runtime grid
  exists. The row's High half — the shared increment and the object arm's statelessness — is what
  FR-5, FR-6 and P-1 cite, and is unchanged.

**The first *Open* item splits, and half of it closes.** "Whether the animated arm is ever entered
at runtime" was one question and is two: *what opens the gate* is answered — a per-drawable runtime
stamp — and *whether it is satisfied in play* is unestablished. This story assigned it no answer and
still assigns none. P-5 is a regression invariant over our own renderer and holds unchanged.

**FR-8's diagnostic is no longer clean *ours by choice*.** Its row reads "Ours entirely: the engine
has no such switch, and any writer of bits 15..14 is Unknown". The second half is now false: the
engine has a writer and it opens exactly these bits. What survives is only that the engine has no
*switch* — ours is a blanket unconditional open where the engine's is per-drawable, bounded to a
41×41 window, and cleared every 32 ticks. So the affordance is a **disclosed divergence standing in
for a mechanism the engine has**, not one built for an arm the original never enters; and with the
diagnostic **off** this renderer opens no cycle anywhere, where the engine opens cycles around its
drawables. Recorded, not redesigned: implementing the stamp, or explicitly declining it, is a story,
and it is reported as owed rather than taken here.

`analysis.md` states the gate has "no writer of those bits anywhere in the image", in "a renderer
whose research says the original draws them still". The first is refuted outright; the second was
never quite what research said, and is now certainly not. Both are left as the story's own record,
with this note governing.

`pkg/render/terrain/statics.go`'s `animGateBits` comment carries the same refuted sentence — "No
writer of them is known". A comment is a code change and belongs to whichever story next owns that
line; it is reported rather than edited here.
## Appended 2026-08-02 — pin `9ff259c`: the arm fires, and the bits are the fog

**`TERR-TILE-044`'s reachability Unknown is CLOSED to High, and `ANIM-OBJ-008`'s withdrawn clause
resolves to *the arm fires*.** The guards are three, not two; `CMapView+0x17cc` is a per-drawable
line-of-sight field, not a shape; and the stamp's `0xc0` puts **both** bits in one word, so one
stamped cell passes it. Bits 15..14 are the **fog of war** and the arm runs where the local player
can see (`TERR-TILE-079`). Both censuses are of **file** bytes and stay true.

**FR-8's diagnostic stands in for a mechanism now named.** With it off this renderer draws every map
as the engine draws an unexplored one. The fog is a story, not taken here. `TERR-LIGHT-064` is
amended on its `> 1` arm; the clause FR-7 cites is untouched (`0044` carries it).
