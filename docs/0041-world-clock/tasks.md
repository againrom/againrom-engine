# Tasks — the period, the switch, the seam, and the digests none of them move

Legend: **files** what the entry may change — a permission, not a prediction · **fences**
what it must not do · **done when** the observable it leaves behind. Every entry below is
an implementation entry, and they land in ascending order, each depending only on entries
before it. SC-9's mutants are one to an entry, applied to production code, measured over
the whole tree and reverted by the entry that owns one; a kill is claimed only where it
was run. An entry owning no mutant says so and breaks something load-bearing of its own
instead.

## T1 — the period in microseconds, and the two ways to compute one

**files** MODIFY `pkg/render/terrain/water.go`, `pkg/render/terrain/water_test.go`,
`pkg/game/world.go`, `pkg/ui/viewer.go`

DD-1, DD-2 — FR-1, FR-2.

**fences** the decoded periods are not recomputed: the index path multiplies the shipped
millisecond quotient rather than dividing a microsecond one, and the pins on 62 ms and on
the 992 ms cycle are not edited. No index survives on the clock. The two production call
sites change their unit and **nothing else** — no rate, no stop and no bound is introduced
here, and neither the paced advance's shape nor the viewer's disabled-animation arm moves.
No rounding of its own appears beside the truncation already in use.

**done when** SC-1 and SC-2 hold, the accuracy asserted by counting ticks over a driven
elapsed schedule rather than by reading the period back, and a clock re-rated mid-run seen
to keep its remainder. Then SC-9's millisecond-quotient mutant is applied, the whole tree
run with the failing tests named, reverted, and byte-identity confirmed.

## T2 — the world clock takes a rate and a stop, and its bound becomes world time

**files** MODIFY `pkg/game/world.go`, `pkg/game/world_test.go`

DD-3, DD-4 — FR-3, FR-6.

**fences** what a tick **does** is not opened: the assembly, the queue truncation, the step
and the push stay where they are and none of them learns about a rate or a stop. The stop
gates the tick loop alone, and the baseline is written on the stopped path exactly as on
the running one. Nothing in `pkg/sim` is touched and no cadence field joins a world. The
cap is derived from the period at the call and is not stored beside it, so no second value
can fall out of step with the rate.

**done when** SC-3 and SC-6 hold, the bound read as a span of world time at three rates
and the one-tick floor witnessed where the bound is shorter than a tick. Then SC-9's
fixed-four mutant is applied, the whole tree run with the failing tests named, reverted,
and byte-identity confirmed.

## T3 — one rate to both consumers, the stop to one, and the keys that select them

**files** MODIFY `pkg/ui/flow.go`, `pkg/ui/app.go`, `pkg/ui/viewer.go`,
`pkg/ui/flow_test.go`, `pkg/ui/app_test.go`, `pkg/game/frontend.go`, `pkg/game/world.go`;
ADD `pkg/ui/cadence_test.go`

DD-5, DD-6, DD-7 — FR-5, FR-7.

**fences** the advance's own seam does not move: it still takes nothing, returns nothing
and is called once per map-screen tick, before which the keys are resolved. The stop does
not reach the viewer — no method there accepts or reports one — and the rate write and the
seam call are one statement. `cmd/mapview` is not in the file list, gains no key and keeps
its summary; the viewer's own input snapshot gains no field. The cadence call fires on a
change, never per tick, and nothing here reads a wall clock.

**done when** SC-5 and SC-7 hold, the water counter's ratio measured across a rate change
rather than inferred from the period, and the menu and picker arms asserted to ignore all
three keys. Then SC-9's coupling-write mutant is applied, the whole tree run with the
failing tests named, reverted, and byte-identity confirmed.

## T4 — what the stop does not stop

**files** ADD `pkg/game/stopped_test.go`; MODIFY `pkg/ui/cadence_test.go`

DD-8 — FR-4.

**fences** no production file is opened, and nothing here adds a drain, a freeze or a
queue: every clause is measured against statements that already stand. Nothing asserts
that water stops or that a frame advances while stopped, and no test here reads a clock or
a game install. The orders are issued through the shipped seam, one call each.

**done when** SC-4 holds in full, the frozen frames asserted on the pushed entities rather
than on the clock behind them, and the resumed orders asserted at the first tick after the
stop clears rather than at the end of a run. Then SC-9's baseline-write mutant is applied,
the whole tree run with the failing tests named, reverted, and byte-identity confirmed.

## T5 — the digests a rate and a stop do not move

**files** ADD `pkg/game/cadence_invariance_test.go`

FR-8 — P-1, P-2, AC-8.

**fences** no production file is opened. The world is built by hand over a layout small
enough to read and no game install is touched. The headless side of every digest
comparison is assembled from the command stream this entry asserts, never from a second
run of the driver under test, and nothing here asserts a wall-clock duration or a real
elapsed span.

**done when** SC-8 holds, the digests compared at every tick index rather than at the end.
This entry **owns no mutant**, and that is stated rather than filled: every production line
it exercises is already carried by an entry above, and one invented here would be measured
here and killed there. What it breaks instead is its own independence — the comparison is
re-run with the headless side taken from the driver under test, seen to pass, and
reverted, which is the measurement that the comparison could otherwise not discriminate.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1, FR-2, AC-1, AC-2 | DD-1, DD-2, SC-1, SC-2 |
| T2 | FR-3, FR-6, AC-3, AC-6, P-4 | DD-3, DD-4, SC-3, SC-6 |
| T3 | FR-5, FR-7, AC-5, AC-7, P-5 | DD-5, DD-6, DD-7, SC-5, SC-7 |
| T4 | FR-4, AC-4 | DD-8, SC-4 |
| T5 | FR-8, AC-8, P-1, P-2 | SC-8 |
