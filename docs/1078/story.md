# Story 1078 — world-marker writer validity

## Player result

Selecting a mission without valid world-map marker art no longer adds inert
history to a native save. A valid picture-bearing mission still gains its
marker after selection and keeps it across save/load.

## Authority and behaviour

`TOWN-123` and `TOWN-040` are High: mission selection appends to the persisted
marker cache only when the mapped `MapObject` carries a picture other than
`"nothing"`; world-map paint reads that same cache.

`markWorldSelected` is the only population writer in this build. It now
requires a present mission mapping, an in-range valid `MapObject`, and a
non-empty `Picture` that does not case-insensitively equal `"nothing"`. Snapshot
serialization and restoration remain unchanged. An older native save that
already contains the former superset therefore round-trips without data loss.
The save form and envelope version do not change.

## Touched surfaces

- `pkg/game/worldmap.go`: the sole writer-side admission rule and the installed
  world-map census assertion.
- `pkg/game/worldmap_test.go`: focused malformed, absent, no-picture and valid
  mapping cases.
- `pkg/game/worldmarker_save_release_test.go`: a lawful-install selection and
  native round trip for one picture-bearing and one no-picture mission.
- divergence ledger: `DIV-138` moves to the closed ledger.

## Proof

The focused test distinguishes seven rejected shapes from one accepted mapping
and requires rejection to allocate no history map. The pre-existing snapshot
round-trip test still seeds a superset directly and requires it to survive,
which guards the writer-only boundary.

The release-gated test selects both installed mission kinds through
`WorldMapClick`, requires only the picture-bearing mission in the encoded
history, restores its marker, and keeps the no-picture mission excluded. The
installed `WorldMapSweep` independently checks every shipped mapping against
the same membership rule.

## Open debt

None for marker-cache writer membership.
