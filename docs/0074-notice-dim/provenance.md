# Provenance — the dim behind a notice

## Backing

| spec anchor | source | confidence |
|---|---|---|
| FR-2 — the picture behind a notice is multiplied per channel by 13/16 | `DLG-DIM-013` | **High for the call and its five arguments** — each a named instruction in two routines read whole, with the instrument (`callto:R0732`, 11 hits / 11 owners / 0 orphan) and its blind spot stated. The *numeric* half is graded no stronger than the law it applies |
| FR-2 — the law behind the level: `out = (in x (16 - L)) >> 4`, so L = 3 is a gain of 13/16 | `TERR-FOG-084` | **High** — the table's row count and fill are cited instructions, and the law is checked against two independent table-free implementations in the same binary over the complete 16-bit input space, 65536/65536 in RGB565 and RGB555. `DLG-DIM-013` re-runs those two anchors before applying it, so the law being applied is visibly the pinned one |
| FR-1 — the operation is a multiplication and adds no light | `TERR-FOG-084` for the law's form; `DLG-DIM-013`'s re-execution at L = 3 for the result — 65 535 of 65 536 pixels strictly darker, 1 (black) unchanged, **0 brightened** | High |
| FR-1, FR-2 — that an alpha composite over a black source *is* that multiplication, so one colour can carry the law | ours, read out of the compositing tier (`analysis.md`) | High — the four sites are named and the arithmetic is closed-form, not sampled |

The gain is the only number this story takes from research. Everything else it fixes is either
`0073`'s, unchanged, or the arithmetic of our own tier.

## Ours by choice

| what the spec fixes | why it is ours |
|---|---|
| That the value shipped is the **nearest** the seam can express rather than the claim's exact ratio | `3/16 x 255 = 47.8125` is not an 8-bit alpha. Nothing decoded bears on which side to round to or how close is close enough; the spec fixes "nearest, and within a stated bound", and the bound is ours |
| That the seam keeps its type — one `color.RGBA`, its transparency carrying its strength | `0073` FR-8, unchanged. A wider type would buy precision below what an 8-bit destination can hold |
| That the dim is re-applied every frame rather than once | Forced by this front-end's own draw loop, which has no locked framebuffer to write into and no gate that stops repainting. It reproduces the original's *appearance*, not its mechanism, and the spec says so |
| That the unit information panel and the debug readout keep full brightness | `0073`'s choice, unchanged and now known to disagree with the original for one of the two. Kept on `0073`'s own reason — a front-end instrument must stay readable exactly when it is being read — which is a product reason and is not contradicted by a fact about the original |
| That the engine margin's own dim stays at half | It is a different value with a different status: `mapBorderDim` is declared in the tree as ours and reproducing nothing. `0073` argued for one answer to "how dark is dim" when both were authored; that argument does not survive one of them acquiring a source |

## Open — deliberately assigned no meaning

| what | why it is left open |
|---|---|
| What the original's *pixels* are, as opposed to its law | `DLG-DIM-013` gives the law and re-executes it over the 16-bit pixel space; reproducing the pixels would need this tree to composite at the original's colour depth with the original's truncation. Nothing here asserts our pixels are the original's, and the divergence is disclosed rather than measured against a target nobody has set |
| Whether the whole screen *should* darken in this product | `DLG-DIM-013` establishes that it does in the original — the extent is a property of the argument list, not an absence of observation. What we ship is the author's call, and the ruling this behaviour rests on reaches whether there is a dim, not what it covers. Recorded as a question for the author, not decided here |
| Whether anything can observe our per-frame application | The one surface that changes behind an open box in this tree is the water, and `0073` already discloses that its suspension does not stop it. So the mechanism difference has no consequence of its own; whether a player *notices* the dimmed moving water is `0073`'s divergence to close, not this one's |

## Removed

Two statements in shipped code are contradicted rather than dropped, and both read as decided design
points, which is why they are named here instead of being quietly overwritten:

- `AuthoredNoticeBackdrop`'s "nothing published says whether the original darkens its backdrop or by
  how much, and this asserts nothing about it". It was true when it was written and `DLG-DIM-013`
  ends it.
- The same comment's "half, because half is what this tree already means by dimmed ... a second,
  different answer to *how dark is dim* is two numbers to keep in step for no gain". The two numbers
  now have different statuses, which is the gain.

`0073`'s `spec.md` is **not** amended. Its FR-6 requires a uniform darkening and its FR-8 makes the
strength a supplied value; both are satisfied by this story exactly as written, and its own
*Out of scope* anticipated in terms that a later answer "is expected to change the dim's value ...
neither of which is a change to anything in this contract but FR-6's strength". This story is that
change, and it supersedes `0073`'s provenance rows for the dim's strength rather than its contract.
