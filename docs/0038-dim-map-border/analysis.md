# Analysis — the margin, and what had changed under an older reading of it

## What we did not know

Every map's playable region is smaller than its tile grid: a fixed margin surrounds it that no ALM
record carries. It draws today at full brightness and reads as play area. Three questions stood open
before the contract could be written — how wide the margin is, where a renderer can learn that, and
what the boundary should look like.

## An older reading of this story, and the four ways it had gone stale

This story was scoped once before, and its notes were the starting point. They were re-checked against
the research at this tree's submodule pin and against the tree itself, and four of their load-bearing
statements are no longer true. Each is recorded because the correction, not the conclusion, is what a
later reader needs.

**1. The width was carried as unproven, and it is proven.** The older reading disclosed the margin's
width as taken from a reference rather than from the game, and demoted it to an open research item. At
this pin it is decided: `TERR-PASS-049` publishes the ingest arm that stamps the margin, grades it
**High**, and the grading rests on the depth being a *literal* in the routine plus a corpus count that
matches the closed form for a ring of that depth to the cell. There is no open question here and this
story carries none. The correction runs the other way too — the older note named the outside project it
had taken the number from, and no such name appears in this repo, in this story or in its history.

**2. The width had already landed.** Story 0036 derives the simulation's block plane and owns the depth
as a constant, with the margin arm written as the same predicate the older notes proposed adding. A
second constant would have been two sources of truth for one decoded literal, and they drift. This
story adds none: it reads the plane 0036 already builds.

**3. The symbols it named are gone.** The older notes located the change in a per-tile brightness
source in `pkg/game`, at two named files and line numbers. None of those exist. The single per-tile
brightness source is now `pkg/ui`'s, which is a tier the DAG grants only the render packages — so the
older plan's "put the constants beside the function" is not available at all, and that constraint is
what shaped the design rather than a preference.

**4. Its brightness arithmetic is false here.** The older notes asserted a corner-scale range of roughly
`[0.667, 1.111]` and derived from it that a dimmed margin cell always reads darker than *any* playable
cell. The multiplier this tree applies is `(96 - level)/32`: **3.0 at level 0, 1.0 at level 64, 1/32 at
level 95**. On that range the derivation fails, and it fails on the shipped range too — corpus levels
run `[30..70]`, i.e. scales `[0.8125, 2.0625]`, and a dimmed bright cell at `1.03` is brighter than an
undimmed dark one at `0.8125`. The claim is dropped rather than repaired. What survives is the weaker
statement that is actually true and actually testable: a margin cell is strictly darker **than the same
cell undimmed**, and never black. A global ordering between cells is not available and is not claimed.

## Where a renderer can learn the answer

The depth is owned two tiers below the one that draws. `pkg/ui` may import the render packages and
nothing else — a pinned entry, guarded by its own test — and `pkg/sim` publishes no reader for the
plane it holds, so neither the constant nor the world's copy is reachable from the draw path.

`TERR-PASS-073` is why that turned out not to matter. It re-publishes both block planes' complete
writer lists, names its instrument and closes both of its blind spots, and pins bit 1 as the bit that
stops an air mover — with the margin stamp as its only writer. Our own plane agrees by construction:
0036's builder sets bit 1 if and only if the cell is in the margin. So the predicate is already a byte
the map carries, and it travels to the draw path as data. Nothing at the render tier needs to know how
wide the margin is; it needs to know which bit to look at.

The one thing this does not give us is a read of the *world's* copy: there is no grid reader on it, so
the loader calls the derivation a second time over the same decoded map. That is the same function over
the same input rather than a second rule, which is the property worth having, but it is not the same
bytes and the contract does not say it is.

## The seam

A margin cell and its playable neighbour share an edge, and dimming per tile makes that edge a hard
step. Everything else this renderer draws is interpolated — corner levels blend bilinearly across every
cell — so the step is conspicuous. It was kept anyway, and the older notes had this right: the story
exists to show *where* the playable region stops, and a feathered edge blurs the one line it is drawing.
A per-vertex dim would have removed the step for free, since both quads would evaluate a shared vertex
identically; it was rejected for what it costs to look at, not for what it costs to build.

## Open

Whether `0.5` is the right amount of dim is not settled here and cannot be: it reproduces nothing, so
the only judge is the owner looking at a window. It is one named constant on one line.
