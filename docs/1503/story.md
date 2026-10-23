# One shop stock generator for both games

## Intent

Both games' town shops draw their stock through one generator, `shopstock.Generate`
(`pkg/shopstock/stock.go`), in the second game's shape: a location's categories in draw order;
per category its candidate list, its lead items, its draw count, then the draws, each making an
item of one picked candidate (the enchant is part of making it); then the items placed without a
draw; then the count multiplier's further rolls. Each game states its rules as a
`shopstock.Profile`. Behaviour is unchanged in both games: the same stock item by item and in the
same order, and the same draws from the random stream.

## Authority

- Owner decision on audit `pipeline/reviews/arch-audit-29d32509.md`, rank 7 (shop stock
  generation): one stock generator in the second game's shape; the first game is profile data
  (shelves, pools from the definition table, ceiling, draw source); the count multiplier and the
  effect-repeat rule are shared steps the profile switches on and parameterises.
- Owner ruling that the first game is a reduced second game, and the seat rule "One builder per
  kind". Refactor only; no ROM fact moves. Each game's pools and enchant tables keep their own
  claims (first game SHOP-GEN-005, SHOP-POOL-006, SHOP-MAGIC-007, SHOP-CONSUME-073; second game
  R2-ENGINE-263..267).

## As built

### The generator

`pkg/shopstock`:

- `Source`: one draw over n choices (`Intn`).
- `Category[C, I]`: `Candidates`, `Lead`, `Count`, `Bound` (first roll) and `Again` (each
  further roll), `Redraw` and `Redraws`, `Make`, `Insert`, `Finish`.
- `Bound`: `Attempts` bounds every attempt, `Failures` the failed ones; each is a function of the
  draw count; nil is no bound.
- `Multiplier[I]`: `Factor`, the further rolls' `Source`, and `Join`. With a factor above one,
  each category whose first roll took items rolls `Factor-1` more times, each roll taking as many
  items as the first took.
- `Repeat` and `Held`: the effect-repeat rule. `RepeatReplace` gives the held entry of a repeated
  kind the new value in place; `RepeatAppend` appends.
- `Profile[C, I]`: `Categories`, `Place`, `Multiplier`, `Repeat`.

### The two profiles

| Step | First game (`Shop.stockProfile`, `pkg/game/shopstockprofile.go`) | Second game (`Tables.Profile`, `pkg/secondshop/fill.go`) |
|---|---|---|
| Location | the shop's ceiling (chapter ShopMax) | the location's assortment record |
| Categories | armour, weapons, Magic Items, books (books only with a draw source) | the record's four categories |
| Candidates | definition-table pools under the ceiling: filtered Shields+Armors, filtered Weapons, both unfiltered for Magic Items; books and scrolls (one draw per scroll) | `Tables.Candidates` (one draw per phase-two scroll spell in a magic category) |
| Lead | none | magic category: admitted books at 2, six potions at 51..100 |
| Draw count | 100, 100, 20; books 1..8 (one draw) | the category's `Draws` |
| Bound | attempts 100, 100, 201; books none | failures 10 x Draws |
| Redraw | none | a book, up to 1000 times |
| Make | plain unit; Magic Items `shopEnchant`; books clone | quantity, `enchant` or stackable quantity, price, price window |
| Insert | append | `secondshop.Insert` (merge) |
| Finish | stack and sort; books first add the guaranteed books (DIV-1025) and the potions | none |
| Place | mods' stocked items on the armour shelf (DIV-1862) | mods' stocked items by `Place` |
| Multiplier | the shop's `stockMultiplier`, own generator from the shop seed (DIV-3960); further-roll attempts 100, 100, 100 x 201 | off (factor zero) |
| Effect repeat | `RepeatReplace` | not used: the enchant gives one spell |
| Draw source | `Intn` of the shared stream in original mode or `random.Go` at the shop seed | `(rand()*n)>>15` over the caller's rand() |

### Sites before and after

| Before (base `18c85b63`) | After |
|---|---|
| `pkg/game/shop.go:314` `generate` | `shop.go` `generate` calls `shopstock.Generate(s.stockProfile(t, r), r)` |
| `pkg/game/shop.go:400` `rollShelf` | `shopstock` `roll` with the first game's categories |
| `pkg/game/shop.go:426` `rollAgain` | `shopstock` multiplier step; `Join` is the first game's stacker and sort |
| `pkg/game/shop.go:362-394` books draw, guaranteed books and potions | books category; `shopGuaranteedBooks` |
| `pkg/game/shop.go:354` mods' stocked items on the armour shelf | the first game's `Profile.Place` |
| `pkg/game/shopenchant.go:65-72` repeated-kind overwrite | `shopstock.Held` under the profile's `Repeat` |
| `pkg/secondshop/fill.go:33` `Fill` | `Fill` calls `shopstock.Generate(t.Profile(r, rnd), picks(rnd))` |
| `pkg/secondshop/fill.go:63` `draw` | `shopstock` `roll`; `Tables.lead` and `Tables.make` |
| `pkg/secondshop/fill.go:115` `enchant` | unchanged rule, draws through `Source` |

Callers are unchanged: `campaignreturn.go` `arriveInTown`, `townshell.go` tavern restock,
`secondtownshop.go` `enter` and `restock`, `campaignsecond.go` LOAD restock.

## Steps outside the shape

- Pools and candidate admission stay each game's own functions (`shopArmourPool`,
  `shopWeaponPool`, `shopPlainShelfPool`, `shopBookPool`, `shopScrollPool`;
  `secondshop.Tables.Candidates`), because each states its own game's claims.
- The enchants stay each game's own: the first game's `shopEnchant` (effect kinds, budgets,
  optional effects, usable-by clash) and the second game's `Tables.enchant` (one spell from seven,
  level cap). Only the effect-repeat rule inside the first game's enchant is the shared step; the
  second game's enchant gives one spell, so no effect repeats and its profile does not use it.
- The first game's mods' stocked items were placed on the armour shelf before the books shelf
  drew; they are now placed after it. They take no draw and touch only the armour shelf, so the
  stock and the stream are unchanged.

## Proof

- Stock equal to main. An instrument test (kept in the lane notes, not committed) dumped every
  generated stock, item by item in order with effects, price and count, and the four draws that
  follow the generation, on the candidate and on a `git archive` of main `18c85b63`. First game:
  14 shop ceilings (each chapter's ShopMax and 1000000000) x seeds 0..199 x multiplier 1, 2, 10,
  1000 x the seeded and the original shared stream (at 1000 the original stream every tenth seed),
  plus one tavern restock at multiplier 2: 19880 generations and 5600 restocks per root. At 1000
  each shelf is written as entry count, unit count and SHA-256 of its items. Second game: locations
  0..3 x seeds 0..199 x both modes, 1600 per root. The dumps are byte-identical on en, ru, rom2-en
  and rom2-ru.
- `TestReleaseSecondShopStockMatchesMain` (`pkg/secondshop/stockmain_test.go`) holds records 1
  and 2 over 100 seeds of the original rand() to main's digest. The first game's
  `TestReleaseShopStockMultiplierDefaultMatchesMain` already holds its default stock to main's
  digest.
- SAV bytes equal to main. A first-game town SAV with a staged deal and its cold-LOAD re-SAVE, the
  stock before and after the LOAD, and the World hash live in the town (en, ru); a second-game
  first-town SAV with an open deal and after leaving the shop (rom2-en, rom2-ru): identical on
  the candidate and on main, run twice on main.
- Loss controls, each restored from the lane's own commit. The first game's repeat rule set to the
  second game's (`RepeatAppend`): `TestShopRepeatedKindOverwritesTheHeldPayloadInPlace`,
  `TestShopNewKindAppendsAndEachKindKeepsOneEntry`,
  `TestFixedShopDrawsProduceOneTwoAndThreeOrderedEffectsAtExactGates`,
  `TestReleaseShopStockHoldsOneEntryPerEffectKind` and
  `TestReleaseShopStockMultiplierDefaultMatchesMain` fail; every second-game test passes. The
  second game's bound set to the first game's form (attempts equal to the draw count):
  `TestReleaseSecondShopStockMatchesMain` fails; every first-game shop test passes. Before the
  digest witness, no second-game test failed under that change.

## Open debt

None known.
