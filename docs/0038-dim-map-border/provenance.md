# Provenance — 0038, dim the non-playable map border

Submodule pin for this story: research `f35be34`. Every claim below was read at that revision.

## What the game decides

| Fact | Claim | Confidence | How it is used here |
|---|---|---|---|
| Every map carries a fixed margin around its playable region, stamped by the map→sim ingest and not by any ALM record | `TERR-PASS-049` | High | The premise of the story: there is a region to dim, and it is not map data |
| The margin is **8 cells** deep on all four sides | `TERR-PASS-049` | High | The width. Owned by 0036's constant; this story adds no second one |
| A map narrower than twice the depth on an axis is margin throughout | `TERR-PASS-049` | High | AC-9's shape. The two rings meet and there is no playable region |
| Bit 1 of a block byte stops an air mover, and the margin stamp is its only writer | `TERR-PASS-073` | High | FR-1: the bit the render tier reads to answer "is this cell margin" |
| The terrain shading multiplier is `(96 − level)/32` — 3.0 at level 0, 1.0 at level 64, 1/32 at level 95 | `TERR-LIGHT-018`, `TERR-LIGHT-019`, `TERR-LIGHT-020` | High | The base the dim multiplies, and the reason no cross-cell brightness ordering is claimed |
| Shipped corpus levels run `[30..70]` at the measured sun angle | `TERR-LIGHT-063` | High | Bounds the practical scale range at `[0.8125, 2.0625]`, which is what refutes the ordering claim |

`TERR-PASS-049`'s High is graded on two independent things, and both matter to us: the depth is a
**literal** in the routine that stamps it, and the shipped count of stamped cells equals the closed form
for a ring of that depth **to the cell** over 38 maps and 880 704 cells. AC-8 asserts that same closed
form over our own derivation, so the criterion and the claim are checkable against each other.

`TERR-PASS-073` carries an **Unknown** beside its High: the origin of bits 3 and 4 outside the margin
literal, and whether the dynamic plane's bit 5 survives a freed record. Neither reaches this story —
bit 1 is the only bit read, and its writer list is the High half.

## What is ours by choice

| Decision | Basis |
|---|---|
| The dim factor `0.5` | Ours. It reproduces nothing and no claim bears on it. A developer or the owner looking at a window is the arbiter, and it is one named constant on one line |
| That the boundary is a **hard step** and not a ramp | Ours (DD-3). Nothing decoded says what the original does at this edge; the choice is made on what the picture is for |
| That the dim **multiplies** the corner lighting rather than replacing it | Ours (DD-4). It follows from wanting the margin's own relief to stay readable, not from a claim |
| That a placeholder fill stays undimmed | Ours, and inherited: 0014 AC-7 already pins that fill at full brightness as a diagnostic |
| That the plane travels as a `Grid` layer | Ours (DD-1), forced by the import DAG rather than chosen freely |

## What was removed, and why

An older scoping of this story disclosed the margin width as **reference-derived and not game-proven**,
carried it as an open research item, and named the outside project it had come from. All of that is
removed rather than carried forward:

- the width is decided at this pin (`TERR-PASS-049`, High), so the disclosure was **false as of this
  tree** and an open item would have been an invented question;
- the width had already landed in 0036, so importing the constant would have created the second source
  of truth the disclosure was worried about;
- the outside project's name is not in this repo, this story, or its history, and is not recorded here
  either — naming it in a provenance ledger would put it in exactly the place this rule exists to keep
  it out of.

The same older scoping asserted a corner-scale range of `[0.667, 1.111]` and derived from it that a
dimmed margin cell always reads darker than *any* playable cell. Both are refused. The tree's actual
range is `[1/32, 3.0]` (`TERR-LIGHT-019`/`020`), and on it — and on the shipped `[0.8125, 2.0625]` too —
the derivation is arithmetically false. What this story claims instead is the per-cell statement its
own tests can witness: strictly darker than the same cell undimmed, and never zero (FR-4, AC-3).

## Open

Nothing about the margin is open. What is unsettled is the dim's *value*, which is a look-at-it
question rather than a research one, and is recorded as such in `verification.md` rather than raised
against research.

## Divergence

The margin is dimmed and is **already impassable**, which is worth stating because it is easy to assume
the opposite of a cosmetic story: 0036's derivation counts the margin among the arms that block a ground
mover, so a unit cannot be ordered into it and this story changes nothing about that either way.

Where we do diverge is in the byte. The game's ingest stamps a margin cell `0x1f` — bits 0 to 4 — while
our plane sets bits 0 and 1 only. Bits 3 and 4 are `TERR-PASS-073`'s own **Unknown**: they arrive inside
the margin literal and have no located origin outside it, so there is nothing to reproduce yet, and the
simulation refuses bits 2 to 7 outright rather than carrying values it cannot explain. The consequence
for this story is nil — bit 1 is what it reads, and bit 1 is decided — but a later story that needs the
margin's other bits will find them absent by decision rather than by oversight.
