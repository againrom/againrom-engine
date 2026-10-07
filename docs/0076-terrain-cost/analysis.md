# Analysis — the ground carries a cost

## What the milestone measured

The milestone build disclosed the gap as *"no terrain cost plane; roads do not speed anybody up and
slopes cost nothing"*. Two consumers were named as missing it, and only one of them turned out to be
missing anything.

The **rate law** is already implemented whole, and its own file says so: `rateOf` takes the two cells'
cost and height bytes as parameters, composes the multiplier, the saturating slope tilt, the divide by
the mean cost and the clamp — and every call site passes four zeros, because a world here carries
neither plane. The file names the consequence itself: the zero-mean substitute is taken on every call.
So half the disclosed gap is a law with no data, not a law with no implementation.

The **search** is the half that is genuinely wrong rather than merely starved. Its step cost is one
flat constant for every mover. Read against the decoded step-cost arms, that constant is not a
placeholder for the ground rule at all — it is the value the *non-ground* arms charge, applied to
everybody. So the tree does not undercharge ground movers uniformly; it charges them somebody else's
rule.

## What we did not know, and where it turned out to be

**Where does the cost byte come from?** This was the question that decided whether the story could be
built at all: an implementation gap only if the production rule is published, an AMBER if not. It is
published, and completely — the classifier's water early-out, its two reject arms, the strip-group
primary/secondary pairing, the (blend column, sub-cell) level table and the five-arm blend between the
pair's two scalars, plus the ten per-class scalars themselves and the fact that the shipped file's
values, not the executable's compiled defaults, are the ones in play. Nothing in the derivation had to
be invented.

Two smaller things came with it, both of which shaped the contract:

- The **water arm never consults the water scalar** — it returns the classifier's entry literal. On
  shipped data the two are the same number, so this is a distinction no corpus can see and only the
  instruction settles. It is the kind of clause that survives being got wrong.
- **Strip groups 13-15 have a reachable arm with no defined answer.** The pair table is written as
  immediates for groups 0-7 and 12 only; a cell naming 13, 14 or 15 reads memory the original never
  initialised. No shipped cell does it. This is the one place the derivation cannot be faithful,
  because there is nothing to be faithful to.

**Is the height plane already here?** Yes. The map decode already carries the altitude plane whole,
and the ingest's own height arm is a verbatim per-cell copy of it — so the height half of the story is
a plumbing change and no derivation at all.

**Does a placed structure change the cost byte?** This mattered because the tree already applies a
structures pass to the block plane, and if buildings moved cost too, the story would have had to reach
into that pass. They do not: the recompute assigns the cell record's snapshotted terrain baseline back
and then applies occupants and the building to the *block* bytes alone. The only thing that mutates a
cost byte is a set of six cell-record pointers whose writers are unlocated and whose liveness is
published Unknown — so there is nothing here to reproduce, and nothing to guess.

**Does anything else read the cost plane?** One amendment mattered. The mutation the cost getter
performs — quartering a byte in place for a cell carrying a hash record — reaches the **duration read
only**, never a path cost, because the getter has exactly two call sites and both are inside the rate
routine. A reader who took the un-amended text would have wired a mutating read into the search.

## What we looked at and did not use

- **The three block-plane arms.** The block plane's derivation is untouched by this story, and that is
  a decision rather than an omission: nothing narrows a mover's verdict after the domain mask — no
  height term, no corner rule, no order-time or step-time test. A cost story that added a slope rule
  to blocking would be inventing a mechanism the search demonstrably does not have.
- **The per-cell record and its bit-5 quartering.** Located, read, and out: this tree has no cell
  records, and the quartering reaches a consumer this story does not change.
- **`data/map.reg` as a file.** The ten cost scalars are map parameters in the original and a reader
  for that section does not exist in this tree. The story compiles the shipped values in, exactly as
  the rate multiplier — another scalar from the same file's other section — is already compiled in.
  Building the reader is a bigger story than this one and would not change a single byte of output
  today, since the values would be the same values.

## The measurement that shaped the risk

Mission 10 reaches its win at tick 3520 on both installs, with the two waypoints reached at 1250 and
2246. That number is what makes the story falsifiable end to end: the cost plane's dominant value is
the same 8 the rate law already substitutes, so an implementation that quietly failed to wire the
plane through would leave the tick count unmoved and look exactly like an implementation that worked.
The corpus's spread — costs from 6 to 16, with 58.9% at 8 — says the number should move but does not
say by how much, and a story that reported only "it still wins" would be reporting nothing.
