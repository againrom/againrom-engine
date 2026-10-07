# Story `1037` — closure

**As built, 2026-08-24.** Candidate derives from implementation
`27e994c5c9f4b6bb5953a8ee62aeca21299394b0` and research
`d7ee0c62cfa4a16083d356f24b1870035e1f0209`.

## Result

The shipped ranged-monster thresholds now travel from Units data into every
applicable production entity, participate in form 60 and the world digest, and
run once in the ordinary phase-6 AI tail. The pointable campaign result is the
same on both lawful roots:

```text
withdrawal mission=100 map="scenario/100.alm" entity=109 class="Bat_Sonic.2" mode=wimpy threshold=4 radius=7 hostiles=1
  before-tail tick=6 cell=(129,62) target=none attack=108
  after  tick=7 cell=(130,61) target=(131,59)
```

The full shipped census reads EN 56 parameterised classes, 12 positive
`Withdraw`, 24 positive `Wimpy`, 38 maps, 8,094 placements, 1,551 positive
`Withdraw`, 3,464 positive `Wimpy`; RU has the same class census, 34 maps,
3,991 placements, 817 positive `Withdraw`, 1,754 positive `Wimpy`. Every
positive-`Withdraw` definition and placement resolves an EquipItem whose attack
classification is `SkillShoot`; no `Reach` proxy enters that result. Maximum
placed health is 2,000 overall and 1,024 among threshold-active actors on both
roots.

## Twelve aspects

| Aspect | Status | Closure |
|---|---|---|
| Data | PASS | Slots 34 and 35 have distinct signed fields, sentinel tests and a complete two-root census. |
| Runtime state | PASS | Both values live on the authoritative entity and are copied by placement and retained-Ghost producers. |
| Simulation | PASS | The phase-6 common tail, ordered gates, hostile blocks, fallback, geometry and normal move target are covered. ROM1's signed-WORD versus canonical-int32 health edge is `DIV-352`. |
| Player input | PASS | A commanded move is observed immediately before the tail; success replaces it and failure preserves it. |
| AI | PASS | Orders 0, Guard, Swarm, Stand Ground, Move and Swarm 2 and actor Patrol, Defend and Follow converge on one observed tail. Dead, off-map, owner-zero and Stone actors are excluded. No second scheduler or working replacement classifier exists; `DIV-351` carries the original dispatcher gates whose state this build lacks. |
| UI/HUD | N-A | The story changes an existing actor's movement decision and adds no drawing or control. |
| Triggers/scripts | PASS | Existing group dispatch remains upstream. `SESS-PARAM-017` identifies opcode `0x46`, parameter 3, `cmd+0x0e` mode, player scope and exact writers `R0186`/`R0187`; `DIV-346` records this build's absent session-command producer. |
| Inventory/equipment | N-A | Equipment remains unchanged. The census resolves the actual EquipItem and requires `SkillShoot` for every positive-`Withdraw` definition and placement. |
| Persistence/save-load | PASS | Form 60, exact offsets, round trip, hash sensitivity, same next decision and disclosed form-59 zero migration are covered. HP 65,566 survives construction and form 60 without narrowing. |
| Campaign/session | PASS | Mission loading supplies the thresholds and the real campaign witness advances the production tick on EN and RU. |
| Shipped content | PASS | All 56 parameterised classes and every placement on 38 EN and 34 RU maps are measured. |
| Interactions | PASS | Death, owner, off-map, Stone, invisibility, zero sight, off-map hostiles, attacks, failed-tail byte identity, routes and blocked-target substitution have explicit outcomes. |

There is no in-scope GAP.

## Research reconciliation

The as-built rule follows `AI-WITHDRAW-026` through `AI-WITHDRAW-028`,
`AI-CLASS-029`, the writer mechanics in `AI-CLASS-030`, `SESS-PARAM-017` and
`UNIT-STREAM-001`. `UNIT-GATE-033` later corrected the EN Units arm from the
6,587 rows used by `AI-WITHDRAW-027` to 6,672. The 85 corrected resolutions add
24 positive-`Withdraw` and 46 positive-`Wimpy` placements, explaining the
implementation witness's EN 1,551 and 3,464 exactly. The corresponding RU
correction leaves positive `Withdraw` at 817 and moves positive `Wimpy` from
1,740 to 1,754. The implementation uses the later producer-resolution result;
it does not conceal the older claim's stale census.

`DIV-345` is closed. `DIV-346` through `DIV-352` carry the absent session-command
producer, unidentified Wimpy radius source, absent fine mover positions,
irrecoverable pre-60 thresholds, established Stone freeze, absent dispatcher
gate state and health-width mismatch. `DIV-402` and `DIV-403` are closed by the
complete common-tail interaction witness and the resolved EquipItem attack
classification witness.
