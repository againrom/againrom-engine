# 1069 — Persisted retreat mode

## Player result

On a live mission map, Ctrl+F cycles retreat through Off, Low, High and back to
Off. Plain F remains pick-up. The setting survives a process restart under the
exact `WimpyMode` option name.

## Authority and as-built behaviour

`AI-KEY-125` binds Ctrl+F and gives the three labels. `SESS-PARAM-017` identifies
opcode `0x46` selector 3. `AI-CLASS-030` gives player scope and the two threshold
writers: modes 0, 1 and 2 set `Withdraw` to 0, 10 and 30 percent of `MaxHP`, and
set `Wimpy` to zero for every actor owned by the addressed player.
`AI-WITHDRAW-026` and `AI-WITHDRAW-027` identify those fields' full-tick
consumers. `TOWN-186` gives the persisted option name `WimpyMode`.

The F press edge is split at input: Ctrl+F reaches retreat and F without Ctrl
reaches pick-up. The setting is map-only and stands below the popup gate. Town,
text-entry and modal surfaces consume it. A player-stopped map accepts each
press into the ordinary pending command queue and applies the commands in order
on the first resumed tick.

`FrontEnd` caches the persisted Off/Low/High label and advances it on each
press. It appends a `KindPlayerParameter` command for `sim.SelfSlot`, selector 3,
then persists the new label. `sim.Step` applies the two writers to entities
whose `Owner` equals the command's player. Other owners remain unchanged.
Percentage arithmetic widens before multiplication and truncates after division.

Missing, malformed and out-of-range `WimpyMode` values fall back to Off. Every
write preserves unknown option keys. Loading the process option does not apply
it to a world: a loaded entity keeps its canonical authored or saved thresholds
until the player issues the next Ctrl+F command. `Withdraw` and `Wimpy` already
belong to the byte form and digest, so this story adds no simulation field,
format version or migration.

`DIV-346` is closed. `DIV-331` now names only Ctrl+L smoothing and Ctrl+U
autohealing. `DIV-332` remains open because no claim establishes where or for
how long the state sentence is displayed.

## Proof

- `pkg/ui/retreatcommand_test.go`: exact disjoint Ctrl+F/plain-F bindings,
  press-edge behaviour, stopped map, popup and town boundaries.
- `pkg/game/retreatcommand_test.go`: persisted three-state cycle, no mutation
  before `Step`, no option-load projection into a world, ordered stopped-world
  commands and unchanged foreign actors.
- `pkg/game/tipstore_test.go`: all three persisted values, unknown-key
  preservation and safe missing, malformed and out-of-range fallback.
- `pkg/sim/playercommand_test.go`: selector 3, all three percentages, player
  scope, invalid-command refusals, existing form/version use and exact
  save/load/hash round trip.

- `go test -trimpath -count=1 ./...` passes on the reconciled code tree.
- `check-no-game-assets.sh` is clean. `check-div-claims.sh` selects 262 live
  rows of 264, with two closed; `DIV-346` is absent from the live ledger.
- The paired release gate runs 87 of 87 tests against each EN and RU install,
  with no subjectless test.
- The selected `0163-mission-to-town` scenario is an exact-base exception,
  not a 1069 pass. Both serialized base `b71e6536` and reconciled candidate
  `7208e415` reach the same step-4 map state, then fail at step 5 because the
  first notice does not appear within 600 ticks. The failure precedes every
  retreat input, queue and simulation path, so this story does not widen into
  an unrelated mission-transition repair.

## Open debt

Ebitengine exposes a per-frame press edge rather than the original's unfiltered
Windows repeat messages. This story does not invent a render-frame repeat
cadence. It adds no state-message banner (`DIV-332`), smoothing, autohealing or
producer for another roster slot.
