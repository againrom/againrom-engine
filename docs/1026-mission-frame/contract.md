# Story `1026` — the mission screen's virtual 1024x768 frame

## Result

The mission screen composes at a virtual 1024x768 frame and that frame is fitted into the player's
window, the way the town family's 640x480 frame already is. The viewport is the frame minus a
160-pixel right strip, so 27 map columns by 24 rows are visible where 15 by 15 are visible today, and
the right strip is 160 by 768 rather than 160 by 480.

Pointable in `builds/current/`: the mission screen shows more map, and the right strip runs the full
height of the frame. Story `1023` then builds the strip's contents, which it cannot do today because
the slot its fourth widget needs has zero height at 640x480.

## Why this story exists

Owner ruling, 2026-08-21. Full text: `pipeline/archive/owner-ruling-2026-08-21-resolution.md`. He
asked for 1024x768 outside the town, said the town keeps a pseudo 640x480 because the original nails
that composition down, and gave his reason for 1024x768: the original's own composition is designed
for it. *Pseudo* is his word and it is the mechanism this tree already ships for the town.

## What is decoded

`SESS-VIEW-028` (High for everything used here) gives the map view's geometry:

- The view object is constructed with the rect `(0, 0, screenW - 0xa0, screenH)` — the screen minus
  a **160-pixel right strip**, which is the panel.
- Columns are `(right - left) / 32` and rows are `(bottom - top) / 32`, a signed divide toward zero.
  The rect's bottom is then snapped to `top + rows * 32`.
- **The spans in cells: 640x480 gives 15 columns by 15 rows; 800x600 gives 20 by 18; 1024x768 gives
  27 by 24.** Every immediate re-read from both roots.
- The shipped default is 640x480. The selection order is the command line for `-800`, `-1024`,
  `-640`; then the `RESOLUTION` registry buffer for `-800` and `-1024`; then fall through to the
  640x480 arm. The registry read's own failure path copies the literal `-640` into its buffer, so
  both the no-switch path and the no-registry-value path terminate on the same arm.
- The viewport is fixed per resolution **except vertically**, where an open panel shortens it:
  rows are recomputed from a child control's own height as `(((h - 1) & ~0x1f) + 0x20) / 32`, and
  mission entry itself invokes the open and close pairs to restore the store's `Inventory/IsOpen`
  and `SpellBook/IsOpen`.

The three spans were checked arithmetically against the rect expression at this seat before this
contract was written: `(1024 - 160) / 32 = 27` and `768 / 32 = 24`; `(640 - 160) / 32 = 15` and
`480 / 32 = 15`; `(800 - 160) / 32 = 20` and `600 / 32 = 18.75`, which truncates toward zero to 18.
All three agree with the claim's own figures. The lane re-derives this rather than taking it here:
read the row whole with `go run ./tools/claim SESS-VIEW-028` from `implementation/research`.

## What is not decoded

- **Windowed presentation.** The original ran at a fixed resolution and nothing about fitting a
  frame into a resizable window is established by research. `pkg/render/frame`'s own header says so.
  The scaling rule is this project's design and stays authored.
- **Which child class is which widget** in the right strip. `SESS-VIEW-028` is **Medium** there: the
  child ids are read, the classes are not decoded. That is story `1023`'s problem and this story does
  not touch it.
- **What the original does at a window size that is not one of its three arms.** It has no such
  concept; the question only exists because this build has a resizable window.

## What this build does today

- `pkg/render/frame` composes at a fixed virtual size and fits it into the window at one uniform
  scale, centred and letterboxed. The scale is held as an exact rational `num/den` and every decision
  is integer arithmetic, so `WindowToFrame` and `FrameToWindow` are exact inverses wherever a frame
  pixel has a window pixel at all. **`W = 640` and `H = 480` are package constants and `Fit` reads
  them directly**, so there is one frame size in the tree.
- `App.Layout` adopts the window size, computes `a.place = frame.Fit(winW, winH)` for the menu, the
  picker, the chargen pages and the town family, and forwards the raw window size to the viewer.
- **`Viewer.Layout` sets `cam.ViewW, cam.ViewH` to the window size.** The mission screen fills the
  window at its native resolution; there is no virtual frame for it at all.
- `applyStartView` sets the start zoom to `ViewW / (cols * CellSize)` with `cols` from
  `AuthoredStartColumns()`, which returns `startColumns = 15` — the 640 arm's span, pinned.
- `DefaultWindowW = 1024` and `DefaultWindowH = 768` size the startup **window**. They are not a
  frame and they do not change the viewport span.

## Behaviours

- **B1 — the frame package takes its size as a value.** `Fit` and every mapping work for any frame
  size, and the three invariants the package's header states continue to hold by construction: the
  whole frame inside the window for any window at least 1x1; a window position mapping to at most one
  frame pixel and a letterbox position to none; `WindowToFrame` after `FrameToWindow` the identity
  wherever `FrameToWindow` reports ok. Its existing tests are the witness and must be extended to run
  at more than one size, because a test that only ever runs at 640x480 cannot see a constant that was
  left behind.
- **B2 — the mission composes at a virtual 1024x768 frame**, fitted into the window by the same rule
  the town family uses.
- **B3 — the mission viewport is that frame minus the 160-pixel right strip**, giving 27 columns by
  24 rows at the start view, and the right strip is 160 by 768.
- **B4 — mission input maps window to frame to world.** Every hit test in the mission currently works
  in window pixels. **This is the risk in this story**: a mapping that loses a pixel row makes a strip
  of a control unclickable, and a hit test left in window space is silently wrong only at scales
  other than 1:1, which is the scale a developer's own window is least likely to be at. Enumerate
  every mission-screen hit test and state that the population is complete.
- **B5 — the town family is unchanged.** Its virtual frame stays 640x480, and the menu's
  integer-multiple window rule is untouched.

## Domains

**Client only.** Verified at this seat before the contract was written: `startColumns` and
`AuthoredStartColumns` have no consumer anywhere in `pkg/sim` or `pkg/game` — the grep is
`grep -rn "startColumns\|AuthoredStartColumns" --include=*.go`, and its only hits outside
`pkg/ui/viewer.go` are none. The lane re-runs it rather than believing this.

**This story does not reach hashed simulation state.** The viewport is a camera property. Nothing
here changes what the simulation computes, only how much of it is on screen.

## Ceiling

**Three adversarial passes.** One domain, five behaviours, no hashed reach. Five behaviours is at the
limit this project's own rule allows without recording a reason, and the reason it is not split
further is that B1 through B4 are one result — a frame the mission draws into — and splitting them
would buy four lanes and four landings for one visible change.

## What is authored and what is not

| Thing | Which |
|---|---|
| 27 columns by 24 rows at 1024x768; the 160-pixel strip | **Decoded**, `SESS-VIEW-028`, High |
| Running the mission on the 1024 arm rather than the shipped 640 default | **Owner directive**, 2026-08-21. A divergence row whose owner-directive cell is filled, so it stays |
| Fitting the frame into a resizable window; the letterbox; the exact-rational scale | **Authored**, and already disclosed for the town family. `pkg/render/frame`'s header states it |
| The wheel zoom | **Owner directive**, earlier. Nothing here withdraws it |
| What happens to the row span when a panel opens | **Open.** `SESS-VIEW-028` decodes the original's recompute; whether this build has the panels it applies to is a question for the lane, and the answer may be that it does not yet |

`DIV-210`..`DIV-216` are reserved for this story, allocated with `pipeline/next-div-id.sh` on
2026-08-21 against a highest id in use of `DIV-209`. Allocate from that range and record any tail as
returned unused; a returned id is never reissued.

## Out of scope

- **800x600.** `SESS-VIEW-028` gives it at 20 columns by 18 rows. The owner named 1024 and named 640
  as the town's, and said nothing about the middle arm. It is not built on his silence.
- **The original's own resolution selection** — the `-640`, `-800` and `-1024` switches and the
  `RESOLUTION` registry key. This build is not selecting an arm at run time; it is composing the
  mission at one frame size the owner named.
- **The right strip's contents.** Story `1023` owns every widget in it. This story gives that story a
  160 by 768 strip and nothing more.
- **A genuine resolution mode.** He asked for pseudo and said so twice.
- The remaining divergence rows the 2026-08-21 "bring everything to the original" ruling reaches.
  That is a programme, not this story.

## Gates

Both repositories' Go chains by glob, exit codes captured and accumulated. `check-release-tests.sh`
and `check-scenarios.sh` on both roots, by name — this story changes what the screen draws, the
repository chain cannot reach those tests, and a skip and a pass both print `ok`.

`pipeline/check-milestone.sh` **before and after**. This story changes the map viewport, and a
viewport change that moved the census is a fact the closure owes either way. Report the number each
script prints, not its verdict. Deletion set empty or explained.
