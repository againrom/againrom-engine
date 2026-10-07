# Tasks — 0091 sight range

`impl` = production code plus its tests, one commit each.

## T1 — the field, the form, its readers and its sources `impl`

Add the entity's own sight range, make the simulation read it, and fill it on every path a unit
reaches a world by.

Files: `pkg/sim/{world,sight,engage,binary}.go`, `pkg/data/hero.go`,
`pkg/mapload/{fromalm,start}.go`, and the tests of those packages that pin the field set, the byte
form, the digests and the form version.

Covers FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, FR-10, FR-11; AC-1, AC-2, AC-3,
AC-4, AC-5, AC-6, AC-7, AC-8, AC-9, AC-10, AC-11, AC-12; P-1, P-2, P-3, P-4, P-5; DD-1, DD-2,
DD-3, DD-4, DD-5, DD-6, DD-7, DD-8, DD-9.

Scope fence: the march's own arithmetic is untouched — the window tables, the recurrence, the ring
walk, the inset test. The definition tier's decode is untouched: both bands already carry the column
at full width, and the conversion to a byte belongs to the loader.

Done when: no constant sight range exists anywhere; the byte form is at version 18 with the range at
the entity record's tail; the pinned bytes and digests are hand transcriptions and a second test
strips the new byte off every record and reaches the previous version's pinned digest; each
placement arm fills the range from its own source in the same composite literal the rest of that
arm's numbers come from; a party member's range is his own derivation.

## T3 — the opening view spans fifteen columns `impl`

Files: `pkg/ui/viewer.go`, `pkg/ui/startview_test.go`.

Covers FR-12, FR-13; AC-13, AC-14; DD-10.

Scope fence: the centring, the arming and the one-shot application are not touched, and no row
count is introduced.

Done when: the count is 15, its documentation states what the figure rests on and carries no
AUTHORED verdict, and the test that fixes the number derives it from the decoded rectangle and the
native cell size rather than from the constant.

## T4 — the range in the template report `impl`

Add the sight range to the per-placement template row the definition-table verb prints, so the
decode can be witnessed against a lawful install.

Files: `cmd/classdump/databin.go`, `cmd/classdump/databin_test.go`.

Covers SC-5.

Scope fence: one column on the existing report. No new verb, no new flag, and no change to what the
report resolves.

Done when: the row carries the range the built world's entity holds, read off that world rather
than off a second reading of the table.

## Traceability

| Task | Requirements | Criteria | Properties | Decisions |
|---|---|---|---|---|
| T1 | FR-1 … FR-11 | AC-1 … AC-12 | P-1 … P-5 | DD-1 … DD-9 |
| T3 | FR-12, FR-13 | AC-13, AC-14 | — | DD-10 |
| T4 | — | — | — | — |
