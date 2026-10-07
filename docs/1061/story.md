# 1061 — Formation player command

## Player result

In a live mission, Ctrl+W cycles Formation through Auto, On, Off and back to
Auto. The press enters the ordinary input → game queue → simulation command
path. It changes the existing per-player canonical byte; no UI-only mode exists.

## Authority and as-built behaviour

`AI-KEY-125` binds Ctrl+W on the mission map. `AI-FORM-037` gives player-command
opcode `0x46`, selector 2 and remap `0→0`, `1→2`, `2→1`, default 2.
`MOVE-GATE-035` gives stored mode 0 as never in formation, mode 2 as
spread-gated and every other nonzero mode as always in formation.

The key is a W press edge under either Ctrl level. It is read only on the map
arm and below the popup gate. Town and modal/text surfaces consume it. A stopped
map accepts the command into its pending queue and applies it on the first
resumed tick, like other map orders. `mapWorld` scopes it to `sim.SelfSlot` and
selects the next authored state from the current stored mode. `sim.Step` owns the
opcode/selector dispatch and remap, then calls the raw store already shared with
trigger instant 7.

No save field or byte-form version was added. `DIV-330` is closed.

## Proof

- `pkg/ui/formationcommand_test.go`: exact Ctrl+W binding, press/held-frame
  behaviour, popup, town and stopped-map boundaries.
- `pkg/game/formationcommand_test.go`: exact queued command, three-state cycle,
  no pre-tick mutation or actor-command mark, and stopped-world deferral.
- `pkg/sim/playercommand_test.go`: opcode/selector, the complete remap and
  refusals, unchanged form version/length, read-write/hash identity, and actual
  group target/rate consumption of the changed mode.

Final repository and EN/RU gate results are recorded on the candidate commit.

## Open debt

`AI-KEY-125` records unfiltered Windows key repeat. Ebitengine exposes this
front end as a per-frame press-edge snapshot, not OS repeat events. A held W
therefore does not invent a repeat cadence. Formation captions remain the
separate open `DIV-332` presentation debt.
