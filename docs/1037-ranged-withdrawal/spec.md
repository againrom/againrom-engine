# Story `1037` — as-built specification

**Canonical, 2026-08-24.** Base
`27e994c5c9f4b6bb5953a8ee62aeca21299394b0`, research pin
`d7ee0c62cfa4a16083d356f24b1870035e1f0209`, byte form 60.

## Functional requirements

**FR-1 — streamed definition.** `UnitDef.Withdraw` and `UnitDef.Wimpy` are
signed `int32` values read from Units slots 34 and 35. A sentinel cell leaves
the constructor's zero. The fields are distinct from slot 36 `SeeInvisible`.

**FR-2 — every applicable producer.** A resolved Units placement copies both
values through the loader's spawn block into its entity. An unresolved record,
the Humans arm and player-party production retain zero. A retained Ghost
template copies both values when it raises its entity. The two lawful installs
contain 56 parameterised classes: 12 have positive `Withdraw`, 24 have positive
`Wimpy`, and every positive-`Withdraw` row resolves an EquipItem whose attack
classification is `SkillShoot`. The complete placed populations are EN 8,094
total, 1,551 `Withdraw`, 3,464 `Wimpy`; RU 3,991 total, 817 `Withdraw`, 1,754
`Wimpy`. The EN delta from `AI-WITHDRAW-027` follows the later `UNIT-GATE-033`
correction to Units-versus-Humans resolution.

**FR-3 — one common tail.** On phase 6 of each full tick, after engagement and
ordinary group-order dispatch and before movement, the world walks entities in
canonical order. A living, owned, on-map, non-Stone actor receives one
withdrawal decision. `HP <= Wimpy` is evaluated first, then `HP <= Withdraw`.
Both comparisons are signed, absolute and inclusive; there is no positive-value
enable check. Dead, owner-zero and off-map actors do nothing. The Stone
exception is `DIV-350`. This build has no state corresponding to the session
and per-group dispatcher gates named by `AI-WITHDRAW-026`, so its owned groups
cannot disable the tail; `DIV-351` carries that missing state. ROM1 reads
current health as a signed WORD; this build compares canonical `Entity.HP` as
`int32`. Both lawful roots stay within the common range, while `DIV-352`
records the custom-state edge.

**FR-4 — hostile blocks and fallback.** Each arm collects actors hostile in
the retreating actor's directional relation, in entity order, from an inclusive
square. Wimpy uses `ScanRange`, as disclosed by `DIV-347`; Withdraw uses radius
2. Off-map records are excluded, but corpses remain in the block. A nonempty
Wimpy block consumes priority even if only corpses remain and invokes ordinary
hostile acquisition. The retreat calculation separately filters the chosen
block to living actors and uses ordinary acquisition when none remain. Sight,
invisibility, attack reach and route reachability do not filter the block.

**FR-5 — retreat geometry and order replacement.** The retreat cell is three
cells away from the arithmetic mean of the living hostile positions. Whole
cells are lifted to 8.8 values, as disclosed by `DIV-348`. Zero deltas become
one. The larger absolute axis advances by `3*256`; the smaller axis is scaled
proportionally with signed truncation. The result shifts to whole cells and
clamps to `[8,width-9]` and `[8,height-9]`. A successful decision clears the
victim, route and group speed, then writes the ordinary movement target. The
existing route finder owns unreachable-cell substitution.

**FR-6 — interactions and command boundary.** Patrol, guard, escort, attack
pursuit, player-commanded groups and script orders keep their ordinary
dispatch. Only a successful later tail replaces that decision. The unreachable
spawn classifier in `AI-CLASS-029` is not recreated. `SESS-PARAM-017`
identifies the writer boundary as opcode `0x46`, parameter 3, the mode at
`cmd+0x0e`, the decoded player scope and the exact writers `R0186` and
`R0187`. This build has no session-command producer; `DIV-346` records
that production debt.

**FR-7 — canonical state and migration.** Entity records append signed
`Withdraw` at offset 282 and signed `Wimpy` at 286. Form 60 has record width
290. Both values participate in the canonical digest and exact round trip.
The form-59 upgrader appends two zeros per record and returns
`SaveFormWithdrawalThresholds`; `DIV-349` records the irrecoverable loss.

**FR-8 — shipped result.** `classdump -withdraw` measures definitions and all
maps under the selected root. It resolves the actual Units-row EquipItem and
its attack classification for every positive-`Withdraw` definition and
placement. It also reports maximum placed health: 2,000 overall and 1,024 among
threshold-active actors on both roots. `missionrun -mission 100 -withdrawal`
clones the ordinary loaded mission world, selects and wounds a real threshold
actor, advances the production phase-6 tail, prints the ordinary decision
immediately before that tail and then the replaced destination. Both EN and RU
select `Bat_Sonic.2` at `(129,62)` and move it to `(130,61)` with target
`(131,59)` on tick 7.

## Design decisions

**DD-1.** The two fields sit beside `HP` and `MaxHP` on `Entity`; no derived
cache or parallel AI record owns them.

**DD-2.** `spawnBlock` is the single ordinary placement carrier. The retained
Ghost template is the only separate non-party producer with definition state.

**DD-3.** `step` calls one `withdrawalPass` in the existing phase-6 common tail,
after both decision producers and before movement consumes their result.

**DD-4.** Spatial collection and living flee selection are separate filters so
corpse-only Wimpy blocks preserve their decoded priority effect.

**DD-5.** The unidentified Wimpy `pth+0x08` source is represented by the
existing actor `ScanRange`, with the uncertainty carried in `DIV-347`.

**DD-6.** Mean and line arithmetic use `int64`. Canonical whole cells are lifted
before arithmetic; the missing fine mover position is carried in `DIV-348`.

**DD-7.** Withdrawal writes the existing normal target and lets the ordinary
route builder perform its reachable substitute. It adds no route search.

**DD-8.** Form 60 is required because both thresholds can change the next
hashed tick. Form 59 remains the predecessor and its loss is explicit.

**DD-9.** Install-backed release tests keep the full population census, resolved
EquipItem classification, health bounds and real mission transition executable
without letting repository tests read the preserved installs by default.

## Divergence reconciliation

`DIV-345` closes the parsed-but-discarded implementation gap. `DIV-346` through
`DIV-352` record seven measured residual boundaries. `DIV-402` and `DIV-403`
record and close the pass-1 witness debts for the common-tail interaction
population and the EquipItem classification proxy. No known in-scope GAP
remains.
