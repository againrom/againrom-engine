# Tasks — 0116-compact-panel

## T1 — the measuring instrument *(implementation)*

For FR-12, FR-13, FR-14 and AC-8, by DD-8 and DD-9. No layout changes: what this task prints is the
BASELINE the story is judged against, so `AuthoredPanelLayout` must not be edited here.

`pkg/ui/panel.go`: a `String` method on `panelItem` giving `LABEL value`, or the value alone for an
empty label; and exported `PanelStatement(PanelLayout, PanelSubject) []string` walking
`panelItems`. `pkg/ui/panel_test.go` (or a new test file) witnesses AC-8, including that a dropped
row contributes no entry.

`pkg/game/panelchars.go`, a NEW file — do not edit `pkg/game/world.go`, another lane holds it:
exported `PartyCharacters(*Mission) map[sim.EntityID]ui.UnitCharacter`, a one-line wrapper over the
existing unexported `partyCharacters`. Its doc block says it exists for the measuring tool and why a
tool may not hold its own copy of the recompute.

New `cmd/paneldump`: flags `-assets`, `-mission` (default 10), `-png`. DD-8 has the load order and
the subject rule. It prints one line per subject — kind, id, `box=WxH`, row count, name — then the
stated set, then the PNG path when `-png` is given. `internal/archtest/dag.go` gains its row:
`pkg/ui`, `pkg/game`, `pkg/mapload`, `pkg/render/terrain`, `pkg/render/text`.

## T2 — a row may state two fields *(implementation)*

For FR-1 to FR-6 and AC-1 to AC-4, by DD-1 to DD-6, in `pkg/ui/panel.go` with tests beside it.

`PanelCell{Field PanelField; Label string}`; `PanelRow.Right *PanelCell`; `PanelLayout.ColumnGap
int`. `panelItem` and `panelLine` gain the second cell per DD-2 and DD-5. `panelItems` resolves both
cells, drops the row only when neither resolved, and applies DD-3's slide. `layoutLines` walks twice
per DD-4. `composeItems` draws the right cell's label and value at the shared x.

Do not change `AuthoredPanelLayout`, `panelText`, `PanelField`, `PanelSubject` or anything the
readout owns. The readout's existing tests passing unchanged is AC-4's witness — if one goes red,
the new passes are not inert and the design is wrong.

Tests witness AC-1 (all four subject shapes over a two-cell fixture layout), AC-2 (the shared origin
follows the drawn left cells), AC-3 (paint stays inside the box, the ink-layout technique
`panelInkLayout` already uses) and AC-10 (`PanelSubject` as a map key).

## T3 — the shipped layout is compact *(implementation)*

For FR-7 to FR-11 and AC-5 to AC-7, by DD-7 and DD-10, in `pkg/ui/panel.go` plus the tests that pin
the row set.

Rewrite `AuthoredPanelLayout`'s `Rows` to DD-7's thirteen entries with DD-7's labels, and set
`ColumnGap`. Change no field constant, no `panelText` arm and no colour.

Existing tests in `pkg/ui/sheet_test.go`, `pkg/ui/panel_test.go`, `pkg/ui/panel_draw_test.go` and
`pkg/ui/selcount_test.go` that read row positions, row counts or label text move with the layout —
extend them to the new shape, do not relax them.

New assertions: AC-5, walking every row and every `Right` cell and comparing the field multiset
against the twenty-one written out by hand; AC-6, with **12** and **6** as literals per DD-10, plus
the stated sets holding eighteen and nine values; AC-7 by the existing fullest-panel window test.

`AuthoredPanelLayout`'s doc block must say that the pairing, the order and every label are ours
(`UNIT-PANEL-011`), that a label may be abbreviated and a value may not, and that DD-3's slide is
what makes row 7's pairing safe.

## Traceability

| Task | Contract | Decisions |
|---|---|---|
| T1 | FR-12, FR-13, FR-14, FR-16 | DD-8, DD-9 |
| T2 | FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-15 | DD-1, DD-2, DD-3, DD-4, DD-5, DD-6 |
| T3 | FR-7, FR-8, FR-9, FR-10, FR-11 | DD-7, DD-10 |
