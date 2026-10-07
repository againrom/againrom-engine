# 1015 — provenance

Where each fact in `spec.md` comes from. `spec.md` itself is self-contained and cites nothing.

## Claims

| Claim | Confidence | What it supplied | Where it is used |
|---|---|---|---|
| `TOWN-061` | High | The school's paint routine `R1487` and its eight-step order. Step 4 draws a state-indexed decorative sprite chosen from `+0x30c` by the active-panel selector `+0x31c`; step 5 draws the active panel's five skill icons | FR-1 is step 4 and FR-2/FR-3 are step 5. That the face is selected by the class selector rather than baked is this claim's, not a measurement |
| `TOWN-067` | High | `+0x31c==0xf` is the mage panel and `==0` the fighter panel, closed by an exact dual-root match of each panel rect to its class mask's dimensions. The two rects: mage `(188,188,288,308)`, fighter `(192,192,284,312)` | FR-2's panel rectangles and FR-4's mask placement |
| `TOWN-017` | High | The school's own top-level click dispatch: a `PtInRect` on the client rect, then `+0x31c` selects one of two independent `{rect, mask-surface}` pairs, each read as cursor position to mask byte to a five-arm switch | FR-4's whole shape |
| `TOWN-018`, `TOWN-019` | High | The three art states per skill per class, and the per-slot enabled flag | FR-3's state selection and the enabled gate in FR-4 |
| `TOWN-068` | High for geometry | The five fighter icon rectangles, at object offset `+0x1c0`..`+0x20c`, all 80 pixels wide | FR-2's fighter row, in `DIV-121`'s order. Its shared-array clause is contradicted by this story's measurement and is recorded in `DIV-145`, not built |
| `TOWN-GENERAL-106` | High | Visual indices 0..4 map to stored slots `1,2,4,3,5` | Not built. It is what identifies the FR-4 defect as an inconsistency rather than a choice: the hit test was reproducing this permutation while the draw side had been re-ordered under `DIV-121` |
| `TOWN-GENERAL-107`, `TOWN-GENERAL-108` | High | The school view's selected-slot and price fields, `view+0x34c` and `view+0x354` | The two fields FR-5 clears |
| `TOWN-138` | High | The school's own picker-step routine, read end to end, and that no routine is shared between the shop, the tavern and the school | FR-5, including its room gate: three routines there, one here |

## Measured rather than cited

Everything below is a property of the shipped art, printed by `cmd/schoolcheck` on both preserved
roots, identical on the two. No claim states any of it.

- The sixteen rotation frames and their size. `column/rt0000.bmp`..`rt0015.bmp`, 148x208 each.
- The face origin (168,176), and that `rt0015` is the frame `trnhall.bmp` bakes: 0.9795 agreement
  at that offset against 0.1900 for the best offset anywhere else, over a search of every offset at
  which the frame fits inside the 480x480 background. `rt0000`'s own best offset is the same point
  at 0.3426, which is the other face of the same cylinder and is why it does not match the baked
  one. No other frame exceeds 0.3345 at the origin.
- The class of each rest face. Each class's five patches are correlated against all sixteen frames
  and the winner must be the frame the build names: the fighter's five win on frame 0 and the mage's
  five on frame 15, at 0.4792..0.7438 against a runner-up frame of 0.1514..0.2459. The measurement
  was correlation against the assumed frame alone until the story's landing, which could not falsify
  the assignment; `cmd/schoolcheck`'s `frame-of` lines now perform the cross assignment and the tool
  exits 1 when `restFrame` is reversed. At the time this was measured no claim named the class of a
  rotation frame and the two claims giving the panel selector's polarity disagreed. `TOWN-146` and
  `TOWN-149`, published by `EXP-0195` and pinned at the landing, name both and agree; `DIV-146` is
  closed.
- The ten patch rectangles of FR-2, each from all three of its lit states agreeing on one offset.
- The mask-code-to-skill correspondence of FR-4, from each code's bounding box in the class mask
  placed at the class panel rectangle and matched to the patch rectangle it overlaps most. The ten
  overlap shares are 0.9253 or better.
- The pure-black populations of FR-3. `fighter/sword` carries 666, 693 and 618 pure-black pixels
  in its three states, `fighter/axe` 174, 74 and 104, and the other eight skills none.

## Divergences this story wrote

`DIV-142` rotation frames unconsumed, `DIV-143` the five reset fields with no counterpart,
`DIV-144` the pure-black key, `DIV-145` two rectangle arrays against `TOWN-068`'s one, `DIV-146`
the class-to-frame assignment and the `TOWN-061`/`TOWN-067` disagreement. `DIV-121` is amended.

## Concurrency

`EXP-0195` is open in the research repo and asks what the original's paint routine does for steps 4
and 5, per class, at instruction level. It was told nothing this story measured. Its answer is the
authority over every rectangle here, and `DIV-145` names it as the row's revisit condition.
