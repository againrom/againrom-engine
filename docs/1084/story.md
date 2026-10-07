# M10 cell-entry Lightning

Mission 10's two installed cell bindings now request Lightning at the entering
ground actor and draw the bolt from the authored source. This is a bounded
hazard slice, not a claim that mission 10 is complete.

## Contract and authority

Research pin: `7f3c5270c78925f46914f7f93e52cf8e50bf2e3c`.
`UNIT-M10CELL-054` selects the type-9 cell arm and its six bytes.
`UNIT-M10ENTRY-055` admits a ground-footprint attachment attempt before the
occupancy test and targets that actor. `UNIT-M10CAST-056` connects the temporary
caster to ordinary Lightning damage. `UNIT-M10LIFE-057` supplies the absence of
a local once flag, cooldown, tail mutation or tower-health dependency.
The local mechanisms are High; composed original runtime outcomes are Medium
or Unknown. `MAGIC-ARM-014`, `MAGIC-RESIST-006`, `MAGIC-DELIVER-035`,
`SAV-CELLLOAD-111` and `TRIG-CAST-033` were read through the claim reader.

## As built

- `alm.CellBindings` narrows the builder's fields and refuses missing tails or
  malformed type-9 tiling. It does not reinterpret item/building arms.
- `mapload` binds cells and carries them through both campaign-world rebuilds.
  The initial writer refuses block bit 0. No mission number or coordinate is
  embedded in production logic.
- Movement/placement attachment visits every destination footprint cell.
  Ground and ghost request casts; air, spell 0 and spell 26 do not. Occupancy
  refusal retains the request. Route probes and standing do not cast.
- The existing temporary-cast queue receives literal byte power, including
  zero, and targets the entering actor. Ordinary damage/protection and effect
  marking run unchanged. Direct Lightning uses the existing installed bolt
  art, a cell-centered source and five visual ticks, without an actor animation.
- Native form 66 already carries cell tails and queued casts. No byte layout
  changed. Its reader now accepts zero-power pending casts. Restore does not
  attach again or inject missing tails into old saves.

## Focused proof

The asset-free cell-binding and entry tests pass. They cover narrowing, arm
selection, missing/oversized input, both ground domains, air exclusion,
footprint corners, multiple cells, blocked attempts, pure probes, literal zero
power, ordinary Air immunity, standing/reentry, and canonical before/pending/
after continuation. Prior-form fixtures 50, 55..59, 62, 63 and 65 preserve empty
binding/cast defaults. Existing focused footprint, temporary-cast, type-9,
loader, start-mission and bolt tests pass.

`TestReleaseM10CellEntry1084UsesInstalledBindingPointerLightningAndNativeSave`
passes on EN and RU. Its independent raw ALM walker reads all 11 type-9 records
and expects exactly these bindings:

| entry cell | source cell | spell/power | observed HP |
|---|---|---|---|
| 22,64 | 23,65 | 13/1 | 145 to 140 |
| 21,63 | 19,61 | 13/1 | 145 to 140 |

Each witness positions the normal campaign hero adjacent as setup, then enters
with a production App pointer order. Installed-art bolt output is present.
Standing, a pointer move to nearby (21,64), and pointer reentry discriminate the
event source. Snapshot, encoded native envelope, decode and production restore
match canonical bytes/hash before entry, with one pending cast, and after its
effect; the next tick's report/hash also match. No physical desktop input was
sent and no on-screen GPU composite is claimed.

The successfully stamped `bin/missionrun1084.exe -mission N -trace -ticks 1`
reports EN M10/M20 `UNSUPPORTED=0/0`. The seat's `milestone-baseline.txt` also
contains no unsupported rows for these missions; the unchanged script counts
are M10 `16 checks, 27 instants, 12 triggers` and M20 `14/15/11`.
This slice changes the production App's damage/bolt output, not that script-node
census. Allocation sweeps before/after DIV-540/541 report 32 namespaces and
`missing answers: 0`; concurrent seat allocations changed only the shared floors.
The reviewed candidate `38e87bcc37e21fd984a9521251dcd7c122cf343b` includes
implementation master `39883733adcf1cd377af8857e6c06a000102ae73` and pins
accepted research `7f3c5270`. Both sides of the additive ledger/manifest merge
were retained.
Focused cell, loader, footprint, temporary-cast, upgrade, bolt, difficulty,
inspection and entity-overlay regressions pass. The individual production
1084 witness passes again on EN and RU for both cells, including all three
native-save stages and nearby/standing/reentry controls.
The sole fresh-context review passed; its independent corner, domain,
target-identity, literal-zero, sound and native-continuation probes found no
player-visible or hashed-state defect. Final broad gates and landing belong to
the seat; this checkpoint does not claim those gates or full mission completion.

## Open debt

`DIV-540` records the explicit attachment integration: terrain prefilter,
row-major destination footprint, no constructor/restore/forced-headless entry,
and unchanged old-save empty defaults. `DIV-541` records the existing executor's
next-tick resolve-once simplification rather than original retry/phase timing.
Original HP loss, repeat cadence, global cleanup, destruction outcome and
original save continuity are not runtime-witnessed. The direct bolt is transient
presentation and is not restored after its already-applied damage. No structure
combat, tower-health link, spell-26 relocation, AI rewrite or original-SAV writer
is included. The two remaining type-9 source families are unchanged.
