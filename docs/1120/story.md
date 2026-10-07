# Town exterior entrance reactions

The town square gains distinct entrance reactions and independent sign/fluger
animation. Existing room destinations, bitmap labels, fonts and pixel anchors
remain. No simulation RNG, hashed state or native-save field changes.

## Contract and authority

Accepted research pin: `7ccbd801f24910aa798184d193b6b2698fe6a294`.
`TOWN-399`..`TOWN-406`, `formats/town/format.md`, `TOWN-001`..`TOWN-008`
and `TOWN-156`..`TOWN-161` govern the exterior. The partial corrections in
`TOWN-405`/`TOWN-406` supersede only the named clauses of 158/211.

- Delivered pointer updates select one label and arm tavern/shop/school
  independently. Leaving a mask region does not reset or reverse their cycles.
- Paint admits at most one hub after strictly more than 67 ms. The gate polls
  the current pointer and reverses in place. Guard direction is separate.
- Sign and fluger have independent random triggers and retained cycles.
- Conditional sounds require retained playback status and cancellation. Missing
  samples or a player without that capability remain silent.

## Presentation plan

Color tokens are the installed sky, stone, roof, foliage and metal bitmap
pixels, unchanged; no replacement hex colors are introduced. Display, body and
utility type remain the existing installed font and bitmap-label roles. Layout
is the original 640x480 square, not a redesigned panel. Researched motion is the
signature. Plan critique: decoration or explanatory player copy would obscure
the original image, so neither is added.

## Explicit policies and exclusions

Each focused, unobscured App update delivers its polled pointer, including an
unchanged position. This is an Againrom delivery policy, not a claim about
stationary physical messages in ROM1. Menu/focus loss freezes animation, hides
the label and cancels retained sounds; resuming starts a fresh paint interval.
Square reentry resets presentation state and directions. Presentation random
choices use a private generator, not the simulation generator. Original RNG
algorithm/seed and cross-visit static initialization remain Unknown. Gate
availability follows the current available-mission list; click behavior stays
unchanged.

Birds, horse/baba/dervish, statue stars and crowd-loop presentation remain
unimplemented exterior debt. This story does not close all-town-animation
DIV-153. Interiors and school-column/diamond redesign are excluded. Conditional
audio calls do not establish original audible playback.

## Proof

Implementation base: `94245f177edf1f800f42424e3654419cab67dd16`.
Reconciled master: `df6eb91ecff366e7e7fe78d81efbb907156217d4`, merged at
`317b2957`. The manifest retains both stories' release tests. The accepted pin
is descended from master's `e30c92d13d75201169fc94fe6eb2d89f8cad537f`.

The result outside unit fixtures is 81 composed App frames on each lawful
EN/RU root: 24,883,200 pixel comparisons per root, 202,815 changed-pixel
observations against entry, and nine exact decoded installed sound waveforms.
The oracle independently decodes literal bitmap/sprite/sound paths and places
them at literal evidence coordinates; it does not call the production loader,
label-anchor accessor or sprite-color resolver. Native bytes remain identical
through the trace. App LOAD starts from an actual generated native save.

`TestReleaseTownExterior1120AppFramesSoundsAndNativeLoad` plus the four existing
TownSquare release tests passed on both roots. The seat's paired full release
chain remains a landing gate; this lane did not repeat that full chain.
Offscreen Draw and its CPU composition were exercised, not a physical window
or an audible device. No owner-desktop input was sent.

The finite fixture assertions drive actual App pointer/key/Draw routes for
67/68-ms boundaries, one-step/no-catch-up, stationary pointer retention,
tavern/shop completion and rearming, school endpoint waits and shared-bit
clearing, gate reversal and unavailable guard, sound status/cancellation and
same-hub release, menu/focus suspension, all four room entries, Back, load/reset
and missing-art fallback. Native bytes, simulation binary state and hash are
unchanged by an 80-paint presentation trace. Partial art failure drops only
its family. The FrontEnd and townScreen reflective reset censuses classify
the new service and per-visit state.

Final code `daf58726` passed `go test -trimpath -count=1 ./...`; the first full
run found the unclassified random-service fixture, corrected before that pass.
`gofmt` and `git diff --check` are clean. Asset guard printed
`check-no-game-assets: clean (tree scan)`. `check-div-claims.sh` selected 299
live rows/451 claim IDs and reported 77 existing retraction matches; none is
an edited row. New policies are `DIV-834`..`DIV-838`; `DIV-150` is closed and
`DIV-153` remains open for fauna/stars. Seat allocation sweeps before/after the
ledger edits reported 32 scans, zero missing answers.

The selected `1013-world-map-one-click` scenario passed 1/1 on each root after
reconciliation. EN missionrun `-trace -ticks 1` reports zero UNSUPPORTED nodes
for both missions 10 and 20. Script populations 16/27/12 and 14/15/11
(checks/instants/triggers) match `pipeline/milestone-baseline.txt`; this story
does not move the script census.

Captured frames and logs are under the seat's ignored `review/story1120/`.
Each `en/` and `ru/` directory contains 12 PNGs, including `entry.png`,
`tavern-04.png`, `shop-15.png`, `school-10.png`, `gate-04.png`,
`gate-reverse.png` and `guard-blocked.png`. Entry, school, gate and entrance
frames were inspected on both roots. Final visual critique: motion stays on
the existing pixel anchors; no added palette, typography, borders or copy
competes with the installed square. Full exterior completeness and original
hardware audibility are not claimed.
