# Tasks — the walk odometer

Kinds: **impl** (one commit each). Verification and the build are pipeline stages, not entries here.

## T1 — the odometer arithmetic (impl)

New file in `pkg/render/terrain`: the sub-cell constant, the per-tick share, the integer square
root and `WalkAdvance(dx, dy, span, tick int) int`. Pure integer, total for every input, no clock
and no allocation.

Scope fence: this task adds no caller and changes no existing file. Nothing here knows what an
entity or a crossing is beyond the four integers it is handed.

Covers: FR-3, FR-4, DD-4, DD-5, DD-6.

Done when: a separate-context test in `pkg/render/terrain` re-executes the engine's own
divide-and-subtract recurrence for every span in `1..256` and requires the closed form to agree
share for share, on both a straight and a diagonal delta (SC-1); asserts a straight cell's shares
sum to 256 and its whole-crossing advance is exactly 16 timeline steps at every span (SC-2); and
asserts a diagonal crossing's advance is strictly greater and no greater than a cell diagonal's
exact euclidean length (SC-3). `go test ./pkg/render/terrain/...` green.

## T2 — the live selection reads the odometer (impl)

`SelectUnitFrame` takes the count as a sixth parameter; the moving arm selects from it, the idle
and standing arms keep the tick, and the fallback chain is untouched. The corpus audit and the
one `cmd` caller move with the signature.

Scope fence: no new file, no change to `SelectDeathFrame`, no change to the object animation
beside it, and no memory anywhere — this task's callers pass literals.

Covers: FR-2, FR-7, DD-7, DD-8, DD-10, DD-11.

Done when: the package's existing selector tests are re-expressed against the count where they
drove the moving arm and are unchanged where they drove the idle or standing arms; a negative
count, a zero-length track and a non-positive frame count each answer without panicking and equal
inputs give equal answers (SC-6); the audit's in-range/guarded split still partitions a domain of
the same size (SC-8). `go build ./... && go test -count=1 -trimpath ./...` green.

## T3 — the memory and the advance (impl)

`pkg/game`: the per-entity count and crossing tick on `mapWorld`, the tick counted where the step
memory is already written or skipped, the count advanced once per tick between the step and the
push, reset on a stationary tick for a class whose idle gate is closed; the push reads the count
and passes it to the selection.

Scope fence: `ui.MapEntity` does not change, no file under `pkg/sim` is touched, `TransitTotal` is
not read, and the push still only reads.

Covers: FR-1, FR-5, FR-6, DD-1, DD-2, DD-3, DD-9.

Done when: a test drives a world across several cells and requires the count to be the running
total of its crossings and the drawn frame at each cell boundary to be the one that total
predicts — on a class whose cycle divides sixteen and on one whose cycle does not (SC-4); a stop
resets the count for a class with no idle cycle and leaves it for one with an idle cycle (SC-5);
the canonical byte form and digest of a driven world are unchanged at every tick index and a
snapshot built twice with no tick between selects identical frames (SC-7); a mover at the rate
floor is given a transit equal to the distance one straight cell costs the odometer (SC-9). Full
local gate green.

## Traceability

| Task | FR | DD | SC |
|---|---|---|---|
| T1 | FR-3, FR-4 | DD-4, DD-5, DD-6 | SC-1, SC-2, SC-3 |
| T2 | FR-2, FR-7 | DD-7, DD-8, DD-10, DD-11 | SC-6, SC-8 |
| T3 | FR-1, FR-5, FR-6 | DD-1, DD-2, DD-3, DD-9 | SC-4, SC-5, SC-7 |
