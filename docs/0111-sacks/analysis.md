# Analysis — the sacks on the ground, and why this story stops before the pick-up

Intensity: **spec-first / static**. Terrain: **brownfield** at the cell-anchored depth merge and the
window tier's push; **greenfield** for the sheet load and the placement.

## What we did not know

**Whether the art ships.** It does. `GRAPHICS.RES` carries `backpack/sprites.256` with its
`backpack/spritesb.256` sibling. Decoded through `cmd/sprtool` against both preserved roots:
**six frames of 32x32 each, and the six PNGs are byte-identical between EN and RU.** So the reason
`0103` placed invisible loot — that nothing published what a sack looks like — was ours and not the
corpus's.

**What the six frames are.** Measured, per frame, opaque pixels and the bounding box of the opaque
region:

| frame | 0 | 1 | 2 | 3 | 4 | 5 |
|---|---|---|---|---|---|---|
| opaque px | 70 | 89 | 115 | 155 | 216 | 317 |
| bbox w x h | 9x12 | 10x14 | 12x17 | 14x19 | 16x22 | 20x27 |

Strictly monotone in every measure, all six centred on the same point of the canvas. That excludes an
animation cycle, which would have to return; it is **a size ladder — a small pouch growing into a
bulging bag.** The `b` sibling pairs one frame per frame and carries 13/13/19/19/23/33 opaque pixels
against the base's 70..317, which is the ~12x sparser overlay layer already published, not a shadow.

**What selects among the six.** Nothing in the pin names it. `Sack` is the simulation object and its
value slot is recomputed as gold plus the sum of its items' own value slots, which makes a
value-driven ladder the obvious hypothesis; the drawn object is a different class in a different
family, and no claim reads its frame selection at all. Reading it out of the image is reverse
engineering, which this repo does not do — so the question goes to research and the story authors a
choice in the meantime.

**What the corpus says about the rival hypotheses.** The 28 campaign maps of the EN root, decoded
through `almtool loot`, hold **133 ground records**, of which **117 carry exactly one item and no
gold**. Eight carry gold; only seven carry more than one element. So a ladder driven by *how many
things are in the sack* would draw 88% of every sack in the campaign identically and would never
reach the top two frames — a poor account of why an artist drew six. A ladder driven by *value*
would spread 117 one-item sacks across the whole ladder, and is the better account. **This build
cannot evaluate it**: an item's value needs the definition tables for armour, shields and magic
items, and only weapons are resolved here. Mission 10's four ground sacks are classes 1, 6, 2 and 10
— three of the four cannot be priced at all.

That is the finding that decided the frame rule: the hypothesis with the best evidence behind it is
the one this tree is furthest from being able to implement, and the hypothesis it could implement
cheaply is the one the corpus argues against.

**Mission 10, in full.** Five type-8 records: four ground, at (20,65), (38,64), (12,50) and (10,14),
one item each and no gold; and one stock record at (36,51) with three elements, which belongs to an
actor and is not a sack. Total gold on the map is zero. On a single-player campaign map the authored
records are the only source of sacks — there is no random scatter — so the drawn set is exactly
those four.

## What already exists here

`0103` put the ground records into `pkg/sim` as a per-world sack list, ordered, one per cell, handed
out by copy, serialized, and queried by the script check that asks whether a sack is still there.
Nothing draws one. `0110` built the inventory window and left the pack area empty because an actor
has no container.

The drawing side already has everything a sack needs except a sack. Cell-anchored content —
structures, map objects and units — is merged into one band by row, placed through one anchor
expression, culled once, and tinted by the day/night light row through one texture cache keyed on
frame identity. A fourth cell-anchored list joins that band rather than getting a pass of its own,
and inherits the lighting, the cull and the depth rule by construction rather than by agreement.

## Where the boundary is drawn, and why there

**This story ends with sacks on the ground that he can see. It does not pick them up.** That is not
half of the thing he asked for delivered quietly; it is the honest split, and the reason is that a
pick-up has nowhere to put what it takes. A pick-up is all-or-nothing — the whole sack, every item
and all the gold, in one act — so it needs an actor container, and an actor container is a model of
its own: an unbounded item list, stacks with counts, a running weight and the load's one consequence.
`0110` already named it as the next story and left the window's pack area empty against it. Adding it
here would be two stories in one commit stream, and the first thing it would do is change the
serialized world form.

What each buys him, in order:

**This one.** He walks onto mission 10 and there are four sacks lying on the ground where the map
put them, in front of the things they should be in front of and behind the things they should be
behind, dimming with the night like everything else. Until now the map's loot existed only inside the
simulation, and the only evidence on screen that it was there at all was a mission script silently
answering questions about it.

**The container, next.** An actor can hold items; the pack area of the window he already has stops
being empty; and the map's *stock* records — the ones that are not sacks, 43 of them across the
corpus and one on mission 10 — give every placed person the inventory the map authored for him,
with no command and no pick-up needed.

**The pick-up, after that.** Walking a unit onto a sack takes the whole of it and the sack goes from
the ground. The walk is the point of the first of its two entry points, so it is a movement story as
much as an item one, and it needs the container to exist first.

**Then the frame ladder**, once research names what selects among the six frames — a change to one
function and a widening of one seam field, with no new art and no new plumbing.

## What is worth asking research

What decides which of the six `backpack` frames a sack draws. The value hypothesis is stated above
with the corpus measurement that favours it; the rivals are a fixed frame, the element count, the
first item's class, and six frames of which the engine uses one. A claim naming the selector, and a
claim naming the price column of an armour, shield or magic-item row, would together retire the
authored choice this story ships.
