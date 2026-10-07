# Spec — the info window is compact, and right for an ordinary unit

## Terms

- **Panel** — the unit information window: one box, anchored to a window corner, describing one
  unit and the selection it was drawn from.
- **Field** — one thing the panel can state (`PanelField`). The set is closed and this story
  changes it in no way.
- **Cell** — a label and the field it states, drawn as `LABEL value` on one line. A cell states
  **nothing** when its field has no value for this subject.
- **Row** — one drawn line: a **left cell** and, optionally, a **right cell**. Before this story
  every row was one cell.
- **Stated set** — the ordered list of `LABEL value` strings a subject and a layout produce, with
  no font and no pixels in it.
- **Fully-known subject** — one whose character and combat block are both known.
- **Placed subject** — one the loader knew no character for: every unit a map places.

## Problem

The panel is 44 % of the window's height for a party member — 337 pixels of 768, eighteen rows of
one value each, measured against the game's own font (analysis.md). An ordinary unit gets nine such
rows in a box pinned at the layout's width floor. Both state the right things in the wrong shape,
and the second is the case the owner named.

The panel's **layout is authored** (`UNIT-PANEL-011`): research establishes positively that the
original's arrangement cannot be recovered without the interface layer. Nothing in this contract
reproduces anything, and nothing in it may be read as a claim to.

## Scope

Layout only. The panel's value set, its resolution and its refresh rule are untouched, and no
simulation, loader or byte form is reached. The debug readout shares the composition path and its
picture must not move.

## The contract

### A row may state two fields

- **FR-1** A row carries a left cell and an **optional** right cell. A row whose cells all state
  nothing is dropped and costs no space, exactly as a one-cell row already is.
- **FR-2** Every drawn row's right cell begins at **one x**, shared by the whole box, so the second
  column is straight. That x is derived from the widest **drawn** left cell plus a layout gap; a
  row a subject dropped contributes nothing to it.
- **FR-3** A row whose left cell states nothing and whose right cell states something draws the
  **right cell in the left position**. There is no such thing as a hole in the left column.
- **FR-4** The composed box covers both cells of every drawn row on both axes; nothing a row states
  is clipped.
- **FR-5** Within a cell, the value is placed by the label's own **pen** plus the layout's label
  gap, and an empty label spends no gap — the rule that already governs a one-cell row, unchanged.
- **FR-6** A layout in which no row carries a right cell composes **the identical picture** it
  composed before this story. The debug readout is such a layout.

### What the shipped panel states

- **FR-7** The shipped layout states **exactly the fields it stated before this story** — the
  twenty-one of `AuthoredPanelLayout` — each **exactly once**, with the same values from the same
  resolver. No field is added, removed, folded into another or given a different value.
- **FR-8** Which values share a row, the order of the rows, and every label are **authored**
  (`UNIT-PANEL-011`). A label may be abbreviated; a value may not be.
- **FR-9** For a **fully-known** subject with no mana pool, no always-hits mark and one unit
  selected, the shipped layout draws **at most 12 rows**.
- **FR-10** For a **placed** subject with no mana pool, no always-hits mark and one unit selected,
  it draws **at most 6 rows**.
- **FR-11** The fullest panel the layout can produce — every field stated — still fits the default
  window with its margins, as it did before.

### Reading what it says, and measuring how big it is

- **FR-12** `PanelStatement(layout, subject)` gives the **stated set**: one string per drawn row,
  cells joined by two spaces, `LABEL value` per labelled cell and the value alone for an unlabelled
  one. It composes nothing and needs no font.
- **FR-13** A developer tool composes the panel against a lawful install and reports, for a named
  campaign mission, the **box in pixels** and the **stated set** for one fully-known subject and one
  placed subject taken out of that mission's own started world. It may write each panel as a PNG to
  a directory the caller names, and it writes nothing into the repo.
- **FR-14** The two subjects the tool measures carry the values the running game pushes: the health
  and mana pairs, the cell, the eight numbers a blow reads and the rate a step reads, all copied off
  the simulation entity; the actor's actual name, with the installed class name retained as the
  empty-name fallback; and the character the loader knew, from the same function the front end uses.

### What must not move

- **FR-15** `PanelSubject` stays **comparable with `==`**. The refresh key holds one, and that is
  the whole redraw rule.
- **FR-16** `pkg/ui` imports no new package outside what its allow-map already permits.

## Acceptance

- **AC-1** A two-cell layout over a subject that states both fields draws both, the right one at the
  shared x; over a subject that states only the left one draws only the left; over a subject that
  states only the right one draws it **at the left cell's own origin**; over a subject that states
  neither draws no row at all and the rows after it move up. (FR-1, FR-2, FR-3)
- **AC-2** The shared right-column x is a function of the **drawn** left cells only: two subjects
  differing in whether a wide left cell is stated get different right-column origins. (FR-2)
- **AC-3** Every pixel a two-cell row paints lies inside the composed box. (FR-4)
- **AC-4** A layout with no right cell anywhere composes the picture it composed before this story —
  witnessed by the readout's own picture and geometry tests passing unchanged. (FR-6)
- **AC-5** The multiset of `PanelField` values across every cell of `AuthoredPanelLayout` is exactly
  the twenty-one this story inherited, each appearing once. (FR-7)
- **AC-6** For a fully-known subject with no mana, no mark and one unit selected, the shipped layout
  draws **12** rows and its stated set holds all eighteen values such a subject has. For a placed
  subject under the same conditions it draws **6** rows and its stated set holds all nine. Both
  counts are written as literals. (FR-9, FR-10)
- **AC-7** The fullest panel — character, combat, mana, mark and a count all stated — is placed
  wholly inside the default window with its margins. (FR-11)
- **AC-8** `PanelStatement` returns one entry per drawn row, in row order, and none for a dropped
  row; a two-cell row's entry holds both cells. (FR-12)
- **AC-9** The tool, run against a lawful install for mission 10, prints a box and a stated set for
  a `party` subject and for a `placed` subject; both boxes are shorter than the baseline in
  analysis.md, and neither stated set has lost a value from it. Recorded in verification.md with the
  install it was run against. (FR-13, FR-14)
- **AC-10** `PanelSubject` is used as a map key in a test, which will not compile if it stops being
  comparable. (FR-15)
- **AC-11** The import-graph check passes with the tool's own row added to the allow-map, and
  `pkg/ui`'s row is unchanged. (FR-16)

## Derived properties

- **P-1** *A dropped row still costs nothing.* The flow index counts kept rows, and the shared
  right-column x is computed over kept rows, so neither leaves a gap behind it.
- **P-2** *Compaction cannot lose a value.* The stated set is derived from the same resolver over
  the same fields; only how many of them share a line changed. AC-5 and AC-6 read that directly.
- **P-3** *The seam is wider, not narrower.* A two-cell row is a value of `PanelLayout`, so
  replacing the whole layout with the original's — if it is ever decoded — still moves no code.

## Out of scope

- The original's own arrangement. `UNIT-PANEL-011` says it cannot be settled from the executable
  without the interface layer; this is authored and stays authored.
- Any change to what the panel states. A value found missing is named in verification.md and left.
- The inventory window, the readout's content, the notice and the dialogue.
- Aligning every value on a single column, and a drawn rule between blocks. Both were weighed and
  dropped for the reasons in analysis.md.

## Verification mapping

FR-1 to FR-6 and AC-1 to AC-4 by `pkg/ui` tests over a synthetic font. FR-7 to FR-11 and AC-5 to
AC-7 by `pkg/ui` tests over the shipped layout with literal counts. FR-12 and AC-8 by a `pkg/ui`
test. FR-13, FR-14 and AC-9 by `cmd/paneldump` against a lawful install, recorded in
verification.md. FR-15, FR-16, AC-10 and AC-11 by `pkg/ui` and `internal/archtest` tests.

## Gate check

`go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -trimpath -count=1 ./...`,
plus `scripts/check-no-game-assets.sh`, `scripts/check-doc-budget.sh`, `scripts/check-sdd-audit.sh`.
