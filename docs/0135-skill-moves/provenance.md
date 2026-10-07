# 0135-skill-moves — provenance

The evidence behind the contract, by spec anchor. Nothing here is normative.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1, FR-2 — a level is its own stored word, not a display of the experience | `HERO-SKILLUP-073` | High |
| FR-3 — `xp[i] = S(skill[i])` exactly at creation, and only there | `HERO-XP-077`, `HERO-SKILLUP-073` | High |
| FR-4 — a level and its experience are two independent carried values | `HERO-XP-077` | High |
| FR-5.1 — a class that trains nothing trains nothing (the plain actor's three thirteen-byte stubs) | `HERO-SKILLGATE-074`, `MAGIC-TRAIN-018` | High |
| FR-5.2 — the two source refusals, and that they are skipped for a source already dead | `HERO-SKILLGATE-074` | High |
| FR-5.4 — the level must be under 100, tested **before** the award | `HERO-SKILLUP-073` | High |
| FR-6 — Mind multiplies every gain, `ftol(amount × (mind/30 + 0.25))` | `HERO-SKILLGATE-074` | High |
| FR-6 — one award capped at `S(n+1) − S(n)` | `HERO-SKILLGATE-074`, `HERO-SKILLUP-073` | High |
| FR-7 — the class bit decides the slot; a book carrier's blow is dropped; slot 0 is raised by nothing | `HERO-SKILLGATE-074`, and `HERO-KILL-027`'s amendment | High |
| FR-7 — the class bit is the mana column being positive | `HERO-CLASS-013` (as overturned by `EXP-0133`, which is the current text) | High |
| FR-8 — strict `xp > S(level)`, `ADD DX,0x1`, no loop | `HERO-SKILLUP-073` | High |
| FR-9 — the blow feed, its amount and its own two caller gates | `HERO-KILL-027`, `HERO-SKILLGATE-074` | High |
| FR-10 — the cast feed, `round(manaCost/2)`, credited to the spell's `Sphere` | `MAGIC-TRAIN-018`, `HERO-SKILLGATE-074` | High |
| FR-11 — `power = clamp(skill[school] + Mind − 30, 0, 100)` | `MAGIC-POWER-004` (formula clause, not the retracted enumeration) | High |
| FR-12 — a raise re-runs the whole derive | `HERO-SKILLUP-073` | High |
| FR-14 — the level is read, not computed, wherever one is shown | `HERO-SKILLUP-073` | High |
| `S(n) = ftol((1.1ⁿ − 1) × 1000)` | `HERO-XP-077` | High |
| The weapon skill in hand is the weapon's own `@.attackType` | `HERO-EQUIP-017` | High |

`HERO-XP-010` is **retracted twice over** and nothing here rests on it. Its
headline — experience is a function of the six levels — is the model this tree
shipped, and correcting that model is what this story is. The two tables `S`
and its inverse survive the retraction and are restated in `HERO-XP-077`.

## Ours by choice

| Statement | Why it is ours |
|---|---|
| FR-6's cap reads the **credited** slot's level, not the slot the award named | `HERO-SKILLGATE-074` places the cap before the slot resolution, which would read the *named* slot; `HERO-SKILLUP-073` states the cap's purpose as "no single event can carry a slot up two levels", which only holds on the credited slot. The two readings differ for a fighter, whose named slot is always 0. The second is taken because it has an argument behind it and the first has only an ordering. |
| FR-10 supplies the **victim** as the cast's source actor | No claim states what `vt+0x68`'s caller passes. The damage feed passes its target, and the diplomacy refusal is meaningless without one, so the same value is passed here. |
| The integer form of Mind's scaling, `(4·mind + 30)/120` | 0125 DD-6's, unchanged and re-used rather than restated. |
| `S(n)` as a hundred-and-one entry integer table inside `pkg/sim` | The curve needs `pow`; `pkg/sim` bans floats. P-1 is what keeps the table and `pkg/data`'s own curve from parting. |
| FR-13's wording, geometry and lifetime | The original draws no such row at all; the log it is posted to is `0124`'s own authored surface. |
| Which of the derive's outputs FR-12 can actually move | Named in the spec as a limitation rather than left implied. The reason is this tree's, not the game's: one setter exists onto a live entity and it carries ten numbers. |

## Open

- The **kill feed** `vt+0x60`. Its amount is decoded (`ftol(victim+0x1c × 0.5)`)
  and its slot rule is the same as the other two, but **no claim names its call
  site**. Wiring it would be authoring a trigger, so it is out of scope.
- The **more-than-one-participant** fifth of the cap. `HERO-SKILLGATE-074`
  grades the flag's *meaning* Medium, and this tree has no session object to
  read it off. The cap is therefore always the full `S(n+1) − S(n)`.
- The original's own **experience-to-level** routine, which the loss path runs.
  Still unread, and no longer needed: nothing in this tree converts an
  experience to a level once the level is stored.
- The **loss at death** and the **purchase for gold** (`HERO-SKILLLOSS-075`,
  `HERO-SKILLBUY-076`), both decoded and both out of scope.
- Whether `Skill.General` has any producer at all. `HERO-SKILLBUY-076` says the
  purchase is the only one, and its own reachability is graded Unknown.

## Removed

| Dropped | Why |
|---|---|
| A second refusal reproducing "the killing blow is exempt from the two source refusals" | Decoded and true, but it reverses `0125` FR-5.4/FR-5.5 on exactly the blow that matters most, for no part of what this story is for. Named in the spec's Out of scope instead of built quietly. |
| A `Fighter` flag on the placement path as the class discriminator | `HERO-CLASS-013`'s provenance clause is retracted: the bit is derived from the mana column at spawn, not streamed, so the tree's existing `MaxMana > 0` predicate **is** the decoded discriminator and a second field would be a second answer. |
