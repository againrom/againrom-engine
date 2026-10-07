# Tasks — a blow leaves a number behind

**Legend.** Kind `impl` = one trailered commit `SDD-Task: 0083-damage-numerals/T<n>` touching at
least one file outside `docs/`. `verification.md` and the build are stages, not tasks.

## T1 — impl — the mission tool tells absent from dead

Files: `cmd/missionrun/main.go`, `cmd/missionrun/main_test.go`.
Covers: FR-12, AC-14, DD-12.

The health reader answers a pair; the shared reference resolver refuses an id the world does not
hold, naming the reference as written; the attack report line renders an absent victim as absent
rather than as a number, and never as a kill.

Scope fence: no other command, no `pkg/`. The waypoint path gains the resolver's refusal and nothing
else — its arrival arithmetic is untouched.

Done when: a drive naming an absent reference returns an error before any order is issued; a drive
naming a present one prints exactly what it prints today; both are witnessed by tests that need no
install.

## T2 — impl — the record, its gates, its merge and its colour

Files: `pkg/ui/numeral.go` (new), `pkg/ui/viewer.go`, `pkg/ui/numeral_test.go` (new).
Covers: FR-1, FR-2, FR-3, FR-4, FR-5, FR-8, FR-13, AC-1..AC-6, AC-9, AC-10, AC-15, P-2, P-3, P-4,
DD-1, DD-2, DD-2a, DD-3, DD-9.

The record type and the viewer fields it needs; the ingest, with the remembered health written
before either gate; the strict-decrease and display gates; the merge into an existing record; the
birth offset and its ownership sign; the owner-indexed colour.

Scope fence: no drawing, no clock, no key. The record's picture, its placement and its expiry are
T3's and T4's. Nothing in `pkg/sim` and no new seam type.

Done when: the ingest is exercised over every AC-1..AC-6, AC-9, AC-10 case from hand-built pushes,
and a schedule digest test witnesses AC-15.

## T3 — impl — the two clocks

Files: `pkg/ui/numeral.go`, `pkg/ui/viewer.go`, `pkg/ui/numeral_test.go`.
Covers: FR-6, FR-7, AC-7, AC-8, P-1, DD-1a, DD-4, DD-5.

The drift, stepped by the delta of the viewer's own animation counter; the expiry, tested against
the timestamp the viewer's step is given. Both hooked into that step.

Scope fence: the drift moves the record's offset and nothing else — it does not touch the picture,
the number or the colour, and it does not decide what is drawn.

Done when: a record drifts by exactly N steps for N counter ticks and by none for zero; a record
survives at 999 and 1000 ms and is gone at 1001, including with the counter never advanced.

## T4 — impl — the figure is composed and placed

Files: `pkg/ui/numeral.go`, `pkg/ui/viewer.go`, `pkg/ui/numeral_test.go`.
Covers: FR-10, FR-11, AC-12, AC-13, DD-6, DD-7, DD-8.

The composed picture — shadow then face, one image — built at birth and on merge; the pure method
that places every live record through the shared cell transform, carrying the entity's own
displacement; the upload and blit in the draw.

Scope fence: the placement reads the record and the frame's entities and derives no new geometry of
its own — the transform is the one the health bar already takes.

Done when: the placement is asserted under a moved camera, a zoom, a displacement and a relief lift,
and a record whose entity is absent from the frame places nothing while remaining present.

## T5 — impl — the toggle

Files: `pkg/ui/app.go`, `pkg/ui/viewer.go`, `pkg/ui/numeral_test.go`.
Covers: FR-9, AC-11, DD-10, DD-11.

The snapshot field, split from the chip key at the binding by the modifier that is already read
there; the toggle method with its default; the call on the map arm.

Scope fence: the map arm gains one statement and no other input changes. Nothing outside the map arm
reads the field.

Done when: the two `L` readings are exclusive in the snapshot, the map arm flips only the display,
and a viewer opens with the display on.

## Traceability

| Upstream | Task |
|---|---|
| FR-1..FR-5, FR-8, FR-13, AC-1..AC-6, AC-9, AC-10, AC-15, P-2..P-4, DD-1, DD-2, DD-2a, DD-3, DD-9 | T2 |
| FR-6, FR-7, AC-7, AC-8, P-1, DD-1a, DD-4, DD-5 | T3 |
| FR-10, FR-11, AC-12, AC-13, DD-6..DD-8 | T4 |
| FR-9, AC-11, DD-10, DD-11 | T5 |
| FR-12, AC-14, DD-12 | T1 |
| AC-16, P-5, SC-1..SC-4, R-1..R-3 | verification stage |
