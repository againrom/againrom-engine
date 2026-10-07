# Provenance — 0119-chargen

Research facts come from the `research/` submodule at this story's pin. Claims are cited by id;
no experiment folder is cited.

## What the research says, and what this story took from each

| Claim | Confidence | What this story takes |
|---|---|---|
| `SESS-HERO-014` | High | A generated character is **not invented**: the appearance byte resolves one of four shipped `Humans` rows by NAME — `PC_Danath`, `PC_Naira`, `PC_Fergard`, `PC_Reniesta`, rows 26–29, resolving 4/4 — and the row is looked up by name in the `Data.bin` `Humans` collection. The same routine re-derives the point-buy budget as `0x8c` minus the four costs. The sex is carried in the name's own suffix (`.m` / `.f`) and parsed back off it; the face index is the suffix's decimal tail. |
| `HERO-BUDGET-004` | High | `ΣT(stat) ≤ 140`, the accepting side's own test, and the client's `4·T(25) + 100` identity for the same number. Already implemented in `pkg/data/chargen.go` by 0113; this story consumes it and adds nothing to it. |
| `HERO-STAT-001` | High | Four statistics and only four, named from the registry's own key literals: Body, Reaction, Mind, Spirit, in that display order. The screen's four rows are these, in this order. |
| `HERO-CLASS-013` | High | `typeID = gender + 0x23` when `+0x4c` bit 2 is set and `gender + 0x21` when it is clear — **the constructor's addend/base split**, which is what lets a shipped row's own type id be decomposed into a class flag and a gender. Also: the class bit picks the `chrgen\fighter\` vs `chrgen\mag\` skill column, and the derive's two class-conditional edges hang off that same bit. |
| `HERO-SKILL-009` | High | Six slots; five shared, renamed by class, plus a `General` slot the sheet does not show. The five carry both names in one shipped string — `Skill.Blade (Fire)` — which is where the screen's mage column comes from. Generation **sets exactly one slot**, zeroing 1..5 first, at level 20 or 10; the ordinary arm is 10. |
| `HERO-HP-005` | High | The health maximum's three truncated steps, and that the first arm is skipped entirely when the `HealthMax` column is 0. Implemented by 0113; this story wires its output. |
| `HERO-MP-006`, `HERO-ORDER-014`, `HERO-CAP-015`, `HERO-XP-010` | High | The mana pool, the two ordering points, the statistic cap and the level/experience pair. All already implemented in `pkg/data/recompute.go`; cited because the numbers this story puts on an entity are theirs. |
| `HERO-APPEAR-045` | High (scoped) | **Of the two axes the sex bit is the one that does NOT reach the drawing.** Both routines that turn a drawable into pixels were read end to end and neither tests it, so the body name, the directory, the sheet pair and the drawn class id are independent of sex. This is why the map sprite does not change with the sex choice, and it is the opposite of what this lane was briefed. |
| `HERO-APPEAR-042`, `-043`, `-046` | High / Medium | A player's character is drawn as whatever its visible equipment says, through a seventeen-arm name chain, and a hero's pixels do not come through `units.reg`'s File. The tree already implements this (`data.HeroBodyFor`, 0105); it is cited because it is why the **skill** choice changes the map sprite — a different trained slot is a different starting weapon is a different body name. |
| `HERO-APPEAR-051` | High | The four shipped figure directories and the addresses under them. `data.FigureDirFor(mage, female)` already implements it; this story stops passing constants into it. |
| `REG-UNITS-061` | — | Named only inside `HERO-APPEAR-045`'s reasoning; nothing here reads it. |

## What is ours by choice, and why

1. **The screen's layout, its labels and its key bindings.** Nothing decoded describes a screen we
   can draw, and the engine's debug font renders no Cyrillic byte. Authored, disclosed in `spec.md`,
   English throughout like the rest of this tree's chrome.
2. **Which gender addend is female.** `HERO-CLASS-013` establishes the addend exists and research
   states outright that *which* of the two values is male "is not established and is not needed". One
   named constant, one edit wide.
3. **The default axes when the flag is absent.** Today's hero is a male fighter trained in Blade at
   43/26/15/15 (`pkg/game/hero.go`, 0113's authored verdict). Unchanged, and now spelled as a
   generation result rather than as three unrelated constants.
4. **The fallback when the four rows do not resolve.** An install this tree misreads must still
   start a mission; it falls back to `SpawnHP` and no pool, which is exactly today's behaviour.

## What is open

- **Which name goes with which (class, sex) pair.** `SESS-HERO-014` names the four arms and the two
  bits but publishes no mapping. This story does not need one: it reads each row's own type id.
- **Whether the flag the pool graph multiplies by means "fighter" or "mage" in English.**
  `HERO-CLASS-013` calls it the fighter/mage flag and `HERO-HP-005` writes the arm as
  `fighter ? 2 : 1`; but `R0842` ORs that same bit when the actor has a spellbook and nonzero
  mana, which reads as *mage*. `pkg/data/recompute.go:44` already refuses to decide it. **This story
  also does not decide it** — it reads the bit off the shipped row, so whatever the shipped rows say
  is what a generated hero gets. A later story that resolves it inverts one function.
- **`logBase11`'s margin was unmeasured** and `recompute.go` named the measurement as a prerequisite
  for wiring the pools. This story measures it; the figure is in `verification.md`.

## What was removed

Nothing. No claim used here is retracted or amended against this story's reading; `SESS-HERO-014`
carries a *Noted, not resolved* about a restore path that this tree has no save format for.
