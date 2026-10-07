# Provenance — 0090

Submodule pin `fe1c626`, moved for this story from master's `96f0b15` because the round that traced
both of the predicate's input grids landed after the fork.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 — one writer, run once at world construction, from geometry and one shift alone | `AI-LOS-087` | High |
| FR-1 — the window is 41x41 and the shift ships as 7 in both roots | `AI-LOS-087`, `TERR-SIGHT-115` | High |
| FR-2 — the step grid holds a signed column/row delta pair naming one Bresenham step toward the centre | `AI-LOS-088` | High |
| FR-2 — the three zones and their two comparisons | `AI-LOS-088` | High |
| FR-2 — the two axis offsets that need the builder's four trailing literal stores, and that exactly two cells fail the chain without them | `AI-LOS-087` | High |
| FR-2 — every non-centre cell's predecessor is strictly closer, measured over the whole window | `AI-LOS-088` | High |
| FR-3 — the cost expression, its FPU sequence and its truncation toward zero | `AI-LOS-088` | High |
| FR-3 — the 128..181 bounds at the shipped shift | `AI-LOS-088` | High |
| FR-4 — the seed written into the accumulator's centre cell | `AI-SIGHT-006`, `AI-LOS-088` | High |
| FR-5 — the four-term recurrence and the order of its terms | `AI-LOS-081` | High |
| FR-5 — the height plane is read signed | `AI-LOS-081`, `AI-LOS-090` | High |
| FR-6 — visible iff the value is positive; the value is stored before the branch; the ring runs on | `AI-LOS-081`, `AI-LOS-090` | High |
| FR-6 — a later cell continues from a blocked predecessor's stored value | `AI-LOS-090` | High |
| FR-7 — rings outward, the `r <= 19` bound, and the stop when a whole ring is blocked | `AI-SIGHT-006`, `AI-LOS-089` | High |
| FR-8 — the observer's own cell is marked by the walk itself | `AI-SIGHT-006` | High |
| FR-9 — the walk tests each ring cell against a rectangle inset 8 cells from every edge | `TERR-SIGHT-116` | High |
| FR-10 — the map is cleared once, every member stamps into it, the union is what is swept | `AI-GROUPSEE-068` | High |
| FR-11 — the candidate sweep keeps every actor whose own cell byte is non-zero | `AI-GROUPSEE-068` | High |
| AC-3, AC-4 — the flat-ground counts and the axis/diagonal reach | `AI-LOS-089` | High for the closed forms; the counts are re-derived here from them |
| AC-7, P-3, D-4 — the 128-vs-127 margin, and that no re-lighting occurs on shipped data | `AI-LOS-090` | High for the arithmetic; **Medium** for the corpus census behind the negative |
| D-1 — the sight range is a `Data.bin` column landing in the byte the stamp reads | `DAT-HUMANS-008`, `HERO-SIGHT-007` | High |
| D-1 — the actor constructor's default is 5 | `AI-SIGHT-006` | High |
| D-3 — the byte-width compare against a 32-bit index, and how often it bites | `TERR-SIGHT-116` | High for the wrap; **Medium** for the four hit counts |
| D-5 — the shift has two sources and the two agree only because the shipped value equals the code default | `TERR-FOG-088` | High |
| Out of scope — the drawn fog is a second implementation on a different object | `TERR-FOG-088`, `TERR-FOG-080` | High for the two objects; **Medium** that the two are the same algorithm |
| Out of scope — the fourth grid built beside these two is a disc test, not this predicate | `TERR-SIGHT-115` | High |
| Nothing else reads or writes either grid | `AI-LOS-091` | High |

## Ours by choice

| Decision | Why it is ours |
|---|---|
| FR-1 — the window is a process-lifetime table rather than per-world derived state | The law rebuilds it per world because it hangs it off a world object; it depends on nothing a world carries, so a per-world copy would be a field no world could vary. |
| FR-3 — the cost is computed with integer arithmetic over an integer square root | The published expression is floating point. `pkg/sim` is held to no float by a source scan. The two forms were checked equal at every window cell before the integer one was adopted. |
| FR-9 — the inset is read off the grid's air bit instead of computed from the bounds | Named as D-2. The two coincide on every loaded map; the grid form leaves a world naming no grid uninset, which is what keeps the predicate meaningful on a synthetic world smaller than the inset. |
| FR-9 — the bounds test is exact rather than byte-wide | Named as D-3. Reproducing the law's width would mean lighting a cell on the opposite edge of the map that no ray reached. |
| FR-10 — one march per member, unioned, rather than a shared array stamped into | Same set, and the union is what the sweep reads. The law's single cleared array is a storage choice its own per-actor caller shares. |
| The march allocates its stamp per group per decision and stores nothing | The law clears a fixed array per use for the same reason: nothing about a stamp survives the decision that built it. |

## Open

| What | Why it is left open |
|---|---|
| A per-unit sight range | Decoded on both bands and already carried by `pkg/data`, but reaching `pkg/sim` with it is a byte-form change and a form version this story may not take. D-1. |
| The shift as a registry read | This tree reads no registry at all. D-5. |
| Whether the flat-ground region at range 6 is 145 cells or 127 | Research has deliberately not adjudicated the two implementations' figures and grades "same algorithm" Medium. Nothing here depends on it: the contract is derived from the server side's own closed forms and 145 falls out of them. |
| The remembered attacker's cell forced into the stamp for twenty decisions | 0086 D-5 already establishes that no world this build can construct can tell the difference; nothing changes that. |
| What the per-player mask ORed into the fourth grid is for | Its two consumers were placed and not read. Out of scope and not needed by anything here. |

## Removed

| Statement | Why |
|---|---|
| 0086 D-1's *"the law's region is a subset of this one and never a superset"* | False as stated. The per-cell term is the observer's altitude minus the cell's, and descending ground returns budget, so the region reaches past the flat radius: over the shipped corpus at range 6 the count runs to 1 248 against a 145-cell flat baseline (`AI-LOS-089`). The subset relation survives only on flat ground, where it is kept as P-4. |
| The comment at `sightRadius` claiming the field's complete writer set is two instructions | Its own source's sweep excludes a store made through a helper that carries no displacement, and `DAT-HUMANS-008` puts a database column in that byte. The value is unchanged and the reason for it is not. |
