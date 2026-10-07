# Provenance — 0122-trigger-opcodes

Claims are cited, experiments are not. Grades are the ledger's own.

## The arms this story builds

| Clause | Claim | Grade | What it fixes here |
|---|---|---|---|
| FR-4 check 15 | `TRIG-DIST-014` | High for the metric and the five arms' agreement; **Medium** for the byte-width clause | The arm is `R2056`, the **minimum** over every member of every group of the named player, **seeded `0xff`** (`L12349 MOV EBP,0xff`, `L12350 CMP EAX,EBP`). It shares the one distance helper with arms 3, 6, 7 and 16, and that helper is `max(\|dx\|,\|dy\|)` — no multiply, no `FSQRT`, no `FILD`, no addition of the two terms |
| FR-3 check 8 | `TRIG-COUNT-015` | High | Arm 8 at `L12358` walks every group of the player and every member of every group; its complete per-member body is `INC EDI` (`L12359`). **No filter in it**, while arms 5 and 6 of the same routine *do* test `unit+0x54 == 0x10` — so the dead test exists in this vocabulary and this arm does not use it |
| P-3 both counts are living-only | `TRIG-REAP-017` | High for the removal, the sub-tick placement and the ordering | `Player::RemoveUnit` takes a dead actor out of its group at `L01863`, **after** the whole script pass at `L00279` in the same sub-tick iteration. A population count containing no health test is living-only *behaviourally*, and no script pass can see a corpse in a group |
| FR-6 action 10 | `TRIG-DIPLO-019` | High | `table[9]` = `L12375`, read out of the PE. Index `50*p0 + p1`; `matrix[p0][p1] = (v &~ 3) + p2`. Five properties, each a named instruction: one direction only, bits 2..7 survive, the parameter is **`ADD`ed not `OR`ed** (`L12382 ADD DL,CL`), **bit 1 is not respected — it is cleared**, and the arm is bare — no notify, no re-scan, no order invalidation |
| FR-5 check 10 | `TRIG-DIPLO-020` | High | `table[9]` = `L12384`; `R1085` is thirteen instructions. The **row** comes from the first resolved reference (`+0x38`) and the **column** from the second (`+0x3c`), both `MOVSX word ptr [+0x4]`; `AND EDX,0x3` before the slot store. **The script reads narrower than it writes** |
| D-5 the addend is unexercised | `TRIG-DIPLO-021` | **Medium** — the counts are an exact census over both roots; the label meanings are the map authors' word | 11 action nodes over 7 maps and 5 check nodes over 5 maps on EN (4 on RU). The addend histogram over the eleven is exactly `0 x 4, 1 x 3, 2 x 4` — **never above 2**, so the carry into bits 2..7 is not exercised by anything that ships. A mutual change costs two nodes |
| FR-7 arms 11 and 13 | `TRIG-COND-003` | High for the table, its extent, the dead arms and the three structural oddities | The 22-arm table at `L12295`, index `opcode−1`, bound by the routine's own `CMP EAX,0x15`. **Arms 11 and 13 point at `L12305`** — the dispatch loop's own continue label — so those opcodes are dispatched and write nothing, leaving the slot at its previous value |
| FR-1, FR-2 the two player references | `TRIG-REC-011` | High for the read order, the three sizes and the three bands | A compiled **check is 80 bytes**: `+0x30` unit, `+0x34` group, `+0x38` player, `+0x3c` **the second reference of whichever kind came second**. That is why a second player rides in a slot of its own rather than in the plain parameters, which sit at `+0x08..+0x2f` in *encounter* order |

## The two stale statements this story corrects

Neither is a behavioural defect. Each left a false open question in shipped Go where the ledger
had closed it.

| Site | Claim | What it settles |
|---|---|---|
| `scriptDistance`'s doc block, which says the metric is the one thing the evidence does not fix | `TRIG-DIST-014` | High **for the metric**: the helper is fifteen instructions and it discriminates by *absence* — Euclidean needs a multiply or an `FSQRT`, Manhattan needs an `ADD`, and the routine contains neither. The claim is already cited at High for exactly this fact in `docs/0075-attack-order/provenance.md`. The **byte-width** clause is separately graded Medium and stays open |
| The group count's doc block, which says the exclusion of the dead is the one clause the evidence does not fix | `TRIG-COUNT-015`, `TRIG-REAP-017` | Both High. The arm contains no health test and reads a cached `group+0x0c`; the per-sub-tick reap removes a dead actor from its group after the script pass in the same iteration. So the count is living-only behaviourally — which is what this tree computes, reached from the other side. **Our behaviour was right and its stated reason was wrong** |

## The refusals, and what each waits on

| Not built | Nodes (EN) | What is missing |
|---|---|---|
| Instants 16, 17, 18, 32, 33 | 81 | `TRIG-ACT-004` names the callees and one error string. Nothing published says what an off-map unit **is** — whether it ticks, is targetable, holds its cell, or is still seen by check 15 and the group count. Four authored behaviours, each reachable by a shipped map |
| Instant 21, the spawn | 35 | `TRIG-ACT-004` grades arm 21 **Medium** and states its six-argument order is *not* interpreted. `pkg/sim` also holds no unit template table to build an entity's numbers from |
| Checks 12, 16, 17; instants 12, 13, 20, 28 | 30 + 56 | A per-actor container that resolves a **named** item. `pkg/sim/carry.go` holds item codes, not identified items, and the binder drops a node's item reference |
| Sub-commands 11, 15 | 16 | `AI-FOLLOWSET-116`, `AI-FOLLOWAUTH-117`, both High — the law is complete. What is absent is this build's actor machine: `ActorState` has `guard` and `patrol` only, and the escort states carry a subject and a range that would widen the entity record a second time |
| Sub-command 10 | 5 | `TRIG-GROUP-005` gives the case; the scorer `R0225` it calls per member is located and not read |
| Checks 4, 9, 20, 21, 22; instants 7, 23, 24, 25, 26, 29, 30, 31, 34 | 78 | A field displacement or a callee that research located and did not interpret. `TRIG-COND-003` grades the meanings of arms 4, 9 and 20 **Medium** in its own words |
