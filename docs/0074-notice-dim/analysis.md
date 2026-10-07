# Analysis — the dim behind a notice

## What we did not know

`0073` shipped the dim behind an open notice and **authored** its strength — black at half opacity —
because nothing published said whether the original darkens what is behind a dialog window, or by
how much. Its provenance says exactly that, twice, in *Ours by choice* and in *Open*.

That is no longer the state of the evidence. So the question this story opens is not "how dark" but
three narrower ones, and only the first is about a number:

1. Can this front-end's compositing tier express what the original does at all, or only approximate
   it? The original's operation is described as a **per-channel gain**, and the tier draws a
   **colour with an alpha**. Those are not obviously the same thing.
2. The original applies its gain **once**. This front-end repaints every frame. What, if anything,
   can observe the difference?
3. The original's gain covers the **whole screen**. `0073`'s dim covers the whole drawable area but
   is composed before three surfaces that keep full brightness. Do those disagree, and does it
   matter?

## What we looked at

### The compositing tier, read rather than assumed

The dim is one `vector.DrawFilledRect` in `Viewer.Draw`. What that does with a colour decides
question 1, so it was read in the module cache rather than inferred from the name:

- `vector/util.go:105` — `DrawFilledRect` is a deprecated alias for `FillRect`.
- `vector/util.go:80` — `FillRect`, on the non-antialiased path, scales a 1x1 **white** sub-image to
  the rectangle and draws it with `op.ColorScale.ScaleWithColor(clr)`.
- `colorscale.go:110` — `ScaleWithColor` divides each of `clr.RGBA()`'s four **premultiplied**
  16-bit components by `0xffff`. For `color.RGBA{0, 0, 0, A}` that is `(0, 0, 0, A/255)`.
- The draw uses the default blend, source-over: `dst' = src + dst*(1 - src.a)`.

With a black source the additive term is identically zero, so the composite reduces to
`dst' = dst * (1 - A/255)` on every channel. **The alpha composite and the per-channel gain are the
same operation here**, not two operations that happen to look alike, and the seam `0073` built
therefore needs no widening to carry the original's law. What it cannot carry is every *value* of
that law: `A` is an 8-bit integer, and `3/16 * 255 = 47.8125`.

That is the whole of the design question, and it was worth reading four files to find out that the
answer is "yes, exactly, except for one rounding".

### The rounding, measured rather than argued

Over all 256 alphas, the gain closest to `13/16 = 0.8125` is `A = 48` at `207/255 = 0.8117647`. The
runners-up are `A = 47` (`0.8156863`, high by 0.0032) and `A = 49` (`0.8078431`, low by 0.0047). The
residual at 48 is `-0.000735`, which over the whole 8-bit channel range is at most **0.19 of one
level** — the largest deviation, at channel value 255, is `207.1875 - 207.0`.

### The other divergence, which is larger and is not this story's

The claim's law runs on the **5- and 6-bit** channels of a 16-bit surface and **truncates**:
`out = (in x 13) >> 4`. At 5 bits the full level `31` maps to `25`, i.e. `0.8065` of full, and the
lowest nonzero level maps to **zero**. This tree composites 8-bit channels through a float blend and
rounds once, so the full level maps to `207/255 = 0.8118` of full. The gap between the original's
own arithmetic and a pure `13/16` is roughly seven times the gap our alpha rounding introduces.

That is a fact about the render tier's colour depth, not about this value, and it is why chasing
exactness in the alpha would be optimising the smaller error.

### The three surfaces

`Viewer.Draw` composes, in order: terrain, both art planes, the overlay passes, the route strokes,
**the dim**, the unit information panel, the debug readout, the notice. `0073`'s ordering test parses
`Draw` and pins the dim between the strokes and the three boxes.

Of the three that stay bright, one **agrees** with the original — the notice itself, because the
original's remap runs before the panel's own draw. One has no counterpart there at all — the debug
readout is this project's instrument. One is a real disagreement, and it is the unit information
panel: in the original it is already in the framebuffer when the remap runs, and darkens with
everything else.

### What the owner's ruling reaches

The ruling `0073` was built on is that there must be a pause and a dimming on any dialog window. It
is a ruling about **whether**, and it does not speak to extent. So the extent stays where `0073` put
it, on `0073`'s own stated reason, and the disagreement is disclosed rather than silently closed —
closing it is the author's call, not a claim's.
