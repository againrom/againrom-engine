# Tasks — the cadence ladder

## T1 the ladder, in the package that owns both period functions

Add the cadence ladder to `pkg/render/terrain` beside `RatePeriod`/`RateOf`/`SpeedIndexPeriod`:
rung bounds, the rungs the shipped speed table occupies, a clamp, rung → period, its exact inverse,
and a shipped-setting lookup that reports absence rather than a nearest match (FR-1, FR-2, FR-4;
DD-1, DD-2, DD-5).

Build the array from the two existing period functions; do not write the nine periods out. Assert
in tests, not in comments, that the extension's ends are the rate model's own bounds and that the
ladder strictly quickens.

Tests: the hand-written ladder table, monotonicity and the ends, both round trips exhaustively, and
the map-load cadence on a rung (SC-1, SC-2).

## T2 the seam carries a period, computed once

Change `MapCadence`, `Viewer.SetRate` → `SetPeriod` and `mapWorld.setCadence` to carry a tick
length in microseconds; make the front-end hold a rung, born from the map-load period, stepped by
one and clamped through the ladder's own clamp (FR-1, FR-2, FR-3, FR-5; DD-3, DD-4, DD-6).

Neither consumer may divide anything. Existing tests that select a cadence by rate move to periods;
the two that assert the old doubling ladder are rewritten for the rung ladder, and any test whose
*claim* changes has its comment rewritten rather than only its code.

Tests: the driven walk of every rung in both directions, the driven round trip, and both consumers
over one driven second (SC-3).

## T3 the screen says which side of the shipped set

Add a readout field and row stating which of the game's nine settings the clock is running at, or
how far past which end our extension has taken it, or that it is on no rung at all (FR-4, FR-6;
DD-5).

Derive it from the pushed period alone. Tests: every rung's text and three off-ladder periods,
hand-written (SC-4, SC-5).
