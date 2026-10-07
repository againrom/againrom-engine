# plan — 0158 in-game menu

One lane implements the whole slice, so there is no `tasks.md`. Every FR and DD is accounted for in
`verification.md`.

## Where the work lands

| File | What changes |
|---|---|
| `pkg/ui/gamemenu.go` | new. The two surfaces, the row model, the geometry, the accelerator rule and the panel paint. |
| `pkg/ui/popup.go` | `popupOpen` gains one disjunct on the viewer and one screen test on the flow (FR-2, FR-3). |
| `pkg/ui/save.go` | the row list moves out; the actions stay and gain the new ones (FR-10, FR-11). |
| `pkg/ui/viewer.go` | one field the flow raises while the menu is up. |
| `pkg/ui/app.go` | the menu arm holds the map; the draw path composes the panel over the surface behind (FR-1, FR-4). |
| `pkg/game/headless.go` | the two row targets the scenario driver names. |

## Design decisions

**DD-1 The menu stays a `Screen` value and gains a surface field.** `ScreenGameMenu` already exists
and eight call sites across two packages key on it — the headless driver, the load window's return
arm, the app's input dispatch. Making the menu a viewer-owned box instead would move all of them for
no behaviour. What changes is that the screen no longer *replaces* the surface behind: `menuBack`
already records which surface that is, and the draw path now reads it (FR-1).

**DD-2 One popup answer, one disjunct.** `popup.go` states the rule this story is the first to
exercise: a second popup inherits the whole of 0077 by making `popupOpen` answer true, and by nothing
else. So `Viewer.popupOpen` becomes `NoticeOpen() || menuUp`, and the flow's screen test widens from
`screen == ScreenMap` to "the map is the surface showing", which is the map screen or the menu
standing over it. The dim, the camera pin, the ambient clock and the input gate then all move
together, with no second condition beside any of them (FR-2, FR-3).

**DD-3 The menu arm holds the map.** The map arm runs three statements above its own popup gate —
the cadence call, the advance and the viewer step — and `app.go` records why each must run under a
popup: the cadence call is what declares the stop, and the other two consume their pacing baselines
on every call including the stopped ones. Skipping them banks the held span and pays it in one jump,
unbounded for the ambient clock. The menu arm therefore runs the same three statements, with the
three cadence keys nailed to false because the menu takes them. This is what FR-4 asks for, and it
is what the four-row screen this story replaces did not do.

**DD-4 The dim comes from the viewer over the map and from the canvas over the town.** Over the map
it is already drawn, by `noticeBackdropOf` under the popup answer DD-2 raises — no new code. The
town screen has no viewer to inherit it from, so the town arm fills its own canvas with the same
authored value. One value, `AuthoredNoticeBackdrop`, two draw sites, because the two surfaces are
composed by two different paths (FR-3).

**DD-5 The panel paints at frame coordinates into whatever image the caller hands it.** The town is
composed into the 640x480 canvas and then placed; the map screen draws at window resolution and
returns before the canvas exists. So the paint routine takes a destination and writes frame
coordinates, the town arm hands it the canvas, and the map arm hands it a transparent overlay image
placed by the same `frame.Placement` transform. Hit-testing is `WindowToFrame` in both cases, so what
is clickable is what is drawn (FR-6, FR-9).

**DD-6 The accelerator is resolved from the label and is not stored.** `MENU-KEY-013`'s whole content
is that the label wins over the immediate. Storing the resolved letter per row would let the two
disagree the moment a label changes, which is the defect the claim names in the other direction. The
fallback is stored, because a label with no `~` has nowhere else to get one (FR-7).

**DD-7 The row list is built at open time with its disables resolved.** The predicates read the save
store, and reading it per frame would let a row change under the player's hand between the frame he
aimed at and the frame he clicked. It is also what `openLoad` already does with its own list, for the
same stated reason (FR-8, FR-11).

**DD-8 The frame art and the label file are seams, not stubs.** Nothing in this story loads
`lm.256` or `dialogs.txt`. The panel rect and row rects are the decoded ones and the accelerator is
resolved from the label, so both arrive as data later and move no code (spec AU-1, AU-2).

**DD-9 The row model carries an action constant, not a callback.** Eight actions switched on by value
in one statement is smaller than eight closures rebuilt on every open, and the order is contract: a
table of callbacks could be reordered without failing anything. This is `save.go`'s existing shape
and it is kept (FR-5, FR-10).

## Checks

**SC-1** `go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -trimpath
-count=1 ./...`, and every `scripts/check-*.sh`.

**SC-2** The script-gap census, mission 10 and mission 20, against
`pipeline/milestone-baseline.txt`. This story adds no script node handler, so both numbers are
expected unchanged; recording them is what makes that a claim rather than an assumption.

**SC-3** The panel and row rectangles are asserted against the decoded numbers directly, in the
frame's own coordinates, rather than against a layout helper's output — a helper that agrees with
itself proves nothing about the decode.
