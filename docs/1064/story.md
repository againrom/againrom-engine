# 1064 — Unpaced owner loop

## Player result

On a live mission map, Ctrl+numpad `+` selects genuine maximum/unpaced play:
each eligible map frame advances exactly one simulation sub-tick, with no
wall-clock deadline and no catch-up burst. Ctrl+numpad `-` restores the paced
ladder at the rung the player left selected and starts it from a clean phase and
time baseline. Bare `+/-` changes persist the normal rung in local settings, so
the next map and the next process start at that paced speed. Unpaced itself is
never restored at startup.

## Authority and as-built behaviour

`AI-KEY-125` binds the two modified numpad keys as set/clear commands and keeps
the bare pair on ordinary speed stepping. `SESS-CLOCK-005` distinguishes the
deadline-paced loop from `R0260`'s one-sub-tick-per-idle-callback loop and
requires phase/epoch reset on return. `SESS-IDLE-007` places the latter on the
process that owns simulation. `TOWN-186` names `GameSpeed` in the same persisted
option table as the already implemented `TipsMode`.

The front end carries a separate unpaced bit beside, not inside, the ordinary
cadence rung. On every eligible unpaced map frame `mapWorld.paceTo` runs one
unchanged `tick()` even when time did not advance, moved backwards or jumped by
hours. Ambient map animation advances by one through the same mode. Player pause
still stops the world but not ambient animation; popups stop both; town, load,
documents and other non-map surfaces never reach the command. Bare numpad `+/-`
continue to move the stored paced rung, including while unpaced is selected.
Every real rung move immediately writes the complete normal ladder rung under
the exact `GameSpeed` key in the existing local `options.txt`; unknown option
keys survive. Missing, malformed and out-of-range values safely select the old
shipped default instead of an edge. Map entry applies a restored nondefault
period to the viewer and world before their first frame, always with unpaced
clear.

Clearing unpaced mode resets the world and viewer wall-clock baselines and their
fractional ticker phases. It preserves the period, animation counter, selected
rung, ordinary cadence readout and simulation state, so the restored arm owes no
elapsed span and rewinds nothing.

Both the mode and normal-rung preference are driver/presentation state. They add
no `pkg/sim` field, campaign save byte or hash input. Only the normal rung enters
the separate local option store; a loaded or newly opened map always enters the
paced arm. `DIV-329` is closed.

## Proof

- `pkg/ui/unpaced_test.go`: exact modified-numpad bindings, disjoint bare keys,
  per-frame ambient advance at equal/backwards/stalled timestamps, clean paced
  return, stored-rung/readout continuity, stop, popup and non-map boundaries;
  persisted nondefault entry, bare-step writes under unpaced and paced-only map
  re-entry.
- `pkg/game/tipstore_test.go`: exact `GameSpeed` round trip across fresh store
  values, complete ladder endpoints, unknown-key preservation, safe malformed
  fallback and FrontEnd-to-App restore wiring.
- `pkg/game/unpaced_test.go`: exactly one canonical simulation tick per call,
  no deadline/catch-up, stopped behaviour, direct-tick digest identity, and the
  reset of remainder/baseline without counter or state rewind.
- `pkg/render/terrain/water_test.go`: the one-tick and phase-reset clock
  primitives preserve period and logical count.
- Existing formation, cadence, halt, save and release tests remain the
  regression witnesses for Ctrl+W, paced timing, suspension and byte form.

On the expanded candidate reconciled with implementation master through stories
1062 and 1063:

- `go test -trimpath -count=1 ./...` passed, as did the focused terrain, UI and
  game packages used while editing the owner-loop and preference paths.
- `scripts/check-no-game-assets.sh` reported a clean tree scan.
- `pipeline/check-div-claims.sh` selected 266 live rows of 267 and accepted the
  move of `DIV-329` to the closed ledger at research pin `e60b8a1`.
- one paired `pipeline/check-release-tests.sh` run passed all 85 gated tests on
  EN and all 85 on RU, with no missing subject.
- the selected input/session witnesses `0163-mission-to-town` and
  `1035-documents-mission10` each passed 1 of 1 on the EN root. The headless key
  vocabulary has no modified-numpad event, so the exact chord and GameSpeed
  write remain witnessed at the controllable App/store seams rather than
  claimed from those scenarios.

The milestone census was not run: this slice changes no script decoder,
dispatch or mission input and therefore cannot change its measured unsupported
node population.

## Open debt

`AI-KEY-125` records unfiltered Windows repeat. Ebitengine exposes a per-frame
press-edge snapshot, not ordered OS repeat messages. The clear command performs
its phase reset once per exposed press edge; this story does not invent a
render-frame repeat cadence for a held key. If both set and clear edges appear
in one snapshot, their lost host order is unresolved and they cancel, matching
the existing simultaneous faster/slower rule.
