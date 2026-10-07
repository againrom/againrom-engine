# Tasks — a dialog window stops the world

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. Criterion numbers are `plan.md` §Success criteria; `R-n` is §Risks.

## T1 an open notice suspends the world

Kind: impl. Carries FR-1, FR-2, FR-3, FR-4, FR-5, DD-1, DD-2, DD-3, DD-4, DD-7, DD-8, R-1, R-2,
R-4.
Criteria SC-1, SC-2, SC-3, SC-5, SC-7.
Files: `pkg/ui/flow.go` MODIFY, `pkg/ui/app.go` MODIFY, `pkg/ui/halt_test.go` ADD,
`pkg/game/world_test.go` MODIFY, and any existing `pkg/ui/*_test.go` the change moves.

Boundary: what the front-end declares about its cadence, and when. The dim is not started here — a
frame composed under this task looks exactly as it does today.

Scope fence: the advance call is not made conditional, no seam gains a parameter, and nothing under
`pkg/sim` or `pkg/game` product code is edited. If a world field, a byte-form record or a serialized
version has to move, the boundary was crossed and this task stops.

The comment corrections DD-7 names are part of this commit, not a follow-up: the map arm's, and the
driver-level test whose wording — not whose assertion — this change falsifies.

Done when: a notice open over the production map arm suspends the advance and a dismissal resumes it
with no burst; a player's pause set before a notice is still set after it, through each of the three
dismissal routes; the three cadence keys leave nothing behind them; a screen with no lettering and
one with no cadence behind it are unaffected; and the whole gate is clean.

## T2 the map behind a notice is dimmed

Kind: impl. Carries FR-4, FR-6, FR-7, FR-8, DD-5, DD-6, R-3. Criteria SC-4, SC-5, SC-6, SC-7.
Files: `pkg/ui/notice.go` MODIFY, `pkg/ui/viewer.go` MODIFY, `pkg/ui/backdrop_test.go` ADD.

Boundary: one value, its setter, the frame decision that reads it, and one draw call. What the
notice itself looks like is not edited — no geometry, no palette, no wrap, no button.

Scope fence: nothing about the suspension is touched; T1 owns it. No second dim is introduced and
the engine margin's existing one is not read, reused or altered.

Done when: a frame composed with a notice open carries one dim over the whole drawable area and one
composed without a notice carries none; the composition order places it after the map's last stroke
and before the unit panel; a fully transparent value yields the frame that was composed before this
story; and the whole gate is clean.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1, FR-2 | SC-1 | T1 |
| FR-3 | SC-2 | T1 |
| FR-4, FR-5 | SC-3 | T1, T2 |
| FR-6, FR-7 | SC-4 | T2 |
| FR-8 | SC-4, SC-7 | T2 |
| AC-1, AC-2, P-4 | SC-1 | T1 |
| AC-3, P-2 | SC-2 | T1 |
| AC-4, AC-5 | SC-3 | T1, T2 |
| AC-6, AC-7 | SC-4 | T2 |
| AC-8, AC-9, P-1 | SC-5 | T1, T2 |
| P-3 | SC-7 | T1, T2 |
| DD-1, DD-2 | SC-1 | T1 |
| DD-3 | SC-2 | T1 |
| DD-4, DD-7 | SC-3 | T1 |
| DD-8 | SC-1 | T1 |
| DD-5, DD-6 | SC-4, SC-6 | T2 |
| R-1 | SC-2 | T1 |
| R-2 | SC-1, SC-5 | T1 |
| R-3 | SC-4, SC-6 | T2 |
| R-4 | SC-3 | T1 |

`verification.md` and the runnable build are pipeline stages, not tasks: neither is an entry above
and neither commit carries a trailer.
