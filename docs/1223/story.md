# Smoothed text overlay (owner decision method G)

## Result

Drawn text can now render softened at window resolution instead of the
original's hard-edged bitmap font, on a stored preference that defaults on
and reproduces today's pixel-exact draw byte for byte when turned off. The
owner picked method G after reviewing renders of nine candidate methods
(disposable, untracked `review/font-smoothing/` and `review/font-smoothing-2/`).
`text.Font.Draw`'s own 74 production call sites are unedited; every glyph
still paints the native 4-bit level ramp exactly as before. A new
presentation-layer pass composites a smoothed copy on top, in window space,
after the already-upscaled frame is drawn.

## Mechanism

`pkg/render/text/capture.go` adds a package-level recorder: `SetCapture`
turns on recording every `Font.Draw` call (`DrawCall{Glyph, X, Y, Color}`)
tree-wide, with an option to also suppress the raster itself, at zero cost
when off. `pkg/render/textsmooth.Composite` resizes each captured glyph's
4-bit coverage with a Mitchell-Netravali (B=1/3, C=1/3) bicubic kernel to
window resolution, remaps the resized coverage with a contrast gain of 1.8
around 0.5 to tighten the edge the resize softens, and composites the result
over the frame in the caller's own colour (straight alpha; DIV-1386 records
why, not the prototype tool's own gamma-correct blend). `pkg/ui/textsmoothing.go`
and `pkg/ui/missiontextsmoothing.go` wire this into `App.Draw` and
`Viewer.Draw`: one capture window spans the whole switch, and each place
that composes a sub-picture on its own small canvas before pasting it onto a
larger destination — the tavern/school dialogue modal, the wide detached
dialogue, the mission HUD's doll/stats pane, its statistics card, and the
hover tooltip — records where its own glyphs begin and shifts them by the
same offset the picture itself is pasted at (`markCapture`/`shiftCapture`,
or the standalone `beginTextCapture`/`endTextCapture` pair for a compose call
outside any already-open window). `Options.TextSmoothing()`
(`pkg/game/tipstore.go`) is read once by `LoadOptions`, mirrors `TipsMode`'s
own shape, and is not itself simulation or save state.

DIV-1385 (the smoothing pass itself, against research silence on whether the
original's own glyph blit applies any filtering) and DIV-1386 (straight
alpha vs. the prototype's own gamma-correct blend) are recorded in
`docs/divergences/rendering.md`.

## A capture-shift bug found and fixed

Producing this story's own render proof (below) surfaced a real defect: the
mission column's statistics card (and the shared town-shop character
statistics view, since both go through `drawCharacterPaneBody`) showed two
overlapping copies of its text when smoothed — the correct pixel-exact
glyphs underneath, and a second, wrongly-offset smoothed copy on top.
`drawCharacterPaneBody` (`pkg/ui/townshell.go`) composes the statistics card
on `RenderCharacterPanel`'s own small canvas and pastes it onto the pane's
destination at a computed offset (`r.Min.Sub(b.Min)`); `capture.go`'s own
`ShiftCaptured` doc comment already named this exact site as one that must
record its glyphs' start and shift them by that same paste offset, but the
wiring itself was missing. Fixed by adding the `markCapture`/`shiftCapture`
pair around the paste, matching every other nested site.
`TestDrawCharacterPaneBodyShiftsStatisticsCardGlyphs` (`pkg/ui/textsmoothing_test.go`)
is the regression test: checked against the pre-fix commit, the same
assertion fails with the captured glyph landing on a transparent pixel
instead of its own painted one.

This was found by inspecting an offline proof render pixel by pixel, not by
a test that existed before this story: the three sites the initial
implementation wired tests for (panel, tooltip, town dialogue) were each
proven correct individually; this fourth site had no test and no render to
catch it until one was built.

## Proof

Renders (untracked, `review/story1223-smoothed-text/`, never committed): the
Valuable Documents panel, the mission column's statistics card, the tavern
Talk dialogue, and a hover tooltip, each at 2560x1440 and 1920x1080, EN and
RU, paired "off" (today's exact pixel output) against "on" (the same art
with the smoothed overlay composited on top). A pixel-diversity check
(distinct-colour count) and direct visual inspection of tight crops confirm:
the "off" renders are flat, few-colour, pixel-exact; the "on" renders carry
thousands of intermediate tones from the bicubic resample and read as
legible, softened text at the same position. RU tavern-talk was not
produced: the reached tavern's mercenary offers no enabled Talk button on
this campaign path, a fact about campaign state, not a tool defect. Hall of
Fame was not rendered: no production headless entry point reaches it (nor
does `cmd/screenshot`).

These renders are an OFFLINE RECONSTRUCTION, not a captured live window.
They reuse the real production compositor (`textsmooth.Composite`) against
captured `DrawCall`s the same way the live window does, so the text pass
itself is exact. The ART upscale underneath it is a CPU nearest-neighbour
stand-in for the live window's GPU sharp-bilinear two-step
(`pkg/ui/sharpblit.go`): Ebitengine's `ReadPixels` panics with no running
game loop (`sharpblit_test.go`'s own documented panic), so no headless tool
can read back what the GPU actually produced. Both the "off" and "on"
renders in each pair share this one stand-in, so the pair still isolates
exactly what method G's own text pass changes. The render tool itself
(`cmd/story1223tmp/`, a disposable copy of `cmd/screenshot`'s own driving
logic plus capture-window wrapping at each site) was never committed and is
deleted, per story1222's own precedent for a kit-generation tool.

Gates: `gofmt` clean; `go test -trimpath -count=1 ./...` — 55 packages ok, 0
FAIL; `internal/archtest`, `internal/storyguard`, `internal/divledger` all
pass with their ratchets updated in the same commits that moved them
(`internal/archtest/dag.go` and `composition_baseline.go`;
`internal/storyguard/baseline.go`). Mission-script census
(`cmd/missionrun -mission {10,20} -trace -ticks 1 | grep -c UNSUPPORTED`):
0 and 0, unchanged — this story touches no `pkg/sim` or script-execution
code.

## Open debt

- **The in-mission notice (dialogue/outcome/success/failure popup) is not
  wired.** `Viewer.Draw`'s own notice paint (`noticePresent`, composed in
  640x480 design space and scaled+translated into the live map viewport)
  opens no capture window at all, so it stays on the pixel-exact path
  regardless of the TextSmoothing setting. This is a scope gap, not a
  rendering bug: the notice simply never smooths.
- **A suspected, unconfirmed defect in the wide detached town dialogue.**
  `App.drawTown` (`pkg/ui/app.go`) calls `a.composeTownScreen()` — which
  internally captures its own dialogue overlay glyphs — before checking
  `a.detachedTownDialogue()`; when that check is true, the composed picture
  is discarded and replaced, but the glyphs already captured from it are
  not, since `markCapture()` for the replacement is taken after the
  discarded call already ran. This was found by static reading of the
  capture order, not reproduced by a render: this story's own proof set
  does not exercise a detached dialogue. Left for a follow-up story with an
  actual detached-dialogue render, rather than a same-session fix against
  unverified behaviour.
