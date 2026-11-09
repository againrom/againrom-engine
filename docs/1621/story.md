# Story 1621: control rows for draw and hit, one tip placement

## Intent

A control's rectangle is written once, as a row of `controlLayout`, and the draw and the hit test both read that row. A tip's position in the page comes from one function. No frame, hit area, text, key, press behaviour, SAV byte, RNG step or World hash changes.

## Authority

Owner direction on the architecture audit: "Берем те 7 видов, пытаемся достигнуть 13". Kind R28 of `pipeline/reviews/arch-audit-5d474c65.md`, row 13: control rows for draw and hit, plus one tip-placement tail. Architecture story: no DIV row, no player-visible change.

## As-built behaviour

Tip placement: `placeTip` in `pkg/ui/widgethover.go`, described by `tipPlace`. `placeHover` (the hover box, above and beside the pointer) and the three branches of `popupTipAt` (portrait tip, row tip, grid column head) take their position from it. `internal/archtest/controllayout.go` lists both as `tipPlacers`: each must call `placeTip` and read no `bounds.Max` of its own.

Control rows, moved to `controlLayout`:

- Tip panel: `tipPanelLayout(rect)`, read by `ComposeTipPanel`, `TipPanelControlAt`, `TipPanelEventAt`.
- Drop gold dialog: `Viewer.goldModalLayout()`, read by `goldModalPresent` and `goldControlAt`.
- Documents panel: `docControls`, read by `composeDocumentsPanel` and `documentControlAt`.
- Character creation pre-create page: `preControlLayout(c)`, built from layout and art; read by `preControlAt` and the sparkle. The switch `preControlRect` is gone from `controlSwitchProducers`; `preControlRect` is a row read kept for its callers. The page's hero and level pictures still blit at their art positions per frame: the row is the hit extent of frame 0, as before.

Guard: the four surfaces join `controlLayoutSurface`. `controlHitFuncsRemaining` ratchets the functions in `pkg/ui` and `pkg/game` that hold a hit test outside the surfaces (162 at this candidate); it may only fall.

## Proof

Census: `review/story1621-control-rows/census-before.txt`, `census-after.txt`, instrument `census.awk` and `census.sh` there. A search of every simple `image.Rect` text in non-test `pkg/ui` found no control rectangle written twice.

Witness: `pkg/ui/controlrows_witness_test.go` with `pkg/ui/testdata/control_rows.golden` (numbers only), recorded at the base before any production change: rectangles and hit answers on a 2-pixel grid of 640x480 for the four surfaces, and tip placements at the page edges and at bounds cut inside the popup. The golden was recorded from a base-tree export (production of `833f2051c`, the witness file copied in) and is byte equal after the moves. `TestControlRowsWitnessFailsAPlantedFault` fails on a changed number. Planted production faults, each run once and reverted, fail it: a one-pixel tip panel control, and in `placeTip` the flip distance, the clip and the push.

Hit grids of the character creation page cover the mask-less path of `preControlAt`; the mask path reads the mask, not rows.

## Open debt

Kind R28 stays open: 162 functions with hit tests remain outside the guarded surfaces. Many read one layout value or one accessor for both draw and hit (`popupLayout`, `ShopLayout`, `noticeLayout`, the widget values), which this story counts as one source. Not moved, by hit sites: inventory, map editor chrome, tooltip targets, world map cards, mod screens, ending, game options, sound options, game menu rows, detailed character page. Inventory was skipped out of hit-count order because the story's budget ended first; its rectangles are one accessor read by both sides.
