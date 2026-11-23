# The second game's shop shelf order is checked against the original's client keys

## Result

In the second game every shop shelf in the first and second towns shows its items in the original's client order, in
both the English and the Russian game. The Russian client was read and is the English one, so one order serves both.
The pack strip keeps the bag's order. The first game's shop is unchanged.

## Authority

- Owner: «the second game's shops use the original's shown order on EN and RU».
- R2-ENGINE-1382: the RU client install, comparator, sort, price, kind, group, qsort, short sort and swap equal EN.
- R2-ENGINE-798, R2-ENGINE-799: the client keys and the equal-key permutations.
- R2-ENGINE-1384: the hero strip list is installed unsorted.
- DIV-4451 closed; DIV-4328 and DIV-4092 narrowed; DIV-3042 unchanged, no claim reads the chooser.

## Rules

1. A shelf is ordered by flag 0x2 set first, kind, weapon row group and shown price, on the EN and the RU root.
2. The pack strip lists the bag in bag order, then the gold; no sort touches it.
3. The first game's shelf order and stock are unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| shown shelf order | `shopstock.ShelfOrder` via `secondshop.Tables.Sort` | none; the row closed on the RU claim |

## Proof

| test | rules |
|---|---|
| `TestReleaseSecondShopShownOrderFollowsTheClientKeys` | 1 |
| `TestSecondShopPageGoldFollowsTheBag` | 2 |
| `TestShelfSortIsStableOnEqualPrices` | 3 |

Release tests at the release: `TestReleaseSecondShopShownOrderFollowsTheClientKeys`.

## Open debt

- DIV-3042 waits for a claim on the phase-two chooser; DIV-4328 for the stale slot values at a MagicItems fill;
  DIV-4092 for the sender of the hero strip list.
