# Provenance — the hero is a real fighter, and his numbers are on screen

Every research fact below is cited by **claim id** at the pin recorded in the submodule (research
`414bc29`). No experiment folder is cited: a claim carries its own amendment and retraction state, an
experiment cannot tell you it has been superseded.

The pin moved from `20921e2` to `414bc29` under this story, at the `0079`/`0080` boundary, and every
row below was **re-checked against the new pin rather than carried forward**. All fourteen are still
`● active`. One carries a supersession that matters and is recorded in its own row.

## What the contract rests on

| spec anchor | id | confidence | what this story takes from it |
|---|---|---|---|
| FR-1 | `HERO-COST-002` | High | The whole of the cost function: `T(n) = ftol(0.349 x pow(1.15, n-1) + 0.5)`, the base an inline immediate and `pow`'s first argument, the two constants named memory operands, the truncation a named call. It is **cumulative** — the total cost of a statistic standing at `n` — and a step is its first difference, which is what makes the refund exactly symmetric. Its enumeration is complete at 8 hits / 3 owners / 0 orphan, so no second cost function exists on any enumerated route. |
| FR-2 | `HERO-BUY-003` | High | The two click bounds as the `CMP` operands themselves: a `+` refuses at `v >= 45`, a `-` refuses at `v <= 15`. The start — four stats at 25, pool 100 — and that the counter on screen counts what **remains**. Nothing depends on class, race or the other three statistics. |
| FR-2 | `HERO-BUDGET-004` | High | The accepting side's test `sum T(stat) <= 140`, the literal `0x8c` plus four subtractions and a sign test. And the identity `4 x T(25) + 100 = 140`, which makes the client's pool and the server's budget one number seen twice. |
| FR-2, FR-3 | `HERO-SKILL-009` | High | Six slots; the order `General, Blade, Axe, Bludgen, Pike, Shooting`; generation zeroes 1..5 and writes **exactly one**. The five names this story prints are the shipped `Data.bin` column titles' own warrior halves. |
| FR-3 | `HERO-START-039` | High for the dispatch and the literals / **Medium** for *a shipped campaign takes the skill-10 arm* | The ordinary arm's blade literal `Iron Short Sword` and that generation hands exactly one weapon, chosen by the one trained slot. The Medium is **carried, not resolved** — this story inherits `0078`'s choice of the ordinary arm and changes nothing about it. |
| FR-3, FR-5 | `HERO-STATDMG-036` | High | Only Body reaches damage; only Body and Reaction reach to-hit — a positive read census inside the sole writer. This is the whole of why a fighter's points go where they go, and it is the fact that makes the flat 25/25/25/25 wrong for a fighter rather than merely dull. |
| FR-2 | `HERO-CAP-015` | High | `stat = min(stat, 50 + (int8)modifier)`, the 50 an immediate, not class-conditional. Recorded to fix that it is a **different bound** from generation's click cap and that it does not bind any legal spread — the highest a spread can reach is 43. |
| FR-5 | `HERO-DERIVE-034`, `HERO-FOLD-035`, `HERO-BARE-037` | High | The fold this story does not touch, cited so that "the numbers move because the inputs moved" is a statement with a source. |
| FR-6 | `UNIT-PANEL-010` | High for the copy map / its **scope clause is SUPERSEDED** | The **value set** copied for the original's unit information display: the four statistics, health, mana, to-hit, defence, absorption, the damage pair, speed, sight, five resistances and five protections. This story shows the subset of that set for which this tree holds a real value, and adds nothing outside it. **What was superseded is the part this story does not use**: EXP-0075 established that the row's three-way health/to-hit/defence arithmetic is **not a display rule** — the `.alm` spawner runs it on the actor at placement (`UNIT-GATE-013`) — so the copy map, the key and the arithmetic were right and only the scoping was wrong. The consequence runs in this story's favour and is the reason it is recorded rather than noted: because the scaling happens at **placement**, an entity's own numbers are already the scaled ones, so a panel printing them verbatim is correct. Under the superseded reading this story would have owed a display-time gate. It implements none, and now owes none. |
| FR-6, FR-7 | `UNIT-PANEL-011` | High as a negative about the instrument | That the **layout** of that display cannot be settled: past `+0x14a` the block is addressed by computed index, so no displacement sweep can name a consumer. This is what licenses an authored layout as a **verdict** rather than a hold — the question is closed against the instrument, not open. |
| FR-7 | `HERO-STAT-001` | High | The four statistics' names come from the game's own registry key literals, and the chargen panel reads them back in **display order** `0x138, 0x13b, 0x139, 0x13a` into rows 0..3. That order — Body, Reaction, Mind, Spirit — is the panel's row order here, and it is the one part of the layout that is not ours. |
| FR-6 | `HERO-SHEET-038` | High | The sheet's damage line is `%d-%d` over `base` and `base + spread`, the two zero-extended and added before the push. The panel's damage row uses that composition and no other. |

## Ours by choice

| what | why it is ours, and what would overturn it |
|---|---|
| **The point spread — Body 43, Reaction 26, Mind 15, Spirit 15.** | Nothing decoded says what the owner's character is. The budget bounds the space; it does not pick a point in it. Overturned by the owner saying so, and by nothing else. The rule that produced it is `plan.md` DD-2; the seam that holds it is DD-3. |
| **The one point left in the pool (139 of 140).** | A consequence of the rule, not a separate choice: no legal spread with Body 43 can raise Reaction past 26, because the next step costs 2 and 1 remains. Spending it on Mind or Spirit would buy a point no blow reads. |
| **Which skill slot is trained.** | `0078`'s choice, unchanged and re-disclosed: Blade, because the owner's measured bands were taken on it. |
| **The panel's rows, labels, order, colours and corner.** | `UNIT-PANEL-011` establishes positively that no layout can be recovered. The row order of the four statistics is the exception and is cited above. The labels are English and are ours; the original's are Russian and come from a registry this tree does not read for text. |
| **Which of a skill's two names is printed.** | `Data.bin` gives each of the five a warrior name and a mage name in one string. This tree has no class axis, so the warrior half is printed. A class axis is what would settle it. |
| **The wording and field order of the headless check line.** | A development instrument. It asserts nothing about the game. |

## Open — undecoded and deliberately assigned no meaning

- **Health, mana, speed, sight, capacity, reach and the two runs of five** are derived from these
  same statistics by the original and are not derived here. The panel states none of them, which is
  why flooring Mind and Spirit costs this tree nothing today and would cost a later story something.
- **The point-buy screen.** The cost function is implemented; nothing spends a point at runtime.
- **Whether a shipped single-player campaign takes the ordinary generation arm.** Research grades it
  Medium; this story carries the grade exactly as `0078` did and resolves nothing.
- **The original's own panel layout.** Closed against the instrument, not against the game.

## Removed

Nothing. No statement of an earlier revision was dropped.
