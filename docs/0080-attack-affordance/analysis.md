# Analysis — a player cannot see that he can fight

The product's author ran the build and reported that nothing on screen says fighting is possible.
What follows is what was read to find out why, and what was set aside.

## The baseline, read rather than taken

`0075` ships a working attack order. It is armed by a key press, spent by the next secondary press,
and it reaches the world through a seam of its own. Nothing about it is broken.

The whole of what states the mode is the debug readout's attack row. **That row is on screen by
default** — `readoutHidden` is stored inverted, so shown is the zero value, and the `F1` binding is
a HIDE rather than a show. This story was opened on the opposite belief, that the readout was off
until a function key turned it on and the row was therefore unreachable in an ordinary run. That
belief is false, and it is recorded here rather than quietly dropped because the answer does not
depend on it: what the row gives is one diagnostic line in a corner box, naming no key, stating a
front-end flag beside a frame rate and a digest. 0075 said as much where it defined the row — it is
"a DIAGNOSTIC statement of front-end state and not a cursor", and what is being reconstructed "shows
the mode by swapping the pointer for its own attack art".

So the report is accurate and its cause is not the readout being hidden. It is that the arming key
is unguessable, the pointer never answers, and nothing marks what a press would hit.

## What the decode says the affordance is

`AI-CLICK-050` settles that a map click becomes an order **by the cursor it was made under** — the
pointer is not a hint about the mode, it *is* the mode. `AI-CURSOR-052` settles which gate produces
the attack cursor at hover, and `AI-KEYMOD-059` reads that gate whole: three globals, each set on
the key-down of one virtual key, cleared on its key-up, and cleared together on focus loss. A
key-held latch by construction, with the claim naming no second model that fits.

`AI-PANEL-060` and `AI-PANEL-061` settle the other arming path, the command panel's: gated on a
selection-derived flags word whose relevant bit is **ownership**, never a unit class. That is the
gate 0075 already implements at its key, and it is unaffected by anything here.

So there are two arming surfaces in what is being reconstructed, they are gated differently, and
0075 built one of them.

## What this tree can already draw

The art is reachable today, which was the open question. `restool list` over either root's
`GRAPHICS.RES` lists `cursors/attack/sprites.16a` — the very path `AI-CURSOR-052` reads out of the
PE for the attack cursor's registration. `sprtool png16a` decodes it to 10 frames of 32x32. Frame 0
is a sword lying corner to corner with its point about three pixels in from the top-left, which is
the shape of a pointer with its hotspot at that corner.

`SPR16A-PIX-011` gives the pixel model at High: palette index from bits 1-8, a four-bit level from
bits 9-12, source multiplier `level+1` against a level-selected term of what is already in the
framebuffer. Read as compositing that is alpha `(level+1)/16` over the destination, which is what a
consumer drawing into an RGBA surface needs and all it needs.

`spr16.DecodeA` and the palette decode already ship. `LoadFont` already joins archive bytes to a
drawing-tier value in the tier that may reach both. Nothing new has to be decoded and no package has
to be added.

## Set aside, and why

- **The swarm cursor and its order.** `AI-CURSOR-052` has the modifier produce the attack cursor
  only where the hover mask carries a unit bit and the **swarm** cursor otherwise; `AI-CLICK-050`
  has that second cursor issue opcode `0x1a`. This tree has no swarm order and building one is a
  different story. The consequence is carried into the contract as a stated divergence rather than
  left to be discovered.
- **The second and third input surfaces.** `AI-MINIMAP-062` and `AI-SURFACE-063` establish that
  three surfaces reach the order builders. This tree has one and gains none here.
- **Retaliation.** `AI-RETAL-056` refutes it as an order; nothing here touches what an unordered or
  a struck unit does.
- **The Shift latch's decoded meaning.** `AI-KEYMOD-059` has vkey `0x10` make a marquee *add* to
  the selection; this tree uses Shift to suppress the selection rectangle in favour of a pan. That
  is a real divergence and it belongs to the marquee, not to the attack path, so it is named here
  and changed nowhere.
- **The cursor's animation.** Ten frames ship and nothing read says at what cadence, or whether the
  cadence is the ambient one this viewer already runs. One frame is drawn and the choice is
  disclosed rather than dressed as a reading.
