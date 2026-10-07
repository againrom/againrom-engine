# 0130 — the hotfix ledger pays its debt: plan

Three tasks, in dependency order. Each is one commit.

```
T1  docs/hotfix/ARCHIVE.md, LEDGER.md   the move and the one-line rows   FR-1, FR-2, FR-3, FR-9, DD-1
T2  docs/<NNNN>/spec|plan|tasks.md      the folds, and the marks         FR-4, FR-5, FR-6, FR-7, FR-8, DD-2
T3  scripts/check-hotfix-ledger.sh      the measure                      FR-10, FR-11, FR-12, FR-13, DD-3
```

T1 before T2 because T2's fold markers point at T1's anchors and T2 fills in
T1's `Owes` cells. T3 last because it measures T1's output. FR-14 is a
constraint on all three: no task touches `pkg`, `cmd` or `internal`.

## Design decisions

**DD-1 — the move is scripted, not retyped.** The `What` cells run to 6 KB and a
hand-retyped 40 KB is where a paraphrase gets in. A throwaway `python` script
splits the table on unescaped `|`, writes each cell into the archive under its
hash, and the result is compared back against the pre-story file before the
ledger is rewritten. (`python3` on this machine is the Windows Store stub and
exits 0 having done nothing; use `python` and check the file.)

**DD-2 — a fold is a marked paragraph appended inside the requirement, not a new
section at the end of the spec.** A clause the reader must apply to a
requirement belongs on that requirement; a "folded hotfixes" appendix would be a
second place a contract is stated, which is the failure `WORKFLOW.md` S-7 names.
The marker form, verbatim, is one line:

> `*Folded from hotfix `<sha>` — see `docs/hotfix/ARCHIVE.md#<sha>`.*`

**DD-3 — the commit check is the script's reason to exist and it runs from `git
log`, not from a file.** The three file measurements would have caught the
formatting drift; only the commit check catches a hotfix that never got a row at
all, which is the failure the ledger exists to prevent and the one a human
command was trusted with.

## The fold table

`amend` appends DD-2's marker plus the rule to an existing id's prose. `NEW`
mints an `FR` and also adds one accounting line to that story's `plan.md` and one
to its `tasks.md` (FR-6). No `AC`, `P`, `DD` or `SC` is minted anywhere, and
nothing is removed (P-1).

| hotfix | target | the rule the spec gains |
|---|---|---|
| `9729459` | `0083` FR-2 amend | The decrease is measured against the health the front end ALREADY HELD, and no figure exists at all when that already-held health was at or below zero — the killing blow still shows, a corpse's decay never does. |
| `c5274d7` | `0020` FR-4 amend | The scripted schedule is test support only: no front-end path wires it, and both front-end paths are pinned against its absence. The wiring plan `DD-2`/`DD-3` describe is void. |
| `2ad4c29` | `0017` FR-11 amend | The diagnostic marker ships OFF in the game and `-markers` brings it back; `cmd/mapview` is a developer's window and its defaults do not move. |
| `2ad4c29` | `0060` FR-7 amend | On by default is the VIEWER's default, not the game's: the game's two front-end call sites open the readout hidden and `F1` (FR-8) shows it. The switch is thrown at the call sites, so what is asserted is where. |
| `65dfdf7` | `0093` FR-9 **NEW** | The campaign drive reports, per tick rather than per snapshot, every unit that changed cell, with its roster slot and its group. |
| `7e4245c` | `0038` FR-4 amend | The constant is **0** and the map's margin is black (owner as author, 2026-08-01). The strictly-between bound, the multiply-so-relief-survives decision that story's own plan records, and its acceptance criterion's reading of both are void rather than retuned. |
| `eef792f` | `0103` FR-20 amend | EVERY path that rebuilds a mission's world carries the map's ground sacks, not the widest load path alone. |
| `e09d2bb` | — | NOT OWED: the rule is `AGENTS.md`'s existing one, that lane state lives in the lane's return text or in that story's own `docs/<NNNN>/`. No spec owes a clause. |
| `872487b` | `0110` FR-6 amend | The window's rectangle has ONE spelling, and a press, a release or a gesture latched inside it reaches neither the command path, the selection nor the camera. One pixel outside is the map's: the world runs beneath and this is not a modal. |
| `7e3176c` | `0119` FR-22 **NEW** | The generation screen states what a point buys: six derived lines under the footer, computed by the same expression the party mint uses, absent entirely while the spread is illegal. The screen takes them as a field on its setup, resolving nothing itself. |
| `c9d0a2c` | `0123` FR-1 amend | The drop set is the container AND the base actor's two weapon slots, unequipped second then first and appended at the tail, all-or-nothing; the humanoid armour fields stay on the corpse. A unit whose loadout the loader withheld leaves nothing of it. And a weapon suitable for no consumer class is not an item: the loader composes no code for it, so an innate attack never enters the equipment set (`026e936`). |
| `c9d0a2c` | `0123` FR-2 amend | *Is this a holding* and *would this body drop* are two questions asked by name; a body wearing only what it cannot drop plants no sack, empty or otherwise. |
| `c9d0a2c` | `0124` P-4 amend | Closed: a unit's class-row weapon reaches the simulation as part of the loadout the loader composes, so the first equip displaces a starting weapon into the pack instead of superseding nothing. |
| `c9d0a2c` | `0112` FR-1 amend | The record states a starting loadout as well as a container, and every world rebuild names it — the field set a rebuild must carry is the record's, not the caller's memory of it. |
| `637e513` | `0118` FR-7 amend | The rule is the SET: every consumer of the entity snapshot on a draw path asks the gate, not the two call sites that first did. Panels and what a press may target are outside it and stay so. |
| `dbd5c89` | `0066` FR-11 amend | A dismissed win names the mission that follows it, and the party that survived is carried into that mission rather than re-minted. |
| `dbd5c89` | `0125` FR-13 amend | A minted member may be seeded from carried exact experience values as well as from levels, and the exact value is carried because the level's inverse floors. |
| `b55f111` | `0125` FR-14 amend | The credited slot follows the loadout: it is written by the same recompute that writes the combat numbers, through the same door, all-or-nothing behind that door's own bounds guard. |
| `b55f111`, `ea870ef` | `0124` FR-11 amend | The recompute's consumers are the whole derived set the equip moves — the combat numbers, the credited experience slot, and the panel's weapon NAME, which is written at the equip and is neither stable nor overlaid. |
| `4ac0744` | `0126` FR-5 amend | A swing is voiced only when the attacker could reach what it is swinging at. The cycle itself is not narrowed and nothing hashed moves; `AC-14`'s count is read under that gate. |
| `29c2f77` | `0126` FR-1 amend | A grunt is gated on the health the front end already held and is refused when that health was at or below zero, so a decaying corpse is silent while the killing blow still sounds. The floor alone was never enough (FR-2). |
| `8e13e8f` | `0111` FR-7 amend | Paint order within a row is three-way and lexicographic on (row, tie): a living unit above, the sack under it, the corpse under the sack (owner as author, 2026-08-09). Plan `DD-2`'s equal-row tie-break is void; the ground tier is the zero value. |
| `ea870ef` | `0113` FR-17 amend | The derived set the window states includes the name of the weapon in hand, written at every equipment change and not once at the open. |

## Risks

The gate is the risk and it is measured, not assumed: `check-sdd-audit.sh` fails
a story whose `spec.md` names an `FR` its `plan.md` or `tasks.md` does not, and
several target specs sit within a few hundred bytes of their `check-doc-budget`
ceiling (`0119` at 13298 of 13312). Where a fold does not fit, the ceiling is
raised by a **declared overrun** in `select_ceilings` with its reason beside it,
measured first — never by cutting contract text to make room.
