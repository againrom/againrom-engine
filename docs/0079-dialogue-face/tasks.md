# Tasks — the dialogue window shows who is speaking

**Reading key.** `FR`/`AC`/`P` → `spec.md`; `DD-x` → `plan.md` §Design decisions; `R-x` → §Risks;
criterion numbers → §Success criteria.

---

## T1 — what a file says about its speakers

**Kind:** implementation.

**Boundary:** the two tests and the scan they share, inside the event-text reader. It stops before
the driver's notice state and before anything in `pkg/ui`.

**Files:** `pkg/game/eventtext.go` `MODIFY` · `pkg/game/eventtext_test.go` `MODIFY`

**Covers:** FR-1, FR-3 · AC-8 · DD-6 · R-1 · criteria 1, 2, 3

**Scope fences:** do not change what `EventPart` returns for any input it answers today. Do not
correct either containment into an equality or a parse. Do not touch the notice state, the viewer or
the front-end.

**Done when:** the file test answers on the whole payload with ASCII case folded and looks for three
letters; the per-part test answers from the SAME tag the part's body comes from, looks for four
characters, and yields the decimal number following them; a fixture in which the two tests disagree
is asserted in both directions; and the package builds and tests green.

---

## T2 — the window has two shapes

**Kind:** implementation.

**Boundary:** the layout type, the two authored values and the composition. It stops before any
viewer state: nothing here decides WHICH shape a window is.

**Files:** `pkg/ui/notice.go` `MODIFY` · `pkg/ui/notice_test.go` `MODIFY`

**Covers:** FR-2, FR-5 · AC-4, AC-7 · DD-1, DD-8 · R-4 · criteria 4, 6

**Scope fences:** do not change the window rectangle, the button, the pitch, the wrap, the clamp, the
dim or any shipped colour, and do not change the outcome layout's geometry. Keep every shipped
wrapping and hit-test assertion pointed at the without-pane layout and add the with-pane case beside
it rather than editing it. Correct the file comment that presents the without-pane rectangle as the
claim's only one; do not delete the paragraph.

**Done when:** a layout resolved for a pane and one resolved without differ in exactly the pane
rectangle and the text rectangle and agree on every other field; the pane is painted whenever it is
non-empty, carrying a supplied picture or nothing; and the package builds and tests green.

---

## T3 — the viewer holds the shape and the face

**Kind:** implementation.

**Boundary:** the viewer's notice state, the one statement that opens a dialogue, and the rebuild
key. It adds no geometry and no tag knowledge.

**Files:** `pkg/ui/viewer.go` `MODIFY` · `pkg/ui/notice.go` `MODIFY` · `pkg/ui/dialogue_test.go`
`ADD`

**Covers:** FR-1, FR-4, FR-6, FR-7, FR-8 · AC-1, AC-2, AC-3, AC-5, AC-6, AC-9, AC-10 · P-3, P-4 ·
DD-2, DD-3, DD-4, DD-5, DD-9 · R-2, R-3 · criteria 1, 2, 3, 5, 7, 8

**Scope fences:** do not add a tag literal or any knowledge of the event-text format to this package.
Do not derive the shape from the words the viewer was handed. Do not change what the outcome path's
call site passes.

**Done when:** the shape and the face are settled by every path that opens a window and dropped by
every path that closes one; a part that names nobody leaves the face standing and a named speaker
with no picture empties it; the cached picture is rebuilt when either changes with the words
unchanged; the outcome path and a fontless viewer are unchanged; and the package builds and tests
green.

---

## T4 — the driver decides both questions

**Kind:** implementation.

**Boundary:** the mission driver's notice state, the picture seam it is handed, and the front-end
field that supplies it.

**Files:** `pkg/game/world.go` `MODIFY` · `pkg/game/frontend.go` `MODIFY` ·
`pkg/game/world_test.go` `MODIFY`

**Covers:** FR-1, FR-3, FR-4, FR-5, FR-6 · AC-1, AC-5, AC-6, AC-7 · P-1, P-4 · DD-3, DD-4, DD-7,
DD-11 · R-2 · criteria 1, 5, 6, 11

**Scope fences:** do not open an archive on the paging path — the payload already read is what
answers. Do not give the picture seam a default implementation, a fallback picture or an error
return. Do not touch `pkg/sim`, the announcer, the outcome transition or the discard rule.

**Done when:** the shape is settled once when a window opens and carried unchanged across every page
of it; each part's push states whether it names a speaker and carries whatever picture the seam
returned; the seam is nil on the shipped path, so every named speaker takes the empty-pane route; and
the package builds and tests green.

---

## T5 — the corpus census

**Kind:** implementation.

**Boundary:** one new developer tool and the two files that police the package graph.

**Files:** `cmd/dlgtool/main.go` `ADD` · `internal/archtest/dag.go` `MODIFY` ·
`docs/ARCHITECTURE.md` `MODIFY`

**Covers:** AC-11 · P-2 · DD-10 · R-5 · criterion 9

**Scope fences:** print counts and nothing that came out of a file — no path, no tag, no line of game
text. Write no file. Take the asset root from `-assets`/`AGAINROM_ASSETS` alone.

**Done when:** the tool reports, for a root, how many event files exist, how many each test answers
yes for, and on how many the two disagree; it runs on both preserved roots; and the package-graph
test accepts the new package with the imports it actually has.

---

## T6 — the runnable build

**Kind:** manual runbook.

**Boundary:** `builds/0079-dialogue-face/` only. It commits nothing.

**Files:** none tracked.

**Covers:** AC-12 · criterion 10

**Done when:** the directory holds this branch's binary and a `README.md` giving the exact invocation
against a lawful install through `-assets`/`AGAINROM_ASSETS`, for **both** preserved roots, and
naming what to look at and what is deliberately absent from it.

---

## Traceability

| Spec | Plan criterion | Task |
|---|---|---|
| FR-1 | 1, 2, 3, 9 | T1, T3, T4, T5 |
| FR-2, AC-4 | 4 | T2 |
| FR-3, AC-8 | 5, 9 | T1, T4, T5 |
| FR-4, AC-5 | 5 | T3, T4 |
| FR-5, AC-7 | 6 | T2, T4 |
| FR-6, AC-6 | 6 | T3, T4 |
| FR-7, AC-9 | 7 | T3 |
| FR-8, AC-10 | 8 | T3 |
| AC-1, AC-2, AC-3 | 1, 2, 3 | T3, T4 |
| AC-11 | 9 | T5 |
| AC-12 | 10 | T6 |
| P-1 | 11 | T4 — and no task touches `pkg/sim` |
| P-2 | 9 | T5 |
| P-3 | 4, 5 | T3 |
| P-4 | 5, 7 | T3, T4 |
