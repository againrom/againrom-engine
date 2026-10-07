# Tasks — the attack affordance

Kinds: `impl` (one commit, trailered). Every task is `impl`.

## T1 — the second writer of the mode, and the two ways it is lowered

**Files:** `pkg/ui/app.go`, `pkg/ui/command.go`, `pkg/ui/viewer.go`, plus tests in `pkg/ui`.

**Covers:** FR-1, FR-2, FR-6, FR-7 (the state half) — AC-1, AC-2, AC-3, AC-4, AC-9 (the state half),
AC-10; DD-1, DD-2, DD-3, DD-4; P-2.

**Boundary:** the front-end snapshot gains the modifier level and the negated focus flag with their
bindings; the viewer gains the latch field and the mode predicate; the front-end's map arm sets the
latch; the front-end's step clears both fields on an unfocused tick; the viewer's step clears both
in the return that drops the gesture latches. The gesture built in `command` takes the predicate in
place of the bare field.

**Fences:** `decide` is not edited and its signature does not change. `armAttack`, `canArmAttack`,
`SetLocalOwner` and the one-press clear in `command` are not edited. Nothing is drawn.

**Done when:** the predicate answers to a held level with no press, ungated, over both selections
the key refuses; a press under the level yields attacks and leaves the level's mode up; an unfocused
tick and a popup each leave it down and neither restores it; and the whole shipped `pkg/ui` suite is
green unedited except where a snapshot literal must name a new field.

## T2 — what the frame draws

**Files:** `pkg/ui/cursor.go` (new), `pkg/ui/viewer.go`, plus tests in `pkg/ui`.

**Covers:** FR-3, FR-4 (the consuming half), FR-5, FR-7 (the drawn half) — AC-5, AC-6, AC-8, AC-9
(the drawn half), AC-12; DD-5, DD-6, DD-7, DD-8, DD-9, DD-13.

**Boundary:** the viewer gains the picture field and its setter, a pure method answering what the
pointer is and where it goes, a pure method answering which unit is marked, the draw statements for
both, and the engine cursor-mode call driven by the pointer method's own bool against a field
holding what was last asked for.

**Fences:** no archive is opened and no container is decoded here — the picture arrives set. The
overlay pass slice is not widened. The unit panel, the readout, the notice and the dim are not
edited, and no statement is moved past them except the two this task adds after the notice.

**Done when:** both pure methods answer every case of AC-5, AC-8, AC-9 and AC-12 with no window;
the cursor-mode field tracks the pointer method's bool and changes only when it changes; and a
viewer given no picture answers with the authored mark rather than with nothing.

## T3 — the picture, out of the install

**Files:** `pkg/game/cursor.go` (new), `pkg/game/frontend.go`, plus tests in `pkg/game`.

**Covers:** FR-4 (the producing half) — AC-7, AC-11; DD-10, DD-11, DD-12, DD-14.

**Boundary:** a loader beside the font's that reads the attack cursor's entry, decodes the container
with its palette declared, resolves frame 0's painted cells to premultiplied colour at the decoded
coverage and its unpainted cells to transparent, and returns one picture; the front-end loads it
once at startup beside the font, keeps it, reports a failure the way the font's is reported, and
pushes it to the viewer on both map-opening paths beside `SetFont`.

**Fences:** `LoadFont`, the language selector and the two bundles are not edited. No test reads a
game install — the container fixture is bytes built in test code. No new package is added to the
import graph.

**Done when:** a synthetic stream with a known palette and known literal words resolves to the
stated colour and coverage, an unpainted cell is fully transparent, a malformed stream yields an
error and no picture, an absent entry is a reported reason rather than a refused mission, and both
map-opening paths push whatever the front-end holds.

## Traceability

| Upstream | T1 | T2 | T3 |
|---|---|---|---|
| FR-1 | x | | |
| FR-2 | x | | |
| FR-3 | | x | |
| FR-4 | | x | x |
| FR-5 | | x | |
| FR-6 | x | | |
| FR-7 | x | x | |
| FR-8 | x | x | x |
| AC-1, AC-2, AC-3, AC-4, AC-10 | x | | |
| AC-5, AC-6, AC-8, AC-12 | | x | |
| AC-7, AC-11 | | | x |
| AC-9 | x | x | |
| P-1 | | x | x |
| P-2 | x | | |
| P-3, P-4 | x | x | x |
| DD-1, DD-2, DD-3, DD-4 | x | | |
| DD-5, DD-6, DD-7, DD-8, DD-9, DD-13 | | x | |
| DD-10, DD-11, DD-12, DD-14 | | | x |
