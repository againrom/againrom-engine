# provenance — 0133

## Backing

| Spec anchor | Source | Confidence | What it carries |
|---|---|---|---|
| FR-1 | `HERO-HP-071` | High | The health column is destroyed one instruction before the arm's own test and nothing recovers it: the clamp reads the field that still holds it, and the constructor's last act overwrites that with the derived maximum. Over the 215 shipped rows, six pairs agree on Body, mana and all six skill columns while differing in the health column, and none of the six derives a different maximum — so the corpus discriminates rather than merely agreeing. |
| FR-1, FR-3 | `HERO-HP-072` | High | The derivation is the actor's own virtual slot and runs at creation; the spawner calls it a second time after its statistic overrides. The current health is assigned from the maximum before the constructor returns. |
| FR-2 | `HERO-HP-072` | High | The order and every term: capped Body, the summed skill experience, the class multiplier, the logarithmic experience term, the exponential growth term, each intermediate truncated and stored as a word. This is the graph this tree already carries. |
| FR-4 | `HERO-HP-071` | High | The store and the test are two instructions apart on the same field of the same receiver, so the branch tests the product and not the streamed column. The mana arm nine instructions later reads its column *before* overwriting it and is genuinely column-gated — the asymmetry is what the correction rests on. |
| FR-4 | `HERO-HP-072` | High | The jump the test takes skips the remaining two terms, not only the first, so a zero product yields a maximum of zero rather than a maximum built from experience alone. |
| FR-5 | `HERO-HP-072` | High for the condition and the direction; Medium for the naming | The doubling belongs to the arm whose class bit is **clear**, and on the placement path that bit is set exactly when the mana column is positive. Which archetype the bit names is the Medium half, and `spec.md` does not depend on it — FR-5 states the condition, not the name. |
| FR-5 | `HERO-CLASS-013` | High (amended; one clause superseded) | The two class-conditional edges inside the derivation and their inverted directions across the two pools. Its clause calling the class field a streamed copy is superseded: the field is derived at spawn from the mana column. |
| FR-4, FR-5 | `HERO-HP-072` | High | At placement the statistic ceiling is a flat 50 and the health addend 0: the modifier block is cleared at construction and nothing on this path writes it. That is why this story needs no modifier term. |
| Out of scope — statistic overrides | `UNIT-PLACE-034` | High for the mechanism; Medium for the file-offset attribution | The placement record carries four statistic bytes the spawner applies before re-deriving. A nonzero Body override appears on one person-arm record in either root's whole corpus, on a map outside the campaign, so a consumer omitting it is right about every campaign placement. |

## Ours by choice

| What | Why it is ours |
|---|---|
| The tolerance of the gate's test — a product of exactly zero, rather than a non-positive one | The image sign-extends the stored word and branches on zero. No shipped row carries a negative Body, so the two readings cannot be told apart from the corpus, and the narrower one is taken. |
| Keeping the column flag on a character's profile, read by the party mint alone | The flag is no longer the health arm's gate. It remains a true statement about a row, and the party mint needs it to tell a generated character with no row from one built off a shipped row. |
| The person collection census and its four figures | No source publishes a report shape. The figures are chosen to be the ones that would fail loudly if the arithmetic were wrong. |
| The owning roster slot on the per-placement line | A reporting choice; the field itself is the map's own. |

## Open

| What | State |
|---|---|
| A person's mana maximum at placement | The same routine derives it, gated on the mana column, and this tree still streams the column. Deliberately left as it is; `spec.md` names it out of scope rather than asserting the column is right. |
| Which archetype the class bit names | Undecided by any source. `HERO-HP-072` grades this Medium explicitly and the contract avoids depending on it. |
| The experience-to-level direction, and the health addend's own writers | Outside this path: the addend is zero at placement, established positively rather than assumed. |

## Removed

| Statement | Why it is not in the contract |
|---|---|
| "The health column gates whether a person derives at all" — the clause of `HERO-HP-005` that this tree implemented | Retracted. `retracted.md` records it as **REFUTED**: the column is neither the value nor the gate, and a consumer holding it is wrong about 214 of the 215 shipped rows, by up to twenty-fold on one of them. |
| "The class field is copied out of the row" — a clause of `HERO-CLASS-013` | Retracted as **SUPERSEDED**: the field is derived at spawn from the mana column, so it cannot be set independently of mana. This story's FR-5 states the derived condition instead. |
| The published corpus totals — the two collection sums, the largest ratio and the row carrying it | They are research's measurement, not ours. `spec.md` asks the tool to produce our own; `verification.md` is where the two are compared. |
