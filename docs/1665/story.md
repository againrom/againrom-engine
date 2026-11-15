# Computer mages heal and strengthen their own side

## Result

In both games a computer mage with Heal in its book now keeps its side going between fights. With more than
ten mana it heals itself when hurt, else the first hurt unit near it, and it spends no mana reserve on that.
With nobody hurt and a full mana pool it picks one unit of its own side and casts a protective spell from its
book on it, unless that spell already holds there. In the first game, when no such spell is left for the
picked unit, the mage casts nothing; it never casts Slow on its own unit.

## Authority

- First game: AI-785 and AI-786 (High body, Medium population), AI-788, MAGIC-237.
- Second game: R2-ENGINE-1244 to R2-ENGINE-1248 (High / Medium; 1245 and 1246 Medium).
- Owner, board card o-rom1-slow-empty-draw: «Не переносить» (DIV-4951).
- DIV-4830 and DIV-4831: an unbidden Bless never chooses a mage. DIV-4756: whom a second-game computer heals.
- DIV-331: a Player's healing mode keeps its own heal, which runs before the support sequence.

## Rules

1. A computer mage without Heal in its book, or with 10 mana or less, casts no support spell.
2. With Heal and more than 10 mana it heals itself when hurt, with no mana reserve kept.
3. Else it heals the first hurt unit in Heal's 6-cell area list: its own cell, then ring by ring outward.
4. Else it draws one unit of the list; a unit of another Player takes nothing.
5. With the caster at full mana, a drawn own unit takes one affordable Defensive spell but Heal, at equal weight.
6. A drawn spell whose effect already stands on the unit is not cast.
7. First game: when the draw keeps no spell, the mage casts nothing, and never Slow (DIV-4951).
8. Second game: self Heal and self Shield come before the draw; no support routine chooses Slow.
9. DIV-4830, DIV-4831, DIV-4756 and DIV-331 keep their behaviour where the sequence meets them.

## Code

| kind | its one builder | change |
|---|---|---|
| computer support sequence | `pkg/sim/autocastsetting.go` `supportCast` | policy data `drawsOne` and `shield`; `supportDraw` |
| computer support policy | `pkg/sim/gameprofile.go` `gameProfile.computerSupport` | first game gains a policy; `selfOnly` gone |
| support candidate list | `pkg/sim/autocastsetting.go` `autoCastCandidates` | area-list order through `walkCellRings` |
| automatic cast admission | `pkg/sim/autocastsetting.go` `autoCastAdmits` | one admission read by `autoCastAt` and the draw |
| unbidden heal order | `pkg/sim/spell.go` `autoCastOrder` | a computer's heal is the sequence's unless a mode is set |

## Proof

| test | rules |
|---|---|
| `TestComputerMageBuffsADrawnOwnUnit` | 4, 5 |
| `TestFirstGameComputerHealKeepsNoReserveAndTakesTheAreaListOrder` | 2, 3 |
| `TestFirstGameComputerEmptyDrawCastsNothing` | 6, 7 |
| `TestComputerMageSelfShield` | 1, 8 |
| `TestComputerMageHealsItsOwnSideAndAllies`, `TestComputerMageHealsItself` | 2, 9 |
| `TestAnAutomaticBlessSparesEveryMageAndTakesAWarrior`, `TestCastRequestRecord` | 9 |

Release tests at the release: none.

## Open debt

- Cadence is Unknown in both games (R2-ENGINE-1244): the engine's current attempt timing is kept (DIV-4630).
- Which Players the list keeps and whether the caster can be drawn are Medium (DIV-4629).
