# Plan — the dim behind a notice is the one the original applies

## Approach

The previous story built the seam this one exists to use: one colour, handed to the map screen at
construction, replaceable through one setter, read by one decision and drawn by one call. So the
work is not to build anything. It is to establish that the seam's **operation** and the required
**operation** are the same operation — which is a question about the compositing tier and not about
this package — then to pick the nearest value the seam can carry, and then to write down the three
places where "the same operation" stops being true.

One of those three turns out to be settled by the seam already, one is unobservable on its own, and
one is a disagreement this story is not entitled to close. All three are contract-level and land in
`spec.md`; none of them changes a line of code.

## Facts verified during planning

- The dim is drawn by `vector.DrawFilledRect` in `Viewer.Draw`, with the default blend.
- `vector.DrawFilledRect` is an alias for `vector.FillRect`, which on the non-antialiased path scales
  a 1x1 **white** sub-image to the rectangle and draws it under
  `op.ColorScale.ScaleWithColor(clr)`.
- `ColorScale.ScaleWithColor` divides each of `clr.RGBA()`'s four **premultiplied** 16-bit components
  by `0xffff`. For `color.RGBA{0, 0, 0, A}` that is `(0, 0, 0, A/255)`.
- The default blend is source-over, `dst' = src + dst*(1 - src.a)`. With a black source the additive
  term is identically zero, so the result is `dst * (1 - A/255)` per channel — a scaling, exactly,
  with no residual term of any kind.
- `3/16 * 255 = 47.8125`, so no 8-bit alpha spells a factor of 13/16.
- Over all 256 alphas the closest factor to 13/16 is `A = 48` at `207/255 = 0.8117647`; the two
  neighbours are `A = 47` at `0.8156863` and `A = 49` at `0.8078431`.
- The shipped value is `{0, 0, 0, 0x80}` — a factor of `127/255 = 0.498`.
- The value has exactly four sites in the tree: the function that returns it, the constructor field
  it initialises, the setter that replaces it, and the frame decision that reads it. Nothing else
  reads or derives from it, and no test asserts a literal alpha.
- The engine margin's own dim is a separate constant declared in the tree as ours, reproducing
  nothing.

## Design decisions

**DD-1 — nothing structural changes, because an alpha composite over a black source already IS a
per-channel scaling.** The two descriptions name one operation, not two, and the tier expresses it
with no error term. So the story is a value and its justification, and the seam, the decision, the
draw call, the rectangle and the composition order are all left alone (FR-1, FR-2, FR-3).

Rejected: **compose the map into an offscreen target and re-draw it through a float colour scale.**
It is the only way to apply a factor at better than 8-bit precision, and it would add a render target
to the map arm, a second upload per frame, and a second answer to "where is the map picture" — to buy
precision the drawn surface cannot hold. The measurement that decides it is in *Facts* above: the
residual it removes is 0.19 of one channel step.

**DD-2 — the shipped value is the nearest expressible one, and "nearest" is what is written down.**
`A = 0x30`. It appears in the source as a literal, because a derivation would have to round and the
rounding rule would then be the thing nobody checked; but it is pinned by **searching all 256 alphas
at test time** and requiring the search to return it, together with a bound on the residual. That
assertion keeps its meaning if the factor changes, if the alpha's precision changes, or if someone
edits the literal (FR-2).

Rejected: **assert the literal `0x30` in the test.** It would witness that the number did not change,
which is not the requirement. The requirement is that no other value is closer, and that is the
sentence a reader of the test needs to be able to check.

**DD-3 — the colour stays black and the type stays `color.RGBA`.** Black is what makes the composite
a scaling rather than a wash — any non-zero channel adds light and moves the map's hue — and it is
also what makes one colour sufficient to carry a factor at all. The type is the previous story's
FR-8, unchanged (FR-1, FR-3).

Rejected: **`color.RGBA64` or `color.Color` at the seam.** A 16-bit alpha would put the factor within
5e-6 of 13/16, and would change the type of one exported function, one exported method, one struct
field and every test that supplies a value — for precision two orders below the destination's own
step.

**DD-4 — the function keeps its name, and its comment carries the split.** `AuthoredNoticeBackdrop`
stays. The `Authored...` prefix in this package already means *the value this project ships at this
substitution point*, not *invented*: `AuthoredDialogueLayout` carries it while its geometry is taken
from research and only its paint is ours. Renaming a seam to record where one of its numbers came
from would put provenance in an identifier, in a tree that has a file for it. What the comment must
do instead is say which half is reproduced and which half is our arithmetic, and name the residual —
because a reader who sees `0x30` beside `13/16` and no residual will read the value as exact
(FR-2, FR-3).

**DD-5 — the new assertions go in a file of their own beside the previous story's.** The previous
story's backdrop tests are about the frame's **decision** — whether there is a dim, over what
rectangle, in what order, and when there is none. These are about the **value**: what it is and why
it is that and not something else. Two functions, two files, and the older file's header keeps saying
what that file asserts (FR-1, FR-2).

Rejected: **appending to the existing file.** It reads as one concern until the header has to
describe both, and then it describes neither.

## Risks

- **R-1 — the residual is read as exactness.** "13/16" is a clean ratio and `0x30` looks like a
  chosen number; a later reader can easily carry "the dim is exactly the original's" into a place
  where it is load-bearing. Answered by DD-4's comment and by DD-2's bound being a criterion rather
  than a remark.

- **R-2 — the dim is now much weaker, and weaker is a product judgement a number cannot make.** The
  map behind a notice goes from 50% of its brightness to 81%, which is a large visible change in the
  direction of *less* separation between the box and the map behind it. The box's own legibility does
  not depend on the backdrop — it is drawn on a near-opaque fill inside a border — but "the world
  reads as stopped" is part of what the dim is for, and only an eye can say whether 81% still does
  that. It is **not** answered inside this story: the look is the product author's to judge against a
  lawful install, which is what SC-5 is, and what this story is entitled to say meanwhile is that the
  strength is the measured one and that changing it costs one value.

- **R-3 — the value stops being nearest without anything failing.** If a later story changes the
  precision the seam carries, or the factor is amended by research, a literal would go on passing.
  Answered by DD-2: the search is the assertion, so the first of those fails loudly and the second is
  a one-line edit to a named constant in the test.

## Success criteria

- **SC-1** — the shipped value's three colour channels are zero and its alpha is strictly between
  fully transparent and fully opaque.
- **SC-2** — a search over every alpha the type can carry returns the shipped one as the closest to
  13/16, and the shipped factor is within a quarter of one 8-bit step of 13/16 across the whole
  channel range.
- **SC-3** — the previous story's whole backdrop suite passes unchanged, including its parse of the
  composition order, so the rectangle, the gate and the position among the composed statements are
  witnessed as unmoved.
- **SC-4** — the full local gate is clean, and the change touches no file under `pkg/sim`, no
  byte-form record and no serialized version.
- **SC-5** — the dim is looked at: a campaign mission played to a dialog window against a lawful
  install, and a judgement that the map behind it reads as stopped at this strength. It is listed as
  a criterion precisely because nothing here can run it, so the gap is visible where the met ones are
  rather than buried in prose.

## Traceability

| FR | design | criterion |
|---|---|---|
| FR-1 | DD-1, DD-3, DD-5 | SC-1 |
| FR-2 | DD-1, DD-2, DD-4, DD-5 | SC-2, SC-5 |
| FR-3 | DD-1, DD-3, DD-4 | SC-3, SC-4 |
